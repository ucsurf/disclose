// Package fixture replays recorded HTTP interactions for tests.
package fixture

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ucsurf/disclose/internal/disclose"
)

type entry struct {
	Status int    `json:"status"`
	Body   string `json:"body"`
}

// Now is the fixed clock used by tests.
var Now = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)

// Client returns a disclose.Client wired to a replay server built from dir/fixtures.json.
// Unrecorded requests fail the test.
func Client(t *testing.T, dir string) *disclose.Client {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, "fixtures.json"))
	if err != nil {
		t.Fatal(err)
	}
	m := map[string]json.RawMessage{}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.EscapedPath()
		if r.URL.RawQuery != "" {
			key += "?" + r.URL.RawQuery
		}
		rawE, ok := m[key]
		if !ok {
			t.Errorf("unrecorded request: %s %s", r.Method, key)
			http.Error(w, "unrecorded", http.StatusTeapot)
			return
		}
		var e entry
		_ = json.Unmarshal(rawE, &e)
		if e.Body != "" {
			b, err := os.ReadFile(filepath.Join(dir, "bodies", e.Body))
			if err != nil {
				t.Errorf("fixture body: %v", err)
			}
			w.WriteHeader(e.Status)
			_, _ = w.Write(b)
			return
		}
		w.WriteHeader(e.Status)
	}))
	t.Cleanup(srv.Close)
	c := disclose.New()
	c.HTTP = srv.Client()
	c.Now = func() time.Time { return Now }
	c.Endpoints = disclose.Endpoints{
		GitHubAPI:  srv.URL + "/gh-api",
		GitHubRaw:  srv.URL + "/gh-raw",
		GitLabAPI:  srv.URL + "/gl-api",
		NPM:        srv.URL + "/npm",
		PyPI:       srv.URL + "/pypi",
		Crates:     srv.URL + "/crates",
		DomainBase: srv.URL + "/web",
	}
	return c
}
