# Elixir sample

The two Models of `SPEC.md`, authored in Elixir 1.20 against the `umpire` library in `lib/`. This
sample builds and runs. It was compiled and tested with Elixir 1.20.4 on Erlang/OTP 28.5, installed
through mise:

```
mise x erlang@28.5 elixir@1.20.4-otp-28 -- mix test
```

Verified on 2026-09-29, first by the team lead in a scratch project and then from this directory
as it stands:

- **Compile.** It compiled on the first try with one type warning, in framework code. That warning
  is fixed, and `mix compile --warnings-as-errors` is clean.
- **Pins.** All 21 tests in `test/pins_test.exs` pass, in 0.2 seconds of test time.
- **Planted mistakes.** Each was caught where the design says. A missing arm, a misspelled fact, a
  dead arm, a call outside the subset, an undeclared action and a broken refinement all stopped
  `mix compile`. An unfindable retry claim failed `mix test` with the searched trace. The texts
  below are the real output.
- **IR export.** `mix umpire.ir` crashed on its first run, because maps keyed by input values
  could not be written as JSON. It is fixed. The export is deterministic: `mix umpire.ir --check`
  passes on a second run.

The layout:

```
mix.exs                       compiles lib/ and the three Model files below
lib/umpire.ex                 Umpire, Umpire.Step, Umpire.Diagnostic, Umpire.Names
lib/umpire/model.ex           the declaration macros and the Model's @before_compile
lib/umpire/machine.ex         the machine block's macros (defstep, defmap, starts, ...) and its hook
lib/umpire/lower.ex           the expression subset: AST to IR, IR back to Elixir
lib/umpire/{ir,domain,eval,table,refinement,compose,check,search,case}.ex
lib/mix/tasks/umpire.ir.ex    mix umpire.ir: canonical JSON of each Model's IR, into priv/umpire/
worker.ex                     the worker entity and its polling machine
nexus_caller.ex               Model 1
standalone_activity.ex        Model 2
test/pins_test.exs            the pins
```

The Model files stay at the top of the directory, beside the other languages' samples, and
`mix.exs` names them in `elixirc_paths`. In a real repository they would live under `lib/`.

## The DSL mechanism

Macros in the style of `Ecto.Schema` and `Phoenix.Router`. A Model module calls `use
Umpire.Model`. That registers accumulating attributes (`Module.register_attribute(...,
accumulate: true)`) and a `@before_compile` hook. Each declaration is one macro call: `entity`,
`domain`, `defstate`, `action`, `observation`, `defmachine`, `defproperty`, `defscenario`,
`deflimits`, `defquery`, `defset`, `defcompose`. Each call appends one IR node, and the hook checks
them together.

The central design choice is that **each `defmachine` emits two things from one source**.

- **An IR.** `Umpire.IR` is plain structs, and patterns, guards, updates and predicates are
  tagged tuples, not closures. The table, the coverage check, the refinement and the search all
  read it through a small interpreter, `Umpire.Eval`. `mix umpire.ir` writes it as canonical
  JSON, one `model.ir.json` per Model, for a Go core or a Quint exporter.
- **Real Elixir functions.** Each machine becomes a nested module named by camelizing its atom,
  such as `StandaloneActivity.ActivityProtocol`. It holds one snake_case function per step, such
  as `attempt_result/2`. The author's patterns and guards are kept verbatim. Each body is lowered
  from the same IR into the `%Umpire.Step{}` list it stands for. The function head pins the state
  struct and each input's domain, so the Elixir 1.20 type checker sees typed step bodies.

A step reads like a `def`: one `case` over tuples, with `when` guards and `p when p in [...]`
or-patterns.

```elixir
defstep attemptResult(state, result) do
  case {state.phase, result} do
    {:started, {:failed, true}} -> moves(:backingOff, [:statusScheduled, :attemptCount])
    {:pauseRequested, {:failed, true}} -> moves(:paused, [:statusPaused])
    {phase, _} when phase not in [:started, :cancelRequested, :pauseRequested] -> []
    ...
  end
end
```

The macros enforce what the language cannot:

- **Finite domains.** A field type is `:boolean`, a `Range`, or a module declared with `domain`
  or `defstate`, so every state space is finite by construction.
- **Exhaustiveness and dead arms.** `Umpire.Table.build!` evaluates every step on every state and
  every class of its action. A (phase, input) combination that no clause matches is an error, and
  so is a clause that never matches first.
