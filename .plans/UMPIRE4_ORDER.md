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
- Lean compiles closed per-Test evaluation contracts and model-bound portable plans ahead of time,
  allowing a resident Go executor to make bounded local decisions without Lean; production
  deployment is not part of the prototype.
- One preprogrammed SDK participant resolves late-bound Nexus identifiers during execution.
- Evidence conclusions use causal or source-local ordering rather than synchronized clocks, proven
  with deliberately skewed timestamps.
- The existing authored variation space and first-class fault support guided exploration that
  prioritizes an uncovered Model Coordinate and retains an Exact Replay and reviewed proposal.

The numbered execution queue contains only open retained work. Each spec keeps the reduced scope
below and depends only on retained or completed prerequisites; numbering expresses delivery priority,
not additional hard dependencies.

## Refactoring and cleanup queue (non-prototype-gating)

These specs preserve public behavior while deepening remaining implementation hotspots. They build
on completed cleanup work rather than reopening its semantics. The Lean Property partition and
canonical-JSON cleanup both refresh shared architecture-document anchors; if they run concurrently,
fn-60's final documentation task follows fn-58's final documentation task. Their production file
surfaces remain disjoint because fn-60 excludes `Umpire.Property`. The Go artifact-copy cleanup is
independent and changes no user-facing documentation. The Go execution-surface simplification starts
only after fn-59 and fn-60, and becomes the execution boundary consumed by the remaining P3 work.

### 1. fn-58 — Partition the Property language implementation

**Depends on:** completed ordinary Property authoring and Model Coordinate cleanup.

**Scope:** separate authoring vocabulary, typed checking and canonicalization, capability-limited
trace projection, and clause evaluation into an acyclic internal module chain behind the unchanged
`Umpire.Property` facade. Preserve all Property errors, canonical identity, Limits, clause meaning,
agreement theorems, fingerprints, trace semantics, and trust inventories.

### 2. fn-59 — Centralize Umpire artifact copies

**Depends on:** completed fn-52 artifact admission and runtime contracts; no open spec dependency.

**Scope:** make the internal artifact-model package the single defensive-copy authority for the
schema-valid artifact graph, then migrate artifact admission and runtime output to that small
root-oriented interface. Preserve copy-on-input and copy-on-output isolation, nil and empty values,
admitted Raw Evidence scalar values, original encoded bytes, checksums, diagnostics, public APIs,
and existing comments. Do not add validation, generic copying for invalid dynamic values, schema or
generated-output changes, or user-facing documentation.

### 3. fn-60 — Deepen authored Lean canonical JSON construction

**Depends on:** no open spec dependency; excludes `Umpire.Property` while fn-58 is active. Sequence
fn-60.7 after fn-58.3 only when both documentation tasks are in flight.

**Scope:** make `Umpire.Json` the single typed construction and exact-rendering interface for Core
Limit JSON and the handwritten Target, Behavior, Query, Space, Exploration, Observation, and
Implementation Link formatters. Preserve public interfaces, validation and diagnostic precedence,
field and element order, escaping and newline policy, canonical metadata, Artifact bytes, Behavior
Fingerprints, imports, trust inventories, performance characteristics, and existing comments. Do
not add parsing, validation hardening, alternate compatibility helpers, generated Lean or protocol
changes, drift verification, or CI work.

### 4. fn-61 — Simplify the Umpire Go execution surface

**Depends on:** completed fn-52 caller-neutral gRPC portable plans, fn-59, and fn-60.

**Scope:** expose one root resident executor that accepts a caller-owned attached Temporal authority
and the existing model-provenance verifier, executes a protobuf `PortableTestPlan`, and returns an
`ExecutionResult` directly or through the generated gRPC service. Migrate generated and handwritten
end-to-end callers to that facade; internalize generated bindings, runtime contracts, the execution
state machine, Temporal/Nexus participants, Evidence closure, and portable evaluation; remove the
legacy HTTP and non-portable resident executor paths. Preserve exact admission, provenance,
runtime-slot, single-flight, cancellation, cleanup-poisoning, eventual Evidence closure, evaluation,
result-limit, and direct/gRPC status behavior. Keep the offline Run Evaluation capability distinct,
and do not change protobuf or Lean output, trust policy, concurrency, or cluster ownership.

**Deferred:** production deployment, executor fleets, queues, autoscaling, multi-run concurrency,
environment selection, credential distribution, and new transports.

## Complete P3 — Exploration and regression lifecycle

### 5. fn-33 — Run model exploration campaigns with umpire-fuzz

**Depends on:** completed fn-40's ordinary PlannerPolicy surface and fn-61's simplified execution
boundary.

**Scope:** a serial bounded `umpire-fuzz run` command that asks the Lean-owned exploration layer for
candidates, executes them through the simplified resident executor and retained offline Run
Evaluation path, and reports semantic coverage and exhaustion honestly.

**Deferred:** concurrency, leases, crash-safe campaign state, and resume.

### 6. fn-22 — Deterministic replay, model minimization, and reviewed promotion

**Depends on:** fn-5's checked review-only promotion source and fn-61's simplified execution
boundary.

**Scope:** consume the fn-21 violation, reproduce it exactly, and try every applicable authored
reduction in fixed order while preserving the same violation. The exact control may be irreducible;
its diagnostic EvidenceCore must still omit one labeled non-responsible evidence fact without
rewriting admitted evidence artifacts. Semantic replay remains distinct from concrete rerun.

Reduction continues until every remaining applicable authored edit conclusively fails to preserve
the violation. Only a complete `minimized` or `irreducible` result may feed one checked review-only
Lean regression proposal.

**Deferred:** SDK history replay, generic reducers, campaign orchestration, and automatic regression
installation.

The remaining dependency shape is:

```text
completed fn-52 + fn-59 + fn-60 -> fn-61
completed fn-40 + fn-61 -> fn-33
completed fn-5 + fn-61 -> fn-22
```

Completed fn-48 may later feed deferred fn-26, fn-29, and fn-30 without pulling them into the
prototype queue.

## Prototype verification gate

The local, ordinary-CI, and portable self-hosted gate is satisfied. Deterministic per-test contracts
preserve the normal artifact hash, format identity, and Behavior Fingerprint across environments;
the tagged disposable-cluster proof covers bounded no-Lean evaluation, explicit Evidence closure,
normal pass, expected uniqueness-only fail, fresh run isolation, resident reuse, and cleanup. The
decision remains local to one exact contract; whole-model validity, cross-test claims, fleet control,
production deployment, release eligibility, and Claim Assessment remain outside this gate. P3 is
unblocked. Dependency edits must pass `flowctl validate --all --json`, and retained tasks must not
depend on deferred work.

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
- **fn-29 — Bounded production canary execution and Claim Assessment.** Starts only after fn-52 and
  fn-61 plus a demonstrated vertical slice; the prototype retains only a no-Lean local decision
  over one portable per-test contract.
- **fn-30 — Release evidence graph and manual authorization.** Release governance after real
  Claim Assessment evidence exists.
