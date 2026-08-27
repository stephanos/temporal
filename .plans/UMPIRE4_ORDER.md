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

Make only these epic-level dependency edits:

- remove fn-15 from fn-5;
- remove fn-17 and fn-33 from fn-22 while retaining its existing semantic, artifact, conformance,
  fn-21 control, and fn-5 promotion prerequisites;
- remove fn-24 and fn-26 from fn-27;
- remove fn-26 from fn-28.

All other epic-level dependencies remain unchanged. Apply the same retained/deferred boundary inside
each reduced spec: a retained task must not depend on a deferred task solely to inherit broader
machinery. Split mixed tasks at that boundary and connect the retained path to its nearest retained
prerequisite. The numbered sections below express delivery priority, not additional hard dependencies.

## P0 — Foundations

### 1. fn-31 — Deepen Umpire Target and simplify Temporal target authoring

Make the Nexus model concise and approachable while preserving its existing semantics. Stop after
the domain-neutral Switch and Temporal Nexus targets prove the smaller public boundary. Do not add a
general model AST, second authoring language, or speculative compatibility facade.

This work is already in progress and can proceed alongside fn-4.

### 2. fn-4 — Umpire observation and semantic verdicts

Finish the evidence-to-verdict seam required by the live prototype. Retain the reusable Observation
boundary, qualified derivations, strict outcomes, and only the synthetic/Nexus evidence needed by
the two examples. Retain concise documentation for that reduced Observation API and live handoff;
defer the broader cross-layer mutation matrix and documentation surface.

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

Implement only the strict formats needed by the prototype:

- read-only compatibility for existing `umpire-experiment/v1` inputs and one current, complete
  executable `umpire-experiment/v2` successor;
- `RuntimeConfiguration`;
- `ExperimentRun`;
- bounded `RawEvidence`;
- semantic evidence and `Result`.

Retain fail-closed admission for one complete prototype artifact set, including strict
cross-document identity closure and one immutable atomic publication/loading path that never exposes
a partial or mixed set. Fn-18 owns the canonical content, semantic-identity, and artifact-identity
formulas for `umpire-experiment/v2`. Compile the retained semantic intent once through fn-16,
construct its complete executable successor through fn-18, and require every execution boundary to
consume the same published bytes and identities without recompilation. Do not add a migration from
the compatibility input. Reject malformed, stale, oversized, or identity-inconsistent inputs.
Defer generic receipt envelopes, coverage checkpoints, other migrations, interrupted-publication
recovery, and artifact-management CLI surfaces.

## P1 — First complete vertical slice

### 6. fn-19 — Bounded local Temporal execution and SDK participant

Execute the normal Nexus caller-closure example in one ephemeral Temporal environment. Use one
closed preprogrammed SDK participant, resolve the operation/run identifiers at runtime, capture
bounded causal evidence, and report cleanup honestly. Do not generalize participants, execution
profiles, or the local test environment into platforms.

Consume the normal fn-18-published executable `ExperimentSpec` built from fn-16's checked intent
without recompiling it.

### 7. fn-20 — Local execution semantic conformance

Interpret the local run through the checked Nexus Observation and Refinement declarations and then
evaluate the unchanged Feature property. The result must distinguish operational success, evidence
qualification, refinement, and property satisfaction. Include a fixture with intentionally skewed
wall-clock timestamps whose sorted order contradicts the causal or source-local order. Use a trace
whose qualification or property result would change under timestamp sorting, then assert the
expected causal derivation and verdict.

### 8. fn-21 — Nexus duplicate-observation control

Run the second example. The same model and normal target-owned plan carry one explicit requested
fault. The participant realizes one labeled duplicate-delivery observation, the evidence layer
records a matching receipt, and conformance reports a uniqueness-only violation without claiming a
Temporal product defect.

Consume the faulted fn-18-published executable `ExperimentSpec` built from fn-16's checked intent;
do not author an alternative space, Feature property, or Refinement inside fn-21.

Completion of fn-21 establishes the core prototype: one satisfied live example and one precise
fault-induced violation using the same semantic model.

## P2 — Portability proof

### 9. fn-27 — Hermetic CI execution

Run the byte-identical normal `ExperimentSpec` consumed by fn-19 through the ordinary CI test command
and the same runner/conformance interfaces used locally. Its canonical byte hash, format identity,
and semantic identity must match the local subject. Reject recompilation or identity drift without
introducing a new provenance schema. Do not build CI qualification profiles, provenance schemas,
new artifact-set versions, or release evidence.

### 10. fn-28 — Black-box staging execution

