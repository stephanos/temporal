# Umpire 4 prototype order

## Goal

Build a minimal but capable vertical slice that demonstrates the possibilities in
`UMPIRE4_VISION.md` with one model and two concrete Nexus examples:

1. **Normal caller closure:** a known deterministic regression executes through a preprogrammed SDK
   participant and satisfies its property.
2. **Duplicate-delivery control:** the same model plus one authored fault produces a qualified
   uniqueness violation, which is replayed, reduced, and proposed as a permanent regression.

The prototype should prove the architecture can support the full vision without first building the
complete production platform. In particular:

- Lean remains the single source of behavioral meaning.
- The same deterministic `ExperimentSpec` is usable locally, in CI, and through a black-box public
  boundary.
- A canary binding is represented by the same environment contract and a dry-run fixture; production
  canary execution is not part of the prototype.
- One preprogrammed SDK participant resolves late-bound Nexus identifiers during execution.
- Evidence conclusions use causal or source-local ordering rather than synchronized clocks, proven
  with deliberately skewed timestamps.
- A small authored variation space and first-class fault demonstrate guided exploration.
- Exploration prioritizes an uncovered semantic coordinate and retains an exact replay and reviewed
  regression proposal.

All retained specs must be reduced to the scope stated below before implementation. Following their
current dependency graph unchanged would pull deferred platform work back into the prototype.

## P0 — Foundations

### 1. fn-31 — Deepen Umpire Target and simplify Temporal target authoring

Make the Nexus model concise and approachable while preserving its existing semantics. Stop after
the domain-neutral Switch and Temporal Nexus targets prove the smaller public boundary. Do not add a
general model AST, second authoring language, or speculative compatibility facade.

This work is already in progress and can proceed alongside fn-4.

### 2. fn-4 — Umpire observation and semantic verdicts

Finish the evidence-to-verdict seam required by the live prototype. Retain the reusable Observation
boundary, qualified derivations, strict outcomes, and only the synthetic/Nexus evidence needed by
the two examples. Broader mutation assurance and documentation are secondary to completing the
working semantic loop.

### 3. fn-32 — Add Umpire Refinement and the first Temporal Feature/System correspondence

Relate one Nexus System trace to its independently authored Feature meaning. Keep observation,
refinement, and property failures distinct. This is the semantic seam that lets local and black-box
execution share the same Feature property without leaking implementation evidence into it.

### 4. fn-16 — Authored variation spaces and deterministic batch compilation

Reduce the general space design to one small Nexus matrix, such as two binary axes, plus the single
duplicate-delivery fault intent. Compile the selected points deterministically through the existing
target-owned planner. Defer generalized coverage vocabularies, arbitrary spaces, and broad catalog
integration.

### 5. fn-18 — Versioned Umpire artifact boundary

Implement only the strict v1 transport needed by the prototype:

- complete executable `ExperimentSpec`;
- `ExperimentRun`;
- bounded `RawEvidence`;
- semantic evidence and `Result`.

Reject malformed, stale, oversized, or identity-inconsistent inputs. Defer generic receipt
envelopes, coverage checkpoints, migrations, complete-set recovery, and artifact-management CLI
surfaces.

## P1 — First complete vertical slice

### 6. fn-19 — Bounded local Temporal execution and SDK participant

Execute the normal Nexus caller-closure example in one ephemeral Temporal environment. Use one
closed preprogrammed SDK participant, resolve the operation/run identifiers at runtime, capture
bounded causal evidence, and report cleanup honestly. Do not generalize participants, execution
profiles, or the local test environment into platforms.

### 7. fn-20 — Local execution semantic conformance

Interpret the local run through the checked Nexus Observation and Refinement declarations and then
evaluate the unchanged Feature property. The result must distinguish operational success, evidence
qualification, refinement, and property satisfaction. Include a fixture with intentionally skewed
wall-clock timestamps that produces the same verdict from causal/source-local ordering.

### 8. fn-21 — Nexus duplicate-observation control

Run the second example. The same model and normal target-owned plan carry one explicit requested
fault. The participant realizes one labeled duplicate-delivery observation, the evidence layer
records a matching receipt, and conformance reports a uniqueness-only violation without claiming a
Temporal product defect.

