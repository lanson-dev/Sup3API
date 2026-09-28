package sup3

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func boolp(b bool) *bool { return &b }
func TestOfficialQuotesAndValidation(t *testing.T) {
	cases := []struct {
		name, provider, op, model string
		params                    Parameters
		options                   map[string]json.RawMessage
		want                      float64
		bad                       bool
	}{
		{name: "tripo default", provider: "tripo", op: "text_to_3d", want: 20},
		{name: "tripo bare", provider: "tripo", op: "text_to_3d", params: Parameters{Texture: boolp(false)}, want: 10},
		{name: "tripo HD", provider: "tripo", op: "image_to_3d", options: map[string]json.RawMessage{"texture_quality": json.RawMessage(`"detailed"`)}, want: 40},
		{name: "meshy default", provider: "meshy", op: "text_to_3d", want: 30},
		{name: "meshy lite", provider: "meshy", op: "image_to_3d", model: "meshy-6-lite", want: 15},
		{name: "meshy 8k", provider: "meshy", op: "text_to_3d", params: Parameters{TextureResolution: "8k"}, want: 35},
		{name: "meshy rig", provider: "meshy", op: "rig", want: 5},
		{name: "tripo rig", provider: "tripo", op: "rig", want: 25},
		{name: "meshy two actions", provider: "meshy", op: "animate", params: Parameters{Animations: []string{"0", "1"}}, want: 6},
		{name: "meshy bound is not target", provider: "meshy", op: "text_to_3d", params: Parameters{MaxFaces: 1000}, bad: true},
		{name: "tripo target is not bound", provider: "tripo", op: "text_to_3d", params: Parameters{TargetFaces: 1000}, bad: true},
		{name: "pbr without texture", provider: "meshy", op: "text_to_3d", params: Parameters{Texture: boolp(false), PBR: boolp(true)}, bad: true},
		{name: "ignored option rejected", provider: "meshy", op: "rig", options: map[string]json.RawMessage{"texture_prompt": json.RawMessage(`"blue"`)}, bad: true},
		{name: "wrong option type", provider: "tripo", op: "text_to_3d", options: map[string]json.RawMessage{"generate_parts": json.RawMessage(`"yes"`)}, bad: true},
		{name: "parts textured rejected", provider: "tripo", op: "text_to_3d", options: map[string]json.RawMessage{"generate_parts": json.RawMessage(`true`)}, bad: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := Request{Provider: c.provider, Operation: c.op, Model: c.model, Parameters: c.params, ProviderOptions: c.options}
			switch c.op {
			case "text_to_3d":
				r.Inputs.Prompt = "game prop"
			case "image_to_3d":
				r.Inputs.Images = []string{"https://example.com/a.png"}
			default:
				r.Inputs.JobID = "source"
			}
			q, e := NewProvider(c.provider, "test").Prepare(&r)
			if c.bad {
				if e == nil {
					t.Fatal("expected rejection")
				}
				return
			}
			if e != nil {
				t.Fatal(e)
			}
			if q.Credits != c.want || q.Multiplier != 1 {
				t.Fatalf("quote=%+v", q)
			}
			if c.provider == "meshy" && q.USD != nil {
				t.Fatal("unverified USD conversion")
			}
		})
	}
}
func TestProviderWireContracts(t *testing.T) {
	for _, provider := range []string{"tripo", "meshy"} {
		t.Run(provider, func(t *testing.T) {
			var bodies []map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "Bearer test-secret" {
					t.Error("missing authorization")
				}
				w.Header().Set("Content-Type", "application/json")
				if r.Method == "POST" {
					var b map[string]any
					_ = json.NewDecoder(r.Body).Decode(&b)
					bodies = append(bodies, b)
					if provider == "tripo" {
						if r.URL.Path != "/generation/text-to-model" {
							t.Error(r.URL.Path)
						}
						_, _ = w.Write([]byte(`{"code":0,"data":{"task_id":"t-1"}}`))
					} else {
						_, _ = w.Write([]byte(`{"result":"m-1"}`))
					}
					return
				}
				if provider == "tripo" {
					_, _ = w.Write([]byte(`{"code":0,"data":{"status":"success","credits_consumed":20,"output":{"model_url":"https://cdn.example.com/model.glb"}}}`))
				} else {
					_, _ = w.Write([]byte(`{"status":"SUCCEEDED","consumed_credits":20,"model_urls":{"glb":"https://cdn.example.com/model.glb"},"texture_urls":[{"base_color":"https://cdn.example.com/base.png"}]}`))
				}
			}))
			defer server.Close()
			p := NewProvider(provider, "test-secret")
			p.BaseURL = server.URL
			r := Request{Operation: "text_to_3d", Inputs: Inputs{Prompt: "hero"}}
			_, err := p.Prepare(&r)
			if err != nil {
				t.Fatal(err)
			}
			s, err := p.Submit(context.Background(), r, stepName(r), "")
			if err != nil {
				t.Fatal(err)
			}
			o, err := p.Poll(context.Background(), s)
			if err != nil || o.Status != "succeeded" || o.Credits == nil || *o.Credits != 20 || len(o.Artifacts) == 0 {
				t.Fatalf("observation=%+v err=%v", o, err)
			}
			if provider == "meshy" {
				_, err = p.Submit(context.Background(), r, "refine", s.UpstreamID)
				if err != nil {
					t.Fatal(err)
				}
				if bodies[1]["preview_task_id"] != "m-1" || bodies[1]["enable_pbr"] != true {
					t.Fatal(bodies[1])
				}
			} else {
				if _, ok := bodies[0]["compress"]; ok {
					t.Fatal("unsupported compression enum")
				}
			}
		})
	}
}
func TestAmbiguousPOSTNeverAdvertisesSafeRetry(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(503) }))
	defer s.Close()
	p := NewProvider("meshy", "key")
	p.BaseURL = s.URL
	_, err := p.Submit(context.Background(), Request{Operation: "rig"}, "rig", "")
	a := apiError(err)
	if !a.Uncertain || a.Retryable {
		t.Fatalf("%+v", a)
	}
}
func TestArtifactCollectorNestedRig(t *testing.T) {
	data := map[string]any{"result": map[string]any{"rigged_character_glb_url": "https://cdn.example.com/rig.glb", "basic_animations": map[string]any{"walking_glb_url": "https://cdn.example.com/walk.glb"}}}
	a := collectArtifacts("meshy", data)
	if len(a) != 2 {
		t.Fatalf("%+v", a)
	}
}

func TestCancelDoesNotDeleteCompletedResults(t *testing.T) {
	deletes := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" {
			deletes++
			t.Error("deleted a completed task")
		}
		_, _ = w.Write([]byte(`{"status":"SUCCEEDED"}`))
	}))
	defer s.Close()
	p := NewProvider("meshy", "test")
	p.BaseURL = s.URL
	err := p.Cancel(context.Background(), Step{Endpoint: "/openapi/v1/rigging", UpstreamID: "id"})
	if err == nil || apiError(err).HTTPStatus != 409 || deletes != 0 {
		t.Fatal("unsafe cancellation", err)
	}
}

func TestMultiviewAndRigPayloadBoundaries(t *testing.T) {
	for _, provider := range []string{"tripo", "meshy"} {
		p := NewProvider(provider, "test")
		r := Request{Operation: "multi_image_to_3d", Inputs: Inputs{Images: []string{"https://example.com/front.png", "https://example.com/left.png", "", ""}}}
		if provider == "meshy" {
			r.Inputs.Images = r.Inputs.Images[:2]
		}
		if _, err := p.Prepare(&r); err != nil {
			t.Fatal(err)
		}
		endpoint, body, err := p.payload(r, stepName(r), "")
		if err != nil {
			t.Fatal(err)
		}
		if provider == "tripo" {
			if endpoint != "/generation/multiview-to-model" || len(body["inputs"].([]string)) != 4 {
				t.Fatal(body)
			}
		} else {
			if endpoint != "/openapi/v1/multi-image-to-3d" || len(body["image_urls"].([]string)) != 2 {
				t.Fatal(body)
			}
		}
		r = Request{Operation: "rig", Inputs: Inputs{JobID: "local-job", UpstreamID: "upstream-job"}}
		if _, err := p.Prepare(&r); err != nil {
			t.Fatal(err)
		}
		_, body, err = p.payload(r, "rig", "")
		if err != nil {
			t.Fatal(err)
		}
		if provider == "tripo" && (body["input"] != "upstream-job" || body["model"] != "v1.0-20240301") {
			t.Fatal(body)
		}
		if provider == "meshy" && body["input_task_id"] != "upstream-job" {
			t.Fatal(body)
		}
	}
}
