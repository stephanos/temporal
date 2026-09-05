# Umpire 4 delivery order

Status: current. The numbered queue contains only retained open work. Entries explicitly marked
historical, superseded, or deferred are planning context and are not runtime recommendations.

## Goal

Build on the completed Case Runtime with approachable model authoring, one separate Nexus2
authoring experiment, and removal of tooling that no retained consumer needs. Keep semantic
authority in Lean and generic execution in Go without reopening the retired scenario-specific
runtime.

## Completed foundation

[Fn-64](../.flow/specs/fn-64-umpire-case-runtime.md) delivered the two-call Case Runtime, the
Lean-produced async Nexus proof, the reviewed
[legacy removal ledger](../.flow/artifacts/fn-64-umpire-case-runtime/task8-migration-ledger.md), and
the independent six-class facade corpus with the full regression gate. That work is a completed
foundation for the open queue, not an active delivery step.

## Current boundaries

- A versioned `Case` contains one bounded `Program` and one deterministic `Contract`; Lean is a
  Producer and is not a runtime dependency.
- The public execution sequence is `PrepareCase(case, profile)` followed by
  `PreparedCase.Run(ctx, host)`. Prepared Cases support isolated sequential and concurrent Runs.
- Scheduling, recording, Slot storage, Monitor construction, typed request construction, and
  response projection remain internal execution responsibilities.
- Temporal server authority supplies authorized descriptors and transports prepared unary calls.
  Temporal worker authority owns SDK workflow, activity, Nexus-handler, reservation, and
  activation-cancellation behavior.
- Contracts consume declared typed Run Observations. Private Slots do not become evidence, and
  randomized runtime identities do not require response digests.
- New scenarios and properties remain Case data. They must not add scenario-specific Go adapters,
  public Monitor selection, or a replacement resident executor service.
- Downstream specs rewritten for the Case Runtime require a fresh plan review. A review of an older
  execution architecture does not approve a rewritten plan.

## Active execution queue

### 1. fn-62 — Make ordinary Temporal model authoring approachable

**Depends on:** completed fn-58; no open spec dependency.

**Deliver:** reduce Lean ceremony for ordinary finite Targets, checked Properties, Behaviors,
Queries, plans, and Observations while preserving the existing languages and checker authority.
Deepen the finite machine and planner adapters, make identities, sources, Limits, transition
contracts, Observation builders, and optional Known Gaps explicit, and migrate the ordinary Nexus
walkthroughs plus one newcomer example.

**Keep:** explicit semantic choices, stable Definition IDs and Behavior Fingerprints, deterministic
plans, public imports, trust inventories, and existing failure boundaries. Do not add another
authoring language or redesign expert and Experimental paths.

[Open fn-62 spec](../.flow/specs/fn-62-make-ordinary-temporal-model-authoring.md)

### 2. fn-65 — Design and prototype approachable Nexus2 feature authoring

**Depends on:** no open spec dependency. Coordinate overlapping authoring changes with fn-62;
fn-65 remains a separate experiment rather than a replacement.

**Deliver:** prototype ordinary feature authoring under `model/Temporal/Feature/Nexus2`, starting
with the current Nexus lifecycle and a separate cancellation/completion race. Compare typed
transition records and focused syntax over the checked Target, Property, Behavior, and Query
interfaces. Keep outcomes, identities, Query forms, typed Limits, diagnostics, and coverage or
conflict analysis explicit.

**Defer from this prototype:** caller closure, general temporal expressions, System/Evidence
integration, live execution, and a generated product-owner view.

[Open fn-65 spec](../.flow/specs/fn-65-design-and-prototype-approachable.md) ·
[Nexus2 design](../model/Temporal/Feature/Nexus2/DESIGN.md)

### 3. fn-66 — Remove unused Umpire tooling after the cutovers

**Depends on:** fn-62 only. Fn-64 is completed foundation and is not an open dependency.

**Deliver:** inventory remaining `tools/umpire` packages and commands against retained runtime,
Producer, authoring, generation, regression, and downstream consumers. Remove only proven-unused
code, exclusively owned tests and fixtures, and obsolete build, workflow, and documentation
references. Resolve ambiguous ownership before deletion and preserve concrete retained contracts.

**Boundary:** extend the deletion accounting without repeating fn-64 cleanup. This work does not
depend on fn-60.

[Open fn-66 spec](../.flow/specs/fn-66-remove-unused-umpire-tooling-after.md)

### 4. fn-60 — Deepen handwritten Lean canonical JSON construction

**Depends on:** no open spec dependency; excludes the completed `Umpire.Property` partition.

**Priority:** optional and distinct from fn-62 and fn-66. It is not a prerequisite for either.

**Deliver:** make `Umpire.Json` the single typed construction and exact-rendering interface for
handwritten Core Limit JSON and the handwritten Target, Behavior, Query, Space, Exploration,
Observation, and Implementation Link formatters. Preserve exact bytes, ordering, escaping,
newlines, metadata, fingerprints, imports, trust inventories, performance, and comments.

**Boundary:** this is handwritten canonical JSON construction and is unrelated to generated
protobuf encoding. Do not add parsing, protocol changes, generated Lean work, drift gates, or CI.

[Open fn-60 spec](../.flow/specs/fn-60-deepen-authored-lean-canonical-json.md)

## Current dependency graph

```text
fn-58 (completed) -> fn-62 -> fn-66
fn-65 (independent Nexus2 authoring experiment; coordinate with fn-62)
fn-60 (optional handwritten JSON maintenance; no dependency on or from fn-62/fn-66)

fn-40 (completed) + fn-64 (completed) -> fn-33
fn-5  (completed) + fn-64 (completed) -> fn-22
fn-48 (completed) + fn-64 (completed) -> fn-26 -> fn-29
```

## Downstream Case Runtime planning

These specs use `Case`, `PreparedCase`, `Run`, and `Verdict`. Each needs a fresh Case Runtime plan
review before implementation. Fn-22 remains blocked on redesign, and fn-26 must establish its
receipt and publication boundary before fn-29 is reviewed.

| Spec | Dependency | Status | Next action |
| --- | --- | --- | --- |
| [fn-33](../.flow/specs/fn-33-run-serial-bounded-semantic-exploration.md) | Completed fn-40 and fn-64 | Structurally valid draft | Review whole-Case candidates, the serial coordinator, lost iterations, semantic coverage, and bounded 10x behavior. |
| [fn-22](../.flow/specs/fn-22-deterministic-replay-semantic.md) | Completed fn-5 and fn-64 | **MAJOR_RETHINK; do not implement** | Separate candidate identity from violation equivalence, prove the negative Case before reduction, add explicit offline semantic replay, and retain the Case-native checked-promotion seam. |
| [fn-26](../.flow/specs/fn-26-local-qualification-receipts-and-staged.md) | Completed fn-48 and fn-64 | Structurally valid draft | Review exact offline Case/Profile/Host/Run/Verdict admission, receipt multiplicity, idempotent publication, and the rule that assessment never creates or replays a Run. |
| [fn-29](../.flow/specs/fn-29-bounded-production-canary-execution-and.md) | fn-26; completed fn-48 and fn-64 | Draft waiting for fn-26 | After fn-26 ships, review external canary policy and credentials, serial Runs, leases, lost Runs, reconciliation without redispatch, and publication. |

Fn-33 remains a bounded serial exploration bridge. Fn-22 must use Contract-relative violation
equivalence and preserve exact candidate identities. Fn-29 keeps canary policy, credentials,
leases, recovery, reconciliation, and publication outside Umpire.

## Removed from the active queue

The following entries are explicitly historical and superseded. Flow has no cancelled child-task
state, so their old tracker tasks may remain `todo`; that representation does not make them active.

| Spec | Historical reason |
| --- | --- |
| [fn-14](../.flow/specs/fn-14-milestone-a-pilot-baseline-and-lean.md) | Milestone A pilot baseline; its own architecture reconciliation marks it historical and forbids using it as a roadmap gate. |
| [fn-61](../.flow/specs/fn-61-simplify-the-umpire-go-execution-surface.md) | Superseded by fn-64 after the `PortableTestPlan` resident-executor boundary was discarded. |
| [fn-63](../.flow/specs/fn-63-consolidate-umpire-go-tests-into-golden.md) | Superseded; any broader test consolidation requires a new Case Runtime proposal and independent-oracle design. |

## Deferred work

These retained specs are explicitly deferred scheduling context, not prerequisites for the active
queue.

| Spec | Defer until / reason |
| --- | --- |
| [fn-15](../.flow/specs/fn-15-standalone-api-and-config-input-catalogs.md) | Platform completeness is needed beyond the first proven model family. |
| [fn-23](../.flow/specs/fn-23-veil-toolchain-compatibility-and.md) | Optional checker adoption becomes valuable. |
| [fn-24](../.flow/specs/fn-24-lean-native-verification-receipts-and.md) | A receipt/profile platform is justified; current bounded Contract checking is sufficient. |
| [fn-25](../.flow/specs/fn-25-optional-callerclosure-veil-binding-and.md) | A second verification backend is justified; caller closure remains historical. |
| [fn-30](../.flow/specs/fn-30-release-evidence-graph-and-manual.md) | Real Claim Assessment evidence exists for later release governance. |