Completion of fn-21 establishes the core prototype: one satisfied live example and one precise
fault-induced violation using the same semantic model.

## P2 — Exploration and regression lifecycle

### 9. fn-5 — Umpire discovery, promotion, and artifact evolution

Reduce this spec to two capabilities:

- coherent `list` and `explain` output for the retained Nexus declarations and examples;
- one checked, review-only promotion path for the minimized duplicate-delivery failure.

Defer the generic semantic graph, generated glossary, machine index, broad stable regression set,
and general artifact evolution.

### 10. fn-17 — Bounded semantic exploration and coverage

Select experiments deterministically from the small fn-16 Nexus space. Support bounded exhaustive
enumeration and one semantic-coverage-guided policy that prioritizes an uncovered coordinate. Keep
pinned known regressions outside the exploration budget. Defer pairwise/t-wise families, symmetry
proofs, multiple source kinds, generalized resume state, and adaptive corpora.

### 11. fn-33 — Run semantic exploration campaigns with umpire-fuzz

Reduce the campaign to a serial bounded `umpire-fuzz run` command that asks the Lean-owned
exploration layer for candidates, executes them through the existing runner/conformance path, and
reports semantic coverage and exhaustion honestly. Defer concurrency, leases, crash-safe campaign
state, and resume.

### 12. fn-22 — Deterministic replay, semantic minimization, and reviewed promotion

Consume the fn-21 violation, reproduce it exactly, remove one unnecessary authored coordinate while
preserving the same violation, and emit one checked review-only Lean regression proposal. Retain the
distinction between semantic replay and concrete rerun. Defer SDK history replay, generic reducers,
campaign orchestration, and automatic regression installation.

## P3 — Portability proof

### 13. fn-27 — Hermetic CI execution

Run the byte-identical checked-in `ExperimentSpec` through the ordinary CI test command and the same
runner/conformance interfaces used locally. Do not build CI qualification profiles, provenance
schemas, new artifact-set versions, or release evidence.

### 14. fn-28 — Black-box staging execution

Run the same `ExperimentSpec` against one controlled nonproduction endpoint using only public gRPC
evidence plus participant-owned receipts. Reuse existing operational authority instead of building
a general target selector, protected workflow, lease system, recovery controller, or qualification
platform.

The environment binding must also support a canary dry-run fixture that proves the same semantic
artifact and evidence contract can be bound without granting production execution authority.

## Removed from the prototype queue

### Close as superseded

- **fn-14 — Milestone A pilot baseline and Lean-first usability decision.** Its own architecture
  reconciliation marks it as historical and prohibits using it as a roadmap gate.

### Defer until the vertical slice demonstrates value

- **fn-15 — Standalone API and config input catalogs.** Platform completeness, not prototype proof.
- **fn-23 — Veil toolchain compatibility and adoption gate.** Optional checker investigation.
- **fn-24 — Lean-native verification receipts and canonical replay.** Receipt/profile platform;
  existing bounded checking is sufficient for the prototype.
- **fn-25 — Optional CallerClosure Veil binding and canonical replay.** Second verification backend.
- **fn-26 — Local qualification receipts and staged profile contract.** Policy infrastructure after
  a useful local `Result` exists.
- **fn-29 — Bounded production canary execution and qualification.** Production control-plane work;
  the prototype retains only a dry-run binding proof.
- **fn-30 — Release evidence graph and manual authorization.** Release governance after real
  qualification evidence exists.

## Preferred consolidation

The existing Flow IDs may be retained for history, but the minimal roadmap has four conceptual
delivery specs:

1. **Author the Nexus model:** fn-31 and fn-32.
2. **Compile portable experiments:** fn-16 and the minimal fn-18 boundary.
3. **Execute and judge two Nexus examples:** fn-4, fn-19, fn-20, fn-21, and the minimal portability
   checks from fn-27/fn-28.
4. **Explore, replay, and promote:** fn-5, fn-17, fn-33, and fn-22.

The first decision point is the completion of the normal fn-19/fn-20 path. The second is completion
of the fn-21 negative control. Work beyond those gates should remain deferred if either example
cannot demonstrate a concise, deterministic, inspectable end-to-end experience.
