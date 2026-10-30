package disclose

import (
	"sort"

	"github.com/ucsurf/disclose/internal/target"
)

// ChannelType identifies a kind of reporting channel.
type ChannelType string

const (
	GitHubPVR        ChannelType = "github-pvr"
	SecurityTxt      ChannelType = "security.txt"
	SecurityMDEmail  ChannelType = "security.md-email"
	SecurityMDLink   ChannelType = "security.md-link"
	GitLabConfIssue  ChannelType = "gitlab-confidential-issue"
	SecurityTxtStale ChannelType = "security.txt-expired"
)

// basePriority: lower is better.
var basePriority = map[ChannelType]int{
	GitHubPVR:        1,
	SecurityTxt:      2,
	SecurityMDEmail:  3,
	SecurityMDLink:   4,
	GitLabConfIssue:  5,
	SecurityTxtStale: 9,
}

// Channel is one way to report a vulnerability.
type Channel struct {
	Priority int         `json:"priority"`
	Type     ChannelType `json:"type"`
	Value    string      `json:"value"`
	Source   string      `json:"source"`
	Note     string      `json:"note,omitempty"`
}

// Result is the full output for one input.
type Result struct {
	Input    string          `json:"input"`
	Target   target.Target   `json:"target"`
	Repo     *target.Target  `json:"repo,omitempty"`
	Homepage string          `json:"homepage,omitempty"`
	Channels []Channel       `json:"channels"`
	Checks   []Check         `json:"checks"`
	SecTxt   *SecurityTxtDoc `json:"security_txt,omitempty"`
}

// Check records each probe and its outcome (for transparency).
type Check struct {
	Name   string `json:"name"`
	Status string `json:"status"` // found, not-found, error, skipped
	Detail string `json:"detail,omitempty"`
}

func (r *Result) add(t ChannelType, value, source, note string) {
	for _, c := range r.Channels {
		if c.Type == t && c.Value == value {
			return
		}
	}
	r.Channels = append(r.Channels, Channel{Priority: basePriority[t], Type: t, Value: value, Source: source, Note: note})
}

func (r *Result) check(name, status, detail string) {
	r.Checks = append(r.Checks, Check{Name: name, Status: status, Detail: detail})
}

func (r *Result) sortChannels() {
	sort.SliceStable(r.Channels, func(i, j int) bool { return r.Channels[i].Priority < r.Channels[j].Priority })
	if r.Channels == nil {
		r.Channels = []Channel{}
	}
}
