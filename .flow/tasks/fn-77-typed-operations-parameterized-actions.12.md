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
- [ ] Identity reaches every meaning-bearing component of the generated operation through a stated digest, and no component is dropped from the fold. A bounded key cannot be injective over an unbounded schema space, so exact injectivity is NOT required; what is required is that the digest's contract is written down — which components are absorbed, at what width, and the argument for why distinct operations are not expected to collide — and that a theorem or explicit statement in the source records it.
- [ ] The identity remains provenance-bearing: a proto revision that preserves method and message names but changes the schema closure must still change the key. Naming `checkRpc`'s binding proof as the sole provenance carrier is insufficient, because it establishes only that a schema is the owner's current selection, not what an artifact was built against.
- [ ] `make umpire-gen-case-runtime-conformance` is re-run and the encoding change is recorded as a named migration under R8; regenerated fixture bytes are accounted for (zero changed bytes is an acceptable outcome if these canonicals provably do not reach committed fixtures).
- [ ] No new axiom dependencies beyond the inherited `propext` / `Classical.choice` / `Quot.sound` boundary, frozen by `#print axioms` where the spec already does so.
## Done summary
Bounded the parameterized-operation identity encoding: `Canonical.rpcSchema` now names the selected
operation (method full name, both payload signature roots, streaming shape) and reaches the
descriptor closure through `Canonical.closure`, a 256-bit structural fold, under the version bump
`parameterized-v2-` (migration `parameterized-operation-identity-v2`). Every gate is green and no
pre-existing assertion was changed.

Measured, StartWorkflowExecution / getWorkflowExecutionHistory:
- `Canonical.key`: 27,718,530 chars @ 5,023 ms -> 4,398 chars @ 30 ms; 26,209,332 @ 4,736 ms ->
  4,702 chars @ 21 ms.
- `ParameterDomain.check` over two real generated samples: 51,745 ms -> 266 ms.
- `ParameterDomain.canonical`, the string `checkTarget` feeds `behaviorFingerprintOf`:
  83,163,420 chars @ 32,114 ms -> 21,024 chars @ 159 ms.

### Resolution of the SPEC_UNCLEAR escalation

Rounds 1-3 held NEEDS_WORK against an acceptance criterion the conductor had authored wrongly: AC3
required that "operations differing in any meaning-bearing component still receive different keys",
which demands injectivity from a bounded key over an unbounded schema space. The reviewer was right
to refuse it and the criterion, not the code, was at fault. AC3 was rewritten to require a stated
digest contract (which components are absorbed, at what width, and the non-collision argument), and
the provenance property was split into its own criterion after the ART-02 regression below.

The one substantive defect the review caught: the conductor's preferred design - identify a schema
by `fullName` plus streaming flags and lean on `checkRpc`'s proof that `schema = owner.schema
reference` - was shipped first and is wrong. That proof establishes only that a schema is the
owner's current selection, not what an artifact was built against, so a proto revision preserving
method and message names left every target/property fingerprint unchanged. The closure digest fixes
it; `Canonical.rpcSchema_inj` states what is retained.

Reviewer independence is degraded and the receipt shows it. Rounds 1-3 ran cross-family on
copilot/gpt-5.4; that bridge hit zero monthly credits mid-task and codex is out until 2026-09-14, so
the shipping verdict came from the same-family `claude` backend, pinned to `claude-fable-5-1` rather
than the implementing model. A cross-family re-review is worth running once either bridge has budget.

Not green, recorded as a skip: `lint-code` did not run. Its `go vet ./...` is the whole-repo Go build
the run prohibits under disk pressure, and the diff has no Go paths.

stage: impl-review - ran [rounds 1-3 copilot/gpt-5.4 NEEDS_WORK; acceptance corrected; round 4 SHIP] (model: claude-fable-5-1)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 1192690835515737ab0cb46379f0f57f184e3abe, 97e469a3a69fc7ca2d7f60a843b8fe7e38d55b61, e4b4412e5d691d3b275ba5a403bf653da05c602d
- Tests: baseline: green via handoff (verified at 9172bf87 by fn-77-typed-operations-parameterized-actions.8), cd model && mise exec -- lake build Umpire.Operation.Tests Umpire.Property.Tests Umpire.Property.ImportTests Umpire.Query.Tests, mise exec -- make umpire-build-model, mise exec -- make umpire-gen-case-runtime-conformance, mise exec -- make umpire-check-case-runtime-conformance, mise exec -- make lint-model (2 findings == confirmed inherited baseline), #print axioms Umpire.Operation.Canonical.{closure,rpcSchema,key,rpcSchema_inj} and ParameterDomain.{check,decode_encode} == [propext, Classical.choice, Quot.sound], SKIPPED: make lint-code - diff contains no Go paths and its go vet ./... is a whole-repo Go build, prohibited by the run's disk constraint
- PRs: