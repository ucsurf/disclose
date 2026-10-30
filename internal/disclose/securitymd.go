package disclose

import (
	"regexp"
	"strings"
)

var (
	emailRe = regexp.MustCompile(`(?i)(?:mailto:)?([a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,})`)
	linkRe  = regexp.MustCompile(`https?://[^\s<>()"'` + "`" + `]+`)
)

// SecurityMDPaths are probed in order.
var SecurityMDPaths = []string{"SECURITY.md", ".github/SECURITY.md", "docs/SECURITY.md"}

// ExtractContacts returns unique emails and links from a SECURITY.md body.
func ExtractContacts(body string) (emails, links []string) {
	seen := map[string]bool{}
	for _, m := range emailRe.FindAllStringSubmatch(body, -1) {
		e := strings.ToLower(strings.TrimRight(m[1], "."))
		if !seen[e] {
			seen[e] = true
			emails = append(emails, e)
		}
	}
	for _, l := range linkRe.FindAllString(body, -1) {
		l = strings.TrimRight(l, ".,;:!?*_]")
		if !seen[l] {
			seen[l] = true
			links = append(links, l)
		}
	}
	return emails, links
}

// linkRelevant keeps links that look like reporting channels.
func linkRelevant(l string) bool {
	ll := strings.ToLower(l)
	for _, k := range []string{"security", "advisor", "vulnerab", "bounty", "hackerone", "bugcrowd", "intigriti", "huntr", "report", "disclos", "/issues/new"} {
		if strings.Contains(ll, k) {
			return true
		}
	}
	return false
}