- **The expression subset.** `Umpire.Lower` rejects every function call except the step
  constructors (`moves`, `stay`, `not_found`, `succ`). It also rejects `send`, `receive`, `if`,
  pipes, comprehensions and rebinding. A body stays data.

**Why two passes.** Elixir expands a whole module body before it evaluates it. A macro that read
`@running` while expanding would see nil. So each macro checks only its own shape at expansion
and returns code that lowers the declaration while the body evaluates. The functions it emits use
unquote fragments (`def unquote(step.fun)(...)` under `bind_quoted`), the technique
`Phoenix.Router` uses. Cross-declaration checks wait for `@before_compile`.

**Naming convention.** The spec's vocabulary is flat. It reuses `completed`, `scheduled`,
`canceled`, `timeout` and `handlerError` across enums, facts, actions and declarations. This
sample keeps every spec name verbatim as an atom, such as `:attemptResult`, `:pauseRequested` and
`:statusCompleted`. Atoms have no namespace, and each declaration kind is its own registry in the
IR. A reference is resolved in the registry its position implies: `in:` looks up scenarios, a
pattern position looks up that subject's domain. So the phase `:completed`, the `AttemptResult`
member `:completed` and the scenario `:completed` coexist. The query `:handlerError` and the reply
`{:handlerError, true}` coexist too. Nothing was renamed.

The only derived names are Elixir identifiers, and one function makes them (`Umpire.Names`):

- Domain and state types keep the spec's PascalCase as nested modules (`Phase`, `ProtocolState`).
- A machine becomes a nested module by `Macro.camelize` (`:nexusProtocol` becomes `NexusProtocol`).
- A step, map or property becomes a function by `Macro.underscore` (`:attemptResult` becomes
  `attempt_result`).

Two machines both step on `:attemptResult` without clashing, because each function lives in its
machine's module. Property predicates live in the Model module, whose only other function is
`__umpire__/1`.

## Where each check runs

| Check | When | How |
|---|---|---|
| Declaration shape: known keys, atom names, `def`-like heads, literal scenario classes, sync lines | `mix compile`, at expansion | each macro, `CompileError` at the call's line |
| Names: actions and timers of a step, phases in `starts`/`ends`, evidence keys, `unobservable` subset, entities | `mix compile`, machine hook | `Umpire.Check.machine!` and `Umpire.Lower.step!` |
| Head arity and input names against the action's `input:` | `mix compile`, body evaluation | `Umpire.Lower.step!` |
| Finite domains, state-space sizes | `mix compile`, body evaluation | `Umpire.Domain`; `ProtocolState.size()` is a compiled-in integer |
| Expression subset: no calls, sends or side effects | `mix compile`, body evaluation | `Umpire.Lower` |
| Exhaustiveness and dead arms of every step, per (phase, input) | `mix compile`, machine hook | `Umpire.Table.build!` |
| Produced values in their domains (phase, updates, facts, outcome) | `mix compile`, machine hook | same pass |
| Map totality and codomain; the refinement under the strict Lean rule | `mix compile`, protocol machine hook | `Umpire.Refinement.check!` |
| Query admission: names, a product property read only through a refinement, `when:` action on the path's machine, scenario length within limits, find needs a same-step claim | `mix compile`, Model hook | `Umpire.Check.model!` |
| Sets: purpose-specific keys, find-only queries, every party on the paths bound; composition sync names | `mix compile`, Model hook | same |
| Type consistency of the emitted step functions and property predicates | `mix compile`, as warnings | the Elixir 1.20 type checker; `--warnings-as-errors` makes them fail |
| One count pinned before any test runs | when ExUnit compiles `pins_test.exs` | `static_assert/1` |
| IR interpreter and compiled functions agree on every row | `mix test` | `Umpire.Case.assert_agrees/2` |
| Find and verify queries, and the vacuity rule | `mix test` | `Umpire.Search.run/2` |
| Table pins, reachability, stuck states, refinement rows | `mix test` | ExUnit |
| Canary admission: every step of the path records evidence | not implemented | described in the Model's comments only |
| Exploratory coverage targets | `mix test`, sketched | `Umpire.Search.explore/2` |
| Case lowering, realization, the Go runtime | not in this sample | |

Almost everything the Lean elaborator decides is decided here by `mix compile` as well, including
the table and the refinement. Those are finite computations, and the macro can run the interpreter
at compile time. Only the queries wait for `mix test`. They could run in the hook too, since the
guided search visits a handful of nodes. They stay in tests for two reasons. A failed query should
be a failing test with a trace, not a build that will not start. And a Model with a wrong claim
should still compile, so its other pins can run.

**What the type checker does and does not add.** Elixir 1.20 infers types from patterns, guards
and bodies, with no user-written signatures yet. It reports violations during `mix compile` as
warnings, for example reading a field a struct does not have, a comparison that can never hold,
or a clause it can show to be redundant or unreachable. It does not report a `case` that lacks a
clause for some value; that is a `CaseClauseError` at run time. In this design the Model-specific
checks all belong to the macros, and the type checker is a second opinion. It checks the code the
macros generate, and it is the only checker for the ordinary Elixir around the Models, such as
test helpers and realization glue.

Observed on this sample, the type checker reported one warning, in framework code. `min/2` was
applied to fields it could not prove were integers, which is a structural comparison that would be
meaningless on structs. The values were integers, so it was not a live bug. A guard now states the
fact. The checker reported nothing on the emitted step functions or on the Models. The Umpire errors
also preempt it: coverage and refinement fail in `@before_compile`, before any function is type
checked. So on a planted mistake inside a step, the author sees Umpire's message and never the
type checker's.

## What an author's error looks like

Each text below was produced by planting the mistake in `standalone_activity.ex` and running
`mix compile` or `mix test`. Under each message, Mix also prints the framework's own stack frames,
such as `lib/umpire/table.ex:84: Umpire.Table.coverage!/3`; they are left out here.

A step naming an undeclared action (`defstep attemptResults(state, result)` on line 181):

```
== Compilation error in file standalone_activity.ex ==
** (CompileError) standalone_activity.ex:181: defstep attemptResults/2 in :activityProduct names no declared action and no timer of this machine
    actions: :attemptResult, :attemptStart, :control, :start, :workerStop
    timers:  :timeout
    did you mean :attemptResult?
```

A non-exhaustive match. The same text reports a combination that is missing, and a typo in a
literal, because both leave some combination with no clause. Here the arm
`{:pauseRequested, :completed}` on line 385 is deleted:

```
** (CompileError) standalone_activity.ex:373: defstep attemptResult/2 has no clause for 1 of its {phase, result} values:
    {:pauseRequested, :completed}
  Every combination needs a clause; where the action is not enabled the clause returns [].
```

Neither `mix compile` without Umpire nor the type checker would say anything. The first
`attemptResult` from `pauseRequested` would raise `CaseClauseError` in a test.

A dead arm, here `{_, :requestCancel} -> []` added after the exhaustive clause on line 227:

```
** (CompileError) standalone_activity.ex:230: this clause of defstep control/2 never matches first: every {phase, control} value it
  matches is taken by an earlier clause, or it matches none (check its literals against
  their domains).
```

A body outside the subset, `attempts: Enum.min([2, state.attempts + 1])` in place of `succ/1`:

```
** (CompileError) standalone_activity.ex:360: defstep attemptStart/1 in :activityProtocol: the call Enum.min/1 is outside the Umpire expression subset.
  Allowed: literals, bound variables, inputs, state fields, succ/1, tuples, lists, @attributes,
  ==, !=, in, not in, and, or, not. No function calls, sends or side effects.
```

Rebinding inside a pattern, here `{x, x} -> []` added to the protocol's `attemptResult`. The BEAM
would read it as an equality test and `Umpire.Eval` as an overwrite, so it is rejected. So is a
pattern variable that shadows the state or an input:

```
** (CompileError) standalone_activity.ex:388: defstep attemptResult/2 in :activityProtocol: x is bound twice in one pattern; use a guard to compare
```

A refinement that does not hold, at compile time. This is the first draft of `SPEC.md`, where
`pauseRequested` mapped to `paused`. The first row the check rejects is a completion under a
requested pause:

```
** (CompileError) standalone_activity.ex:473: :activityProtocol does not refine :activityProduct (rule :strict):
    the row pauseRequested-0-unset-unset-unset --attemptResult-completed--> completed-0-unset-unset-unset
    maps paused --> completed, which is not a stutter,
    and :activityProduct has no row between them
```

