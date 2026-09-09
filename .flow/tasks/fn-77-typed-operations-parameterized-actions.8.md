---
satisfies: [R4, R6, R8, R9]
---
# fn-77-typed-operations-parameterized-actions.8 Lower field requirements with checked complete Case coverage

## Description
Lower field requirements with checked complete Case coverage for the referenced parent requirements.

**Size:** M
**Files:** model/Umpire/Case/**; model/Umpire/Case.lean; model/Umpire/Observation/Projection/**; model/Umpire/Property/Scoped/**; model/Temporal/System/Nexus/ImplementationLink.lean
**Touches:** [model/Umpire/Case/**, model/Umpire/Case.lean, model/Umpire/Observation/Projection/**, model/Umpire/Property/Scoped/**, model/Temporal/System/Nexus/ImplementationLink.lean]

### Approach
- Introduce checked input-field construction, result/event Observation and clause-lowering coverage linked to operation/schema/source identities; validate complete selected coverage before compiling any Case.
- Lower typed request assignments and same-step/captured operands using task7 capability; prove emitted executable field denotation against task5/6 source meaning and existing fn78 lowering.
- Separate projection/correlation correspondence conditions from independent product clauses. Missing support/correlation fails Link admission; field mismatches fail only their authored requirement.
- Reject any missing/unsupported requested mapping as a whole-Case source-owned error. Add removal/mutation controls and field-specific online/offline/model equivalence fixtures.

### Investigation targets
**Required:**
- model/Umpire/Case/Compiler.lean:42 — lowering diagnostics; line93 whole-Case assembly.
- model/Umpire/Case/Scoped.lean:69 — checked decoded-meaning Lowered evidence.
- model/Umpire/Case/ScopedProofs.lean — actual-history source correspondence.
- model/Umpire/Observation/Projection/Declaration.lean:33 — submission versus confirmed evidence.

### Quick commands
`cd model && mise exec -- lake build Umpire.Case.CompilerTests Umpire.Property.Tests.Scoped.Evidence Temporal.Feature.Nexus3.Tests`
`go test -tags test_dep ./common/testing/testpilot/internal/verification ./common/testing/testpilot`
`make umpire-check-case-runtime-conformance`

`cd model && mise exec -- lake build Umpire.Case.Tests.FieldLowering`

### Execution constraints
Read the full parent spec. Preserve existing comments and unrelated dirty source. New paths listed here are proposed owners; reuse an established equivalent before creating one. Run Lean jobs serially; fn76 lint recovery is complete. Preserve fn75 semantic import isolation and fn78 obligation/evidence authority. No operation cancellation, new dependency/toolchain, or implicit fixture promotion. Do not stage, commit, or push; the user owns commits. Capture task-local before/after trust for changed load-bearing declarations; task11 additionally compares to the original task1 substrate.

Create and wire the proposed test module Umpire.Case.Tests.FieldLowering into its existing aggregate test root; if an existing equivalent is reused, record its exact name and run it explicitly. The new-module Quick command applies after creation; baseline existing roots before editing instead of treating an absent module as an environmental failure.

## Acceptance
- [ ] Every selected input/result/event field and requested clause has inspected checked coverage; removal/unsupported mappings reject before Driver I/O.
- [ ] Actual emitted expressions/requests/results/captures have source-denotation correspondence, not only serialized round-trip evidence.
- [ ] Correlation corruption reports Link failure; authored field mutation yields its clause violation or supported admission rejection, never missing-evidence success.
- [ ] Model/online/offline field fixtures agree and old scoped/Nexus3 fixture bytes remain unchanged.

## Done summary
A checked coverage map now carries a modeled field's identity across the two boundaries a Case has,
so keyed captures and correlations lower into the portable capability and an evidence-driven model
Run admits them.

The rejection task 7 handed over had one cause: the portable contract names a retained value by the
declared evidence field the projector emits, and a model `PropertyFieldPath`'s structural
coordinates carry no declared evidence identity. `Umpire/Observation/Projection/Coverage.lean` is
that identity. One entry binds one `PropertyFieldPath` to one declared evidence field, keyed by the
field alone exactly as the portable runtime names a retained occurrence, and `check` admits a
requested map only against the projection declaration it will run under: the field must be declared
and retained, its declared scalar type must be the modeled field's own, one retained type must be
declared for it everywhere, and both directions must be injective.

The map runs in both directions. `Umpire/Case/Scoped.lean` lowers each clause's captures and its
correlation through it into `ScopedCaptureDeclaration` and `ScopedCorrelation`, and emits the exact
capture and correlation-depth ceilings the emitted capability needs; the existing decode-equality
check still binds the emitted bytes to the meaning the model declared. `Coverage.evidence` runs the
other way for `Umpire/Observation/Evaluation/Scoped.lean`: it rebuilds the admitted projection a
declared evidence value denotes at its covered coordinates, through a real checked cursor over the
smallest admitted payload that supplies exactly that scalar there and nothing else. The rebuildable
step vocabulary is field selection, optional presence, oneof selection and keyed map lookup. A
presence read, a repeated element and a cardinality reject by name, because the payload witnessing
them would have to carry sibling data no Observation reported. A request operand denotes the
selected Action's own arguments; it is coverable for lowering, not rebuildable, and replay skips it
rather than failing a Run that never reads it.

`Umpire/Case/Coverage.lean` is the whole-Case half ACT-4 names separately: a modeled input field must
be constructed by exactly one request assignment of one named instruction, at the same field path
(map-key selectors included) and with the same exact value, and every requested clause must be
lowered exactly once. `Compiler.Input` carries the request and `compile` admits it before it
assembles anything, so a missing or unsupported mapping is a whole-Case source-owned rejection
before any Driver I/O.

`Umpire.Case.Tests.FieldLowering` (wired into `UmpireTests`) drives three evaluators from one stream
of declared Observations -- the model kernel over the projections the coverage rebuilds, the
evidence adapter over the Observations themselves, and the portable interpreter over the lowered
Case -- and requires them to agree at every chunk boundary across six scenarios, including two the
correlation rejects and one where the evidence is missing. Expected answers, the decoded keyed
fragment and the constructed request value are written out in the module rather than read back from
any encoder. Corrupting the declared count a correlation reads is a link failure on both sides: the
model returns `invalidTransition` and the portable runtime "correlation rejected this operation's
step", never a product violation and never a missing-evidence success. A clause declaring neither
captures nor a correlation still emits both ceilings unset, and the Case runtime conformance
fixtures are byte-identical.

Known Gaps unchanged from task 7: the scoped evidence scalar domain is still text/natural/boolean,
and a guard-context same-step field operand is limited to request and prior-state roots by the
Property checker, so a request-rooted correlation operand remains portable-only.

Concurrent local edits this run swept in that this task did not author: the user committed
`.flow/tasks/fn-77-typed-operations-parameterized-actions.7.md` themselves as `fc2b9b1`, and
`.flow/config.json` plus the new `fn-80` spec drafts changed on disk and were included by the
catch-all staging.

stage: impl-review - ran [round 1 NEEDS_WORK (copilot/gpt-5.4) .. round 2 SHIP (copilot/gpt-5.4)]
## Evidence
- Commits: fc2b9b1569eeabe521c20a8c65c56ba787e45f17, a898ddd6aa97f7fb72bb1a6361e4c56ee4d9fee9, 5890fe08fc8a8815f887f57f944fb10da39e8ae7, e4e7a1c6f49eb697af0ee23b4792a88df7f8b6ed
- Tests: cd model && mise exec -- lake build Umpire.Case.CompilerTests Umpire.Property.Tests.Scoped.Evidence Temporal.Feature.Nexus3.Tests, cd model && mise exec -- lake build Umpire.Case.Tests.FieldLowering, cd model && mise exec -- lake build Umpire.Property.Tests.Scoped.Fields Testpilot.Tests Testpilot.Tests.Fields, go test -count=1 -tags test_dep ./common/testing/testpilot/internal/verification ./common/testing/testpilot, make umpire-check-case-runtime-conformance (conformance fixtures byte-identical), make umpire-build-model (548 targets, includes UmpireTests), make lint-model (inherited red: the same 2 findings as the base - unusedArguments on Umpire.instReprPropertyFieldProjection from task .5, simpNF on Umpire.Operation.CheckedRpc.mk.injEq from task .1; no third finding added), make lint-code GOLANGCI_LINT_FIX=false (inherited red: exactly 1,284 diagnostics, identical to the recorded baseline; no Go source changed)
- PRs: