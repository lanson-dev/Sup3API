package sup3

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUnifiedInputFormatAliases(t *testing.T) {
	for _, format := range []string{"", "sup3api", "agraphs"} {
		t.Run(format, func(t *testing.T) {
			body := `{"provider":"meshy","operation":"text_to_3d","input_format":"` + format + `","inputs":{"prompt":"a wooden chest"}}`
			r, err := readRequest(httptest.NewRecorder(), httptest.NewRequest("POST", "/v1/assets/quotes", strings.NewReader(body)))
			if err != nil {
				t.Fatal(err)
			}
			if _, err = NewEngine(nil, t.TempDir(), NewProvider("meshy", "test")).Prepare(&r); err != nil {
				t.Fatal(err)
			}
			if r.InputFormat != "" || r.Inputs.Prompt != "a wooden chest" {
				t.Fatal("unified input format changed the request")
			}
		})
	}
}

func TestNativeRequestsReachProviderContracts(t *testing.T) {
	cases := []struct {
		body, endpoint, field string
		value                 any
	}{
		{`{"provider":"tripo","input_format":"tripo","operation":"text_to_3d","payload":{"model":"v3.1-20260211","prompt":"a wooden chest","texture":false,"face_limit":10000}}`, "/generation/text-to-model", "face_limit", 10000},
		{`{"provider":"meshy","input_format":"meshy","operation":"image_to_3d","payload":{"ai_model":"meshy-7.1","image_url":"https://example.com/chest.png","target_polycount":10000,"target_formats":["glb","obj"]}}`, "/openapi/v1/image-to-3d", "image_url", "https://example.com/chest.png"},
		{`{"provider":"meshy","input_format":"meshy","operation":"text_to_3d","payload":{"mode":"preview","prompt":"chest"}}`, "/openapi/v2/text-to-3d", "mode", "preview"},
		{`{"provider":"tripo","input_format":"tripo","operation":"rig","payload":{"input":"https://example.com/chest.glb","spec":"mixamo"}}`, "/animations/rig", "spec", "mixamo"},
	}
	for _, c := range cases {
		r, err := readRequest(httptest.NewRecorder(), httptest.NewRequest("POST", "/v1/assets/quotes", strings.NewReader(c.body)))
		if err != nil {
			t.Fatal(err)
		}
		p := NewProvider(r.Provider, "test")
		e := NewEngine(nil, t.TempDir(), p)
		if _, err = e.Prepare(&r); err != nil {
			t.Fatal(err)
		}
		endpoint, body, err := p.payload(r, stepName(r), "")
		if err != nil || endpoint != c.endpoint || body[c.field] != c.value {
			t.Fatalf("unexpected contract: %s %+v %v", endpoint, body, err)
		}
		if r.InputFormat != "" || r.Payload != nil {
			t.Fatal("native envelope was not normalized")
		}
		if r.Provider == "meshy" && r.Operation == "text_to_3d" && r.Textured() {
			t.Fatal("native preview must not add a paid refine")
		}
	}
}

func TestNativeRejectsLossyOrUnsafeMappings(t *testing.T) {
	for _, body := range []string{
		`{"provider":"tripo","input_format":"meshy","operation":"text_to_3d","payload":{"prompt":"chest"}}`,
		`{"provider":"tripo","input_format":"tripo","operation":"text_to_3d","inputs":{"prompt":"conflict"},"payload":{"prompt":"chest"}}`,
		`{"provider":"tripo","input_format":"tripo","operation":"text_to_3d","payload":{"prompt":"chest","unknown":1}}`,
		`{"provider":"meshy","input_format":"meshy","operation":"text_to_3d","payload":{"mode":"refine","preview_task_id":"another-tenant"}}`,
		`{"provider":"meshy","input_format":"meshy","operation":"rig","payload":{"input_task_id":"another-tenant"}}`,
		`{"provider":"tripo","input_format":"tripo","operation":"text_to_3d","payload":{"prompt":null}}`,
		`{"provider":"tripo","input_format":"tripo","operation":"image_to_3d","payload":{"input":"file_someone_else"}}`,
		`{"provider":"meshy","operation":"rig","inputs":{"model_url":"http://localhost/private.glb"}}`,
	} {
		r, err := readRequest(httptest.NewRecorder(), httptest.NewRequest("POST", "/v1/assets/quotes", strings.NewReader(body)))
		if err == nil {
			_, err = NewEngine(nil, t.TempDir(), NewProvider(r.Provider, "test")).Prepare(&r)
		}
		if err == nil {
			t.Fatalf("accepted unsafe or lossy input: %s", body)
		}
	}
}

func TestMeshyDataURIAndFormatSelection(t *testing.T) {
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	u := "data:image/png;base64," + base64.StdEncoding.EncodeToString(b.Bytes())
	r := Request{Provider: "meshy", Operation: "image_to_3d", Inputs: Inputs{Images: []string{u}}, Parameters: Parameters{Formats: []string{"glb", "obj", "3mf"}}}
	e := NewEngine(nil, t.TempDir(), NewProvider("meshy", "test"))
	if _, err := e.Prepare(&r); err != nil {
		t.Fatal(err)
	}
	_, body, err := NewProvider("meshy", "test").payload(r, "image_to_3d", "")
	if err != nil || len(body["target_formats"].([]string)) != 3 || body["image_url"] != u {
		t.Fatal("image/format information lost")
	}
	for _, bad := range []string{strings.Replace(u, "image/png", "image/jpeg", 1), "data:image/png;base64,aGVsbG8=", "data:image/svg+xml;base64,AAAA"} {
		if validateImageInput("meshy", bad) == nil {
			t.Fatal("accepted invalid image")
		}
	}
	if validateImageInput("tripo", u) == nil {
		t.Fatal("Tripo cannot receive a data URI")
	}
	r.Parameters.Formats = []string{"exe"}
	if _, err := e.Prepare(&r); err == nil {
		t.Fatal("accepted unsupported output format")
	}
}

func TestFullProviderResultsSurvivePersistence(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":"upstream-job","status":"SUCCEEDED","progress":100,"model_urls":{"glb":"https://example.com/a.glb","obj":"https://example.com/a.obj"},"texture_urls":[{"base_color":"https://example.com/a.png"}],"polycount":1234,"consumed_credits":20}`))
	}))
	defer srv.Close()
	p := NewProvider("meshy", "never-return-this-key")
	p.BaseURL = srv.URL
	obs, err := p.Poll(context.Background(), Step{Endpoint: "/task", UpstreamID: "test"})
	if err != nil {
		t.Fatal(err)
	}
	j := &Job{Steps: []Step{{ProviderResult: obs.ProviderResult}}, Artifacts: obs.Artifacts}
	b, err := encode(j)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := decode(b)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Steps[0].ProviderResult["polycount"] != float64(1234) || len(restored.Artifacts) != 3 {
		t.Fatal("provider result lost")
	}
	public, _ := json.Marshal(restored)
	if bytes.Contains(public, []byte(p.Key)) {
		t.Fatal("provider credential exposed")
	}
}
