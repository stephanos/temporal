# Umpire 4 prototype order

## Goal

Build a minimal but capable vertical slice that demonstrates the possibilities in
`UMPIRE4_VISION.md` with one model and two concrete Nexus examples:

1. **Normal caller closure:** a known deterministic regression executes through a preprogrammed SDK
   participant and satisfies its property.
2. **Duplicate-delivery control:** the same model plus one authored fault produces a accepted
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
- The existing authored variation space and first-class fault demonstrate guided exploration.
- Exploration prioritizes an uncovered Model Coordinate and retains an Exact Replay and reviewed
  regression proposal.

Open specs must retain the reduced scope stated below. A retained task must not depend on a deferred
task solely to inherit broader machinery. Split mixed tasks at that boundary and connect the retained
path to its nearest retained prerequisite. The numbered sections below express delivery priority,
not additional hard dependencies. Completed specs are prerequisites, not entries in this queue.

## P0 — Foundations

### 1. fn-38 — Consolidate layered model helpers without API churn

Move repeated Definition ID, Source Location, and Definition Metadata construction into the
narrowest valid helper layer. Use `Umpire.Shared` for reusable Umpire production construction,
`Umpire.Shared.Test` for proven Umpire fixture reuse, and `Temporal.Shared` for Temporal-specific
production construction. Keep `Shared.Test` and `Temporal.Shared.Test` absent until a concrete,
multi-consumer helper qualifies for those ownership boundaries.

Preserve every existing public declaration, import path, observable value, and comment. Existing
Switch and Nexus modules remain the consumer-facing facades and delegate internally. Extend the
executable import policy so production code cannot reach test-support modules and
`Temporal.Shared` cannot reach Feature or System modules. Fn-38 is next in the unstarted queue.

### 2. fn-41 — FiniteMachine target authoring

Add one deep, Lean-native `FiniteMachine` adapter for complete finite Targets whose enumerators are
the authoritative behavior. Derive the membership relations, routine soundness/completeness proofs,
complete behavior domain, and target-owned finite planning from explicit ordered domains, encoders,
initial states, transitions, closure evidence, and executable-action evidence.

Migrate the Temporal Feature and System Nexus lifecycle Targets without changing public names,
proof seams, source provenance, comments, checked meaning, Query values, planner results, or Artifact
bytes. Keep direct `TransitionKernel` construction as the expert path for independently authored
relations. Fn-41 follows fn-38 because both touch Nexus Lifecycle construction, and it must finish
before fn-39 splits that simplified implementation across modules.

### 3. fn-39 — Make the Temporal Nexus Feature model easier to browse

Preserve the existing public Nexus namespaces, import paths, source provenance, semantic identities,
canonical artifacts, and comments while making the ordinary Feature model easier for new
contributors to navigate. Add one documented Nexus facade; split Lifecycle internally into
Semantics and Target; split Operations internally into AsyncStart, Cancellation,
SuccessfulCompletion, and Planning; and mirror that structure in descriptive named tests.

Keep Observation ownership and paths unchanged, leave AutoClose and CallerClosure physically
intact, and add only a navigation map to CallerClosure. Fn-39 follows both fn-38's source-compatible
helper consolidation and fn-41's Lifecycle migration, and is consumed by fn-19's Nexus facade work.

### 4. fn-42 — Centralize configuration authoring with ConfigUseSpec

Introduce one typed `ConfigUseSpec α` authoring interface that owns each independently authored
configuration key, identity, schema, default, policy, fingerprint, decoder, and change meaning once.
Project the existing classification, interpretation, and definition forms, and delegate checking to
the current validator through an explicit proof-taking checked extraction seam.

Hard-cut the four Callback and two Matching declarations to that interface while preserving their
contexts, registries, use functions, resolution behavior, provenance, diagnostics, ordering, and
comments. Fn-42 is independent of the Nexus/runtime critical path and may run alongside fn-38 through
fn-41; it does not gate fn-18 or fn-19.

### 5. fn-18 — Versioned Umpire Artifact boundary