Run the same normal `ExperimentSpec` against one controlled nonproduction endpoint using only public
gRPC evidence plus participant-owned receipts. Before implementation, name the owner-supplied fixed
staging profile and harness that provide fail-closed authority and target preflight, concurrency one,
fixed execution/evidence bounds, isolated namespace or run-owned resources, cleanup verification,
and postflight target identity. If those existing controls are unavailable, fn-28 is blocked; do not
build replacements in Umpire. Do not build a general target selector, protected workflow, lease
system, recovery controller, or qualification platform.

The environment binding must also support a canary dry-run fixture that consumes the same normal
artifact bytes, format identity, and semantic identity, proving the semantic artifact and evidence
contract can be bound without granting production execution authority.

## P3 — Exploration and regression lifecycle

### 11. fn-5 — Umpire discovery, promotion, and artifact evolution

Reduce this spec to two capabilities:

- coherent `list` and `explain` output for the retained Nexus declarations and examples;
- one checked, review-only promotion path for the minimized duplicate-delivery failure.

Defer the generic semantic graph, generated glossary, machine index, broad stable regression set,
and general artifact evolution.

### 12. fn-17 — Bounded semantic exploration and coverage

Select experiments deterministically from the small fn-16 Nexus space. Support bounded exhaustive
enumeration and one semantic-coverage-guided policy that prioritizes an uncovered coordinate. Keep
pinned known regressions outside the exploration budget. Defer pairwise/t-wise families, symmetry
proofs, multiple source kinds, generalized resume state, and adaptive corpora.

Keep the uncovered-coordinate policy independent of the deferred symmetry, generalized reporting,
and resume machinery when reducing the existing mixed tasks.

### 13. fn-33 — Run semantic exploration campaigns with umpire-fuzz

Reduce the campaign to a serial bounded `umpire-fuzz run` command that asks the Lean-owned
exploration layer for candidates, executes them through the existing runner/conformance path, and
reports semantic coverage and exhaustion honestly. Defer concurrency, leases, crash-safe campaign
state, and resume.

### 14. fn-22 — Deterministic replay, semantic minimization, and reviewed promotion

Consume the fn-21 violation, reproduce it exactly, and try every applicable authored reduction in a
fixed order while preserving the same violation. The exact control may complete as irreducible; its
diagnostic EvidenceCore must still omit one labeled non-responsible evidence fact without rewriting
the admitted evidence artifacts. Emit one checked review-only Lean regression proposal. Retain the
distinction between semantic replay and concrete rerun. Defer SDK history replay, generic reducers,
campaign orchestration, and automatic regression installation.

Continue the fixed-order bounded reduction until every remaining applicable authored edit
conclusively fails to preserve the same violation. Only a complete `minimized` or `irreducible`
result may feed the review-only proposal.

## Prototype verification gate

Complete this gate after P2 and before starting P3:

- `flowctl validate --all --json` passes after the epic and retained-task dependency edits, with no
  retained task depending on a deferred task;
- `go test -count=1 ./tools/umpire/artifact/... ./tools/common/artifactio/...` proves strict successor
  admission, partial/mixed-set rejection, atomic visibility, and exact byte/identity preservation;
- `cd model && mise exec -- lake build Umpire.Artifact.Tests.Codecs` emits the canonical successor
  golden that the Go admission tests consume with exact bytes and independently recomputed identities;
- the fn-20 skew fixture proves timestamp sorting would change the outcome while causal/source-local
  ordering produces the expected derivation and verdict;
- the documented fn-19/fn-20 normal commands and fn-21 duplicate-delivery commands complete with the
  expected satisfied and uniqueness-only results; and
- local, ordinary CI, fixed-profile staging, and canary dry-run records name the same normal artifact
  byte hash, format identity, and semantic identity, while staging also records bounded execution,
  isolation, and complete cleanup.

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

1. **Author and interpret the Nexus model:** fn-31, fn-4, and fn-32.
2. **Compile portable experiments:** fn-16 and the minimal fn-18 boundary.
3. **Execute and judge two Nexus examples portably:** fn-19, fn-20, fn-21, and the minimal fn-27/fn-28
   portability checks.
4. **Explore, replay, and promote:** fn-5, fn-17, fn-33, and fn-22.

The first decision point is the completion of the normal fn-19/fn-20 path. The second is completion
of the fn-21 negative control. Complete the reduced portability proof after that second gate and
before the exploration lifecycle. Work beyond those gates should remain deferred if either example
cannot demonstrate a concise, deterministic, inspectable end-to-end experience.
