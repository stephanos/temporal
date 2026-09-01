# Bounded semantic exploration and coverage

> HTML render lens (local): open `.flow/artifacts/fn-17-bounded-semantic-exploration-and/spec.html` — regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Overview

Add one pure `Umpire.Exploration` layer over the existing checked finite Experiment Space. It offers
exactly two deterministic policies: bounded exhaustive enumeration and one policy that prioritizes a
caller-named uncovered Model Coordinate. Checked pinned Regressions are selected independently of
the exploration budget and take precedence over duplicate exploratory candidates.

The layer does not execute experiments, interpret Evidence, persist campaign state, or promote
Regressions. Fn-33 owns the first runtime campaign and consumes only the narrow in-memory
one-candidate session seam defined here.

## Goal & Context
<!-- scope: business -->

The retained Nexus prototype needs a small, explainable way to enumerate its finite candidate space
or choose a candidate that covers one known gap. The same checked inputs and Limits must yield the
same selected identities, while partial work must never claim exhaustive coverage.

## Architecture & Data Models
<!-- scope: technical -->

```text
CheckedExperimentSpace + exact planner kernel
                    |
                    v
        canonical CandidateUniverse (<= 256)
                    |
      exhaustive | uncovered-coordinate
                    |
                    v
    pinned partition + exploratory selections
                    |
                    v
      one-candidate in-memory session for fn-33
```

`CandidateUniverse` is compiled atomically from one fn-16 `CheckedExperimentSpace` with the caller's
exact target kernel. It contains canonical ExperimentSpec bytes and identities plus the Model
Coordinates already present in each selected model trace. Duplicate identities, invalid artifacts,
or more than 256 candidates reject the request before selection.

`ExplorationPolicy` is closed to `exhaustive` and `uncoveredCoordinate coordinate`. Both use an
explicit `experiment-specs` Limit. Exhaustive succeeds only after every non-pinned candidate in the
finite universe has been considered; reaching the Limit first is `limit-reached`, not exhaustion.
The guided policy selects candidates containing the named coordinate first, with ExperimentSpec
semantic identity as the tie-break, then stops at its Limit. It does not mutate the space, learn a
corpus, or alter subsequent scoring from runtime outcomes.

`PinnedExperimentSpec` inputs are checked canonical ExperimentSpecs with unique identities and the
same target-kernel contract. They form a separate ordered partition, do not consume the exploration
Limit, and win semantic-identity overlap.

The in-memory `ExplorationSession` fixes one checked request and selected order. `next` returns at
most one not-yet-observed candidate; `observe` accepts exactly the checked admission binding for that
candidate before another can be returned. The value is process-local and has no encoder, decoder,
checkpoint, compatibility version, or restart contract.

## Selection Algorithms
<!-- scope: technical -->

- `exhaustive` walks non-pinned candidates in canonical semantic-identity order and reports
  `exhausted` only when the complete checked universe was considered within the Limit.
- `uncoveredCoordinate coordinate` puts candidates containing that exact Model Coordinate first,
  then orders ties by semantic identity. If no candidate contains it, the result says
  `coordinate-uncovered`; it does not claim the coordinate is unreachable unless exhaustive
  enumeration also completed.
- Pinned Regressions precede exploratory selections, consume no exploration budget, and remove an
  overlapping exploratory identity with the explicit reason `pinned-precedence`.

## API Contracts
<!-- scope: technical -->

- `checkExplorationRequest` checks one finite Space, exact kernel, closed policy, Limit, coordinate,
  and pinned partition atomically and returns canonical typed errors.
- `buildCandidateUniverse` delegates to fn-16 `compileBatch`, preserves canonical ExperimentSpec
  bytes, rejects duplicate identities, and never reads a catalog or filesystem path.
- `explore` returns the pinned and exploratory partitions, the selected identities, the requested
  coordinate outcome, and `exhausted|limit-reached`; it exposes no general reporting schema.
- `beginSession`, `next`, and `observe` provide the minimal process-local one-candidate seam consumed
  by fn-33. Crossed, stale, duplicate, missing, or extra observations return no successor session.
- Canonical ordering depends only on checked semantic inputs, Limits, and ExperimentSpec identities,
  never source order, timestamps, platform randomness, or runtime completion order.

## Edge Cases & Constraints
<!-- scope: technical -->

- Empty or oversized universes, invalid or duplicate artifacts, incompatible pinned inputs, zero or
  oversized Limits, and a guided coordinate outside the checked coordinate vocabulary fail before
  selection.
