---
satisfies: [R3, R4]
---
# fn-223-gomad-delete-the-native-backend.3 Record WASM identity changes, requalify the candidate and refresh docs

## Description
Rerun the identity probes used in fn-184 and fn-219, comparing against the closed fn-219 candidate. Record every retained identity and preparation binding in .flow/artifacts/fn223-closeout-<date>.md. Account for the shared-cluster digest rotation caused by the declared glue removal, its composed WASM cluster digest and proven transitive boot, retained-execution, scenario-provenance, BuildKey and choice-execution changes. Record exact required module/tag edits behind any selected source-inventory changes; measure compilation keys and module digests separately. All remaining implementation identities stay unchanged, with complete hash coverage and current-source digests. On the final frozen changed candidate, run all current WASM gates required by QUALIFICATION.md, including runtime, mandatory workload/retained-replay qualification and make -C tools/gomad test-wasm; retain commands, numeric exits and source/helper/runtime/module/model identities. fn-128 and fn-149 were already closed as won't-do on 2026-10-19 (see their Closure sections); cite that in the close-out. Update tools/gomad/README.md, ARCHITECTURE.md and MILESTONES.md so native appears only as removed history. Apply AGENTS.md closed-owner cleanup, preserving inputs still required by open owners such as fn-185 and their verified references before deleting closed-owner copies.

**Size:** M
**Files:** .flow/artifacts/fn223-*.md, tools/gomad/README.md, tools/gomad/ARCHITECTURE.md, tools/gomad/CLI.md, tools/gomad/internal/backends/wasm/README.md, MILESTONES.md, AGENTS.md
**Touches:** Final identity ledger and qualification evidence; retained guides' historical wording and qualification links; milestone and agent instructions.

## Acceptance
- [ ] Close-out ledger records exact before/after identities and source causes; only declared and proven rotations occur, with no stale digest or weakened hash coverage
- [ ] Runtime, mandatory workload/retained-replay qualification and test-wasm pass on the frozen changed candidate with numeric exits and all required identities retained
- [ ] Existing WASM CLI fixtures preserve bytes and statuses
- [ ] All retained README, ARCHITECTURE, CLI and WASM guides, MILESTONES and AGENTS describe native only as removed history; task .2's zero-reference check remains green

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