Retain the v2-only DrivePlan and ExperimentSpec families while replacing the pre-release compact
encoding with one deterministic pretty JSON representation and newly derived checksums. Implement
only the additional strict formats needed by the prototype:

- `RuntimeConfiguration`;
- `ExperimentRun`;
- bounded `RawEvidence`;
- interpreted Evidence and `Result`.

Retain fail-closed admission for one complete prototype artifact set, including strict
cross-document identity closure and one immutable atomic publication/loading path that never exposes
a partial or mixed set. Preserve the current v2 Definition IDs, Behavior Fingerprints, and canonical
content, but complete the explicit pre-release cut to pretty canonical bytes and their checksum
preimages atomically across Lean, Go, fixtures, and generated views. Consume the checked intent and
compiled experiment produced by the completed fn-16, complete its executable contract through fn-18,
and require every Execution boundary to consume the same published bytes and identifiers without
recompilation. Reject compact or alternate-whitespace v2, v1, malformed, stale, oversized, or
checksum-inconsistent inputs without a compatibility reader or migration.
Expose only read-only checks for one Artifact and one complete set; checking never publishes.
Immutable publication may clean its own abandoned private staging directories while holding its
lock. Defer generic receipt envelopes, coverage checkpoints, post-v2 migrations, multi-root or
remote recovery, mutating artifact-management CLI surfaces, and platform orchestration.

## P1 — First complete vertical slice

### 6. fn-19 — Bounded local Temporal execution and SDK participant

Execute the normal Nexus caller-closure example in one ephemeral Temporal environment. Use one
closed preprogrammed SDK participant, resolve the operation/run identifiers at runtime, capture
bounded causal evidence, and report cleanup honestly. Do not generalize participants, execution
profiles, or the local test environment into platforms.

Consume the normal fn-18-published executable `ExperimentSpec` from the existing checked intent
without recompiling it.

### 7. fn-20 — Local Run Evaluation

Interpret the local Run through the checked Nexus Observation and Implementation Link declarations
and then evaluate the unchanged Feature Property. The Result must distinguish operational success,
Observation Evaluation, Implementation Link, and Property satisfaction. Include a fixture with
intentionally skewed wall-clock timestamps whose sorted order contradicts the causal or source-local
order. Use a Model Trace whose Observation Evaluation or Property Result would change under
timestamp sorting, then assert the expected Evidence Link and Result.

### 8. fn-21 — Nexus duplicate-observation control

Run the second example. The same model and normal target-owned plan carry one explicit Fault Request.
The participant realizes one labeled duplicate-delivery observation, the Evidence layer records a
matching Execution Receipt, and Run Evaluation reports a uniqueness-only violation without claiming
a Temporal product defect.

Consume the faulted fn-18-published executable `ExperimentSpec` from the existing checked intent;
do not author an alternative space, Feature Property, or Implementation Link inside fn-21.

Completion of fn-21 establishes the core prototype: one satisfied live example and one precise
fault-induced violation using the same model.

## P2 — Portability proof

### 9. fn-27 — Hermetic CI execution

Run the byte-identical normal `ExperimentSpec` consumed by fn-19 through the ordinary CI test command
and the same runner/Run Evaluation interfaces used locally. Its Artifact Checksum, format version,
and Behavior Fingerprints must match the local subject. Reject recompilation or checksum drift
without introducing a new provenance schema. Do not build CI Claim Assessment profiles, provenance schemas,
new artifact-set versions, or release evidence.

### 10. fn-28 — Black-box staging execution

Run the same normal `ExperimentSpec` against one controlled nonproduction endpoint using only public
gRPC Evidence plus participant-owned Execution Receipts. Before implementation, name the owner-supplied fixed
staging profile and harness that provide fail-closed authority and target preflight, concurrency one,
fixed Execution/Evidence Limits, isolated namespace or Run-owned resources, cleanup verification,
and postflight target identity. If those existing controls are unavailable, fn-28 is blocked; do not
build replacements in Umpire. Do not build a general target selector, protected workflow, lease
system, recovery controller, or Claim Assessment platform.