- Requested fault coordinates remain model intent. Selecting them is not Evidence that a fault was
  realized or that a Property passed.
- A reached Limit is inconclusive. Only complete exhaustive enumeration can establish finite-space
  exhaustion.
- The package is pure Lean and performs no runtime I/O, command handling, persistence, or promotion.

## Quick commands
<!-- scope: technical -->

```bash
cd model && mise exec -- lake build Umpire.Exploration.Tests.Validation
cd model && mise exec -- lake build Umpire.Exploration.Tests.Selection
cd model && mise exec -- lake build Umpire.Exploration.Tests.Session
cd model && mise exec -- lake build Temporal.Feature.Nexus.Examples.ExplorationTests
cd model && mise exec -- lake build UmpireTests TemporalModelTests
make umpire-build-model
```

## Acceptance Criteria
<!-- scope: both -->

- **R1:** One checked finite candidate universe of at most 256 canonical ExperimentSpecs is compiled
  atomically from one fn-16 Space with the exact planner kernel. Errors: empty/oversized input,
  compilation failure, invalid artifacts, duplicate identities, or incompatible bounds reject with
  no partial universe.
- **R2:** Bounded exhaustive enumeration is deterministic and reports exhaustion only after every
  non-pinned candidate in the checked finite universe has been considered. Reaching the explicit
  exploration Limit first reports `limit-reached` and proves no absence claim.
- **R3:** The only guided policy prioritizes one caller-named uncovered Model Coordinate and uses
  semantic identity for ties. It reports whether that coordinate was selected or remains uncovered
  without changing the Space, Query, Target, or policy from observations.
- **R4:** Valid pinned Regressions are checked and selected in a separate identity-sorted partition,
  consume no exploration budget, and win identity overlap. Invalid or incompatible pinned inputs
  reject the request atomically.
- **R5:** Pure focused fixtures and the exact Nexus Space prove deterministic exhaustive and guided
  selections, pinned precedence, truthful Limit/exhaustion outcomes, and the minimal in-memory
  one-candidate session. No runtime, persistence, or command surface enters `Umpire.Exploration`.

## Early proof point
<!-- scope: technical -->

Task `.3` must prove bounded exhaustive ordering and Limit semantics on a small finite fixture.
Task `.4` then proves that one requested uncovered coordinate changes the first eligible selection
without changing the universe or inventing runtime feedback. Integration work must not proceed if
either proof is nondeterministic.

## Boundaries
<!-- scope: business -->

- Pairwise and t-wise families, symmetry proofs, seeded sampling families, multiple source kinds,
  generalized resume state, generalized coverage reporting, and adaptive corpora are deferred.
- No persisted reader, migration, checkpoint, lease, campaign service, runtime Evidence handling,
  replay, minimization, promotion, or alternate Regression registry.
- No changes to Property, Behavior, Query, target-owned transition semantics, or ExperimentSpec.
- No model-local Makefile, Umpire3 compatibility path, or new command.

## Decision Context
<!-- scope: both -->

The checked finite Space already contains the small Nexus prototype universe. Two policies are
enough to demonstrate complete bounded search and one semantic-gap-directed choice. Keeping the
session process-local avoids defining recovery infrastructure before the serial campaign proves
useful. Pinned precedence stays in the pure layer because independence from exploration Limits is a
model-selection invariant.

## References
<!-- scope: technical -->

- `.plans/UMPIRE4_ORDER.md` — retained bounded fn-17 scope.
- `.plans/UMPIRE4_SPEC.md` — PLN-01 through PLN-05 and EXP-01 through EXP-05.
- `.flow/specs/fn-16-authored-variation-spaces-and.md` — checked finite Space and atomic compilation.
- `model/Umpire/Search.lean` — current planner policy and semantic-identity ordering vocabulary.
- `model/Umpire/Artifact.lean` — canonical ExperimentSpec identity/content rules.

## Requirement coverage
<!-- scope: both -->

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Checked finite universe | `.1`, `.2` | — |
| R2 | Bounded exhaustive enumeration | `.3`, `.5` | — |
| R3 | Uncovered-coordinate guidance | `.4`, `.5`, `.6` | — |
| R4 | Pinned precedence outside budget | `.5`, `.6` | — |
| R5 | Nexus proof, session, facades, docs | `.6`, `.7`, `.8` | — |
