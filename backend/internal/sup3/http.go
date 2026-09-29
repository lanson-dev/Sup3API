package sup3

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func reply(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func replyError(w http.ResponseWriter, err error) {
	a := apiError(err)
	status := a.HTTPStatus
	if status == 0 {
		status = 500
	}
	reply(w, status, map[string]any{"error": a})
}
func readRequest(w http.ResponseWriter, r *http.Request) (Request, error) {
	var req Request
	r.Body = http.MaxBytesReader(w, r.Body, 16<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(&req); err != nil {
		return req, invalid("request must be valid JSON matching the asset schema (maximum 16 MiB)")
	}
	var trailing any
	if d.Decode(&trailing) != io.EOF {
		return req, invalid("request must contain one JSON object")
	}
	return normalizeInput(req)
}

// ServeHTTP is identity-agnostic; upstream auth passes the verified owner/key IDs.
func (e *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request, owner, key int64) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/assets/")
	if r.Method == "GET" && path == "capabilities" {
		out := []Capability{}
		for _, p := range e.Providers {
			out = append(out, p.Capability())
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Provider < out[j].Provider })
		reply(w, 200, map[string]any{"schema_version": SchemaVersion, "providers": out, "billing": "provider_native_credits", "multiplier": 1})
		return
	}
	if r.Method == "POST" && (path == "quotes" || path == "jobs") {
		req, err := readRequest(w, r)
		if err != nil {
			replyError(w, err)
			return
		}
		if path == "quotes" {
			price, err := e.Prepare(&req)
			if err != nil {
				replyError(w, err)
				return
			}
			reply(w, 200, map[string]any{"request": req, "quote": price})
			return
		}
		j, created, err := e.Create(r.Context(), owner, key, r.Header.Get("Idempotency-Key"), req)
		if err != nil {
			replyError(w, err)
			return
		}
		status := 200
		if created {
			status = 202
		}
		w.Header().Set("Location", "/v1/assets/jobs/"+j.ID)
		reply(w, status, j)
		return
	}
	if r.Method == "GET" && path == "jobs" {
		jobs, err := e.Store.List(r.Context(), owner, key)
		if err != nil {
			replyError(w, err)
			return
		}
		reply(w, 200, map[string]any{"jobs": jobs})
		return
	}
	parts := strings.Split(path, "/")
	if len(parts) >= 2 && parts[0] == "jobs" {
		id := parts[1]
		j, err := e.Store.Get(r.Context(), id, owner, key)
		if err != nil {
			replyError(w, err)
			return
		}
		if r.Method == "GET" && len(parts) == 2 {
			reply(w, 200, j)
			return
		}
		if r.Method == "POST" && len(parts) == 3 && (parts[2] == "cancel" || parts[2] == "retry-delivery" || parts[2] == "refresh-artifacts") {
			j, err = e.Mutate(r.Context(), id, owner, key, parts[2])
			if err != nil {
				replyError(w, err)
				return
			}
			reply(w, 200, j)
			return
		}
		if r.Method == "GET" && len(parts) == 4 && parts[2] == "artifacts" {
			for _, a := range j.Artifacts {
				if a.ID != parts[3] || a.Path == "" {
					continue
				}
				f, err := os.Open(a.Path)
				if err != nil {
					replyError(w, &APIError{Code: "artifact_unavailable", Message: "artifact is unavailable; retry delivery", HTTPStatus: 503})
					return
				}
				defer f.Close()
				info, err := f.Stat()
				if err != nil {
					replyError(w, err)
					return
				}
				w.Header().Set("Content-Type", a.MediaType)
				w.Header().Set("X-Content-Type-Options", "nosniff")
				w.Header().Set("Cache-Control", "private, no-store")
				w.Header().Set("Content-Disposition", "attachment; filename=\""+filepath.Base(a.Path)+"\"")
				http.ServeContent(w, r, filepath.Base(a.Path), info.ModTime(), f)
				return
			}
		}
	}
	replyError(w, &APIError{Code: "not_found", Message: "asset route or artifact not found", HTTPStatus: 404})
}
