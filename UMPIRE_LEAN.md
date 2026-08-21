# Umpire3 roadmap to the full Umpire vision

Status: active implementation plan, consolidated 2026-08-20.

This is the single forward-looking plan for Umpire3. It supersedes the completed bootstrap roadmap
and the separate verification-architecture plan. Completed milestones have been removed; this file
contains only enduring architectural constraints, remaining work, and the evidence required to call
Umpire3 complete against `UMPIRE_VISION.md`.

The existing `tests/umpire3` tree is the implementation baseline. Its generated catalog, typed
authoring facade, protobuf projection, exact explorer, Veil integration, temporal portfolio, native
certificate producer, campaign/replay pipeline, isolated canary, and side-by-side Umpire2/Umpire3
root tests are foundations, not unfinished milestones. They still count as complete only when the
final verification matrix proves that later work has not weakened them.

## 1. End state

Umpire3 is complete when one compositional Lean model family supports all of these paths without a
second semantic authority:

1. feature contracts and independently defined system mechanisms;
2. refinement, interference, safety, and temporal proofs;
3. exact finite exploration and checked certificates;
4. exact, native, and Lean-side Veil search through explicit proved views;
5. generated regressions and sparse developer-authored tests;
6. real Temporal execution under local, CI, remote, black-box, and authorized canary profiles;
7. typed evidence interpretation with established, violated, unknown, and conflict results;
8. deterministic campaign, minimization, replay, and promotion; and
9. a release assurance graph that makes every trust boundary and omission visible.

The outcome is not a claim that all of Temporal is proved correct. The terminal claim is narrower
and mechanically inspectable:

> Important Temporal contracts have explicit semantics; selected mechanisms are proved to refine
> them; independent engines search them; counterexamples replay through canonical Lean and real
> tests; live claims are evidence-qualified; and every green result exposes its scope and trust.

## 2. Non-negotiable decisions

### 2.1 Semantic authority

- Lean is the sole authority for behavior, properties, observations, composition, and checker views.
- “One model” means one typed `ModelFamily` graph, not one monolithic state machine.
- Feature and system models are independently defined. A system state or step must not contain a
  feature state, feature run, or proof that the feature transition already occurred.
- Refinement relations and action mappings live outside both models and must fail for an executable
  nearby mutation.
- Catalog entries, theorem names, hashes, ranks, and statuses are derived metadata. Strings and
  Booleans cannot establish a proof, completeness, preservation, or composition claim.

### 2.2 Lean/Go boundary

- Lean and model checkers remain offline build and test tools. They do not enter production request
  paths through FFI, subprocesses, or generated executable logic.
- Go owns bounded orchestration, Temporal API calls, evidence transport, cleanup, persistence, and
  process isolation. It does not restate a Temporal state machine or property rule.
- `tests/umpire3/protocol` is a strict, versioned transport boundary. It may represent an explicit
  Lean-generated `ExecutableView`, `FirstOrderView`, `TemporalView`, certificate, trace, or evidence
  program. It is not an independently authored semantic IR.
- A non-Lean portability adapter may validate and mechanically render a proved view. If it evaluates
  or replays a view for differential checking, that implementation is checked against canonical Lean
  vectors and cannot become the source of the claim.
- Veil is a Lean extension. An Umpire3 Lean command consumes the canonical model or a proved view and
  elaborates Veil declarations in the same Lean build. Go may isolate that build and transport its
  receipt, but it does not render Veil syntax. No intermediate Veil `.lean` source tree is generated
  or checked in; only source-bound backend results and receipts are materialized.
- TLC and Apalache are deferred portability adapters. They receive no new scope until Lean proofs,
  exact exploration, native certificates, Lean-side Veil, model migration, live observation, and
  developer UX satisfy their gates. They cannot block primary Umpire3 qualification; any result that
  is published from them must still meet the same provenance, bounds, and replay rules.
- Scenario authoring is a separate intent language. Compiling a requested action attempt must not
  silently equate the attempt with a successful abstract transition.

### 2.3 Explicit views and trust

- Arbitrary Lean is never silently translated. Every executable, finite, first-order, temporal,
  observation, and backend representation has a declared soundness or equivalence obligation.
