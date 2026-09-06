---
satisfies: [R2, R3, R4]
---
# fn-66-remove-unused-umpire-tooling-after.2 Retire the unused public artifact package and CLI

## Description
Apply the inventory's first retirement slice atomically; .1 owns the consumer and per-case analysis.

**Size:** M
**Files:** tools/umpire/artifact/**, tools/umpire/cmd/umpire-artifact/**, Makefile, active compatibility/component docs, tools/umpire/CLEANUP_INVENTORY.md
**Touches:** [tools/umpire/artifact/**, tools/umpire/cmd/umpire-artifact/**, Makefile, model/Umpire/Property/COMPATIBILITY.md, .plans/UMPIRE4_SPEC_COMPS.md, .plans/UMPIRE4_COMPONENTS.md, tools/umpire/CLEANUP_INVENTORY.md]

### Approach
- Re-anchor the frozen inventory and verify every removal path still has its approved ownership. Remove only the public artifact package, CLI, and exclusively owned fixtures/tests from this slice.
- Remove the command variable, check/check-set Make wrappers and PHONY entries. Update the active public-consumer/command claims; describe the retained Experiment reader accurately without claiming the subsequent internal trim is already complete.
- Record the realized 92-Test/1-Fuzz/27-fixture removal against individual baseline rows. Preserve fn-64 history, full live selectors, existing workflows, generator inputs/outputs, generic Lean promotion and tools/planindex.
- Leave internal artifactv2 intact for .3. Verify both retained generated-view consumers and real retained command checks, using existing tests/generation checks; do not add deletion-only tests or a replacement CLI.

### Investigation targets
**Required:**
- `tools/umpire/CLEANUP_INVENTORY.md` — authoritative frozen removal evidence from .1.
- `tools/umpire/artifact` — full first-slice source/test/fixture ownership.
- `tools/umpire/cmd/umpire-artifact` — retired process entrypoint and tests.
- `Makefile:1000` — exact wrappers and associated variable/PHONY entries.
- `model/Umpire/Property/COMPATIBILITY.md:21` — obsolete public consumer claim.
- `.plans/UMPIRE4_SPEC_COMPS.md:797` — obsolete command row.
- `.plans/UMPIRE4_COMPONENTS.md:199` — current implementation claims.

### Quick commands
```bash
go test -count=1 -tags test_dep ./tools/umpire/internal/artifactv2 ./tools/umpire/cmd/umpire-gen-regression-views ./tools/umpire/regression
make umpire-check-regression-views
make lint-code GOLANGCI_LINT_FIX=false
```

## Acceptance
- [ ] Approved package/CLI/fixtures and direct Make/doc references are removed, with each deleted Test/Fuzz/fixture reconciled to .1 and no dangling active consumer.
- [ ] Internal codecs, retained generators/output bytes, full selectors/workflows, generic promotion, plan-index and comments on retained code remain unchanged.
- [ ] Focused tests and existing generated-view/process checks pass with unchanged retained bytes and diagnostics.
- [ ] Lint equals the frozen baseline minus only approved first-slice deleted-file headers. Current expected subtraction is five headers: 1311 remain, SHA-256 `5afaccdacfc74c7940a6f6d059065481113b32406b8bc0ecb5004d5be93c325a`; re-derive if the reviewed inventory changes, and reject any new or unexplained difference.

## Done summary
Removed the ledger-approved public Artifact package and CLI atomically: 51 files containing 92 Tests, one Fuzz target, and 27 fixtures, plus their direct Make and active documentation references. The retained internal Experiment reader and generated outputs remain unchanged; focused tests and the real generated-view check pass, and lint matches the exact approved five-header subtraction.

Baseline: focused Go tests and `make umpire-check-regression-views` exited 0; Go lint exited 2 with the exact inherited 1,316-header baseline, SHA-256 `aee7770bec1fe01dab8826427cc89e9ffa7e764fbac25ce6b68bf5f2e3c0b077`.

Verification: focused Go tests and `make umpire-check-regression-views` exited 0; Go lint exited 2 with exactly 1,311 headers, SHA-256 `5afaccdacfc74c7940a6f6d059065481113b32406b8bc0ecb5004d5be93c325a`, removing only the five approved `tools/umpire/artifact/result_test.go` headers and adding none.

No commit was created because the user requires commits to remain under their control.

stage: impl-review - ran [2026-09-06T15:36Z..2026-09-06T15:39:36Z] (snapshot correction -> SHIP)
stage: plan-sync - skipped(config: planSync.enabled != true)

## Evidence
- Commits:
- Tests: TMPDIR=/private/tmp/fn66-task2 go test -count=1 -tags test_dep ./tools/umpire/internal/artifactv2 ./tools/umpire/cmd/umpire-gen-regression-views ./tools/umpire/regression (baseline exit 0), TMPDIR=/private/tmp/fn66-task2 make umpire-check-regression-views (baseline exit 0), TMPDIR=/private/tmp/fn66-task2 make lint-code GOLANGCI_LINT_FIX=false (baseline exit 2; accepted exact inherited 1316 headers, SHA-256 aee7770bec1fe01dab8826427cc89e9ffa7e764fbac25ce6b68bf5f2e3c0b077), TMPDIR=/private/tmp/fn66-task2 go test -count=1 -tags test_dep ./tools/umpire/internal/artifactv2 ./tools/umpire/cmd/umpire-gen-regression-views ./tools/umpire/regression (exit 0), TMPDIR=/private/tmp/fn66-task2 make umpire-check-regression-views (exit 0), TMPDIR=/private/tmp/fn66-task2 make lint-code GOLANGCI_LINT_FIX=false (exit 2; accepted exact 1311 headers, SHA-256 5afaccdacfc74c7940a6f6d059065481113b32406b8bc0ecb5004d5be93c325a; only five approved headers removed, zero added), flowctl gate classify --base ff9ea9827157255068a87086630651a43cc01060 (exit 1: FULL, unmatched .plans/UMPIRE4_COMPONENTS.md), task-owned source tree 2642f10988940f5a6935f7e763cca4de9deba3f6 equals reviewed tree 2642f10988940f5a6935f7e763cca4de9deba3f6
- PRs: