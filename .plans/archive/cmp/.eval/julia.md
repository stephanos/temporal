# Review: `cmp/julia/`

## Scores (1-5, 5 best)

1. **Spec fidelity: 4.** Both Models are complete with every enum, action, step, property, scenario, limit, query, set, composition and pin present under the spec's exact names and in the Lean section order, and the revised Model 2 is applied (`pauseRequested` maps to `started` at `standalone_activity.jl:406`; the visible retry rows at `standalone_activity.jl:166-168`). Two small deviations: the composition properties and scenarios bind to `nexusCaller.machine` / `standaloneActivity.machine` rather than the spec name (`nexus_caller.jl:764,772`, `standalone_activity.jl:752,760`), and `workerStop` is declared only in `worker.jl:47` rather than in the Nexus Model as the spec lists it (explained in the comment at `nexus_caller.jl:105-107`).

2. **Language plausibility: 4.** The macro design is genuinely idiomatic Julia (block-of-`=` DSL, `LineNumberNode` tracking, `var"for"`, `∥` as a real parser-known comparison operator, `Base.@kwdef` states, `@enumx`/`@data` domains, inner kwarg constructor for `Step`), but the sketch has real bugs that could not work even in principle: `ending`/`starting` compare enum values to `Symbol`s (`Umpire.jl:331,334`), `classof` cannot resolve dotted member actions with inputs (`Umpire.jl:544-546` versus `nexus_caller.jl:774`), and the Moshi reflection names (`variants`, `variant_fieldtypes`, `variant_fields`, `MatchError`) plus `@match` on EnumX values with `||` or-patterns are unverified.

3. **Authoring readability: 4.** The declarative commands (`@machine`, `@property`, `@scenario`, `@query`, `@set`, `@compose`) read as configuration within a few characters of the Lean; the step functions carry more ceremony than Lean because every member is module-qualified (`ProductPhase.succeeded`, `ProtocolFact.nexusOperationCompleted`) and empty results must be typed (`PStep[]`, `TStep[]`), and `var"for"` / `var"in"` and the `.T` / `.Type` split are honest warts the README names.

4. **Check story accuracy: 4.** The check table (`README.md:93-106`) is careful and mostly right: macro-time checks are real static checks, "static exhaustiveness: not at all" is honest, and "compile time" is explicitly defined as `const` evaluation during package precompile. Two soft overclaims: precompile-time `const` evaluation is ordinary execution that is cached, not analysis, and the sample's own `pins.jl:22-26` `include`s the Models into a plain module, so as written nothing is precompiled and every check runs at test time.

5. **Framework realism: 4.** Finite enumeration (`Umpire.jl:65-78`), the state table (`Umpire.jl:203-213`) and the refinement walk by mapped states (`Umpire.jl:277-297`) are fully written and match the spec's rule; search (`Umpire.jl:463-466`) and composition (`Umpire.jl:356-362`) are `error("sketched")` stubs, so every `@query` const and both `@compose` consts would throw at load.

6. **README honesty and library table: 5.** Costs are stated plainly (cold start 20-60 s, sysimage as an extra artifact, no sum types, no static exhaustiveness, Go team unfamiliarity, gRPC thin), and the library spot-checks match exactly.

Spot-checks (gh api, 2026-09-29):

| Package | README claims | gh api | Match |
| --- | --- | --- | --- |
| Moshi.jl | pushed 2026-09-27, 114 stars, maintained | 2026-09-27, 114, not archived | yes |
| EnumX.jl | pushed 2026-07-01, 117 stars | 2026-07-01, 117, not archived | yes |
| JET.jl | pushed 2026-09-29, 882 | 2026-09-29, 882 | yes |
| PropCheck.jl | 2024-03-13, no-go | 2024-03-13, 82 | yes |
| Match.jl | 2025-09-29, at the line | 2025-09-29, 274 | yes |
| SumTypes.jl | 2026-09-03, 122 | 2026-09-03, 122 | yes |
| gRPCClient.jl | 2026-09-14, 63 | 2026-09-14, 63 | yes |
| MLStyle.jl | 2025-09-09, no-go | request failed twice (network) | unverified |

## Verbatim snippets

**`handlerReplyStep` (Nexus product), `nexus_caller.jl:155-167`:**
```julia
function handlerReplyStep(state::ProductState, reply::Reply.Type)::Vector{PStep}
    state.phase == ProductPhase.scheduled || return PStep[]
    return @match reply begin
        Reply.syncSuccess       => productStep(ProductPhase.succeeded, ProductFact.nexusOperationCompleted)
        Reply.async             => productStep(ProductPhase.started, ProductFact.nexusOperationStarted)
        Reply.operationFailed   => productStep(ProductPhase.failed, ProductFact.nexusOperationFailed)
        Reply.operationCanceled => productStep(ProductPhase.canceled, ProductFact.nexusOperationCanceled)
        Reply.handlerError(true)  => PStep[]
        Reply.handlerError(false) => productStep(ProductPhase.failed, ProductFact.nexusOperationFailed)
    end
end
```

**`syncSucceeds`, `nexus_caller.jl:457-462`:**
```julia
@property syncSucceeds begin
    machine = nexusProtocol
    when = handlerReply(syncSuccess)
    holds = step -> step.state.phase == Phase.succeeded &&
        ProtocolFact.nexusOperationCompleted in step.facts
end
```

**`syncReplied` and `syncCompletion`, `nexus_caller.jl:533-537, 616-620`:**
```julia
@scenario syncReplied begin
    model = nexusProtocol
    starts = unscheduled
    actions = [schedule(unset, unset, unset), handlerReply(syncSuccess)]
end

@query syncCompletion begin
    find = syncSucceeds
    var"in" = syncReplied
    limits = two
end
```

**`nexusCaller` compose block, `nexus_caller.jl:746-759`:**
```julia
@compose nexusCaller begin
    var"for" = [operation, Worker.worker]
    state = NexusCallerState
    members = (
        operation = nexusProtocol,
        worker = handlerWorker,
    )
    sync = (
        workerStop = operation.workerStop ∥ worker.workerStop,
        handlerReply = operation.handlerReply ∥ worker.serve,
    )
    starts = [operation.unscheduled, worker.polling]
    ends = [operation.succeeded, operation.failed, operation.canceled, operation.timedOut]
end
```

## Line counts

| File | Lines |
| --- | --- |
| README.md | 241 |
| Umpire.jl | 784 |
| nexus_caller.jl | 786 |
| standalone_activity.jl | 773 |
| pins.jl | 216 |
| worker.jl | 99 |
| Total | 2899 |
| **Two Model files** | **1559** |

For reference, the Lean `Model.lean` is 777 lines including the `case` section the spec excludes.

## Red flags

- **`ending` / `starting` are broken** (`Umpire.jl:331,334`). `s.phase in phases` compares an EnumX value against a `Vector{Symbol}` built by `phases(k)` at `Umpire.jl:653`, so `ends` is always empty and `starts` throws on `first` of an empty generator. Every `@machine` const would fail at load. `start_of` at `Umpire.jl:724` does it correctly, so the fix is obvious, but as written the pins for `ends == 4` and `== 96` cannot pass.
- **Dotted member actions with inputs cannot be classed.** `class_expr` turns `operation.schedule(unset, expires, unset)` into `classof(Symbol("operation.schedule"), (...))`; that key is not in `ACTIONS`, so `classof` falls back to `timer()` with zero inputs and throws an `ArgumentError` on the length check (`Umpire.jl:544-546`). Both composition scenarios (`nexus_caller.jl:774`, `standalone_activity.jl:762`) hit this before `build_compose` is even reached.
- **Search and composition are stubs**, `Umpire.jl:465` and `Umpire.jl:360`, so all eighteen `@query` result consts and the two `@compose` consts throw. The spec permits sketched bodies, but a decision maker should know that the two things the README describes most concretely (the `SearchFailed` message at `README.md:147-150`, the composite table) do not exist in the sample.
- **The `pins.jl` "already built at precompile" story does not match its own code.** `pins.jl:22-26` defines `module TemporalModels` with three `include`s at test time; nothing is a package, so every table, refinement and query runs during `Pkg.test()`, not at precompile. The docstring at `pins.jl:20-21` concedes this. The README's per-check "when" column is therefore about a hypothetical package layout.
- **The expansion-time action registry does not survive a precompile boundary.** `Umpire.ACTIONS` is a global in `Umpire` mutated by `@action` while another package precompiles; that mutation is not persisted in `Umpire`'s cache, so after a cached load the registry is empty. Within one package this is fine, but the README's "the actions declared above are a real thing a macro can read" (`README.md:180-182`) is only true within one sequential load. Two models declaring the same action name would also silently overwrite each other, which is why `workerStop` had to move to `worker.jl`.
- **Unverified library behaviour presented as fact.** `@enumx WorkerFact` with zero members (`worker.jl:37`), the Moshi reflection functions used in `finite`/`keypart` (`Umpire.jl:69-73,101-102`), `Moshi.Match.MatchError` (`pins.jl:215`), `Reply.handlerError(retryable::Bool)` as a named-field call-form variant (`nexus_caller.jl:54`), and `@match` over EnumX values with `||` or-patterns (`nexus_caller.jl:395-402`) are all plausible but none could be confirmed without a toolchain. `finite` also calls `ctor()` on singleton variants (`Umpire.jl:72`), which is unlikely to work for a Moshi singleton value.
- **`nexusCaller.machine` leaks the framework struct** into authored text (`nexus_caller.jl:764,772`; `standalone_activity.jl:752,760`), where the spec and Lean write the bare composition name.
- MLStyle's "no-go" verdict could not be spot-checked (network failure); the other seven checks all matched.

## Strengths

- **The DSL parser is real, not hand-waved.** `keyed` (`Umpire.jl:491-509`) and the ten command macros are fully written, and the line-pinned `DSLError` story is backed by an actual test that checks the line number arithmetic (`pins.jl:191-207`).
- **The refinement checker matches the spec's rule exactly**, by mapped states rather than action names (`Umpire.jl:277-297`), and the pins exercise it with concrete row keys, including the `startToClose -> timeout` remap (`pins.jl:106-111`).
- **Model 2 semantics are faithful to the revised spec in every branch traced**, including the awkward `cancelRequested`/`pauseRequested` arms of `attemptResult` (`standalone_activity.jl:321-335`), the split `pause` behaviour (`standalone_activity.jl:346-350`), and `pausedIsNotDispatched` over the full product table.
- **The README is unusually candid**: it names both awkward spellings, admits "not at all" for static exhaustiveness, quantifies cold start, flags JET as opt-in and slow, and states the Go-team unfamiliarity cost.
- **Line-number cross-references in the README are consistent with the files** (`nexus_caller.jl:209`, `:224`, `:157`, `:616` all point where the README says they do), which suggests the sample was written with care rather than pasted.

## One-paragraph verdict

This sample shows that Julia can host the Umpire model layer with a declarative surface that reads within a few characters of the Lean, using nothing exotic: assignment-block macros, `LineNumberNode`-pinned errors, and an expansion-time registry give real static checks for undeclared actions and malformed commands, while the total table build turns missing `@match` arms into load-time failures without any static exhaustiveness. The Models themselves are complete, correctly named, and faithful to the revised Model 2. The single biggest reservation is that the "compile-time" story rests on package-precompile evaluation of `const`s, which is cached execution rather than checking, and the sample's own framework sketch would not survive that execution: `ending`/`starting` compare enums to symbols, member-qualified scenario actions cannot be classed, and search and composition are `error("sketched")`. A team choosing Julia should expect to write and maintain that runtime itself, pay tens of seconds of cold start per CI job, and carry two sum-type packages with different spellings.
