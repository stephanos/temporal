# Nexus race authoring prototype

## Checked learning path

Read and build these files in order:

1. `Lifecycle.lean` declares the four-state baseline as explicit finite catalogs and transition
   rows. `Cancellation.lean` supplies the three Property, Behavior, and Query journeys with named
   providers, Model Outcomes, stage-specific Limit units, and successful raw-checker admission.
2. `Race.lean` starts from an already-started operation. `requestCancel` moves to the nonterminal
   `cancelRequested` state; the later `resolve` Action has two Target-owned alternatives, canceled
   and succeeded. The bounded universal Query establishes a terminal model response within one
   semantic transition. This is finite model assurance, not an Observation, runtime delivery, or
   wall-clock claim.
3. `Authoring.lean` shows the default typed constructors for the baseline and guarded race.
   `Tests.lean` checks semantic admission, equivalence, negative cases, coverage, and conflicts.
   `AuthoringTests.lean` compares those constructors with the isolated `property%`, `scenario%`,
   and `query%` specimens. Both test roots are imported by `TemporalModelTests`.
4. `COVERAGE.md` maps every prototype requirement to executable declarations and records the
   exact residual differences from the deferred ordinary-authoring specification.

The authoring sequence is `FiniteTable.validate` / `checkModel`, then `Property.check`,
`Scenario.check`, and `Query.check`, followed by planning with
`SearchView.ofCheckedQuery`. Raw declarations remain available for negative tests.
A checked value exists only on the successful checker branch; the frontend alternatives do not
create kernel-checked constants automatically.

`Authoring.lean` keeps ordinary typed constructors as the default race-tree authoring surface. The
compiled `property%`, `scenario%`, and `query%` alternatives remain in `AuthoringTests.lean` for
comparison. All three lower to the existing Property, Behavior, and Query checkers; they add source
occurrence capture and do not add an evaluator or migrate any production declaration.

The constructor and frontend specimens use the same explicit definition-family roots, kinds, keys,
Target, capabilities, Property clauses, named Behavior occurrences, Limits, and planner policy.
Compilation checks all three baseline cases plus the cancellation race and guarded cases. The tests
compare every checked identity, canonical metadata value, behavior fingerprint, semantic field, and
the selected planner outcomes. They also compare the guarded Query's `found` result and all three
baseline planning outcomes. Moving a Behavior or Query declaration changes its checked source;
Behavior's display-oriented canonical JSON changes as well, while semantic fingerprints remain
equal. Query canonical metadata deliberately excludes source and remains equal.

## Diagnostics and admission

Closed frontend expressions run the existing checker during elaboration only to select a diagnostic.
Failures freeze the complete typed error JSON, all related definition IDs, the selected source role,
and the exact start/end compiler span. The matrix covers malformed and duplicate IDs, unknown Action,
State, and result references, wrong reference kinds, missing capabilities and Properties, Target
mismatch, resulting-state guards, unsupported roles/operators, empty Boolean/case/Query groups,
invalid and omitted units, and duplicate Query Properties. Parent, case, exception, clause, setup,
occurrence, Action, Target, Property, Behavior, Limits, and policy roles are retained where relevant.
The omitted-unit case is a Lean structure-construction diagnostic because every raw `Limit` requires
its unit; `Limits` avoids that omission by mapping its three named fields to fixed units.

Contradictory setup bindings are valid Behavior input with an explicit `unsatisfiable` space status;
the frontend test freezes that status instead of inventing an error. Contradictory reachable Property
scenarios and missing exception replacements remain findings from the existing case analysis, with
their realized state, Action, case, exception, and clause provenance. Resulting and future conditions
in a trigger or exception remain rejected by the existing Property checker.

For terms containing local variables, compiler evaluation is skipped and the frontend returns the
ordinary typed checker expression. Invalid open-term Behavior and Query fixtures evaluate to their
typed error variants and cannot produce a partial checked value. The successful-branch Query form
accepts one authored `Query` and the Model it asks it of and returns the Query checker's typed
`Except`; closed diagnostic fixtures exercise its `.error` branch.

## Conditions, exceptions, and bounded analysis

The portable Boolean vocabulary is limited to typed atoms over the declared prior state, selected
Action, resulting state, Model Outcome, and Model Facts, composed with `all`, `any`, `not`, and
`oneOf`. Guard inputs are checked before evaluation. Resulting-state and future-value conditions
are rejected where trigger-time semantics cannot supply them; arbitrary comparisons, callbacks,
and cross-field equality are unsupported.

A conditional Property can pass because its trigger never occurred. That does not establish case
coverage. Complete case groups separately report whether the parent and every named case were
exercised. An exception narrows applicability at the original trigger; an exception without a
replacement is reported and never invents behavior, and a condition first true on a later step
cannot withdraw an already-triggered temporal obligation.

All applicable clauses and Properties are conjunctive. Source order, case order, and specificity
do not choose a winner. Mutually exclusive scalar expectations are a logical conflict; expectations
that are separately satisfiable but cannot coexist on any admitted Target continuation are modeled
incompatibility. A single failed obligation is only a violation. Exhaustive absence applies solely
to the enumerated finite space and declared Limits; `limit-reached` is inconclusive.

## Admission trust experiment

