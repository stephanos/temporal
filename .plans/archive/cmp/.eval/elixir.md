# Review: cmp/elixir/

Reviewer read SPEC.md with all three revision notes, every file in `cmp/elixir/`, and
`.walkthrough-outline.md`. No toolchain was run. Paths below are relative to `cmp/elixir/`.

## Scores (1-5, 5 best)

1. **Spec fidelity: 5.** Both Models are complete and use spec names verbatim as atoms. They keep
   the Lean section order behind `# authoring:` markers. Model 2 carries the revision:
   `pauseRequested -> started` is at `standalone_activity.ex:476`, and the visible retry rows are
   at `:189-193`. The strict-rule fact `[:statusScheduled, :attemptCount]` is at `:377`, and
   `stoppedBeforeRetry` with `:six` is at `:759-774`. All pins in both lists are present in
   `test/pins_test.exs`.
2. **Language plausibility: 4.** The macros are what a strong Elixir developer would write, in the
   style of Ecto and Phoenix. They use accumulating attributes, `@before_compile`, `bind_quoted`
   with unquote fragments, `Module.create` for nested domain modules, and `Code.ensure_compiled`
   for cross-Model domains. The timing premise, that a module body is expanded before it is
   evaluated, is correct and is handled consistently. The step is dropped from 5 because the code
   has never compiled and the design leans on subtle macro timing. Two small claims are also
   unimplemented; see Red flags.
3. **Authoring readability: 4.** Queries, sets, limits and scenarios read as one-line
   configuration, and step functions are an ordinary `case` over tuples with `when` guards, close
   to the Lean rows. There is some ceremony. Nine-line identity `evidence` maps repeat the fact
   names (`:327-336`). `@completedOnRetry` needs a `struct!/2` workaround (`:515-521`), and Nexus
   `complete` needs two arms per resolution because `++` is outside the subset
   (`nexus_caller.ex:339-350`). The camelCase atoms also sit against Elixir style.
4. **Check story accuracy: 4.** The README's check table (`README.md:95-114`) matches what the
   macro code does. Its type-checker paragraph (`README.md:123-131`) says what Elixir 1.20 does:
   it infers types and reports redundant or unreachable clauses, has no user signatures, and does
   not report a missing `case` clause. Exhaustiveness is correctly credited to the macro's
   enumeration. The step is dropped because of two small overclaims. The subset "rejects
   rebinding" but nothing does. One sample error message is not what the checker would print
   first.
5. **Framework realism: 5.** This is the most complete framework surface a sample could offer
   without compiling. Enumeration, table building with fall-through and dead-arm detection, domain
   checks on outputs, map totality, and the strict refinement are all written out. So are the
   scenario-guided search with node budget and `:exhausted`, the vacuity rule, restriction,
   composition with sync classes, admission checks, and canonical JSON export. Only
   `Search.explore/2` is a sketch, and the README says so.
6. **README honesty and library table: 5.** Costs are stated plainly. They include two pattern
   matchers to keep in step, a narrow subset, a second runtime in a Go monorepo, hand-written
   diagnostics with no column numbers, and unwritten canary admission. Four spot-checks with
   `gh api` match the table exactly (see below), and so does the Elixir v1.20.4 release date.

Library spot-checks (`gh api repos/<r> --jq '{pushed_at,archived,stargazers_count}'`,
2026-09-29):

| Repository | README says | gh api | Match |
|---|---|---|---|
| ash-project/spark | 2026-09-15, 205, maintained | 2026-09-15, 205, not archived | yes |
| alfert/propcheck | 2025-04-21, 393, not maintained | 2025-04-21, 393, not archived | yes |
| bitwalker/libgraph | 2024-08-20, 571, not maintained | 2024-08-20, 571, not archived | yes |
| whatyouhide/stream_data | 2026-09-08, 947, maintained | 2026-09-08, 947, not archived | yes |
| elixir-lang/elixir releases | v1.20.4 on 2026-08-28 | v1.20.4 published 2026-08-28 | yes |

## Does the macro do what the README claims?

