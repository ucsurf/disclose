package disclose_test

import (
	"context"
	"testing"
	"time"

	"github.com/ucsurf/disclose/internal/disclose"
	"github.com/ucsurf/disclose/internal/fixture"
)

const fixtures = "../../testdata/fixtures"

func TestGitHubPVRAndSecurityMD(t *testing.T) {
	c := fixture.Client(t, fixtures)
	r, err := c.Run(context.Background(), "https://github.com/acme/widget")
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Channels) == 0 || r.Channels[0].Type != disclose.GitHubPVR {
		t.Fatalf("want PVR first, got %+v", r.Channels)
	}
	if r.Channels[0].Value != "https://github.com/acme/widget/security/advisories/new" {
		t.Errorf("pvr url = %s", r.Channels[0].Value)
	}
	has := func(typ disclose.ChannelType, v string) bool {
		for _, ch := range r.Channels {
			if ch.Type == typ && ch.Value == v {
				return true
			}
		}
		return false
	}
	if !has(disclose.SecurityMDEmail, "mailto:security@acme.example") {
		t.Error("missing SECURITY.md email")
	}
	if !has(disclose.SecurityMDLink, "https://hackerone.com/acme") {
		t.Error("missing bounty link")
	}
	if has(disclose.SecurityMDLink, "https://acme.example/docs") {
		t.Error("irrelevant link should be filtered")
	}
	// homepage acme.example -> security.txt valid
	if r.SecTxt == nil || !r.SecTxt.Valid {
		t.Errorf("expected valid security.txt, got %+v", r.SecTxt)
	}
}

func TestGitHubPVRDisabled(t *testing.T) {
	c := fixture.Client(t, fixtures)
	r, err := c.Run(context.Background(), "github.com/plain/lib")
	if err != nil {
		t.Fatal(err)
	}
	for _, ch := range r.Channels {
		if ch.Type == disclose.GitHubPVR {
			t.Fatal("PVR should be absent")
		}
	}
	if len(r.Channels) != 1 || r.Channels[0].Value != "mailto:jane.doe@plain.dev" {
		t.Fatalf("got %+v", r.Channels)
	}
}

func TestGitHubMissingRepo(t *testing.T) {
	c := fixture.Client(t, fixtures)
	r, err := c.Run(context.Background(), "https://github.com/ghost/none")
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Channels) != 0 || r.Checks[0].Status != "error" {
		t.Fatalf("got %+v / %+v", r.Channels, r.Checks)
	}
}

func TestGitLab(t *testing.T) {
	c := fixture.Client(t, fixtures)
	r, err := c.Run(context.Background(), "https://gitlab.com/group/sub/tool")
	if err != nil {
		t.Fatal(err)
	}
	if r.Channels[0].Value != "mailto:security@tool.example" {
		t.Fatalf("got %+v", r.Channels)
	}
	last := r.Channels[len(r.Channels)-1]
	if last.Type != disclose.GitLabConfIssue {
		t.Errorf("want confidential issue fallback last, got %+v", last)
	}
}

func TestPackages(t *testing.T) {
	cases := map[string]string{
		"npm:@acme/widget": "github:acme/widget",
		"pypi:plainlib":    "github:plain/lib",
		"crates:tool":      "gitlab:group/sub/tool",
	}
	for in, want := range cases {
		c := fixture.Client(t, fixtures)
		r, err := c.Run(context.Background(), in)
		if err != nil {
			t.Fatal(err)
		}
		if r.Repo == nil || r.Repo.String() != want {
			t.Errorf("%s: repo = %v, want %s", in, r.Repo, want)
		}
		if len(r.Channels) == 0 {
			t.Errorf("%s: no channels", in)
		}
	}
}

func TestDomainExpired(t *testing.T) {
	c := fixture.Client(t, fixtures)
	r, err := c.Run(context.Background(), "old.example")
	if err != nil {
		t.Fatal(err)
	}
	if r.SecTxt == nil || r.SecTxt.Valid {
		t.Fatalf("expected invalid security.txt: %+v", r.SecTxt)
	}
	if r.Channels[0].Type != disclose.SecurityTxtStale || r.Channels[0].Priority != 9 {
		t.Errorf("got %+v", r.Channels[0])
	}
}

func TestParseSecurityTxt(t *testing.T) {
	now := time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		name, body string
		valid      bool
		problems   int
	}{
		{"valid", "Contact: mailto:a@b.c\nExpires: 2027-01-01T00:00:00Z\n", true, 0},
		{"expired", "Contact: mailto:a@b.c\nExpires: 2025-01-01T00:00:00Z\n", false, 1},
		{"no expires", "Contact: mailto:a@b.c\n", false, 1},
		{"bad date", "Contact: mailto:a@b.c\nExpires: tomorrow\n", false, 1},
		{"dup expires", "Contact: x\nExpires: 2027-01-01T00:00:00Z\nExpires: 2027-01-02T00:00:00Z\n", false, 1},
		{"no contact", "Expires: 2027-01-01T00:00:00Z\n", false, 1},
		{"too far", "Contact: x\nExpires: 2029-01-01T00:00:00Z\n", true, 1},
		{"signed + comments", "-----BEGIN PGP SIGNED MESSAGE-----\n# hi\ncontact: x\nexpires: 2027-01-01T00:00:00+03:00\n", true, 0},
	}
	for _, c := range cases {
		d := disclose.ParseSecurityTxt(c.body, "u", now)
		if d.Valid != c.valid || len(d.Problems) != c.problems {
			t.Errorf("%s: valid=%v problems=%v", c.name, d.Valid, d.Problems)
		}
	}
}

func TestExtractContacts(t *testing.T) {
	e, l := disclose.ExtractContacts("Mail SEC@Foo.org. or mailto:sec@foo.org, see <https://foo.org/security>.")
	if len(e) != 1 || e[0] != "sec@foo.org" {
		t.Errorf("emails = %v", e)
	}
	if len(l) != 1 || l[0] != "https://foo.org/security" {
		t.Errorf("links = %v", l)
	}
}
