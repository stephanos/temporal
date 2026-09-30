# TypeScript

`nexus_caller.ts` and `standalone_activity.ts` are the two Models of `../SPEC.md`, `worker.ts` is
the Worker module both compositions need, `umpire.ts` is the framework surface, and `pins.test.ts`
holds the pins. Nothing here has been compiled or run; the files are written against TypeScript
5.5+ (`const` type parameters, `NoInfer`, `satisfies`) and vitest. Model 2 follows the revised spec
of 2026-09-29: `pauseRequested` maps to `started`, a retryable failure is a visible product row, and
the refinement rule matches by mapped states under any product action class.

## DSL mechanism

Object literals handed to builders whose generic parameters are `const`, so they capture the
literal types of what was written, and each later declaration is typed by the earlier ones:

- `enumOf([...])` and `struct({...})` enumerate a domain at runtime; the type is read back with
  `Member<typeof X>`, so `ProtocolState` is finite by construction and has one source of truth.
- A constructor with fields (`handlerError(retryable)`) is a discriminated union on `kind`, listed
  one class per assignment in `domain<Reply>()([...])`; step functions `switch` on `kind` with
  `assertNever` in `default`.
- A fact with a payload is a template-literal string, `` `nexusOperationTimedOut:${TimeoutType}` ``,
  so `facts.includes(...)`, `toEqual` and JSON need no helper; `FactName<F>` strips the payload for
  the `evidence:` keys.
- `defineMachine({...})` derives `S`, `O`, `F` from `state:` and the step functions' return types
  (with `NoInfer` where a position would otherwise compete), so the `steps:` block must have exactly
  one key per declared action and timer, and `starts:`/`ends:` must be phases of the state.
- `scenario`, `property`, `query` take the machine as a type parameter, so a Scenario's actions are
  `ClassedAction<M>`, its `starts:` is `PhaseOf<M>`, and a Query's Property and Scenario must name
  the same machine (or the machine it refines).
- `compose` builds `operation.unscheduled`, `operation.handlerReply` and friends as template-literal
  types from the member names, and parses a `sync:` line's `"operation.handlerReply"` back to that
  action to type the synced action's input.

No macros, no code generation. Everything above is the ordinary type checker.

## Where checks happen

| Check | Where | Notes |
|---|---|---|
| `steps:` names only declared actions and timers, and all of them | `tsc` | TS2353 / TS2741 |
| Step function input matches the action's `input:` fields | `tsc` | |
| Non-exhaustive `switch` over an input domain or phase | `tsc` | TS2345 at `assertNever` |
| `starts:`, `ends:`, Scenario `starts:` are phases of the Model | `tsc` | |
| Scenario actions belong to the Model; classed inputs well-typed | `tsc` | |
| Query's Property and Scenario are on the same machine | `tsc` | product Property allowed via `RefinedBy<M>` |
| `refines.map` returns the refined machine's state type | `tsc` | |
| Every `kind` of a domain is listed in its enumeration | `tsc` | but not every field assignment |
| Finite table, class count, reachability, refinement rows | vitest | `machine.table`, `checkRefinement` |
| Bounded search (`find` / `verify`) | vitest | `run(query)` |
| Every party a set's Queries use is bound | not checked | would need party inference through machines |
| Canary admissibility (no silent step on the path) | not checked | runtime, not written |

A wrong Model fails `tsc` for the structural mistakes and `vitest` for the semantic ones; Lean fails
the build for both.

## Author errors

A `steps:` line naming an undeclared action:

```
nexus_caller.ts:262:5 - error TS2353: Object literal may only specify known properties, and
'awaitFinish' does not exist in type 'StepsOf<{ readonly phase: "scheduled" | "started" | ... },
"accepted" | "notFound", ProductFact, readonly [Action<"handlerReply", ...>, ...], readonly ["timeout"]>'.
```

A forgotten `case` in `protocolHandlerReplyStep`:

```
nexus_caller.ts:331:25 - error TS2345: Argument of type '{ readonly kind: "handlerError";
readonly retryable: boolean; }' is not assignable to parameter of type 'never'.
```

A Scenario naming another Model's action:

```
pins.test.ts:171:17 - error TS2322: Type '"attemptStart"' is not assignable to type
'"workerStop" | "transportFault" | "backoff" | "scheduleToClose" | "scheduleToStart" |
"startToClose" | Classed<"schedule", {...}> | Classed<"handlerReply", {...}> | Classed<"complete", {...}>'.
```

The location is right and the message is mechanical; what it lacks is the Lean elaborator's
sentence saying *why* (`a steps: line names the action its function steps on`). The expanded types
in the message run to several hundred characters for the machines here and grow with the Model.

A failed Query is a vitest assertion:

```
FAIL pins.test.ts > nexus caller: the Queries > retry finds its claim on its path
AssertionError: expected { outcome: 'notFound' } to match object { outcome: 'found' }
```

`run` returns the witness or counterexample trace alongside, which the diff prints.

## Toolchain and loop

