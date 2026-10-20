# Test-space patterns: cover and the full path! grammar (model)

## Conversation Evidence

> user (2026-10-19): "what if we allowed authors in tests to define these paths quite liberally; eg different kinds of actions, different kinds of outcomes, maybe even faults ... as specific or vague about the test space they want to explore ... give authors a dense, flexible but precise tool to define the scope - and have confidence that the cases they think are covered, are covered ... defining ranges (or sets?) of permutations very easily. it must replace the extremely verbose functional tests into sth very short and dense but still readable and able to follow and explain without doubt or error." — then "yes do".

## Goal & Context

fn-215 gives tests a pattern over paths with state waypoints and gaps; fn-213 gives `find` and `Run::along`. Functional tests today are one hand-written interleaving each, hundreds of lines, and silent about what they do not cover. This spec grows the pattern into a test-space language and adds the question that makes it trustworthy: `cover` finds one shortest witness per alternative the author wrote and fails on any alternative with no admitted path, so a pattern is both the scope and the proof of its coverage. Waypoints gain actions with input ranges, outcomes, faults (actions of a fault actor), alternatives, repetition and permutations; the expansion of every test is a golden a reviewer reads as the list of cases that ran; a coverage table shows what the test set touched against the Model's catalog. The claim form (`#[progress]`) keeps exactly two state waypoints; this grammar serves `find`, `cover` and `Run::along`. Later, each witnessed path is a Case against the real system, so the same pattern defines live scope.

## Architecture & Data Models

