# Umpire 4 prototype order

## Goal

Build a minimal but capable vertical slice that demonstrates the possibilities in
`UMPIRE4_VISION.md` with one model and two concrete Nexus examples:

1. **Normal caller closure:** a known deterministic regression executes through a preprogrammed SDK
   participant and satisfies its property.
2. **Duplicate-delivery control:** the same model plus one authored fault produces an accepted
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

## Remaining P0 foundation

### 1. fn-42 — Centralize configuration authoring with ConfigUseSpec

Introduce one typed `ConfigUseSpec α` authoring interface that owns each independently authored
configuration key, identity, schema, default, policy, fingerprint, decoder, and change meaning once.
Project the existing classification, interpretation, and definition forms, and delegate checking to
the current validator through an explicit proof-taking checked extraction seam.

Hard-cut the four Callback and two Matching declarations to that interface while preserving their
contexts, registries, use functions, resolution behavior, provenance, diagnostics, ordering, and
comments. Fn-42 is independent of the Nexus/runtime critical path and does not gate staging.

## Remaining P2 portability proof

### 2. fn-28 — Black-box staging execution

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

### 3. fn-5 — Umpire discovery, promotion, and Artifact evolution

Reduce this spec to two capabilities:

- coherent `list` and `explain` output for the retained Nexus declarations and examples;
- one checked, review-only promotion path for the minimized duplicate-delivery failure.

Defer the generic semantic graph, generated glossary, machine index, broad stable regression set,
and general artifact evolution.

### 4. fn-17 — Bounded model exploration and coverage

Select experiments deterministically from the existing small Nexus space. Support bounded exhaustive
enumeration and one semantic-coverage-guided policy that prioritizes an uncovered coordinate. Keep
pinned known regressions outside the exploration budget. Defer pairwise/t-wise families, symmetry
proofs, multiple source kinds, generalized resume state, and adaptive corpora.

Keep the uncovered-coordinate policy independent of the deferred symmetry, generalized reporting,
and resume machinery when reducing the existing mixed tasks.

### 5. fn-40 — Centralize PlannerPolicy constructors and default seed

After fn-17 renames Query's seed-rotated strategy to `seeded`, add canonical
`PlannerPolicy.shortest`, `PlannerPolicy.exhaustive`, and `PlannerPolicy.seeded` constructors. Use
seed `17` and Definition-ID tie-breaking for ordinary policies while keeping the public record as the
escape hatch for deliberate non-default seeds, breadth-first policies, and generic fixtures.

Migrate ordinary Umpire and Temporal callers, then refresh the complete canonical Query, Artifact,
generated-view, and checksum sets whose identity intentionally changes when seeds `23` and `29`
become `17`. Traversal for shortest and exhaustive policies must remain unchanged. Fn-40 follows
fn-17's strategy rename and should settle the ordinary policy surface before fn-33 builds the campaign
command around exploration.

### 6. fn-33 — Run model exploration campaigns with umpire-fuzz

Reduce the campaign to a serial bounded `umpire-fuzz run` command that asks the Lean-owned
exploration layer for candidates, executes them through the existing runner/Run Evaluation path, and
reports semantic coverage and exhaustion honestly. Defer concurrency, leases, crash-safe campaign
state, and resume.

### 7. fn-22 — Deterministic replay, model minimization, and reviewed promotion

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

The completed fn-41 and fn-44 boundaries provide the finite-target and accepted-trace prerequisites.
The remaining simplicity-specific dependency shape is:

```text
fn-50 -> fn-43 -> {fn-48, fn-49, fn-51}
fn-47 -----------> fn-48
```

Fn-47 retains its own inventory dependencies, including fn-44. Fn-48 may later feed the deferred
fn-26, fn-29, and fn-30 governance work without pulling those specs into the prototype queue.

## Prototype verification gate

The local and ordinary-CI portions of this gate are complete. Before starting P3, fn-28 must produce
fixed-profile staging and canary dry-run records that name the same normal artifact byte hash, format
identity, and Behavior Fingerprint as those completed runs. The staging record must also prove
bounded execution, isolation, and complete cleanup. Run `flowctl validate --all --json` after the
remaining dependency edits and require no retained task to depend on a deferred task.

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

The existing Flow IDs remain the source of history. The remaining roadmap has four conceptual
delivery tracks:

1. **Centralize configuration authoring:** fn-42 hard-cuts Callback and Matching declarations to the
   checked `ConfigUseSpec` seam.
2. **Complete the portability gate:** fn-28 binds the completed local and CI subject to fixed-profile
   staging and a canary dry-run without adding a control plane.
3. **Explore, standardize policy authoring, replay, and promote:** fn-5, fn-17, fn-40, fn-33, and
   fn-22.
4. **Harden and shorten model authoring without gating the prototype:** fn-50 settles the remaining
   finite-machine seam; fn-43 deepens ordinary authoring; fn-48, fn-49, and fn-51 then centralize
   Known Gaps and Observation structure and remove repetitive record literals.

The next prototype decision point is completion of fn-28. Keep P3 deferred until the fixed staging
and canary dry-run evidence makes the end-to-end experience portable and inspectable.
