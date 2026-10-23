# Native symbolic back end: BTOR2 + rIC3 instead of Apalache

Status: spike done 2026-10-23, verdict positive. Probe kept at `.flow/tmp/native-symbolic/`
(README, `run.sh`, emitted files, logs, reports). Built against f76f622a79.

## Answer in brief

Umpire does not need Apalache for symbolic checking of what the front end admits today. A
direct emitter from the admitted tree to BTOR2 (word-level bitvector transition systems), checked
by rIC3, gives unbounded proofs for the safe Kettle claims in 0.1 s each, finds the 62-step
`never_beeps` counterexample in 0.1 s, replays it through `checked_step`, and agrees with the
native checker on every property. The emitter plus witness replay is 1.1k lines, written in one
session, and skips everything fn-211 needed only because the SMT path detoured through TLA+:
Snowcat annotations, the no-ENABLED safety projection, ITF decoding, JVM and release pinning.
Symbolic World constants fall out of the encoding (a World field is a free state with
`next = self` and its domain as a constraint); they are implemented but not yet exercised.

Apalache keeps one thing a word-level encoding cannot replace: its symbolic encoding of TLA+
sets, functions and unbounded sequences. No current Model uses set-valued state. Apalache is
therefore deferred, not rejected: it comes back when the admitted subset grows past bounded
word-level state, and by then the question is whether to emit SMT-LIB with arrays/datatypes
directly rather than TLA+.

## What was built

| piece | lines | notes |
|---|---:|---|
| `btor2.rs` emitter | 643 | fields, World constants, first-match branches, enabledness = taken branch changes state (or preserves a self-loop), free `act` input (value n = no step), one-step history (`prev_*`, `prev_act`, `has_step`) for postconditions incl. `after =` filters, checked arithmetic as a latched `fault` (overflow by widening, since rIC3's logic library has no `saddo`), one `bad` per property |
| `witness.rs` | 225 | BTOR2 witness decoding (assignments are keyed by state/input declaration index) and replay from a declared start through `checked_step`, matching each frame to a native successor by typed field values (`Values`), then re-judging the violated claim natively |
| `smt.rs` | 146 | the same node graph unrolled as QF_BV SMT-LIB 2 with push/pop per depth, piped to z3 (the shape the `easy-smt` crate wraps) |
| `main.rs` + Kettle snapshots | 251 | driver, controls, scaled Kettle |

Input is `World::__umpire_lifted().model`, the embedded `umpire_ir::Lifted`; no source parsing,
no `umpire-lift`, no `umpire-front` at run time. The fixture's negative control `never_beeps`
is added as one IR expression.

## Measurements

Baseline Kettle (283 states, native depth 62; native exploration and verify 0.1 ms):

| property | kind | ic3 | kind | bmc (bound 63) | smt-bmc z3 (bound 63) |
|---|---|---|---|---|---|
| beeps_only_when_off | invariant | safe, unbounded, 0.10 s | safe, 0.11 s | no violation, 0.10 s | no violation, 0.23 s |
| never_beeps | negative | unsafe, 62 steps, replayed and confirmed, 0.10 s | unsafe, 0.31 s | unsafe, 0.21 s | violation at depth 62, 1.74 s |
| warms_only_when_on | postcondition | safe, unbounded, 0.10 s | safe, 0.11 s | no violation, 0.10 s | no violation, 0.34 s |
| fails_over_at_boiling | postcondition (after Failover) | safe, unbounded, 0.10 s | timeout 120 s | no violation, 0.20 s | no violation, 2.02 s |
| arithmetic | fault latch | safe, unbounded, 0.11 s | safe, 0.10 s | no violation, 0.10 s | no violation, 2.17 s |

All agree with the native checker. The replayed `never_beeps` path is the fixture's: start at
40 degrees, PressButton, 60 Heats, Failover.

Controls (mutating the admitted tree only, so the native Model disagrees and the cross-check
must say so):

- `claim99`: the postcondition literal 100 becomes 99. IC3 reports unsafe at 62 steps; replay
  walks the path natively and the native claim holds on the final edge, so the result is
  "NOT confirmed, DISAGREES". Flagged as intended.
- `heat2e9`: Heat's guard bound becomes `i32::MAX` and it adds 2e9. IC3 reports the arithmetic
  fault unsafe (overflow on the second heat); replay rejects the witness at frame 2 because the
  real Model has no such edge. Flagged as intended. (A first version of this control added 2e9
  without lifting the guard; the encoding correctly reported it safe, because the guard
  `temperature < 100` fails before the addition is evaluated, which is Rust's order too.)

Scaled Kettle, boiling at 1 000 000 (a real second `#[machine]` Model, not a mutant):

| | native checker | ic3 |
|---|---|---|
| explore and verify all | 2 000 083 states, depth 999 962, 19.0 s (needs an unlimited stack) | |
| four safe claims | verified | safe, unbounded proof, 0.10 s each |
| never_beeps | refuted (path depth ~10^6) | timeout at 120 s |

IC3 proof time did not move with the state space; the explicit checker's did. The deep
counterexample is where IC3 is weak and BMC or explicit search is right; a production back end
runs IC3 for proofs and BMC to the native depth for violations, and both are the same file.

## What was not proved

- Only Kettle-shaped Models: bools, ranged ints, fieldless enums, unit outcomes, no action
  inputs, no computed starts (`#[starts]` bodies), no sequences. Each is an emitter case, not a
  design question; `parameterized_world` needs computed starts before symbolic Worlds can be
  exercised.
- Width choice derives from the declared range (`Type::Int { min, max }`), so a `Finite`
  narrowing hides the Rust width; the fault latch and range constraints keep it sound for the
  spike but a production emitter should carry the Rust type width.
- The SMT-LIB path has no witness replay (no `get-model` decoding); it is a timing data point.
- rIC3 witnesses list only states the engine assigned; replay tolerates missing fields at
  later frames by taking choice 0, which Kettle never needs. Frame 0 is always complete.
- rIC3's certificate checking (`cerbtora`) and portfolio mode were not run.
- No fixture-runner integration, no goldens, no `make` target.

## Recommendation

Replace the Apalache direction with a native symbolic back end spec:

1. `umpire/backends/btor2` (or `symbolic`): emitter over the shared admitted tree, one file
   per Model with one `bad` per property, rIC3 pinned as a built binary (it is a Rust crate with
   C++ SAT solvers; building needs cmake and clang or a shim), IC3 for proofs and BMC to the
   native depth plus one for violations, witness replay through `checked_step` exactly as
   fn-211 R3 specified, goldens `expected/symbolic.txt`.
2. Symbolic World constants (fn-211 R6) are part of the first task, since the encoding already
   has them; computed starts need the source-lifted `#[starts]` bodies or a native start table.
3. Keep TLC as the exhaustive reference. Keep Apalache deferred until set-valued state is
   admitted, and decide then between Apalache and direct SMT-LIB with arrays.
4. Carry Rust widths on `Type::Int` so the emitter does not derive them from ranges.

## Rerun

```
cd temporal/.flow/tmp/native-symbolic
RIC3=/path/to/ric3 SMT_SOLVER=/path/to/z3 ./run.sh           # baseline + controls
RIC3=/path/to/ric3 RUNS=big ./run.sh                         # scaled Kettle, ~3 min
```
