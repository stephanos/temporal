# Bounded semantic exploration and coverage

> HTML render lens: local file `.flow/artifacts/fn-17-bounded-semantic-exploration-and/spec.html` — regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Umpire4 architecture reconciliation

The Lean-owned module exposes the serializable semantic protocol `initialize`, `nextBatch`, and `observe`. It owns semantic candidate identity, selection, mutation meaning, coverage, priorities, corpus decisions, and opaque resumable state. This spec remains pure: it performs no runtime I/O, leasing, checkpoint publication, or command handling.

The downstream campaign spec owns Go concurrency and the `umpire-fuzz` surface while consuming this protocol, artifacts, the shared runner, and conformance. Exhaustive model-only checks stay under `umpire-check-model`; runtime fuzzing never claims completeness.

## Overview

Add one pure reusable `Umpire.Exploration` layer that consumes fn-16's checked finite experiment spaces and atomically compiled candidate universe, then selects useful `ExperimentSpec`s with exhaustive, pairwise, t-wise, seeded-random, or genuinely coverage-guided strategies. It maintains immutable semantic coverage state, supports compatible in-memory resume, honors proof-carrying coverage symmetry, selects pinned regressions outside the exploration budget, and emits an inspectable semantic coverage report.

The layer never executes an experiment, reads persisted artifacts, interprets evidence, realizes a fault, replays or minimizes a result, promotes a regression, or changes Property/Behavior/Query/target semantics.

## Goal & Context
<!-- scope: business -->

Model authors need an exploratory mode that can say “find a small deterministic set of paths that exercises these semantic goals” without confusing case count, requested controls, or model traces with live evidence. Success means the same checked space can be explored repeatedly or resumed with stable selections and an exact explanation of covered, uncovered, pinned, selected, and omitted semantic coordinates.

## Architecture & Data Models
<!-- scope: technical -->

```text
CheckedExperimentSpace + exact base kernel
                    |
                    v
       fn-16 compileBatch (atomic, <= 256)
                    |
                    v
      canonical CandidateUniverse + pinned specs
                    |
     strategy + budget + CoverageState + symmetry
                    |
                    v
       Umpire.Exploration pure selector
             /                  \
            v                    v
 selected ExperimentSpecs   CoverageReport
```

`Umpire.Exploration` is a terminal reusable package above `Umpire.Space`, `Umpire.Planning`, and `Umpire.Artifact`. It does not redefine axes, choices, fault intents, coverage goals, point lowering, target kernels, or artifact compilation. Temporal-owned semantic bindings stay under `Temporal.Feature`; the downstream campaign owns the first concrete command.

`ExplorationStrategy` is separate from the existing per-Query `SearchStrategy`. Its closed v1 variants are `exhaustive`, `pairwise`, `tWise strength`, `seededRandom`, and `coverageGuided`. The existing Query policy still controls target-trace planning inside each point. The misleading existing `SearchStrategy.coverageGuided` seed rotation is renamed to `seeded` with canonical name `seeded`, and legacy `coverage-guided` is rejected rather than aliased; only this package may claim coverage guidance.

An `ExplorationPolicy` contains strategy, an explicit selection-budget ceiling in `experiment-specs`, a canonical seed field, and optional `CheckedCoverageSymmetry`. Bounds are fixed at one to 256 exploratory selections; t-wise strength is two to four and cannot exceed the number of axes. Pairwise is exactly t-wise strength two. Only seeded-random accepts a nonzero seed; every other strategy requires zero. Every output records the canonical seed.

Candidate construction first calls fn-16's atomic `compileBatch` with the caller's exact base kernel. Therefore every strategy sees the same complete, canonically ordered, at-most-256 `ExperimentSpec` universe, and a point-lowering/planning failure rejects exploration before selection. Candidate identity is `ExperimentSpec.semanticIdentity`; duplicate identities reject the universe rather than being silently deduplicated.

Semantic coverage is not raw case count. A candidate's canonical `CoverageSignature` contains:

- selected axis/choice coordinates;
- requested-fault-intent coordinates, explicitly labeled as planned intent rather than realization;
- initial and resulting state, selected action, target-owned outcome, observation, and relation coordinates from the model-selected trace; and
- pure Property evaluation coordinates, including property identity and satisfied/violated result.

Every fn-16 coverage goal is evaluated against those signatures. One distinct spec may credit a given goal at most once, even if its trace repeats the subject. Axis-choice and requested-fault goals match intent coordinates. State/action/outcome/observation/relation goals match target-owned model coordinates. Property goals match a resolved pure model evaluation and the report retains satisfied/violated counts; neither result is live conformance evidence. The report also lists all discovered semantic coordinates so coverage remains meaningful when no explicit goal names them.

