package target

import (
	"strings"
	"testing"
)

func FuzzParse(f *testing.F) {
	for _, s := range []string{"https://github.com/foo/bar", "git@github.com:foo/bar.git", "gitlab.com/a/b/c", "npm:@s/p", "example.com", "http://[::1]/", "github.com//", "https://gitlab.com/-/x", "%zz"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, in string) {
		got, err := Parse(in)
		if err != nil {
			return
		}
		switch got.Kind {
		case GitHub:
			if got.Owner == "" || got.Repo == "" || strings.Contains(got.Owner, "/") || strings.Contains(got.Repo, "/") {
				t.Fatalf("Parse(%q) = %+v: bad github slug", in, got)
			}
		case GitLab:
			if got.Owner == "" || got.Repo == "" || strings.Contains(got.Repo, "/") || strings.HasPrefix(got.Owner, "/") || strings.HasSuffix(got.Owner, "/") || strings.Contains(got.Owner, "//") {
				t.Fatalf("Parse(%q) = %+v: bad gitlab slug", in, got)
			}
		case Domain:
			if got.Name == "" || !strings.Contains(got.Name, ".") || strings.ContainsAny(got.Name, "/ ") {
				t.Fatalf("Parse(%q) = %+v: bad domain", in, got)
			}
		case NPM, PyPI, Crates:
			if got.Name == "" {
				t.Fatalf("Parse(%q): empty package", in)
			}
		default:
			t.Fatalf("Parse(%q): unknown kind %q", in, got.Kind)
		}
	})
}
