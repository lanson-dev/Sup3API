package sup3

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"
)

type Engine struct {
	Store          *Store
	Providers      map[string]Provider
	DataDir        string
	DownloadClient *http.Client
}

func NewEngine(store *Store, dir string, providers ...Provider) *Engine {
	e := &Engine{Store: store, DataDir: dir, Providers: map[string]Provider{}, DownloadClient: assetClient()}
	for _, p := range providers {
		e.Providers[p.Name()] = p
	}
	return e
}
func (e *Engine) Prepare(r *Request) (Price, error) {
	p, ok := e.Providers[r.Provider]
	if !ok {
		return Price{}, invalid("unknown provider")
	}
	for _, u := range append(append([]string{}, r.Inputs.Images...), r.Inputs.ModelURL) {
		if u != "" {
			if err := validRemoteURL(u); err != nil {
				return Price{}, err
			}
		}
	}
	return p.Prepare(r)
}
func (e *Engine) Create(ctx context.Context, owner, key int64, idempotency string, r Request) (*Job, bool, error) {
	if len(idempotency) < 8 || len(idempotency) > 128 || strings.TrimSpace(idempotency) != idempotency {
		return nil, false, invalid("Idempotency-Key header must be 8-128 characters")
	}
	quote, err := e.Prepare(&r)
	if err != nil {
		return nil, false, err
	}
	if !e.Providers[r.Provider].Capability().Available {
		return nil, false, &APIError{Code: "provider_not_configured", Message: "provider credentials are missing", HTTPStatus: 503}
	}
	resolved := ""
	if r.Inputs.JobID != "" {
		source, err := e.Store.Get(ctx, r.Inputs.JobID, owner, key)
		if err != nil {
			return nil, false, err
		}
		if source.Status != "succeeded" || len(source.Steps) == 0 {
			return nil, false, invalid("source job must have succeeded")
		}
		if source.Request.Provider != r.Provider {
			return nil, false, invalid("cross-provider job references require exporting and hosting a model_url")
		}
		if r.Operation == "animate" && source.Request.Operation != "rig" {
			return nil, false, invalid("animate requires a rig job")
		}
		if r.Provider == "meshy" && r.Operation == "rig" && !source.Request.Textured() {
			return nil, false, invalid("Meshy rig requires a textured model")
		}
		resolved = source.Steps[len(source.Steps)-1].UpstreamID
	}
	b, err := json.Marshal(r)
	if err != nil {
		return nil, false, err
	}
	hash := sha256.Sum256(b)
	now := time.Now().UTC()
	cost := quote
	cost.Credits = 0
	cost.USD = nil
	cost.Kind = "pending"
	j := &Job{ID: "job_" + randomID(), SchemaVersion: SchemaVersion, OwnerID: owner, KeyID: key, Request: r, Status: "queued", Steps: []Step{}, Quote: quote, Cost: cost, DeliveryStatus: "pending", Artifacts: []Artifact{}, Components: map[string]Component{}, CreatedAt: now, UpdatedAt: now, IdempotencyKey: idempotency, RequestHash: hex.EncodeToString(hash[:]), ResolvedInput: resolved}
	return e.Store.Create(ctx, j)
}
func apiError(err error) *APIError {
	var a *APIError
	if errors.As(err, &a) {
		return a
	}
	return &APIError{Code: "internal_error", Message: "asset service operation failed", HTTPStatus: 500}
}
func (e *Engine) Start(ctx context.Context) {
	for i := 0; i < 2; i++ {
		go func() {
			timer := time.NewTicker(2 * time.Second)
			defer timer.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-timer.C:
					if err := e.Tick(ctx); err != nil {
						log.Printf("sup3 worker: %s", apiError(err).Code)
					}
				}
			}
		}()
	}
}
func (e *Engine) Tick(ctx context.Context) error {
	j, token, err := e.Store.Claim(ctx, "")
	if err != nil || j == nil {
		return err
	}
	workCtx, cancel := context.WithTimeout(ctx, 4*time.Minute)
	defer cancel()
	if err = e.advance(workCtx, j, token); err != nil {
		return err
	}
	return e.Store.Save(ctx, j, token, true)
}
func (e *Engine) advance(ctx context.Context, j *Job, token string) error {
	p, ok := e.Providers[j.Request.Provider]
	if !ok {
		return errors.New("provider missing")
	}
	if j.Status == "submitting" {
		j.Status = "submission_unknown"
		j.Error = &APIError{Code: "submission_unknown", Message: "worker stopped during submission; reconcile upstream before creating another job"}
		j.DeliveryStatus = "unavailable"
		return nil
	}
	if j.Status == "succeeded" && j.DeliveryStatus == "pending" {
		if err := e.download(ctx, j); err != nil {
			j.DeliveryStatus = "failed"
			j.Error = &APIError{Code: "delivery_failed", Message: "generation succeeded; artifact delivery failed, retry delivery without regenerating", Retryable: true}
			return nil
		}
		j.DeliveryStatus = "ready"
		j.Error = nil
		return nil
	}
	if len(j.Steps) == 0 {
		return e.submit(ctx, j, token, stepName(j.Request), "")
	}
	idx := len(j.Steps) - 1
	step := &j.Steps[idx]
	obs, err := p.Poll(ctx, *step)
	if err != nil {
		a := apiError(err)
		j.Error = a
		if !a.Retryable {
			j.Status = "failed"
			j.DeliveryStatus = "unavailable"
		}
		return nil
	}
	if obs.Status == "unknown" {
		j.Error = &APIError{Code: "unknown_upstream_status", Message: "upstream returned an unrecognized task state", Retryable: true}
		return nil
	}
	j.Error = obs.Error
	step.Status = obs.Status
	step.Credits = obs.Credits
	j.Progress = obs.Progress
	j.Status = obs.Status
	if j.Request.Provider == "meshy" && j.Request.Operation == "text_to_3d" && j.Request.Textured() {
		if step.Name == "preview" {
			j.Progress = obs.Progress / 2
		} else {
			j.Progress = 50 + obs.Progress/2
		}
	}
	e.updateCost(j)
	if obs.Status == "succeeded" {
		if j.Request.Provider == "meshy" && j.Request.Operation == "text_to_3d" && j.Request.Textured() && step.Name == "preview" {
			return e.submit(ctx, j, token, "refine", step.UpstreamID)
		}
		j.Progress = 100
		j.Artifacts = obs.Artifacts
		if len(j.Artifacts) == 0 {
			j.DeliveryStatus = "failed"
			j.Error = &APIError{Code: "no_artifacts", Message: "upstream succeeded without supported artifacts"}
		} else {
			j.DeliveryStatus = "pending"
		}
	} else if j.Status == "failed" || j.Status == "canceled" {
		j.DeliveryStatus = "unavailable"
	}
	return nil
}
func (e *Engine) submit(ctx context.Context, j *Job, token, stage, previous string) error {
	// Commit the intent before the external side effect. Lease recovery cannot
	// distinguish a crashed successful POST from an unsent one, so neither retries.
	j.Status = "submitting"
	if err := e.Store.Save(ctx, j, token, false); err != nil {
		return err
	}
	r := j.Request
	r.Inputs.UpstreamID = j.ResolvedInput
	step, err := e.Providers[r.Provider].Submit(ctx, r, stage, previous)
	if err != nil {
		j.Error = apiError(err)
		j.Status = "failed"
		if j.Error.Uncertain {
			j.Status = "submission_unknown"
		}
		j.DeliveryStatus = "unavailable"
		return nil
	}
	j.Steps = append(j.Steps, step)
	e.updateCost(j)
	j.Status = "queued"
	j.Error = nil
	return nil
}
func (e *Engine) updateCost(j *Job) {
	total := 0.0
	reported := true
	known := 0
	for _, s := range j.Steps {
		if s.Credits == nil {
			reported = false
		} else {
			total += *s.Credits
			known++
		}
	}
	j.Cost = j.Quote
	j.Cost.USD = nil
	if reported {
		j.Cost.Credits = total
		j.Cost.Kind = "reported"
	} else {
		j.Cost.Credits = total
		j.Cost.Kind = "partially_reported"
	}
	if known == 0 {
		j.Cost.Kind = "pending"
	}
	if j.Request.Provider == "tripo" {
		usd := j.Cost.Credits * 0.01
		j.Cost.USD = &usd
	}
}
func (e *Engine) Mutate(ctx context.Context, id string, owner, key int64, action string) (*Job, error) {
	if _, err := e.Store.Get(ctx, id, owner, key); err != nil {
		return nil, err
	}
	j, token, err := e.Store.Claim(ctx, id)
	if err != nil {
		return nil, err
	}
	if j == nil {
		return nil, &APIError{Code: "job_busy", Message: "job is being processed; retry shortly", HTTPStatus: 409, Retryable: true}
	}
	defer func() { _ = e.Store.Save(context.WithoutCancel(ctx), j, token, true) }()
	switch action {
	case "retry-delivery", "refresh-artifacts":
		if j.Status != "succeeded" || (action == "retry-delivery" && j.DeliveryStatus != "failed") {
			return nil, invalid("refresh requires a succeeded job; retry-delivery requires failed delivery")
		}
		obs, err := e.Providers[j.Request.Provider].Poll(ctx, j.Steps[len(j.Steps)-1])
		if err != nil {
			return nil, err
		}
		if obs.Status != "succeeded" || len(obs.Artifacts) == 0 {
			return nil, invalid("upstream produced no supported artifacts")
		}
		j.Artifacts = obs.Artifacts
		j.Components = map[string]Component{}
		j.DeliveryStatus = "pending"
		j.Error = nil
	case "cancel":
		if j.Terminal() {
			return nil, &APIError{Code: "cancel_not_supported", Message: "job is already terminal", HTTPStatus: 409}
		}
		if j.Status == "submitting" {
			return nil, &APIError{Code: "submission_unknown", Message: "submission must be reconciled first", HTTPStatus: 409}
		}
		if len(j.Steps) > 0 {
			if err = e.Providers[j.Request.Provider].Cancel(ctx, j.Steps[len(j.Steps)-1]); err != nil {
				return nil, err
			}
		}
		j.Status = "canceled"
		j.DeliveryStatus = "unavailable"
	default:
		return nil, invalid("unknown action")
	}
	if err = e.Store.Save(ctx, j, token, false); err != nil {
		return nil, err
	}
	return j, nil
}
