package main

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/ucsurf/disclose/internal/fixture"
)

var update = flag.Bool("update", false, "rewrite golden files")

func TestGolden(t *testing.T) {
	cases := []struct {
		name string
		args []string
		code int
	}{
		{"github_table", []string{"https://github.com/acme/widget"}, 0},
		{"github_json", []string{"--json", "https://github.com/acme/widget"}, 0},
		{"github_verbose", []string{"-v", "github.com/plain/lib"}, 0},
		{"gitlab_table", []string{"https://gitlab.com/group/sub/tool"}, 0},
		{"npm_table", []string{"npm:@acme/widget"}, 0},
		{"pypi_json", []string{"--format", "json", "pypi:plainlib"}, 0},
		{"crates_table", []string{"crates:tool"}, 0},
		{"domain_expired_json", []string{"--json", "old.example"}, 0},
		{"none_found", []string{"https://github.com/ghost/none"}, 3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var out, errb bytes.Buffer
			code := run(c.args, &out, &errb, fixture.Client(t, "../../testdata/fixtures"))
			if code != c.code {
				t.Fatalf("exit %d want %d; stderr=%s", code, c.code, errb.String())
			}
			path := filepath.Join("../../testdata/golden", c.name+".golden")
			if *update {
				if err := os.WriteFile(path, out.Bytes(), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(out.Bytes(), want) {
				t.Errorf("golden mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", c.name, out.String(), want)
			}
		})
	}
}

func TestUsageErrors(t *testing.T) {
	c := fixture.Client(t, "../../testdata/fixtures")
	for _, args := range [][]string{{}, {"--format", "xml", "x.com"}, {"npm:"}, {"a", "b"}} {
		var out, errb bytes.Buffer
		if code := run(args, &out, &errb, c); code != 2 {
			t.Errorf("%v: exit %d, want 2", args, code)
		}
	}
	var out bytes.Buffer
	if run([]string{"--version"}, &out, &out, c) != 0 || out.String() != "disclose dev\n" {
		t.Errorf("version: %q", out.String())
	}
}