The environment binding must also support a canary dry-run fixture that consumes the same normal
Artifact bytes, format version, Artifact Checksum, and Behavior Fingerprints, proving the Artifact
and Evidence contract can be bound without granting production Execution authority.

## P3 — Exploration and regression lifecycle

### 11. fn-5 — Umpire discovery, promotion, and Artifact evolution

Reduce this spec to two capabilities:

- coherent `list` and `explain` output for the retained Nexus declarations and examples;
- one checked, review-only promotion path for the minimized duplicate-delivery failure.

Defer the generic semantic graph, generated glossary, machine index, broad stable regression set,
and general artifact evolution.

### 12. fn-17 — Bounded model exploration and coverage

Select experiments deterministically from the existing small Nexus space. Support bounded exhaustive
enumeration and one semantic-coverage-guided policy that prioritizes an uncovered coordinate. Keep
pinned known regressions outside the exploration budget. Defer pairwise/t-wise families, symmetry
proofs, multiple source kinds, generalized resume state, and adaptive corpora.

Keep the uncovered-coordinate policy independent of the deferred symmetry, generalized reporting,
and resume machinery when reducing the existing mixed tasks.

### 13. fn-40 — Centralize PlannerPolicy constructors and default seed

After fn-17 renames Query's seed-rotated strategy to `seeded`, add canonical
`PlannerPolicy.shortest`, `PlannerPolicy.exhaustive`, and `PlannerPolicy.seeded` constructors. Use
seed `17` and Definition-ID tie-breaking for ordinary policies while keeping the public record as the
escape hatch for deliberate non-default seeds, breadth-first policies, and generic fixtures.

Migrate ordinary Umpire and Temporal callers, then refresh the complete canonical Query, Artifact,
generated-view, and checksum sets whose identity intentionally changes when seeds `23` and `29`
become `17`. Traversal for shortest and exhaustive policies must remain unchanged. Fn-40 follows
fn-17's strategy rename and should settle the ordinary policy surface before fn-33 builds the campaign
command around exploration.

### 14. fn-33 — Run model exploration campaigns with umpire-fuzz

Reduce the campaign to a serial bounded `umpire-fuzz run` command that asks the Lean-owned
exploration layer for candidates, executes them through the existing runner/Run Evaluation path, and
reports semantic coverage and exhaustion honestly. Defer concurrency, leases, crash-safe campaign
state, and resume.

### 15. fn-22 — Deterministic replay, model minimization, and reviewed promotion

Consume the fn-21 violation, reproduce it exactly, and try every applicable authored reduction in a
fixed order while preserving the same violation. The exact control may complete as irreducible; its
diagnostic EvidenceCore must still omit one labeled non-responsible evidence fact without rewriting
the admitted evidence artifacts. Emit one checked review-only Lean regression proposal. Retain the
distinction between semantic replay and concrete rerun. Defer SDK history replay, generic reducers,
campaign orchestration, and automatic regression installation.

Continue the fixed-order bounded reduction until every remaining applicable authored edit
conclusively fails to preserve the same violation. Only a complete `minimized` or `irreducible`
result may feed the review-only proposal.

## Developer simplicity follow-up (non-prototype-gating)

Run this track opportunistically after each owning surface is stable. None of these specs is a
prerequisite for P0 through P3 or for the prototype verification gate. They preserve existing
checked semantics, public proof seams, canonical identities, artifacts, generated bytes, and
comments while making ordinary model code shorter and easier to review.

- **fn-44 — Seal Observation traces and centralize semantic coordinates.** Introduce one accepted
  Evidence-backed trace boundary and one shared Model Trace coordinate vocabulary so consumers stop
  repeating trace acceptance and coordinate derivation.
- **fn-50 — Migrate System CallerClosure to FiniteMachine.** Reuse the completed finite Target deep
  module while retaining a compatibility kernel for the equality and conjunction proof shapes used
  by existing Implementation Link consumers.
- **fn-43 — Deepen ordinary Property, Behavior, and Query authoring.** Complete the existing ordinary
  authoring surface after fn-38, fn-40, fn-41, fn-44, and fn-50 settle its dependencies.
- **fn-48 — Canonicalize Known Gaps as a checked set.** Replace repeated list validation and lookup
  with one checked semantic boundary, consuming fn-43 and the fn-47 semantic outcome/Known Gap
  inventory while retaining independent Go wire admission.
- **fn-49 — Centralize Observation field and structural contracts.** Give Observation one field-spec
  vocabulary and one normalized ordering/closure analysis shared by checking, runtime admission,
  canonicalization, and artifacts.
- **fn-51 — Shorten ordinary model authoring.** Add inert typed constructors for named Model Values,
  bounded Query limits, ordinary Space leaves, and forward Implementation Link mappings, backed by
  repository-wide migration inventories and unchanged expert record paths.

The simplicity-specific dependency shape is:

```text
fn-41 -> fn-50 -> fn-43 -> {fn-48, fn-49, fn-51}
fn-44 -----------> fn-43
fn-47 ---------------------> fn-48
```

Fn-47 retains its own inventory dependencies, including fn-44. Fn-48 may later feed the deferred
fn-26, fn-29, and fn-30 governance work without pulling those specs into the prototype queue.

## Prototype verification gate

Complete this gate after P2 and before starting P3:

- `flowctl validate --all --json` passes after the epic and retained-task dependency edits, with no
  retained task depending on a deferred task;
- `go test -count=1 ./tools/umpire/artifact/... ./tools/common/artifactio/...` proves strict pretty-v2
  admission, compact/alternate-whitespace rejection, partial/mixed-set rejection, atomic visibility,
  and exact byte/identity preservation;
- `cd model && mise exec -- lake build Umpire.Artifact.Tests.Codecs` emits the canonical pretty-v2
  golden that the Go admission tests consume with exact bytes and independently recomputed identities;
- the fn-20 skew fixture proves timestamp sorting would change the outcome while causal/source-local
  ordering produces the expected derivation and verdict;
- the documented fn-19/fn-20 normal commands and fn-21 duplicate-delivery commands complete with the
  expected satisfied and uniqueness-only results; and
- local, ordinary CI, fixed-profile staging, and canary dry-run records name the same normal artifact
  byte hash, format identity, and Behavior Fingerprint, while staging also records bounded execution,
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
- **fn-26 — Local Evaluation Receipts and staged profile contract.** Policy infrastructure after
  a useful local `Result` exists.
- **fn-29 — Bounded production canary execution and Claim Assessment.** Production control-plane work;
  the prototype retains only a dry-run binding proof.
- **fn-30 — Release evidence graph and manual authorization.** Release governance after real
  Claim Assessment evidence exists.

## Preferred consolidation

The existing Flow IDs may be retained for history, but the remaining roadmap has seven conceptual
delivery tracks:

1. **Consolidate layered model helpers:** fn-38, preserving the current public API.
2. **Deepen and organize ordinary Nexus authoring:** fn-41 followed by fn-39, preserving the public
   API and canonical outputs while simplifying Target construction and then separating semantic,
   target, walkthrough, planning, and test concerns.
3. **Centralize configuration authoring:** fn-42, independently hard-cutting Callback and Matching
   declarations to the checked `ConfigUseSpec` seam.
4. **Persist portable experiments:** the minimal fn-18 boundary over the completed fn-16 output.
5. **Execute and judge two Nexus examples portably:** fn-19, fn-20, fn-21, and the minimal fn-27/fn-28
   portability checks.
6. **Explore, standardize policy authoring, replay, and promote:** fn-5, fn-17, fn-40, fn-33, and
   fn-22.
7. **Harden and shorten model authoring without gating the prototype:** fn-44 and fn-50 settle the
   reusable trace and finite-machine seams; fn-43 deepens ordinary authoring; fn-48, fn-49, and
   fn-51 then centralize Known Gaps and Observation structure and remove repetitive record literals.

The first decision point is the completion of the normal fn-19/fn-20 path. The second is completion
of the fn-21 negative control. Complete the reduced portability proof after that second gate and
before the exploration lifecycle. Work beyond those gates should remain deferred if either example
cannot demonstrate a concise, deterministic, inspectable end-to-end experience.
