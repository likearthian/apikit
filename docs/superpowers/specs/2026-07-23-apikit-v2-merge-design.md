# APIKit v2 Merge Design

## Purpose

Merge the v2-improvements worktree branch into main while preserving v1 at root and placing v2 in a `v2/` subdirectory. Both major versions coexist on main following Go semantic import versioning conventions. v1 consumers are unaffected; v2 consumers import `github.com/likearthian/apikit/v2`.

## Context

- **v1 (main)** — module `github.com/likearthian/apikit`, no `/v2` prefix. Zero tests, insecure JWT defaults, duplicate response models.
- **v2 (v2-improvements worktree)** — module `github.com/likearthian/apikit/v2`, 29 commits ahead. Comprehensive tests, security fixes, consolidated API, renamed symbols.
- **External consumers** — legacy apps depend on v1. v1 must remain importable for at least 1 year.
- **No existing tags** — no version tags exist yet on the repository.

## Final Layout

```
apikit/
├── api/                    ← v1 (unchanged)
├── transport/              ← v1 (unchanged)
├── logger/                 ← v1 (unchanged)
├── logging.go              ← v1 (unchanged)
├── response.go             ← v1 (unchanged)
├── go.mod                  ← v1: module github.com/likearthian/apikit
├── go.sum                  ← v1
├── README.md               ← updated: explains v1 + v2 availability
├── .gitignore              ← updated to include /v2 (worktree-safe)
│
└── v2/                     ← v2 code from v2-improvements
    ├── api/
    ├── transport/
    ├── logger/
    ├── logging.go
    ├── example_test.go
    ├── go.mod              ← module github.com/likearthian/apikit/v2
    ├── go.sum
    ├── README.md
    ├── CHANGELOG.md
    └── MIGRATION.md
```

## Merge Procedure

1. **Read-tree**: `git read-tree --prefix=v2/ v2-improvements` — places the entire v2 tree under `v2/` on the current index without touching v1 files at root.
2. **Cleanup**: Remove `.worktrees/`, `improvement_plan.html`, `graphify-out/`, and any v2 files that landed at root from the index.
3. **Update `.gitignore`**: Add entries for `/v2` worktree artifacts if needed.
4. **Update root `README.md`**: Add a note that v1 is at root, v2 is in `v2/`, link to `v2/MIGRATION.md`.
5. **Commit**: Single merge-style commit on main.
6. **Verify**: `go test ./...` passes for both v1 and `v2/`.
7. **Tag**:
   - `v1.0.0` → the merge commit (v1 at root)
   - `v2.0.0` → the merge commit (v2 at `v2/`)

## Tag Strategy

| Tag | Module path | Consumers see |
|---|---|---|
| `v1.0.0` | `github.com/likearthian/apikit` | Root `go.mod` |
| `v2.0.0` | `github.com/likearthian/apikit/v2` | `v2/go.mod` |

Go's module proxy resolves `github.com/likearthian/apikit/v2@v2.0.0` to the `v2/` subdirectory automatically when the root module is at the same tagged commit.

## Consumer Impact

- **v1 consumers**: No change. `go.mod` continues referencing `github.com/likearthian/apikit`.
- **v2 consumers**: Change import to `github.com/likearthian/apikit/v2`, follow `v2/MIGRATION.md`.
- **New consumers**: Use v2.

## Risks

- **Read-tree can be tricky** — dry-run first with `git read-tree --prefix=v2/ -n -u v2-improvements` to preview.
- **Worktree artifacts** — the v2-improvements worktree directory `.worktrees/v2-improvements` must not be included in the read-tree.
- **Post-merge tests must pass** — both v1 and v2 must compile and pass tests from a clean checkout.

## Verification

```bash
# After merge, from a clean checkout of main:
go test ./...                    # v1 tests
cd v2 && go test ./...           # v2 tests
go test -race ./...              # v1 race
cd v2 && go test -race ./...     # v2 race
go vet ./...
cd v2 && go vet ./...
```
