package sup3

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func nativeRequest(e *Engine, method, path, body, idem string, owner, key int64) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Idempotency-Key", idem)
	w := httptest.NewRecorder()
	e.ServeNativeHTTP(w, r, owner, key)
	return w
}

func TestNativeRouteBoundary(t *testing.T) {
	for _, path := range []string{"/openapi/v2/text-to-3d", "/openapi/v2/text-to-3d/../../balance", "/openapi/v2/text-to-3d/other/tasks", "/openapi/v1/balance"} {
		endpoint, _, _ := nativeRoute("meshy", "GET", path)
		if endpoint != "" {
			t.Fatalf("unexpected route %s", path)
		}
	}
	endpoint, id, stream := nativeRoute("meshy", "GET", "/openapi/v2/text-to-3d/abc-123/stream")
	if endpoint != "/openapi/v2/text-to-3d" || id != "abc-123" || !stream {
		t.Fatal("SSE route not recognized")
	}
}

func TestNativeWorkflowOwnershipReplayAndUnmappedFields(t *testing.T) {
	store := testStore(t)
	var posts, requests atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Header.Get("Authorization") != "Bearer provider-secret" {
			t.Error("gateway key leaked or upstream key missing")
		}
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "POST" {
			n := posts.Add(1)
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["mode"] == "preview" {
				if body["alpha_thumbnail"] != true || body["future_feature"] != "preserved" {
					t.Error("native fields were lost")
				}
				if _, ok := body["enable_pbr"]; ok {
					t.Error("unified default leaked into native request")
				}
				_, _ = w.Write([]byte(`{"result":"preview-task","new_field":{"ok":true}}`))
				return
			}
			if body["mode"] != "refine" || body["preview_task_id"] != "preview-task" || n != 2 {
				t.Error("preview/refine workflow changed")
			}
			_, _ = w.Write([]byte(`{"result":"refine-task"}`))
			return
		}
		if strings.HasSuffix(r.URL.Path, "/stream") {
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("event: message\ndata: {\"status\":\"SUCCEEDED\"}\n\n"))
			return
		}
		if r.Method == "DELETE" {
			w.WriteHeader(409)
			_, _ = w.Write([]byte(`{"message":"Task is IN_PROGRESS"}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"refine-task","status":"SUCCEEDED","model_urls":{"glb":"https://example.com/model.glb"},"new_field":7}`))
	}))
	defer upstream.Close()
	p := NewProvider("meshy", "provider-secret")
	p.BaseURL = upstream.URL
	e := NewEngine(store, t.TempDir(), p)
	path := "/providers/meshy/openapi/v2/text-to-3d"
	body := `{"mode":"preview","prompt":"crate","alpha_thumbnail":true,"future_feature":"preserved"}`
	created := nativeRequest(e, "POST", path, body, "request-preview", 1, 10)
	if created.Code != 200 || !strings.Contains(created.Body.String(), `"new_field"`) {
		t.Fatalf("create: %d %s", created.Code, created.Body)
	}
	replayed := nativeRequest(e, "POST", path, body, "request-preview", 1, 10)
	if replayed.Body.String() != created.Body.String() || posts.Load() != 1 {
		t.Fatal("native submission replay charged twice")
	}
	conflict := nativeRequest(e, "POST", path, `{"mode":"preview","prompt":"different"}`, "request-preview", 1, 10)
	if conflict.Code != 409 || posts.Load() != 1 {
		t.Fatal("conflicting idempotency key reached upstream")
	}
	for _, identity := range [][2]int64{{2, 10}, {1, 11}} {
		if nativeRequest(e, "GET", path+"/preview-task", "", "", identity[0], identity[1]).Code != 404 {
			t.Fatal("cross-tenant task disclosed")
		}
		if nativeRequest(e, "POST", path, `{"mode":"refine","preview_task_id":"preview-task"}`, "", identity[0], identity[1]).Code != 404 {
			t.Fatal("cross-tenant reference allowed")
		}
	}
	if posts.Load() != 1 {
		t.Fatal("foreign references caused submission")
	}
	refine := nativeRequest(e, "POST", path, `{"mode":"refine","preview_task_id":"preview-task"}`, "", 1, 10)
	if refine.Code != 200 || posts.Load() != 2 {
		t.Fatalf("refine: %d %s", refine.Code, refine.Body)
	}
	query := nativeRequest(e, "GET", path+"/refine-task", "", "", 1, 10)
	if query.Code != 200 || !strings.Contains(query.Body.String(), `"new_field":7`) {
		t.Fatal("native response changed")
	}
	if nativeRequest(e, "GET", path+"/refine-task/stream", "", "", 1, 10).Body.String() != "event: message\ndata: {\"status\":\"SUCCEEDED\"}\n\n" {
		t.Fatal("SSE not preserved")
	}
	deletion := nativeRequest(e, "DELETE", path+"/refine-task", "", "", 1, 10)
	if deletion.Code != 409 || deletion.Body.String() != `{"message":"Task is IN_PROGRESS"}` {
		t.Fatal("native error changed")
	}
	before := requests.Load()
	if nativeRequest(e, "GET", "/providers/meshy/openapi/v1/retexture/refine-task", "", "", 1, 10).Code != 404 {
		t.Fatal("wrong task endpoint allowed")
	}
	if nativeRequest(e, "GET", path, "", "", 1, 10).Code != 404 {
		t.Fatal("operator task listing exposed")
	}
	p.Key = "rotated-account-key"
	if nativeRequest(e, "GET", path+"/refine-task", "", "", 1, 10).Code != 404 || requests.Load() != before {
		t.Fatal("task routed to a different provider account")
	}
}

