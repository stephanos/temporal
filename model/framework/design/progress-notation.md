# Progress notation: `whenever … eventually`, an optional bound, and `without`

Status: decided with the owner 2026-10-17 and implemented in the working tree on `e2eebb0a02`
(uncommitted). Paths are relative to `model/framework` unless they start with `model/` or `.flow/`.

## Summary

A progress claim used to read `leadsTo(from = p, to = q, within = n) in env`. The words were
TLA⁺'s (`~>`), `from`/`to` read as a range rather than as "whenever p, eventually q", every claim
had to state a step bound, and Kettle's environment listed the actions it kept when the point was
the one it dropped. Now:

- **The claim reads as a sentence**, in the block shape postconditions already use:
  `whenever(p): eventually(q)`.
- **The bound is optional.** `eventually(q)` is unbounded leads-to, checked under the scenario's
  weak fairness; `eventually(q, within = n)` is the bounded claim it always was. A claim states a
  bound only where the bound is the promise.
- **The environment stays a Scenario after `in`**, as fn-190 R13 decided. A block claim is
  parenthesised before `in`, as `(after(action): ...) in env` already is (SEMANTICS §9).
- **Scenarios gain `without(actions)`**, beside `only(actions)`, for the environment that drops
  actions instead of listing the ones it keeps.

`leadsTo(from, to, within)` stays as the same bounded claim. The export and the IR are untouched:
progress is a native model check only.

## Decisions (owner, 2026-10-17)

| # | Question | Decision |
|---|----------|----------|
| 1 | The claim's notation | A1: `whenever(p): eventually(q)`, block claims parenthesised before `in` |
| 2 | Must every claim state `within`? | No. The bound is optional; an unbounded claim is checked under weak fairness. This reverses the "no unbounded liveness" line of fn-112, fn-120, fn-149 and SEMANTICS §11 |
| 3 | Kettle's environment | `without(user.depressButton)`, a new Scenario builder |
| 4 | Claim layout | One statement per claim, starting with what scopes it; a long `if` breaks after `then` (below) |

## The claims

| Sample | Before | After | Why |
|---|---|---|---|
| Kettle | `leadsTo(from = heatingOn, to = beeping, within = 141) in heatingAlone` | `(whenever(heatingOn): eventually(beeping)) in neverSwitchedOff`, `neverSwitchedOff = without(user.depressButton)` | 141 was a count of the model's steps (140 heats and the failover); the checker still reports it as the longest path |
| ReliableBroadcast | `leadsTo(from = someoneDelivered(current), to = everyoneDelivered(current), within = 9)` | `whenever(someoneDelivered(current)): eventually(everyoneDelivered(current))` | 9 was the steps that may happen meanwhile, counted |
| TwoPhaseCommit | `leadsTo(from = tmState == committed, to = allCommitted(current), within = 3) in fairCommits` | `(whenever(tmState == committed): eventually(allCommitted(current), within = 3)) in fairCommits` | Its comment states the bound as the promise, "within three steps" |
| TwoLayeredCache | `leadsTo(from = !agreed(current), to = agreed(current), within = 1)` | `whenever(!agreed(current)): eventually(agreed(current), within = 1)` | "One step after they part" is the cache's promise |

A claim with an environment, as scalafmt lays it out:

```scala
val boils =
  (whenever(heatingOn):
    eventually(beeping)
  ) in neverSwitchedOff
```

The `model/temporal/features` claims are all in `.disabled` files in the retired named form
(`DispatchingProgress.leadsTo("workerStarts")(…)`) and are unchanged.

## Why `within` became optional

`within` was mandatory by decision, not by necessity of the checker. fn-112 (2026-10-03) deferred
an unbounded `eventually` because it "would need a path-formula node, a bounded-trace and
fair-lasso checker beyond `leadsTo`, and widens the P export gap", and reserved the word for TLA's
unbounded meaning; fn-120, fn-149 and fn-150 repeated "no unbounded liveness".

The native checker has since grown what that needed. `NativeCheck.leading` (`dsl/NativeCheck.scala:547`)
finds deadlocks and fair non-progress cycles (strongly connected parts of the region that no weakly
fair class leaves) before it looks at `within`; the bound was only the last comparison, against the
longest path. Two consequences followed:

- **A bound made fairness inert.** Any cycle that avoids `to` failed a bounded claim, fair or not;
  fairness only chose its label: a cycle where no fair class leaves it, a missed deadline where one
  does. TwoPhaseCommit's
  `commitReachesEveryone` verifies with longest path 3 with and without `fairCommits` (checked with
  a scratch probe). Under an unbounded claim a cycle that a fair class always leaves is no
  counterexample, which is what `fair` says.
- **Derived bounds were noise.** Kettle's 141 and ReliableBroadcast's 9 counted the model's steps.
  The checker reports the longest path of a verified claim anyway, and the tests pin it
  (`samples/Kettle.test.scala` asserts Kettle's 141).

The unbounded check: verified on a complete exploration with no deadlock and no fair non-progress
cycle, the longest path reported where no path cycles; a cut exploration answers unknown, as every
check does. Progress generates no Case and the export lifts only the named form, so no IR or P
export change follows. Receipts keep fn-24's rule: verified means the finite model, fully explored.

## When a claim gets a word

A claim starts with a word of the DSL only when the word changes *what is checked*: which steps a
claim reads, or at which time it reads them. A claim that only skips some states or steps and
checks the same state or step otherwise is a plain Scala `if`, as R11 decided ("a block of `check`s
reads like a unit test", `.flow/specs/fn-190-models-are-plain-scala-state-as-vars.md:532`).

| Leading form | What it changes | Plain Scala could say it? |
|---|---|---|
| `if cond then` | Nothing: it skips some states or steps and checks the same one | Yes; it is plain Scala |
| `after(action):` | Which steps: one action's. The checker judges only those, and reports an action that never fires | No: the checker needs the trigger as structure |
| `whenever(p):` with `eventually(q)` | When `q` is read: in a later state | No: `if p then check(q)` reads `q` in the same state |

So Bank's `failedSendChangesNothing` keeps its `if`: "if the send failed, the balances did not
change" reads one step once, and a `when(cond):` word would be a second spelling of `if`. A word
would earn its place for a claim triggered by an outcome, were the checker to report "no step was
ever rejected" as it reports an `after(action)` that never fires; Bank's claim is vacuous today if
no send fails, and nothing says so.

Every claim then has one shape: its name, and one statement that starts with what scopes it, the
checks below. A long `if` breaks after `then`, so the checks sit where they sit under `after` and
`whenever`:

```scala
val failedSendChangesNothing =
  if outcome != Outcome.accepted then
    check(balances == before.balances)

val failsOverAtBoiling =
  after(heater.failover):
    check(temperature == boilingTemperature)
    check(beeping)

val agreement =
  whenever(!agreed(current)):
    eventually(agreed(current), within = 1)
```

The samples follow this layout (Kettle, Bank, Voting). scalafmt would join each `if` back onto one
line, so `model/.scalafmt.conf` keeps the author's line breaks (`newlines.source = keep`) in
`framework/samples` and `temporal/features`; tooling code keeps the default.

## Why the parentheses

`in` has to scope the whole claim, and without parentheses it cannot:

- **Inside the block it binds to the wrong thing.** `whenever(heatingOn):` followed by
  `eventually(…) in neverSwitchedOff` would scope the `Deadline`, the last statement, not the
  claim. The block-form lint refuses the same mistake under `after` (SEMANTICS §9).
- **After the block it does not parse.** An `in` on the line after the block would be a leading
  infix operator, and Scala 3 continues a line with a leading operator only when it is symbolic
  (`+`, `||`), not a word like `in`.

So the block is closed with `)` and `in` follows it, as `(after(action): ...) in off` does. The
parentheses are a wart both sections share. The way to drop them in both is to name the
environment before the block, `after(heater.failover, in = off):` and
`whenever(heatingOn, in = neverSwitchedOff):`; the environment would stay a Scenario that results
name, as R13 requires. That is open question 1.

## The implementation

**Words** (`dsl/Assume.scala:168-205`):

- `whenever(from)(deadline: Whenever ?=> Deadline): LeadsTo`, a macro (`QueryMacros.whenever`,
  `dsl/Syntax.scala:1880`) that keeps `leadsTo`'s checks, placement in `progress` and the name of
  the enclosing `val` (`progressClaim`, `dsl/Syntax.scala:1904`), and builds the same `LeadsTo`
  with the deadline's `to` and bound.
- `eventually(to)` builds an unbounded `Deadline`; `eventually(to, within)` is a macro that checks
  the bound as `leadsTo` did (`bound`, `dsl/Syntax.scala:1915`): a positive constant, a literal or
  arithmetic over `inline val`s.
- `eventually` needs a `Whenever` in scope, which only a `whenever` block supplies, so a stray
  `eventually` is the compile error "eventually is the deadline of a whenever block" rather than a
  value the progress section would silently ignore.
- `LeadsTo.within` (`dsl/Assume.scala:129`) is an `Option[Int]`; `leadsTo` gives `Some(within)`.

**Checker** (`dsl/NativeCheck.scala:625`): with no bound, the cycles a fair class leaves are no
counterexample, and the claim is verified on a complete exploration; with a bound, any cycle fails
the claim, as before: a cycle where no fair class leaves it, a missed deadline where one does. The named `Progress` form passes its bound unchanged.

**`without`** (`dsl/Claims.scala`): `ScenarioDecl` gains `excluded` (line 131) and `Scenario` the
builder `without` (line 202), with a top-level `without` beside `only` (`dsl/Syntax.scala:1813`).

- Admission (line 149): an action is admitted when `only` names it, if it says `only`, and
  `without` does not name the whole action; dropping one input class leaves the action's other
  classes admitted.
- A scenario says `only`, `without` or a pinned `actions` list, one of the three;
  `without` twice accumulates, as `fair` does.
- `modelOnly` (line 145) counts `excluded`: a realization cannot drive "the user never switches it
  off" any more than it can drive `only`.

**Tests** (`dsl/Whenever.test.scala`): a worker that may finish while a clock ticks forever. The
unbounded claim under `fair(worker.finish)` is verified with no longest path; the same claim with
`within = 1` misses its deadline on the tick cycle; with no fairness it reports the cycle. Also:
`whenever` builds the expected `LeadsTo`; `without` admission, `modelOnly` and the refusal beside
`only`; `without` of one input class runs only the other class; and the compile errors for a stray
`eventually`, a `whenever` outside `progress` and `within = 0`.

## Variants considered

| Variant | Kettle reads | Verdict |
|---|---|---|
| **A1. Block, trailing `in`** (chosen) | `(whenever(heatingOn): eventually(beeping)) in neverSwitchedOff` | Mirrors postconditions; keeps R13's environment. The parentheses are the cost. |
| A2. Fluent, trailing `in` | `whenever(heatingOn).eventually(beeping) in neverSwitchedOff` | No parentheses and no block macro, but a long claim breaks across lines, and it shares no shape with the other sections. |
| A3. Environment as a parameter | `whenever(heatingOn, in = neverSwitchedOff): eventually(beeping)` | Reads best and needs no parentheses. Uniform only if `after` gains the same parameter, `after(action, in = env):`; a one-line claim keeps its trailing `in`. |
| A4. Exclusion and fairness on the claim | `whenever(heatingOn, unless = user.depressButton): …`, `assuming = fair(rm.commit)` | Rejected: R13 took fairness off the claim, and the verdict would name no scenario. |
| C. Operator | `(heatingOn ~> beeping) within n` | TLA⁺'s symbol. Cryptic to others, and `~>` binds tighter than `==`, so `tmState == committed ~> …` needs parentheses. |

## Why Quint has no counterpart in the samples

Quint has temporal properties (`temporal p = eventually(q)`, `always`, `weakFair(action, vars)`), but
only one of the four originals uses them (`samples-vs-quint.md` has the per-sample detail):

- **Kettle** (`samples/Kettle.qnt.test`) is the REPL tutorial and has no properties at all. Its
  invariant, postconditions and `boils` are the port's.
- **TwoLayeredCache** has `temporal agreement = eventually(agreed)`, checked with
  `quint verify --temporal=agreement`. The port's claim is stronger: every disagreement is repaired
  within one step, where Quint's holds once from the start.
- **ReliableBroadcast** states eventual delivery in a comment only
  (`samples/ReliableBroadcast.qnt.test:143`).
- **TwoPhaseCommit** has an invariant and runs, nothing temporal.

Quint's everyday checker, `quint run`, simulates and checks invariants; temporal properties need
`quint verify` (Apalache) or TLC, with fairness written into the spec. Its `eventually` is
unbounded, and so is the port's now unless a claim states `within`.

## Open questions

1. **A3 for both sections.** `whenever(p, in = env):` and `after(action, in = env):` would drop the
   parentheses everywhere a block claim has an environment.
2. **Retiring `leadsTo`.** It is now a second spelling of `whenever(p): eventually(q, within = n)`.
   The fixtures in `dsl/*.test.scala` and `check/lint/BlockFormRule.test.scala` still use it.
3. **`without` beside `only`, or instead of it.** Kettle drops one action; a dispatch design keeps a
   few. Both read better somewhere.
4. **fn-190 R13's text** still says "a progress claim is `leadsTo(...)`" and Kettle "carries
   `heatingAlone`"; the spec records history, so it is left as written.
5. **TwoPhaseCommit's `fairCommits`** changes no verdict: its claim is bounded and its graph has no
   cycle. It documents an assumption the claim does not need.
