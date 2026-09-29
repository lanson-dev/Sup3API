package sup3

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
)

// ProviderResolver selects a managed account when binding is empty, otherwise
// resolves exactly the original account and credential revision. Never fail over
// a submitted task to another upstream account.
type ProviderResolver func(context.Context, string, string) (Provider, string, error)

func CredentialFingerprint(key string) string {
	digest := sha256.Sum256([]byte(key))
	return hex.EncodeToString(digest[:])
}

func AccountBinding(id int64, key string) string {
	return strconv.FormatInt(id, 10) + ":" + CredentialFingerprint(key)
}

func (e *Engine) resolve(ctx context.Context, provider, binding string) (Provider, string, error) {
	if e.ResolveProvider != nil {
		return e.ResolveProvider(ctx, provider, binding)
	}
	p := e.Providers[provider]
	if p == nil || !p.Capability().Available {
		return nil, "", &APIError{Code: "provider_not_configured", Message: "no available upstream account", HTTPStatus: 503}
	}
	if remote, ok := p.(*RemoteProvider); ok {
		return p, CredentialFingerprint(remote.Key), nil
	}
	return p, binding, nil
}

func (e *Engine) jobProvider(ctx context.Context, j *Job) (Provider, error) {
	if e.ResolveProvider != nil && j.AccountBinding == "" {
		return nil, &APIError{Code: "legacy_account_unbound", Message: "legacy task requires upstream account reconciliation", HTTPStatus: 409}
	}
	p, _, err := e.resolve(ctx, j.Request.Provider, j.AccountBinding)
	return p, err
}

// Routing context is attached only by the authenticated gateway, never decoded
// from client JSON. Workers use their persisted account binding instead.
type routingKey struct{}
type Routing struct {
	GroupID int64
	Model   string
}

func WithRouting(ctx context.Context, groupID int64) context.Context {
	return context.WithValue(ctx, routingKey{}, Routing{GroupID: groupID})
}
func RoutingFromContext(ctx context.Context) Routing {
	value, _ := ctx.Value(routingKey{}).(Routing)
	return value
}
func withModel(ctx context.Context, model string) context.Context {
	routing := RoutingFromContext(ctx)
	routing.Model = model
	return context.WithValue(ctx, routingKey{}, routing)
}

func (e *Engine) prepareAccount(ctx context.Context, r *Request, binding string) (Price, string, string, error) {
	if _, ok := e.Providers[r.Provider]; !ok {
		return Price{}, "", "", invalid("unknown provider")
	}
	if err := normalizeContract(r); err != nil {
		return Price{}, "", "", err
	}
	if remote, ok := e.Providers[r.Provider].(*RemoteProvider); ok && r.Model == "" {
		r.Model = remote.DefaultModel(r.Operation)
	}
	model := r.Model

	provider, binding, err := e.resolve(withModel(ctx, model), r.Provider, binding)
	if err != nil {
		return Price{}, "", "", err
	}
	upstream := *r
	if remote, ok := provider.(*RemoteProvider); ok && remote.ResolveModel != nil && upstream.Model != "" {
		upstream.Model = remote.ResolveModel(upstream.Model)
	}
	price, err := e.Prepare(&upstream)
	if err != nil {
		return Price{}, "", "", err
	}
	publicModel := r.Model
	if publicModel == "" {
		publicModel = upstream.Model
	}
	*r = upstream
	r.Model = publicModel
	return price, binding, upstream.Model, nil
}