A failed query is a failed test. `retryCompletes` is written with `attempts: 1`, copied from the
Nexus Model:

```
  1) test standalone activity every find query finds its claim on its path (Temporal.Feature.PinsTest)
     test/pins_test.exs:186
     expected :found, got :not_found
     query :retry did not find :retryCompletes in :retriedThenCompleted within :six
       6 steps searched, 1 paths, the claim fired 1 times
         1. start-unset-unset-unset -> %Temporal.Feature.StandaloneActivity.ProtocolState{phase: :scheduled, attempts: 0, scheduleToClose: :unset, scheduleToStart: :unset, startToClose: :unset} [:statusScheduled]
         2. attemptStart -> %Temporal.Feature.StandaloneActivity.ProtocolState{phase: :started, attempts: 1, ...} [:statusStarted, :attemptCount]
         3. attemptResult-failed-true -> %Temporal.Feature.StandaloneActivity.ProtocolState{phase: :backingOff, attempts: 1, ...} [:statusScheduled, :attemptCount]
         4. backoff -> %Temporal.Feature.StandaloneActivity.ProtocolState{phase: :scheduled, attempts: 1, ...} []
         5. attemptStart -> %Temporal.Feature.StandaloneActivity.ProtocolState{phase: :started, attempts: 2, ...} [:statusStarted, :attemptCount]
         6. attemptResult-completed -> %Temporal.Feature.StandaloneActivity.ProtocolState{phase: :completed, attempts: 2, ...} [:statusCompleted]
```

The trace shows the mistake directly: the claim fired at step 6 on `attempts: 2`, and the claim
asked for 1. The states are shortened with `...` from step 2 on; the real output prints every
field.

A verify whose claim never fires is an error, not a pass, because `require_firing:` defaults to
true. The Lean framework defaults it to false and reports "coverage not exercised". This is the
old `stoppedBeforeDispatch` path, formatted by `Umpire.Search.format/2`:

```
query :stoppedWorkerStartsNothing is vacuous: :startedByPollingWorker never fired on :stoppedBeforeDispatch; a verify whose claim never fires proves nothing (add the action its when: names to the path, or pass require_firing: false)
  3 steps searched, 1 paths, the claim fired 0 times
```

## Toolchain and loop

The toolchain is Erlang/OTP with Elixir 1.20. Elixir v1.20.4 was published 2026-08-28, per
`gh api repos/elixir-lang/elixir/releases`. Builds go through Mix. There are no dependencies
beyond the standard library: `JSON` has been in Elixir since 1.18, and ExUnit and `Mix.Task` ship
with it. Editors get inline diagnostics from Expert, the official language server, or ElixirLS.
A `CompileError` from a macro shows on the author's line like a syntax error, because every IR
node carries the file and line of its declaration.

Measured on an Apple M2 Max:

| Step | Measured |
|---|---|
| `mix compile` with nothing changed | 0.5 s |
| Recompile after editing `standalone_activity.ex` | 5.0 to 5.7 s |
| Clean build of the library and three Models | 7 to 11 s |
| `mix test`, wall clock, with VM boot | 0.8 s, of which the tests take 0.2 s |
| Recompile after editing a file in `lib/umpire` | over 10 s for the activity Model alone |

- **The Model edit is the slow edge.** About five seconds is slower than a hand-written module of
  the same size. Every table is compiled into its module as a literal: the exported IR is 1.5 MB
  for Nexus and 2.1 MB for the activity. That is the likely cost, but it was not profiled. Keeping
  the tables out of the BEAM files, for example building them on first use, is the obvious fix to
  try.
- **Dependent Models recompile too.** Mix follows compile-time dependencies. `StandaloneActivity`
  depends on `Worker` and on the Nexus `Timeout` domain, so it recompiles when either changes.
- **`mix test.watch` from `mix_test_watch`** reruns on save.
- **A framework edit recompiles every Model.** Each Model's hook calls the framework at compile
  time. Authoring a Model does not touch `lib/umpire`.

## Honest notes

What Elixir made easy:

- **Declarations read as configuration.** `defquery :retry, find: :retryCompletes, in:
  :retriedThenCompleted, limits: :six` is one line with no ceremony. Keyword lists accept `for:`,
  `when:` and `in:` as keys, so the spec's words survive.
- **Tuple patterns with guards are the step table.** `{phase, :expires} when phase in @running`
  is the Lean row almost verbatim. Payload constructors are just tuples (`{:failed, true}`), so a
  class and its pattern are the same literal.
- **The compile-time VM is the whole language.** The interpreter, the table and the refinement run
  inside `@before_compile` as ordinary Elixir. Nothing is restricted to `const` evaluation, so the
  compile-time tier is as large as the Lean one.
- **Macros that read the AST.** Because a `case` arrives as data, the same body can be enumerated,
  exported and compiled. Rust's proc macros would need `syn` for that, and Kotlin's builders
  cannot see a lambda's body at all.

What it made awkward:

- **Two readings of one body.** The BEAM runs the emitted `case`, while the table and the search
  run `Umpire.Eval`'s matcher. They are generated from one source, but pattern matching is
  implemented twice. `assert_agrees/2` compares them on every row, so a divergence fails a test,
  but it is still a second implementation to maintain.
- **The expression subset is small, and growing it is framework work.** Nexus `complete` needs two
  arms per resolution because `++` is not in the subset. Each new construct needs a lowering, an
  evaluator clause, an emitter clause and a JSON form.
- **No exhaustiveness from the language.** The only exhaustiveness check is the macro's
  enumeration. It works because every domain is small and finite. A Model with an integer or
  string input would need a different check.
- **Expansion versus evaluation.** A struct module created by `defstate` exists only once the body
  runs, so the Models build the retry states with `struct!(ProtocolState, %{...})`, not a
  `%ProtocolState{}` literal. Attributes of the Model are invisible inside a machine's module, so
  the macros resolve `@productTerminal` themselves and substitute its value. Both are invisible
  when they work and baffling when they do not.
- **Atoms are untyped outside the DSL.** Inside a `defstep`, a misspelled `:complted` is caught
  because the macro checks every combination. In a test helper it is just another atom. The type
  checker narrows emitted functions to their input domains and may flag it, but nothing is
  guaranteed there.
- **Style friction.** Spec names are camelCase atoms, keys and attributes (`:attemptResult`,
  `scheduleToClose:`, `@attemptBound`) where Elixir style is snake_case. Credo's naming checks
  would need relaxing for the Model directory. `after` is reserved, so a transition claim's second
  step is `next`.
- **Macro diagnostics are ours to write.** Every message above is a string in `lib/umpire`. The
  quality of the errors is exactly the care put into them, and they carry a line, not a column.
- **A second runtime.** A Go monorepo gains Erlang/OTP and Mix in CI, and contributors who read
  quoted AST to change the DSL. Elixir's hot compile loop and REPL (`iex -S mix`) are the upside.
- **Macro errors carry framework stack frames.** Mix prints the `lib/umpire` frames under every
  `CompileError`. An author must learn to read only the first lines.
- **Unwritten parts.** The canary evidence check is described in the Model's comments, not
  implemented. `explore/2` is a sketch. Case lowering and realization are out of scope, as for
  every non-Lean sample.

Spec fidelity:

- **The revised Model 2 is applied.** `pauseRequested` maps to `started`. The retry rows are
  visible on the product, `started` to `scheduled` and `cancelRequested` to `canceled`.
- **The strict rule is carried.** This sample implements the stricter Lean refinement rule as the
  default, and the mapped-states rule as `rule: :mapped_states`. It therefore carries the extra
  fact: protocol `attemptResult({:failed, true})` from `:started` records `[:statusScheduled,
  :attemptCount]`. A pin shows the strict rule rejects the row without it and the mapped-states
  rule does not.
- **The composition path is `stoppedBeforeRetry`, with limits `:six`.** A pin checks that its
  claim fires once, and another that the old path is now `:vacuous`.

## Libraries to leverage

Maintenance was checked on 2026-09-29 with `gh api repos/<owner>/<repo> --jq '{pushed_at,
archived, stargazers_count}'`. The rule is that a repository archived, or with no push in the last
twelve months, is "not maintained, no-go". Only the maintained rows are recommendations.

