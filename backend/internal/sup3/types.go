// Package sup3 implements provider-independent, asynchronous 3D asset APIs.
// It deliberately does not import Sub2API's handler/service packages.
package sup3

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

const SchemaVersion = "sup3.asset.v1"

type Inputs struct {
	Prompt   string   `json:"prompt,omitempty"`
	Images   []string `json:"images,omitempty"`
	ModelURL string   `json:"model_url,omitempty"`
	JobID    string   `json:"job_id,omitempty"`
	// UpstreamID is filled by the worker after checking source-job ownership.
	UpstreamID string `json:"-"`
}

type Parameters struct {
	Formats           []string `json:"formats,omitempty"`
	Texture           *bool    `json:"texture,omitempty"`
	PBR               *bool    `json:"pbr,omitempty"`
	TextureResolution string   `json:"texture_resolution,omitempty"`
	TargetFaces       int      `json:"target_faces,omitempty"`
	MaxFaces          int      `json:"max_faces,omitempty"`
	Topology          string   `json:"topology,omitempty"`
	Pose              string   `json:"pose,omitempty"`
	Animations        []string `json:"animations,omitempty"`
}

type Request struct {
	Output          *OutputRequirements                   `json:"output,omitempty"`
	Extensions      map[string]map[string]json.RawMessage `json:"extensions,omitempty"`
	InputFormat     string                                `json:"input_format,omitempty"`
	Payload         map[string]json.RawMessage            `json:"payload,omitempty"`
	Provider        string                                `json:"provider"`
	Model           string                                `json:"model,omitempty"`
	Operation       string                                `json:"operation"`
	Inputs          Inputs                                `json:"inputs"`
	Parameters      Parameters                            `json:"parameters"`
	ProviderOptions map[string]json.RawMessage            `json:"provider_options,omitempty"`
}

func (r Request) Textured() bool { return r.Parameters.Texture == nil || *r.Parameters.Texture }
func (r Request) PBR() bool      { return r.Textured() && (r.Parameters.PBR == nil || *r.Parameters.PBR) }

type Price struct {
	Provider   string   `json:"provider"`
	Credits    float64  `json:"credits"`
	Unit       string   `json:"unit"`
	Multiplier float64  `json:"multiplier"`
	USD        *float64 `json:"usd,omitempty"`
	Source     string   `json:"source"`
	AsOf       string   `json:"as_of"`
	Kind       string   `json:"kind"` // estimate or reported
}

type Capability struct {
	NativeAPI         map[string]any      `json:"native_api"`
	OperationDetails  map[string]any      `json:"operation_details"`
	InputFormats      []string            `json:"input_formats"`
	Provider          string              `json:"provider"`
	Models            []string            `json:"models"`
	ModelsByOperation map[string][]string `json:"models_by_operation"`
	Operations        []string            `json:"operations"`
	Authentication    string              `json:"authentication"`
	InputSchema       map[string]any      `json:"input_schema"`
	Formats           []string            `json:"formats"`
	Notes             []string            `json:"notes"`
	Available         bool                `json:"available"`
}

type Artifact struct {
	ID          string         `json:"id"`
	Role        string         `json:"role"`
	Format      string         `json:"format"`
	MediaType   string         `json:"media_type"`
	URL         string         `json:"url,omitempty"`
	SourceURL   string         `json:"-"`
	Path        string         `json:"-"`
	Size        int64          `json:"size,omitempty"`
	SHA256      string         `json:"sha256,omitempty"`
	DerivedFrom string         `json:"derived_from,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type Component struct {
	Status      string   `json:"status"` // available, absent, unsupported, pending
	ArtifactIDs []string `json:"artifact_ids,omitempty"`
	Reason      string   `json:"reason,omitempty"`
}

type Step struct {
	ProviderResult map[string]any `json:"provider_result,omitempty"`
	Name           string         `json:"name"`
	UpstreamID     string         `json:"upstream_id,omitempty"`
	Endpoint       string         `json:"-"`
	Status         string         `json:"status"`
	Credits        *float64       `json:"credits,omitempty"`
}

type APIError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Retryable  bool   `json:"retryable"`
	HTTPStatus int    `json:"-"`
	Uncertain  bool   `json:"-"`
}

func (e *APIError) Error() string { return e.Code + ": " + e.Message }
func invalid(message string) error {
	return &APIError{Code: "invalid_request", Message: message, HTTPStatus: 400}
}

type Job struct {
	ResolvedModel    string               `json:"resolved_model,omitempty"`
	OutputValidation *OutputValidation    `json:"output_validation,omitempty"`
	ID               string               `json:"id"`
	SchemaVersion    string               `json:"schema_version"`
	OwnerID          int64                `json:"-"`
	KeyID            int64                `json:"-"`
	Request          Request              `json:"request"`
	Status           string               `json:"status"`
	Progress         int                  `json:"progress"`
	Steps            []Step               `json:"steps"`
	Quote            Price                `json:"quote"`
	Cost             Price                `json:"cost"`
	DeliveryStatus   string               `json:"delivery_status"`
	Artifacts        []Artifact           `json:"artifacts"`
	Components       map[string]Component `json:"components"`
	Error            *APIError            `json:"error,omitempty"`
	CreatedAt        time.Time            `json:"created_at"`
	UpdatedAt        time.Time            `json:"updated_at"`
	// These fields are persisted separately from the public JSON view.
	IdempotencyKey string `json:"-"`
	RequestHash    string `json:"-"`
	ResolvedInput  string `json:"-"`
	AccountBinding string `json:"-"`
}

func (j *Job) Terminal() bool {
	return j.Status == "succeeded" || j.Status == "failed" || j.Status == "canceled" || j.Status == "submission_unknown"
}

type Observation struct {
	ProviderResult map[string]any
	Status         string
	Progress       int
	Credits        *float64
	Artifacts      []Artifact
	Error          *APIError
}

type Provider interface {
	Name() string
	Capability() Capability
	Prepare(*Request) (Price, error)
	Submit(context.Context, Request, string, string) (Step, error)
	Poll(context.Context, Step) (Observation, error)
	Cancel(context.Context, Step) error
	Balance(context.Context) (map[string]any, error)
}

func stepName(r Request) string {
	if r.Provider == "meshy" && r.Operation == "text_to_3d" {
		return "preview"
	}
	return r.Operation
}

func normalizedStatus(s string) string {
	switch s {
	case "PENDING", "queued", "pending":
		return "queued"
	case "IN_PROGRESS", "running", "processing":
		return "running"
	case "SUCCEEDED", "success", "succeeded":
		return "succeeded"
	case "FAILED", "failed", "banned", "expired":
		return "failed"
	case "CANCELED", "cancelled", "canceled":
		return "canceled"
	default:
		return "unknown"
	}
}

func option[T any](r Request, key string, fallback T) T {
	if raw, ok := r.ProviderOptions[key]; ok {
		var value T
		if json.Unmarshal(raw, &value) == nil {
			return value
		}
	}
	return fallback
}

func validateOptions(r Request, allowed map[string]bool) error {
	for key := range r.ProviderOptions {
		if !allowed[key] {
			return invalid(fmt.Sprintf("unsupported provider_options.%s", key))
		}
	}
	return nil
}
