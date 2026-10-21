---
satisfies: [R8, R13, R14]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.9 minimize in Rust

Touches: gomad/src/minimize/{proposals.rs,predicate.rs,checkpoint.rs}, cli/minimize.rs

## Description
Port the minimizer: range and suffix proposals per dimension, fresh engine per candidate, predicate (normalized failure, outcome, exact runtime-choice replay, exact simulation replay), per-parent checkpoint under exclusive lock with resume validation of parent, implementation, bounds and accepted artifacts, final publication byte cap. Add to the harness comparing accepted reductions by removed decision identity.

Producer integration dependency from fn-232.5: typed authenticated evidence/inspection.json is bound through record.streams, so adding minimization metadata changes the R2 full-record-minus-seed failure_signature. The minimizer must keep the immutable original/parent behavioral predicate identity distinct from the decorated artifact's record identity and export the original predicate/parent signatures separately for consumers. Preserve actual Go/Rust accepted-reduction and failure-group parity; do not normalize away stream/hash fields or alter R2's canonical projection to manufacture a stable decorated signature. Integrate the inspect producer's forthcoming handover contract without inventing exact types here. This assigns task .9 producer integration responsibility and claims no .5 minimizer pass, .9 completion or acceptance waiver.

## Acceptance
The owned behavior has reviewed golden fixtures, repeated-execution agreement and byte-identical second regeneration (R14); public verbs also run their fresh Go/Rust semantic parity lane (R13).
Harness lane matches accepted reductions on the corpus failures; changed bounds, corrupt state, concurrent writer and changed identities refuse with status 3; a signature-changing candidate is rejected.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
