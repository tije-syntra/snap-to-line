---
name: snap-to-line-deploy
description: >-
  Release and deploy github.com/tije-syntra/snap-to-line: semver tagging, merge
  to main, push tags, CHANGELOG, and bump downstream consumers (e.g.
  snap-to-line-dashboard). Use when the user asks to release, deploy, bump
  version, tag, push to GitHub, merge to main, or update the library version in
  dashboard.
---

# snap-to-line — Deployment & Release

Workflow for publishing **github.com/tije-syntra/snap-to-line** and updating dependents.

**Reference:** [snap-to-line-viterbi/SKILL.md](../snap-to-line-viterbi/SKILL.md) (sections *Versioning GitHub*, *Branch Maintenance*, *Changelog*).

**Remote:** `https://github.com/tije-syntra/snap-to-line.git`

---

## When to use

- User asks: release, deploy, tag, push, merge main, bump version, publish module
- After meaningful library changes (features, fixes, breaking API)
- Updating **snap-to-line-dashboard** (or other repos) to a new module version

---

## Pre-release checklist

Run from repo root (`snap-to-line/`):

```bash
go test ./...
go vet ./...
git status
git log -5 --oneline
git tag -l 'v*' | sort -V | tail -5
```

Before tagging:

- [ ] All tests pass
- [ ] README reflects new API/config (if changed)
- [ ] `CHANGELOG.md` entry drafted for the new version
- [ ] Bump type chosen (PATCH / MINOR / MAJOR)
- [ ] No secrets or local paths committed

---

## Semantic versioning

```txt
vMAJOR.MINOR.PATCH
```

| Bump | When | Example |
|------|------|---------|
| **PATCH** | Bug fix, no API change | v0.1.0 → v0.1.1 |
| **MINOR** | New feature, backward compatible | v0.1.1 → v0.2.0 |
| **MAJOR** | Breaking change | v0.x → v1.0.0 |

**Breaking change (Go v2+):** change module path:

```go
module github.com/tije-syntra/snap-to-line/v2
```

Imports: `import "github.com/tije-syntra/snap-to-line/v2"`

---

## Branch model

```txt
main        → stable releases only
develop     → active integration (optional)
feature/*   → new features
fix/*       → bug fixes
release/*   → release prep (optional)
```

Prefer: merge to `main` only when ready to tag. Use `develop` only if the team already uses it.

---

## Workflow A — Patch (bug fix)

```bash
cd /path/to/snap-to-line
git checkout main
git pull origin main

git checkout -b fix/short-description
# ... fix, test ...
go test ./...

git add -A
git commit -m "$(cat <<'EOF'
fix: short description of the bug

Explain root cause and what changed.
EOF
)"

git checkout main
git merge fix/short-description

# NEXT_VERSION = e.g. v0.1.1
git tag -a v0.1.1 -m "fix: short description"
git push origin main
git push origin v0.1.1
```

---

## Workflow B — Minor (new feature)

```bash
git checkout main && git pull origin main
git checkout -b feature/short-description
# ... implement, test ...
go test ./...

git add -A
git commit -m "$(cat <<'EOF'
feat: short description

What was added and why.
EOF
)"

git checkout main
git merge feature/short-description

# NEXT_VERSION = e.g. v0.2.0
git tag -a v0.2.0 -m "feat: short description"
git push origin main
git push origin v0.2.0
```

With `develop` branch:

```bash
git checkout develop && git pull && git merge feature/short-description
git checkout main && git merge develop
git tag -a v0.2.0 -m "feat: short description"
git push origin main develop
git push origin v0.2.0
```

---

## Workflow C — Major (breaking change)

1. Document breaking changes in CHANGELOG and README
2. If Go module v2+: update `go.mod` module path to `.../v2`
3. Tag `v1.0.0` or next major

```bash
git tag -a v1.0.0 -m "breaking: describe API changes"
git push origin main
git push origin v1.0.0
```

---

## CHANGELOG

Maintain `CHANGELOG.md` at repo root (create if missing):

```markdown
# Changelog

## v0.2.0 — YYYY-MM-DD
- feat: RouteSnapConfig with optional params
- feat: backward transition guard and terminal clamp

## v0.1.0 — YYYY-MM-DD
- Initial release
```

Update **before** tagging. Tag message can mirror the top CHANGELOG section.

---

## Commit message style

```txt
feat: add direction validation for parallel route
fix: loop stop projection for same start/end stop
docs: update RouteSnapConfig examples
test: backward guard at terminal overlap
chore: release v0.2.0
```

Use HEREDOC for multi-line commits (see user git rules).

---

## GitHub release (optional)

After push tag:

```bash
gh release create v0.2.0 --title "v0.2.0" --notes-file /tmp/release-notes.md
```

Or create release from GitHub UI using CHANGELOG excerpt.

---

## Update downstream: snap-to-line-dashboard

After tag is **pushed** and available on GitHub:

```bash
cd /path/to/snap-to-line-dashboard

# Remove local replace if present
# Edit go.mod: require github.com/tije-syntra/snap-to-line v0.2.0
# Delete: replace github.com/tije-syntra/snap-to-line => ../snap-to-line

go get github.com/tije-syntra/snap-to-line@v0.2.0
go mod tidy
go test ./...
go build ./...
```

Commit dashboard:

```bash
git add go.mod go.sum
git commit -m "$(cat <<'EOF'
chore: bump snap-to-line to v0.2.0

Use published module; remove local replace.
EOF
)"
```

If still developing library locally, keep `replace` until tag is pushed:

```go
replace github.com/tije-syntra/snap-to-line => ../snap-to-line
```

Switch to remote version only after `git push origin vX.Y.Z` succeeds.

---

## Full release checklist (agent)

Copy and track:

```txt
- [ ] go test ./... in snap-to-line
- [ ] CHANGELOG.md updated
- [ ] README updated (if API changed)
- [ ] Branch merged to main
- [ ] Annotated tag created (vX.Y.Z)
- [ ] git push origin main
- [ ] git push origin vX.Y.Z
- [ ] (optional) gh release create
- [ ] dashboard: go get @vX.Y.Z, remove replace, test, commit
```

---

## Safety rules

- **Never** force-push `main` unless user explicitly requests
- **Never** skip hooks (`--no-verify`) unless user requests
- **Never** amend pushed commits unless user requests
- Confirm tag version with `git tag -l 'v*' | sort -V | tail -3` before creating new tag
- Only commit when user asks; for release workflow, user usually explicitly requests push/tag

---

## Related docs

| File | Content |
|------|---------|
| [snap-to-line-viterbi/SKILL.md](../snap-to-line-viterbi/SKILL.md) | Architecture, versioning examples, acceptance criteria |
| [README.md](../../README.md) | Public API and RouteSnapConfig |
| Dashboard [docs/snap-to-line.md](https://github.com/tije-syntra/snap-to-line-dashboard/blob/main/docs/snap-to-line.md) | Integration notes |
