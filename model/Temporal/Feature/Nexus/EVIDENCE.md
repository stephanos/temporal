# Established Nexus authoring evidence

This record consolidates the executable evidence for the established `Temporal.Feature.Nexus`
journey. `Temporal.Feature.NexusTests` imports only the public facade. The separately named owner
and migration tests below remain independent so a facade smoke check cannot replace detailed
language diagnostics.

## Compiled journey and compatibility

| Boundary | Executable evidence |
| --- | --- |
| Finite Target | `Umpire.ModelTests.FiniteMachine` compares old/new complete authored values, preserves all author proofs, and compiles missing-proof specimens. `Temporal.Feature.Nexus.LifecycleTests` compares the migrated Lifecycle to independent old-shape assembly and exact provider errors. |
| Property, Behavior, Query, plan | `Temporal.Feature.Nexus.OperationsTests` preserves every ID, source, fingerprint, metadata field, selected trace, outcome, Limit, planner admission, and golden artifact byte. Its malformed clauses/references, missing capabilities, unsatisfiable Behavior, Target mismatch, invalid Limits, and omitted-proof specimens retain the owning boundary. |
| Observation | `Umpire.ObservationTests.Compilation` covers the complete typed compile-error matrix and structural cost fixture. `Temporal.Feature.Nexus.ObservationTests` compares raw and helper-built profile/mapping/checked values exactly and exercises accepted, missing, ambiguous, conflicting, over-limit, profile, ordering, closure, and causal Evidence paths. |
| Authored Known Gaps | `Umpire.Query.Tests.AuthoredKnownGaps` proves exact attachment through check, Space, Promotion, and record update. `Umpire.SearchTests.Artifacts` proves exact union, overlap, conflict precedence, unchanged default bytes and selected plans. `Umpire.Case.CompilerTests` proves exact ordered conversion into generated Case provenance bytes. |
| Public facade | `Temporal.Feature.NexusTests` follows Target → Property → Behavior → Query → plan → Observation, publishes a checked authored gap through a real selected artifact, evaluates empty Evidence fail-closed, and runs representative malformed ID/reference, missing proof, incomplete raw Target, invalid transition, and invalid Observation specimens. |

No established production ID, source, metadata, fingerprint, provider choice, selected trace/plan,
outcome, default-empty artifact byte, warning, or diagnostic changed. The facade walkthrough's
test-only `authoredQuery` retains the established Query ID and Behavior Fingerprint; its selected
artifact differs only through the explicitly authored Known Gap and the checksums that cover that
gap. The gap states that the synthetic Evidence example makes no live-system claim. There is no
production fallback to empty data and no claim that omitted gaps were inferred.

The residual requirements are covered without inheriting Nexus2's prototype exceptions: R1 is the
compiled established facade journey and its runnable failures; R2 is the proof-carrying finite
Target seam and Lifecycle migration; R4 is Temporal-rooted identity/source/Limit authoring; R5 is
the three constructor-backed operation migrations; R6 is typed Observation construction and
evaluation; R7 is checked Query attachment, conflict-reporting publication, and exact Case rows;
R8 is the independent old-shape, identity, trace, outcome, artifact, source, and diagnostic
compatibility evidence; R9 is the public guide, import boundaries, trust inventory, and gates. R3
was fully covered by fn-65 and is intentionally absent from the residual plan.

## Trust inventory

The facade test prints the transitive axiom sets of the named load-bearing declarations
`Lifecycle.targetAuthoring`, `AsyncStart.run`, `Observation.checkedPlan`, and its test-only
`authoredRun`. `SearchView.ofCheckedQuery_isSome` remains separately printed by
`OperationsTests`; it uses only `propext`, `Classical.choice`, and `Quot.sound`. The established
Lifecycle Model and checked Property/Behavior/Query/Observation values retain their historical
native witnesses. Task 7 adds no production `native_decide`, `axiom`, `implemented_by`, `sorry`, or
`admit`. Its `authoredGapSet_isSome` is a test-only extraction witness and is disclosed by the
`authoredRun` printout.

The follow-up quality pass prints
`Temporal.Feature.Nexus.Operations.lifecycleIncrementalKernelResult_isSome` transitively. It has
the established `propext`, `Classical.choice`, `Quot.sound`, Lifecycle step-result native witnesses,
and checked Target native witness; it adds no native, compiler, custom axiom, or placeholder. The
helper takes explicit Query Target equality, completeness evidence, and canonical Lifecycle action
evidence, then applies the existing `SearchView.ofCheckedQuery_isSome` theorem. Each
operation still calls `SearchView.ofCheckedQuery` directly.

## Structural cost audit

