---
satisfies: [R4, R5]
---
# fn-213-modelsuite-in-rust-examples-and.4 Kettle tests.rs: the Scala suite's questions in examples and explorations

## Description
Kettle's `umpire/tests/fixtures/kettle/tests.rs` with `mod examples` and `mod explorations` as plain `#[test]` functions, asking the questions of `model/archived/framework/samples/Kettle.test.scala` that the surface admits: the press/heat/depress enabledness table from the coldest (-40) and room (20) starts; the 99-to-100 boundary and the `>=` failover guard (101 fails over, `fails_over_at_boiling` then fails); `boils` from room temperature reaching the beep in 81 steps within `never_switched_off`; `never_beeps` refuted by press, 80 heats, failover; a budget of 10 answering unknown; vacuity of `fails_over_at_boiling` when the heater is excluded. No registration, no runner beyond `cargo test`; the existing gate runs it. Inner-loop and gate budgets of fn-210 R30 hold; paste `make model-time` into the done summary.

**Touches:** `model/umpire/tests/fixtures/kettle/tests.rs`, `model/README.md` (how to write a Model's tests).

**Cross-spec prerequisite:** fn-214.3's committed integrated checkpoint provides the progress questions and registered-query seam. This task supplies their ergonomic suite presentation, including claim selection and `path!` notation. The conductor must verify that prerequisite before claiming this task. The CLI refuses cross-spec task edges; do not invent one or replace this with a cyclic whole-spec dependency. Illustrative `Model::boils` notation does not imply fn-214 already exposes a public callable claim descriptor.
## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs: