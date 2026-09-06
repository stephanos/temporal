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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
