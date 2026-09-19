# Make the Temporal Nexus Feature model easier to browse

## Overview

Reorganize the ordinary Temporal Nexus Feature model so a contributor can start from one documented facade, read the small lifecycle semantics before Umpire target machinery, and open one file for each basic operation walkthrough. The refactor preserves the existing model authority, checked interfaces, namespaces, import paths, source provenance, semantic identities, canonical artifacts, and runtime behavior.

## Goal & Context
<!-- scope: business -->

The current ordinary Nexus model presents the right concepts but places several independently understandable concerns in large files. New Temporal and Lean contributors must scroll through target construction and planning proofs to find the state machine, or through unrelated operation walkthroughs to find cancellation. The target audience is model authors and maintainers; operators and Temporal users observe no behavior change.

## Scope

- Add one documented `Temporal.Feature.Nexus` entry facade and a consistent newcomer reading order.
- Keep `Temporal.Feature.Nexus.Lifecycle` and `Temporal.Feature.Nexus.Operations` as the stable public interfaces while moving their implementation into focused child modules.
- Mirror the physical decomposition in named Lean test declarations.
- Document the advanced Caller Closure module without physically decomposing it.
- Align the model guide, architecture guide, and Umpire module-decomposition guidance with the implemented learning path.

## Architecture & Data Models
<!-- scope: technical -->

The external seam remains the current checked Nexus Feature model. Internal module direction is acyclic and follows semantic altitude:

```text
Temporal.Feature.Nexus
├── Lifecycle
│   ├── Semantics       states, events, and authoritative transition meaning
│   └── Target          Umpire values, proofs, kernel, provider, and checked target
├── Operations
│   ├── Planning        shared deterministic planning machinery
│   ├── AsyncStart      Property → Behavior → Query → run
│   ├── Cancellation    Property → Behavior → Query → run
│   └── SuccessfulCompletion
├── Observation         unchanged in this spec
└── Experimental
    ├── AutoClose       unchanged literate proof
    └── CallerClosure   unchanged implementation with a new navigation map
```

`Semantics` does not import `Target`. `Planning` owns only shared planner machinery and imports neither walkthroughs nor the `Operations` aggregate; each operation child imports that lower seam and keeps its operation-specific run beside its Property, Behavior, and Query. Facades aggregate lower modules and continue to expose the existing namespaces and values. Public authoritative-initial lemmas complete the Lifecycle seam so the System Implementation Link no longer unfolds the Feature target's initial-state representation.

## API Contracts
<!-- scope: technical -->

- Existing imports of `Temporal.Feature.Nexus.Lifecycle` and `Temporal.Feature.Nexus.Operations` continue to elaborate without consumer changes.
- Existing fully qualified declarations, types, visibility, Definition IDs, capability contracts, canonical behavior text, source locations, Behavior Fingerprints, Query values, planner results, and Artifact bytes remain unchanged.
- `Temporal.Feature.Nexus` becomes the recommended aggregate import and documentation entry point; it does not re-export Experimental modules.
- Each operation retains the authored-to-checked sequence Property → Behavior → Query → deterministic Planning and retains its existing public namespace.
- Existing test-oriented public declarations remain available in this compatibility-preserving refactor.

## Approach

1. Split Lifecycle implementation behind its stable facade and add the missing authoritative-initial proof seam used by the Implementation Link.
2. Split Operations by walkthrough and planning concern behind its stable facade, keeping each operation-specific run with its walkthrough while extracting only shared planner machinery.
3. Mirror each production split in stable test facades and replace anonymous assertions with descriptive theorem names without weakening coverage.
4. Add the Nexus entry facade and update human-facing architecture and learning documentation after the physical layout is settled.

## Quick commands

```bash
cd model && mise exec -- lake build Temporal.Feature.NexusTests Temporal.Feature.Nexus.LifecycleTests Temporal.Feature.Nexus.OperationsTests Temporal.ImplementationLinkTests.Nexus TemporalModelTests
make umpire-check-regression
make lint-model
```

## Edge Cases & Constraints
<!-- scope: technical -->

- Public `SourceLocation.path` values remain anchored to the stable Lifecycle and Operations facades even when implementation declarations move to child files. Any unexpected source, fingerprint, Query, plan, or Artifact byte delta is a failed compatibility check rather than an automatically regenerated fixture.
- All existing comments move with the declarations they explain; new module documentation is additive and cannot replace them.
- The import graph must remain within the existing Feature/System isolation rules and must not introduce child-to-facade cycles.
- The model import policy must reject any direct or transitive path from the ordinary `Temporal.Feature.Nexus` facade to `Temporal.Feature.Nexus.Experimental`, with focused direct/transitive policy tests.
- Existing unrelated worktree changes are preserved and excluded from this spec's implementation and verification accounting.
- Generated regression views remain generator-owned and unchanged.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** A newcomer can import one ordinary Nexus facade and follow its documented reading order from lifecycle semantics through the three operation walkthroughs, observation, System correspondence, and explicitly advanced Experimental material. A facade-only smoke test exercises representative Lifecycle, Operations, and Observation declarations, and the import policy rejects direct or transitive Experimental reachability. Errors: the facade importing Experimental modules, omitting an ordinary module, or creating an import cycle fails the focused build and import-policy checks.
- **R2:** Lifecycle is internally separated into semantic meaning and checked-target machinery while its existing public facade, declarations, source provenance, semantic behavior, and comments remain compatible. The public initial-state proof seam is sufficient for the System Implementation Link to avoid unfolding Lifecycle's initial-state representation. Errors: unsupported transitions changing, direct downstream representation unfolding remaining, declaration loss, or identity/provenance drift fails compatibility tests.
- **R3:** Async Start, Cancellation, Successful Completion, and their shared Planning concern are independently browsable behind the unchanged Operations facade and namespaces. `Planning` owns only shared planner machinery; every operation-specific run stays in its walkthrough so each file retains Property → Behavior → Query → Planning order. Errors: operation cross-coupling through the aggregate facade, a walkthrough split across files, missing declarations, planner-result drift, or Query/Artifact byte drift fails focused tests.
- **R4:** Lifecycle and Operations tests mirror the focused implementation modules, every existing anonymous Nexus assertion becomes a descriptively named declaration, and all current positive, negative, deterministic, compatibility, fingerprint, Query, and Artifact checks remain covered. Errors: dropped assertions, weakened negative cases, fixture churn, or unstable aggregate test imports fails the test inventory and builds.
- **R5:** All existing public imports, fully qualified names, declaration visibility, Definition IDs, metadata-sensitive values, source paths, Behavior Fingerprints, canonical Queries, planned traces, and Artifact bytes remain unchanged across the refactor, and every existing comment is preserved with its declaration. Errors: any unapproved interface, provenance, semantic, serialization, or comment-accounting delta blocks completion.
- **R6:** Human-facing model and architecture documentation consistently teaches the new Nexus facade and internal reading map, and Caller Closure gains a concise module/section guide without physical decomposition. Observation ownership/path, AutoClose structure, CallerClosure structure, generated regression views, Umpire authoring languages, and runtime behavior remain unchanged. Errors: conflicting learning paths, generated-file edits, or any out-of-scope structural/semantic change fails review.
- **R7:** Focused Nexus, Implementation Link, and Temporal model builds plus the complete Umpire regression and model-lint gates pass from the final tree, with unrelated pre-existing worktree changes untouched. Errors: any new warning, lint/import-policy failure, test failure, stale fixture, or task-owned diff outside the declared scope blocks completion.

## Boundaries
<!-- scope: business -->

- No Observation relocation, renaming, or ownership redesign.
- No physical split of Experimental AutoClose or CallerClosure.
- No new Nexus-specific authoring DSL, macro language, or compatibility alias layer.
- No public-surface narrowing, runtime behavior, Temporal execution, Evidence behavior, or generated regression change.
- No broad Umpire Target redesign; this spec consumes the helper consolidation completed by its predecessor.

## Decision Context
<!-- scope: both — conditionally substructured -->

The stable facades preserve a small external interface while focused implementation files improve locality and navigation. Family-level source provenance remains anchored to those facades because existing artifacts and predecessor compatibility tests treat it as contractual. Removing test-oriented public declarations was rejected for this refactor because the predecessor explicitly freezes the current public surface; the readability goal is achievable without API churn. Moving Observation or splitting the literate Experimental models was rejected as unrelated scope.

## Early proof point

Task `fn-39-make-the-temporal-nexus-feature-model.1` validates the core approach by splitting Lifecycle behind its existing facade while preserving downstream Implementation Link and artifact-sensitive compatibility. If it fails, re-evaluate the facade/provenance seam before starting the Operations or test-layout tasks.

## References

- Umpire 4 development rules: module ownership, focused mappings, small interfaces, clear seams, stable IDs, and artifact identity.
- Lean authoring guidelines: deep modules, approachable module documentation, semantic proofs, comment preservation, and complete Lean verification.
- Nexus lifecycle cleanup design: ordinary versus Experimental learning surface and the three walkthrough sequence.
- Umpire component decomposition: Model, Target, Properties, Scenarios, and Examples responsibilities.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
|-----|-------------|---------|-------------------|
| R1 | Ordinary Nexus facade and newcomer reading order | fn-39-make-the-temporal-nexus-feature-model.5 | — |
| R2 | Lifecycle semantic/target split and proof seam | fn-39-make-the-temporal-nexus-feature-model.1, fn-39-make-the-temporal-nexus-feature-model.3 | — |
| R3 | Per-operation and planning split | fn-39-make-the-temporal-nexus-feature-model.2, fn-39-make-the-temporal-nexus-feature-model.4 | — |
| R4 | Mirrored, named, coverage-preserving tests | fn-39-make-the-temporal-nexus-feature-model.3, fn-39-make-the-temporal-nexus-feature-model.4 | — |
| R5 | Complete public/provenance/artifact/comment compatibility | fn-39-make-the-temporal-nexus-feature-model.1, fn-39-make-the-temporal-nexus-feature-model.2, fn-39-make-the-temporal-nexus-feature-model.3, fn-39-make-the-temporal-nexus-feature-model.4 | — |
| R6 | Consistent documentation and protected non-goals | fn-39-make-the-temporal-nexus-feature-model.5 | — |
| R7 | Focused and full verification with dirty-tree isolation | fn-39-make-the-temporal-nexus-feature-model.1, fn-39-make-the-temporal-nexus-feature-model.2, fn-39-make-the-temporal-nexus-feature-model.3, fn-39-make-the-temporal-nexus-feature-model.4, fn-39-make-the-temporal-nexus-feature-model.5 | — |