func TestNativeAmbiguousSubmissionDoesNotRetry(t *testing.T) {
	store := testStore(t)
	var posts atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		posts.Add(1)
		_, _ = io.WriteString(w, `{"code":0,"data":{}}`)
	}))
	defer upstream.Close()
	p := NewProvider("tripo", "secret")
	p.BaseURL = upstream.URL
	e := NewEngine(store, t.TempDir(), p)
	path := "/providers/tripo/v3/generation/text-to-model"
	if nativeRequest(e, "POST", path, `{"prompt":"crate"}`, "ambiguous-request", 1, 1).Code != 502 {
		t.Fatal("missing task ID did not fail")
	}
	if nativeRequest(e, "POST", path, `{"prompt":"crate"}`, "ambiguous-request", 1, 1).Code != 409 || posts.Load() != 1 {
		t.Fatal("uncertain request retried")
	}
	var count int
	if err := store.DB.QueryRowContext(context.Background(), "SELECT count(*) FROM sup3_native_calls WHERE status=0").Scan(&count); err != nil || count != 1 {
		t.Fatal("submission intent was not durable")
	}
}

func TestNativeReferenceShapesRejectedBeforeUpstream(t *testing.T) {
	e := NewEngine(nil, t.TempDir())
	for _, body := range []string{
		`{"input":{"url":"task_other"}}`, `{"inputs":[{"id":"task_other"}]}`,
		`{"preview_task_id":["task_other"]}`, `{"task_ids":[12]}`,
		`{"input":12}`, `{"file_token":"file_other"}`, `{"webhook_url":"https://example.com/hook"}`,
	} {
		var payload any
		if err := json.Unmarshal([]byte(body), &payload); err != nil {
			t.Fatal(err)
		}
		if err := e.checkNativeReferences(context.Background(), &nativeScope{}, payload, ""); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
	if err := e.checkNativeReferences(context.Background(), &nativeScope{}, map[string]any{"inputs": []any{"https://example.com/a.png", ""}, "prompt": "task_in_prompt"}, ""); err != nil {
		t.Fatal(err)
	}
}
