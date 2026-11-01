# disclose

**Find where to report a vulnerability, in one command.**

`disclose` takes a GitHub/GitLab repo, an npm/PyPI/crates.io package, or a domain, checks every common security reporting channel, and prints them ranked by priority. It's built for security researchers and maintainers who want to report privately and skip the public issue tracker.

```console
$ disclose https://github.com/acme/widget
Target: github:acme/widget
Home:   https://acme.example

#  PRIORITY  TYPE               CHANNEL                                                 SOURCE                                                        NOTE
1  1         github-pvr         https://github.com/acme/widget/security/advisories/new  GitHub API                                                    private vulnerability reporting enabled
2  2         security.txt       mailto:security@acme.example                            https://acme.example/.well-known/security.txt
3  2         security.txt       https://acme.example/security/report                    https://acme.example/.well-known/security.txt
4  3         security.md-email  mailto:security@acme.example                            https://github.com/acme/widget/blob/main/.github/SECURITY.md
5  4         security.md-link   https://acme.example/security/pgp.asc                   https://github.com/acme/widget/blob/main/.github/SECURITY.md
6  4         security.md-link   https://hackerone.com/acme                              https://github.com/acme/widget/blob/main/.github/SECURITY.md
```

## What it checks

| Check | How |
|---|---|
| GitHub private vulnerability reporting | `GET /repos/{owner}/{repo}/private-vulnerability-reporting` |
| SECURITY.md | Looks in `SECURITY.md`, `.github/SECURITY.md` and `docs/SECURITY.md` on the default branch, then pulls out email addresses and links that look like reporting channels (security, advisories, bug bounty, HackerOne, Bugcrowd, etc.) |
| security.txt ([RFC 9116](https://www.rfc-editor.org/rfc/rfc9116)) | Fetches `https://<domain>/.well-known/security.txt` and validates `Contact` and `Expires` (present exactly once, RFC 3339, not expired; warns if it's more than a year ahead) |
| GitLab | Reads project metadata and SECURITY.md through the GitLab API, and falls back to a confidential issue link |
| Packages | Resolves the source repo and homepage from npm, PyPI (`project_urls`) or crates.io metadata, then runs the repo and homepage checks |

## Priority

Lower numbers come first:

1. `github-pvr`: GitHub private vulnerability reporting
2. `security.txt`: a contact from a valid security.txt
3. `security.md-email`: an email address found in SECURITY.md
4. `security.md-link`: a reporting link found in SECURITY.md
5. `gitlab-confidential-issue`: a GitLab confidential issue (fallback)
9. `security.txt-expired`: a contact from an expired or invalid security.txt (use with caution)

## Install

```sh
go install github.com/ucsurf/disclose/cmd/disclose@latest
```

Prebuilt binaries for Linux, macOS and Windows (amd64/arm64) are attached to each GitHub release.

## Usage

```
disclose [flags] <github/gitlab URL | npm:pkg | pypi:pkg | crates:pkg | domain>

  -format string   output format: table or json (default "table")
  -json            shorthand for --format=json
  -v               show every check performed
  -timeout dur     overall timeout (default 30s)
  -version         print version
```

### Try it (real output, 2026-11-01)

```console
$ go install github.com/ucsurf/disclose/cmd/disclose@latest
$ disclose npm:express
Target: npm:express
Repo:   github:expressjs/express
Home:   https://expressjs.com/

#  PRIORITY  TYPE          CHANNEL                                                       SOURCE                                          NOTE
1  1         github-pvr    https://github.com/expressjs/express/security/advisories/new  GitHub API                                      private vulnerability reporting enabled
2  2         security.txt  mailto:express-security@lists.openjsf.org                     https://expressjs.com/.well-known/security.txt

$ disclose --json https://github.com/sigstore/cosign | jq -r '.channels[0].value'
https://github.com/sigstore/cosign/security/advisories/new
```

Examples:

```sh
disclose https://github.com/owner/repo
disclose https://gitlab.com/group/subgroup/project
disclose npm:@scope/package
disclose pypi:requests
disclose crates:serde
disclose example.com
disclose --json npm:left-pad | jq '.channels[0]'
```

Environment variables:

- `GITHUB_TOKEN` (optional): raises the GitHub API rate limit. The PVR endpoint may need a token on some repos.
- `GITLAB_TOKEN` (optional): use it for private or rate-limited projects.

Exit codes: `0` means at least one channel was found, `3` means none were found, and `2` means a usage or input error.

## JSON output

`--json` returns `input`, `target`, `repo` (for packages), `homepage`, `channels[]` (`priority`, `type`, `value`, `source`, `note`), `checks[]` (each probe with `found`, `not-found` or `error`), and the parsed `security_txt` with its `valid` and `problems` fields. That makes it easy to use in scripts and CI.

## How is it different?

- **OpenSSF Scorecard / Allstar** score whether a SECURITY.md exists. `disclose` tells you *who to contact and how*, ranked.
- **security.txt validators** only check one domain. `disclose` combines security.txt, GitHub PVR, SECURITY.md and package metadata into one answer.
- It's a single static binary with no dependencies outside the Go standard library.

## Development

```sh
go vet ./...
go test ./...
go test ./cmd/disclose -update   # regenerate golden files
```

Tests never touch the network. HTTP responses are replayed from recorded fixtures in `testdata/fixtures/`, and CLI output is compared against the golden files in `testdata/golden/`.

## Security

To report a vulnerability in `disclose` itself, use GitHub private vulnerability reporting on this repository.

## License

[Apache-2.0](LICENSE)
