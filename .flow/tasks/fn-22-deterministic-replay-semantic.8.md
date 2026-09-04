---
satisfies: [R6, R8, R10]
---

# fn-22-deterministic-replay-semantic.8 Close replay matrices, gates, and documentation
## Description
Complete admission, identity, preparation, rerun, reduction, EvidenceCore, proposal, cancellation, cleanup, limit, and output mutation matrices. Reconcile replay documentation with the Case Runtime and remove active references to replay bundles, Run Evaluation, caller-closure runtime support, and SDK replay as proof.

**Size:** M
**Touches:** `tools/umpire/replay/**`, `docs/**`, `.plans/UMPIRE4_COMPONENTS.md`, `Makefile`

## Acceptance
- [ ] Focused Lean, Go, `-tags test_dep`, integration, regression, formatting, and lint gates pass with the full `^TestUmpire` selector where applicable.
- [ ] Docs preserve the three replay classes and all retired/deferred boundaries.
- [ ] Existing comments are preserved or reworded only where the described invariant changes.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
