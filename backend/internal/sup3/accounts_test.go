package sup3

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestManagedNativeAccountsPinQueriesReferencesAndReplay(t *testing.T) {
	store := testStore(t)
	posts := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer first" {
			t.Error("existing task routed to another account")
		}
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "POST" {
			posts++
			fmt.Fprintf(w, `{"result":"task-%d"}`, posts)
		} else {
			fmt.Fprint(w, `{"status":"SUCCEEDED"}`)
		}
	}))
	defer upstream.Close()
	p := NewProvider("meshy", "first")
	p.BaseURL = upstream.URL
	second := NewProvider("meshy", "second")
	second.BaseURL = upstream.URL
	firstBinding, secondBinding := AccountBinding(1, p.Key), AccountBinding(2, second.Key)
	selected := firstBinding
	disabled := false
	e := NewEngine(store, t.TempDir(), NewProvider("meshy", ""))
	e.ResolveProvider = func(_ context.Context, provider, binding string) (Provider, string, error) {
		if binding == "" {
			binding = selected
		}
		if binding == firstBinding && !disabled {
			return p, binding, nil
		}
		if binding == secondBinding {
			return second, binding, nil
		}
		return nil, "", &APIError{Code: "provider_account_unavailable", HTTPStatus: 503}
	}
	path := "/providers/meshy/openapi/v2/text-to-3d"
	body := `{"mode":"preview","prompt":"crate"}`
	if w := nativeRequest(e, "POST", path, body, "managed-preview", 1, 10); w.Code != 200 {
		t.Fatal(w.Body)
	}
	selected = secondBinding
	if w := nativeRequest(e, "POST", path, body, "managed-preview", 1, 10); w.Code != 200 || posts != 1 {
		t.Fatal("replay resubmitted", w.Body)
	}
	if w := nativeRequest(e, "GET", path+"/task-1", "", "", 1, 10); w.Code != 200 {
		t.Fatal(w.Body)
	}
	if w := nativeRequest(e, "POST", path, `{"mode":"refine","preview_task_id":"task-1"}`, "managed-refine", 1, 10); w.Code != 200 || posts != 2 {
		t.Fatal(w.Body)
	}
	if w := nativeRequest(e, "GET", path+"/task-1", "", "", 1, 11); w.Code != 404 {
		t.Fatal("foreign key accessed task")
	}
	disabled = true
	if w := nativeRequest(e, "GET", path+"/task-1", "", "", 1, 10); w.Code != 503 {
		t.Fatal("disabled account failed over", w.Body)
	}
	if w := nativeRequest(e, "POST", path, `{"mode":"refine","preview_task_id":"task-1"}`, "new-refinement", 1, 10); w.Code != 503 || posts != 2 {
		t.Fatal("disabled reference resubmitted", w.Body)
	}
}

func TestManagedUnifiedStagesPersistBinding(t *testing.T) {
	store := testStore(t)
	first, second := &fakeProvider{}, &fakeProvider{}
	selected := "1:first"
	e := NewEngine(store, t.TempDir(), first)
	e.ResolveProvider = func(_ context.Context, _ string, binding string) (Provider, string, error) {
		if binding == "" {
			binding = selected
		}
		if binding == "1:first" {
			return first, binding, nil
		}
		return second, binding, nil
	}
	ctx := context.Background()
	j, _, err := e.Create(ctx, 1, 2, "managed-unified", Request{Provider: "meshy", Operation: "text_to_3d", Inputs: Inputs{Prompt: "crate"}})
	if err != nil {
		t.Fatal(err)
	}
	public, _ := json.Marshal(j)
	if strings.Contains(string(public), "1:first") {
		t.Fatal("private binding exposed")
	}
	selected = "2:second"
	for i := 0; i < 2; i++ {
		forceDue(t, store)
		if err := e.Tick(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if first.submits.Load() != 2 || second.submits.Load() != 0 {
		t.Fatal("preview/refine switched accounts")
	}
	loaded, err := store.Get(ctx, j.ID, 1, 2)
	if err != nil || loaded.AccountBinding != "1:first" {
		t.Fatal("binding not durable", err)
	}
}

func TestManagedQueuedJobCanCancelAfterAccountDisabled(t *testing.T) {
	store := testStore(t)
	p := &fakeProvider{}
	e := NewEngine(store, t.TempDir(), p)
	enabled := true
	e.ResolveProvider = func(context.Context, string, string) (Provider, string, error) {
		if enabled {
			return p, "1:original", nil
		}
		return nil, "", &APIError{Code: "provider_account_unavailable", HTTPStatus: 503}
	}
	ctx := context.Background()
	j, _, err := e.Create(ctx, 1, 2, "cancel-queued-job", Request{Provider: "meshy", Operation: "text_to_3d", Inputs: Inputs{Prompt: "crate"}})
	if err != nil {
		t.Fatal(err)
	}
	enabled = false
	j, err = e.Mutate(ctx, j.ID, 1, 2, "cancel")
	if err != nil || j.Status != "canceled" || p.submits.Load() != 0 {
		t.Fatal("local cancellation must not need upstream access", err)
	}
}

func TestManagedModelMappingIsPinnedAndNativePayloadIsMapped(t *testing.T) {
	store := testStore(t)
	model := "meshy-6"
	p := NewProvider("meshy", "private")
	p.ResolveModel = func(string) string { return model }
	e := NewEngine(store, t.TempDir(), NewProvider("meshy", ""))
	e.ResolveProvider = func(ctx context.Context, provider, binding string) (Provider, string, error) {
		if requested := RoutingFromContext(ctx).Model; requested != "" && requested != "asset-fast" {
			return nil, "", &APIError{Code: "model_not_allowed", HTTPStatus: 403}
		}
		return p, "1:pinned", nil
	}
	r := Request{Provider: "meshy", Model: "asset-fast", Operation: "text_to_3d", Inputs: Inputs{Prompt: "crate"}}
	job, _, err := e.Create(context.Background(), 1, 2, "alias-request", r)
	if err != nil || job.Request.Model != "asset-fast" || job.ResolvedModel != "meshy-6" {
		t.Fatal(job, err)
	}
	model = "meshy-7.1"
	replay, created, err := e.Create(context.Background(), 1, 2, "alias-request", r)
	if err != nil || created || replay.ResolvedModel != "meshy-6" {
		t.Fatal("mapping changed an existing job", err)
	}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if json.NewDecoder(r.Body).Decode(&body) != nil || body["ai_model"] != "meshy-7.1" || body["prompt"] != "crate" {
			t.Error("wrong native model mapping", body)
		}
		fmt.Fprint(w, `{"result":"native-alias-task"}`)
	}))
	defer upstream.Close()
	p.BaseURL = upstream.URL
	w := nativeRequest(e, "POST", "/providers/meshy/openapi/v2/text-to-3d", `{"ai_model":"asset-fast","prompt":"crate"}`, "native-alias-request", 1, 2)
	if w.Code != 200 {
		t.Fatal(w.Body)
	}
	w = nativeRequest(e, "POST", "/providers/meshy/openapi/v2/text-to-3d", `{"ai_model":"blocked","prompt":"crate"}`, "native-blocked-model", 1, 2)
	if w.Code != 403 {
		t.Fatal("native whitelist bypass", w.Body)
	}
}
