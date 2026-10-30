package disclose

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/ucsurf/disclose/internal/target"
)

// Run resolves the input and gathers channels.
func (c *Client) Run(ctx context.Context, input string) (*Result, error) {
	t, err := target.Parse(input)
	if err != nil {
		return nil, err
	}
	r := &Result{Input: input, Target: t}
	repo := t
	switch t.Kind {
	case target.NPM, target.PyPI, target.Crates:
		repoURL, home, err := c.resolvePackage(ctx, t)
		if err != nil {
			r.check("registry:"+string(t.Kind), "error", err.Error())
		} else {
			r.check("registry:"+string(t.Kind), "found", strings.TrimSpace("repo="+repoURL+" homepage="+home))
		}
		r.Homepage = home
		repo = target.Target{}
		if repoURL != "" {
			if rt, err := target.Parse(repoURL); err == nil && (rt.Kind == target.GitHub || rt.Kind == target.GitLab) {
				repo = rt
				r.Repo = &rt
			}
		}
	}
	switch repo.Kind {
	case target.GitHub:
		c.github(ctx, r, repo)
	case target.GitLab:
		c.gitlab(ctx, r, repo)
	}
	// security.txt on domain target or package homepage.
	domain := ""
	if t.Kind == target.Domain {
		domain = t.Name
	} else if r.Homepage != "" {
		if u, err := url.Parse(r.Homepage); err == nil {
			h := strings.ToLower(u.Hostname())
			if h != "" && h != "github.com" && h != "gitlab.com" && !strings.HasSuffix(h, ".github.io") {
				domain = h
			}
		}
	}
	if domain != "" {
		c.securityTxt(ctx, r, domain)
	}
	r.sortChannels()
	return r, nil
}

func (c *Client) resolvePackage(ctx context.Context, t target.Target) (repo, home string, err error) {
	switch t.Kind {
	case target.NPM:
		b, err := c.get(ctx, c.Endpoints.NPM+"/"+strings.ReplaceAll(url.PathEscape(t.Name), "%40", "@"), nil)
		if err != nil {
			return "", "", err
		}
		var m struct {
			Homepage   string          `json:"homepage"`
			Repository json.RawMessage `json:"repository"`
		}
		if err := json.Unmarshal(b, &m); err != nil {
			return "", "", err
		}
		var rs string
		var ro struct {
			URL string `json:"url"`
		}
		if json.Unmarshal(m.Repository, &rs) != nil {
			_ = json.Unmarshal(m.Repository, &ro)
			rs = ro.URL
		}
		return normalizeRepoURL(rs), m.Homepage, nil
	case target.PyPI:
		b, err := c.get(ctx, c.Endpoints.PyPI+"/pypi/"+url.PathEscape(t.Name)+"/json", nil)
		if err != nil {
			return "", "", err
		}
		var m struct {
			Info struct {
				HomePage    string            `json:"home_page"`
				ProjectURLs map[string]string `json:"project_urls"`
			} `json:"info"`
		}
		if err := json.Unmarshal(b, &m); err != nil {
			return "", "", err
		}
		home = m.Info.HomePage
		cands := []string{m.Info.HomePage}
		for _, k := range []string{"Source", "Source Code", "Repository", "Code", "GitHub", "Homepage", "Home"} {
			if v, ok := m.Info.ProjectURLs[k]; ok {
				cands = append(cands, v)
				if home == "" && (k == "Homepage" || k == "Home") {
					home = v
				}
			}
		}
		for _, v := range m.Info.ProjectURLs {
			cands = append(cands, v)
		}
		for _, cnd := range cands {
			if rt, err := target.Parse(cnd); err == nil && (rt.Kind == target.GitHub || rt.Kind == target.GitLab) {
				return normalizeRepoURL(cnd), home, nil
			}
		}
		return "", home, nil
	case target.Crates:
		b, err := c.get(ctx, c.Endpoints.Crates+"/api/v1/crates/"+url.PathEscape(t.Name), nil)
		if err != nil {
			return "", "", err
		}
		var m struct {
			Crate struct {
				Repository string `json:"repository"`
				Homepage   string `json:"homepage"`
			} `json:"crate"`
		}
		if err := json.Unmarshal(b, &m); err != nil {
			return "", "", err
		}
		return normalizeRepoURL(m.Crate.Repository), m.Crate.Homepage, nil
	}
	return "", "", fmt.Errorf("not a package target")
}

func normalizeRepoURL(s string) string {
	s = strings.TrimPrefix(s, "git+")
	s = strings.Replace(s, "git://", "https://", 1)
	s = strings.Replace(s, "ssh://git@", "https://", 1)
	if strings.HasPrefix(s, "git@") {
		s = "https://" + strings.Replace(strings.TrimPrefix(s, "git@"), ":", "/", 1)
	}
	if strings.HasPrefix(s, "github:") {
		s = "https://github.com/" + strings.TrimPrefix(s, "github:")
	}
	return s
}

func (c *Client) github(ctx context.Context, r *Result, t target.Target) {
	api := c.Endpoints.GitHubAPI
	slug := url.PathEscape(t.Owner) + "/" + url.PathEscape(t.Repo)
	var meta struct {
		DefaultBranch string `json:"default_branch"`
		Homepage      string `json:"homepage"`
		HTMLURL       string `json:"html_url"`
	}
	b, err := c.get(ctx, api+"/repos/"+slug, c.githubHeaders())
	if err != nil {
		r.check("github:repo", "error", err.Error())
		return
	}
	_ = json.Unmarshal(b, &meta)
	r.check("github:repo", "found", "default_branch="+meta.DefaultBranch)
	if r.Homepage == "" {
		r.Homepage = meta.Homepage
	}
	if meta.HTMLURL == "" {
		meta.HTMLURL = "https://github.com/" + t.Slug()
	}
	// Private vulnerability reporting.
	b, err = c.get(ctx, api+"/repos/"+slug+"/private-vulnerability-reporting", c.githubHeaders())
	switch {
	case err != nil:
		r.check("github:private-vulnerability-reporting", "error", err.Error())
	default:
		var pvr struct {
			Enabled bool `json:"enabled"`
		}
		_ = json.Unmarshal(b, &pvr)
		if pvr.Enabled {
			r.check("github:private-vulnerability-reporting", "found", "enabled")
			r.add(GitHubPVR, meta.HTMLURL+"/security/advisories/new", "GitHub API", "private vulnerability reporting enabled")
		} else {
			r.check("github:private-vulnerability-reporting", "not-found", "disabled")
		}
	}
	branch := meta.DefaultBranch
	if branch == "" {
		branch = "HEAD"
	}
	for _, p := range SecurityMDPaths {
		u := c.Endpoints.GitHubRaw + "/" + slug + "/" + url.PathEscape(branch) + "/" + p
		body, err := c.get(ctx, u, nil)
		if isStatus(err, 404) {
			r.check("github:"+p, "not-found", "")
			continue
		}
		if err != nil {
			r.check("github:"+p, "error", err.Error())
			continue
		}
		r.check("github:"+p, "found", "")
		c.addSecurityMD(r, string(body), meta.HTMLURL+"/blob/"+branch+"/"+p)
		break
	}
}

func (c *Client) gitlab(ctx context.Context, r *Result, t target.Target) {
	api := c.Endpoints.GitLabAPI
	id := url.PathEscape(t.Slug())
	b, err := c.get(ctx, api+"/projects/"+id, c.gitlabHeaders())
	if err != nil {
		r.check("gitlab:project", "error", err.Error())
		return
	}
	var meta struct {
		DefaultBranch string `json:"default_branch"`
		WebURL        string `json:"web_url"`
		IssuesEnabled bool   `json:"issues_enabled"`
	}
	_ = json.Unmarshal(b, &meta)
	r.check("gitlab:project", "found", "default_branch="+meta.DefaultBranch)
	if meta.WebURL == "" {
		meta.WebURL = "https://gitlab.com/" + t.Slug()
	}
	branch := meta.DefaultBranch
	if branch == "" {
		branch = "HEAD"
	}
	found := false
	for _, p := range SecurityMDPaths {
		u := api + "/projects/" + id + "/repository/files/" + url.PathEscape(p) + "/raw?ref=" + url.QueryEscape(branch)
		body, err := c.get(ctx, u, c.gitlabHeaders())
		if isStatus(err, 404) {
			r.check("gitlab:"+p, "not-found", "")
			continue
		}
		if err != nil {
			r.check("gitlab:"+p, "error", err.Error())
			continue
		}
		r.check("gitlab:"+p, "found", "")
		c.addSecurityMD(r, string(body), meta.WebURL+"/-/blob/"+branch+"/"+p)
		found = true
		break
	}
	if meta.IssuesEnabled {
		note := "fallback; tick 'This issue is confidential'"
		if found {
			note = "fallback; prefer SECURITY.md instructions"
		}
		r.add(GitLabConfIssue, meta.WebURL+"/-/issues/new?issue[confidential]=true", "GitLab API", note)
	}
}

func (c *Client) addSecurityMD(r *Result, body, src string) {
	emails, links := ExtractContacts(body)
	for _, e := range emails {
		r.add(SecurityMDEmail, "mailto:"+e, src, "")
	}
	for _, l := range links {
		if linkRelevant(l) {
			r.add(SecurityMDLink, l, src, "")
		}
	}
}

func (c *Client) securityTxt(ctx context.Context, r *Result, domain string) {
	base := "https://" + domain
	if c.Endpoints.DomainBase != "" {
		base = c.Endpoints.DomainBase + "/" + domain
	}
	u := base + "/.well-known/security.txt"
	shown := "https://" + domain + "/.well-known/security.txt"
	b, err := c.get(ctx, u, nil)
	if err != nil {
		if isStatus(err, 404) {
			r.check("security.txt:"+domain, "not-found", "")
		} else {
			r.check("security.txt:"+domain, "error", err.Error())
		}
		return
	}
	if strings.Contains(strings.ToLower(string(b[:min(len(b), 512)])), "<html") {
		r.check("security.txt:"+domain, "not-found", "returned HTML, not security.txt")
		return
	}
	d := ParseSecurityTxt(string(b), shown, c.Now())
	r.SecTxt = d
	status := "found"
	detail := "valid"
	if !d.Valid {
		detail = strings.Join(d.Problems, "; ")
	}
	r.check("security.txt:"+domain, status, detail)
	typ, note := SecurityTxt, ""
	if !d.Valid {
		typ, note = SecurityTxtStale, "security.txt invalid: "+strings.Join(d.Problems, "; ")
	}
	for _, ct := range d.Contacts {
		r.add(typ, ct, shown, note)
	}
}