`CoverageState` is an immutable checked value with space digest, universe digest, policy-compatibility digest, goal digest, symmetry digest, pinned-set digest, the recorded selection-budget ceiling, selected and omitted candidate identities, per-coordinate hit sets, per-goal distinct credit identities, and selection cursor/provenance. Resume accepts the same checked inputs and a new ceiling greater than or equal to the prior recorded ceiling; it rejects any changed space, universe, algorithm/version, strategy parameter, seed, goals, symmetry, pinned set, reduced ceiling, or non-monotone/tampered counts. This spec exposes no persisted state decoder or migration.

`CheckedCoverageSymmetry` is an optional proof-carrying Lean value over the compiled universe, not an inferred authoring heuristic. It supplies canonical orbit representatives, an explicit axis/choice renaming, and proofs that members have the same goal-credit set and semantic coverage signature under that renaming. It also induces a total quotient on pair/t-wise interaction coordinates and proves that renaming maps every concrete interaction to the representative interaction class. Validation requires total, idempotent representatives in the same universe and disjoint closed orbits. Reduction retains the lexicographically least representative. Reports distinguish directly selected concrete interactions from symmetry-equivalent credited interactions. Without such a witness no symmetry reduction or equivalent credit occurs.

`CoverageReport` has canonical format identity `umpire-coverage-report/v1`, source/policy/state/universe digests, selected and omitted candidates with reasons, direct and symmetry-equivalent interaction coverage, semantic coordinate hit counts, per-goal credited spec identities/minimum/deficit, pinned/exploratory partitions, seed and budget ceilings, and one termination: `goals-satisfied`, `interactions-satisfied`, `universe-exhausted`, or `budget-exhausted`. Goal or interaction satisfaction is not verification. Only exhaustive universe exhaustion can establish that an uncovered model coordinate or goal is unreachable within the checked finite universe; sampled termination never does.

Pinned regressions enter as checked canonical `PinnedExperimentSpec`s. They must have recomputable artifact identity, unique identities, a matching target-kernel contract, and compatible semantic vocabulary. They are included and credited before exploration, never consume the exploration budget, and remain a separate output partition. If a candidate has the same semantic identity, the pinned copy wins and the exploratory candidate is omitted as `pinned-precedence`. Any invalid pinned input rejects the run.

## Selection Algorithms
<!-- scope: technical -->

- `exhaustive` selects canonical non-pinned representatives until the universe ends or the selection ceiling is reached.
- `pairwise` and `tWise` build the exact finite interaction universe from canonical axis-choice assignments, then greedily select the candidate covering the most uncovered interactions. Ties use candidate semantic identity. They stop at `interactions-satisfied` as soon as every direct interaction—or every induced quotient interaction when checked symmetry is present—is covered; an insufficient ceiling returns a valid incomplete report.
- `seededRandom` orders candidates by a stable hash of algorithm version, nonzero seed, and candidate semantic identity; it uses no platform RNG or source order and stops only at universe or budget exhaustion.
- `coverageGuided` greedily maximizes a closed score tuple: newly satisfied required goal credits, then total goal-deficit reduction, then previously unseen semantic coordinates, then candidate semantic identity. It stops at `goals-satisfied`, or at universe/budget exhaustion. Pinned credits and resumed state participate before the first exploratory choice.
- Optional symmetry reduction is a sound preselection quotient for all strategies. Pair/t-wise selectors operate over the proof-induced interaction quotient and report direct versus equivalent credits separately. Reports retain every member-to-representative omission so reduction cannot make the concrete universe appear smaller without explanation.

All algorithms are total over the compiled bounded universe and recompute their canonical result from checked inputs. Pair/t-wise interaction coverage is reported separately from semantic goal coverage. No strategy changes a Query, target kernel, planned trace, or artifact.

## API Contracts
<!-- scope: technical -->

- `checkExplorationRequest` returns one complete checked request or the first structured error in canonical identity order. It validates policy bounds, t strength, pinned specs, symmetry, and optional resume compatibility before selection.
- `buildCandidateUniverse` delegates exclusively to fn-16 `compileBatch` with `kernel : IncrementalPlannerKernel space.baseQuery.target`; it does not construct, copy, or reinterpret a kernel.
- `explore` returns `Except ExplorationError ExplorationResult`. Preselection validation or candidate compilation is atomic: failures return no state/report/specs. Budget termination after a valid universe returns a successful partial result with `budget-exhausted`.
- `resumeExplore` is monotonic. It may only retain prior selections/credits and add new ones; the selection ceiling may stay equal or increase but cannot fall below the prior recorded ceiling. Recomputing fresh at the new ceiling must equal resumed state, selected bytes, and report bytes.
- Canonical output order is pinned specs by identity followed by exploratory specs in selection order; report maps and omission lists are identity-sorted. Strategy decisions record their score/reason.
- `CoverageReport` has a canonical encoder for inspection. There is no report/state reader, filesystem persistence, migration, or compatibility alias in this spec; fn-18 owns strict versioned persisted decoding.
- Versioned pure `initialize`, `nextBatch`, and `observe` functions expose the semantic campaign protocol over canonical values. They perform no leasing, execution, persistence, or command parsing; the downstream campaign spec owns those effects.

## Edge Cases & Constraints
<!-- scope: technical -->

- Empty/duplicate universes, noncanonical or identity-invalid artifacts, mismatched target contracts, impossible strategy parameters, and malformed symmetry or resume state fail before selection.
- The candidate universe never exceeds fn-16's 256-point maximum. Exploration budget counts only newly selected non-pinned specs; per-point Query candidate-evaluation bounds remain independently visible in each artifact.
- Repeated trace subjects credit a goal once per distinct spec. Repeated invocation, resume, pinned/exploratory overlap, and symmetry overlap cannot double-credit.
- Requested fault selection credits only a `fault-intent` coordinate and never target outcome, realization, receipt, or success.
- A statically feasible but dynamically uncovered goal is `uncovered`; it is `unreachable-in-universe` only after exhaustive universe exhaustion. Budget- or goal-terminated runs cannot claim unreachability.
- Reordering authored axes/choices/goals, compiled candidate input, pinned input, or symmetry declarations cannot affect checked identities, selected bytes, or report bytes.
- The package and its tests perform no runtime I/O. Command handling and durable campaign state are downstream concerns.

## Quick commands
<!-- scope: technical -->

```bash
cd model && mise exec -- lake build Umpire.Exploration.Tests.Coverage
cd model && mise exec -- lake build Umpire.Exploration.Tests.Selection
cd model && mise exec -- lake build Umpire.Exploration.Tests.Resume
cd model && mise exec -- lake build Umpire.Exploration.Tests.Symmetry
cd model && mise exec -- lake build Temporal.Feature.Nexus.Examples.ExplorationTests
cd model && mise exec -- lake build Umpire.Exploration.Tests.Protocol
cd model && mise exec -- lake build UmpireTests TemporalModelTests
make umpire-build-model
```

## Acceptance Criteria
<!-- scope: both -->

- **R1:** One reusable checked Exploration language accepts fn-16 spaces plus closed strategy, explicit selection budget/seed, optional proof-carrying symmetry, pinned specs, and compatible prior state. Exact bounds, typed errors, canonical ordering, and semantic identities are enforced before selection. [paraphrase]
- **R2:** Every strategy consumes the same atomic fn-16 candidate universe produced with the caller's exact target kernel. Candidate compilation remains bounded to 256 points, duplicate/noncanonical/non-artifact points reject the run, and per-point planner bounds remain distinct from the exploration selection budget. [user]
- **R3:** Semantic coverage signatures and reports distinguish choice intent, requested-fault intent, target-owned state/action/outcome/observation/relation coordinates, and pure property results. Each distinct spec credits each goal at most once; case count, fault request, or model satisfaction is never represented as execution/conformance evidence. [user]
- **R4:** Exhaustive, exact pairwise, t-wise strength 2–4, stable seeded-random, and genuinely coverage-guided selectors are deterministic, bounded, and explain every selection/omission. Pair/t-wise coverage and semantic-goal coverage remain separate. [paraphrase]
- **R5:** Optional symmetry reduction requires a checked proof that every orbit preserves goal credits and semantic coverage under an explicit axis/choice renaming and induces a total quotient over pair/t-wise interactions; no symmetry is inferred. Representatives and omissions are deterministic, direct/equivalent interaction credits remain distinct, and coverage cannot inflate. [paraphrase]
- **R6:** Immutable state/report values support monotonic compatible in-memory resume with a nondecreasing recorded selection ceiling. Fresh and resumed runs at the same larger ceiling are byte-identical; stale/tampered/incompatible state fails. Termination distinguishes goals satisfied, interactions satisfied, universe exhausted, and budget exhausted without overclaiming verification or reachability. [paraphrase]
- **R7:** Valid pinned regressions are selected and credited before exploration, stay in a separate result partition, consume no exploration budget, and win semantic-identity overlap. Invalid pinned inputs fail the run; no second regression registry or promotion path is created. [user]
- **R8:** Synthetic fixtures and the exact Temporal Nexus fault-matrix example prove deterministic selection, semantic reports, protocol behavior, and vertical package purity. No runtime, evidence, conformance, fault realization, persisted reader/migration, replay/minimization/promotion, Go facade, command, model-local Makefile, or Umpire3 use is introduced. [user]
- **R9:** Lean Exploration provides versioned pure `initialize`, `nextBatch`, and `observe` operations whose canonical state binds the ordered candidate universe, strategy/version, bounds, seed, model/checker identities, issued and observed candidate identities, coverage, priorities, corpus, omissions, and exhaustion status. Errors: stale or crossed state, incompatible strategy/model/bounds, unknown or duplicate result identity, non-monotone update, or incomplete reproduction tuple fails without silently resetting exploration.

## Early proof point
<!-- scope: technical -->

Task `.3` is the algorithm proof gate. On independent three-axis and four-axis finite fixtures, compare pairwise and t-wise strengths two, three, and four with a brute-force interaction oracle at every relevant ceiling; prove seeded-random permutation stability; prove interaction-complete early stopping; prove budget-exhausted reports expose every missing interaction; and prove reordered input is byte-identical. Tasks `.4`–`.7` must not proceed if this oracle disagrees.

## Boundaries
<!-- scope: business -->

- No changes to Property, Behavior, Query, target-owned transition semantics, or the ExperimentSpec schema.
- No alternate Space checker/compiler or target kernel.
- No live runtime, SDK participant, server, fault realization, receipt, evidence, Observation qualification, or semantic conformance.
- No persisted artifact/state/report reader, schema migration, retained campaign store, or compatibility alias.
- No replay, minimization, discovery promotion, source emission, generated Go projection, or second catalog/glossary/registry.
- No release/CI qualification claim.
- No model-local Makefile.
- No Umpire3 inspection, import, invocation, dependency, compatibility, or migration path.

## Decision Context
<!-- scope: both -->

Compiling the complete fn-16 universe before selection is intentionally bounded and gives every strategy one authoritative candidate set. It avoids teaching pairwise or coverage algorithms to fabricate queries, outcomes, or partial artifacts. The 256-point cap makes the pure selection algorithms and brute-force proof fixtures practical.

Exploration strategy is separate from Query search strategy because they act at different levels: Query search chooses one target-owned trace for one point; Exploration chooses among already compiled point artifacts. Renaming the existing seed-rotation policy prevents a false coverage claim while retaining deterministic seeded target enumeration.

Proof-carrying symmetry is optional because automatic semantic equivalence inference would be unsound. Pinned inputs are explicit checked values so the package can enforce precedence without owning fn-5's catalog or regression projection authority.

Persisted resume is deferred to fn-18's versioned decoding boundary. This spec establishes the immutable state identity and canonical report encoder that such a reader must validate, while still proving in-memory resume and command-level fresh exploration.

## References
<!-- scope: technical -->

- `.plans/UMPIRE4_COMPONENTS.md:362-390` — C8 responsibility, strategies, semantic coverage, and pinned precedence.
- `.flow/specs/fn-16-authored-variation-spaces-and.md:14-86` — checked space, goals, proof-carrying lowering, and atomic batch contract.
- `model/Umpire/Search.lean:5-76` — existing per-Query strategy, seed, bounds, and completeness vocabulary.
- `model/Umpire/Planning/Engine.lean:210-230` — current seed rotation that must stop claiming coverage guidance.
- `model/Umpire/Planning/Engine.lean:260-470` — bounded target-owned enumeration and artifact boundary.
- `model/Umpire/Artifact.lean:36-80,228-382` — canonical ExperimentSpec and reserved intent fields.
- `model/Umpire/Property/Language.lean:1162-1228` — unchanged pure model evaluation.
- `model/Temporal/Tool/Inspect.lean:17-88` — effect-thin canonical command pattern.
- `Makefile:988-1032` — root-only model command conventions.

## Requirement coverage
<!-- scope: both -->

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Checked request, strategy, bounds, state | `.1`, `.4`, `.5` | — |
| R2 | Atomic candidate universe | `.2`, `.5` | — |
| R3 | Semantic coordinates, goals, reports | `.2`, `.4`, `.5`, `.6` | — |
| R4 | Five deterministic selectors | `.3`, `.5`, `.6` | — |
| R5 | Proof-carrying symmetry | `.4`, `.5` | — |
| R6 | Resume and exact termination | `.4`, `.5`, `.6` | — |
| R7 | Pinned precedence | `.5`, `.6` | — |
| R8 | Fixtures, Temporal example, protocol, docs | `.1`–`.7` | — |
| R9 | Serializable pure exploration protocol | `.1`, `.4`, `.5`, `.7` | — |
