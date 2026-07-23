# APIKit v2 Merge to Main — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Merge the v2-improvements branch into main by placing v2 code under `v2/` subdirectory while preserving v1 at root, following Go semantic import versioning.

**Architecture:** Use `git read-tree` to extract the v2-improvements tree under `v2/` on main, avoiding merge conflicts. Both versions coexist: v1 at root (`github.com/likearthian/apikit`), v2 at `v2/` (`github.com/likearthian/apikit/v2`).

**Tech Stack:** Git read-tree, Go 1.19, gofmt, go vet, go test.

## Global Constraints

- v1 code at root must remain unchanged and importable via `github.com/likearthian/apikit`
- v2 code must live in `v2/` subdirectory and be importable via `github.com/likearthian/apikit/v2`
- Both v1 and v2 must compile and pass tests after merge
- No existing tags exist; we will create `v1.0.0` and `v2.0.0`
- v1 must be supported for at least 1 year for external legacy consumers

---

### Task 1: Ensure main is clean and on the right commit

**Files:** None (repo-state check)

- [ ] **Step 1: Verify we are on main with a clean working tree**

```bash
cd /home/ziska/Projects/go/likearthian/apikit && git branch --show-current && git status --porcelain
```

Expected: `main` branch, no uncommitted changes (empty output from `git status --porcelain`). If dirty, stash or commit first.

- [ ] **Step 2: Commit the design doc**

```bash
cd /home/ziska/Projects/go/likearthian/apikit && git add docs/superpowers/specs/2026-07-23-apikit-v2-merge-design.md && git commit -m "docs: add v2 merge design spec"
```

---

### Task 2: Read-tree v2-improvements into v2/ prefix

**Files:**
- Modify: index (staging area)
- Create: `v2/` (entire v2 tree)

- [ ] **Step 1: Dry-run read-tree to preview what will land**

```bash
cd /home/ziska/Projects/go/likearthian/apikit && git read-tree --prefix=v2/ -n -u v2-improvements
```

Expected: Shows what would change without actually doing it. Verify output shows only files under `v2/` prefix, nothing at root.

- [ ] **Step 2: Perform the actual read-tree**

```bash
cd /home/ziska/Projects/go/likearthian/apikit && git read-tree --prefix=v2/ v2-improvements
```

- [ ] **Step 3: Verify the staged changes look correct**

```bash
cd /home/ziska/Projects/go/likearthian/apikit && git status
```

Expected: All v2 files staged under `v2/`. No root-level files staged or changed. Files like `v2/api/auth.go`, `v2/transport/http/server.go`, `v2/go.mod`, `v2/go.sum`, `v2/CHANGELOG.md`, `v2/MIGRATION.md`, `v2/README.md` should appear as new staged files.

- [ ] **Step 4: Verify v1 files are untouched**

```bash
cd /home/ziska/Projects/go/likearthian/apikit && ls api/ transport/ logger/ logging.go response.go go.mod go.sum
```

Expected: All v1 files still present and unchanged.

---

### Task 3: Remove artifacts that should not be in the v2/ tree

