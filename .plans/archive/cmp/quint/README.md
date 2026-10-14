# Quint sample

[Quint](https://quint.sh) is Informal Systems' specification language with TLA+ semantics, a
random simulator (`quint run`, `quint test`) and a bridge to the Apalache and TLC model checkers
(`quint verify`). It is the one language in this comparison that is already a model layer: the
question is not how to build a DSL but how far Umpire's vocabulary fits a language that was
designed for state machines and nothing else.

Files: `umpire.qnt` (framework surface and the `Worker` module), `nexus_caller.qnt`,
`standalone_activity.qnt`, `pins.qnt`. Nothing here was run: the numbers pinned in `pins.qnt` (states,
rows, reachable states, stutters) are the author's arithmetic over the tables, not tool output.

## How Quint maps onto the Umpire vocabulary

| Umpire | Quint | Where |
|---|---|---|
| entity, party, schema, examples, observation | `pure val` catalog data (records, strings) | `nexusCallerVocabulary` |
| enum | sum type `type Reply = SyncSuccess \| ... \| HandlerError(bool)` | vocabulary modules |
| Step S O F | `type Step[s, f] = { outcome: Delivery, state: s, facts: List[f] }` | `umpire` |
| step function | `pure def xStep(s, inputs): List[Step]`, `[]` when not enabled | `*Table` modules |
| machine `steps:` | `pure def steps(s, a: Action): List[Step]`, one `Action` sum type per machine | `*Table` |
| machine `starts`, `ends`, `timers`, `unobservable` | `pure val` sets over the table | `*Table` |
| finite state enumeration | `pure val states = tuples(phases, attemptCounts, timeouts, ...)` | `*Table` |
| machine (stateful) | module with `var state`, `var last`, `action init`, `action fire(a)`, one `action` per action | `nexusProduct`, `nexusProtocol`, ... |
| action class input | `action handlerReply(reply: Reply)` parameters | machine modules |
| recorded evidence of the last step | `var last: Option[Fired[Action, State, Fact]]` (action, before, outcome, facts) | machine modules |
| same-step property | `val p = claim(fired(X), holds)`; `asInvariant(p)` for `--invariant`, `found(p)` for a find | machine modules |
| transition property | `pure def p(before, after): bool` on the product, read through `last.before` and `productOf` | `*Table` + machine |
| refines + map | `pure def productOf`; by mapped states: `rejectedRows(...) == Set()` over the table and `val refinesProduct` over `last` | `umpire`, `*Table`, machine |
| scenario | `run s = init.then(a1).then(a2)` | machine modules |
| limits | `pure val` record; `--max-steps`, `--max-samples` on the CLI | machine modules |
| query `find` | `run q = s.expect(found(p))`, run by `quint test` | machine modules |
| query `verify` | `.expect(inv)` after every step of the run; whole machine via `quint verify --invariant inv --max-steps n` | machine modules |
| set, realization | `pure val` data only; consumed by the Go runtime, which reads ITF traces | machine modules, Go |
| compose + sync | module importing both members (`import nexusProtocol as operation`, `import Worker(taskQueue = "handler") as worker`), `action sync = all { m1, m2 }` | `nexusCaller`, `standaloneActivity` |
| restrict | the composition's `step` names only the allowed member actions | composition modules |

Each machine is two modules. The `Table` module holds types and pure step functions and no
state, so the pins, the refinement and the composition can import it without dragging a second
copy of `var state` along. The machine module imports the table and turns it into actions with
a six-line `fire`. Quint has no way to abstract over which `var` an operator assigns, so `fire`
is repeated per machine.

Constructor labels are module-global in Quint: `type Resolution = Succeeded | ...` cannot live
beside `type Phase = ... | Succeeded`. The samples keep phases bare and prefix the two input
enums that collide (`ResolvedSucceeded`, `AttemptCompleted`). The same rule forces the timer
facts to read `StatusTimedOut(ByScheduleToStart)` beside the `ScheduleToStart` timer action, and
the product's one timer to be the `TimeoutFired` class beside the `Timeout` type; everything else
keeps the spec's name, capitalized as Quint constructors are.

## Which checks run where

| Check | Simulator (`quint test` / `quint run`) | Apalache (`quint verify`) | Not in Quint |
|---|---|---|---|
| well-typed model, exhaustive `match`, undeclared names | `quint typecheck`, in the editor via LSP | | |
| state count, ends count, class count | `pins.qnt`, pure evaluation, exact | | |
| reachable states, stuck states, row count | `pins.qnt`, pure fixpoint over the table, exact | | |
| refinement over the whole table | `pins.qnt`, `rejectedRows == Set()`, exact | `--invariant refinesProduct`, exhaustive to `--max-steps` | |
| scenario reaches its property (find) | `run q = scenario.expect(found(p))`, deterministic | | |
| verify on a path | `.expect(inv)` after each step of the run | | |
| verify over the machine | `quint run --invariant`, random, bounded by samples | `quint verify --invariant`, exhaustive to depth | |
| exploratory coverage (rows, results, class members) | `quint run --mbt --out-itf --n-traces` produces the traces | | counted on the Go side |
| canary admission (silent step on the path) | | | Go side; Quint has no evidence notion |
| schemas, parties, examples resolve | | | nothing checks catalog strings |

The `pins.qnt` checks are exact because they are pure expressions over finite sets; the
simulator evaluates them once. They are not compile-time: a wrong number is a failed `run`, not a
typecheck error. The cost is interpretation speed: the Activity protocol table has 288 states and
22 classes, so the refinement pin evaluates about 6,000 protocol step calls plus a scan of the
product's rows for each non-stutter, which is fast; a machine ten
times larger would move these pins to Apalache or Go.

## What an error looks like

A step function naming an undeclared action, or a `steps` dispatch missing an `Action` arm:

```
$ quint typecheck nexus_caller.qnt
nexus_caller.qnt:281:20 - error: [QNT404] Name 'timeoutt' not found
nexus_caller.qnt:281:20 - error: [QNT000] Couldn't unify variant and variant
Trying to unify variants
  | Schedule(Deadlines) | HandlerReply(Reply) | ... | StartToClose
and
  | Schedule(Deadlines) | HandlerReply(Reply) | ... | ScheduleToStart | r
```

The second message is how a non-exhaustive `match` surfaces: the inferred row type is open and
does not unify with the closed `Action`. It points at the `match`, not at the missing arm.

A find Query whose scenario does not reach its claim (`quint test`):

```
$ quint test nexus_caller.qnt --main nexusProtocol
  nexusProtocol
    [PASS] syncReplied
    [FAIL] retry
      nexus_caller.qnt:423:15 - error: [QNT508] Assertion failed
      Use --seed=0x1a2b3c --match=retry to repeat.
  7 passed, 1 failed (output shape approximate; not run)
```

An invariant violation under exploration (`quint run`), which is also the shape of an Apalache
counterexample:

```
$ quint run nexus_caller.qnt --main nexusProtocol --invariant properties --max-steps 4
An example execution:
[State 0] { last: None, state: { attempts: 0, phase: Unscheduled, ... } }
[State 1] { last: Some({ action: Schedule({ ... }), before: ..., facts: [NexusOperationScheduled], outcome: Accepted }), state: { phase: Scheduled, ... } }
[State 2] { last: Some({ action: HandlerReply(HandlerError(true)), ..., facts: [PendingAttempts] }), state: { phase: BackingOff, attempts: 1, ... } }
[violation] Found an issue (1234ms).
error: Invariant violated
```

The trace is the value of every `var` per state, so `last` is what tells a reader which class
fired and what it recorded. There is no token-pinned error: Quint reports the line of the
`expect`, never the line of the action or the property that was wrong.

## Edit loop and toolchain

- `npm i -g @informalsystems/quint`; VS Code and Neovim extensions run the typechecker on save.
- `quint typecheck`: under a second. `quint test`: seconds, including the pins. `quint run`
  with `--max-samples 10000 --max-steps 4`: seconds. Since v0.33 the Rust evaluator is the
  default backend for `run`, `test` and the REPL and takes `--mbt`; `--backend typescript` is
  the fallback.
- `quint verify` needs Java 17 and Apalache. First call downloads it; each call pays JVM
  startup unless an Apalache server is kept running (`--server-endpoint`). A depth-4 check on
  either protocol module is seconds of solving after a ten-second startup.
- Traces: `quint test --out-itf` and `quint run --out-itf --mbt` write ITF JSON. `--mbt` adds
  `mbt::actionTaken` (the action *name*, which for the exploratory `step` is always `fire`) and
  `mbt::nondetPicks`; the samples record the class themselves in `last`, so the Go side needs
  only `last` and `state`.

## Libraries to leverage

Maintenance checked on 2026-09-29 with `gh api repos/<owner>/<repo>`. "No-go" means archived
or no push in the last 12 months. Only the maintained rows are recommendations.

| Repo | What it is | Last push | Status | What it replaces / what is still Go work |
|---|---|---|---|---|
| [quint-co/quint](https://github.com/quint-co/quint) (was informalsystems/quint) | Quint CLI: parser, typechecker, `test`/`run`/`verify`, REPL. Also the `vscode/` extension and `evaluator/`, the Rust simulator that is the default backend since v0.33 | 2026-09-28, v0.33.0 | maintained | Replaces the whole model layer's checker, simulator and editor support. Go still has to invoke it and read its output. The `evaluator/README.md` checklist that still calls TypeScript the default is stale against the v0.33 CLI |
| [apalache-mc/apalache](https://github.com/apalache-mc/apalache) | Symbolic model checker for TLA+ and Quint; the backend of `quint verify` | 2026-09-24, v0.62.2 | maintained | Replaces the exhaustive `verify` Query. Nothing to write; JVM and a server process to run in CI |
| [quint-co/quint-connect](https://github.com/quint-co/quint-connect) | Rust MBT framework: generates traces from a Quint spec, replays them against a Rust implementation, checks state after each step | 2026-05-25, v0.1.1 | maintained | Rust only. It is the design to copy for the Go adapter (trace generation, per-step replay, state comparison), not a dependency |
| [quint-co/quint-trace-explorer](https://github.com/quint-co/quint-trace-explorer) | Rust TUI for stepping through ITF traces | 2026-01-30 | maintained | Debugging aid for authors reading a failed `quint run` or `verify` trace; replaces nothing in the pipeline |
| [ITF format (ADR-015)](https://apalache-mc.org/docs/adr/015adr-trace.html) | The JSON trace format `--out-itf` writes: `#meta`, `vars`, `states`, with `#bigint`, `#set`, `#map`, `#tup` encodings and `{ tag, value }` for sum types | spec document | stable | The one contract between Quint and Go |
| [informalsystems/itf-go](https://github.com/informalsystems/itf-go) | Go library for unmarshalling ITF files; used by cosmos/interchain-security's MBT tests | 2023-11-10 | not maintained, no-go | Would have been the parser. Small enough to reimplement |
| [informalsystems/itf-rs](https://github.com/informalsystems/itf-rs) | Rust ITF parser | 2025-05-28 | not maintained, no-go | Rust only anyway |
| [informalsystems/vscode-itf-trace-viewer](https://github.com/informalsystems/vscode-itf-trace-viewer) | VS Code ITF viewer | 2024-04-11 | not maintained, no-go | Use quint-trace-explorer instead |
| [informalsystems/modelator](https://github.com/informalsystems/modelator), [informalsystems/atomkraft](https://github.com/informalsystems/atomkraft) | Earlier Informal MBT tooling | 2025-03-12, 2023-04-06 | not maintained, no-go | Predecessors of quint-connect |
| [leowmjw/go-quint-raft](https://github.com/leowmjw/go-quint-raft) | Personal Go port of quint-connect with Raft, 2PC and tic-tac-toe examples | 2026-05-11 | active by date, 0 stars, single author | Not a dependency to adopt. Worth reading as an existence proof of the Go adapter shape |
| [aburan28/tlacuilo](https://github.com/aburan28/tlacuilo) | Pure-Go TLA+ toolkit: builder, parser, TLC runner, ITF import/export in its `trace` and `value` packages | 2026-07-26 | active by date, 0 stars, single author | TLC-oriented, not Quint. Its ITF value package shows the decoding rules but is not something to depend on |
| [konnov/itf-py](https://github.com/konnov/itf-py) | Python ITF parser and emitter by Quint's original author | 2026-03-16 | maintained, not applicable | Python; useful only for one-off trace analysis |

What the Go side still has to write, with these in place:

- An ITF reader (a few hundred lines: the ADR-015 encodings, plus decoding `last` and `state`
  into Go structs). No maintained Go library exists for this.
- The trace-to-Case producer: for each state of a `quint test --out-itf` trace, map `last.action`
  to a Program instruction and `last.facts` through the evidence names to evidence declarations.
  This is the part quint-connect does for Rust and nothing does for Go.
- The set and realization layer, which Quint does not model at all.
- CI plumbing: `quint typecheck`, `quint test` for pins and Queries, `quint run` under a fixed
  `--seed` for the exploratory coverage, `quint verify` with an Apalache server for the
  whole-machine invariants.

## Honest costs

- No realizations, no Cases. Quint ends at the trace. The Go runtime needs an ITF adapter that
  reads `state`, `last.action`, `last.facts`, `last.outcome` per step and maps them to Program
  instructions and evidence declarations. The evidence names are `pure def evidence(fact)` in the
  tables and would have to be either duplicated in Go or extracted with `quint compile`.
- No proto schemas, parties or examples with any meaning. They are strings in a `pure val`.
  Nothing checks that `handlerError(false) -> BadRequest` names a real HandlerError type.
- Sets are inert data. A query named in a set is a string; renaming the `run` does not break
  the set. The canary admission rule (no silent step on the path) has no Quint expression because
  Quint has no notion of observability; `unobservable` is a `pure val` nothing reads.
- The `Limits` triple has no home. Runs have fixed length; the exploratory `step` is bounded
  from the command line. The search count is meaningless in Quint.
- No `deriving Finite`. Each table lists its phases and builds its state set by hand; a phase
  added to the type but not to `phases` is silently absent from every count and from the
  refinement pin. The typechecker does catch the missing `match` arm in `steps` and `productOf`.
- Constructor labels are module-global, hence the `Resolved*` and `Attempt*` prefixes and the
  Table/machine module split. Six modules per Model file is more scaffolding than the Lean.
- `fire` and the `last` bookkeeping repeat per machine; there is no way to write them once.
- Stutter-invariance and the product-claim-through-the-map argument are pins, not a compile-time
  derivation, and the transition claims are checked on `last.before`, a history variable, rather
  than on a true two-state formula.
- Small team, small ecosystem. One implementation, a handful of maintainers; breaking syntax
  changes happened in 2023 and 2024. Apalache is a JVM dependency and its own project.

Both refinements hold over the whole table under the mapped-states rule (a protocol row is a
product stutter or a product row between the mapped states under any class); `pins.qnt`
asserts `rejectedRows(...) == Set()` for each Model, and the Activity pins name the rows that
needed the rule to be by states rather than by action name.

What Quint made easy: the machines are short, the step functions are the Lean ones with
different brackets, the refinement is fifteen lines of pure code plus a one-line invariant, and
every pin is an expression a reader can paste into `quint -r nexus_caller.qnt::nexusProtocolTable`
and evaluate in the REPL.
