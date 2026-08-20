# Umpire3 plan from candidate to the full Umpire vision

Status: Umpire3 is an independent local candidate, not a qualified realization of the full vision.

This document is the authoritative plan for the work remaining after `UMPIRE_LEAN.md`. It is based
on a fresh comparison of:

- every goal in `UMPIRE_VISION.md`;
- the retained Umpire2 implementation and root tests;
- the current Umpire3 Lean model, generated artifacts, compiler, runtime, drivers, evidence,
  campaigns, authoring API, CLI, and release protocol; and
- the independently named Umpire2/Umpire3 root test files.

The decisive distinction is semantic depth. Umpire3 now has most of the framework-shaped pieces,
but a generated catalog entry, a boolean in a model state, a deterministic campaign test, or a
profile declaration is not proof that the corresponding Temporal behavior has been modeled,
observed, or qualified. The remaining milestones replace those approximations with relational
models, live mechanisms, learned runtime data, and deployment-bound evidence.

## 1. Non-negotiable decisions

### 1.1 Umpire3 remains fully independent

Umpire3 owns its model, compiler, protocol, runtime, evidence, participants, Temporal adapters,
faults, exploration, campaigns, replay, canary, qualification, and authoring surface. It must not
import Umpire1, Umpire2, or `common/testing/umpire`.

A shared package may be extracted later only when both implementations independently expose the same
stable responsibility, have symmetric contract tests, and still pass when built separately. No
remaining milestone depends on such extraction.

### 1.2 Lean is the semantic authority

Lean owns product states, relations, transitions, properties, permitted interference, refinements,
monitor meaning, exploration identities, and protobuf interpretation. Go may execute generated
programs and collect evidence; it must not silently supply semantic rules missing from Lean.

### 1.3 Protobuf shape is generated; meaning is explicit

Selected Temporal protobuf messages are imported automatically from descriptors. Umpire3 must never
copy message definitions into Lean by hand. A generator may recover presence, oneofs, maps, repeated
fields, enum numbers, nested types, and dependencies. It must not infer that a field establishes
identity, causality, ownership, lineage, completion, or safety. Those interpretations remain explicit
Lean definitions with coverage obligations and conformance fixtures.

### 1.4 Evidence and budgets fail closed

Missing capabilities, ambiguous identity, unproved lineage, clock-only cross-process order, lost
history, contradictory sources, an unrealized fault, incomplete cleanup, or an exhausted bound cannot
produce a conforming claim. A time budget is called hard only across a killable process boundary.

### 1.5 Candidate and qualified are distinct release states

Local proofs and tests can produce a candidate. Qualification also requires real CI,
remote-deployment, public-gRPC-only, and approved production-canary receipts bound to the exact model,
descriptor, experiment, build, configuration, evidence, fault, and cleanup digests. No local test may
manufacture those receipts.

## 2. Current audited state

### 2.1 What is implemented

The candidate has the following real foundations:

- a standalone Lean kernel for transitions, executable equivalence, refinements, bounded exploration,
  generated monitors, composition, and parity manifests;
- a generated semantic catalog and strict versioned experiment/result formats;
- deterministic sparse compilation with bounded partial-order completion and runtime identity binding;
- a selected protobuf descriptor pipeline with 24 root messages, 84 transitive messages, 21 enums,
  and 397 fields at the current descriptor hash;
- explicit semantic, opaque, ignored, and rejected field dispositions plus generated conformance
  fixtures and typed fuzz domains;
- typed Go regression constructors, domain handles, source-located diagnostics, explain, run, replay,
  campaign, and qualify commands;
- generated SDK participant programs and process-isolated participant crash, restart, and response
  modes;
- public-history and independent server-history evidence sources normalized into one evidence model;
- real local mechanisms for ordinary Nexus completion, completion before the start response,
  cancellation retry with an observed external drop, start-to-close timeout, callback completion after
  caller closure, a shared callback handler, bidirectional Nexus/Activity links, continuation, reset,
  task-queue routing, and Workflow Task ownership fencing;
- fail-closed evidence qualification, exact violation identity, deterministic minimization, strict
  replay bundles, and ordinary-regression promotion;
