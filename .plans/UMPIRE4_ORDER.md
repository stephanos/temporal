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
- The same deterministic `ExperimentSpec` is usable locally, in CI, and through a disposable
  self-hosted cluster boundary.
- Lean compiles one closed per-test evaluation contract ahead of time, allowing a resident Go
  executor to make a bounded local canary decision without Lean; production deployment is not part
  of the prototype.
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

## Completed portability prerequisite

### fn-28 — Portable evaluation contract and disposable-cluster qualification

**Status (2026-09-02): complete.** The protobuf contract vocabulary, deterministic packing and
strict admission, Lean lowering, Go evaluation, Lean/Go parity fixtures, resident executor,
attached disposable-cluster adapter, bounded HTTP transport, and tagged no-Lean end-to-end proof
are complete. The proof reuses one `testcore.NewEnv` cluster and one resident executor for the
normal caller-closure and duplicate-delivery contracts, with fresh per-run isolation, explicit
bounded Evidence closure, independent detailed statuses, complete cleanup, and conservative local
`pass` and trustworthy uniqueness-only `fail` decisions. The fail-closed contract, Limit,
Evidence-closure, transport, cancellation, and resident-reuse matrix is complete.

Compile the exact selected Test, Observation, Implementation Link, Property clauses, Limits, Known
Gaps, and bindings into a closed protobuf evaluation contract. A resident Go executor with no Lean
runtime must admit the contract, run it, wait for explicit bounded Evidence closure, evaluate only
the bundled per-test semantics, and retain the detailed stage statuses while mapping them
conservatively to local `pass`, `fail`, or `inconclusive`.

Qualify that architecture with one tagged Go integration test using `testcore.NewEnv`. Keep one
disposable self-hosted cluster and one Go executor process alive while running the pre-generated
normal caller-closure and duplicate-delivery contracts with fresh run isolation. The normal contract
must pass locally and the negative control must fail for its expected uniqueness violation without
invoking Lean, Make, a nested Go test, or a per-verification process.

The local decision applies only to one exact contract. Whole-model validity, exhaustive coverage,
compiler correctness, cross-test claims, fleet scheduling, leases, persistence, crash recovery,
production deployment, release eligibility, and Claim Assessment remain offline or deferred.

## Remaining P3 — Exploration and regression lifecycle

### 1. fn-5 — Umpire discovery, promotion, and Artifact evolution

Reduce this spec to two capabilities:

- coherent `list` and `explain` output for the retained Nexus declarations and examples;
- one checked, review-only promotion path for the minimized duplicate-delivery failure.

Defer the generic semantic graph, generated glossary, machine index, broad stable regression set,
and general artifact evolution.

### 2. fn-17 — Bounded model exploration and coverage

Select experiments deterministically from the existing small Nexus space. Support bounded exhaustive
enumeration and one semantic-coverage-guided policy that prioritizes an uncovered coordinate. Keep
pinned known regressions outside the exploration budget. Defer pairwise/t-wise families, symmetry
proofs, multiple source kinds, generalized resume state, and adaptive corpora.

Keep the uncovered-coordinate policy independent of the deferred symmetry, generalized reporting,
and resume machinery when reducing the existing mixed tasks.

### 3. fn-40 — Centralize PlannerPolicy constructors and default seed

After fn-17 renames Query's seed-rotated strategy to `seeded`, add canonical
`PlannerPolicy.shortest`, `PlannerPolicy.exhaustive`, and `PlannerPolicy.seeded` constructors. Use
seed `17` and Definition-ID tie-breaking for ordinary policies while keeping the public record as the
escape hatch for deliberate non-default seeds, breadth-first policies, and generic fixtures.

Migrate ordinary Umpire and Temporal callers, then refresh the complete canonical Query, Artifact,
generated-view, and checksum sets whose identity intentionally changes when seeds `23` and `29`
become `17`. Traversal for shortest and exhaustive policies must remain unchanged. Fn-40 follows
fn-17's strategy rename and should settle the ordinary policy surface before fn-33 builds the campaign
command around exploration.

### 4. fn-33 — Run model exploration campaigns with umpire-fuzz

Reduce the campaign to a serial bounded `umpire-fuzz run` command that asks the Lean-owned
exploration layer for candidates, executes them through the existing runner/Run Evaluation path, and
reports semantic coverage and exhaustion honestly. Defer concurrency, leases, crash-safe campaign
state, and resume.

### 5. fn-22 — Deterministic replay, model minimization, and reviewed promotion

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

The completed fn-41, fn-44, and fn-50 boundaries provide the finite-target, accepted-trace, and
CallerClosure migration prerequisites.
The remaining simplicity-specific dependency shape is:

```text
fn-43 -> {fn-48, fn-49, fn-51}
fn-47 ----------------> fn-48
```

Fn-47 retains fn-20 as its completed hard dependency. The completed fn-44 accepted-trace migration
and fn-45 plan-authority reconciliation are prerequisite provenance rather than readiness edges: all
of their tasks are done and their completion reviews are SHIP, while spec landing remains a later
lifecycle step. Fn-48 may later feed the deferred fn-26, fn-29, and fn-30 governance work without
pulling those specs into the prototype queue.

## Prototype verification gate

The local, ordinary-CI, and portable self-hosted portions of this gate are complete. Fn-28 produced
deterministic protobuf contracts bound to the same normal artifact byte hash, format identity, and
Behavior Fingerprint as the completed local and CI runs. Its tagged disposable-cluster proof shows
bounded execution, explicit Evidence closure, local evaluation without Lean, fresh run isolation,
and complete cleanup while reusing one resident Go process and cluster. This completed prerequisite
unblocks P3. Continue to run `flowctl validate --all --json` after dependency edits and require no
retained task to depend on a deferred task.

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
  the prototype retains only a no-Lean local decision over one portable per-test contract.
- **fn-30 — Release evidence graph and manual authorization.** Release governance after real
  Claim Assessment evidence exists.

## Preferred consolidation

The existing Flow IDs remain the source of history. The remaining roadmap has two conceptual
delivery tracks:

1. **Explore, standardize policy authoring, replay, and promote:** fn-5, fn-17, fn-40, fn-33, and
   fn-22.
2. **Harden and shorten model authoring without gating the prototype:** with the fn-50
   finite-machine seam complete, fn-43 deepens ordinary authoring; fn-48, fn-49, and fn-51 then
   centralize Known Gaps and Observation structure and remove repetitive record literals.

The next prototype work is P3. Completed fn-28 is its portability prerequisite, not an entry in the
remaining queue.
