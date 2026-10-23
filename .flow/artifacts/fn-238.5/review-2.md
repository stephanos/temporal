# fn-238.5 re-review: review fixes 8d6c013f47 and 869500aa44

Branch `work/fn2385-interpreted`, HEAD 869500aa44. I compared it with the first review (`review.md`) and checked each finding against the code and focused tests. Every probe was reverted, and the worktree is clean at 869500aa44.

## Verdict: SHIP

All ten findings are resolved or properly carried forward. The two majors are fixed, and controls guard them. I re-applied the old behaviour, and both controls fail against it. One small inaccuracy remains: DESIGN.md says "every admitted Model in the repository", and that is not quite true (R1 below). It does not block shipping. Fix the sentence at merge, or in .7.

## What I ran (worktree target dir, alone)
- **Focused tests.** All of these are green:
  - `cargo test -p umpire-fixtures -p umpire-temporal-tests -p umpire-checker -p umpire-testing -p umpire-ir -p umpire-front -p umpire-lift -p umpire-dsl-macros`
  - this covers `interpreted` (7 passed, 1 ignored), `fixtures` (27), the Temporal `fixtures` (13, including the activity differential), the checker `fixtures` (12), and the lift and macro goldens.
- **fmt and clippy.** `cargo fmt --all --check` returns 0. `cargo clippy -p umpire-checker -p umpire-testing -p umpire-fixtures -p umpire-temporal-tests --all-targets --all-features -D warnings` is clean.
- **Gate receipt.** `model-check-default-5.log` ends with `exit 0 after 1138s`. It has 166 ok test results and no FAILED or error lines. The fix commits touch no golden, snapshot or `.stderr` file.
- **Re-applying the old behaviour.** I patched `interpret.rs` in two ways and then reverted both:
  - evaluate every start domain up front;
  - drop the `usize` range arm.

  `start_domains_evaluate_lazily` then fails with "compiled: explored / interpreted: starts panicked: attempt to add with overflow". `lengths_compute_in_usize` fails with "report.txt line 8: Verified vs Refuted" and "only the interpreted exploration writes paths/wide.txt". Both controls bite, and the new start-panic reporting shows up exactly as intended.
- **Admission census.** A scratch test walked every `#[machine]` module under `model/umpire` and `model/temporal`, excluding trybuild `ui/`, and passed each through `umpire_front::declare::machine` + `admit`, exactly as the macro does. It found 41 admitted modules.

## Findings from review 1

1. **Eager starts (major): resolved.**
   - `product` now evaluates the domain at each depth only when that depth is reached, and keeps it after the first evaluation (interpret.rs `product` / `Domain::values`). That matches the compiled nested `for` loops (`state.rs:164-165`): an empty earlier domain skips a later one, and the first evaluation decides any panic.
   - `start_empty_first` agrees with 0 states. `start_fault_first` and `start_later_fault` agree on the same start panic.
2. **Coverage (major): resolved in code.** A small accuracy gap remains in DESIGN.md (R1).
   - The activity's `run()` calls `umpire_testing::interpreted(world, None, 64, None)?`. It is a single line, and only the activity's `tests.rs` changed under `temporal/`.
   - `backend_controls_agree` covers:
     - all nine Apalache and TLC backend Models;
     - the arithmetic fixture over 11 × 5 × 4 = 220 Worlds, the same sweep I ran;
     - `empty_condition` over 6 Worlds;
     - `replay` over 2 Worlds.
3. **Start panic escapes (minor): resolved.**
   - `testing::interpreted::enumerated` catches a panic during `starts()`, turns it into a "starts panicked: <msg>" refusal, and compares that against the other side.
   - No regressions: explicit `starts: Some(..)` skips the catch, the timing path is unaffected, and `difference` now lists every differing file in name order.
4. **`len()` width (minor): resolved in the interpreter.** The lifter/TLC mismatch is correctly reported as a follow-up (R2).
   - `range()` maps the front's index type `Int{0, i32::MAX}` to `0..=usize::MAX`.
   - I checked that nothing else can produce that range. `declare::integer` (fields.rs:155-170) maps no Rust type to it, and only `vector.rs:10` builds it. The sentinel is sound today, though brittle (R3).
   - `lengths.rs` is the probe from review 1, and both sides now say Verified.
5. **Claim cap (minor): resolved for .5.** `claims_beyond_the_cap_are_refused` covers both edges: 65 claims are refused with the exact message, and 64 are accepted. The handoff gives .6 a sound way to remove the cap: generate an exact-size callback table in the `Interprets` impl.
6. **Mutants (minor): resolved.** There are now six mutants: update, postcondition, start, progress target, outcome and `check!` index. Each asserts the file that shows the change.
   - The outcome mutant asserts the report *starts with* `rows.txt`. Because files are listed in name order, that proves `paths/*` and `report.txt` are unchanged. It does not by itself prove `states.txt` is unchanged, but a rejection-variant swap cannot change states, so this is fine.
7. **Dead API (minor): resolved.** `compiled()`, `check_texts()` and the `checks` field are gone, and `CLAIM_LIMIT` is private.
8. **Handoff/`run()` (nit): resolved.** Decision 5 is corrected. The inserted line uses the function's existing (odd) 8-space indentation.
9. **`Check` index scope (nit): resolved.** The comment in expr.rs now says the position is meaningful only in the tree `#[machine]` embeds.
10. **Acceptance marking (nit): resolved.** Item 2 is marked [~], and item 3 stays open until `make model-time` is recorded.

## The implementer's decision on `progress_chain_path` and `rejecting_mutation`
**Agreed. Both are refused by the front end, so they are correctly out of scope.** My first review was wrong to list them as admitted. The census reproduces the implementer's refusals exactly:
- `progress_chain_path/model.rs` 26:36 "expected a single expression body";
- `rejecting_mutation/model.rs` 17:5 "unsupported declaration in machine module".

`#[machine]` generates `Interprets` only for admitted modules, so a refused one cannot be interpreted. That is the documented soft-refusal policy. Both Models stay checked compiled-only, and their goldens are unchanged.

## Remaining findings

### R1. minor: DESIGN.md says "every admitted Model in the repository", but four admitted Models are not in the gate
`model/DESIGN.md:84`

The census finds these admitted Models outside the differential:

| Model | Location | Probe result |
| --- | --- | --- |
| `catalog_admission::worlds` | `umpire/checker/tests/catalog_admission.rs:139` | agrees on 8 Worlds |
| `counter` | `umpire/dsl/tests/readme/parameterized.rs` | agrees through its `fixture!` hook and on 3 more Worlds |
| `kettle` | `umpire/suite/tests/support/kettle.rs` | agrees, 163 states and 244 edges |
| `switch` | `umpire/testing/tests/fixture_composition.rs` | not probed (see below) |

`switch`'s World is private, so `world()` returns an opaque `impl Machine` and nothing outside the module can interpret it.

The first three agree today, so nothing is hidden. The sentence is still wrong as written. README's wording is accurate, because it lists what it covers.

**Fix:** either change DESIGN to "every admitted Model in a fixture, backend or lifter test" and name the four, or add one line each for the three reachable Models (for example, a call in their own tests, as the activity has). Either fits in the merge commit or .7.

### R2. minor (follow-up, outside .5): compiled and TLC disagree on `lengths.rs`
The front types `len()` arithmetic as 32-bit, and TLC's integers are 32-bit. On `lengths.rs`, the lifter asserts the range and TLC overflows, while compiled Rust (and now the interpreter) verifies the claim. The TLC runner also fails to parse that output, so it surfaces as a runner refusal rather than a located counterexample. Log: `lengths-tlc-probe.log`.

This is a front-end typing gap that exists independently of .5, and the handoff reports it accurately. No golden lifts `lengths.rs`, so nothing breaks today.

**Fix:** a follow-up task, either refusing arithmetic on `len()` beyond comparison and indexing, or adding a located refusal in emission. The TLC runner's parse failure should be a second item in that task.

### R3. nit: the `usize` sentinel
`model/umpire/checker/src/interpret.rs` `range()`

Recognising `usize` by the exact range `0..=i32::MAX` is sound today: no admitted Rust type has that range, and only `vector.rs` builds it. But if the front ever admits a type with that range, or widens `index_type`, the interpreter silently changes width. The R2 follow-up should give the index its own marker in the IR, such as `Type::Index` or a flag, so the interpreter reads the width from the IR instead of guessing it from the range.

## Still open
- Acceptance item 3 (`make model-time` within 60 s / 10 s) is unmeasured, as the handoff says.
- The 2.6× ratio stands against .6's 2× gate. I reproduced it in review 1.
