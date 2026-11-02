package disclose

import (
	"bufio"
	"strings"
	"time"
)

// SecurityTxtDoc is a parsed RFC 9116 security.txt.
type SecurityTxtDoc struct {
	URL       string   `json:"url"`
	Contacts  []string `json:"contacts"`
	Expires   string   `json:"expires,omitempty"`
	Policy    []string `json:"policy,omitempty"`
	Signed    bool     `json:"signed"`
	Valid     bool     `json:"valid"`
	Problems  []string `json:"problems,omitempty"`
	expiresAt time.Time
}

// ParseSecurityTxt parses and validates a security.txt body at time now.
func ParseSecurityTxt(body, url string, now time.Time) *SecurityTxtDoc {
	d := &SecurityTxtDoc{URL: url, Contacts: []string{}}
	expiresCount := 0
	sc := bufio.NewScanner(strings.NewReader(body))
	sc.Buffer(make([]byte, 0, 64*1024), len(body)+1)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "-----BEGIN PGP SIGNED MESSAGE") {
			d.Signed = true
		}
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		v = strings.TrimSpace(v)
		switch strings.ToLower(strings.TrimSpace(k)) {
		case "contact":
			if v == "" {
				d.Problems = append(d.Problems, "empty Contact field")
				continue
			}
			d.Contacts = append(d.Contacts, v)
		case "expires":
			expiresCount++
			d.Expires = v
		case "policy":
			d.Policy = append(d.Policy, v)
		}
	}
	if len(d.Contacts) == 0 {
		d.Problems = append(d.Problems, "missing required Contact field")
	}
	switch {
	case expiresCount == 0:
		d.Problems = append(d.Problems, "missing required Expires field")
	case expiresCount > 1:
		d.Problems = append(d.Problems, "Expires field must appear only once")
	}
	if d.Expires != "" {
		t, err := time.Parse(time.RFC3339, d.Expires)
		if err != nil {
			d.Problems = append(d.Problems, "Expires is not RFC 3339: "+d.Expires)
		} else {
			d.expiresAt = t
			if !t.After(now) {
				d.Problems = append(d.Problems, "expired on "+t.UTC().Format("2006-01-02"))
			} else if t.Sub(now) > 366*24*time.Hour {
				d.Problems = append(d.Problems, "Expires is more than a year in the future (RFC 9116 recommends < 1 year)")
			}
		}
	}
	d.Valid = len(d.Contacts) > 0 && expiresCount == 1 && !d.expiresAt.IsZero() && d.expiresAt.After(now)
	return d
}

// Expired reports whether the file is present but expired/invalid Expires.
func (d *SecurityTxtDoc) Expired(now time.Time) bool {
	return d.expiresAt.IsZero() || !d.expiresAt.After(now)
}
