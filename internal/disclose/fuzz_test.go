package disclose

import (
	"strings"
	"testing"
	"time"
)

func FuzzParseSecurityTxt(f *testing.F) {
	f.Add("Contact: mailto:a@b.c\nExpires: 2099-01-01T00:00:00Z\n")
	f.Add("-----BEGIN PGP SIGNED MESSAGE-----\nHash: SHA256\n\nContact: https://x\nExpires: 2000-01-01T00:00:00Z\n")
	f.Add("Contact:\nExpires: nope\nExpires: again\n")
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	f.Fuzz(func(t *testing.T, body string) {
		d := ParseSecurityTxt(body, "https://x/.well-known/security.txt", now)
		if d.Valid && (len(d.Contacts) == 0 || d.Expired(now)) {
			t.Fatalf("valid doc without contact or with expiry: %+v", d)
		}
		for _, c := range d.Contacts {
			if strings.TrimSpace(c) == "" {
				t.Fatalf("empty contact in %+v", d)
			}
		}
	})
}

func FuzzExtractContacts(f *testing.F) {
	f.Add("Email security@example.com or see https://example.com/security.")
	f.Add("mailto:A@B.CO, https://x.y/(a) **https://h1.com/x**")
	f.Fuzz(func(t *testing.T, body string) {
		emails, links := ExtractContacts(body)
		seen := map[string]bool{}
		for _, e := range emails {
			if e != strings.ToLower(e) || !strings.Contains(e, "@") || seen[e] {
				t.Fatalf("bad email %q in %v", e, emails)
			}
			seen[e] = true
		}
		for _, l := range links {
			if !strings.HasPrefix(l, "http") || seen[l] {
				t.Fatalf("bad link %q in %v", l, links)
			}
			seen[l] = true
		}
	})
}

func TestSecurityTxtEdgeCases(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	long := "Contact: https://x/" + strings.Repeat("a", 100000) + "\nExpires: 2026-06-01T00:00:00Z\n"
	if d := ParseSecurityTxt(long, "u", now); !d.Valid {
		t.Errorf("long line: %+v", d.Problems)
	}
	if d := ParseSecurityTxt("Contact:\nExpires: 2026-06-01T00:00:00Z\n", "u", now); d.Valid {
		t.Error("empty Contact must not make file valid")
	}
	if d := ParseSecurityTxt("CONTACT: mailto:a@b.co\r\nexpires: 2026-06-01T00:00:00Z\r\n", "u", now); !d.Valid {
		t.Errorf("case/CRLF: %+v", d.Problems)
	}
}