| Library | Repository | Last push | Stars | Status | What it would replace or add |
|---|---|---|---|---|---|
| ExUnit, Mix, `JSON` | elixir-lang/elixir | 2026-09-29 | 26,674 | maintained | The test tier, the mix task, and JSON encoding. `JSON` has been in the standard library since 1.18, so no dependency is needed for the IR export. |
| Erlang/OTP | erlang/otp | 2026-09-30 | 12,341 | maintained | The runtime. Also `:digraph` for reachability and strongly connected components if the table walk grows. |
| StreamData | whatyouhide/stream_data | 2026-09-08 | 947 | maintained | Generators and `check all` properties, for example random class sequences over the table for the exploratory set. It has no stateful model-testing module (its `lib/` holds `stream_data.ex` and `ex_unit_properties.ex` only), so it does not replace a state-machine harness. |
| PropEr | proper-testing/proper | 2026-06-24 | 919 | maintained | Stateful property testing (`proper_statem`), callable from Elixir as `:proper_statem`. This is the closest thing to a reference-versus-system harness for driving a realization from the table. |
| PropCheck | alfert/propcheck | 2025-04-21 | 393 | not maintained, no-go | The Elixir wrapper for PropEr. Use PropEr directly instead. |
| protobuf (elixir-protobuf) | elixir-protobuf/protobuf | 2026-09-25 | 907 | maintained | The `schema:` strings: generated message modules instead, so a renamed message fails compilation. Also the Case format. |
| grpc (elixir-grpc) | elixir-grpc/grpc | 2026-09-08 | 1,527 | maintained | A gRPC client to the Go Testpilot runtime, if the search ever drives it directly. |
| Jason | michalmuskala/jason | 2026-05-05 | 1,678 | maintained | Optional. `Jason.OrderedObject` gives ordered keys, but the built-in `JSON` plus the sorted writer in `Umpire.IR` covers the export. |
| rfc8785 | yaglo/rfc8785 | 2026-07-24 | 0 | maintained, unproven | RFC 8785 canonical JSON for the Behavior Fingerprint. It passes the push rule but has no users. The IR has no floats, so the sorted-key writer plus a golden test suffices. |
| jcs | pzingg/jcs | 2025-03-31 | 12 | not maintained, no-go | The older RFC 8785 implementation. |
| NimbleParsec | dashbitco/nimble_parsec | 2026-08-13 | 882 | maintained, not needed | A parser, if Models were ever written in a `.umpire` text syntax instead of Elixir. The macro DSL makes it unnecessary. |
| Spark | ash-project/spark | 2026-09-15 | 205 | maintained | The DSL framework behind Ash. Declarative sections and entities with schema-validated options, transformers and verifiers (our hooks), generated docs, and editor autocompletion for DSL keys. It would replace most of `Umpire.Model`'s option parsing and `Diagnostic.keys!`, but not `Umpire.Lower`, because Spark validates options, not function bodies. |
| Credo | rrrene/credo | 2026-09-29 | 5,221 | maintained | Linting. Custom checks could enforce Model conventions (every `defstep` commented, no `_` catch-all hiding a phase). Its naming checks need relaxing for camelCase atoms. |
| Dialyxir | jeremyjh/dialyxir | 2026-09-21 | 1,796 | maintained | Dialyzer over the generated `@type t` of each domain and state. It overlaps with the 1.20 type checker. |
| mix_test_watch | lpil/mix-test.watch | 2025-12-19 | 955 | maintained | Rerun `mix test` on save. |
| Expert | elixir-lang/expert | 2026-09-30 | 2,060 | maintained | The official language server. Inline `CompileError`s from the macros as you type. |
| ElixirLS | elixir-lsp/elixir-ls | 2026-08-16 | 1,779 | maintained | The older language server, same role. |
| libgraph | bitwalker/libgraph | 2024-08-20 | 571 | not maintained, no-go | A graph library for the search and reachability. Erlang's `:digraph` covers it. |
| typed_struct | ejpcmac/typed_struct | 2025-12-18 | 770 | maintained, not needed | Struct plus typespec. `defstate` generates both. |
| temporalex | cgreeno/temporalex | 2026-09-11 | 9 | maintained, not an option | A community Temporal SDK on the Rust Core SDK through Rustler NIFs. It is the most active of several small ones: hansihe/temporal_ex (2026-06-25, 2 stars, "prototype") and polymorfiq/temporal-elixir-sdk (2026-07-27, 2 stars). None is official. Realization runs through the Go Testpilot runtime anyway, so none is needed. |

The sketched framework uses only the standard library. A real build would add these first:
protobuf for the schemas and the Case format, PropEr for a stateful harness if one is wanted, and
possibly Spark to shrink the declaration macros.
