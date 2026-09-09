---
title: Canonical identity embeds whole generated schema (27MB per payload)
date: "2026-09-09"
track: bug
category: performance
module: model/Umpire/Operation/Canonical.lean
tags: [umpire, lean, identity, canonical, generated-api, fn-77]
problem_type: performance
symptoms: ParameterDomain.check on a real Temporal method takes 43s; Target/Property fingerprints take minutes
root_cause: "Canonical.key renders the exact encoded RpcSchema bytes, so every model payload key is ~27MB on the generated Temporal API"
resolution_type: fix
related_to: [bug/performance/authoring-comparisons-must-force-equal-2026-09-06]
---

## Problem
Qualifying a generated `WorkflowService.StartWorkflowExecution` operation (fn-77 task 9) is blocked
by the identity encoding delivered in tasks 1/4/5. Every model payload identity built from a real
generated `Temporal.API` schema is a multi-megabyte string, so nothing downstream of it can be
evaluated in reasonable time.

`Umpire/Operation/Canonical.lean:61` renders identity as the exact encoded tree bytes:

```lean
def key (data : Tree) : String :=
  "parameterized-v1-" ++ String.intercalate "_" ((Encoding.encode data).map fun b => toString b.toNat)
```

`Canonical.rpcSchema` (`:56`) feeds it the *complete* `RpcSchema`: `fullName`, the request and
response descriptor closures, and `schemaInputs` -- every node carrying exact hex `descriptor` and
`fileContext` bytes.

Measured on `Temporal.Api.Workflowservice.V1.WorkflowService.startWorkflowExecution`
(62 request nodes, 159 response nodes, 148 schemaInputs nodes):

- `Canonical.key (Canonical.rpcSchema startSchema)` = **27,718,530 characters**, ~6.6 s per call.
- `getWorkflowExecutionHistory` = 26,209,332 characters.
- With `schemaInputs := []` it is still 5,914,132 characters, so dropping the shared input closure
  does not rescue it.

Both `ActionInstance.modelValue` (`Umpire/Operation/Action.lean:52`, via `.canonical` at `:47`) and
`PropertyFieldValue.ofCursor` (`Umpire/Property/Evaluation.lean:28`) build a `ModelValue` whose key
is exactly this string, for *every* payload root (request, prior/resulting state, outcome, event).
`PropertyFieldPath.canonical` (`Umpire/Property/Fields.lean:139`) and
`PropertyFieldComparison.canonical` (`:143`) embed it once per operand.

Consequences measured or derived:

- `ParameterDomain.check` with two samples: **43 s** (its `Nodup` / `exactIdentity` obligations
  compare `modelValue`s). Cost is quadratic in the sample count.
- `ParameterDomain.checkTarget` sets `canonicalBehavior := domain.canonical`, so a ~55 MB string
  reaches `behaviorFingerprintOf`. `Umpire.Fingerprint.sha256Hex` is pure Lean and measures
  **~4 s/MB**, i.e. minutes for one Target.
- `checkProperty` on a Property with one field comparison plus its presence atoms canonicalizes
  ~10 operand paths, each embedding a full schema.

`native_decide` does not help: three `native_decide` theorems over the same domain took 128 s
(~43 s each), and three `#eval`s took 135 s. Lean caches nothing between them, so every `#guard`,
every theorem, the fixture renderer and each `umpire-check-case-runtime-conformance` render pays
the full cost again.

## What Didn't Work
- Building the Target over `ActionInstance` through `ParameterDomain.checkTarget`: the domain's
  canonical string becomes the action definition's `canonicalBehavior`.
- Building the Target over plain `ModelValue`s instead: a request-rooted field operand still needs
  `PropertyFieldProjection.ofAction`, whose `modelValue` is the same multi-megabyte key, and the
  Target vocabulary keys are those values.
- Restricting the domain to one `.fixed` sample: reduces the constant, not the order.
- Dropping `schemaInputs` from the identity: still ~5.9 MB per payload.
- `native_decide` instead of `#guard`: same cost, no sharing.

## Solution
Not resolved in task 9. The identity encoding has to become bounded before any generated Temporal
operation can be qualified. Candidate directions, all owned by fn-77 rather than by a qualification
task, and all semantic/format changes needing a named migration under R8:

1. Have the API generator emit a per-method schema digest and let `Canonical.rpcSchema` reference
   that digest instead of re-encoding the descriptor closure in Lean.
2. Identify a schema inside a payload key by its `fullName` plus streaming flags only, relying on
   `checkRpc`'s existing exact-schema admission for provenance.
3. Replace `Canonical.key`'s byte rendering with a bounded digest -- but note `Fingerprint.sha256Hex`
   is pure Lean at ~4 s/MB, so it must not be applied to the megabyte payload either.

No committed artifact carries a `parameterized-v1-` key today (`grep -rl parameterized-v1` matches
only `model/Umpire/Operation/Canonical.lean`), so the encoding can still be changed cheaply. Field
operand canonicals do reach property behavior fingerprints, so any change needs the conformance
fixtures regenerated through `make umpire-gen-case-runtime-conformance`.

## Prevention
A qualification task against the real generated API should run before, not after, an identity
encoding is frozen: the synthetic owners used by tasks 1-8 have single-node schemas of a few dozen
bytes, so the encoding's O(schema bytes) identity never showed up. A cheap guard would be an
executable check that the canonical key of a real generated method schema stays under a declared
ceiling.
