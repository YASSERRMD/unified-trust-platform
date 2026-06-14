# Unified Trust Platform — Git Workflow

## Author Configuration

Always configure before committing:

```bash
git config user.name "YASSERRMD"
git config user.email "arafath.yasser@gmail.com"
```

Author identity on all commits: `YASSERRMD <arafath.yasser@gmail.com>`

---

## Branch Strategy

### Naming
```
phase-XX-short-name
```

Examples:
- `phase-01-foundation`
- `phase-05-oauth-oidc`
- `phase-15-security`

### Lifecycle

```
main
 └── phase-XX-short-name       ← create from main
      ├── atomic commit 1
      ├── atomic commit 2
      ├── ...
      └── atomic commit N
           └── PR → merge → main  ← merge and delete branch
```

### Commands

```bash
# Start a phase
git checkout main
git pull origin main
git checkout -b phase-XX-short-name

# During the phase: many small commits
git add <specific files>
git commit -m "phase X: <small action completed>"

# End the phase
git push -u origin phase-XX-short-name
gh pr create --title "phase X: <phase name>" --body "..."
# After review/approval:
gh pr merge <number> --merge
git checkout main
git pull origin main
git branch -d phase-XX-short-name
git push origin --delete phase-XX-short-name
```

---

## Commit Message Format

```
phase <N>: <small action completed>
```

### Rules
- Use lowercase
- Present tense, imperative ("add", not "added")
- One small task per commit
- No co-author lines, no AI attribution

### Good Examples
```
phase 1: initialize go backend module
phase 1: add backend folder structure
phase 2: add tenant schema migration
phase 3: add request id middleware
phase 5: add pkce validation
phase 7: add rbac evaluator
```

### Bad Examples
```
phase 1: add everything               ← too broad
Phase 1: Initialize Go Module         ← wrong case
fix stuff                             ← no phase, no action
phase 5: completed oauth2             ← past tense, vague
```

---

## Pull Request Rules

- PR title: `phase X: <phase name>`
- One PR per phase
- PR description includes:
  - Branch name
  - Commits list
  - Tests run
  - Files changed
  - Any risks
- Merge strategy: merge commit (not squash, not rebase)
- Delete branch after merge

---

## Verification Before PR

```bash
git status                  # must be clean
go fmt ./...
go test ./...
go vet ./...
npm run lint
npm run build
docker compose config
```

All checks must pass before creating the PR.
