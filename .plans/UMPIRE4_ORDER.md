# Umpire 4 delivery order

## Goal

Build and hard-cut over to the standalone, data-driven Umpire Case Runtime specified by
`fn-64-umpire-case-runtime` and `.plans/UMPIRE_CASE_RUNTIME_DESIGN.md`. The first complete proof is
one Lean-produced async Nexus-success Case; caller closure and the scenario-specific
`PortableTestPlan` path are not part of the replacement.

The runtime proves these boundaries:

- Umpire's versioned `Case` contains one bounded `Program` and one deterministic `Contract`; Lean is
  one Producer, not a runtime dependency.
- `PrepareCase(case, profile)` performs static validation once and returns an immutable
  `PreparedCase` that can drive isolated sequential or concurrent Runs.
- Execution and verification are independent modules joined by a narrow Monitor callback. A safety
  violation unconditionally stops ordinary execution; bounded liveness fails only when its recorded
  horizon closes.
- The Temporal Host separates server and worker authority. The server side can invoke any authorized
  unary protobuf RPC dynamically; the worker side uses only replay-safe SDK clients and APIs.
- Typed assignments and projections use bounded protobuf payload paths such as `foo.bar.baz`.
  Scenario behavior and checks remain data in the Case rather than new Go adapters.
- The initial Contract consumes only declared authoritative server-history Observations. Runtime
  timestamps, UUIDs, and other randomized payload data do not require response digests.

The numbered execution queue contains only open retained work. Numbering expresses delivery
priority; explicit dependencies are shown below.

## Active execution queue

### 1. fn-64 — Umpire Case Runtime

**Depends on:** no open spec dependency.

**Scope:** define the public Case IR and standalone Go preparation, execution, verification, and
Temporal Host APIs; compile and run the async Nexus Case; then remove the legacy
`PortableTestPlan`, property-specific checker, Run Evaluation, scenario Nexus adapter, and
caller-closure path. No compatibility reader, replacement executor service, streaming RPC support,
production canary controller, or replay/audit digest is included.

The validated implementation waves are:

1. `fn-64.1` — define the versioned Umpire Case IR.
2. `fn-64.2` — implement typed admission and immutable preparation.
3. In parallel: `fn-64.3` — deterministic Contract evaluation; `fn-64.5` — Temporal server Host;
   `fn-64.6` — Temporal worker Host.
4. `fn-64.4` — generic DAG scheduling, typed dataflow, and Run recording.
5. `fn-64.9` — unconditional abort, cleanup, terminal precedence, and reusable PreparedCase Runs.
6. `fn-64.7` — compose the Host and compile/execute the async Nexus Case.
7. `fn-64.8` — remove the legacy Umpire execution path.
8. `fn-64.10` — reconcile normative docs, generated artifacts, and regression gates.

### 2. fn-59 — Centralize Umpire artifact copies

**Depends on:** completed fn-52 artifact admission and runtime contracts; no open spec dependency.

**Scope:** make the internal artifact-model package the single defensive-copy authority for the
schema-valid artifact graph, then migrate artifact admission and runtime output to that small
root-oriented interface. Preserve copy-on-input and copy-on-output isolation, nil and empty values,
admitted Raw Evidence scalar values, original encoded bytes, checksums, diagnostics, public APIs,
and existing comments. Do not add validation, generic copying for invalid dynamic values, schema or
generated-output changes, or user-facing documentation.

### 3. fn-60 — Deepen authored Lean canonical JSON construction

**Depends on:** no open spec dependency; excludes the completed `Umpire.Property` partition.

**Scope:** make `Umpire.Json` the single typed construction and exact-rendering interface for Core
Limit JSON and the handwritten Target, Behavior, Query, Space, Exploration, Observation, and
Implementation Link formatters. Preserve public interfaces, validation and diagnostic precedence,
field and element order, escaping and newline policy, canonical metadata, Artifact bytes, Behavior
Fingerprints, imports, trust inventories, performance characteristics, and existing comments. Do
not add parsing, validation hardening, alternate compatibility helpers, generated Lean or protocol
changes, drift verification, or CI work.

### 4. fn-62 — Make ordinary Temporal model authoring approachable

**Depends on:** completed fn-58's frozen Property facade and fn-60's canonical-JSON cleanup across the
overlapping handwritten Lean surfaces.

**Scope:** reduce the Lean-specific ceremony required to author an ordinary finite Target, checked
Property, Behavior, Query, plan, and Observation while preserving Umpire's existing languages and
checker authority. Deepen `FiniteMachine` and finite planner-kernel adapters; add explicit
family-rooted identities, source locations, named Query Limits, transition-contract helpers, typed
Observation builders, and optional model-owned Known Gaps; migrate the ordinary Nexus lifecycle and
operation walkthroughs; and add one checked newcomer example. Preserve explicit semantic choices,
public imports, Definition IDs, Behavior Fingerprints, deterministic plans, artifact identity except
for reviewed source/Known Gap deltas, trust inventories, failure boundaries, and existing comments.
Do not add another authoring language, infer providers or Model Outcomes, hide checker-success
evidence, redesign the expert `TransitionKernel` or Experimental fault-space paths, or add broad
generated-API drift and CI coverage.

## Downstream work after the Case Runtime

The existing fn-22, fn-26, fn-29, and fn-33 specifications predate the hard cutover. They depend on
fn-64 but must be replanned around `PreparedCase`, `Run`, and `Verdict` before implementation; they
must not retain a dependency on the superseded `PortableTestPlan` execution model.

### fn-33 — Run model exploration campaigns with umpire-fuzz

**Depends on:** completed fn-40's ordinary PlannerPolicy surface and fn-64, followed by replan.

**Scope:** a serial bounded `umpire-fuzz run` command that asks the Lean-owned exploration layer for
candidates, prepares each Case through the standalone API, executes it to a `Run` and `Verdict`, and
reports semantic coverage and exhaustion honestly. The replan must replace its old resident-executor
and Run Evaluation assumptions.

**Deferred:** concurrency, leases, crash-safe campaign state, and resume.

### fn-22 — Deterministic replay, model minimization, and reviewed promotion

**Depends on:** fn-5's checked review-only promotion source and fn-64, followed by replan.

**Scope:** consume the fn-21 violation through the Case Runtime's `Run` and `Verdict`, reproduce it
exactly, and try every applicable authored reduction in fixed order while preserving the same
violation. The exact control may be irreducible; its diagnostic EvidenceCore must still omit one
labeled non-responsible evidence fact without rewriting admitted evidence artifacts. Semantic
replay remains distinct from concrete rerun.

Reduction continues until every remaining applicable authored edit conclusively fails to preserve
the violation. Only a complete `minimized` or `irreducible` result may feed one checked review-only
Lean regression proposal.

**Deferred:** SDK history replay, generic reducers, campaign orchestration, and automatic regression
installation.

### fn-26 — Local qualification receipts and staged profiles

**Depends on:** fn-64 and its retained policy/evidence prerequisites, followed by replan.

**Scope:** retain the qualification and Claim Assessment intent, but bind receipts to the new Case,
Run, and Verdict identities rather than the removed Run Evaluation result.

### fn-29 — Bounded production canary execution and Claim Assessment

**Depends on:** fn-64 and its retained canary prerequisites, followed by replan.

**Scope:** prepare and validate a Case once, then execute that immutable PreparedCase repeatedly
under separately owned canary orchestration. The replan must preserve the server/worker Host split
and must not move canary policy, credentials, leases, or recovery into Umpire.

The current dependency shape is:

```text
fn-64.1 -> fn-64.2
fn-64.2 -> {fn-64.3, fn-64.5, fn-64.6}
fn-64.3 -> fn-64.4 -> fn-64.9
{fn-64.4, fn-64.5, fn-64.6, fn-64.9} -> fn-64.7 -> fn-64.8 -> fn-64.10

fn-60 -> fn-62
fn-59 (independent)

fn-64 -> replan {fn-22, fn-26, fn-29, fn-33}
```

Completed fn-48 may still feed fn-26 and fn-29 during their replans. Fn-30 remains later release
governance work.

## Case Runtime verification gate

Fn-64 closes only when the async Nexus Case passes through the public standalone APIs, one prepared
Case supports isolated repeated Runs, live and offline Contract evaluation agree, server and worker
authority remain separate, safety-stop and cleanup precedence are race-tested, and the legacy path
is gone. The focused unit, race, integration, model-build, regression, import-format, and lint gates
listed in the fn-64 spec must pass. Dependency edits must pass `flowctl validate --all --json`.

The former portable self-hosted prototype gate was satisfied for the now-superseded
`PortableTestPlan` architecture. It remains historical evidence only and does not unblock downstream
work against the new runtime.

## Removed from the current queue

### Close as superseded

- **fn-14 — Milestone A pilot baseline and Lean-first usability decision.** Its own architecture
  reconciliation marks it as historical and prohibits using it as a roadmap gate.
- **fn-61 — Simplify the Umpire Go execution surface.** It deepens the discarded
  `PortableTestPlan` resident-executor boundary; fn-64 replaces that boundary instead.
- **fn-63 — Consolidate Umpire Go tests into golden scenarios.** Its ownership and baselines depend
  on fn-61 and the legacy runtime, so any useful consolidation must be reconsidered after fn-64.

### Defer until the Case Runtime demonstrates value

- **fn-15 — Standalone API and config input catalogs.** Platform completeness, not prototype proof.
- **fn-23 — Veil toolchain compatibility and adoption gate.** Optional checker investigation.
- **fn-24 — Lean-native verification receipts and canonical replay.** Receipt/profile platform;
  the Case Runtime's bounded checking is sufficient for its first proof.
- **fn-25 — Optional CallerClosure Veil binding and canonical replay.** Second verification backend.
- **fn-30 — Release evidence graph and manual authorization.** Release governance after real
  Claim Assessment evidence exists.