**Files:**
- Remove: `v2/.git` (worktree git metadata from read-tree)
- Remove: `v2/.gitignore` (v2's gitignore, not needed in subdirectory)
- Remove: `v2/.worktrees/` (if present)
- Remove: `v2/improvement_plan.html` (if present)
- Remove: `v2/graphify-out/` (if present)
- Remove: `v2/docs/` (if present — design docs stay at root `docs/`)

- [ ] **Step 1: Check which non-essential files landed in v2/**

```bash
cd /home/ziska/Projects/go/likearthian/apikit && ls -la v2/ | grep -E '^\.|graphify|improvement|\.html'
```

- [ ] **Step 2: Remove non-essential artifacts from the index and working tree**

```bash
cd /home/ziska/Projects/go/likearthian/apikit
# Remove .git metadata if it landed (it shouldn't, but check)
test -d v2/.git && git rm -r --cached v2/.git 2>/dev/null; rm -rf v2/.git 2>/dev/null

# Remove .gitignore — v2 subdirectory inherits root's
test -f v2/.gitignore && git rm --cached v2/.gitignore 2>/dev/null; rm -f v2/.gitignore 2>/dev/null

# Remove graphify artifacts if present
test -d v2/graphify-out && git rm -r --cached v2/graphify-out 2>/dev/null; rm -rf v2/graphify-out 2>/dev/null

# Remove improvement_plan.html if present
test -f v2/improvement_plan.html && git rm --cached v2/improvement_plan.html 2>/dev/null; rm -f v2/improvement_plan.html 2>/dev/null

# Remove docs if present (keep root docs/ only)
test -d v2/docs && git rm -r --cached v2/docs 2>/dev/null; rm -rf v2/docs 2>/dev/null
```

- [ ] **Step 3: Verify v2/ tree is clean**

```bash
cd /home/ziska/Projects/go/likearthian/apikit && ls v2/
```

Expected: `api/`, `transport/`, `logger/`, `logging.go`, `response.go`, `example_test.go`, `go.mod`, `go.sum`, `README.md`, `CHANGELOG.md`, `MIGRATION.md`

---

### Task 4: Update root .gitignore

**Files:**
- Modify: `.gitignore`

- [ ] **Step 1: Read current .gitignore**

```bash
cat /home/ziska/Projects/go/likearthian/apikit/.gitignore
```

- [ ] **Step 2: Update .gitignore to be worktree-safe and exclude artifacts**

Replace `.gitignore` content with:

```
# Local worktree metadata
.worktrees/

# Generated artifacts
graphify-out/
improvement_plan.html

# IDE
*.swp
*.swo
```

```bash
cat > /home/ziska/Projects/go/likearthian/apikit/.gitignore << 'EOF'
# Local worktree metadata
.worktrees/

# Generated artifacts
graphify-out/
improvement_plan.html

# IDE
*.swp
*.swo
EOF
```

---

### Task 5: Update root README.md to document v1 + v2 coexistence

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Read current README**

```bash
cd /home/ziska/Projects/go/likearthian/apikit && head -20 README.md
```

- [ ] **Step 2: Prepend a version notice at the top of README.md**

Add this at the very top of `README.md`:

```
## Versions

- **v2** (recommended for new projects) — see [`v2/`](v2/) and the [Migration Guide](v2/MIGRATION.md)
- **v1** (legacy support) — this directory. Maintained through at least 2027.

Import paths:

| Version | Module path |
|---------|-------------|
| v2 | `github.com/likearthian/apikit/v2` |
| v1 | `github.com/likearthian/apikit` |

```

Use `ed` or `sed` to prepend:

```bash
cd /home/ziska/Projects/go/likearthian/apikit
cat > /tmp/version_notice.md << 'EOF'
## Versions

- **v2** (recommended for new projects) — see [`v2/`](v2/) and the [Migration Guide](v2/MIGRATION.md)
- **v1** (legacy support) — this directory. Maintained through at least 2027.

Import paths:

| Version | Module path |
|---------|-------------|
| v2 | `github.com/likearthian/apikit/v2` |
| v1 | `github.com/likearthian/apikit` |

EOF
mv README.md README.md.bak
cat /tmp/version_notice.md README.md.bak > README.md && rm README.md.bak /tmp/version_notice.md
```

- [ ] **Step 3: Verify the updated README starts correctly**

```bash
cd /home/ziska/Projects/go/likearthian/apikit && head -15 README.md
```

---

### Task 6: Verify v1 and v2 both build and pass tests

**Files:** All (verification only)

- [ ] **Step 1: Test v1 (root)**

```bash
cd /home/ziska/Projects/go/likearthian/apikit && go vet ./... && go test ./... && go test -race ./...
```

Expected: All pass. v1 has no `_test.go` files, so `go test` will pass with no tests (or report `?` for packages with no tests).

- [ ] **Step 2: Test v2 (v2/ subdirectory)**

```bash
cd /home/ziska/Projects/go/likearthian/apikit/v2 && go vet ./... && go test ./... && go test -race ./...
```

Expected: All pass. v2 has comprehensive tests — you should see actual test results, not just `?` lines.

- [ ] **Step 3: Run gofmt on all Go files**

```bash
cd /home/ziska/Projects/go/likearthian/apikit && gofmt -w api/ transport/ logger/ *.go
cd /home/ziska/Projects/go/likearthian/apikit/v2 && gofmt -w api/ transport/ logger/ *.go
```

---

### Task 7: Commit the merge

**Files:** All staged changes

- [ ] **Step 1: Stage all remaining changes**

```bash
cd /home/ziska/Projects/go/likearthian/apikit && git add -A
```

- [ ] **Step 2: Review what will be committed**

```bash
cd /home/ziska/Projects/go/likearthian/apikit && git status
```

Expected: New files under `v2/`, modified `.gitignore` and `README.md`. No v1 files deleted or modified.

- [ ] **Step 3: Commit**

```bash
cd /home/ziska/Projects/go/likearthian/apikit && git commit -m "feat: merge v2 improvements into v2/ subdirectory

Merge v2-improvements branch via read-tree under v2/ prefix.

Coexisting versions:
- v1 at root (github.com/likearthian/apikit) — legacy support
- v2 at v2/ (github.com/likearthian/apikit/v2) — recommended

v2 brings: tests, TokenVerifier, fixed JWT defaults, consolidated
response API, field-aware binding errors, and migration guide."
```

- [ ] **Step 4: Push to remote**

```bash
cd /home/ziska/Projects/go/likearthian/apikit && git push origin main
```

---

### Task 8: Tag versions

**Files:** None (tags only)

- [ ] **Step 1: Tag v1.0.0**

```bash
cd /home/ziska/Projects/go/likearthian/apikit && git tag -a v1.0.0 -m "v1.0.0: legacy API at root (github.com/likearthian/apikit)"
```

- [ ] **Step 2: Tag v2.0.0**

```bash
cd /home/ziska/Projects/go/likearthian/apikit && git tag -a v2.0.0 -m "v2.0.0: new API in v2/ subdirectory (github.com/likearthian/apikit/v2)"
```

- [ ] **Step 3: Push tags**

```bash
cd /home/ziska/Projects/go/likearthian/apikit && git push origin v1.0.0 v2.0.0
```

- [ ] **Step 4: Verify tags are on remote**

```bash
cd /home/ziska/Projects/go/likearthian/apikit && git ls-remote --tags origin
```

Expected: Shows `v1.0.0` and `v2.0.0` tags pointing to the merge commit.

---

### Task 9: Clean up worktrees

**Files:** Worktree metadata

- [ ] **Step 1: Remove the v2-improvements worktree**

```bash
cd /home/ziska/Projects/go/likearthian/apikit && git worktree remove .worktrees/v2-improvements 2>/dev/null || rm -rf .worktrees/v2-improvements
```

- [ ] **Step 2: Optionally delete the v2-improvements branch**

```bash
cd /home/ziska/Projects/go/likearthian/apikit && git branch -d v2-improvements && git push origin --delete v2-improvements 2>/dev/null
```

Only do this if you're confident the merge is complete. Otherwise, keep the branch for reference.
