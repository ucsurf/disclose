package target

import "testing"

func TestParse(t *testing.T) {
	cases := []struct {
		in   string
		want Target
		err  bool
	}{
		{in: "https://github.com/foo/bar", want: Target{Kind: GitHub, Owner: "foo", Repo: "bar"}},
		{in: "github.com/foo/bar.git", want: Target{Kind: GitHub, Owner: "foo", Repo: "bar"}},
		{in: "https://github.com/foo/bar/tree/main/x", want: Target{Kind: GitHub, Owner: "foo", Repo: "bar"}},
		{in: "https://gitlab.com/grp/sub/proj/-/issues", want: Target{Kind: GitLab, Owner: "grp/sub", Repo: "proj"}},
		{in: "npm:@scope/pkg", want: Target{Kind: NPM, Name: "@scope/pkg"}},
		{in: "pypi:requests", want: Target{Kind: PyPI, Name: "requests"}},
		{in: "crates:serde", want: Target{Kind: Crates, Name: "serde"}},
		{in: "Example.COM", want: Target{Kind: Domain, Name: "example.com"}},
		{in: "https://www.example.com/", want: Target{Kind: Domain, Name: "example.com"}},
		{in: "git@github.com:foo/bar.git", want: Target{Kind: GitHub, Owner: "foo", Repo: "bar"}},
		{in: "git@gitlab.com:grp/proj.git", want: Target{Kind: GitLab, Owner: "grp", Repo: "proj"}},
		{in: "gitlab.com/a//b", err: true},
		{in: "  ", err: true},
		{in: "https://github.com/foo", err: true},
		{in: "npm:", err: true},
		{in: "", err: true},
		{in: "localhost", err: true},
		{in: "https://example.com/some/path", err: true},
	}
	for _, c := range cases {
		got, err := Parse(c.in)
		if c.err {
			if err == nil {
				t.Errorf("Parse(%q) expected error, got %+v", c.in, got)
			}
			continue
		}
		if err != nil || got != c.want {
			t.Errorf("Parse(%q) = %+v, %v; want %+v", c.in, got, err, c.want)
		}
	}
}