- Proof-grade finite success requires collision-safe identity and a certificate accepted by the
  small Lean checker.
- Veil testing, trusted SMT, reconstructed proofs, finite lasso evidence, unbounded temporal proof,
  and live conformance remain different result classes. There is no generic `verified` Boolean.
- Every backend counterexample must replay through canonical Lean semantics before promotion or a
  semantic violation claim.

### 2.4 Observation and evidence

- Live adapters emit typed raw facts: source identity, clock domain, source-local position, entity
  identity, lineage, causal references, event fields, mechanism receipts, and explicit omissions.
- An adapter never emits the Boolean truth of the property it is supposed to test.
- Lean-generated programs normalize and interpret facts, then return `true`, `false`, `unknown`, or
  `conflict` with supporting fact identifiers.
- Absence establishes a result only after an authoritative evidence window is closed. Missing facts,
  ambiguous identity, incompatible clocks, conflicting sources, or an open window fail closed.
- Cross-source order follows causal references or a declared ordering guarantee, never timestamp
  comparison alone.

### 2.5 Protobuf boundary

- Selected Temporal protobuf messages are imported automatically and reproducibly from descriptors.
- Selection closes recursively over nested messages, enums, oneofs, maps, repeated fields, presence,
  and supported well-known types. Every selected field has an explicit disposition.
- Product meaning is handwritten and proved in Lean. A generator must not infer semantic identity,
  completion, ordering, or absence from field names.
- Go and Lean share generated conformance fixtures for the selected wire interpretation.
- Umpire3 imports only the bounded wire surface needed by a model family; it does not mirror all
  Temporal protobufs preemptively.

### 2.6 Independence and extraction

- Umpire3 must not import or wrap Umpire1, Umpire2, or `common/testing/umpire`.
- Umpire2 remains a behavioral reference and executable baseline, never a runtime oracle.
- The retained Umpire2 root tests and independent Umpire3 copies continue to run side by side.
- Shared extraction is considered only after Umpire3 is complete and two independent
  implementations demonstrate a stable, symmetric seam. Extraction is not required for completion.

## 3. Authoritative model-family shape

Each qualifying family must expose a coherent set of typed declarations:

```text
Feature
  behavior
  user-visible properties
  non-vacuity witnesses

System
  mechanism-only behavior
  shared requirements and guarantees
  sound and mutated executable views

Refinement
  relation
  action/event mapping
  safety or temporal simulation
  mutation rejection theorem

Observation
  typed evidence vocabulary
  generated interpretation program
  identity, lineage, order, and closure obligations

Targets
  exact finite/first-order/temporal views
  declared omissions and preservation evidence
  source and theorem provenance
```

A target projection is narrower than its model family. It selects a world, modules, property,
bounds, checker views, and omissions. An omission is either proved property-preserving or labeled
heuristic; a heuristic omission cannot support proof-grade completeness.

## 4. Remaining milestones

The milestones are ordered. A later milestone may add tests early, but it cannot be declared complete
while an earlier semantic dependency remains open.

### R2 — Finish model-family migration

**Goal:** apply independent feature/system semantics across every active catalog target.

**Work**

- Finish the active migrations for Workflow ownership, lineage, routing, speculative delivery,
  callbacks, Nexus closure/timeout/Activity links, Workflow progress, and Update lifecycle.
- Point experiments, catalog targets, composition, coverage, parity, and proof manifests only at the
  independent replacements once their gates pass.
- Give every family a reachable good outcome and a reachable mutated bad outcome; prove property
  non-vacuity rather than relying on a truth table over unreachable states.
- Give every system/feature pair a concrete mapped execution, safety or temporal simulation, exact
  executable view, and independent mutation that breaks the declared simulation.
- Demonstrate multiple system realizations refining one feature contract.
- Use the shared current-completion/task-delivery contract from at least two families through actual
  theorems, not matching metadata.
- Complete cross-feature compositions and prove at least one interference-preservation theorem.
- Retire legacy proof-by-construction models from active targets. Old source may remain temporarily
  only when clearly quarantined and excluded from qualification.

**Exit gate:** every catalog target resolves to independent typed models and non-vacuous evidence;
searching active system modules finds no embedded feature state or feature-run witness.

### R3 — Complete typed live observation

**Goal:** remove property truth from every production Temporal adapter.

**Work**

- Complete the four-layer seam: source adapter, source-specific normalizer, generated interpreter,
  and monitor/qualifier.
- Replace the remaining `Observation{Satisfied: ...}` implementations and property switches with
  `FactSession` implementations for SDK, Update, Workflow Task, callback, lineage, routing,
  ownership, progress, timeout, and link paths.
- Extend the small Lean observation language only where required to bind projected identities,
  compare bounded typed fields, follow lineage and causal edges, check source-local order, and close
  authoritative windows.
- Keep history events and mechanism receipts distinct; normalized event names must not disguise an
  adapter-computed final property verdict.
- Generate all programs and four-valued fixtures from Lean. Mutation tests must fail when Go maps a
  source field, event kind, identity, order, or closure incorrectly.
- Preserve evidence digests and the complete support set in stored runtime results.
- Make dual-history profiles compare independently sourced normalized facts without identifier
  collisions or loss of corroboration requirements.
- Demonstrate established, violated, unknown, conflict, missing closure, wrong identity, wrong
  lineage, clock ambiguity, and contradictory-source cases.

**Exit gate:** no production adapter implements property truth; all 18 current observations and every
future registered observation are interpreted by generated programs with Lean/Go differential
fixtures.

### R4 — Complete model-derived compilation and primary checker coverage

**Goal:** connect developer intent, model semantics, every primary checker, and live execution
without making Go a semantic compiler.

**Work**

- Define the relation between a requested live action attempt, its observed outcome, and zero or more
  abstract transitions. Suppressed, rejected, retried, and fault-intercepted attempts must not be
  rejected merely because the successful abstract transition is disabled.
- Replay every compiled path through the exact Lean-derived view before allocation while preserving
  valid live-only behavior and rejecting genuinely impossible intent.
- Give every qualifying finite target a nonempty exact executable view and generated coverage
  denominator for transitions, relations, properties, faults, observations, refinements, and
  evidence alternatives.
- Replace reduction attestations with checked symmetry/closure evidence, or label the reduction
  heuristic and withhold completeness.
- Replace Go-rendered, checked-in Veil modules with an Umpire3 Lean command that consumes the
  canonical model or a proved first-order view and elaborates Veil declarations directly. Remove the
  generated Veil source directory. Mutation-test the elaboration bridge against canonical states,
  transitions, traces, and explored counts.
- Keep exact and native checking as the smallest proof-carrying baseline. A Veil failure must not
  weaken or bypass their certificates, and a Veil success retains its distinct solver trust class.
- Enforce advertised state, time, memory, CPU, output, and tool bounds at the process boundary and in
  result validation.
- Normalize exact, Veil, native, Lean-temporal, and live counterexamples into one semantic trace with
  canonical Lean replay receipts.
- Feed normalized traces through the same campaign minimization, replay, bundle, and ordinary
  `RequireRegression` promotion path.
- Preserve shortest-witness, deterministic merge, collision safety, checkpoint/resume, and corrupt
  certificate failure tests while broadening target coverage.

**Exit gate:** derived artifacts are drift-free; every qualifying target has honest exact/native and,
where supported, Veil coverage status; the full migration ledger compiles; and checker/live traces
converge on one replay and promotion path. TLC and Apalache are outside this gate.

### R5 — Make the developer workflow exceptional

**Goal:** make Umpire3 strictly easier than Umpire2 for ordinary tests while exposing more assurance
to model authors.

**Test-author contract**

An ordinary test declares only:

- a generated target/property identifier;
- typed resources and symbolic identities;
- a sparse partial-order scenario;
- optional typed parameters, alternatives, and faults; and
- a profile selected by the test environment.

The author does not edit JSON, hashes, capabilities, protobuf descriptors, model modules, evidence
rules, participant plumbing, cleanup, or replay code. `umpire3test.RequireRegression` compiles,
preflights, executes, qualifies, cleans up, and reports source-located diagnostics.

**Model-author contract**

- One command builds a selected family, runs its Lean proofs, exact exploration, native certificate
  checks, supported Lean-side Veil jobs, mutations, fixtures, and derived-artifact drift checks.
- Diagnostics point to the originating Lean declaration, scenario term, unsupported profile
  capability, missing evidence selector, or failed replay edge.
- A generated dependency graph selects affected targets for PR checks; nightly jobs broaden worlds
  and backend portfolios without changing semantics.

**Remaining work**

- Eliminate residual protocol/generated-file knowledge from public test APIs.
- Add concise first-test, model-family, failure-diagnosis, backend-trust, and profile guides.
- Verify the UX budget with a first regression, a partial-order test, a runtime-bound identity, a
  typed fault, and a campaign-promoted regression.
- Ensure copied root Umpire3 tests use normal public Umpire3 APIs and real evidence, not compatibility
  shortcuts or migration-only helpers.
- Keep deterministic source diagnostics and byte-identical compiled intent for identical inputs.

**Exit gate:** representative tests are shorter or clearer than their Umpire2 equivalents, use no
generated plumbing, and run unchanged across every eligible profile.

### R6 — Finish real-mechanism parity and environment qualification

**Goal:** turn catalog parity into demonstrated Temporal behavior across deployment profiles.

**Work**

- Exercise Nexus, Workflow, Workflow Task, Activity, Update, Callback, ownership, routing, lineage,
  timeout, and cross-entity paths through the real Temporal mechanism selected by each family.
- Keep all retained Umpire2 root behaviors and independent Umpire3 counterparts enabled. Require
  qualified semantic agreement while allowing architecture-specific implementation differences.
- Replace nearest-mechanism substitutes where they weaken the original behavioral contract; record
  any intentional semantic-equivalence boundary explicitly.
- Run one immutable scenario digest under local in-process, CI cluster, remote deployment, and
  public-gRPC-only profiles. Profile differences may change realization/evidence digests, never the
  semantic intent.
- Obtain external build/configuration, isolation, observation, cleanup, and retention evidence rather
  than manufacturing qualification receipts locally.
- Demonstrate public-only and internal/white-box evidence for every eligible claim, or record a
  precise unsupported result when public evidence is insufficient.
- Exercise an authorized canary rehearsal with digest allowlists, least authority, redaction,
  process isolation, enforceable hard budgets, persisted recovery metadata, and resumable cleanup.
- Prove blocking prepare, execute, wait, observe, and cleanup paths cannot exceed profiles that claim
  hard execution bounds.

**Exit gate:** the release manifest contains genuine local, CI, remote, black-box, and authorized
canary evidence for every supported profile; unavailable authority remains unsupported, never
inferred.

### R7 — Full-vision convergence and release audit

**Goal:** prove the system is coherent rather than declaring success from individually green pieces.

**Work**

- Re-audit every goal in `UMPIRE_VISION.md`, every requirement below, every generated artifact, and
  every retained Umpire2 behavioral contract against current source and fresh results.
- Run semantic mutations through proof, exact exploration, native certificates, Lean-side Veil,
  Lean temporal checking, live evidence, minimization, replay, and promotion.
- Demonstrate coverage-guided exploration or fuzzing finding and minimizing an approved cross-layer
  mutation or genuine defect, with deterministic corpus selection.
- Run the 10x native-search benchmark and record state storage, certificate size, Lean checking time,
  peak memory, worker-count determinism, checkpoint/resume, and partial-publication recovery.
- Verify hostile artifact decoding, secret redaction, dependency isolation, bounded cardinalities,
  subprocess sandboxing, and recovery after controller or worker crashes.
- Publish support, limits, trust, authoring, modeling, operations, security, and incident-recovery
  documentation.
- Move the release from `candidate` to `qualified` only after all mandatory evidence is present.
- Evaluate extraction after qualification; do not delay completion on an optional shared seam.
- After primary qualification, evaluate TLC and Apalache as optional portability adapters. Keep them
  only when they add independent defect-finding value without introducing a second semantic model.

**Exit gate:** every final requirement has direct current evidence and no green result depends on a
stale artifact, cached-only test, authored attestation, hidden omission, or unavailable external run.

## 5. Result and trust model

The result vocabulary remains explicit:

