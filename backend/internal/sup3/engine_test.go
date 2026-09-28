package sup3

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	dsn := os.Getenv("SUP3_TEST_DSN")
	if dsn == "" {
		t.Skip("set SUP3_TEST_DSN to run PostgreSQL durability tests")
	}
	db, e := sql.Open("postgres", dsn)
	if e != nil {
		t.Fatal(e)
	}
	schema := "sup3_test_" + randomID()
	if _, e = db.Exec("CREATE SCHEMA " + schema); e != nil {
		t.Fatal(e)
	}
	s, e := OpenStore(context.Background(), dsn+" search_path="+schema)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.DB.Close(); _, _ = db.Exec("DROP SCHEMA " + schema + " CASCADE"); db.Close() })
	return s
}

type fakeProvider struct {
	submits   atomic.Int32
	ambiguous bool
}

func (p *fakeProvider) Name() string           { return "meshy" }
func (p *fakeProvider) Capability() Capability { return Capability{Provider: "meshy", Available: true} }
func (p *fakeProvider) Prepare(r *Request) (Price, error) {
	return NewProvider("meshy", "fake").Prepare(r)
}
func (p *fakeProvider) Submit(_ context.Context, _ Request, stage, previous string) (Step, error) {
	p.submits.Add(1)
	if p.ambiguous {
		return Step{}, &APIError{Code: "timeout", Uncertain: true}
	}
	return Step{Name: stage, UpstreamID: "up-" + stage, Endpoint: "/test", Status: "queued"}, nil
}
func (p *fakeProvider) Poll(_ context.Context, s Step) (Observation, error) {
	n := 20.0
	if s.Name == "refine" {
		n = 10
	}
	return Observation{Status: "succeeded", Progress: 100, Credits: &n, Artifacts: []Artifact{{ID: "a1", Role: "model", Format: "glb", SourceURL: "https://assets.example.com/model.glb"}}}, nil
}
func (p *fakeProvider) Cancel(context.Context, Step) error              { return nil }
func (p *fakeProvider) Balance(context.Context) (map[string]any, error) { return nil, nil }
func forceDue(t *testing.T, s *Store) {
	t.Helper()
	if _, e := s.DB.Exec("UPDATE sup3_jobs SET next_run=now(),lease_until=now()"); e != nil {
		t.Fatal(e)
	}
}
func TestDurableIdempotencyOwnershipAndPreviewRefine(t *testing.T) {
	s := testStore(t)
	p := &fakeProvider{}
	e := NewEngine(s, t.TempDir(), p)
	ctx := context.Background()
	r := Request{Provider: "meshy", Operation: "text_to_3d", Inputs: Inputs{Prompt: "hero"}}
	var wg sync.WaitGroup
	ids := make(chan string, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			j, _, err := e.Create(ctx, 1, 2, "same-request", r)
			if err != nil {
				t.Error(err)
				return
			}
			ids <- j.ID
		}()
	}
	wg.Wait()
	close(ids)
	id := ""
	for v := range ids {
		if id != "" && id != v {
			t.Fatal("duplicate idempotent job")
		}
		id = v
	}
	if _, err := s.Get(ctx, id, 1, 3); apiError(err).HTTPStatus != 404 {
		t.Fatal("key isolation failure")
	}
	if _, err := s.Get(ctx, id, 2, 2); apiError(err).HTTPStatus != 404 {
		t.Fatal("owner isolation failure")
	}
	r.Inputs.Prompt = "different"
	if _, _, err := e.Create(ctx, 1, 2, "same-request", r); apiError(err).HTTPStatus != 409 {
		t.Fatal("idempotency conflict accepted")
	}
	if err := e.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	forceDue(t, s)
	// A fresh Engine (process restart) picks up the persisted preview endpoint/ID.
	restarted := NewEngine(s, t.TempDir(), p)
	if err := restarted.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	forceDue(t, s)
	if err := restarted.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	j, err := s.Get(ctx, id, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if j.Status != "succeeded" || len(j.Steps) != 2 || j.Cost.Credits != 30 || j.Cost.Kind != "reported" || p.submits.Load() != 2 {
		t.Fatalf("job=%+v submits=%d", j, p.submits.Load())
	}
	encoded, _ := json.Marshal(j)
	if strings.Contains(string(encoded), "assets.example") || strings.Contains(string(encoded), "owner_id") {
		t.Fatal("private metadata leaked")
	}
}
func TestCrashDuringSubmissionNeverReplaysPOST(t *testing.T) {
	s := testStore(t)
	p := &fakeProvider{}
	e := NewEngine(s, t.TempDir(), p)
	ctx := context.Background()
	j, _, err := e.Create(ctx, 1, 1, "crash-request", Request{Provider: "meshy", Operation: "text_to_3d", Inputs: Inputs{Prompt: "prop"}})
	if err != nil {
		t.Fatal(err)
	}
	claimed, token, err := s.Claim(ctx, j.ID)
	if err != nil {
		t.Fatal(err)
	}
	claimed.Status = "submitting"
	if err = s.Save(ctx, claimed, token, false); err != nil {
		t.Fatal(err)
	}
	forceDue(t, s)
	if err = e.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	j, _ = s.Get(ctx, j.ID, 1, 1)
	if j.Status != "submission_unknown" || p.submits.Load() != 0 {
		t.Fatal("unsafe POST replay")
	}
	if err = s.Save(ctx, claimed, token, true); err == nil {
		t.Fatal("stale worker was allowed to overwrite newer state")
	}
}
func TestAmbiguousSubmitPersistsUnknown(t *testing.T) {
	s := testStore(t)
	p := &fakeProvider{ambiguous: true}
	e := NewEngine(s, t.TempDir(), p)
	j, _, err := e.Create(context.Background(), 1, 1, "ambiguous-request", Request{Provider: "meshy", Operation: "text_to_3d", Inputs: Inputs{Prompt: "prop"}})
	if err != nil {
		t.Fatal(err)
	}
	if err = e.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	j, _ = s.Get(context.Background(), j.ID, 1, 1)
	if j.Status != "submission_unknown" {
		t.Fatal(j.Status)
	}
	forceDue(t, s)
	_ = e.Tick(context.Background())
	if p.submits.Load() != 1 {
		t.Fatal("ambiguous request repeated")
	}
}
func TestHTTPRejectsUnknownFieldsAndExposesNoCredential(t *testing.T) {
	e := NewEngine(nil, t.TempDir(), NewProvider("meshy", "must-not-leak"))
	for _, body := range []string{`{"provider":"meshy","unknown":1}`, `{} {}`} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/v1/assets/quotes", strings.NewReader(body))
		e.ServeHTTP(w, r, 1, 1)
		if w.Code != 400 {
			t.Fatal(w.Code)
		}
	}
	w := httptest.NewRecorder()
	e.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/v1/assets/capabilities", nil), 1, 1)
	if strings.Contains(w.Body.String(), "must-not-leak") {
		t.Fatal("key leaked")
	}
}