| Claim | Implemented? | Where |
|---|---|---|
| Finite domains | Yes. A field type must be `:boolean`, a step-1 non-empty `Range`, or a module that exports `__umpire_domain__/0` or `__umpire_state__/0`. Anything else raises `CompileError`. Action inputs must be domains. | `lib/umpire/domain.ex:53-76`, `lib/umpire/lower.ex:205-211` |
| Exhaustiveness by enumeration | Yes. Every step is evaluated on every state times every class. Unmatched subject values are collected per action and reported at the `defstep` line. | `lib/umpire/table.ex:52-68`, `:80-89` |
| Dead arms | Yes, for steps. A clause index never answered first raises at that clause's line. `defmap` gets totality and codomain checks but no dead-arm check. | `lib/umpire/table.ex:91-97`, `lib/umpire/refinement.ex:96-106` |
| Out-of-domain outputs | Yes. Next state, outcome and every fact are checked against their domains. | `lib/umpire/table.ex:107-123` |
| Expression subset | Mostly. Any local, remote or anonymous call is rejected, as are `if`, `cond`, `with`, pipes, `for`, interpolation, `send` and `receive` (they are either not a step body or outside `expr!`). Patterns are limited to literals, `_`, variables and tuples. **Rebinding is not rejected**: `pattern!` binds any name without checking it against earlier bindings or inputs. | `lib/umpire/lower.ex:354-365`, `:371-392`, `:403-472`; claim at `README.md:63-65`, `lib/umpire/lower.ex:17-19` |
| Strict Lean refinement rule | Yes, and it is the default. A matching product row must have the same outcome, and its facts' evidence names must be a subset of the protocol row's. `:mapped_states` is opt-in. A pin shows that removing `:statusScheduled` flips strict to rejected while mapped-states still passes. | `lib/umpire/refinement.ex:68-94`, `lib/umpire/machine.ex:102`, `test/pins_test.exs:165-184` |
| `require_firing` defaults to true | Yes. `defquery` defaults it to true. A verify whose claim fired zero times finishes as `:vacuous`, and a pin rebuilds the old path and gets `:vacuous`. | `lib/umpire/model.ex:350`, `lib/umpire/search.ex:69`, `test/pins_test.exs:206-217` |
| One source, two outputs | Yes. `defstep` lowers to `%IR.StepFn{}` and emits `def <fun>(%State{} = s, inputs...) when input in domain`. The author's patterns and guards are kept, with `@attrs` substituted, and each body is replaced by its lowered `%Umpire.Step{}` list. `assert_agrees/2` compares both readings on every row. | `lib/umpire/machine.ex:138-156`, `lib/umpire/lower.ex:502-563`, `lib/umpire/case.ex:41-61` |

Semantics were checked row by row against SPEC.md for both machines of both Models. No drift was
found. In Model 2, every product `control` arm matches the spec. The `requestCancel` arm at
`standalone_activity.ex:227` covers exactly the four non-terminal product phases, so omitting a
fallback there is correct. Every protocol source phase of `attemptResult` and `control` matches.
Timers fire in the right phases. Under the strict rule every non-stutter protocol row has a product
row with the same outcome and a fact subset. That includes the `pauseRequested` retry mapping to
product `control(:pause)` and the typed `statusTimedOut` facts matching the untyped one.

## Verbatim snippets (comments trimmed)

`handlerReply` of `nexusProduct` (`nexus_caller.ex:147-159`):

```elixir
    defstep handlerReply(state, reply) do
      case {state.phase, reply} do
        {phase, _} when phase != :scheduled -> []
        {_, :syncSuccess} -> moves(:succeeded, [:nexusOperationCompleted])
        {_, :async} -> moves(:started, [:nexusOperationStarted])
        {_, :operationFailed} -> moves(:failed, [:nexusOperationFailed])
        {_, :operationCanceled} -> moves(:canceled, [:nexusOperationCanceled])
        {_, {:handlerError, true}} -> []
        {_, {:handlerError, false}} -> moves(:failed, [:nexusOperationFailed])
      end
    end
```

`syncSucceeds` (`nexus_caller.ex:423-426`):

