---
satisfies: [R5, R6, R7]
---
# fn-84-deepen-five-shallow-module-clusters-in.5 Contract rule derived from the checked field Property

## Description
Give the monitor-rule path the module the correlated path already has (R5): `Umpire.Case.Projection.lower` takes the checked field Property, the declared Observation and a realization record, and returns the Contract lowering, the coverage request and a correspondence certificate; `Compiler.compile` admits it beside the correlated lowering; the typed unary and typed Nexus Producers stop hand-writing read paths, rules, coverage requests and lowering-error helpers. Depends on the admitted-Query task because both edit the typed Producers.

**Size:** M
**Files:** `model/Umpire/Case/Projection/*.lean` (post-fn-82 home of the field-step walkers; gains `Realization`, `lower`, `Lowered`, the certificate), `model/Umpire/Case/Compiler.lean`, `model/Umpire/Case/Coverage.lean` (becomes a private walker or folds into Projection), `model/Temporal/Feature/Nexus/Success/{TypedUnary,TypedNexus}.lean`, `model/Temporal/Testpilot/CaseSupport.lean` (the shared history root), `model/Umpire/Case/Tests/{FieldLowering,ObservedPath}.lean`, `model/UmpireTests.lean`, `model/Umpire/ARCHITECTURE.md`, `model/ARCHITECTURE.md`, `model/README.md`, `tools/umpire/CONTEXT.md`, `tools/umpire/internal/retiredvocabulary/check.go` (if a scanned path moves)
**Touches:** [model/Umpire/Case/**, model/Temporal/Feature/Nexus/Success/**, model/Temporal/Testpilot/CaseSupport.lean, model/UmpireTests.lean, model/Umpire/ARCHITECTURE.md, model/ARCHITECTURE.md, model/README.md, tools/umpire/CONTEXT.md, tools/umpire/internal/retiredvocabulary/check.go]

### Approach
- Read `.plans/LEAN_GUIDELINES.md` first.
- Baseline: `make umpire-check-case-runtime-conformance` green and the checksums of every file under `tests/testcore/testpilot/testdata` and the conformance testdata recorded; ART-11 says they compare byte-for-byte and this task must not move one byte.
- Copy the shape of `Umpire.Case.Correlated.lower` (lowering record, `contractLowering` adapter, the authorized-table and window-property theorems). `lower` takes the Property, the Observation, the message root and a `Realization` record carrying what the Property does not state: the request-side literal assignments the Program makes (today the workflow-type constant in the typed unary Case and the per-operation name in the typed Nexus Case), the rule identity suffix, and the capture policy (`.none` for the two-transition safety rule, `.crossEvent` for the three-state capture rule with its captured-then-projected comparison). From those it derives the rule's read paths, comparison operator, presence structure and expected literal from the closed predicate vocabulary of the checked field Property, produces the coverage request the same Property implies, and returns a certificate that every field the rule reads is a coordinate the Property compares and every literal is one the realization assigns.
- Do the safety rule first (typed unary) and prove it byte-identical; then the capture rule (typed Nexus). If the capture rule cannot be produced byte-identically, narrow to the safety-rule shape, leave the typed Nexus rule in its Producer under a `CONSIDER(umpire)`, and say so in the summary; that is the only admitted deviation.
- The three field-step walkers differ per side today: the request-side walker accepts a keyed map step the read-side walker rejects, and the projection walker accepts a first-index step. Build one private walker with per-side exceptions from the start; list each exception with the Case that relies on it; resolve a disagreement to the stricter answer only where no checked-in Case relies on the looser one.
- `Compiler.compile` keeps its signature and admits the lowered rule beside the correlated lowering; a Property with no field atom lowers to no rule; a rule the Property does not imply rejects by name; a literal the realization does not assign rejects by name.
- Delete the two `readPathOf` copies, the three `loweringError` helpers and the duplicated history-root string; the typed Producers call `lower` and pass its outputs to the Compiler.
- Keep fn-82 R4's Case submodule set: extend `Projection`, add no `Rule` submodule. fn-83's generic Producer calls the same `lower`, so coordinate names with what fn-83 landed.
- Tests: move the observed-path tests under `lower` and assert derived rules; extend the field-lowering test with the monitor arm; add: a moved coordinate moves the rule; a dropped field is no longer read; an unimplied rule rejects; an unassigned literal rejects; each unsupported step kind rejects by name once per side.
- Docs: the architecture document's typed-field-lowering sentence (read path derived from the Property) becomes true rather than reworded; update the Case-production paragraph and the model README's and `Observed` header's copies of the same claim; add a `Derived rule` glossary entry distinct from the set-level `Contract` entry.

### Investigation targets
**Required** (pre-fn-82 lines at HEAD ebb94a44e; fn-82 .6/.7/.8 move these files):
- `model/Umpire/Case/Scoped.lean:239-375` — `Lowered`, `lower`, `contractLowering`, theorems (the pattern)
- `model/Umpire/Case/Observed.lean:78-111`, `Coverage.lean:86-89, 126-170`, `model/Umpire/Observation/Projection/Coverage.lean:35-38` — the three walkers and their differing rejection lists
- `model/Umpire/Case/Compiler.lean:93-127` — `compile` and the request-side admission
- `model/Temporal/Feature/Nexus3/TypedUnary.lean:336-357, 441-521` — the safety rule, its workflow-type literal (assigned at 443, compared at 495), coverage request and helpers
- `model/Temporal/Feature/Nexus3/TypedNexus.lean:428-505, 819-906` — the three-state capture rule, its per-operation literal, rule-ID suffixes and helpers
- `model/Umpire/Case/Tests/FieldLowering.lean:295-360`, `Tests/ObservedPath.lean`

**Optional:**
- `.flow/specs/fn-83-author-a-live-case-from-a-model-file.md` API Contracts — the generic Producer that will call `lower`
- `model/Temporal/Testpilot/CaseSupport.lean:44` — the history root declared a third time
- `.plans/UMPIRE4_SPEC.md` SEM-16, SEM-17, ART-11

### Key context
- fn-82 renames `Compiler.LoweringError` to `Compiler.Error`, moves the projection coverage walker into `Umpire/Case/Projection/`, and moves the typed Producers to `Nexus/Success/`; use the landed paths.
- Avoid retired tokens `TargetProjection`, `Projection{Record,Manifest}` in new names.
- The retired-vocabulary scan fails closed on a moved path; mirror any relocation in its registry in the same change.
- 2026-09-10: start only after fn-83 .15 is done (Flow cannot express the cross-spec task dependency). fn-85 later replaces the Producer's Program assembly; keep this task's change inside the projection lowering so fn-85 inherits it unchanged.
## Acceptance
- [ ] `Umpire.Case.Projection.lower` takes the Property, the Observation, the root and a `Realization` (literals, rule suffix, capture policy) and returns `Lowered` with `contract`, `coverage` and `certificate`; `Compiler.compile` admits it beside the correlated lowering with an unchanged signature
- [ ] the typed unary Producer holds no `readPathOf`, hand-written rule, coverage request or lowering-error helper; the typed Nexus Producer likewise, or its capture rule is left in place under a `CONSIDER(umpire)` with the narrowing reported in the summary; the history root is declared once
- [ ] one private field-step walker remains; its per-side exceptions are listed in the summary with the Case each serves
- [ ] new tests: moved coordinate moves the rule; dropped field no longer read; unimplied rule rejects by name; unassigned literal rejects by name; each unsupported step kind rejects by name once per side; a Property with no field atom lowers to no rule
- [ ] every checked-in Case fixture is byte-identical to the baseline; `make umpire-check-case-runtime-conformance` and `make umpire-check-live-tests` green with unchanged Verdicts
- [ ] focused: `lake build Umpire.Case.CompilerTests Umpire.Case.Tests.FieldLowering umpire-scoped-fixtures temporal-testpilot` and `lake build UmpireTests` green; `make lint-model` green
- [ ] architecture documents, model README, module headers and `CONTEXT.md` updated; documentation gate passes; `make umpire-check-regression` green
## Done summary
`Umpire.Case.Projection.lower` (new `model/Umpire/Case/Projection/Lowering.lean`) derives a Case's monitor rule from the checked field Property. It takes the Property, the declared Observation and a `Realization`: request literals as `Coverage.InputMapping`s, a rule suffix, and a `CapturePolicy` of `.none` or `.crossEvent selector state response`. It returns `Lowered`, which holds an optional `DerivedRule` (the shape plus its certificate), the implied `Coverage.Request` and a coverage certificate. `Lowered.contractLowering` hands the rule to `Compiler.compile` as `.monitor`, beside `.correlated`; the Compiler's signature is unchanged. Both typed Producers now call `lower`. Their `readPathOf`, `startedRule`/`operationRule`, `coverage` and `compilerError` are gone, and the history root is declared once, as `CaseSupport.historyEventNode`.

stage: impl-review - ran (claude backend, SHIP first round; receipt /tmp/impl-review-receipt-3684356e61f5-fn-84-deepen-five-shallow-module-clusters-in.5.json)

Equivalence pin (R6): the conformance and regression gates show both Case fixtures byte-identical. Every file under the two testdata trees hashes the same as the pre-edit baseline (`.flow/tmp/fn84.5-fixtures-base.sha`). No fixture was regenerated or edited.

Gates: `make umpire-check-regression` exit 0 (nine passing live identities); `make umpire-check-case-runtime-conformance` green; `make lint-model` 163, all in generated `Temporal/API/Proto.lean`, with `Umpire.Lint` clean; `make lint-code` 161 after `go clean -cache` (14505 → 161), none in touched files; `lake build` of all default targets green (591 jobs).

Decisions taken autonomously / deviations:
- No narrowing. The typed Nexus three-state capture rule is produced byte-identically by `lower`, so no `CONSIDER(umpire)` was left.
- The certificate is wider than the spec's wording. Every read coordinate is one the Property compares *or* the capture selector the realization names. The Nexus rule selects its scheduled event by `scheduledOperationPath`, which the field Property does not compare. The Link Property does, and so does the coverage map. Every literal is one the realization assigns, the selector value included.
- `CapturePolicy.crossEvent` also carries the state and response labels ("scheduled", "completion"). The capture id and transition ids cannot be derived from the Property, and byte identity needs them.
- `lower` reads the message root from the `ObservationDefinition` type instead of taking a separate `root` string. That removes one more copy of the history root.
- Rejections carry these construct names: `rule.clause-shape`, `rule.comparisons`, `rule.operator`, `rule.unimplied` (also used for a field compared in a guard), `rule.literal-unassigned`, `rule.property-literal` and `rule.observation-type`. A coordinate with no read path rejects with the walker's own reason.
- A Property with no field comparison lowers to no rule. Presence atoms (a field whose last step is `.present`, compared `.equal` with `true`) are consumed, not compared.
- The generic `Umpire.Case.Producer` (fn-83) is not changed. It lowers correlated clauses only and has no field Property to pass to `lower`.
- Post-SHIP polish commit 19d1c1ba, from the reviewer's P3s: the `rule.property-literal` name; docs now say presence atoms are consumed; the rule-level `Realization` is distinguished from `Producer.Realization` in the module header. The name was kept as the spec spells it.

One private walker: `walk` in the new `model/Umpire/Case/Projection/Coordinates.lean`, with one step-kind table, `Use.refusal`. It serves construct (`Coverage.targetPath`), read (`Projection.readPath`) and rebuild (`Projection.rebuild`, used by `Projection.Coverage`). `Umpire/Case/Observed.lean` is deleted and `Umpire.Case.Observed` added to the retired vocabulary. The module imports no generated Testpilot protocol; importing it leaked the protobuf `message` token into `Projection.Coverage` dependents. Per-side exceptions, none tightened (behavior-neutral):
- construct accepts a keyed map lookup. Serves `Umpire.Case.Tests.FieldLowering` (`tagInputCoverage`, keyed request assignment).
- rebuild accepts a keyed map lookup and the first repeated element. The first element serves the typed Nexus Case (coverage of `scheduledOperationPath` walks `.index 0`); the key and first-index rebuild also serve the FieldLowering Case (`tagPath`, `firstIndexPath`).
- read refuses nothing until the Observation's message is reached. Serves both typed Cases, whose paths walk `.index 0` before `HistoryEvent`. After that point it also refuses a keyed map lookup; no Case relies on anything looser.

Tests:
- `Umpire/Case/Tests/FieldLowering.lean` gains a monitor arm. It covers the derived safety read and literal, `notEqual`, and a moved coordinate moving the rule. A dropped comparison (presence atom only) and a Property with no field atom each lower to no rule. It also covers the capture shape and each named rejection (unassigned and Property literal, unimplied ×3, operator, comparisons, clause shape, Observation type, read-path reason), Compiler admission with a coverage mismatch rejecting, and an axiom pin on `lower`.
- `Tests/ObservedPath.lean` gains a table of each refused step kind by name once per side, plus the per-side exceptions.
- The Success tests `Tests/TypedUnary.lean` and `Tests/TypedNexus.lean` now assert the rules derived through `lower` instead of calling `readPathOf`.

Touches extensions: `model/Umpire.lean` (facade import), `model/Temporal/Feature/Nexus/Success/Tests/{TypedUnary,TypedNexus}.lean` (under Success/**), `model/Umpire/Case/Tests/ObservedPath.lean`.

Follow-ups (not done): none required. A generic Producer that authors field Properties would call `lower` the same way.
## Evidence
- Commits: ddaaf809a984464738ead753af39b15585e798f7, 19d1c1ba626a82ff3771d700a435c66fb1916a2d
- Tests: cd model && mise exec -- lake build (all default targets, 591 jobs, incl. UmpireTests, Umpire.Case.Tests.FieldLowering, Umpire.Case.Tests.ObservedPath, Temporal.Feature.Nexus.Success.Tests.TypedUnary/TypedNexus, umpire-case, umpire-correlated-fixtures), make umpire-check-case-runtime-conformance, make umpire-check-regression (exit 0, nine passing live identities), make lint-model (163, all in generated Temporal/API/Proto.lean), go clean -cache && make lint-code GOLANGCI_LINT_FIX=false (161, inherited; none in touched files), fixture sha256 of common/testing/testpilot/testdata/case-runtime-conformance and tests/testcore/testpilot/testdata identical to baseline
- PRs: