---
satisfies: [R6, R7]
---
# fn-69-extract-testpilot-from-umpire.7 Migrate remaining Umpire producers and generated Case consumers

## Description
Retarget Umpire-owned Case generation, conformance, fixtures, Lean descriptor projection, and remaining active protocol consumers to Testpilot while keeping model authoring under Umpire (R6, R7).

**Size:** L — coordinated generated identity cutover; inputs, generators, checked outputs, and consumers must change together
**Files:** Umpire Case generators/schema tests, `common/testing/testpilot/conformance_test.go`, `common/testing/testpilot/testdata/case-runtime-conformance/**`, remaining active Testpilot proto importers, Makefile paths, migration ledger
**Touches:** [tools/umpire/caseartifact/**, tools/umpire/cmd/umpire-gen-case-runtime-conformance/**, tools/umpire/cmd/umpire-gen-lean-api/**, tools/umpire/conformance_test.go, tools/umpire/testdata/case-runtime-conformance/**, common/testing/testpilot/conformance_test.go, common/testing/testpilot/testdata/case-runtime-conformance/**, Makefile, .flow/artifacts/fn-69-extract-testpilot-from-umpire/migration-ledger.md]

### Approach
- Replace caseartifact callers with top-level Testpilot ingestion while keeping Lean Producer/compiler and generator commands under Umpire.
- Retain the task-5 Lean/API and functional-fixture identity cutover; regenerate the generic conformance Case corpus through established commands and inventory each remaining descriptor full name, type URL, embedded `NamedType`, and derived-identity change.
- Move generic conformance ownership to Testpilot and retarget generator/check roots without changing the six behavioral classes or ordinary wire values.
- Migrate remaining production/test imports in coherent groups and reject any unexplained generated diff; do not add a broad drift gate.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/cmd/umpire-gen-case-runtime-conformance/generate.go:15-180` — fixture generation and ownership
- `tools/umpire/cmd/umpire-gen-lean-api/case_schema_test.go` — generated Lean schema checks
- `model/Temporal/CaseRuntime.lean:30-38` — embedded protocol type identity
- `model/Temporal/API/Types.lean:1506-1792` — generated namespace baseline
- `Makefile:1010-1067` — generation and conformance targets
- `.flow/memory/declined/generated-api-drift-verification.md` — existing-gate-only boundary

### Acceptance

## Acceptance
- [ ] Umpire model authoring/Producer/generator commands remain Umpire-owned while all generated Case protocol values and Go imports target Testpilot; old caseartifact is removed after its callers migrate.
- [ ] The generic six-class corpus is regenerated only by established commands after the task-5 Lean/API and functional-fixture cutover; every namespace-derived byte/identity change matches the task-1 allowlist.
- [ ] Ordinary wire values, six conformance classes, Case admission, Run/Verdict behavior, generated output determinism, and Umpire authoring semantics remain exact.
- [ ] Focused generator/schema/conformance tests and existing generation checks pass, with no new drift system and no stale active Umpire proto import outside the final temporary runtime owner.

## Done summary
Retargeted the Umpire-owned generic Case renderer and generator to Testpilot ingestion, regenerated and moved the deterministic six-class corpus to common Testpilot, moved conformance ownership without a functional-adapter dependency, migrated Lean schema checks and selectors, removed caseartifact after caller closure, and recorded every refined-schema and namespace-derived identity change in the migration ledger.

Verification passed for focused generator/schema/conformance tests, all common Testpilot packages, all functional Testpilot packages, all remaining Umpire packages, proto and Lean API generation, Temporal.CaseRuntimeTests, the deterministic conformance check, the 325-target aggregate regression gate, import closure, and git diff checks. Baseline generation was red only for the seven task-owned NamedType substitutions; an unmodified-CGO baseline also could not find stddef.h, while the required serial CGO-disabled selectors passed. The task-scoped no-fix golangci run stayed within 657 MiB but exited 1 on inherited importas/revive/staticcheck findings already present in the dependency snapshot; it made no changes.

stage: impl-review - ran | SHIP after one valid P2 dependency-direction fix
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: TMPDIR=/private/tmp go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-gen-case-runtime-conformance, GOFLAGS='-p=1' CGO_ENABLED=0 TMPDIR=/private/tmp go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-gen-case-runtime-conformance ./tools/umpire/cmd/umpire-gen-lean-api, GOFLAGS='-p=1' CGO_ENABLED=0 TMPDIR=/private/tmp go test -count=1 -tags test_dep ./common/testing/testpilot/..., GOFLAGS='-p=1' CGO_ENABLED=0 TMPDIR=/private/tmp go test -count=1 -tags test_dep ./tests/testcore/testpilot/..., GOFLAGS='-p=1' CGO_ENABLED=0 TMPDIR=/private/tmp go test -count=1 -tags test_dep ./tools/umpire/..., TMPDIR=/private/tmp make proto, TMPDIR=/private/tmp make umpire-gen-lean-api, cd model && TMPDIR=/private/tmp mise exec -- lake build Temporal.CaseRuntimeTests, GOFLAGS='-p=1' CGO_ENABLED=0 TMPDIR=/private/tmp make umpire-check-case-runtime-conformance, GOFLAGS='-p=1' GOLANGCI_LINT_FIX=false CGO_ENABLED=0 TMPDIR=/private/tmp make umpire-check-regression, GOFLAGS='-p=1' CGO_ENABLED=0 TMPDIR=/private/tmp go list -tags test_dep -deps -test ./common/testing/testpilot (no tests/testcore/testpilot dependency), git diff --check, SCOPED_LINT_FAILED: GOLANGCI_LINT_FIX=false task-package invocation exited 1 on inherited importas/revive/staticcheck findings; no auto-fix, max memory 657 MiB
- PRs:
