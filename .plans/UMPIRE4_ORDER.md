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
  `PreparedCase`; normal execution is the single root call `PreparedCase.Run(ctx, host)`, which can
  drive isolated sequential or concurrent Runs.
- Execution and Monitor composition are internal deep modules. The prepared Contract supplies the
  authoritative verifier; alternate Hosts, not arbitrary Monitors, are the public extension seam. A
  safety violation unconditionally stops ordinary execution; bounded liveness fails only when its
  recorded horizon closes.
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
Temporal Host APIs behind a two-call root facade; compile and prepare an orthogonal `GetSystemInfo`
Case, then compile and run the async Nexus Case; preserve fn-5's generic checked-promotion seam and
remove the legacy `PortableTestPlan`, property-specific checker, Run Evaluation, scenario Nexus
adapter, and caller-closure path through an explicit migration ledger. Add only a six-class
independent-oracle facade corpus, not broad test consolidation. No public Monitor replacement,
compatibility reader, replacement executor service, streaming RPC support, production canary
controller, or replay/audit digest is included.

The validated implementation waves are:

1. `fn-64.1` — define the versioned Umpire Case IR.
2. `fn-64.2` — implement typed admission, immutable preparation, and the root facade over internal
   execution contracts.
3. In parallel: `fn-64.3` — deterministic Contract evaluation; `fn-64.5` — Temporal server Host;
   `fn-64.6` — Temporal worker Host.
4. `fn-64.4` — internal generic DAG scheduling, typed dataflow, and Run recording.
5. `fn-64.9` — complete root `Run`, unconditional abort, cleanup, terminal precedence, and reusable
   PreparedCase Runs.
6. `fn-64.7` — compile/prepare the orthogonal `GetSystemInfo` Case, then compose the Host and
   compile/execute the async Nexus Case without runtime specialization.
7. `fn-64.8` — account for the legacy test surface, preserve fn-5's generic promotion primitives,
   and remove the legacy Umpire execution path.
8. `fn-64.10` — add the six-class conformance corpus and reconcile normative docs, generated
   artifacts, and regression gates.

### 2. fn-60 — Deepen authored Lean canonical JSON construction

**Depends on:** no open spec dependency; excludes the completed `Umpire.Property` partition.

**Scope:** make `Umpire.Json` the single typed construction and exact-rendering interface for Core
Limit JSON and the handwritten Target, Behavior, Query, Space, Exploration, Observation, and
Implementation Link formatters. Preserve public interfaces, validation and diagnostic precedence,
field and element order, escaping and newline policy, canonical metadata, Artifact bytes, Behavior
Fingerprints, imports, trust inventories, performance characteristics, and existing comments. Do
not add parsing, validation hardening, alternate compatibility helpers, generated Lean or protocol
changes, drift verification, or CI work.

### 3. fn-62 — Make ordinary Temporal model authoring approachable

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

Draft replans for fn-22, fn-26, fn-29, and fn-33 now use `Case`, `PreparedCase`, `Run`, and
`Verdict`. Their dependency graphs no longer reach the superseded `PortableTestPlan`, Run
Evaluation, resident-executor, or caller-closure execution model. These drafts are not
implementation-ready until they pass a fresh plan review against fn-64; any older `SHIP` receipt
predates the Case Runtime replan and is not evidence for the rewritten plan.

Current planning state:

| Spec | State | Next planning action |
| --- | --- | --- |
| fn-33 | Case Runtime draft; structurally valid | Review the full-Case bridge, serial coordinator, lost-iteration handling, and bounded 10x behavior. |
| fn-22 | `MAJOR_RETHINK` from Case Runtime review | Separate candidate identity from violation equivalence, move the negative-Case proof before reduction, add explicit offline semantic replay, and preserve a Case-native fn-5 promotion seam across fn-64 deletion. |
| fn-26 | Case Runtime draft; structurally valid | Review exact offline Case/Profile/Host/Run/Verdict admission, receipt multiplicity, and idempotent publication. |
| fn-29 | Case Runtime draft; structurally valid | Review only after fn-26's receipt boundary is stable; verify external canary ownership, lost Runs, and reconcile-without-redispatch. |

### fn-33 — Run model exploration campaigns with umpire-fuzz

**Depends on:** completed fn-40's ordinary PlannerPolicy surface and fn-64.

**Planning status:** draft replan; fresh Case Runtime plan review required.

**Scope:** a serial bounded `umpire-fuzz run` command that asks the Lean-owned exploration layer for
candidates, prepares each Case through the standalone API, executes it to a `Run` and `Verdict`, and
reports semantic coverage and exhaustion honestly. Each bridge candidate is a whole canonical Case;
preparation rejection, inconclusive work, cleanup uncertainty, and lost iterations receive no
coverage.

**Deferred:** concurrency, leases, crash-safe campaign state, and resume.

### fn-22 — Deterministic replay, model minimization, and reviewed promotion

**Depends on:** fn-5's checked review-only promotion source and fn-64.

**Planning status:** blocked on redesign after `MAJOR_RETHINK`; do not implement the current task
graph.

**Scope:** consume the fn-21 violation through the Case Runtime's `Run` and `Verdict`, reproduce it
exactly, and try every applicable authored reduction in fixed order while preserving the same
violation. The exact control may be irreducible; its diagnostic EvidenceCore must still omit one
labeled non-responsible evidence fact without rewriting admitted evidence artifacts. Semantic
replay remains distinct from concrete rerun.

“Same violation” must use a Contract-relative equivalence signature, not canonical Case identity.
Each reduction candidate retains its own exact Case/Program identity and checked lineage, while the
equivalence signature binds the unchanged Contract, Profile/catalog, violated terminal state,
responsible clause, and canonical supporting-Observation relation. Intentionally reducible Program
coordinates are excluded from that signature.

Fn-22's first executable gate is the Case-native negative control: compile the replacement Case,
evaluate the original closed Run offline, and reproduce the violation in two fresh Runs before
building the reducer or command. The review-only promotion task follows completed minimization; it
cannot run in parallel with the reduction-result contract.

Fn-64 may remove the caller-closure scenario binding, fixtures, and runtime, but its cutover must
leave fn-5's scenario-neutral checked-promotion primitives buildable. Fn-22 later supplies a new
Case-native expected-behavior anchor; it must not restore the deleted caller-closure model or import.

Reduction continues until every remaining applicable authored edit conclusively fails to preserve
the violation. Only a complete `minimized` or `irreducible` result may feed one checked review-only
Lean regression proposal.

**Deferred:** SDK history replay, generic reducers, campaign orchestration, and automatic regression
installation.

### fn-26 — Local qualification receipts and staged profiles

**Depends on:** completed fn-48 Known Gap policy and fn-64.

**Planning status:** draft replan; fresh Case Runtime plan review required.

**Scope:** retain the qualification and Claim Assessment intent, but bind receipts to the new Case,
preparation Profile/catalog, live Host, Run, and Verdict identities. Assessment is offline and never
creates or replays a Run.

### fn-29 — Bounded production canary execution and Claim Assessment

**Depends on:** fn-26, completed fn-48, and fn-64.

**Planning status:** draft replan; review after fn-26's receipt and publication contracts ship.

**Scope:** prepare and validate a Case once, then execute that immutable PreparedCase through bounded
serial Runs under separately owned canary orchestration. Preserve the server/worker Host split; keep
canary policy, credentials, leases, lost-iteration recovery, reconciliation, and publication outside
Umpire.

The current dependency shape is:

```text
fn-64.1 -> fn-64.2
fn-64.2 -> {fn-64.3, fn-64.5, fn-64.6}
fn-64.3 -> fn-64.4 -> fn-64.9
{fn-64.4, fn-64.5, fn-64.6, fn-64.9} -> fn-64.7 -> fn-64.8 -> fn-64.10

fn-60 -> fn-62

{fn-5, fn-64} -> fn-22
{fn-48, fn-64} -> fn-26
{fn-40, fn-64} -> fn-33
{fn-26, fn-48, fn-64} -> fn-29
```

Fn-30 remains later release-governance work.

## Case Runtime verification gate

Fn-64 closes only when the async Nexus Case passes through the public standalone APIs, one prepared
Case supports isolated repeated Runs through `PreparedCase.Run(ctx, host)`, and the orthogonal
`GetSystemInfo` Case compiles and prepares without Host I/O or runtime specialization. Live and
offline Contract evaluation must agree; server and worker authority remain separate; safety-stop
and cleanup precedence are race-tested; and the legacy path is gone only after the migration ledger
accounts for every deleted top-level Test/Fuzz and inherited failure identity. The six facade proof
classes must use independent expected results, exact bytes or named stable projections, and
interruption-safe temporary-tree generation. The full tagged live selector remains
`-run '^TestUmpire'`. The focused unit, race, integration, model-build, regression, import-format,
and lint gates listed in the fn-64 spec must pass. Dependency edits must pass
`flowctl validate --all --json`.

The former portable self-hosted prototype gate was satisfied for the now-superseded
`PortableTestPlan` architecture. It remains historical evidence only and does not unblock downstream
work against the new runtime.

Fn-64's deletion gate must also prove that the scenario-neutral fn-5 checked-promotion primitives
still build without caller-closure imports. This preserves a valid downstream seam without retaining
the old scenario, runtime, fixtures, or compatibility path.

## Removed from the current queue

### Tombstoned as superseded

Flow currently has no superseded/cancelled child-task state. These specs therefore remain open and
intentionally unready with historical `todo` tasks; that tracker representation is not an
implementation queue.

- **fn-14 — Milestone A pilot baseline and Lean-first usability decision.** Its own architecture
  reconciliation marks it as historical and prohibits using it as a roadmap gate.
- **fn-61 — Simplify the Umpire Go execution surface.** Its spec is tombstoned and its dependencies
  removed because fn-64 replaces the discarded `PortableTestPlan` resident-executor boundary.
- **fn-63 — Consolidate Umpire Go tests into golden scenarios.** Its spec is tombstoned and its
  dependencies removed; any useful consolidation requires a new Case Runtime proposal after fn-64.

### Defer until the Case Runtime demonstrates value

- **fn-15 — Standalone API and config input catalogs.** Platform completeness, not prototype proof.
- **fn-23 — Veil toolchain compatibility and adoption gate.** Optional checker investigation.
- **fn-24 — Lean-native verification receipts and canonical replay.** Receipt/profile platform;
  the Case Runtime's bounded checking is sufficient for its first proof.
- **fn-25 — Optional CallerClosure Veil binding and canonical replay.** Second verification backend.
- **fn-30 — Release evidence graph and manual authorization.** Release governance after real
  Claim Assessment evidence exists.