| Added seam | Calls and traversals | 1×/10× pass condition |
| --- | --- | --- |
| Finite Target assembly | `modelSpec` calls `machineAvailability` → `kernel` and projects `setups`; `draftModel` also calls `DraftModel.make` and `authoredPlanning` → `machineAvailability`, `kernel`, and `planning`. All are record assembly/projection: zero added traversal, normalization, validation, nested scan, or checker call. | The independent fixtures contain exactly 1 and 10 assemblies. Existing `checkModel` work is excluded. |
| Temporal identity/source/Limits | Each declaration adds one `Temporal.Shared.definitionFamily`, one `DefinitionFamily.id`, one `sourceLocation`, and one `Limits.bounded` record assembly. No registry or declaration scan is introduced. | The fixtures contain exactly 1 and 10 independent identities; language checker work is excluded. |
| Observation construction | `ObservationKindSpec.declaration` maps fields once; `ObservationProfileSpec.declaration` maps kinds once and calls that helper per kind; a rule projects one field; `ObservationMappingSpec.declaration` maps dispositions once. `check` and `checked` each delegate to one existing checker call and add no normalization or rescan. | The fixtures contain exactly 1 and 10 independent profile/rule/mapping constructions, so wrapper work is at most 10×; unchanged checker work is excluded. |
| Query Known Gap attachment | Query, Space, Promotion, and record-update paths copy the checked set directly with zero validation or traversal. Existing identifier validation scans rows, canonical validation scans adjacent rows, and canonicalization/union retain `mergeSort`/`eraseDups`; no linear claim is made for those unchanged algorithms. | Ten independent attachments add exactly ten field copies; set work is not attributed to attachment. |
| Planning and Case publication | `composeSearchKnownGaps` performs the only new production `KnownGapSet.union`, once before traversal for each planning or direct-artifact request; no row/candidate loop calls it. Case publication performs one `KnownGapSet.toList.map KnownGap.toCaseKnownGap` and no second conversion. | Ten independent equal-sized requests perform exactly ten unions and at most 10× new orchestration work; each Case performs one conversion pass. Existing merge/sort/deduplicate complexity is disclosed separately. |
| Shared Nexus admission proof | `lifecycleIncrementalKernelResult_isSome` is proof-only: it applies the existing semantic admission theorem with Lifecycle completeness and canonicality lemmas. It adds zero runtime calls, traversals, conversions, checker passes, or fallback paths. | Runtime work at 1× and 10× is unchanged because Lean erases the proof; each operation retains its one direct planner-admission call. |

These are structural pass conditions, not timing claims or cached admission measurements. The
separate Nexus2 prototype measured only bounded experimental cases and still does not establish
editor responsiveness, cold/repeated elaboration, human readability, product-owner usability, or
approval of a broader syntax.

## Verification gates

The full gates ran serially for the original task-7 implementation. After the quality audit reopened
the task, the changed Target tests and proof modules received focused compilation, named transitive
trust inspection, and the required lint checks. The proof refactor and restored tests do not alter
runtime behavior, so the already-green full regression and model-build gates were not repeated.

| Command | Terminal result | Captured output |
| --- | --- | --- |
| Focused spec Quick command | Exit 0; 130 jobs | `/tmp/fn62-task7-baseline.log` |
| `lake build Temporal.Feature.NexusTests Umpire.Query.Tests Umpire.Search.Tests Temporal.Feature.Nexus.ObservationTests Temporal.Feature.Nexus.OperationsTests` | Exit 0; 86 jobs | `/tmp/fn62-task7-focused-final.log` |
| `(cd model && mise exec -- lake build UmpireTests TemporalModelTests TemporalExperimentalTests)` | Exit 0; 254 jobs | `/tmp/fn62-task7-final-aggregate.log` |
| `make umpire-build-model` | Exit 0; 329 jobs | `/tmp/fn62-task7-final-umpire-build-model.log` |
| `TMPDIR=/private/tmp/fn62-task7-regression.a6A6VH make umpire-check-regression` | Exit 0; tagged Go packages, exact inherited live identities, and 324-job Lean build passed | `/tmp/fn62-task7-final-regression.log` |
| `make lint-model` | Exit 0; 260 targets and complete import graph | `/tmp/fn62-task7-final-lint-model.log` |
| `make lint-code GOLANGCI_LINT_FIX=false` | Inherited exit 2; exactly 1,316 sorted diagnostic headers, byte-identical to task 8, SHA-256 `aee7770bec1fe01dab8826427cc89e9ffa7e764fbac25ce6b68bf5f2e3c0b077` | `/tmp/fn62-task7-final-lint-code.log` |

Follow-up quality verification:

| Command | Terminal result | Captured output |
| --- | --- | --- |
| `lake build Umpire.ModelTests Temporal.Feature.Nexus.OperationsTests Temporal.Feature.NexusTests` | Exit 0; 80 jobs; includes named transitive trust output | `/tmp/fn62-task7-quality-focused.log` |
| `lake build Temporal.Feature.Nexus.Operations.Search Temporal.Feature.NexusTests` | Exit 0; 59 jobs after the theorem documentation addition | `/tmp/fn62-task7-quality-doc-focused.log` |
| `make lint-model` | Exit 0; 260 targets and complete import graph | `/tmp/fn62-task7-quality-lint-model-rc.log` and `/tmp/fn62-task7-quality-lint-model.rc` |
| `make lint-code GOLANGCI_LINT_FIX=false` | Inherited exit 2; exactly 1,316 sorted diagnostic headers, byte-identical to the prior task-7 signature, SHA-256 `aee7770bec1fe01dab8826427cc89e9ffa7e764fbac25ce6b68bf5f2e3c0b077` | `/tmp/fn62-task7-quality-lint-code.log` and `/tmp/fn62-task7-quality-diagnostic-headers.txt` |

The quality pass also restores the two original finite-machine behavioral examples and their exact
comments beside the newer missing-proof elaboration guards. The former assert undeclared-initial-state
closure and unreachable-advertised-action obligations; the latter assert that authors cannot omit
the corresponding proof fields.

Whole-spec completion review remains required before closing fn-62.