| Result class | Meaning |
| --- | --- |
| `trace-witness` | A canonical replay accepted a violating finite trace |
| `tested-instance` | One bounded backend or live instance was exercised |
| `finite-exhaustive` | A collision-safe closure certificate was checked for the declared world |
| `bounded-safety` | Safety holds only through an explicit depth/resource bound |
| `invariant-proved` | Lean accepted an invariant proof with an inventoried axiom set |
| `temporal-proved` | Lean accepted a temporal theorem under named fairness assumptions |
| `refinement-proved` | Lean accepted the declared feature/system simulation |
| `implementation-conforming` | One live implementation run met a property under its profile/evidence set |
| `metadata-validated` | Artifact structure and references are internally valid; no semantic proof follows |
| `unknown` | Evidence or checking is incomplete, conflicting, unsupported, or exhausted |

Trust badges must distinguish kernel proof, checked certificate, reconstructed solver proof, trusted
solver, external testing, live conformance, and metadata-only validation. A stronger-sounding label
cannot be selected by authored status text.

## 6. Failure semantics

- Unsupported capabilities fail before resource allocation.
- Invalid scenarios, impossible abstract intent, unbound identities, and stale views fail compilation.
- Search exhaustion, timeout, memory pressure, cancellation, or partial checkpoints retain explicit
  termination reasons and never imply exhaustive success.
- Invalid, incomplete, or world-mismatched certificates fail closed.
- A backend crash cannot publish a final receipt transactionally.
- Observation loss, open windows, causal ambiguity, or contradictory facts yield unknown/conflict,
  not conformance.
- Cleanup always runs under an independent bounded authority. Primary and cleanup failures are both
  retained, with recoverable resource metadata.
- Production actions stop on digest drift, isolation uncertainty, evidence loss, budget exhaustion,
  or cleanup uncertainty.

## 7. Performance, scalability, complexity, and security

- Exact exploration favors correctness and certificates; replaceable native producers supply scale.
- Deterministic output must not depend on map iteration, scheduler order, worker count, or checkpoint
  boundaries.
- Worlds, descriptors, evidence, cardinalities, traces, output, and retained artifacts are bounded
  before allocation.
- Backend projects and tool revisions remain isolated and pinned; no toolchain becomes a runtime
  dependency.
- Sensitive or unbounded wire fields are omitted, redacted, or represented by explicit digests.
- Credentials never enter generated artifacts, command arguments, logs, replay bundles, or model
  checker input.
- Canary subprocesses receive least authority, scoped filesystem/network access, resource limits,
  and killable process groups.
- The deep public seams remain `ModelFamily`, `Explorer`, `Backend`, `ObservationInterpreter`,
  `Verifier`, `Runtime`, and `RequireRegression`; new surface area requires a demonstrated need.

## 8. Verification matrix

Fresh verification is required. Cached results, existence checks, or metadata consistency alone are
not evidence for a semantic claim.

### 8.1 Lean

- all umbrellas and selected export roots elaborate;
- executable initial/successor soundness and completeness;
- independent refinement and mutation rejection for every family;
- non-vacuous good and bad reachability;
- composition and interference proofs;
- explorer frontier/visited, shortestness, closure, and depth theorems;
- certificate corruption and wrong-world rejection;
- observation semantics and cross-language fixtures;
- theorem/source resolution with exact axiom inventories; and
- temporal proofs with every fairness assumption named.

### 8.2 Go and generators

- strict decode, unknown-field, hostile-size, digest, and source-binding tests;
- generated catalog, identifiers, facade, protobuf, monitors, observations, composition, parity,
  coverage, experiments, proofs, backend views/results, and migration ledger drift checks;
- scenario attempt/outcome and impossible-intent tests;
- exact explorer collision, nondeterminism, deadlock, depth, cancellation, and determinism tests;
- Lean-side Veil elaborator mutations and canonical trace replay;
- observation true/false/unknown/conflict, closure, lineage, order, and source-conflict tests;
- campaign novelty, minimization identity, deterministic merge, replay, and promotion tests;
- process timeout, CPU, memory, output, cancellation, cleanup, and crash-recovery tests; and
- layout/import independence tests.

### 8.3 Integration and operations

