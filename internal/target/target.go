// Package target parses user input into a lookup target.
package target

import (
	"fmt"
	"net/url"
	"strings"
)

// Kind is the type of target.
type Kind string

const (
	GitHub Kind = "github"
	GitLab Kind = "gitlab"
	NPM    Kind = "npm"
	PyPI   Kind = "pypi"
	Crates Kind = "crates"
	Domain Kind = "domain"
)

// Target is a parsed input.
type Target struct {
	Kind  Kind   `json:"kind"`
	Owner string `json:"owner,omitempty"` // GitHub owner or GitLab namespace path
	Repo  string `json:"repo,omitempty"`
	Name  string `json:"name,omitempty"` // package name or domain
}

// Slug returns owner/repo.
func (t Target) Slug() string { return t.Owner + "/" + t.Repo }

func (t Target) String() string {
	switch t.Kind {
	case GitHub, GitLab:
		return string(t.Kind) + ":" + t.Slug()
	default:
		return string(t.Kind) + ":" + t.Name
	}
}

// Parse parses a repo URL, package spec (npm:, pypi:, crates:) or domain.
func Parse(in string) (Target, error) {
	in = strings.TrimSpace(in)
	if in == "" {
		return Target{}, fmt.Errorf("empty target")
	}
	for _, p := range []Kind{NPM, PyPI, Crates} {
		if rest, ok := strings.CutPrefix(in, string(p)+":"); ok {
			if rest == "" {
				return Target{}, fmt.Errorf("missing package name in %q", in)
			}
			return Target{Kind: p, Name: rest}, nil
		}
	}
	raw := in
	// scp-like git remotes: git@github.com:owner/repo.git
	if at := strings.Index(raw, "@"); at >= 0 && !strings.Contains(raw, "://") {
		if c := strings.Index(raw[at:], ":"); c > 0 {
			raw = "ssh://" + raw[:at+c] + "/" + raw[at+c+1:]
		}
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return Target{}, fmt.Errorf("cannot parse %q", in)
	}
	host := strings.ToLower(strings.TrimPrefix(u.Hostname(), "www."))
	path := strings.Trim(strings.TrimSuffix(strings.Trim(u.Path, "/"), ".git"), "/")
	switch host {
	case "github.com":
		parts := strings.Split(path, "/")
		if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
			return Target{}, fmt.Errorf("github URL needs owner/repo: %q", in)
		}
		return Target{Kind: GitHub, Owner: parts[0], Repo: parts[1]}, nil
	case "gitlab.com":
		if i := strings.Index(path, "/-/"); i >= 0 {
			path = path[:i]
		}
		if strings.Contains(path, "//") {
			return Target{}, fmt.Errorf("gitlab URL has empty path segment: %q", in)
		}
		i := strings.LastIndex(path, "/")
		if i <= 0 || i == len(path)-1 {
			return Target{}, fmt.Errorf("gitlab URL needs namespace/project: %q", in)
		}
		return Target{Kind: GitLab, Owner: path[:i], Repo: path[i+1:]}, nil
	}
	if path != "" || !strings.Contains(host, ".") {
		return Target{}, fmt.Errorf("unsupported target %q (want GitHub/GitLab URL, npm:/pypi:/crates: package, or domain)", in)
	}
	return Target{Kind: Domain, Name: host}, nil
}