- `Path` (umpire-dsl, extending fn-215) gains waypoint kinds: an action (`Worker::Poll`), an action with input patterns (`User::Send { amount: 0..=3, from: king, .. }`: ranges, sets, exact values, `_`/`..` for any), every action of an actor (`Deadline::*`), an action with an outcome (`Client::Start => Outcome::Refused`, outcomes may be alternatives), and, as structure, alternatives `(a | b | c)`, repetition `[x; n]` and `[x; a..=b]`, permutations `unordered![a, b, c]`, and gaps `..`, `..=n`, `..!(x, y)` (any steps without these actions). A fault is an action of a fault actor and needs no syntax. The notation is Rust's own (`|`, struct patterns, `[x; n]`, `..`) plus one-word macros; `rustfmt` and rust-analyzer read it.
- **Alternatives.** A pattern denotes a finite set of alternatives: one choice per `|`, per input range/set value, per repetition count, per permutation. The count is the product; it is computed before any search and printed; a count over the space's budget is a `Refusal` naming the count.
- **`cover(&space, Cases(n), path) -> Result<Covered, Refusal>`** (umpire-suite; owner, 2026-10-19): `Cases(n)` is required, the number of alternatives the author believes the pattern has, often written as its derivation (`Cases(2 * 4)`); the enumerator checks it before any search and a mismatch fails first, `pattern has 24 alternatives, test states Cases(8)`, followed by the alternative list. Then for each alternative, the shortest admitted path matching it (fn-213's `find`), in alternative order; an alternative with no path fails with its index, its concrete choices and the last reached state (`alternative [Poll; 0], Client::Cancel: no path reaches closed(); Cancel is not enabled before dispatch`). `Covered` holds the runs (each an ordinary `Run`), `len()`, `all(..)`, and `coverage()`: which actions, outcomes, faults and claims of the Model's catalog the runs touched, and which they did not.
- **Expansion goldens.** A `cover` test records its expansion in `expected/<test>.txt`: one line per alternative with its concrete choices, the action sequence with outcomes and the final state, spelled as the checker's paths are. The suite's runner compares it like other goldens, so a Model or upstream change that alters the expansion is a golden diff.
- `sample(&space, path, n, seed)` is the companion for random walks within a pattern (reproducible by seed), reported the same way; it proves a finite experiment only.
- `Run::along` and `find` accept the full grammar; a pattern with alternatives in `Run::along` is refused (`a story has one path; use cover`).

## API Contracts

- `path![..]` with the waypoint kinds and structure above `-> Path<S, A>`.
- `cover(&Space<M>, Cases, Path) -> Result<Covered<M>, Refusal>`; `Cases(usize)`; `Covered::{runs, len, all, coverage, expansion}`.
- `sample(&Space<M>, Path, n: usize, seed: u64) -> Result<Covered<M>, Refusal>`.
- Goldens: `expected/<test>.txt` expansion per `cover`/`sample` test; `coverage.txt` per test file.

## Edge Cases & Constraints

- `..` inside a pattern is bounded by the space's budget, as `find` is; a witness search that is cut answers `Refusal::Cut`, never a silent pass.
- Input ranges must be finite (`Finite` domains or explicit ranges); an open input is refused naming it.
- Permutations of more than a declared limit (default 6 elements, 720 orders) are refused with the count unless the limit is raised in the call.
- The claim form is unchanged: `#[machine]` still refuses more than two state waypoints, and refuses any action, outcome or structural waypoint in a claim (`a progress claim is two state predicates`).
- fn-210 R30 applies to `make model-check`; Kettle's and the mini activity's cover tests must fit the inner-loop budget.

## Acceptance Criteria

- **R1:** `path!` accepts action, input-pattern, actor-wildcard, outcome, alternative, repetition, permutation and exclusion-gap waypoints with the spellings above; each has a trybuild-pinned refusal for its malformed form; claims refuse all of them. No error surface beyond those.
- **R2:** `cover` takes a required `Cases(n)` and fails first on a mismatch with the enumerated count, listing the alternatives; then yields one shortest witness per alternative in alternative order, fails on an un-witnessed alternative naming it as specified, and refuses a count over budget. Errors: as stated, plus `Refusal::Cut` on a cut search.
- **R3:** Every `cover`/`sample` test has an expansion golden listing each case as choices, action sequence with outcomes and final state; `coverage.txt` lists touched and untouched actions, outcomes, faults and claims. No error surface.
- **R4:** Kettle's `tests.rs` has a `cover` test over `(User::DepressButton | Heater::Failover)` endings with `[Heater::Heat; 0..=80]` and the mini activity's has the four closings times polled-or-not (eight cases), both with goldens; the archived `Kettle.test.scala` enabledness table is one `cover` test. No error surface.
- **R5:** `sample` is reproducible by seed and reported like `cover`. No error surface.

## Boundaries

- No change to the claim form or to the checker's verdicts.
- No live Cases from witnessed paths yet (the Case slice consumes `Covered` later); no conformance.
- No negative patterns beyond gap exclusion, no "until", no monitors.
- Does not start before fn-215 and fn-213.3 (`find` through waypoints).

## Decision Context

2026-10-19, owner: tests may describe their space liberally, with actions, outcomes, faults, ranges, sets and permutations, as long as the tool is dense, precise and explainable without doubt. The design answers with Rust's own pattern notation, a finite alternative set computed up front, `cover` failing on any un-witnessed alternative (no silent vacuity), the expansion as a golden a reviewer reads as the list of cases, and a coverage table against the Model's catalog. Alternation is `(a | b)`, Rust's pattern alternation; a braces form was not adopted. The claim form stays two state waypoints.

2026-10-19, owner: the expected number of cases is a required argument of `cover` (`Cases(n)`), so the author states the size of the space before seeing it; a mismatch fails before any search and lists the alternatives. The expansion golden pins which cases, the argument pins how many.

2026-10-19, naming: the claim value type is `LeadsTo` (see fn-210.20); nothing else in this spec changes.

## Implementation decisions (2026-10-20)

The root agent settled these details under the owner's standing instruction to follow its recommendations.
They preserve R1–R5 and the existing state-waypoint and claim meanings.

- The normal Action derive must support payload fields, not only manual Machine examples. The current
  Rust workspace has no Finite trait. Each payload field declares a finite `#[inputs(0..=3)]` range
  or `#[inputs([Account::King, Account::Queen])]` set. Missing/open domains refuse naming the
  actor, action and field. The Model enumerates its complete declared input catalog independently
  of a requested path; explicit impossible path values remain alternatives and fail witness search.
  Payload equality, hashing, cloning and formatting distinguish complete values.
- R3 requires declared outcome/fault metadata; observed graph rows alone cannot enumerate untouched
  kinds. Generated worlds supply complete stock/unit outcome classes and a fault subset from
  explicitly marked fault actors (`#[fault]`, independent of `#[internal]`). Borrowed typed outcome
  classifiers avoid requiring Clone or Eq on every outcome. Manual worlds may declare catalogs;
  undeclared metadata is reported as unknown, never as complete empty coverage. Claim coverage is
  tied to witness evaluations/triggers and registration identity, not whole-graph verdicts.
- The literal adjacent Kettle example in task .3 contains eighty disabled failover alternatives.
  Insert an explicit gap after the repeated Heat prefix and before the ending alternatives to
  retain all 162 authored alternatives. Expansions show every additional heat in that gap; the
  repetition count describes the explicit prefix. Keep the literal adjacent form as a refusal
  control rather than changing Model enabledness. Likewise the activity's zero-poll Complete/Fail
  branches need an explicit gap before closing actions; retain eight authored alternatives and
  use the appropriate declared deadline action. Actual paths are `fixtures/sample_kettle` and
  `temporal/features/activity`, not the stale illustrative task paths.
- `sample` performs reproducible finite random walks within the admitted pattern. Randomly
  selecting cover's shortest witnesses alone does not implement that contract. State the walk
  bound and cut behavior explicitly, and report the seed and experiment size with its expansion.
- Task .1 owns the supporting DSL catalogs/actor projections as well as grammar and enumeration;
  .2 consumes them for matching, cover/sample and coverage; .3 supplies the real Model examples.
- Keep `Space::budget(n)` exclusively a distinct-Model-state budget (fn-226). Add separately
  configurable alternative-admission and random-walk limits; R2's count-over-budget refusal
  uses the alternative limit. Covering two alternatives on a one-state graph must work with
  Model-state budget one. Sampling reports its finite walk bound and refuses a selected
  alternative that cannot finish within that bound rather than silently resampling an easier one.
