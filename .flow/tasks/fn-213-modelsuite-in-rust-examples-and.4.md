---
satisfies: [R4, R5]
---
# fn-213-modelsuite-in-rust-examples-and.4 Kettle tests.rs: the Scala suite's questions in examples and explorations

## Description
Kettle's `umpire/tests/fixtures/kettle/tests.rs` with `mod examples` and `mod explorations` as plain `#[test]` functions, asking the questions of `model/archived/framework/samples/Kettle.test.scala` that the surface admits: the press/heat/depress enabledness table from the coldest (-40) and room (20) starts; the 99-to-100 boundary and the `>=` failover guard (101 fails over, `fails_over_at_boiling` then fails); `boils` from room temperature reaching the beep in 81 steps within `never_switched_off`; `never_beeps` refuted by press, 80 heats, failover; a budget of 10 answering unknown; vacuity of `fails_over_at_boiling` when the heater is excluded. No registration, no runner beyond `cargo test`; the existing gate runs it. Inner-loop and gate budgets of fn-210 R30 hold; paste `make model-time` into the done summary.

**Touches:** `model/umpire/tests/fixtures/kettle/tests.rs`, `model/README.md` (how to write a Model's tests).

**Cross-spec prerequisite (2026-10-19):** fn-214 (progress vocabulary: `only`, `fair`, `.within(n)`, named verdicts) must be done first; the deadlock, cycle and missed-deadline questions here use `Model::boils.assuming(..)` and `Model::boils.within(80)` from that spec; write the claims in fn-215's `path!` notation where the tests restate one. Not a Flow dependency because task deps are spec-local.
## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
