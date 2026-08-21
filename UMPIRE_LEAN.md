# Umpire3 roadmap to the full Umpire vision

Status: active implementation plan, consolidated 2026-08-21.

This is the single forward-looking plan for Umpire3. It supersedes the completed bootstrap roadmap
and the separate verification-architecture plan. Completed milestones have been removed; this file
contains only enduring architectural constraints, remaining work, and the evidence required to call
Umpire3 complete against `UMPIRE_VISION.md`.

The existing `tests/umpire3` tree is the implementation baseline. Its generated catalog, typed
authoring facade, protobuf projection, exact explorer, Lean temporal checking, native certificate
producer, unified semantic traces, campaign/replay pipeline, isolated canary, primary-project Veil
dependency and Lean source declarations, family-scoped checks, TLA quarantine guardrails,
Veil source-authorship drift checks and evidence-only export/record targets, digest-bound release
qualification, retained coverage-guided mutation and 10x native performance/recovery reports,
developer-UX and clock-skew audits, a retained hostile-input,
isolation, and recovery audit, published operational documentation, and side-by-side
Umpire2/Umpire3 root tests are foundations, not unfinished milestones.
Any family that uses Veil still has to bind those Lean declarations to its canonical semantics.
These foundations count as complete only when the final verification matrix proves that later work
has not weakened them.

## 1. End state

Umpire3 is complete when one compositional Lean model family supports all of these paths without a
second semantic authority:

1. feature contracts and independently defined system mechanisms;
2. refinement, interference, safety, and temporal proofs;
3. exact finite exploration and checked certificates;
4. exact and native search, with optional Veil declarations used as Lean-native modeling and proof
   support inside the primary Lean project;
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
  program. It is not an independently authored semantic IR, a backend-neutral compiler IR, or an API
  designed around hypothetical future model checkers.
- No current plan item translates Umpire3 semantics into another modeling language. A future
  portability experiment may validate and mechanically render an explicit proved view only under a
  separately approved plan. If it evaluates or replays that view, the implementation must be checked
  against canonical Lean vectors and cannot become the source of the claim.
- Veil is a Lean library and embedded DSL, not a source-generation target, target language, or
  independent semantic backend.
  Umpire3 imports the pinned Veil dependency in the primary Lake project and authors Veil declarations
  as Lean source beside the owning model family. A supported family proves the relationship between
  that declaration and its canonical view. Lean elaboration or metaprogramming may remove repetitive
  declaration boilerplate, but neither Go, JSON, a backend-neutral IR, nor a text template generates
  Veil source. Go may isolate a Lean invocation and transport its source-bound result; it does not
  compile semantics into Veil. Veil is optional per family; qualification does not require adding a
  Veil declaration to families that already have sufficient Lean proof and checked-search evidence.
  Files exported from a Veil-owning family are source-bound JSON bindings, normalized results, or
  proof receipts. They are derived evidence about authored Lean declarations, never generated Veil
  programs. Generation and drift checks must treat every `Temporal/Veil/**/*.lean` file as an input
  that no generator may create or rewrite.
- TLA+, TLC, and Apalache are parked portability experiments, not planned Umpire3 work. Existing
  adapters remain quarantined from default generation, checks, qualification, release gates,
  developer setup, and transitive Lean targets. Do not extend, repair, or install them while executing
  R5 through R7. They may be deleted if quarantine costs ongoing maintenance. Any future activation is
  a separate post-qualification decision and must meet the same provenance, bounds, omission, and
  canonical replay rules.
- Scenario authoring is a separate intent language. Compiling a requested action attempt must not
  silently equate the attempt with a successful abstract transition.

### 2.3 Explicit views and trust

- Arbitrary Lean is never silently translated. Every executable, finite, first-order, temporal,
  observation, and backend representation has a declared soundness or equivalence obligation.
- Proof-grade finite success requires collision-safe identity and a certificate accepted by the
  small Lean checker.
- A Veil-authored declaration may use testing, trusted SMT, or reconstructed proof support, but those
  trust modes remain distinct from kernel proof, finite lasso evidence, unbounded temporal proof, and
  live conformance. There is no generic `verified` Boolean.
- Every checker counterexample must replay through canonical Lean semantics before promotion or a
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

Execution priority is R6 and R7. Veil remains optional Lean-native modeling and proof support, not a
parallel model compilation track. TLA+, TLC, and Apalache have no scheduled milestone, release gate,
default check, or required developer dependency.