```elixir
  defproperty :syncSucceeds,
    machine: :nexusProtocol,
    when: handlerReply(:syncSuccess),
    holds: fn step -> step.state.phase == :succeeded and :nexusOperationCompleted in step.facts end
```

`syncReplied` and `syncCompletion` (`nexus_caller.ex:499-502`, `:566`):

```elixir
  defscenario :syncReplied,
    model: :nexusProtocol,
    starts: :unscheduled,
    actions: [schedule(:unset, :unset, :unset), handlerReply(:syncSuccess)]

  defquery :syncCompletion, find: :syncSucceeds, in: :syncReplied, limits: :two
```

`nexusCaller` compose block (`nexus_caller.ex:652-663`):

```elixir
  defmachine :handlerWorker, from: {Worker, :polling}, only: [:workerStop, :serve]

  defcompose :nexusCaller,
    for: [:operation, {Worker, :worker}],
    state: NexusCallerState,
    members: [operation: :nexusProtocol, worker: :handlerWorker],
    sync: [
      workerStop: operation.workerStop || worker.workerStop,
      handlerReply: operation.handlerReply || worker.serve
    ],
    starts: [operation: :unscheduled, worker: :polling],
    ends: [operation: [:succeeded, :failed, :canceled, :timedOut]]
```

## Line counts

| File | Lines |
|---|---|
| README.md | 327 |
| WALKTHROUGH.md | 700 |
| nexus_caller.ex | 690 |
| standalone_activity.ex | 777 |
| worker.ex | 88 |
| test/pins_test.exs | 224 |
| lib/umpire.ex | 149 |
| lib/umpire/model.ex | 440 |
| lib/umpire/machine.ex | 178 |
| lib/umpire/lower.ex | 564 |
| lib/umpire/ir.ex | 223 |
| lib/umpire/domain.ex | 144 |
| lib/umpire/eval.ex | 110 |
| lib/umpire/table.ex | 156 |
| lib/umpire/refinement.ex | 123 |
| lib/umpire/check.ex | 265 |
| lib/umpire/compose.ex | 108 |
| lib/umpire/search.ex | 219 |
| lib/umpire/case.ex | 84 |
| lib/mix/tasks/umpire.ir.ex | 51 |
| **Total** | **5620** |
| **Model files alone (nexus + standalone)** | **1467** |

## Red flags

- **The "rejects rebinding" claim is not implemented.** `README.md:65` and
  `lib/umpire/lower.ex:18-19` both say it. `pattern!` (`lib/umpire/lower.ex:356-358`) turns any
  name into `{:bind, name}` without checking it. The two readings then diverge: `Eval.match` just
  overwrites the binding (`lib/umpire/eval.ex:69`), while the BEAM treats a repeated variable in
  `{x, x}` as an equality test. `assert_agrees/2` would catch this in `mix test`, so it is a
  test-time check described as a compile-time one.
- **One README error text is not what the checker prints first.** The sample refinement failure
  (`README.md:181-185`) names `pauseRequested-1-... --control-unpause-->`. Rows are enumerated
  with attempts starting at 0 (`lib/umpire/domain.ex:16-18`), and classes are sorted by key
  (`lib/umpire/table.ex:41`), so `attemptResult-*` comes before `control-*`. Under the old map,
  the first rejected row would be `pauseRequested-0-unset-unset-unset
  --attemptResult-completed-->`, which maps paused to completed with no product row. This is
  cosmetic, but the section says "the texts are what the sketched framework formats".
- **Search depth ignores `limits.actions`.** `Search.walk` bounds depth by `limits.steps` alone
  (`lib/umpire/search.ex:75`). The `min(steps, actions)` rule from SPEC.md is enforced only at
  admission (`lib/umpire/check.ex:186-187`). Where a scenario is longer than the depth, the query
  becomes a compile error rather than the spec's `not found`. A query built directly as IR, as
  the vacuity pin does, bypasses admission. This is a small deviation and arguably stricter.
- **`defmap` gets no dead-arm check.** Only its totality and codomain are checked. The claim
  "exhaustiveness and dead arms of every step" is scoped to steps, so this is not an overclaim,
  but maps are less checked than steps.