- a killable canary process boundary with bounded cleanup and recovery metadata; and
- a candidate release protocol that cannot become qualified while external gates remain.

### 2.2 Side-by-side root test inventory

Migration is additive. The originals are preserved under explicit Umpire2 names and Umpire3 has
independent copies:

| Retained Umpire2 | Independent Umpire3 |
| --- | --- |
| `tests/umpire2_test.go` | `tests/umpire3_test.go` |
| `tests/umpire2_probe_test.go` | `tests/umpire3_probe_test.go` |
| `tests/umpire2_regress_test.go` | `tests/umpire3_regress_test.go` |

The generated migration ledger contains 28 explicit behavior contracts and checks both AST
locations. Helper-only regressions are explicit Umpire3 tests, including their HSM/CHASM variants.
The Umpire3 copies do not call Umpire2.

This is inventory completeness, not yet full behavioral equivalence. Several Umpire3 copies execute
real Temporal traffic but retain a smaller generated scenario than the corresponding Umpire2 probe.
P5 below closes that gap contract by contract.

At the time of this audit, the retained Umpire2 exploration and generated-completion probes fail in
isolation because the current root environment observes none of their expected CHASM transitions.
Only symbol names changed in the retained files. This baseline problem must be diagnosed separately;
Umpire3 must not weaken or overwrite the originals to make the combined target green.

### 2.3 Vision matrix

| Vision goal | Current state | Work required for full completion |
| --- | --- | --- |
| Single model for software behavior | Partial | Replace the broad assurance boolean model with compositional relational product/system models and honest parity states |
| Specify and verify known regressions | Partial | Give all 28 contracts exact model targets, mechanisms, negative controls, and evidence equivalence |
| Deterministic regression plans | Implemented locally | Preserve canonical source/digest/order under every new generator and driver |
| Same tests local/CI/CICD/canary | Protocol implemented, execution incomplete | Collect exact-digest receipts from each real profile |
| White-box and gRPC-only black-box | Adapters implemented locally | Qualify every eligible property through both independent evidence routes on real deployments |
| Developer-friendly model/test API | Promising, not yet fantastic | Reduce imports and boilerplate, generate deeper domain handles, add one-command environments and measured UX gates |
| Exploration for unknown bugs | Mechanism demonstrated | Explore the real model denominator and find a non-preseeded product defect |
| First-class faults | Algebra and selected real faults implemented | Learn call footprints, realize the full safe fault matrix, and qualify fault effects |
| Non-linear steps/unknown IDs | Implemented in compiler/runtime | Exercise multi-entity aliasing, late binding, rebinding rejection, and remote replay at scale |
| Kitchensink/programmed workers | Participant language implemented | Prove every command/response/failure/crash mode against real SDK execution |
| Guided exploration | Small bounded examples | Cover lifecycle edges, parameters, schedules, topologies, and symmetry reductions from model obligations |
| Distributed processes/clock skew | Evidence vocabulary implemented | Run skew/partition/restart cases and prove causality without wall-clock ordering |
| Guided automatic fuzzing | Deterministic campaign kernel | Feed it learned semantic/wire coverage and runtime footprints, then enforce corpus novelty and promotion |

## 3. Protobuf awareness and how to use it

### 3.1 Current answer

Yes, the Umpire3 model is aware of selected Temporal protobuf messages. The import is automatic and
descriptor-derived, not handwritten. `model/Temporal/API/selection.json` selects the roots. The API
generator computes their recursive closure, emits Lean wire types and Go protocol metadata, records
the descriptor digest, produces field dispositions and conformance fixtures, and checks drift.

That is useful because request shape is part of the behavior boundary:

- presence versus default can change validity;
- unknown enum values are legitimate wire inputs and important fuzz cases;
- oneofs constrain legal combinations;
- durations, retry policies, links, callbacks, reset requests, and Activity/Nexus requests contain
  values that drive lifecycle semantics; and
- descriptor drift should create an explicit modeling obligation rather than silently changing a
  test generator.

Importing the entire Temporal API would be counterproductive. It would enlarge hashes and fuzz spaces
without adding product meaning. Selection must remain use-case driven and recursively complete.