The active checker portfolio is:

| Mechanism | Role in Umpire3 | Completion status |
| --- | --- | --- |
| Lean kernel proofs | Semantic and theorem authority | Required |
| Exact exploration and checked certificates | Small finite proof-carrying search | Required for qualifying finite targets |
| Native certificate producer | Replaceable scalable search with Lean-checked output | Required for qualifying scalable finite targets |
| Lean temporal checking | Safety/liveness under explicit assumptions | Required for qualifying temporal targets |
| Veil | Optional embedded Lean declarations and proof/search support in the primary Lake project | Family-scoped; never a generated backend |
| TLA+, TLC, and Apalache | Quarantined historical portability experiments | Excluded from completion and default tooling |

No remaining milestone creates a Veil generator, a Veil-specific semantic IR, or a second Lake
project. Work on Veil is limited to ordinary Lean declarations, checked canonical-model bindings,
honest trust classification, and family-scoped build/test ergonomics. Work on the TLA experiments is
limited to preserving their quarantine or deleting them if that is cheaper than maintaining it.

The current focus is R6: turn the completed authoring, model-family, copied-regression, embedded-Veil,
and TLA-quarantine work into independently retained deployment qualification.

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
  than manufacturing qualification receipts locally. Pin a reviewed Ed25519 authority per external
  profile, sign the exact candidate/result/evidence/environment binding outside the repository, and
  retain and reverify the signature in the qualified manifest.
- Demonstrate public-only and internal/white-box evidence for every eligible claim, or record a
  precise unsupported result when public evidence is insufficient.
- Exercise an authorized canary rehearsal with digest allowlists, least authority, redaction,
  a controller-pinned Ed25519 approval authority, process isolation, enforceable hard budgets,
  persisted recovery metadata, and resumable cleanup.
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
- Run semantic mutations through Lean proof, exact exploration, native certificates, Lean temporal
  checking, live evidence, minimization, replay, and promotion. For a family that owns a Veil
  declaration, include that declaration and its canonical binding in the same mutation audit.
- Move the release from `candidate` to `qualified` only after all mandatory evidence is present.
- Evaluate extraction after qualification; do not delay completion on an optional shared seam.

**Exit gate:** every final requirement has direct current evidence and no green result depends on a
stale artifact, cached-only test, authored attestation, hidden omission, or unavailable external run.

### Out of scope — TLA+, TLC, and Apalache

TLA+, TLC, and Apalache are not part of R5 through R7 or Umpire3's definition of done. Keep existing
experimental code quarantined from default generation, CI, release, installation, Lean dependencies,
family selection, Go package sweeps, and `make umpire3-check`; delete it if quarantine becomes a
maintenance burden. Do not add a mise task or install bootstrap for these tools. Reconsidering this
portability track after qualification requires a separate plan and explicit approval.

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
- Checker dependencies and tool revisions remain pinned and build-isolated; Veil shares the primary
  Lean project without becoming a runtime dependency.
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
- Veil-authored declaration/binding mutations and canonical trace replay for families that use the
  library, all within the primary Lean project;
- a no-generated-Lean guard proving that export and result-recording commands cannot rewrite authored
  Veil declarations;
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

TLA+, TLC, and Apalache availability or results are not prerequisites for any verification gate in
this roadmap.

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
- Any Veil declarations used by a qualifying family are ordinary source in the primary Lean project,
  with checked bindings to canonical views and concrete, symbolic-trace, and invariant results
  carrying honest trust.
- Veil export and recording commands produce evidence artifacts only; the source declarations remain
  developer-authored Lean inputs.
- Lean temporal results preserve the bounded/unbounded distinction.
- Every checker counterexample replays through canonical Lean semantics.
- All search and backend resource limits are enforced and accurately reported.
- A retained and freshly repeated 10x native benchmark records state and certificate storage,
  search and Lean-check timing/peak memory, worker-count determinism, checkpoint resume, and
  partial-publication recovery without elevating performance evidence into a proof claim.

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
- Model authors run a selected Lean proof/exact/native family check, including Veil when that family
  imports it, in the primary Lean project with one command and source diagnostics.
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
- Exact, temporal, native, and embedded Veil checking: `tests/umpire3/model`,
  `tests/umpire3/explore`, and `tests/umpire3/model-checkers`.
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
