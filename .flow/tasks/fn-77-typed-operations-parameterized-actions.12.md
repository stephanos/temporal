---
satisfies: [R1, R8, R9]
---
# fn-77-typed-operations-parameterized-actions.12 Bound the generated operation identity encoding

## Description
Bound the payload-key identity of a generated operation so qualification tasks can run.

**Size:** M
**Files:** model/Umpire/Operation/Canonical.lean; model/Umpire/Operation/Action.lean; model/Umpire/Property/Evaluation.lean; model/Umpire/Operation/Tests*; model/Umpire/Property/Tests*
**Touches:** [model/Umpire/Operation/Canonical.lean, model/Umpire/Operation/Action.lean, model/Umpire/Property/Evaluation.lean, model/Umpire/Operation/Tests*, model/Umpire/Property/Tests*]

### Why this task exists
Task .9 escalated `BLOCKED: DESIGN_CONFLICT` after measuring the identity encoding delivered by
tasks .1/.4/.5 against the real `Temporal.API` generated declarations. `Canonical.rpcSchema`
encodes the complete request and response `Schema`, and every `SchemaNode` carries `descriptor`
and `fileContext`; `Canonical.key` then renders each encoded byte as a decimal numeral joined by
`_`, roughly four characters per byte. The result is a payload key of 27,718,530 characters for
`StartWorkflowExecution` (26,209,332 for `getWorkflowExecutionHistory`), about 6.6 s per call.
Setting `schemaInputs := []` still leaves 5,914,132, so dropping the shared input closure does not
rescue it.

Because `PropertyFieldValue.ofCursor` (`Umpire/Property/Evaluation.lean:28`) and
`ActionInstance.canonical` (`Umpire/Operation/Action.lean:47`) each build a `ModelValue` whose key
*is* that string, a Target's whole vocabulary carries megabyte keys. `ParameterDomain.check` with
two real samples took 43 s (quadratic in sample count through its `exactIdentity`/`Nodup`
obligations), and `ParameterDomain.checkTarget` feeds a ~55 MB `domain.canonical` to
`behaviorFingerprintOf`, whose pure-Lean SHA-256 runs at ~4 s/MB. `native_decide` does not help and
nothing is cached: three `native_decide` theorems over one domain took 128 s against 135 s for three
`#eval`s, so every `#guard`, the fixture renderer, and each `umpire-check-case-runtime-conformance`
render pays it again.

Tasks .1-.8 are unaffected and stay valid: their synthetic owners have single-node schemas of a few
dozen bytes, which is exactly why an O(schema bytes) identity never surfaced.

### Approach
Bound the identity before any qualification task resumes. Two candidate designs, to be chosen on
evidence rather than assumed:
- Identify a schema inside a payload key by `fullName` plus the two streaming flags, and lean on
  `checkRpc`'s existing exact-schema admission (`schema = owner.schema reference`, proven at
  binding) to carry provenance. Preferred if it holds: it stops re-encoding what the checked
  binding already proves.
- A per-method schema digest emitted by the API generator. Note `Umpire.Fingerprint.sha256Hex` is
  pure Lean at ~4 s/MB, so a digest must not be computed over the megabyte payload inside Lean.

Do not weaken what the identity distinguishes: two operations that differ in any meaning-bearing
component must still get different keys. State explicitly which distinctions the chosen encoding
preserves and by what argument.

### Migration
No committed artifact carries a `parameterized-v1-` key today, so the encoding is still cheap to
change. Field-operand canonicals do reach property behavior fingerprints, so re-run
`make umpire-gen-case-runtime-conformance` and record the change as a named migration under R8.

### Quick commands
`cd model && mise exec -- lake build Umpire.Operation.Tests Umpire.Property.Tests Umpire.Property.ImportTests Umpire.Query.Tests`
`mise exec -- make umpire-build-model`
`mise exec -- make umpire-check-case-runtime-conformance`

## Acceptance
- [ ] `Canonical.key` over a real generated `Temporal.API` operation (`StartWorkflowExecution` and `getWorkflowExecutionHistory`) is bounded and fast: report the measured character length and per-call time for both, against the recorded 27,718,530 / 26,209,332 chars at ~6.6 s baseline.
- [ ] `ParameterDomain.check` over a Target with two real generated samples completes in a time recorded in the evidence, against the 43 s baseline; `checkTarget`'s `behaviorFingerprintOf` input is no longer megabyte-scale.
- [ ] Identity is not weakened: operations differing in any meaning-bearing component still receive different keys, with the preserved distinctions and the argument for them stated explicitly.
- [ ] `make umpire-gen-case-runtime-conformance` is re-run and the encoding change is recorded as a named migration under R8; regenerated fixture bytes are accounted for.
- [ ] No new axiom dependencies beyond the inherited `propext` / `Classical.choice` / `Quot.sound` boundary, frozen by `#print axioms` where the spec already does so.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