`Property.checked`, `Scenario.checked`, and `Query.checked` expose explicit-proof
seams. The prototype attempted kernel success proofs through `rfl`, `decide`, and `decide +kernel`.
Property still fails for both the full baseline and a minimal closed declaration, including an
owner-local probe where private helpers were visible. The actual guarded Behavior and the explicit
successful-branch Query also fail `decide +kernel`; checked compiler fixtures retain those failures.
No attempt reported a heartbeat or recursion-depth limit, so the deeper reduction barrier remains
unidentified.

The practical frontend therefore returns and recomputes the existing successful checker branch. It
does not turn elaborator evaluation into a `CheckedProperty`, `CheckedScenario`, or `CheckedQuery`
constant. The successful-branch Query input is assembled only from existing checked constructor
results. No new native extraction, `model` default, `sorry`, `admit`, or custom axiom is used.
`#print axioms` audits the constructor families, each check and explicit-proof seam, the guarded and
baseline frontend admissions, the measured Query frontend, and `Race.targetResult`. The trust-bearing
results match the established baseline exactly: `propext`, `Classical.choice`, and `Quot.sound`.

## Measurements and editor observations

The commands below used the pinned Lean 4.33.1 toolchain under `mise` on macOS. Profiler values are
one warm run and are not performance guarantees. Declaration elaboration measures the authored
spelling: constructor declarations defer checker evaluation, while closed frontends also evaluate
the checker once to select diagnostics.

| Declaration | Constructor 1 | Constructor 10 | Frontend 1 | Frontend 10 |
| --- | ---: | ---: | ---: | ---: |
| Property | 0.000593 s | 0.002720 s | 0.013429 s | 0.119365 s |
| Behavior | 0.000970 s | 0.002705 s | 0.009243 s | 0.069988 s |
| Query | 0.000814 s | 0.004107 s | 0.052329 s | 0.496610 s |

The Property declaration measurements are the immediately preceding prototype run. Behavior and
Query declarations were measured in the same way. These values show syntax/elaboration overhead;
they are not an equal-work admission comparison and do not decide the default surface.

Admission was measured separately by compiling permanent `#guard` checks that force every named
checker result. Each constructor/frontend pair therefore performs the same checker work after its
declaration has elaborated:

| Forced admission | Constructor 1 | Constructor 10 | Frontend 1 | Frontend 10 |
| --- | ---: | ---: | ---: | ---: |
| Property | 0.011164 s | 0.012127 s | 0.011509 s | 0.012066 s |
| Behavior | 0.006103 s | 0.006932 s | 0.006074 s | 0.006810 s |
| Query | 0.050297 s | 0.050475 s | 0.049900 s | 0.050952 s |

The 10x fixtures intentionally repeat one admitted declaration, so Lean can share or cache work;
these measurements do not predict ten distinct declarations. Both measurement passes temporarily
wrapped only the named definitions or admission guards with `set_option trace.profiler true` and
`set_option trace.profiler.threshold 0`, then ran:

```sh
(cd model && mise exec -- lake build Temporal.Feature.Nexus.Race.AuthoringTests)
```

Those temporary profiler options were removed after capture. The admission guards remain compiled
tests. Cold-cache costs, repeated-run variance, and interactive latency were not measured.

Lean's bundled language server was launched with:

```sh
(cd model && mise exec -- lake env lean --server)
```

JSON-RPC `didOpen`, hover, definition, completion, and `didChange` requests produced the same matrix
for `constructorOneProperty`/`frontendOneProperty`, `constructorOneBehavior`/`frontendOneBehavior`,
and `constructorOneQuery`/`frontendOneQuery`:

| Operation | Constructor result | Frontend result |
| --- | --- | --- |
| Completion | Each partial declaration name completed to its full Property, Behavior, or Query name | Each corresponding partial frontend name completed to its full name |
| Hover | Returned the full `Except` or Query `Option (Except ...)` type for all three declarations | Returned the same corresponding result type for all three declarations |
| Navigation | Resolved all three references to their definition spans in `AuthoringTests.lean` | Resolved all three references to their definition spans in `AuthoringTests.lean` |
| Recovery | Six incomplete constructor/frontend names first produced six unknown-identifier errors | Replacing all six names in document version 2 cleared every error |

A separate owner-API completion probe returned `declaration`, `check`, `checked`, and `error?` for
both `Scenario.` and `Query.`. Compiler recovery was also observed in the fixture
module: exact negative declarations did not prevent later positive declarations and axiom audits
from elaborating. No VS Code, Neovim, or Lean editor client was installed, so rendered client UI,
completion ranking, navigation gestures, and interactive latency remain unmeasured. Human
readability, product-owner usability, and final grammar approval also remain unmeasured.

## Decision boundary

Typed constructors remain the single default because they have equal semantics, admission cost, and
trust, work uniformly for open and closed terms, and already satisfy the matched LSP matrix. The
frontend alternatives provide more precise authored spans for closed failures, but add separate
closed-expression elaboration work and have not demonstrated a human authoring advantage sufficient
to replace the constructor path.

The compiled alternatives remain test specimens under the narrow Nexus.Race AUT-07 exception. The prose
notation in `DESIGN.md` is explicitly labeled proposed, uncompiled syntax. Production adoption still
requires a recorded AUT-07 single-authoring-path reconciliation and AUT-08 remains unchanged: no
macro language belongs in `FiniteMachine`, all finite evidence remains present, and native diagnostics
cannot discharge proof obligations. A human evaluation must assess readability, error recovery in a
real editor client, and grammar preference before any production migration.
