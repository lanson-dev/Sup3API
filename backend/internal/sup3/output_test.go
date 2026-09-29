package sup3

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestPortableGLBContractAndExtensions(t *testing.T) {
	for _, provider := range []string{"meshy", "tripo"} {
		r := Request{Provider: provider, Operation: "text_to_3d", Inputs: Inputs{Prompt: "crate"}, Output: &OutputRequirements{Formats: []string{"glb"}, RequiredComponents: []string{"geometry", "materials", "textures"}}}
		e := NewEngine(nil, t.TempDir(), NewProvider(provider, "test"))
		if _, err := e.Prepare(&r); err != nil {
			t.Fatal(err)
		}
		if len(r.Parameters.Formats) != 1 || r.Parameters.Formats[0] != "glb" {
			t.Fatal("format requirement lost")
		}
		j := &Job{Request: r, Artifacts: []Artifact{{Role: "model", Format: "glb"}}, Components: map[string]Component{"geometry": {Status: "available"}, "materials": {Status: "available"}}}
		if validateOutput(j) || j.OutputValidation.Missing[0] != "component:textures" {
			t.Fatal("missing output was accepted")
		}
		j.Components["textures"] = Component{Status: "available"}
		if !validateOutput(j) {
			t.Fatal("complete output rejected")
		}
	}
	r := Request{Provider: "tripo", Operation: "text_to_3d", Inputs: Inputs{Prompt: "crate"}, Extensions: map[string]map[string]json.RawMessage{"tripo": {"geometry_quality": json.RawMessage(`"detailed"`)}}}
	e := NewEngine(nil, t.TempDir(), NewProvider("tripo", "test"))
	if _, err := e.Prepare(&r); err != nil {
		t.Fatal(err)
	}
	if r.Extensions != nil || string(r.ProviderOptions["geometry_quality"]) != `"detailed"` {
		t.Fatal("extension was lost")
	}
}

func TestOutputContractRejectsUnsupportedBeforeSubmission(t *testing.T) {
	for _, body := range []string{
		`{"provider":"tripo","operation":"text_to_3d","inputs":{"prompt":"crate"},"output":{"formats":["fbx"]}}`,
		`{"provider":"tripo","operation":"text_to_3d","inputs":{"prompt":"crate"},"parameters":{"topology":"quad"},"output":{"formats":["glb"]}}`,
		`{"provider":"meshy","operation":"text_to_3d","inputs":{"prompt":"crate"},"output":{"required_components":["animations"]}}`,
		`{"provider":"meshy","operation":"text_to_3d","inputs":{"prompt":"crate"},"parameters":{"texture":false},"output":{"required_components":["textures"]}}`,
		`{"provider":"meshy","operation":"text_to_3d","inputs":{"prompt":"crate"},"parameters":{"formats":["fbx"]},"output":{"formats":["glb"]}}`,
		`{"provider":"tripo","operation":"text_to_3d","inputs":{"prompt":"crate"},"extensions":{"meshy":{"geometry_resolution":"4k"}}}`,
	} {
		var r Request
		_ = json.Unmarshal([]byte(body), &r)
		if _, err := NewEngine(nil, t.TempDir(), NewProvider(r.Provider, "test")).Prepare(&r); err == nil {
			t.Fatalf("accepted incompatible request: %s", body)
		}
	}
}

func TestOutputValidationSurvivesDeliveryRetry(t *testing.T) {
	s := testStore(t)
	p := &fakeProvider{}
	e := NewEngine(s, t.TempDir(), p)
	e.DownloadClient = &http.Client{Transport: fixtureTransport{fixtureGLB(t)}}
	ctx := context.Background()
	j, _, err := e.Create(ctx, 1, 2, "required-textures", Request{Provider: "meshy", Operation: "text_to_3d", Inputs: Inputs{Prompt: "crate"}, Output: &OutputRequirements{Formats: []string{"glb"}, RequiredComponents: []string{"textures"}}})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4; i++ {
		forceDue(t, s)
		if err = e.Tick(ctx); err != nil {
			t.Fatal(err)
		}
	}
	for _, action := range []string{"retry-delivery", "refresh-artifacts"} {
		current, err := s.Get(ctx, j.ID, 1, 2)
		if err != nil {
			t.Fatal(err)
		}
		if current.Status != "succeeded" || current.DeliveryStatus != "failed" || current.Error.Code != "output_requirements_unmet" || current.OutputValidation.Status != "failed" || len(current.Artifacts) == 0 {
			t.Fatalf("unexpected delivery: %+v", current)
		}
		pending, err := e.Mutate(ctx, j.ID, 1, 2, action)
		if err != nil {
			t.Fatal(err)
		}
		if pending.OutputValidation != nil {
			t.Fatal("stale validation on pending delivery")
		}
		forceDue(t, s)
		if err = e.Tick(ctx); err != nil {
			t.Fatal(err)
		}
	}
	final, err := s.Get(ctx, j.ID, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if final.DeliveryStatus != "failed" || p.submits.Load() != 2 {
		t.Fatal("retry bypassed contract or resubmitted paid work")
	}
}

func TestCapabilityOperationConstraints(t *testing.T) {
	c := NewProvider("tripo", "test").Capability()
	rig := c.OperationDetails["rig"].(map[string]any)
	if _, ok := rig["format_conditions"]; ok {
		t.Fatal("rig advertised generation topology")
	}
	fields := rig["extension_fields"].(map[string]any)
	if fields["rig_type"].(map[string]any)["type"] != "string" {
		t.Fatal("extension type missing")
	}
	if _, ok := fields["generate_parts"]; ok {
		t.Fatal("generation option advertised for rig")
	}
}
