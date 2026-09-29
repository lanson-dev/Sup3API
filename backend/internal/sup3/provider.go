package sup3

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)

type RemoteProvider struct {
	ID, Key, BaseURL string
	Client           *http.Client
}

func NewProvider(name, key string) *RemoteProvider {
	base := "https://api.meshy.ai"
	if name == "tripo" {
		base = "https://openapi.tripo3d.ai/v3"
	}
	return &RemoteProvider{ID: name, Key: key, BaseURL: base, Client: &http.Client{Timeout: 90 * time.Second}}
}
func (p *RemoteProvider) Name() string { return p.ID }
func (p *RemoteProvider) Capability() Capability {
	models := []string{"meshy-7.1", "meshy-6", "meshy-6-lite", "meshy-t2"}
	notes := []string{"text_to_3d with texture runs preview then refine", "target_faces is an approximate target; max_faces is unsupported", "rig and animate require a compatible humanoid model", "multi_image_to_3d accepts 1-4 views"}
	if p.ID == "tripo" {
		models = []string{"v3.1-20260211", "v3.0-20250812", "v2.5-20250123"}
		notes = []string{"max_faces is an upper limit; target_faces is unsupported", "quad topology produces FBX; portable component extraction requires GLB", "multi_image_to_3d requires four positional slots: front,left,back,right, at least two nonempty"}
	}
	byOperation := map[string][]string{"text_to_3d": models, "image_to_3d": models, "multi_image_to_3d": models, "retexture": {"meshy-7", "meshy-6", "meshy-6-lite"}, "rig": {}, "animate": {}}
	if p.ID == "tripo" {
		byOperation["retexture"] = []string{"v3.5-20260815"}
		byOperation["rig"] = []string{"v1.0-20240301", "v2.5-20260210"}
	} else {
		byOperation["multi_image_to_3d"] = []string{"meshy-7.1", "meshy-6", "meshy-6-lite"}
	}
	formats := []string{"glb", "fbx"}
	if p.ID == "meshy" {
		formats = []string{"glb", "fbx", "obj", "stl", "usdz", "3mf"}
	}
	return Capability{InputFormats: []string{"sup3api", p.ID}, Provider: p.ID, Models: models, ModelsByOperation: byOperation, Operations: []string{"text_to_3d", "image_to_3d", "multi_image_to_3d", "retexture", "rig", "animate"}, Authentication: "api_key", Formats: formats, Available: p.Key != "", Notes: notes, InputSchema: map[string]any{"type": "object", "properties": map[string]any{"prompt": map[string]any{"type": "string"}, "images": map[string]any{"type": "array", "maxItems": 4, "items": map[string]any{"type": "string"}}, "model_url": map[string]any{"type": "string", "format": "uri"}, "job_id": map[string]any{"type": "string"}}}}
}

func (p *RemoteProvider) call(ctx context.Context, method, endpoint string, body any) (map[string]any, error) {
	if p.Key == "" {
		return nil, &APIError{Code: "provider_not_configured", Message: p.ID + " API key is not configured", HTTPStatus: 503}
	}
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, p.BaseURL+endpoint, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.Key)
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, &APIError{Code: "upstream_transport_error", Message: p.ID + " request did not return a response", HTTPStatus: 502, Retryable: method == "GET", Uncertain: method == "POST"}
	}
	defer resp.Body.Close()
	data := map[string]any{}
	err = json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&data)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		code := "upstream_error"
		status := 502
		switch resp.StatusCode {
		case 401, 403:
			code = "upstream_authentication_failed"
		case 402:
			code = "upstream_insufficient_credits"
			status = 402
		case 400, 422:
			code = "upstream_invalid_request"
			status = 422
		case 404:
			code = "upstream_not_found"
			status = 404
		case 409:
			code = "upstream_conflict"
			status = 409
		case 429:
			code = "upstream_rate_limited"
			status = 429
		}
		return nil, &APIError{Code: code, Message: fmt.Sprintf("%s returned HTTP %d", p.ID, resp.StatusCode), HTTPStatus: status, Retryable: method == "GET" && (resp.StatusCode == 429 || resp.StatusCode >= 500), Uncertain: method == "POST" && resp.StatusCode >= 500}
	}
	if method == "DELETE" && (resp.StatusCode == 204 || err == io.EOF) {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, &APIError{Code: "upstream_invalid_response", Message: p.ID + " returned invalid JSON", HTTPStatus: 502, Uncertain: method == "POST"}
	}
	if p.ID == "tripo" {
		if code, _ := data["code"].(float64); code != 0 {
			name := "upstream_error"
			status := 422
			msg := strings.ToLower(fmt.Sprint(data["message"], data["msg"]))
			if strings.Contains(msg, "credit") || strings.Contains(msg, "balance") {
				name = "upstream_insufficient_credits"
				status = 402
			}
			return nil, &APIError{Code: name, Message: fmt.Sprintf("tripo rejected request with code %.0f", code), HTTPStatus: status}
		}
		if nested, ok := data["data"].(map[string]any); ok {
			return nested, nil
		}
	}
	return data, nil
}

func (p *RemoteProvider) Balance(ctx context.Context) (map[string]any, error) {
	endpoint := "/openapi/v1/balance"
	if p.ID == "tripo" {
		endpoint = "/account/balance"
	}
	return p.call(ctx, "GET", endpoint, nil)
}

func (p *RemoteProvider) Prepare(r *Request) (Price, error) {
	if p.ID != "meshy" && p.ID != "tripo" {
		return Price{}, invalid("unknown provider")
	}
	r.Provider = p.ID
	if r.Model == "" {
		if p.ID == "meshy" {
			if r.Operation != "rig" && r.Operation != "animate" {
				r.Model = "meshy-7.1"
				if r.Operation == "retexture" {
					r.Model = "meshy-7"
				}
			}
		} else {
			switch r.Operation {
			case "rig":
				r.Model = "v1.0-20240301"
			case "retexture":
				r.Model = "v3.5-20260815"
			case "animate":
			default:
				r.Model = "v3.1-20260211"
			}
		}
	}
	if r.Parameters.Texture != nil && !*r.Parameters.Texture && r.Parameters.PBR != nil && *r.Parameters.PBR {
		return Price{}, invalid("pbr requires texture")
	}
	if r.Parameters.TargetFaces < 0 || r.Parameters.MaxFaces < 0 {
		return Price{}, invalid("face counts must be nonnegative")
	}
	if p.ID == "meshy" && r.Parameters.MaxFaces != 0 {
		return Price{}, invalid("Meshy supports target_faces, not max_faces")
	}
	if p.ID == "tripo" && r.Parameters.TargetFaces != 0 {
		return Price{}, invalid("Tripo supports max_faces, not target_faces")
	}
	if t := r.Parameters.Topology; t != "" && t != "triangle" && t != "quad" {
		return Price{}, invalid("topology must be triangle or quad")
	}
	if t := r.Parameters.TextureResolution; t != "" && t != "2k" && t != "4k" && t != "8k" {
		return Price{}, invalid("texture_resolution must be 2k, 4k or 8k")
	}
	if p.ID == "tripo" && r.Parameters.TextureResolution != "" {
		return Price{}, invalid("Tripo exposes texture_quality, not an exact texture_resolution; use provider_options.texture_quality")
	}
	if pose := r.Parameters.Pose; pose != "" && (p.ID != "meshy" || (pose != "a-pose" && pose != "t-pose")) {
		return Price{}, invalid("pose is supported by Meshy only: a-pose or t-pose")
	}
	switch r.Operation {
	case "text_to_3d":
		limit := 800
		if p.ID == "tripo" {
			limit = 1024
		}
		if len([]rune(strings.TrimSpace(r.Inputs.Prompt))) == 0 || len([]rune(r.Inputs.Prompt)) > limit {
			return Price{}, invalid(fmt.Sprintf("prompt must contain 1-%d characters", limit))
		}
	case "image_to_3d":
		if len(r.Inputs.Images) != 1 {
			return Price{}, invalid("image_to_3d requires one image")
		}
	case "multi_image_to_3d":
		if len(r.Inputs.Images) < 1 || len(r.Inputs.Images) > 4 {
			return Price{}, invalid("requires 1-4 image inputs")
		}
		if p.ID == "tripo" {
			nonempty := 0
			for _, s := range r.Inputs.Images {
				if s != "" {
					nonempty++
				}
			}
			if len(r.Inputs.Images) != 4 || r.Inputs.Images[0] == "" || nonempty < 2 {
				return Price{}, invalid("Tripo requires [front,left,back,right], with front and at least one other view")
			}
		}
	case "retexture", "rig", "animate":
		if (r.Inputs.JobID == "") == (r.Inputs.ModelURL == "") {
			return Price{}, invalid("provide exactly one inputs.job_id or inputs.model_url")
		}
		if r.Operation == "animate" && (r.Inputs.JobID == "" || len(r.Parameters.Animations) == 0) {
			return Price{}, invalid("animate requires a rig job_id and animations")
		}
	default:
		return Price{}, invalid("unsupported operation")
	}
	if r.Operation != "animate" && len(r.Parameters.Animations) > 0 {
		return Price{}, invalid("animations require the animate operation")
	}
	if err := validateSemantics(*r); err != nil {
		return Price{}, err
	}
	return p.price(r)
}

func (p *RemoteProvider) Submit(ctx context.Context, r Request, stage, previous string) (Step, error) {
	endpoint, body, err := p.payload(r, stage, previous)
	if err != nil {
		return Step{}, err
	}
	data, err := p.call(ctx, "POST", endpoint, body)
	if err != nil {
		return Step{}, err
	}
	id := str(data["result"])
	if p.ID == "tripo" {
		id = str(data["task_id"])
	}
	if id == "" {
		return Step{}, &APIError{Code: "submission_unknown", Message: "upstream accepted request but returned no task ID", HTTPStatus: 502, Uncertain: true}
	}
	query := endpoint
	if p.ID == "tripo" {
		query = "/tasks"
	}
	return Step{Name: stage, UpstreamID: id, Endpoint: query, Status: "queued"}, nil
}

func (p *RemoteProvider) Poll(ctx context.Context, step Step) (Observation, error) {
	data, err := p.call(ctx, "GET", step.Endpoint+"/"+url.PathEscape(step.UpstreamID), nil)
	if err != nil {
		return Observation{}, err
	}
	obs := Observation{Status: normalizedStatus(str(data["status"])), ProviderResult: data}
	if progress, ok := data["progress"].(float64); ok {
		obs.Progress = int(progress)
	}
	key := "consumed_credits"
	if p.ID == "tripo" {
		key = "credits_consumed"
	}
	if credits, ok := data[key].(float64); ok {
		obs.Credits = &credits
	}
	if obs.Status == "failed" {
		obs.Error = &APIError{Code: "upstream_task_failed", Message: p.ID + " task failed; upstream task ID is available in steps", HTTPStatus: 502}
	}
	if obs.Status == "succeeded" {
		obs.Progress = 100
		obs.Artifacts = collectArtifacts(p.ID, data)
	}
	return obs, nil
}

func (p *RemoteProvider) Cancel(ctx context.Context, step Step) error {
	// Meshy's DELETE also destroys completed results. Only pending tasks are cancellable.
	if p.ID == "meshy" {
		obs, err := p.Poll(ctx, step)
		if err != nil {
			return err
		}
		if obs.Status != "queued" {
			return &APIError{Code: "cancel_not_supported", Message: "Meshy can cancel pending tasks only", HTTPStatus: 409}
		}
		_, err = p.call(ctx, "DELETE", step.Endpoint+"/"+url.PathEscape(step.UpstreamID), nil)
		return err
	}
	return &APIError{Code: "cancel_not_supported", Message: "Tripo V3 cancellation is not exposed by this adapter", HTTPStatus: 409}
}

func str(v any) string            { s, _ := v.(string); return s }
func object(v any) map[string]any { m, _ := v.(map[string]any); return m }

func collectArtifacts(provider string, data map[string]any) []Artifact {
	out := []Artifact{}
	seen := map[string]bool{}
	add := func(role, format, u string, meta map[string]any) {
		if u == "" || seen[role+"|"+u] {
			return
		}
		parsed, err := url.Parse(u)
		if err != nil || parsed.Scheme != "https" {
			return
		}
		seen[role+"|"+u] = true
		if format == "" {
			format = strings.TrimPrefix(strings.ToLower(path.Ext(parsed.Path)), ".")
		}
		mime := "application/octet-stream"
		switch format {
		case "glb":
			mime = "model/gltf-binary"
		case "gltf":
			mime = "model/gltf+json"
		case "png":
			mime = "image/png"
		case "jpg", "jpeg":
			mime = "image/jpeg"
		case "webp":
			mime = "image/webp"
		}
		out = append(out, Artifact{ID: fmt.Sprintf("a%d", len(out)+1), Role: role, Format: format, MediaType: mime, SourceURL: u, Metadata: meta})
	}
	if provider == "meshy" {
		models := object(data["model_urls"])
		keys := []string{}
		for k := range models {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			role := "model"
			if k == "mtl" {
				role = "material"
			}
			add(role, k, str(models[k]), nil)
		}
		if textures, ok := data["texture_urls"].([]any); ok {
			for i, t := range textures {
				m := object(t)
				keys := []string{}
				for k := range m {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					add("texture."+k, "", str(m[k]), map[string]any{"material_index": i})
				}
			}
		}
		result := object(data["result"])
		keys = []string{}
		for k := range result {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			role := "model"
			if strings.Contains(k, "armature") {
				role = "skeleton"
			} else if strings.Contains(k, "animation") || strings.Contains(k, "walking") || strings.Contains(k, "running") {
				role = "animation"
			}
			if strings.HasSuffix(k, "_url") {
				add(role, "", str(result[k]), map[string]any{"provider_field": k})
			}
			if nested := object(result[k]); nested != nil {
				nk := []string{}
				for n := range nested {
					nk = append(nk, n)
				}
				sort.Strings(nk)
				for _, n := range nk {
					add("animation", "", str(nested[n]), map[string]any{"provider_field": k + "." + n})
				}
			}
		}
		add("preview", "", str(data["thumbnail_url"]), nil)
		add("preview", "", str(data["alpha_thumbnail_url"]), map[string]any{"transparent": true})
	} else {
		output := object(data["output"])
		keys := []string{}
		for k := range output {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			u := str(output[k])
			if u == "" {
				u = str(object(output[k])["url"])
			}
			role := "model"
			if strings.Contains(k, "image") || strings.Contains(k, "render") {
				role = "preview"
			}
			if strings.Contains(k, "texture") {
				role = "texture"
			}
			if strings.Contains(k, "animation") {
				role = "animation"
			}
			add(role, "", u, map[string]any{"provider_field": k})
		}
	}
	return out
}

func animationIDs(values []string) ([]int, error) {
	ids := []int{}
	seen := map[int]bool{}
	for _, v := range values {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 || seen[n] {
			return nil, invalid("Meshy animations must be distinct nonnegative action IDs")
		}
		seen[n] = true
		ids = append(ids, n)
	}
	if len(ids) > 10 {
		return nil, invalid("Meshy supports at most 10 actions")
	}
	return ids, nil
}