`node`, `typescript`, `vitest`; one `package.json`, one `tsconfig.json` with `strict`. The
edit-to-feedback loop is `tsc --watch` or `vitest --watch`, both well under a second for files of
this size, and `vitest --typecheck` for the `expectTypeOf` and `@ts-expect-error` pins. The 192-
and 288-state tables are a few thousand step calls; the widest search here (`six`, budget 262144)
is at most a few seconds in V8. No build directory to speak of.

## Libraries to leverage

Verified 2026-09-29 with `gh api repos/<owner>/<repo> --jq '{pushed_at, archived, stargazers_count}'`.

| Library | Repo | Last push | Status | Replaces |
|---|---|---|---|---|
| fast-check | dubzzz/fast-check | 2026-09-29 | maintained | The driver harness for the exploratory set: `fc.commands` + `fc.modelRun` run generated command sequences against a real system with the Model as oracle, with shrinking. Not the bounded exhaustive search, which stays hand-written. |
| @bufbuild/protobuf (protobuf-es) | bufbuild/protobuf-es | 2026-09-29 | maintained | Hand-written Case JSON: generated message types for the Case format; `schema:` could hold a generated descriptor instead of a string, so a misspelled message name is a type error. |
| connect-es | connectrpc/connect-es | 2026-09-29 | maintained | Hand-rolled gRPC to the Temporal frontend for realizations that call `StartActivityExecution`, `Describe...`, `Poll...`. |
| Temporal TypeScript SDK | temporalio/sdk-typescript | 2026-09-29 | maintained | A second driven SDK for the `caller` and `worker` parties (workflow scheduling a Nexus operation, an activity worker), next to Go. |
| ts-pattern | gvergnaud/ts-pattern | 2026-09-11 | maintained | `switch` + `assertNever`: `match(result).with({ kind: "failed", retryable: true }, ...).exhaustive()` matches nested payloads in one arm. Optional; the samples stay on native `switch`. |
| zod | colinhacks/zod | 2026-09-29 | maintained | Boundary validation only: decoding Case fixtures and Describe responses read back from JSON. Marginal here, since Model inputs come from the enumerations. |
| arktype | arktypeio/arktype | 2026-09-29 | maintained | Same role as zod with a type-syntax DSL; pick one, not both. |
| canonicalize (RFC 8785 JCS) | erdtman/canonicalize | 2026-09-18 | maintained | Hand-written key-sorted `JSON.stringify` for fingerprints and fixtures. |
| json-canonicalization (reference JCS) | cyberphone/json-canonicalization | 2024-12-13 | not maintained, no-go | Use `canonicalize` above. |
| vitest | vitest-dev/vitest | 2026-09-29 | maintained | The test runner; bundles `expectTypeOf` (mmkal/expect-type, pushed 2026-09-29) for the type-level pins. |

## Easy and awkward

Easy: the declaration blocks are the most approachable of the eight, close to the Lean surface with
nothing exotic in a Model file; the type of every declaration is checked against the ones before it
without a macro or a generator; template-literal types make `operation.unscheduled` and
`nexusOperationTimedOut:scheduleToStart` typed strings, not stringly typed ones; `tsc --watch`
answers in well under a second; and the Temporal TypeScript SDK gives a second driven SDK in the
same language as the Model.

Awkward, honestly:

- **No runtime sum types.** A union is erased. `domain<Reply>()([...])` has to list the classes by
  hand, and the compiler can check every `kind` appears but not every `retryable` assignment; that
  is a runtime pin. Every enumeration is an `as const` array and its type a `typeof`, twice per
  domain.
- **Type-level programming is its own maintenance burden.** `StepsOf`, `ClassedAction`,
  `MemberAction`, `Resolve` and `Fin<N>` are the DSL. They are about 150 lines here and they
  are what a reader has to understand when an error message expands them. `Fin<N>` by tuple length
  is a trick with a depth limit; `saturatingSucc` needs a cast because arithmetic widens to
  `number`; no partial type-argument inference means curried builders (`domain<T>()(...)`) and
  `NoInfer` annotations where inference sites compete. Machine `refines` needs an intersection type
  on the return value to stay precise.
- **JavaScript runtime semantics.** `===` on objects is identity, so state equality goes through a
  canonical key (`same(ProtocolState, a, b)`) and vitest's `toEqual`. Nothing is immutable unless
  `readonly` says so, and `readonly` is compile-time only.
- **Canonical artifacts.** Object key order is insertion order for string keys, so `domain`'s key
  and `struct`'s key depend on the literal being written in the declared order (`kind` first); the
  fingerprint would have to sort keys or use JCS. Numbers are IEEE doubles with no integer type,
  which is fine for `attempts: 0 | 1 | 2` and a problem for anything past 2^53 or for `-0`.
- **Names twice.** A `const` does not know its own binding name, so every declaration carries a
  `name:` field that must match the identifier for Case fixtures to be named after it; Lean gets
  that from the command.
- **The compiler checks shape, not meaning.** Whether the refinement holds, whether a Query is
  found, whether every party is bound: none of that is a type error. It is a test, and it runs only
  when someone runs it.
