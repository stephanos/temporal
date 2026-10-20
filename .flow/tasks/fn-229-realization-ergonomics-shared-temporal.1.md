---
satisfies: [R1]
---
# fn-229-realization-ergonomics-shared-temporal.1 Runtime constructors and Candidate outcomes in umpire-wasm; Realize::matches removed

## Description
Add `From<A> for Transition<A>` (choice zero), `Projection::step` and
`Projection::steps`, `Candidate { path, outcome: Option<Outcome> }` inside
`Projection::Candidates`, and `AnswerPlan::new`. Remove `Realize::matches`;
the Runtime compares each candidate's expected outcome with the replayed final
edge outcome (`None` matches any). Adapt every trait implementation, including
stock/control and test worlds, and minimally adapt the activity Realization so
it compiles and behaves identically. Wire goldens, Go host parity, the dedicated
slice and all Model goldens stay unchanged. Preserve the prior matches behavior
in negative controls. If removal changes any actual judgment, keep matches and
record the concrete reason in DESIGN.md rather than weakening judgment.

**Touches:** `model/umpire/wasm/**`, `model/temporal/features/activity/temporal.rs`, `model/temporal/realization/tests/**`, `model/DESIGN.md`
**Quick:** `cd model && mise exec -- cargo test -p umpire-wasm -p umpire-temporal`; `cd model && mise exec -- cargo clippy -p umpire-wasm -p umpire-temporal --all-targets -- -D warnings`
**Full gates:** `make model-check`; `make model-check UMPIRE_BACKENDS=require`; `make umpire-check-model-slice`.
DESIGN.md is writable only for the specified evidence-backed compatibility fallback.
## Acceptance
- [ ] Constructors and Candidate present; matches removed from Realize
- [ ] Expected-outcome mismatch is Unexplained as before
- [ ] Wire goldens, Go parity, slice and goldens unchanged

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
