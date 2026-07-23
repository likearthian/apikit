# APIKit Improvement Plan Design

## Purpose

Create `improvement_plan.html` as an execution-ready runbook for maintainers improving APIKit. The document must translate the repository assessment into dependency-ordered engineering work with exact files, test-first steps, verification commands, breaking-change checkpoints, and migration guidance.

The plan is for engineers performing the work, not stakeholders reviewing a presentation. It may prescribe breaking API cleanup when the replacement and migration path are explicit.

## Selected Approach

Use a hardening-first phased cleanup:

1. Capture current observable behavior in tests.
2. Consolidate the response API.
3. Isolate request-binding responsibilities.
4. Unify authentication verification.
5. Clarify server composition and defaults.
6. Update migration material, documentation, and release checks.

This approach keeps each phase independently testable and releasable. Breaking changes happen only after characterization tests establish the behavior being preserved or intentionally changed.

## Repository Evidence

The runbook will cite these current-state findings:

- The repository contains no mapped `_test.go` files.
- Response concepts such as `BaseResponse`, `PaginationDTO`, and `SuccessResponse` exist in both the root package and `api/`.
- Request binding combines public decoding entry points with reflection traversal and scalar conversion.
- JWT behavior is exposed at both endpoint and HTTP middleware layers.
- `NewServer` connects request lifecycle, error encoding, finalization, and default logging/error handling.
- The graph reports cohesion of `0.09` for both “Request Binding & Decode” and “Generic Endpoint & Transport Types.”
- The graph reports 85 weakly connected nodes and 23 thin communities, indicating documentation gaps, narrow helpers, extraction duplication, or some combination of those factors.

Graph-derived observations are planning evidence, not proof of a defect. Tasks that depend on inferred relationships must begin by confirming the relevant code behavior.

## HTML Document Design

`improvement_plan.html` will be a single self-contained HTML file with embedded CSS and no external assets or runtime dependencies. It must remain readable in a browser, when printed, and with CSS disabled.

The page will contain:

- A title block with scope, audience, recommended sequence, and definition of done.
- A compact current-state assessment with repository evidence.
- A dependency map showing why phases run in the selected order.
- Six phase sections containing detailed task cards.
- A breaking-change inventory and old-to-new API migration table.
- A risk register with prevention and rollback actions.
- A complete verification and release checklist.
- Internal navigation links and stable section IDs.

Each task card will include:

- Objective and rationale.
- Exact files to create, modify, or remove.
- Dependencies on earlier tasks.
- A test-first sequence with failing-test intent.
- Implementation actions with concrete symbol names.
- Exact commands and expected outcomes.
- Compatibility and migration impact.
- A small, focused commit checkpoint.
- Exit criteria that can be evaluated without interpretation.

The HTML will use semantic elements such as `header`, `nav`, `main`, `section`, `article`, `table`, and `footer`. Styling will emphasize scanning and printability rather than decorative presentation.

## Engineering Scope

### Phase 1: Safety Net

Add characterization and contract tests for:

- Root and `api/` response helpers.
- Query and form binding, including pointers, maps, tags, malformed scalar values, and empty inputs.
- JWT validation, claims creation, missing or malformed tokens, context propagation, and error mapping.
- HTTP server error encoding, response headers, and finalizers.
- Logger adapter behavior against the shared `logger.Logger` contract.

Tests must focus on exported behavior and stable contracts instead of reproducing internal implementation.

### Phase 2: Response API Consolidation

Make generic response DTOs and helpers in `api/` the canonical API. Remove the legacy root response model in a breaking release. The plan must enumerate every removed or renamed symbol and provide a direct replacement or explain why no replacement exists.

Migration documentation will include import changes and before/after examples for base, paged, success, and error responses.

### Phase 3: Binding Isolation

Keep public binding functions thin. Separate:

- Entry-point validation and destination handling.
- Struct-tag discovery and traversal.
- Scalar and collection conversion.
- Error construction and field context.

The implementation should remain in the HTTP transport package unless a package boundary provides a concrete testing or dependency benefit. Behavior changes must be intentional, tested, and documented.

### Phase 4: Authentication Unification

Define one token-and-claims verification core in `api/`. Endpoint middleware and HTTP middleware become adapters that:

- Extract input from their transport context.
- Call the common verifier.
- Store verified token and claims in the appropriate context.
- Translate shared failures into layer-specific outputs.

Authentication must fail closed. Signing method, audience, claims factory, malformed-token handling, and error semantics must be tested once at the core and minimally at each adapter.

### Phase 5: Server Composition

Make server dependencies and defaults explicit. The plan will cover logger/error-handler behavior, error encoders, request and response hooks, and finalizers. Construction should validate incompatible or missing configuration where the current API cannot safely supply a default.

The plan should prefer extending the existing option pattern over introducing a second configuration mechanism.

### Phase 6: Migration, Documentation, and Release

Deliver:

- A migration guide with a complete old-to-new symbol table.
- Updated README examples using only the canonical APIs.
- Breaking-change and release-note entries.
- Compiling examples or tests that prevent documentation drift.
- A final verification checklist and release rollback notes.

## Error and Compatibility Policy

- Authentication failures are fail-closed and must not expose sensitive parsing details.
- Binding failures are deterministic and identify the relevant field or input without leaking reflection internals.
- Response encoding must not write conflicting status codes, headers, or response bodies.
- Logger adapters must preserve field propagation and severity behavior defined by the shared interface.
- Removed exported APIs require a documented replacement or explicit removal rationale.
- Compatibility shims are allowed only when they shorten migration without preserving duplicate concepts indefinitely.

## Testing and Verification

Every behavior-changing task follows:

1. Add a focused failing test.
2. Run the narrow test and confirm the expected failure.
3. Implement the smallest change that satisfies the test.
4. Run the narrow test and confirm it passes.
5. Run the affected package suite.
6. Run repository-wide checks.
7. Commit the self-contained change.

Each phase ends with:

```bash
gofmt -w path/to/each/changed.go
go vet ./...
go test ./...
go test -race ./...
```

The `gofmt` path above is illustrative. The implementation plan must replace it
with every exact Go file changed by that phase, name the narrow package or test
command before each repository-wide command, and state the expected success
condition.

## Definition of Done

The improvement program is complete when:

- Characterization and contract tests cover the identified high-risk seams.
- The generic response API is canonical and legacy response symbols are removed.
- Binding responsibilities are separated and edge cases are documented by tests.
- Endpoint and HTTP authentication use the same verification core.
- Server defaults and extension points are explicit and tested.
- The migration guide maps every breaking public API change.
- README examples use canonical APIs and compile under test.
- `gofmt`, `go vet ./...`, `go test ./...`, and `go test -race ./...` succeed.

## Out of Scope

- Replacing go-kit concepts with a new framework.
- Adding unrelated transports or logger backends.
- Redesigning business-service APIs outside this repository.
- Introducing new runtime dependencies without a demonstrated need.
- Preserving duplicate response models indefinitely.
