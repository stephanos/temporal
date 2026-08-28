# Centralize PlannerPolicy constructors and default seed

## Overview

Replace repeated planner-policy record literals and unexplained per-model seeds with a small, documented `PlannerPolicy` constructor interface. Ordinary shortest and exhaustive policies share seed `17` and definition-ID tie-breaking; deterministic seeded traversal accepts an optional seed whose default is also `17`.

## Goal & Context

Model authors currently repeat the complete `PlannerPolicy` representation and choose local magic-number seeds even when the selected strategy does not consume them for traversal. This makes the interface noisy and lets equivalent ordinary policies acquire unrelated semantic identities. The change gives Lean authors canonical policy values while retaining explicit record updates for tests and exceptional policies.

## Scope

- Add canonical shortest, exhaustive, and seeded planner-policy constructors.
- Migrate ordinary Umpire and Temporal model callers to those constructors.
- Preserve deliberate non-default seed and generic-strategy fixtures.
- Refresh checked canonical queries, artifacts, generated views, and checksum expectations whose identity changes when seeds `23` and `29` become `17`.
- Document constructor semantics and the distinction between identity-bearing seeds and traversal-affecting seeds.

## Architecture & Data Models

The constructors live in the namespace of `PlannerPolicy`, behind the existing `Umpire.Query` facade. They construct the unchanged `PlannerPolicy` representation; query checking, canonical serialization, behavior fingerprints, artifact checksums, and planner execution continue to consume that representation exactly as before.

The work depends on `fn-17-bounded-semantic-exploration-and`, whose first task renames the Query strategy from `coverageGuided` to `seeded`. This spec exposes `PlannerPolicy.seeded`, not a compatibility alias that would continue claiming coverage guidance.

## API Contracts

- `PlannerPolicy.shortest : PlannerPolicy` selects the shortest strategy, seed `17`, and definition-ID tie-breaking.
- `PlannerPolicy.exhaustive : PlannerPolicy` selects the exhaustive strategy, seed `17`, and definition-ID tie-breaking.
- `PlannerPolicy.seeded (seed : Nat := 17) : PlannerPolicy` selects deterministic seeded traversal, preserves every supplied natural seed including zero, and uses definition-ID tie-breaking.
- The underlying structure remains public for parameterized fixtures, breadth-first policies, deliberate identity mutations, and future tie-break choices.
- A seed remains part of canonical Query identity for every strategy; it changes traversal only for the seeded strategy.

## Edge Cases & Constraints

- Do not add a `coverageGuided` constructor or text compatibility alias after the `fn-17.1` rename.
- Preserve the deliberate non-default-seed identity regression and generic arbitrary-strategy/seed planning fixtures.
- Preserve existing comments while changing or refactoring Lean and generated-consumer code.
- Changing shortest policies from seeds `23` and `29` to `17` intentionally changes Query fingerprints and downstream artifact checksums, but not selected traversal order.
- Canonical fixtures and generated views must be regenerated or refreshed as complete owned sets; stale expected bytes are failures, not compatibility baselines.
- No new runtime, filesystem, configuration, or error-handling surface is introduced. Every natural seed is constructible.

## Approach

1. After the Query strategy rename lands, add documented constructors at the principal-type seam and focused field/identity tests.
2. Replace ordinary default-policy literals and fixture defaults with the constructor interface while retaining explicit exceptional mutations.
3. Refresh every canonical query, artifact, generated view, and checksum consumer affected by the standardized seed, then run focused and repository-level model checks.

## Quick commands

```bash
cd model && mise exec -- lake build UmpireTests
cd model && mise exec -- lake build Umpire.Examples.Switch Temporal.Feature.Nexus.LifecycleTests Temporal.Feature.Nexus.OperationsTests Temporal.Feature.Nexus.Experimental.CallerClosureTests
make umpire-check-regression
make lint-model
```

## Acceptance Criteria

- **R1:** The public `PlannerPolicy` interface provides canonical shortest and exhaustive values plus a seeded constructor defaulting to `17`; each produces the documented strategy, seed, and definition-ID tie-break, and an explicit seed including zero is preserved. Errors: no construction error surface; every `Nat` is valid.
- **R2:** Ordinary default policies across reusable Umpire fixtures/examples and Temporal Nexus models use the canonical constructors, with no unexplained `17`, `23`, or `29` policy record literals remaining. Deliberate seed-identity mutations, migration-compatibility record updates, breadth-first coverage, and arbitrary strategy/seed fixtures remain expressible and tested. Errors: unsupported convenience constructors fall back to the existing public record representation rather than being silently coerced.
- **R3:** Standardizing ordinary shortest/exhaustive policies to seed `17` leaves their planner traversal results unchanged while deliberately updating canonical Query identities, artifact checksums, checked fixtures, and generated views. The non-default seed regression still proves that seed changes are identity-significant. Errors: stale, partial, or manually inconsistent fixture sets fail byte/checksum checks.
- **R4:** Public source documentation and the Umpire architecture guide explain the constructor interface, fixed tie-break, shared default, and that seed affects traversal only for the seeded strategy while remaining identity-bearing for every policy. Errors: no runtime error surface; missing or contradictory documentation is an acceptance failure.

## Early proof point

Task fn-40-centralize-plannerpolicy-constructors.1 validates the core approach by compiling the post-rename constructor interface and proving its exact fields and identity semantics.
If it fails, re-evaluate constructor placement and naming against the Query surface produced by `fn-17-bounded-semantic-exploration-and.1` before migrating callers.

## Boundaries

- No `PlannerPolicy.coverageGuided` compatibility alias.
- No breadth-first convenience constructor in this spec.
- No new public `defaultSeed` constant; the default is encapsulated by the constructor interface.
- No change to `PlannerPolicy` representation, Query serialization, planner traversal algorithms, validation, or persisted decoder support.
- No Temporal runtime, API-generation, dynamic-configuration, or deployment work.

## Decision Context

- Use `seeded` instead of the requested `coverageGuided` name because the approved `fn-17.1` contract reserves real coverage guidance for Exploration and renames Query seed rotation without an alias.
- Centralize complete policies instead of only a loose seed constant so callers do not repeat strategy/tie-break representation details.
- Keep the raw record interface as the escape hatch rather than adding convenience constructors for unrequested strategy/tie-break combinations.
- Accept and refresh the seed-`17` identity migration instead of preserving undocumented `23`/`29` values through record updates.
- Reject a public default-seed constant as unnecessary interface surface; constructor results are the single source of the ordinary default.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
|-----|-------------|---------|-------------------|
| R1 | Canonical constructor contracts | fn-40-centralize-plannerpolicy-constructors.1 | — |
| R2 | Migrate ordinary callers while preserving exceptional policies | fn-40-centralize-plannerpolicy-constructors.1, fn-40-centralize-plannerpolicy-constructors.2 | — |
| R3 | Preserve traversal and refresh identity-bearing artifacts | fn-40-centralize-plannerpolicy-constructors.2, fn-40-centralize-plannerpolicy-constructors.3 | — |
| R4 | Document public semantics | fn-40-centralize-plannerpolicy-constructors.1 | — |

## References

- `fn-17-bounded-semantic-exploration-and` — prerequisite spec whose first task renames Query coverage-guided traversal to seeded traversal; flowctl cannot encode a cross-spec task-only edge.
- Umpire 4 rules PLN-02 and ART-01–ART-07 — deterministic selection and complete canonical artifact ownership.
- Lean Authoring Guidelines — principal-type namespaces, public docstrings, focused compilation, and preservation of existing comments.
