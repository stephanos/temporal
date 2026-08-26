---
satisfies: [R1, R4, R6]
---
# fn-13-deterministic-go-regression-projections.3 Compose inspector generation and transactional publication

## Description
Turn the pure renderer and verifier contracts into one repository generator command for R1, R4, and R6. This task owns subprocess orchestration, production fixture cross-checking, complete-set validation, and all-or-nothing publication. It does not yet check in outputs or edit the stable Make/documentation surface.

**Size:** M
**Files:** `tools/umpire/internal/generate/regression/generate.go`, `tools/umpire/internal/generate/regression/generate_test.go`, `tools/umpire/cmd/umpire-gen-regressions/main.go`
**Touches:** [tools/umpire/internal/generate/regression/generate.go, tools/umpire/internal/generate/regression/generate_test.go, tools/umpire/cmd/umpire-gen-regressions/main.go]

### Approach
- Follow the existing thin command/`Run` split and inject inspector execution, filesystem reads, and publication for focused failure tests. Invoke the existing model inspector once for the closed manifest identity from the configured repository/model root.
- Cross-check inspector output with the checked-in fixture before rendering: supported format, query identity, sources, and semantic fingerprint must agree. Preserve structured inspector diagnostics while adding stage/identity context and avoid echoing the large artifact.
- Reuse `artifactio.Set` as the deep publication boundary for the exact Go and Markdown paths. Validate the full artifact map and formatted Go before calling it; do not recreate locks, rollback, recovery, symlink, or containment machinery.
- Keep command flags limited to repository/source root and output root so tests/check mode can publish into a temporary tree without exposing arbitrary scenario selection.

### Investigation targets
**Required** (read before coding):
- `tools/common/artifactio/set.go:16-103` — transactional complete-set publication API
- `tools/common/artifactio/set_test.go:13-213` — validation, concurrency, rollback, and recovery guarantees to reuse
- `tools/umpire/cmd/umpire-gen-api/main.go:1-17` — thin generator command convention
- `tools/umpire/internal/generate/api/main.go:10-51` — injectable `Run` and publication orchestration pattern
- `Makefile:1097-1126` — existing inspector execution and diagnostic behavior

**Optional** (reference as needed):
- `tools/common/artifactio/artifact.go:10-39` — single-file staging primitive under the set publisher

### Key context
The shared set publisher already owns path safety, locking, atomic install, rollback, and interrupted-transaction recovery. This generator should supply a fixed complete artifact map and candidate validation, not build a parallel publication subsystem.

### Acceptance

## Acceptance
- [ ] The command runs the existing inspector exactly once for the fixed stable identity and publishes exactly the two rendered artifacts on valid, fixture-matching output.
- [ ] Inspector exit/error/output contradictions, malformed or unsupported artifacts, fixture drift, and render validation failures return non-zero before publication with concise stage and identity context.
- [ ] The generator accepts only source/repository and output roots; no arbitrary scenario, exploration, promotion, runtime, or Umpire3 input is exposed.
- [ ] Publication uses `artifactio.Set` for complete-map validation, safe paths, concurrent-writer rejection, rollback, and interruption recovery; injected failure tests prove prior outputs remain complete.
- [ ] Tests cover missing/unwritable roots, traversal/symlink escape, incomplete maps, subprocess failure, stale fixture, repeated generation equality, and no partial publication.
- [ ] `go test -count=1 -tags test_dep ./tools/umpire/internal/generate/regression` passes.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
