---
satisfies: [R1, R4, R6]
---

# fn-13-deterministic-go-regression-projections.3 Compose inspector generation and transactional publication
## Description
Turn the pure renderer and verifier contracts into one repository generator command for R1, R4, and R6. This task owns subprocess orchestration, complete production fixture cross-checking, complete-set validation, and all-or-nothing publication. It does not yet check in outputs or edit the stable Make/documentation surface.

**Size:** M
**Files:** `tools/umpire/internal/generate/regression/generate.go`, `tools/umpire/internal/generate/regression/generate_test.go`, `tools/umpire/cmd/umpire-gen-regressions/main.go`
**Touches:** [tools/umpire/internal/generate/regression/generate.go, tools/umpire/internal/generate/regression/generate_test.go, tools/umpire/cmd/umpire-gen-regressions/main.go]

## Approach
- Follow the existing thin command/`Run` split and inject inspector execution, filesystem reads, and publication for focused failure tests. Invoke the existing model inspector once for the closed manifest identity from the configured repository/model root.
- Extract projection records independently from inspector output and the checked-in fixture, then require whole-record equality across supported format, query identity, canonical sources, property identities, observation-requirement identities, and semantic fingerprint before rendering. Preserve structured inspector diagnostics while adding stage/identity context and avoid echoing the large artifact.
- Reuse `artifactio.Set` as the deep publication boundary for the exact Go and Markdown paths. Validate the full artifact map and formatted Go before calling it; do not recreate locks, rollback, recovery, symlink, or containment machinery.
- Keep command flags limited to repository/source root and output root so tests/check mode can publish into a temporary tree without exposing arbitrary scenario selection.

## Investigation targets
**Required** (read before coding):
- `tools/common/artifactio/set.go:16-103` — transactional complete-set publication API
- `tools/common/artifactio/set_test.go:13-213` — validation, concurrency, rollback, and recovery guarantees to reuse
- `tools/umpire/cmd/umpire-gen-api/main.go:1-17` — thin generator command convention
- `tools/umpire/internal/generate/api/main.go:10-51` — injectable `Run` and publication orchestration pattern
- `Makefile:1097-1126` — existing inspector execution and diagnostic behavior

**Optional** (reference as needed):
- `tools/common/artifactio/artifact.go:10-39` — single-file staging primitive under the set publisher

## Key context
The shared set publisher already owns path safety, locking, atomic install, rollback, and interrupted-transaction recovery. This generator should supply a fixed complete artifact map and candidate validation, not build a parallel publication subsystem. Fingerprint equality alone is insufficient: every metadata field rendered by either output must match between live inspector and fixture.

## Acceptance
- [ ] The command runs the existing inspector exactly once for the fixed stable identity and publishes exactly the two rendered artifacts on valid, fixture-matching output.
- [ ] Inspector and fixture records are compared across format, identity, canonical sources, properties, observation requirements, and fingerprint; isolated stale-fixture tests mutate each displayed field without updating semantic identity and still fail before publication.
- [ ] Inspector exit/error/output contradictions, malformed or unsupported artifacts, fixture drift, nonexistent provenance, and render validation failures return non-zero before publication with concise stage and identity context.
- [ ] The generator accepts only source/repository and output roots; no arbitrary scenario, exploration, promotion, runtime, or Umpire3 input is exposed.
- [ ] Publication uses `artifactio.Set` for complete-map validation, safe paths, concurrent-writer rejection, rollback, and interruption recovery; injected failure tests prove prior outputs remain complete.
- [ ] Tests cover missing/unwritable roots, traversal/symlink escape, incomplete maps, subprocess failure, repeated generation equality, and no partial publication.
- [ ] `go test -count=1 -tags test_dep ./tools/umpire/internal/generate/regression` passes.

## Done summary
Composed the fixed regression inspector, strict live-versus-fixture projection cross-check, deterministic rendering, and exact two-file transactional publication behind a repository generator command. Added injected coverage for inspector contradictions, every stale displayed field, unsafe roots and paths, incomplete render sets, concurrent publication, deterministic repetition, and preservation of prior outputs.

baseline: green (go test -count=1 -tags test_dep ./tools/umpire/... passed pre-edit; make umpire-check-regression reused receipt 959674ea)
GATE_SKIPPED:unittest:green-receipt 959674ea - baseline reused from prior post-gate pass
stage: impl-review - ran [2026-08-26T05:16:13Z..2026-08-26T05:19:19Z] (SHIP)
## Evidence
- Commits: e1a635b275b58da766f4023f48cadda6543dacc5
- Tests: GATE_SKIPPED:unittest:green-receipt 959674ea - baseline reused from prior post-gate pass, go test -count=1 -tags test_dep ./tools/umpire/internal/generate/regression, mise exec -- go run -tags test_dep ./tools/umpire/cmd/umpire-gen-regressions --repository-root . --output-root /tmp/tmp.3AOEqWP7jQ, go test -count=1 -tags test_dep ./tools/umpire/..., make umpire-check-regression
- PRs: