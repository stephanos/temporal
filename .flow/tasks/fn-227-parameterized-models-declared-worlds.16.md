---
satisfies: [R3]
---
# fn-227-parameterized-models-declared-worlds.16 Named intermediates in the IR so admitted trees stay linear in conditional returns

## Description
From fn-227.15's reviews (`.flow/tmp/fn22715-review.md` L1, `fn22715-review2.md` N2). Emitted TLA+ is linear in conditional returns since fn-227.15, but the front end's admitted tree and the macro-embedded `__lifted` data still grow about 7x per return when a later condition or update reads a field changed after a conditional return (and a run of updates after one early return doubles the tree per update). fn-227.15 added a 10,000-node budget so such steps refuse at the statement instead of exhausting rustc memory (measured: about 24k-64k nodes OOM), which caps the pattern at about four sequential returns.

Introduce named intermediates in the IR (`ExprKind::Local` plus per-branch definitions, emitted as TLA+ `LET`) so the tree, the embedded data and compile time stay linear. Then raise or remove the budget. Run before fn-227.3, whose retries add conditional rejections to the activity.

### Verification
Size tests over the admitted tree and embedded data (not only emitted TLA+) for sequential returns and repeated updates; TLC verdicts, state counts and replay unchanged on every fixture; listed TLA+ text goldens may change, justified. Quick then Full gates; record compile time for a pathological probe before/after.


**Size:** L
**Touches:** `model/umpire/ir/**`, `model/umpire/front/**`, `model/umpire/lift/**`, `model/umpire/backends/tlc/**`, `model/umpire/tests/**`, `model/umpire/dsl/macros/**`, `model/README.md`, `model/DESIGN.md`, `model/temporal/feature/activity/expected/**` (only justified emitted TLA+ text), `model/temporal/hosts/go/umpire_temporal_guest.wasm` (generated refresh).

### Execution and gates

Consume fn-238.9's derived constructor printer before adding the new IR fields and variant. Keep generic IR/front-end/lifter ownership; use the existing emitted LET/name-collision machinery where it fits. Preserve first-return semantics, conditional execution, checked arithmetic, payloads, selected World substitution, phase/rejection and terminal-law behavior. An authored name must not capture a generated intermediate. Source/TLC replay must resolve the same typed values. This changes the admitted data representation, not the authored Model semantics.

Before changing expectations, retain baseline semantic/golden hashes and per-fixture TLC counts/verdicts, and measure the existing pathological source probe. Add discriminating linear-growth tests for both sequential returns and repeated updates over admitted tree size and embedded constructor data, plus actual compilation past the old refusal/OOM range. Emitted-text size alone is insufficient. Raise or remove the obsolete node cap only with measured safe compilation. Preserve canonical Model source, native reports/states/rows, verdicts and replay outputs. List each necessary emitted TLA+ text golden change with its mechanism; do not globally refresh unrelated goldens. A needed path outside Touches must be reported to the conductor before editing.

Quick from model/: `mise exec -- cargo test -p umpire-front -p umpire-ir`; `mise exec -- cargo test -p umpire-lift --test phased`; relevant source/replay controls under required TLC. Include the new growth/collision/compilation controls in focused runs. Full from root: one `UMPIRE_BACKENDS=require make model-time`, whose nested `make model-check` must explicitly show required TLC execution and all stages passing. This satisfies the complete full gate and timing together; preserve the actual command. Use an outer bound of at least 1800s for this full/timing run, and at least 1300s for the gap target whose Go test has a 20m bound. Do not repeat a completed unchanged full gate merely for a second command spelling. If changed closure bytes require the embedded guest refresh, generate through `make umpire-guest` in a separate commit once source settles, then actual `make umpire-check-model-slice` and `make umpire-check-model-gaps`; don't fit expected semantics to the server. Preserve literal timing output, probe before/after, TLC counts/verdicts/replays and golden comparisons in task-owned artifacts. Record soft-target overruns honestly.
## Acceptance
- [ ] Admitted tree and embedded data grow linearly with sequential conditional returns and repeated updates (size tests).
- [ ] Node budget raised or removed with a probe showing no rustc blow-up.
- [ ] TLC verdicts, state counts and replay unchanged; changed goldens justified.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
