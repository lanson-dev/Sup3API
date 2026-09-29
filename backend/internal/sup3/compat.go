package sup3

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Explicit API versions, not an open reverse proxy. Native requests retain their
// defaults and extra provider fields; they never pass through the unified mapper.
var nativeEndpoints = map[string][]string{
	"meshy": {"/openapi/v2/text-to-3d", "/openapi/v1/image-to-3d", "/openapi/v1/multi-image-to-3d", "/openapi/v1/retexture", "/openapi/v1/rigging", "/openapi/v1/animations", "/openapi/v1/convert"},
	"tripo": {"/generation/text-to-model", "/generation/image-to-model", "/generation/multiview-to-model", "/models/texture", "/animations/rig", "/animations/retarget", "/models/convert"},
}
var nativeID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,160}$`)

func nativeCompatibility(provider string) map[string]any {
	base := "/providers/" + provider
	if provider == "tripo" {
		base += "/v3"
	}
	paths := append([]string{}, nativeEndpoints[provider]...)
	sort.Strings(paths)
	return map[string]any{"base_url": base, "create_paths": paths, "task_query": true, "delete": provider == "meshy", "sse": provider == "meshy", "list": false, "file_upload": false, "webhooks": false, "historical_tasks": false, "optional_idempotency": true, "response": "provider_native", "billing": "provider_native_credits", "artifact_retention": "provider_managed"}
}

func nativeRoute(provider, method, path string) (collection, task string, stream bool) {
	if provider == "tripo" && method == "GET" && strings.HasPrefix(path, "/tasks/") {
		id := strings.TrimPrefix(path, "/tasks/")
		if nativeID.MatchString(id) {
			return "/tasks", id, false
		}
		return
	}
	for _, endpoint := range nativeEndpoints[provider] {
		if path == endpoint && method == "POST" {
			return endpoint, "", false
		}
		if provider == "meshy" && (method == "GET" || method == "DELETE") && strings.HasPrefix(path, endpoint+"/") {
			id := strings.TrimPrefix(path, endpoint+"/")
			if strings.HasSuffix(id, "/stream") && method == "GET" {
				id = strings.TrimSuffix(id, "/stream")
				stream = true
			}
			if nativeID.MatchString(id) {
				return endpoint, id, stream
			}
		}
	}
	return "", "", false
}

func nativeError(w http.ResponseWriter, err error) {
	w.Header().Set("X-Sup3-Error-Origin", "gateway")
	replyError(w, err)
}

// Check resource references recursively, including new provider *_task_id fields.
// Prompts are never interpreted as IDs. Provider fields are not mapped to the
// unified schema; JSON is canonicalized so duplicate keys cannot bypass checks.
func (e *Engine) checkNativeReferences(ctx context.Context, scope nativeScope, value any, field string) error {
	field = strings.ToLower(field)
	taskReference := strings.HasSuffix(field, "task_id") || strings.HasSuffix(field, "task_ids")
	resource := taskReference || field == "input" || field == "inputs" || strings.Contains(field, "file_token")
	if resource {
		switch value.(type) {
		case string:
		case []any:
			if field != "inputs" && !strings.HasSuffix(field, "task_ids") {
				return invalid("native resource reference must be a string")
			}
		default:
			return invalid("native resource references must be strings or supported arrays of strings")
		}
	}
	switch v := value.(type) {
	case map[string]any:
		for k, item := range v {
			if strings.Contains(strings.ToLower(k), "webhook") || strings.Contains(strings.ToLower(k), "callback") {
				return invalid("native webhooks are not supported; use task polling or Meshy SSE")
			}
			if err := e.checkNativeReferences(ctx, scope, item, k); err != nil {
				return err
			}
		}
	case []any:
		for _, item := range v {
			if resource {
				if _, ok := item.(string); !ok {
					return invalid("native resource arrays must contain strings")
				}
			}
			if err := e.checkNativeReferences(ctx, scope, item, field); err != nil {
				return err
			}
		}
	case string:
		reference := resource
		if (strings.HasPrefix(v, "task_") || strings.HasPrefix(v, "file_")) && field != "prompt" && field != "text" && field != "negative_prompt" {
			reference = true
		}
		if !reference || v == "" {
			return nil
		}
		if strings.Contains(field, "file_token") || strings.HasPrefix(v, "file_") {
			return invalid("native file tokens are not supported; use public HTTPS inputs")
		}
		if !taskReference && strings.HasPrefix(v, "https://") {
			return validRemoteURL(v)
		}
		_, err := e.Store.nativeTask(ctx, scope, v)
		return err
	}
	return nil
}

func (e *Engine) ServeNativeHTTP(w http.ResponseWriter, r *http.Request, owner, key int64) {
	parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/providers/"), "/", 2)
	if len(parts) != 2 {
		nativeError(w, invalid("missing native provider path"))
		return
	}
	provider, path := parts[0], "/"+parts[1]
	p, ok := e.Providers[provider].(*RemoteProvider)
	if !ok {
		nativeError(w, invalid("unknown native provider"))
		return
	}
	if provider == "tripo" {
		if !strings.HasPrefix(path, "/v3/") {
			nativeError(w, invalid("Tripo native compatibility requires /v3"))
			return
		}
		path = strings.TrimPrefix(path, "/v3")
	}
	endpoint, task, stream := nativeRoute(provider, r.Method, path)
	if endpoint == "" || r.URL.RawQuery != "" || r.URL.RawPath != "" {
		nativeError(w, &APIError{Code: "native_endpoint_unsupported", Message: "unsupported native endpoint, method, or query; see capabilities.native_api", HTTPStatus: 404})
		return
	}
	if p.Key == "" {
		nativeError(w, &APIError{Code: "provider_not_configured", Message: "provider credentials are missing", HTTPStatus: 503})
		return
	}
	digest := sha256.Sum256([]byte(p.Key))
	scope := nativeScope{Owner: owner, Key: key, Provider: provider, Credential: hex.EncodeToString(digest[:])}
	if task != "" {
		original, err := e.Store.nativeTask(r.Context(), scope, task)
		if err != nil {
			nativeError(w, err)
			return
		}
		if provider == "meshy" && original != endpoint {
			nativeError(w, &APIError{Code: "native_task_not_found", Message: "task belongs to a different native endpoint", HTTPStatus: 404})
			return
		}
	}
	var body []byte
	var call nativeCall
	if r.Method == "POST" {
		media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || media != "application/json" {
			nativeError(w, invalid("native create endpoints require application/json"))
			return
		}
		body, err = io.ReadAll(http.MaxBytesReader(w, r.Body, 16<<20))
		if err != nil {
			nativeError(w, invalid("native request exceeds 16 MiB"))
			return
		}
		var payload map[string]any
		decoder := json.NewDecoder(bytes.NewReader(body))
		decoder.UseNumber()
		if decoder.Decode(&payload) != nil || payload == nil {
			nativeError(w, invalid("native request must be a JSON object"))
			return
		}
		var trailing any
		if decoder.Decode(&trailing) != io.EOF {
			nativeError(w, invalid("native request must contain one JSON object"))
			return
		}
		if err = e.checkNativeReferences(r.Context(), scope, payload, ""); err != nil {
			nativeError(w, err)
			return
		}
		body, err = json.Marshal(payload)
		if err != nil {
			nativeError(w, invalid("invalid native JSON"))
			return
		}
		idem := r.Header.Get("Idempotency-Key")
		if idem != "" && (len(idem) < 8 || len(idem) > 128 || strings.TrimSpace(idem) != idem) {
			nativeError(w, invalid("Idempotency-Key must be 8-128 characters when supplied"))
			return
		}
		hash := sha256.Sum256(append([]byte(endpoint+"\n"), body...))
		var created bool
		call, created, err = e.Store.reserveNative(r.Context(), scope, idem, hex.EncodeToString(hash[:]), endpoint)
		if err != nil {
			nativeError(w, err)
			return
		}
		w.Header().Set("X-Sup3-Request-ID", call.ID)
		if !created {
			if call.Hash != hex.EncodeToString(hash[:]) {
				nativeError(w, &APIError{Code: "idempotency_conflict", Message: "native idempotency key was used with different request bytes", HTTPStatus: 409})
				return
			}
			if call.Status == 0 {
				nativeError(w, &APIError{Code: "submission_unknown", Message: "native submission is pending or uncertain; do not resubmit with a new key", HTTPStatus: 409})
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store")
			w.WriteHeader(call.Status)
			_, _ = w.Write(call.Body)
			return
		}
	}
	request, err := http.NewRequestWithContext(r.Context(), r.Method, p.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		nativeError(w, err)
		return
	}
	request.Header.Set("Authorization", "Bearer "+p.Key)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	if stream {
		request.Header.Set("Accept", "text/event-stream")
	}
	client := *p.Client
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	if stream {
		client.Timeout = 30 * time.Minute
	}
	response, err := client.Do(request)
	if err != nil {
		nativeError(w, &APIError{Code: "upstream_transport_error", Message: "native upstream did not return a response; POST outcome may be unknown", HTTPStatus: 502, Uncertain: r.Method == "POST", Retryable: r.Method == "GET"})
		return
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 && response.StatusCode < 400 {
		nativeError(w, &APIError{Code: "upstream_redirect_rejected", Message: "native upstream redirects are not followed", HTTPStatus: 502})
		return
	}
	if stream && response.StatusCode == 200 && strings.HasPrefix(response.Header.Get("Content-Type"), "text/event-stream") {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Accel-Buffering", "no")
		w.WriteHeader(200)
		buffer := make([]byte, 32<<10)
		for {
			n, readErr := response.Body.Read(buffer)
			if n > 0 {
				if _, err = w.Write(buffer[:n]); err != nil {
					return
				}
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
			}
			if readErr != nil {
				return
			}
		}
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, (16<<20)+1))
	if err != nil || len(data) > 16<<20 {
		nativeError(w, &APIError{Code: "upstream_invalid_response", Message: "native response is incomplete or too large", HTTPStatus: 502})
		return
	}
	if r.Method == "POST" {
		var result map[string]any
		parseErr := json.Unmarshal(data, &result)
		id := ""
		if response.StatusCode >= 200 && response.StatusCode < 300 {
			if parseErr != nil || result == nil {
				nativeError(w, &APIError{Code: "submission_unknown", Message: "native upstream returned invalid task JSON", HTTPStatus: 502})
				return
			}
			if provider == "tripo" {
				if _, ok := result["code"].(float64); !ok {
					nativeError(w, &APIError{Code: "submission_unknown", Message: "native Tripo response has no result code", HTTPStatus: 502})
					return
				}
			}
			if provider == "meshy" {
				id = str(result["result"])
			} else if code, ok := result["code"].(float64); ok && code == 0 {
				id = str(object(result["data"])["task_id"])
			}
			accepted := provider == "meshy" || result["code"] == float64(0)
			if accepted && !nativeID.MatchString(id) {
				nativeError(w, &APIError{Code: "submission_unknown", Message: "native upstream accepted the request without a usable task ID", HTTPStatus: 502})
				return
			}
		}
		// A canceled client must not prevent ownership from being persisted.
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err = e.Store.finishNative(ctx, call.ID, id, response.StatusCode, data)
		cancel()
		if err != nil {
			nativeError(w, &APIError{Code: "submission_unknown", Message: "native response could not be persisted; reconcile before retrying", HTTPStatus: 502})
			return
		}
	}
	for _, header := range []string{"Content-Type", "Retry-After", "X-Request-ID"} {
		if value := response.Header.Get(header); value != "" {
			w.Header().Set(header, value)
		}
	}
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(response.StatusCode)
	_, _ = w.Write(data)
}