### 3.2 What is still missing

The generated wire model does not yet drive enough live invalid or boundary requests. Current
reflected string/duration probes check generated descriptors and interpretations but then execute a
generic live behavior. They do not prove that the mutated protobuf was the request accepted or
rejected by the real Temporal endpoint.

P2 must connect the pipeline end to end:

```text
descriptor selection
  -> generated legal/illegal field domains
  -> typed request mutation
  -> actual public or task-protocol driver
  -> captured request digest and field provenance
  -> Temporal response/history evidence
  -> Lean interpretation and generated property monitor
```

Every selected field must have one of four checked dispositions:

- **semantic** — interpreted in Lean and included in property/coverage obligations;
- **opaque** — preserved and digest-bound but not interpreted;
- **ignored** — excluded for a documented reason and protected by a drift test; or
- **rejected** — generated only for a negative validity test with an expected failure class.

No field may enter fuzzing merely because it exists. Values that can allocate excessive resources,
address external systems, carry secrets, or affect production scope require profile-specific safety
policies.

## 4. Umpire2 feature port ledger

Umpire2 is a behavioral reference, not a code dependency. The following capabilities are still
needed in Umpire3 unless explicitly marked complete.

| Umpire2 capability | Umpire3 state | Required disposition |
| --- | --- | --- |
| Sparse regression terms and automatic completion | Implemented | Preserve and deepen typed domain APIs |
| Deterministic plans and replay | Implemented | Add cross-profile digest equivalence and large-corpus tests |
| Rich entity inventory | Partial | Model Namespace, Task Queue, Workflow, Run, Workflow Task, Activity, Activity Execution, Nexus operation/worker, Update, Callback, and participant lifecycles relationally |
| Rich relation inventory | Partial | Model containment, ownership, hosting, delivery, continuation/reset lineage, callbacks, links, and persistence references as first-class relations |
| Lifecycle guards/effects | Partial/shallow | Replace assurance booleans with explicit legal/illegal transitions and non-vacuity witnesses |
| Product/system/refinement targets | Complete for narrow Nexus/Update/TaskAck slices | Add equivalent depth for every retained target |
| Known permitted and forbidden examples | Partial | Add both witnesses and mutation sensitivity for each property |
| Recovered/Degraded/Flagged outcome taxonomy | Missing | Define terminal and liveness semantics; stop mapping both degraded and flagged to the same negative control |
| Reflective invalid-request enumeration | Structural only | Drive descriptor-derived mutations through real APIs and gate field/variant coverage |
| Learned RPC/HTTP footprint | Static examples only | Capture real baseline calls and reconcile declared versus observed footprints |
| Fault-each-observed-call resilience | Missing | Schedule each eligible learned occurrence under bounded policy and classify recovery/degradation |
| Coverage over lifecycle edges | Small generic candidates | Derive the denominator from the Lean transition graph and cover all eligible Nexus edges/variants |
| Random plan generation | Tiny fixed candidates | Generate valid plans from model guards, selected protobuf domains, schedules, and topology holes |
| Risk/novelty prioritization | Kernel implemented | Use semantic transition, wire-field, fault-occurrence, evidence-source, and topology novelty |
| Causal trace/source/clock metadata | Implemented | Stress with contradictory sources, retention loss, skew, and replay |
| White-box telemetry observation | Server-history source implemented | Decide whether CHASM transition telemetry is required; if retained, fix and independently qualify it |
| Black-box public evidence | Implemented for current targets | Expand to every black-box-eligible property and remote deployment |
| Programmed kitchensink workers | Umpire3 participant implemented | Finish exhaustive real integration for commands, responses, failures, crash, and restart |
| Guided exploration templates | Implemented narrowly | Add constraints, symmetry, model-derived coverage goals, and complete/exhausted truthfulness at realistic breadth |
| Campaign/minimize/replay/promote | Implemented mechanically | Prove discovery on runtime-learned inputs and an unknown defect |
| Canary envelopes and cleanup | Implemented locally | Integrate an approval-bound production realizer and collect real receipts |
| Multiple formal backends | Intentionally not ported | Lean remains sufficient only if equivalent properties and counterexamples are demonstrated |

Do not port Umpire2's duplicated registries, global instrumentation coupling, hand-maintained message
shapes, shallow compatibility wrappers, or a second formal backend merely to claim tool parity.

## 5. Root behavior fidelity ledger

The current 28-entry ledger should be extended with a machine-readable fidelity level:

- **exact** — the same user-visible Temporal mechanism and outcome are driven and independently
  evidenced;
- **semantic-equivalent** — a different mechanism intentionally realizes the same proved product
  property;
- **partial** — real Temporal traffic exists but the original dimensions or oracle are reduced; or
- **inventory-only** — the named test exists but has not executed its behavior.

Current exact or near-exact local mechanisms include ordinary Nexus completion, completion before
start response, cancellation retry/drop occurrence, shared callback handler behavior, start-to-close
timeout, callback after caller closure with server rejection, Nexus/Activity bidirectional links,
continuation, reset, routing, ownership fencing, and generated Workflow/Update traffic.

The important remaining partial behaviors are:

- rejected Nexus start: currently an early negative-control response, not an unknown-endpoint or
  equivalent real start rejection;
- reflected required-field and duration variants: wire structure is checked but the malformed request
  is not the request driven live;
- degraded and flagged: both collapse to violating controls instead of distinct lifecycle/liveness
  dispositions;
- learned footprint and resilience: selection is based on supplied footprints rather than a captured
  baseline call stream;
- exploration: two generic candidates do not cover the original Nexus lifecycle graph;
- coverage-guided and randomized probes: candidates and coverage are supplied rather than learned
  from real plans and calls; and
- seeded mutation discovery: it proves the pipeline against an approved mutation, not an unknown
  product defect.

No partial entry may be represented as equivalent in the parity ledger or as passed release evidence.

## 6. Developer experience: Umpire2 versus Umpire3

### 6.1 Current comparison

| Developer task | Umpire2 | Umpire3 now | Umpire3 target |
| --- | --- | --- | --- |
| Write a common regression | Concise sparse terms | Concise generated terms, but usually three imports plus an environment factory | One domain import and one test facade; environment selected by test harness/profile |
| Discover valid actions | Go APIs and existing examples | Generated constructors and several domain handles | State-aware fluent handles with editor-visible compatible operations only |
| Express unknown runtime IDs | Existing refs/trace machinery | Typed symbols and runtime grounding | Same, with inference for ordinary single-entity flows and visual explain output |
| Add a protobuf case | Reflection helpers | Selection manifest + generated projection + explicit interpretation | One command that reports uncovered fields and creates source-located obligations |
| Run locally | Familiar Go test | Go test or CLI plus explicit factory/config | `RequireRegression` works in package tests; one CLI command for standalone execution |
| Understand a failure | Test logs and reports | Structured claim, explain, evidence, cleanup, artifact, replay | One concise causal narrative plus opt-in full graph/timeline |
| Replay CI | Seed/report dependent | Strict bundle and replay command | One artifact link/command with automatic compatible profile selection |
| Move to remote/canary | Separate harnesses | Common protocol exists, operator wiring remains | Same semantic source and command, with capability preflight and approval workflow |

Umpire3 is already stronger in semantic provenance, deterministic compilation, fail-closed evidence,
and replay. It is not yet unequivocally better for a developer writing a routine test. The normal path
still exposes framework concepts that should be inferred or owned by a deep module.

### 6.2 Fantastic test-author UX

The target API should make the correct path shorter than Umpire2:

```go
func TestCancellationRetry(t *testing.T) {
    operation := nexus.Operation("operation")

    umpire3test.RequireRegression(t,
        nexus.Regression("cancellation-retry", operation,
            operation.CancelWithRetry(nexus.DropFirstCancel()),
            operation.CancellationWins(),
        ),
    )
}
```

The test should not contain capability lists, checkpoint IDs, dependency edges, entity bindings,
monitor names, descriptor paths, evidence classes, cleanup code, hashes, or a local cluster factory.
The generated domain module and test harness infer them and fail before allocation when they cannot.
Advanced APIs remain available for explicit partial orders, profile selection, faults, and exploration.

Required UX work:

- generate lifecycle/state-aware domain handles, not only flat action constructors;
- combine the structural and domain packages behind a stable author package where Go permits it;
- make local test profile selection automatic while keeping explicit remote/canary configuration;
- produce typo suggestions and source spans for actions, values, properties, bindings, and profile
  capabilities;
- render `Explain` as a compact plan/identity/evidence graph as well as JSON;
- include a one-command replay line in every failure and CI artifact;
- add golden examples for common Workflow, Activity, Nexus, Update, callback, lineage, routing, and
  ownership tests;
- measure first-test authoring, failure diagnosis, and replay in CI/documented exercises; and
- enforce a ten-minute fresh-contributor path for authoring and locally running a simple regression.

## 7. Dependency-ordered milestones

### P0 — Make every claim honest — implemented

**Goal:** ensure manifests and docs distinguish modeled, generated, executed, and externally
qualified behavior.

**Deliverables**

- Add fidelity and evidence-level fields to the parity and migration ledgers.
- Mark shallow or partial assurance targets `not-yet-implemented` rather than `equivalent`.
- Permit incomplete parity in a `candidate` manifest, but reject it in a `qualified` manifest.
- Bind every evidence row to a proof, focused test, integration result, or external receipt class.
- Update implementation/support documents from generated truth rather than prose-only status.
- Record the retained Umpire2 telemetry baseline failure without changing its expected behavior.

**Acceptance tests**

- changing a partial parity row to equivalent without the required anchors fails generation;
- changing candidate to qualified with any partial row or external gate fails validation;
- stale docs/manifests fail a generated status check; and
- cached results, constructed profiles, and skipped tests cannot count as evidence.

**Failure handling:** validation errors name the exact goal, row, missing anchor, and permitted next
state. No migration or fallback upgrades the claim automatically.

**Implemented outcome:** parity ledger v2 records fidelity and evidence level, with nine
exact/equivalent local-integration rows and the remaining 11 assurance rows explicitly
`not-yet-implemented`, partial, and exploration-incomplete. The promoted rows are Task acknowledgement,
the relational `NexusOperationClosure`, `NexusActivityLinkConsistency`, and
`NexusOperationTimeoutSemantics`, `CallbackReferenceConsistency`, and
`CallbackResponseConsistency` properties, plus the reciprocal Nexus/Activity and two callback
integration targets.
Migration ledger v3 records 14 exact, 4
semantic-equivalent, and 10 partial live root behaviors. The assurance composition obligation is
pending. Candidate validation accepts those truthful gaps; qualified validation rejects pending
composition, incomplete or non-profile-qualified parity, partial migration fidelity, partial vision
evidence, and outstanding external gates.

### P1 — Replace the assurance umbrella with real relational models

**Progress:** five vertical slices are implemented. `NexusOperationClosure` models two operation
identities and their caller relation, every Workflow and operation terminal outcome, ordinary
completion, completion before the start response, task attempts, ownership epochs, persistence, and
caller closure. `NexusOperationTimeoutSemantics` models operation-to-evidence identity, configured
timeout kind, failure metadata, ordered history observation, and duplicate delivery. The reciprocal
Nexus/Activity slice models two operations, two Activities, independent observation, and both public
link directions. The two callback slices model two callback/operation/handler/delivery identities,
attachment and operation reference kind/value/order, accepted response fingerprints, idempotent
redelivery, conflicting responses, operation settlement order, and late-response terminality. Each
slice has product safety, system refinement/stuttering, bounded exploration,
reachable permitted and forbidden cases, mutation sensitivity, Lean monitor equivalence, and a live
SDK history oracle. Their property rows are exact/local-integration, and the reciprocal link target is
also exact/local-integration; both callback integration targets are exact/local-integration using
mechanism-qualified live receipts and public history. The broader `feature-nexus` and timeout integration targets remain
partial until their full lifecycle and exploration denominators are complete.

**Goal:** give every retained property the same semantic depth as the narrow Nexus, Update, and
TaskAck slices.

**Deliverables**

- Model entity identity, lifecycle state, attempts, ownership epochs, task queues, workflow runs,
  callbacks, Activities, Nexus operations, Updates, and persistence-visible references.
- Model containment, delivery, hosting, lineage, linking, callback, and terminal relations explicitly.
- Split product contracts, threatening system mechanisms, environmental interference, and refinements
  into deep modules with small interfaces.
- Replace state booleans that begin true or are only set true with derivations from relational state.
- Prove executable transition equivalence, safety, refinement/stuttering, monitor equivalence, and
  composition assumptions for every target.
- Add non-vacuity witnesses: permitted traces, forbidden traces, reachable antecedents, and a mutation
  that breaks each theorem or monitor.
- Generate the semantic coverage denominator from model transitions and property obligations.

**Acceptance tests**

- each property has at least one accepted and one rejected trace;
- removing a guard/effect/refinement premise fails a theorem, monitor vector, or mutation test;
- no safety theorem is discharged only because its antecedent is unreachable; and
- the parity ledger reaches equivalent only after product, system, refinement, monitor, and negative
  control anchors exist.

**Trade-offs:** more precise state increases proof and exploration cost. Keep models compositional and
project only the target-relevant state; do not weaken relations to make global exploration cheap.

### P2 — Drive descriptor-derived cases through real Temporal protocols

**Goal:** turn protobuf awareness into implementation conformance.

**Deliverables**

- Generate boundary and invalid values from selected field dispositions and explicit constraints.
- Build public gRPC, Workflow Task, Activity Task, Nexus task, callback, and history request drivers as
  required by each message family.
- Bind the exact serialized request digest and field mutation provenance into action evidence.
- Classify responses into modeled accepted, rejected, unsupported, timed out, and transport-failed
  outcomes without treating every error as a product violation.
- Implement the Recovered/Degraded/Flagged outcome taxonomy in Lean, monitors, Go results, and root
  tests.
- Gate semantic/opaque/ignored/rejected field coverage and descriptor drift.

**Acceptance tests**

- required-field absence, unknown enum numbers, duration boundaries, oneof conflicts, repeated/map
  boundaries, and selected link/callback/reset messages are actually sent to the intended endpoint;
- evidence identifies the mutated field, serialized digest, endpoint, response, and resulting history;
- a generator/runtime encoding mismatch fails cross-language conformance; and
- unsafe values are rejected before allocation or network use.

**Security:** generated fuzz values are size bounded, redacted, profile checked, and prohibited from
introducing arbitrary addresses, namespaces, task queues, payloads, or credentials.

### P3 — Learn and reconcile runtime footprints

**Goal:** make resilience and fault selection depend on observed execution rather than supplied call
lists.

**Deliverables**

- Capture normalized gRPC and Nexus HTTP occurrences for each fault-free baseline.
- Preserve protocol, service, route, occurrence, direction, namespace, participant, attempt, interval,
  and causal references without retaining secrets.
- Reconcile declared model footprint against observed calls with explicit allowed noise.
- Derive eligible fault targets from learned calls while excluding setup/client-entry calls whose
  failure merely prevents the behavior under test.
- Persist the learned footprint and reconciliation digest in replay artifacts.

**Acceptance tests**

- a changed internal route produces deterministic drift;
- undeclared/missing calls are reported separately from allowed background traffic;
- identical semantic runs normalize to identical footprints despite ports, IDs, and timestamps; and
- every selected fault has positive occurrence and cleanup evidence.

**Scalability:** stream and hash occurrences, cap retained samples per normalized identity, and spill
large artifacts through the existing bounded artifact store.

### P4 — Complete model-guided exploration and fuzzing

**Goal:** explore the real model and live protocol surface, not a tiny fixed candidate list.

**Deliverables**

- Generate plans from legal model transitions, parameter holes, selected protobuf domains, partial
  orders, response modes, participant failures, and bounded topologies.
- Add symmetry reduction for interchangeable entities/participants and partial-order reduction for
  independent actions.
- Cover every eligible Nexus lifecycle edge and then every retained target's semantic denominator.
- Run fault-each-learned-occurrence resilience with risk and semantic novelty prioritization.
- Keep deterministic serial/parallel corpora for a seed and record every budget drop/omission.
- Minimize actions, order, resources, bindings, values, participant programs, faults, and topology
  while preserving exact qualified violation identity.

**Acceptance tests**

- the original 17-edge Nexus exploration denominator is represented or explicitly dispositioned;
- serial and parallel runs select the same corpus/digests for one seed;
- exact exhaustion is reported complete when the frontier is empty and incomplete otherwise;
- seeded mutations at model, interpretation, adapter, evidence, and integration layers are found; and
- promoted source compiles and fails against the mutation after campaign-only state is removed.

**Ten-times load:** bounded queues and deterministic admission prevent candidate explosion; coverage
summaries are mergeable, and each worker receives an independent candidate/profile allocation.

### P5 — Make all 28 root contracts behaviorally exact

**Goal:** preserve the Umpire2 files and make each Umpire3 copy an independently executing equivalent
at the user-visible boundary.

**Deliverables**

- Preserve every existing root Umpire test under an explicit `tests/umpire2_*_test.go` name and create
  an independently implemented `tests/umpire3_*_test.go` copy for every one; never overwrite, wrap,
  import, or weaken the Umpire2 original.
- Keep all six named files and the layout/import guard, and make the migration generator fail when a
  Umpire2 test function has no Umpire3 behavior contract or either file loses its expected package
  independence.
- Add exact fidelity criteria to every migration contract: model target, mechanism, variants,
  parameters, faults, expected lifecycle disposition, evidence, and cleanup.
- Replace rejected-start, reflected variants, degraded/flagged, learned-footprint, resilience,
  exploration, coverage-guided, and randomized approximations using P1–P4.
- Compare semantic verdict, terminal outcome, grounded identities/lineage, relevant payload/link
  digests, fault realization, and cleanup—not private implementation traces.
- Diagnose the retained Umpire2 transition-telemetry baseline independently and keep the original
  assertions intact.
- Run Umpire2 and Umpire3 sequentially in the combined target to avoid global test-infrastructure
  contention while preserving separate results.

**Acceptance tests**

- every one of 28 entries is exact or explicitly approved semantic-equivalent; no partial entry
  remains;
- both AST locations and independent implementation/package imports are checked;
- HSM/CHASM variants run where declared;
- a deliberately corrupted Umpire3 driver fails the Umpire3 entry and parity gate without changing
  Umpire2; and
- both suites pass uncached in the same revision.

### P6 — Make authoring and operation fantastic

**Goal:** make the safe, explainable path the shortest path.

**Deliverables**

- Implement the state-aware generated domain API in section 6.2.
- Provide a default local test harness and profile registry while keeping dependency injection for
  specialized environments.
- Generate model/protobuf source links and correction suggestions into diagnostics.
- Add compact human output for plan, evidence, counterexample, cleanup, and replay, backed by the
  stable diagnostic JSON.
- Publish executable examples and migration recipes for all major domains.
- Add IDE/build compile tests that prevent invalid combinations where Go's type system can express
  them.
- Measure and gate first-test, diagnosis, and replay budgets.

**Acceptance tests**

- ordinary cancellation retry needs one domain import, the test facade, one resource, one behavior
  term, and one property term;
- no routine test specifies capabilities, checkpoints, dependencies, identities, or cleanup;
- every failure includes the first divergence, evidence reason, artifact path, and redacted replay
  command; and
- a fresh contributor can author and run the documented first regression in ten minutes.

**Complexity:** keep one deep authoring facade over compiler/runtime details. Avoid convenience
wrappers that merely mirror every lower-level type.

### P7 — Qualify real distributed profiles

**Goal:** execute one semantic source unchanged across the environments named by the vision.

**Deliverables**

- Run the candidate in local and CI profiles with retained artifacts.
- Deploy the least-authority remote participant/controller and execute against a real remote cluster.
- Build a public-gRPC-only binary with an import guard excluding server-internal observers.
- Add skew, partition, process crash/restart, delayed evidence, history pagination/retention, and
  contradictory-source cases.
- Integrate a deployment-owned, approval-bound production fault realizer with immutable allowlists.
- Emit and merge signed or deployment-attested receipts for the exact candidate hashes.

**Acceptance tests**

- the same experiment digest has local, CI, remote, gRPC-only, and approved canary receipts;
- black-box claims use only public APIs/tasks/history and satisfy the same generated predicates;
- no cross-process order is inferred from wall-clock timestamps alone;
- killed workers/controllers leave bounded, resumable cleanup records and no untracked resources; and
- credentials and raw customer payloads never enter semantic inputs, results, logs, receipts, or
  replay bundles.

### P8 — Discover and promote an unknown defect

**Goal:** prove exploration adds value beyond replaying known and deliberately seeded failures.

**Deliverables**

- Run bounded model/wire/schedule/fault/topology campaigns against real implementations.
- Triage a non-preseeded qualified counterexample and rule out adapter/evidence defects.
- Minimize it, replay it across the eligible profiles, and promote it to an ordinary typed regression.
- Record the product bug or model correction, fix, and mutation sensitivity.

**Acceptance tests**

- the violation was not selected by a fixed known candidate or an approved mutation hook;
- evidence establishes exact property, identity, lineage, ordering, implementation version, and
  realized fault;
- the promoted regression fails before and passes after the fix; and
- the corpus retains the discovery's semantic novelty identity without environment-specific noise.

If no defect is found within the approved budget, the campaign result is useful coverage evidence but
does not satisfy this milestone.

### P9 — Qualify Umpire3 and decide whether any seam is shared

**Goal:** close the vision with auditable evidence while preserving independence.

**Deliverables**

- Run generated drift, Lean, unit, integration, root parity, mutation, campaign, remote, black-box,
  canary, formatting, lint, and repository gates on one candidate.
- Resolve every vision row and parity target to machine-verifiable evidence.
- Validate all receipts against exact semantic and implementation digests.
- Change release status to qualified only after no partial or external-required row remains.
- Evaluate shared extraction candidates using symmetric APIs/tests and record an explicit decision.

**Acceptance tests**

- every conforming or violating result has complete qualified evidence and cleanup;
- every known regression is deterministic, independently executable, and replayable;
- no unsupported/skipped/profile-construction result counts as passing evidence;
- Umpire3 builds and passes without Umpire1/Umpire2/common-Umpire dependencies; and
- declining all extraction candidates does not block qualification.

## 8. Verification ladder

Each milestone starts with focused red tests and finishes, as applicable, with:

1. `make umpire3-gen` after intentional semantic or selected-descriptor changes;
2. `make umpire3-check-generated`;
3. `make -C tests/umpire3/model check`;
4. `go test -count=1 -tags test_dep ./tests/umpire3/...`;
5. `make umpire3-integration` in an eligible real local environment;
6. `make umpire3-root` for retained Umpire2 and independent Umpire3 root tests;
7. deterministic campaign/mutation/replay gates;
8. required remote, public-gRPC-only, and canary receipt validation;
9. `make fmt-imports` and `make lint-code`; and
10. repository `make unit-test` when the full resource envelope is available.

Every command is run uncached when producing release evidence. Resource exhaustion, unavailable
external infrastructure, current baseline failures, and skipped profiles are recorded as blockers,
not converted into passes.

## 9. End state

Umpire3 is complete when a developer writes one concise typed semantic regression and the system can:

- derive deterministic executable plans and late-bound identities from one compositional Lean model;
- automatically project the selected protobuf wire surface while requiring explicit proved meaning;
- run the same semantic source against local, CI, remote, public-gRPC-only, and approved production
  canary profiles;
- program real SDK participants, processes, failures, response modes, and cross-entity behavior;
- learn runtime call footprints, inject safe scoped faults, and establish causal evidence without a
  shared clock;
- explore and fuzz model, wire, schedule, fault, participant, and topology dimensions by semantic
  novelty;
- minimize, replay, and promote both known and newly discovered failures;
- independently reproduce every retained Umpire2 root behavior in the six-file side-by-side layout;
  and
- issue a qualified release whose proofs, tests, artifacts, and external receipts all bind to the
  same semantic and implementation digests.

The end state does not require extracting shared code. Umpire3 remains complete, testable, and
operable on its own; extraction is permitted only after independence has revealed a genuinely stable
seam.