- focused real Temporal integration tests for every supported family;
- retained Umpire2 and independent Umpire3 root suites with `-tags test_dep`;
- remote and gRPC-only qualification jobs;
- authorized non-production canary rehearsal;
- `make umpire3-check`;
- relevant `go test -count=1 -tags test_dep` packages;
- formatting and `make lint-code`; and
- repository-wide `make unit-test` when feasible, with any resource limitation reported explicitly.

## 9. Definition of done

Umpire3 is complete only when all statements below have direct, current evidence.

### Semantic authority

- Every qualifying target resolves to typed feature, system, property, refinement, observation, and
  view declarations.
- Active system steps contain no feature state, feature run, or feature proof witness.
- Every feature/system pair has a non-vacuous good execution and an executable mutation that breaks
  refinement.
- Exported theorem names resolve; axiom inventories and source/artifact hashes are derived.
- Composition and parity claims are proof- or certificate-backed, not metadata-backed.
- `protocol` remains transport for Lean-derived views, never an independently authored model.

### Checking

- Every qualifying finite target has an exact executable view and derived coverage denominator.
- Proof-grade finite success comes only from a collision-safe checked certificate.
- Veil runs as a Lean-derived extension for supported targets with concrete, symbolic-trace, and
  invariant results carrying honest trust.
- Lean temporal results preserve the bounded/unbounded distinction. Optional TLC/Apalache results do
  the same whenever those adapters are enabled.
- Every backend counterexample replays through canonical Lean semantics.
- All search and backend resource limits are enforced and accurately reported.

### Feature and composition support

- Multiple system realizations refine at least one feature contract.
- A shared system guarantee supports at least two feature families through actual theorems.
- At least one composed target proves interference preservation.
- Every target omission is proved property-preserving or explicitly heuristic.
- Every current Umpire2 behavioral contract has an enabled independent Umpire3 realization with the
  required evidence level.

### Live conformance

- Production Temporal adapters emit typed raw evidence rather than property truth.
- Generated observation programs evaluate identically in Lean and Go.
- Absence, missing evidence, contradiction, identity ambiguity, and clock ambiguity cannot establish
  conformance.
- Public and internal sources retain their independent provenance and corroboration requirements.
- Checker and live counterexamples use the same normalized replay/minimization/promotion path.

### Developer and operational quality

- Test authors use the generated domain facade and `RequireRegression` without artifact plumbing.
- Model authors run a selected Lean proof/exact/native/Veil family check with one command and source
  diagnostics.
- PR checks select affected targets from generated transitive semantic dependencies.
- Nightly jobs scale independently and produce deterministic, resumable bundles.
- The same scenario digest runs in every eligible profile.
- Canary execution is digest-bound, process-isolated, least-authority, redacted, recoverable, and
  separately authorized.
- Every `UMPIRE_VISION.md` goal is linked to passing evidence in the qualified release manifest.

## 10. Source anchors

- Vision: `UMPIRE_VISION.md`.
- Semantic kernel and registrations: `tests/umpire3/model/Umpire3`.
- Feature, system, refinement, observation, composition, and target models:
  `tests/umpire3/model/Temporal`.
- Generated transport and trust vocabulary: `tests/umpire3/protocol`.
- Sparse authoring and model-aware compilation: `tests/umpire3/scenario` and
  `tests/umpire3/umpire3test`.
- Exact, Lean-side Veil, temporal, and native checking: `tests/umpire3/explore` and
  `tests/umpire3/model-checkers`.
- Live facts and Temporal realization: `tests/umpire3/observation`,
  `tests/umpire3/execution`, and `tests/umpire3/temporal`.
- Campaign, replay, process isolation, and canary: `tests/umpire3/campaign`,
  `tests/umpire3/replay`, `tests/umpire3/process`, and `tests/umpire3/canary`.
- Root parity: `tests/umpire2_test.go`, `tests/umpire2_probe_test.go`,
  `tests/umpire2_regress_test.go`, `tests/umpire3_test.go`,
  `tests/umpire3_probe_test.go`, and `tests/umpire3_regress_test.go`.
- Checked migration evidence: `tests/umpire3/migration/ledger.json`.

These sources are evidence and migration references. Except for ordinary Temporal infrastructure,
they do not authorize Umpire3 to depend on earlier Umpire implementations.
