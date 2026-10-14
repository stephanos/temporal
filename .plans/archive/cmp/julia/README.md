# Julia sample

The two Models, authored in Julia against a small framework surface (`Umpire.jl`). Nothing here has
been run through a toolchain; the sample shows what the authoring would look like and where each
check lands. Package maintenance in the last section was checked with `gh api` on 2026-09-29.

Files:

- `Umpire.jl`: `Step{S,O,F}`, `Machine`, `Fin`, finite enumeration (`finite`), `Refinement`,
  `Property`, `Scenario`, `Limits`, `Query`, `Umpire.Set`, `Compose`, search, and the command
  macros `@entity`, `@action`, `@observation`, `@machine`, `@compose`, `@property`, `@scenario`,
  `@limits`, `@query`, `@set`.
- `worker.jl`: the worker entity the compositions synchronize with.
- `nexus_caller.jl`, `standalone_activity.jl`: the two Models, in the Lean file's section order.
- `pins.jl`: the pins, as `Test.@testset` blocks that read tables already built at precompile.

## The DSL mechanism

Julia macros receive Julia's own parse tree, hygienically, with a `LineNumberNode` before every
line of a `begin ... end` block. A Lean command

```lean
machine nexusProduct
  for: operation
  state: ProductState
  starts: [scheduled]
  steps:
    handlerReply: handlerReplyStep
```

is written as

```julia
@machine nexusProduct begin
    var"for" = operation
    state = ProductState
    outcome = ProductOutcome.T
    fact = ProductFact.T
    starts = [scheduled]
    steps = (handlerReply = handlerReplyStep,)
end
```

and arrives at `macro machine(name, block)` as `Expr(:block, LineNumberNode, Expr(:(=), :for,
:operation), LineNumberNode, ...)`. One shared helper, `keyed`, walks the block, tracks the line,
and rejects an unknown or repeated key at that line. There is no parser to write: `=` lines,
`(a = b, c = d)` tuples for the nested tables, `[a, b]` for lists, `a | b | c` for `cover`, and
`operation.handlerReply ∥ worker.serve` for `sync` (`∥` is an infix operator Julia's parser already
knows). `@machine` is about sixty lines.

Two choices are worth naming:

- **`=` rather than `key: value`.** Julia parses `starts: [scheduled]` as a range expression and it
  would work, but `holds: step -> ...` parses as `(holds:step) -> ...` and `cover: rows | results`
  as `(cover:rows) | results`, so every right-hand side with an operator would need parentheses.
  Assignment has the lowest precedence and swallows anything. The price is `var"for"` and
  `var"in"`: both are Julia keywords, and `var"..."` is the only way to write them as keys. That is
  the one place the authored text reads worse than the Lean.
- **Two sum-type packages.** Julia has no sum types in the base language. Flat domains (`Timeout`,
  the phases, the outcomes) are EnumX enums: `Timeout.unset`, type `Timeout.T`, `<: Enum`, so
  `instances` enumerates them and `Base.@enum`'s name-collision problem goes away (`scheduled` is a
  member of both `ProductPhase` and `Phase`). Domains with a fielded member (`Reply`,
  `ProtocolFact`, `AttemptResult`) are Moshi `@data` types: `Reply.handlerError(true)`, type
  `Reply.Type`. Both spell a member module-qualified, which is what lets a Scenario write the bare
  `handlerError(false)` and have `Umpire.resolve` look it up against the action's input type. The
  wart is real: `.T` for one, `.Type` for the other.

**What runs when.** A top-level `include` expands and evaluates one form at a time, so by the time
`@machine` expands, every `@action` above it has been evaluated into `Umpire.ACTIONS`. That gives the
macro a registry to check `steps` against at expansion time, pinned to the offending line. Everything
heavier is a `const`: `@machine` emits `const nexusProduct = Umpire.build_machine(...)`, which
enumerates the state type, builds the table, and walks the refinement; `@query` emits the `Query` and
a second `const syncCompletion_result = Umpire.run(syncCompletion)`. A `const` in a package is
computed during precompilation and cached in the `.ji`, so a rejected refinement or an unfound Query
is a precompile failure of the Model package. That is Julia's compile time, and it is the closest
analogue to Lean's elaboration. A `@generated` function was considered and rejected: generated
functions may not call arbitrary code at generation time, and top-level `const`s already do the job.

**Multiple dispatch** carries the framework rather than the Models. `finite`, `keypart`, `key` and
`factname` are one generic function each with a method per representation (Base/EnumX enums,
Moshi data, `Fin{N}`, `Bool`, plain structs), so a `Base.@kwdef struct ProtocolState` is finite,
keyed and comparable with no derive step. `Property` reads the arity of `holds` off the function
(`hasmethod`) to tell a same-step claim from a transition claim, so the author writes
`holds = step -> ...` or `holds = (before, after) -> ...` with no annotation.

**Enumeration** is `Iterators.product` over the fields' domains; EnumX values and `Fin` are isbits,
so the 288 activity states live inline in one `Vector` and the 288 × 27 table builds in
milliseconds. `Umpire.with(state; phase = ...)` is Lean's `{ state with phase }`, written once
generically over `fieldnames`.

## Where each check runs

| Check | When | How |
| --- | --- | --- |
| Unknown or repeated key in a command | macro expansion (load / precompile) | `keyed` throws `DSLError` at the key's `LineNumberNode` |
| `steps` names an undeclared action; a timer with no step | macro expansion | `@machine` reads `Umpire.ACTIONS`, filled by the `@action`s above it |
| `creates`/`on` names no entity | macro expansion | `@action` reads `Umpire.ENTITIES` |
| `refines` without `map`; exploratory set without `machine`/`cover`/`budget`; two of `find`/`verify` | macro expansion | shape checks in the macro |
| Wrong arity of `holds` | `const` evaluation (precompile) | `Property` constructor, `hasmethod` |
| Non-exhaustive `@match` | `const` evaluation (precompile) | the table build calls every step on every state × class; Moshi throws `MatchError` on the arm that is missing |
| Wrong return type of a step | `const` evaluation | `::Vector{Step{S,O,F}}` assertion in `build_table` |
| State table, ends, class count | `const` evaluation; read in `pins.jl` | `build_table`; `@test length(m.table.states) == 192` |
| Refinement | `const` evaluation | `check_refinement`, `error` if a row is rejected |
| Query find / verify | `const` evaluation | `Umpire.run` throws `SearchFailed` |
| Pins | `Pkg.test()` | `@testset` over the consts |
| Static exhaustiveness, static typing of `holds` | not at all (see JET below) | |

The refinement rule is the spec's, by mapped states: for every protocol transition `(s, a, s')`,
either `map(s) == map(s')` (a stutter) or the product has some transition from `map(s)` to `map(s')`
under any action class. Action names play no part, which is what lets a protocol timer row land on
the product's `timeout` row and a retryable failure under `cancelRequested` land on the product's
`attemptResult(canceled)` row.

## What an author's error looks like

**A step function naming an undeclared action** (a typo in `steps`). Macro-time, pinned to the
`steps = (...)` line, before any table builds:

```
ERROR: LoadError: Umpire: machine nexusProduct: `steps` names `handlerRepy`, which is neither a
declared action nor one of this machine's `timers`
  at /…/nexus_caller.jl:224
in expression starting at /…/nexus_caller.jl:209
```

**A non-exhaustive match.** Moshi's `@match` has no static exhaustiveness check; a missing arm is a
runtime error. What saves the Model is that the table build is total: every state times every class
calls the step function, so the missing arm is hit while the `const` is evaluated, during precompile,
with a stack through the step function:

```
ERROR: LoadError: MatchError: no arm matched Reply.operationFailed
Stacktrace:
 [1] handlerReplyStep(state::ProductState, reply::Reply.Type) at nexus_caller.jl:157
 [2] build_table(...) at Umpire.jl:216
 [3] build_machine(...) at Umpire.jl:316
in expression starting at nexus_caller.jl:209
```

It points at the function, not at the `@match`, and it is one arm at a time. JET does not close the
gap: Moshi's fallthrough is an explicit `throw`, which JET's error-report mode considers intended.

**A failed Query.** `const syncCompletion_result = Umpire.run(syncCompletion)` throws
`SearchFailed`, again at precompile:

```
ERROR: LoadError: query syncCompletion: no candidate of syncReplied within 2 steps / 512 searched
satisfies syncSucceeds; closest: [schedule-unset-unset-unset, handlerReply-syncSuccess] fails
`step.state.phase == Phase.succeeded`
in expression starting at nexus_caller.jl:616
```

If the team would rather have a red test than a failed `using`, the `_result` const moves into
`pins.jl` and the Model package always loads. Both are one-line changes; the sample keeps the Lean
placement.

## Toolchain and feedback loop

Julia 1.11 or 1.12, `Pkg` for the environment (`Project.toml` + `Manifest.toml`), `Test` from the
standard library. Editor: VS Code with the Julia extension (LanguageServer.jl) or any editor plus a
REPL. The realistic loop is a long-lived REPL with Revise.jl: edit a step function, `include("pins.jl")`,
answer in one to three seconds because the tables rebuild in milliseconds and nothing recompiles
beyond the changed methods. A macro edit needs the Model file re-included (Revise tracks method
edits, not macro re-expansion of existing consts).

Cold start is the honest cost. `julia --project -e 'using Pkg; Pkg.test()'` on a fresh environment
pays package precompile once (Moshi, EnumX, JET if used; a minute or two), then 20 to 60 seconds per
run for loading and the first call of everything. PrecompileTools workloads in `Umpire.jl` and a
PackageCompiler sysimage for CI bring that to a few seconds, at the cost of one more build artifact
to maintain.

## What Julia made easy, and what it made awkward

Easy:

- The block DSL. Julia's parser does the work; `keyed` is thirty lines, and every command is a
  dictionary lookup over an already-parsed block with line numbers attached. The Model files read
  within a few characters of the Lean, `var"for"` aside.
- Pinned errors. `LineNumberNode`s are in the tree; `DSLError` carries one and `showerror` prints it.
- Registries at expansion time. Sequential top-level evaluation means "the actions declared above"
  is a real thing a macro can read, with no separate compile-time store.
- Finite enumeration and record update, once, generically, by dispatch on the field types.
- Compile-time-ish checks without a separate stage: a `const` in a package is precompiled.

Awkward:

- No sum types in the language. Two packages, two spellings (`.T` / `.Type`), and equality on Moshi
  variants is structural but not guaranteed `isbits`, so the `Vector{ProtocolFact.Type}` in a step
  may allocate. LightSumTypes.jl is a third option (one concrete type, faster) with yet another
  spelling.
- No static exhaustiveness. The total table build is a strong safety net for step functions, but a
  `@match` inside a `holds` predicate is only exercised on the paths a search visits.
- Dynamic typing. `holds = step -> step.state.phas == ...` is a `FieldError` at the first evaluation,
  not a type error. JET.jl's `@report_opt`/`@test_opt` over `build_machine` catches most of these
  ahead of time and belongs in `pins.jl`, but it is opt-in and slow on first run.
- `Set` shadows `Base.Set`. Kept as `Umpire.Set`, unexported; inside `Umpire` the framework writes
  `Base.Set` where it means one.
- `using` only at module top level, and modules export nothing by default, so `pins.jl` either lists
  every name it reads or wraps each Model's pins in a module. There is no `open`.
- Latency. Time-to-first-test is tens of seconds cold; every CI job pays it unless a sysimage is
  built.
- gRPC. `gRPCClient.jl` exists and is maintained but has one maintainer and a thin user base; the
  Go Testpilot runtime is where the wire is, so this only matters if the model layer ever talks to a
  server directly.
- Not a language a Temporal Go team knows. Multiple dispatch, macros and the module system have
  their own idioms; a Go engineer will write Go-shaped Julia for a while, and the review burden lands
  on whoever knows the language.

## Libraries to leverage

Checked with `gh api repos/<owner>/<repo> --jq '{pushed_at, archived, stargazers_count}'` on
2026-09-29. "Maintained" means a push within the last twelve months and not archived.

| Package | Last push | Stars | Verdict | Replaces |
| --- | --- | --- | --- | --- |
| Moshi.jl (`Roger-luo/Moshi.jl`) | 2026-09-27 | 114 | maintained | sum types with fields and `@match` in one package; the `Reply`/`ProtocolFact` domains and every step function's match |
| EnumX.jl (`fredrikekre/EnumX.jl`) | 2026-07-01 | 117 | maintained | namespaced flat enums (`Phase.scheduled`, `<: Enum`, `instances`); avoids `Base.@enum` name collisions |
| SumTypes.jl (`MasonProtter/SumTypes.jl`) | 2026-09-03 | 122 | maintained | alternative to Moshi for the fielded domains; has its own `@cases` with an exhaustiveness *warning*. Pick one of Moshi/SumTypes, not both |
| LightSumTypes.jl (`JuliaDynamics/LightSumTypes.jl`) | 2026-09-27 | 62 | maintained | a faster, single-concrete-type sum type if `Vector{ProtocolFact.Type}` allocation shows up; no matcher of its own |
| MLStyle.jl (`thautwarm/MLStyle.jl`) | 2025-09-09 | 422 | **not maintained, no-go** | would have been the `@match`; over twelve months since the last push |
| Match.jl (`JuliaServices/Match.jl`) | 2025-09-29 | 274 | at the twelve-month line; not recommended | a `@match` over plain structs and enums; Moshi covers it |
| JET.jl (`aviatesk/JET.jl`) | 2026-09-29 | 882 | maintained | static analysis of `build_machine` and the `holds` lambdas: field typos, method errors, type instabilities. The only static-typing story Julia has |
| Supposition.jl (`Seelengrab/Supposition.jl`) | 2025-11-02 | 97 | maintained (slow cadence) | property-based tests of the framework itself: random step sequences against `stuck`, refinement invariants, `key` round-trips |
| PropCheck.jl (`Seelengrab/PropCheck.jl`) | 2024-03-13 | 82 | **not maintained, no-go** | superseded by Supposition.jl from the same author |
| ProtoBuf.jl (`JuliaIO/ProtoBuf.jl`) | 2026-06-22 | 221 | maintained | generating Julia types from the Case protobuf so the model layer could emit the Case format directly |
| gRPCClient.jl (`JuliaComputing/gRPCClient.jl`) | 2026-09-14 | 63 | maintained (thin) | a gRPC client over `ProtoBuf.jl` if the model layer must talk to a Testpilot server; small community |
| HTTP.jl (`JuliaWeb/HTTP.jl`) | 2026-09-28 | 687 | maintained | the HTTP transport `gRPCClient.jl` sits on; also enough for a REST/JSON bridge to Go |
| JSON3.jl (`quinnj/JSON3.jl`) | 2026-09-17 | 222 | maintained | reading and writing fixtures such as the exploratory coverage JSON with struct mapping |
| JSON.jl (`JuliaIO/JSON.jl`) | 2026-09-28 | 359 | maintained | the simpler alternative to JSON3 if struct mapping is not needed |
| SHA.jl (`JuliaCrypto/SHA.jl`) | 2026-07-14 | 53 | maintained (stdlib) | the Behavior Fingerprint hash over the table |
| PrecompileTools.jl (`JuliaLang/PrecompileTools.jl`) | 2026-05-19 | 252 | maintained | a precompile workload in `Umpire.jl` so `using` and the first table build are fast |
| PackageCompiler.jl (`JuliaLang/PackageCompiler.jl`) | 2026-09-24 | 1556 | maintained | a sysimage for CI to bring cold start to seconds |
| Aqua.jl (`JuliaTesting/Aqua.jl`) | 2026-09-17 | 456 | maintained | package hygiene tests: method ambiguities from the `finite`/`keypart` dispatch, undefined exports, stale deps |
| ReTest.jl (`JuliaTesting/ReTest.jl`) | 2026-09-18 | 112 | maintained | filtered re-running of `@testset`s from the REPL; nice with Revise, optional |
| JuliaFormatter.jl (`domluna/JuliaFormatter.jl`) | 2026-09-06 | 644 | maintained | formatting; a `.JuliaFormatter.toml` keeps the Model files diff-stable |

Recommended set: Moshi + EnumX for the domains, JET in the test suite, Supposition for framework
properties, ProtoBuf.jl only if the model layer emits Cases itself, PrecompileTools from day one,
PackageCompiler when CI time becomes a complaint, Aqua and JuliaFormatter as hygiene. MLStyle and
PropCheck are out.
