---
satisfies: [R8]
---
# fn-18-versioned-umpire-artifact-boundary.11 Expose artifact commands and synchronize persistence documentation

## Description
Complete R8 with exact CLI/root validation surfaces, integrated fixture/corruption coverage, public facades, and honest roadmap status.

**Size:** M
**Files:** `tools/umpire/cmd/umpire-artifact/main.go`, `tools/umpire/artifact/integration_test.go`, `Makefile`, `model/Umpire/Artifact.lean`, `model/Umpire.lean`, `model/README.md`, `model/Umpire/ARCHITECTURE.md`, `.plans/UMPIRE4_COMPONENTS.md`
**Touches:** [tools/umpire/cmd/umpire-artifact/main.go, tools/umpire/artifact/integration_test.go, Makefile, model/Umpire/Artifact.lean, model/Umpire.lean, model/README.md, model/Umpire/ARCHITECTURE.md, .plans/UMPIRE4_COMPONENTS.md]

### Approach
- Implement the exact check/check-set/publish-set/migrate-set grammar and canonical stdout/stderr/status contracts from the parent spec.
- Wire every vertical Lean artifact module through `model/Umpire/Artifact.lean` and the top-level facade; retain public comments and add no compatibility aliases.
- Add root-only check targets with required variable validation; checks never publish or migrate and no model-local Makefile changes.
- Exercise every artifact family and one complete conformance-shaped set through cross-language golden, mutation, fuzz-seed, interruption, and direct/root command tests.
- Document strict persistence, exact field/string/byte limits, version/migration policy, artifact versus semantic references, identity formulas, atomic visibility, and the transport-only boundary.
- Update C4/C7/C8 roadmap status only for completed codecs/persistence, retaining runtime, conformance, replay, and qualification gaps.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Tool/Inspect.lean:46-85`
- `Makefile:988-1032,1254`
- `model/README.md:130-165` and `model/Umpire/ARCHITECTURE.md:207-235`
- `.plans/UMPIRE4_COMPONENTS.md:23-60,118-140,253-390`
- all prior tasks in this spec

### Acceptance
- [ ] Direct and root checks return identical canonical status/bytes and checks never modify the checkout.
- [ ] Migration reports `no-migration-route` for every production v1 source until a real route exists.
- [ ] `model/Umpire/Artifact.lean` publicly imports all vertical modules and focused facade builds pass.
- [ ] Docs distinguish canonical values, persisted admission, semantic references, and still-unimplemented producers/consumers.
- [ ] No runtime, interpretation, scoring, replay/promotion, CI, model-local Makefile, or prohibited legacy dependency is added.

## Acceptance
- [ ] R8 commands, integration verification, public facades, docs, and roadmap status are complete.
- [ ] All focused suites and root checks pass.
- [ ] Existing comments remain preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
