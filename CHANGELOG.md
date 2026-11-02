# Changelog

## v0.1.2

- Fix: GitLab URLs with an empty path segment (`gitlab.com/a//b`) were parsed into an invalid namespace. Found by fuzzing.
- Fix: an empty `Contact:` line in security.txt counted as a contact and could mark the file valid. It's now reported as a problem.
- Fix: security.txt lines longer than 64 KiB stopped parsing silently.
- New: accept scp-style git remotes (`git@github.com:owner/repo.git`, `git@gitlab.com:grp/proj.git`).
- Fuzz tests for target parsing, security.txt and SECURITY.md contact extraction, plus edge-case tests.
- README: badges, Why, CI usage and limitations. All workflow actions pinned by SHA; OpenSSF Scorecard workflow.
