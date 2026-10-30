// Package disclose discovers and ranks security reporting channels.
package disclose

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Endpoints holds base URLs so tests can point at recorded fixtures.
type Endpoints struct {
	GitHubAPI string // https://api.github.com
	GitHubRaw string // https://raw.githubusercontent.com
	GitLabAPI string // https://gitlab.com/api/v4
	NPM       string // https://registry.npmjs.org
	PyPI      string // https://pypi.org
	Crates    string // https://crates.io
	// DomainBase, if set, replaces "https://<domain>" (used in tests).
	DomainBase string
}

// DefaultEndpoints are the real public endpoints.
var DefaultEndpoints = Endpoints{
	GitHubAPI: "https://api.github.com",
	GitHubRaw: "https://raw.githubusercontent.com",
	GitLabAPI: "https://gitlab.com/api/v4",
	NPM:       "https://registry.npmjs.org",
	PyPI:      "https://pypi.org",
	Crates:    "https://crates.io",
}

// Client performs lookups.
type Client struct {
	HTTP        *http.Client
	Endpoints   Endpoints
	GitHubToken string
	GitLabToken string
	UserAgent   string
	Now         func() time.Time
}

// New returns a Client with sane defaults.
func New() *Client {
	return &Client{
		HTTP:      &http.Client{Timeout: 15 * time.Second},
		Endpoints: DefaultEndpoints,
		UserAgent: "disclose/0.1 (+https://github.com/ucsurf/disclose)",
		Now:       time.Now,
	}
}

type httpError struct {
	URL    string
	Status int
}

func (e *httpError) Error() string { return fmt.Sprintf("GET %s: HTTP %d", e.URL, e.Status) }

func isStatus(err error, code int) bool {
	he, ok := err.(*httpError)
	return ok && he.Status == code
}

func (c *Client) get(ctx context.Context, url string, hdr map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.UserAgent)
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &httpError{URL: url, Status: resp.StatusCode}
	}
	return body, nil
}

func (c *Client) githubHeaders() map[string]string {
	h := map[string]string{"Accept": "application/vnd.github+json", "X-GitHub-Api-Version": "2022-11-28"}
	if c.GitHubToken != "" {
		h["Authorization"] = "Bearer " + c.GitHubToken
	}
	return h
}

func (c *Client) gitlabHeaders() map[string]string {
	if c.GitLabToken == "" {
		return nil
	}
	return map[string]string{"PRIVATE-TOKEN": c.GitLabToken}
}
