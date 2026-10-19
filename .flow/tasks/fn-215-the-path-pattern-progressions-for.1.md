---
satisfies: [R1, R2, R3]
---

# fn-215-the-path-pattern-progressions-for.1 Path and path!: pattern, claim restriction, Kettle in path! form, reading rules
# fn-215-the-path-progression-notation-for.1 path! macro, Kettle in path! form, reading rules in README

## Description
`Path<S>` and `path!` in `umpire-dsl` (`progress.rs` or a new `path.rs`): waypoints from `bool` / `Fn(&S) -> bool` / `S`, gaps `..` and `..=n`; `macro_rules!` arms for two or more waypoints, catch-all arms for the refusals (no gap; leading or trailing gap), each pinned by a trybuild `.stderr` fixture in `dsl/tests/ui/`. `Progress` wraps a two-waypoint `Path`; `whenever(..).eventually(..)` / `.within(n)` build the same value and stay public. `#[machine]` refuses a claim with more than two waypoints (`a progress claim has one trigger and one target; ask a longer progression with find or Run::along`), pinned by a trybuild fixture in `dsl/macros/tests/ui/`. Rewrite Kettle's `boils` as `path![self.heating_on, .., self.beeping].assuming(Self::never_switched_off())` and the `machine` macro fixture likewise; goldens byte-identical. `README.md`: the reading rules beside Kettle (claim: every admitted path; `find`/`Run::along`: some path, shortest per leg; `..=n` bounds a gap; nothing is said between or after waypoints). DESIGN.md's comparison shows the Scala claim and the `path!` claim side by side. `find`/`Run::along` themselves are fn-213.3/.2 (R4), not this task.

**Prerequisite:** fn-214 done. **Touches:** `model/umpire/dsl/src/**`, `dsl/tests/ui/`, `dsl/macros/src/machine.rs`, `dsl/macros/tests/ui/`, `model/umpire/tests/fixtures/kettle/model.rs`, `dsl/macros/tests/fixtures/machine`, `model/README.md`, `model/DESIGN.md`. **Feedback loop (R30):** paste `make model-time` into the done summary.
## Acceptance
- [ ] TBD

## Done summary
`Path<S>`, `Waypoint<S>` (`bool` | closure | state), `Gap` (`..` | `..=n`) and `path!` (`macro_rules!`) in `umpire-dsl` (`dsl/src/path.rs`). The macro refuses no gap between waypoints, a leading gap and a trailing gap, each with a trybuild `.stderr` in `dsl/tests/ui/path_*.rs`. Closure waypoints go straight to `then_judged`, so `|k| ..` infers its parameter type from an earlier waypoint.

`#[machine]`: the module-local `LeadsTo` now holds trigger and target `Waypoint<State>`s, judged per state by the evaluator. `From<Path<State>>` and a local `.assuming(..)` on `Path` are generated, and every progress body is wrapped in `From::from`, so a body returns a `path!` or a built `LeadsTo`. `whenever(..).eventually(..).within(n)` stays public. A literal chain in a claim is refused at compile time (`dsl/macros/tests/ui/machine_progress_chain.rs`). A chain built elsewhere is refused by `Path::claim` with the same message, which the checker reports as a claim panic.

Kettle's `boils` is `path![self.heating_on, .., self.beeping].assuming(Self::never_switched_off())`. Every Kettle golden is byte-identical, including the lift and TLA goldens. The lifter reads `path![p, ..=n, q]` as `whenever(p).eventually(q).within(n)`. The machine fixture's `reaches_two` is `path![self.value == 1, ..=2, self.value == 2]`, and its `values.txt` is unchanged. Only its expansion golden changed. The new `dsl/tests/fixtures/path` golden pins all three waypoint kinds, both gaps, a chain, and the chain-as-claim refusal.

README: the reading rules beside Kettle's claim. DESIGN: the Scala and `path!` claims side by side, plus the updated comparison row.

Deviations:
- Two extra refusals beyond the spec: two gaps in a row, and fewer than two waypoints.
- Two lift source tests that edited Kettle's builder text now edit the `path!` form (bound test) or switch back to the builder first (repeated-builder test).

make model-time:
```
timing-controls                   0.015s
capability-controls               0.110s
source-boundary                   0.209s
protobuf-freshness                0.107s
temporal-protobuf-freshness       0.070s
fmt                               0.286s
clippy                            0.127s
docs                              0.165s
tests                             2.641s
wasm32-wasip1                     0.060s
core-wasm-clippy                  0.085s
wasm-guest                        0.097s
go-host                           1.344s
temporal-go-host                  6.447s
inner loop (edit+test)            6.356s
gate total                       12.026s  (soft target 30s)
inner loop                        6.356s  (soft target 10s)
```

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 4c18870070
- Tests: make model-check, make model-time
- PRs: