---
satisfies: [R6, R8, R10]
---
# fn-22-deterministic-replay-semantic.8 Close the replay matrices, live proof, gates and documentation

## Description
Complete the admission, key, semantic replay, rerun, reduction, evidence core, proposal, cancellation, limit and output matrices; run the negative control end to end through `umpire-replay run` against the test cluster (reproduced, irreducible or minimized, proposal compiled and written under a scratch root) in the live suite; reconcile the documentation with the Case Runtime, amend the UMPIRE4 spec's Exploration section for the replay classes and the key, and remove active references to replay bundles, Run Evaluation, caller-closure runtime support and SDK replay as proof.

### Quick commands
`cd model && lake build && LEAN_NUM_THREADS=1 make -C .. lint-model; make umpire-check-goldens umpire-check-inventory umpire-check-model-module-index umpire-check-exploration-bridge; go test -count=1 -tags test_dep ./tools/umpire/... ./common/testing/testpilot/...; make lint-code-fast`

**Size:** M
**Files:** `tools/umpire/replay/**`, `tests/testpilot_nexus_control_case_test.go`, `model/README.md`, `model/ARCHITECTURE.md`, `.plans/UMPIRE4_SPEC.md`, `.plans/UMPIRE4_COMPONENTS.md`, `Makefile`

### Re-plan note (2026-09-22)
Rewritten on fn-85, fn-86, fn-87 and fn-33 after the first plan's MAJOR_RETHINK; see the spec's **Re-plan** section. Start only after the spec's fresh plan review.
## Acceptance
- [ ] Focused Lean, Go, `-tags test_dep`, integration, formatting and lint gates pass, and the live negative-control proof runs through the command.
- [ ] Docs keep the three replay classes, the key against the identity, and every retired or deferred boundary.
- [ ] Existing comments are preserved or reworded only where the invariant they describe changed.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
