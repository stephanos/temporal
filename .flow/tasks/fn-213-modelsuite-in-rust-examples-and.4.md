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
Kettle's `model/umpire/tests/fixtures/sample_kettle/tests.rs` gains `mod examples` (3 tests) and `mod explorations` (6 tests), plain `#[test]` functions returning `Result<(), Refusal>`, discovered by `cargo test` in the umpire-tests `fixtures` binary (umpire-suite added as a dev-dependency). The `fixture!` hook is unchanged; no golden changed.

Asked (R4): the press/heat/depress enabledness table from -40 and 20; 99->100 boundary, heat disabled at 100, failover beeps, beeps_only_when_off holds and never_beeps fails at the beep, press re-arms; the `>=` guard (failover at 101) and `fails_over_at_boiling` refuted from (101, on); boils verified from room with longest 81 (141 from coldest and from all -40..=40 starts), plus `repeat(80, Heat)` and `Run::along(path![on, .., beeping])` = 81 steps, no depress; never_beeps refuted by press, 80 heats, failover (and `find` = 82 steps); budget 10 -> cut, every claim Unknown, `refuted` -> `Refusal::Unknown`; a counterexample before a cut stands (from room + (99, on), budget 12); a heating-and-beeping start refutes beeps_only_when_off at once.

Not admitted (listed in model/README.md): vacuity with the heater excluded, the whole-Kettle cycle, only(press) deadlock, the 80-step deadline and the fair(heat) cycle (a Space chooses starts and budget, never actions or assumptions; these remain in the kettle_progress* fixtures); claim truth tables over non-step before/after pairs; boils metadata (no claim descriptors); the Scala report's prose line and empty query list.

Deviation: the Scala "one state short of the whole graph" cut question does not carry over: from room the beep is the deepest (last-admitted) state, so a budget of reached-1 answers Unknown. The cut-but-refuted question uses two starts instead; README notes it. `find`/`along` use concrete first waypoints, so the greedy leg caveat does not show.

Gate: `make model-check` passes. `make model-time`:
```
timing-controls 0.015s; capability-controls 0.074s; source-boundary 0.083s; protobuf-freshness 0.055s;
temporal-protobuf-freshness 0.053s; fmt 0.180s; clippy 0.081s; docs 0.139s; tests 2.221s;
wasm32-wasip1 0.053s; core-wasm-clippy 0.069s; wasm-guest 0.068s; go-host 1.097s; temporal-go-host 6.490s
inner loop (edit+test)            2.221s
gate total                       10.882s  (soft target 30s)
inner loop                        2.221s  (soft target 10s)
```

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 74adbab0c4
- Tests: make model-check, make model-time, cargo test -p umpire-tests
- PRs: