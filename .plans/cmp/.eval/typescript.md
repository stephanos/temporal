# Review: cmp/typescript

## Scores (1-5, 5 best)

1. **Spec fidelity: 5.** Both Models complete, all names match SPEC.md, Lean section order kept via `// authoring:` markers, revised Model 2 applied (`pauseRequested -> started` at `standalone_activity.ts:494-496`, visible retry rows at `standalone_activity.ts:211-215`), all pins present in `pins.test.ts`; only deviations are `pausedIsNotDispatched` placed second instead of eighth (`standalone_activity.ts:562`) and `workerStop` declared in `worker.ts:33` and re-exported rather than declared in the Model.
2. **Language plausibility: 4.** Expert-grade TypeScript (const generics, `NoInfer`, `satisfies`, curried `domain<T>()`, deliberate method-bivariance for `Finite.key` at `umpire.ts:19-25`), but `checkRefinement`'s parameter type `Refinement<unknown, unknown>` (`umpire.ts:501`) cannot accept `Refinement<ProtocolState, ProductState>` under `strictFunctionTypes` because `map` is a property-typed function (`umpire.ts:227`), so `pins.test.ts:94` and `:150` would not compile as written.
3. **Authoring readability: 3.** The `machine`/`scenario`/`query`/`set` blocks read as configuration, but every domain is declared twice (`nexus_caller.ts:64-65`), every declaration repeats its name in a `name:` field, classed actions are verbose (`handlerReply.of({ reply: { kind: "syncSuccess" } })` versus Lean's `handlerReply (syncSuccess)`), and the step functions are plainly code.
4. **Check story accuracy: 3.** The table (`README.md:37-50`) is candid about what is not checked, but two described compile-time behaviors are doubtful: `O`/`F` inference "from the step functions' return types" (`README.md:23`, `umpire.ts:326-327`) goes through `as`-remapped mapped types over a generic constraint, which TypeScript does not infer from, so `O` and `F` most likely fall back to `string` and `evidence:` keys go unchecked; and the `RefinedBy<M>` route for `terminalHolds` (`README.md:44`) depends on `infer S2` against an intersection `Refinement<S, unknown> & Refinement<S, S2>` produced by `umpire.ts:264` plus `:339`.
5. **Framework realism: 4.** Finite enumeration is real code, not a sketch (`struct` at `umpire.ts:82-92`, `classesOf` at `:351-361`, `upTo` at `:64-67`); table derivation, BFS search, refinement walk and compose product are elided but each carries an algorithm comment matching the spec's revised rule (`umpire.ts:364-369`, `:481-484`, `:502-503`, `:607-609`).
6. **README honesty and library table: 5.** Costs are stated plainly in "Awkward, honestly" (`README.md:127-152`); all four spot-checked repos match the table exactly (fast-check pushed 2026-09-29, canonicalize 2026-09-18, ts-pattern 2026-09-11, json-canonicalization 2024-12-13 and correctly marked no-go). Only blemish is the unsubstantiated "most approachable of the eight" (`README.md:120`).

## Verbatim snippets

**`handlerReplyStep`** (`nexus_caller.ts:170-188`):
```ts
export function handlerReplyStep(state: ProductState, { reply }: Inputs<typeof handlerReply.input>): ProductStep[] {
  if (state.phase !== "scheduled") return [];
  switch (reply.kind) {
    case "syncSuccess":
      return productStep("succeeded", "nexusOperationCompleted");
    case "async":
      return productStep("started", "nexusOperationStarted");
    case "operationFailed":
      return productStep("failed", "nexusOperationFailed");
    case "operationCanceled":
      return productStep("canceled", "nexusOperationCanceled");
    case "handlerError":
      return reply.retryable ? [] : productStep("failed", "nexusOperationFailed");
    default:
      return assertNever(reply);
  }
}
```

**`syncSucceeds`** (`nexus_caller.ts:532-537`):
```ts
export const syncSucceeds = property({
  name: "syncSucceeds",
  machine: nexusProtocol,
  when: handlerReply.of({ reply: { kind: "syncSuccess" } }),
  holds: (step) => step.state.phase === "succeeded" && step.facts.includes("nexusOperationCompleted"),
});
```

**`syncReplied` and `syncCompletion`** (`nexus_caller.ts:618-625`, `:707`):
```ts
const noDeadline = schedule.of({ scheduleToClose: "unset", scheduleToStart: "unset", startToClose: "unset" });

export const syncReplied = scenario({
  name: "syncReplied",
  model: nexusProtocol,
  starts: "unscheduled",
  actions: [noDeadline, handlerReply.of({ reply: { kind: "syncSuccess" } })],
});

export const syncCompletion = query({ name: "syncCompletion", find: syncSucceeds, in: syncReplied, limits: two });
```

**`nexusCaller` compose** (`nexus_caller.ts:798-809`):
```ts
export const handlerWorker = restrict(polling, ["workerStop", "serve"]);

export const nexusCaller = compose({
  name: "nexusCaller",
  members: { operation: nexusProtocol, worker: handlerWorker },
  sync: {
    workerStop: ["operation.workerStop", "worker.workerStop"],
    handlerReply: ["operation.handlerReply", "worker.serve"],
  },
  starts: ["operation.unscheduled", "worker.polling"],
  ends: ["operation.succeeded", "operation.failed", "operation.canceled", "operation.timedOut"],
});
```

## Line counts

| File | Lines |
|---|---|
| umpire.ts | 619 |
| nexus_caller.ts | 842 |
| standalone_activity.ts | 841 |
| worker.ts | 73 |
| pins.test.ts | 231 |
| README.md | 152 |
| **Two Model files (nexus + standalone)** | **1683** |

## Red flags

- **`checkRefinement` will not typecheck.** `umpire.ts:501` demands `Refinement<unknown, unknown>`, whose `map: (state: unknown) => S2` is a property-typed function; under `strict` the protocol machine's `map: (state: ProtocolState) => ProductState` is not assignable. The README's headline vitest check (`README.md:47`) is unreachable until `map` becomes a method signature, the same fix the author applied to `Finite.key`.
- **`O` and `F` are almost certainly not inferred from `steps:`.** `StepsOf` (`umpire.ts:220-222`) is an intersection of a mapped type with an `as X["name"]` clause and one keyed by `T[number]`; TypeScript skips mapped-type inference for both shapes and resolves no properties while `A`/`T` are still type parameters. `O` and `F` fall back to `string`, so `evidence:` (`umpire.ts:320`) accepts any key and `StepOf<M>["facts"]` is `string[]`. The Models still compile, but weaker than `umpire.ts:326-327` and `README.md:23-24` describe.
- **Every machine claims a refinement.** `defineMachine` returns `& { readonly refines: Refinement<S, S2> }` with `S2 = never` by default (`umpire.ts:336-339`), intersected with the optional `refines?: Refinement<S, unknown>` at `:264`. `RefinedBy<M>` (`:284`) infers `S2` from that intersection and may resolve to `unknown`, which would reject `verify: terminalIsFinal` in `terminalHolds` (`nexus_caller.ts:729`) rather than allow it as `README.md:44` claims.
- **`run` returns a hard-coded `exhausted`** (`umpire.ts:486`), so as shipped every Query pin would fail; acceptable for a sketch but the README should say the runtime is unwritten rather than only "not compiled or run".
- **Minor spec drift:** property order in Model 2 (`standalone_activity.ts:562` before `:568`); example labels `ApplicationFailureNonRetryable` versus spec's `ApplicationFailure nonRetryable` (`standalone_activity.ts:111-112`); `workerStop` lives in `worker.ts:33`, not the Model, so the `sync:` line pairs one action object with itself (documented at `nexus_caller.ts:126-130`).
- **Comparative puffery:** "the most approachable of the eight" (`README.md:120`) is not something a single sample can establish.

No semantic drift found in step enablement: every phase guard, fact list, timer window and `productOf` arm in both Models matches SPEC.md, including `attemptResult(failed true)` from `pauseRequested -> paused` (`standalone_activity.ts:416-417`) and `requestCancel` idempotence.

## Strengths

- **Complete and faithful on the hard parts.** Both revised Model 2 rules are applied in the product step and the map, and the refinement reasoning is spelled out in comments (`standalone_activity.ts:199-205`, `:482-487`).
- **Real finite enumeration.** `struct`, `enumOf`, `upTo`, `domain` and `classesOf` are working code that produce the 192/288-state tables and the 23 action-class keys the pins count, with a documented canonical key scheme.
- **Template-literal facts.** `` `nexusOperationTimedOut:${TimeoutType}` `` (`nexus_caller.ts:312`) keeps payload-carrying facts as typed strings, so `facts.includes`, `toEqual` and JSON need no helper, and `FactName<F>` strips the payload for `evidence:`.
- **Composition typing is genuinely clever and plausible.** `MemberPhase`, `MemberAction`, `Resolve` and `QualifiedActions` (`umpire.ts:546-576`) make `operation.unscheduled` and a `sync:` line's input typed by parsing the member name back, and `pins.test.ts:189-193` pins it.
- **Type-level pins exist.** `@ts-expect-error` and `expectTypeOf` cases (`pins.test.ts:183-230`) turn the README's compile-time claims into checkable artifacts rather than prose.
- **Honest README.** The "Awkward" list names erased unions, `Fin<N>` depth limits, insertion-order keys, IEEE doubles and the names-twice tax, and the library table is fully consistent with GitHub.

## One-paragraph verdict

The sample shows TypeScript can carry the Umpire model layer with no macros or codegen: object-literal builders with `const` generics type each declaration against the ones before it, the finite enumeration is ordinary runtime code, and everything semantic (table, search, refinement) is a vitest run with an under-a-second edit loop. The Models are complete and semantically exact, and the README is the most candid cost accounting a decision maker could ask for. The single biggest reservation is that the type-level DSL is fragile exactly where it is most ambitious: the refinement typing does not compile as written because of function-parameter contravariance, and outcome/fact inference through key-remapped mapped types silently degrades to `string`, so the compile-time story is somewhat narrower than the README's table implies. These are fixable with method signatures and explicit type arguments, but they illustrate that in TypeScript the DSL's guarantees are only as good as someone actually running `tsc` on it, and the sample was never compiled.