- **Nothing was compiled.** The README says this first (`README.md:3-5`), and the design leans on
  macro timing: `Module.create` at body evaluation, unquote fragments inside nested modules, and
  `Code.ensure_compiled` across Models. The reasoning is sound, but first contact with
  `mix compile` will probably surface fixes, as the walkthrough's own weakness section admits.
- **Canary admission is not implemented.** The README says so (`README.md:112`, `:280`). No
  claim is misleading here, but a decision maker should know one spec'd admission rule exists
  only in comments.
- **Nit.** The `nexusProduct` `evidence` list omits `nexusOperationScheduled`
  (`nexus_caller.ex:139-143`). The protocol lists it. This is harmless because `names/2` falls
  back to the constructor name (`lib/umpire/refinement.ex:94`).

No invented APIs were found. There are no misattributed type-checker capabilities and no library
maintenance misstatements.

## Strengths

- **One declaration yields both an IR and real, typed Elixir functions.** An agreement pin
  (`lib/umpire/case.ex:41-61`) turns drift between the two into a failing test. This is the most
  convincing design for a model layer that must also export to a Go core
  (`lib/umpire/ir.ex:196-222`).
- **Exhaustiveness and dead arms are real compile errors at the author's line.** The table build
  enumerates every state and class (`lib/umpire/table.ex:52-98`). It covers exactly the gap the
  1.20 type checker leaves, and the README draws that line correctly.
- **The strict Lean refinement rule is the default.** It comes with a pin that reproduces the
  second revision note under both rules (`test/pins_test.exs:165-184`). The third revision note is
  also pinned: the new path fires once and the old path is rebuilt and comes back `:vacuous`.
- **Verbatim spec names without renaming.** Registries are position-scoped, so `:completed` as a
  phase, a result and a scenario coexist, and declarations read as configuration.
- **An honest README.** The check-location table matches the code, the costs section is candid,
  and every library row checked matches `gh api`.

## Walkthrough check (against `.walkthrough-outline.md`)

- **Sections.** All 17 are present, in order, with the outline's headings, numbered
  (`WALKTHROUGH.md:3` through `:681`).
- **Excerpts.** All 33 fenced code excerpts carry a `file:line` header. A script compared each
  one with the cited range of the cited file, and every one matches byte for byte.
- **Length and style.** The file is 700 lines, at the top of the 400 to 700 target. It has no
  em-dashes. Language features are introduced before use in section 2: atoms, tuples and
  patterns, `case` with guards, keyword lists, structs, attributes, macros, and the
  expansion-before-evaluation timing rule.
- **Required content.** The four action classes each of `AttemptResult` and `control` are shown
  (`:66-72`). The enumeration mechanism is shown (`domain.ex:94-99`). The product and protocol
  `attemptResult` are walked arm by arm, and macro-enforced exhaustiveness is explained. The
  `started -> backingOff` row is worked by hand under the strict rule (`:352-360`). Section 14
  says plainly that the Case and realization layer is not in the sample.
- **Section 17 needs a refresh.** It says no `cmp/.eval/elixir.md` exists and lists only the
  author's own points (`:683-684`). The outline asks for bullets drawn from the eval. It should
  now cite at least the rebinding gap and the README error-text mismatch.

## Verdict

This sample shows Elixir's macro system at its best for the Umpire model layer. Declarations read
as configuration. Step functions stay ordinary `case` expressions. `@before_compile` can run the
whole finite table, the coverage and dead-arm check, and the strict refinement during
`mix compile`, which puts nearly the entire Lean compile-time tier at compile time. The README
does not pretend the 1.20 type checker helps more than it does: every Model-specific guarantee is
credited to the macros. The biggest reservation is that the guarantee rests on a large,
hand-written, never-compiled macro framework with two implementations of pattern matching. That
framework includes about 560 lines of AST lowering, an interpreter and an emitter. Its correctness
is only as good as the care put into `Umpire.Lower` and `Umpire.Eval`. The small gaps already
visible, the unimplemented rebinding check and one misformatted example, show what kind of
maintenance a Go team would take on.
