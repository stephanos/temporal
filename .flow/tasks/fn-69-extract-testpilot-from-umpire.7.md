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
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: