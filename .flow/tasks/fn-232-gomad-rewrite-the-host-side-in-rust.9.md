---
satisfies: [R8]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.9 minimize in Rust

Touches: gomad/src/minimize/{proposals.rs,predicate.rs,checkpoint.rs}, cli/minimize.rs

## Description
Port the minimizer: range and suffix proposals per dimension, fresh engine per candidate, predicate (normalized failure, outcome, exact runtime-choice replay, exact simulation replay), per-parent checkpoint under exclusive lock with resume validation of parent, implementation, bounds and accepted artifacts, final publication byte cap. Add to the harness comparing accepted reductions by removed decision identity.

## Acceptance
Harness lane matches accepted reductions on the corpus failures; changed bounds, corrupt state, concurrent writer and changed identities refuse with status 3; a signature-changing candidate is rejected.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
