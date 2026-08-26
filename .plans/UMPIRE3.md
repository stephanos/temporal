# Umpire3 architecture and roadmap

Status: active architecture and remaining qualification roadmap.

Umpire3 is the independent implementation under `tools/umpire3`. This document defines its
semantic, trust, and completion boundaries. It does not govern the Go Umpire2 runtime described in
[`UMPIRE.md`](UMPIRE.md) or the separate Lean model described in
[`UMPIRE_DSL.md`](UMPIRE_DSL.md). All three efforts serve the goals in
[`UMPIRE_VISION.md`](UMPIRE_VISION.md).

## End state

Umpire3 is complete when one compositional Lean model family supports:

1. independently defined feature contracts and system mechanisms;
2. refinement, interference, safety, and temporal proofs;
3. exact finite exploration with checked certificates;
4. exact and native search, with optional family-owned Veil declarations;
5. generated regressions and sparse developer-authored tests;
6. real Temporal execution in local, CI, remote, public-gRPC, and authorized canary profiles;
7. typed evidence interpretation with established, violated, unknown, and conflict results;
8. deterministic campaign, minimization, replay, and promotion; and
9. a release assurance graph exposing every trust boundary and omission.

The terminal claim is deliberately narrower than proving Temporal correct:

> Important Temporal contracts have explicit semantics; selected mechanisms refine them;
> independent engines search them; counterexamples replay through canonical Lean and real tests;
> live claims are evidence-qualified; and every green result exposes its scope and trust.

## Semantic authority

- Lean is the sole authority for behavior, properties, observations, composition, and checker views.
- One model means one typed `ModelFamily` graph, not one monolithic state machine.
- Feature and system models are independently defined. System state cannot contain feature state,
  feature executions, or a proof that the feature transition already occurred.
- Refinement relations and action mappings live outside both models and must fail for an executable
  nearby mutation.
- Catalog entries, theorem names, hashes, ranks, and statuses are derived metadata. Strings and
  Booleans cannot establish proof, completeness, preservation, or composition.
- Scenario authoring is intent. A requested action attempt is not a successful abstract transition.

## Lean and Go boundary

- Lean and model checkers are offline build and test tools. They do not enter production request
  paths through FFI, subprocesses, or generated executable logic.
- Go owns bounded orchestration, Temporal API calls, evidence transport, cleanup, persistence, and
  process isolation. It does not restate state machines or property rules.
- `tools/umpire3/protocol` is strict versioned transport for explicit Lean-generated views,
  certificates, traces, and evidence programs. It is not an independently authored semantic IR.
- Arbitrary Lean is never silently translated. Every executable, finite, first-order, temporal,
  observation, or checker view has a declared soundness or equivalence obligation.
- Proof-grade finite success requires collision-safe identity and a certificate accepted by the
  small Lean checker.
- Every checker counterexample replays through canonical Lean before promotion or a semantic
  violation claim.

### Optional Veil use

Veil is a Lean library and embedded DSL, not a source-generation target or independent backend.
Declarations are handwritten beside their owning family in the primary Lake project. Each supported
family checks the relation between those declarations and its canonical view.

Lean metaprogramming may reduce local boilerplate, but Go, JSON, templates, and generators must not
create or rewrite `Temporal/Veil/**/*.lean`. Export and recording commands produce source-bound
bindings, normalized results, or proof receipts only.

Testing, trusted SMT, reconstructed solver proof, and kernel proof remain distinct trust modes.
Veil is optional per family and is not required where direct Lean proofs and checked search provide
sufficient evidence.

### Quarantined TLA experiments

TLA+, TLC, and Apalache are historical portability experiments, not completion work. They remain
outside default generation, checks, qualification, release gates, developer setup, and transitive
Lean targets. They may be deleted if quarantine costs more than it preserves. Reactivation requires
a separate approved plan with explicit provenance, bounds, omission, and canonical replay rules.

## Observation and evidence

- Live adapters emit typed raw facts with source identity, clock domain, source-local position,
  entity identity, lineage, causal references, fields, mechanism receipts, and omissions.
- Adapters never emit the truth of the property they are testing.
- Lean-generated programs normalize and interpret facts and return `true`, `false`, `unknown`, or
  `conflict` with supporting fact identities.
- Absence establishes a result only after an authoritative evidence window closes.
- Missing facts, ambiguous identity, incomparable clocks, conflicting sources, and open windows fail
  closed.
- Cross-source order uses causal references or declared ordering guarantees, never timestamp
  comparison alone.

## Protobuf boundary

- Selected Temporal Protobuf messages are imported reproducibly from descriptors.
- Selection closes over nested messages, enums, oneofs, maps, repeated fields, presence, and
  supported well-known types.
- Every selected field has an explicit disposition.
- Product meaning is handwritten and proved in Lean. Generators do not infer semantic identity,
  completion, ordering, or absence from field names.
- Go and Lean share generated conformance fixtures for selected wire interpretations.
- A family imports only its bounded wire surface rather than mirroring all Temporal Protobufs.

## Independence

- Umpire3 does not import or wrap Umpire1, Umpire2, or `common/testing/umpire`.
- Umpire2 is a behavioral reference and executable baseline, never a runtime oracle.
- Retained Umpire2 root tests and independent Umpire3 counterparts run side by side.
- Shared extraction is considered only after qualification and evidence of a stable symmetric seam.
  Extraction is not required for completion.

## Model-family contract

Each qualifying family exposes:

```text
Feature
  behavior, user-visible properties, non-vacuity witnesses

System
  mechanism-only behavior, requirements, guarantees, executable views

Refinement
  relation, action/event mapping, simulation, mutation rejection

Observation
  evidence vocabulary, interpretation program, identity/order/closure obligations

Targets
  finite/first-order/temporal views, omissions, preservation, provenance
```

A target is narrower than its family: it selects a world, modules, property, bounds, checker views,
and omissions. Every omission is proved property-preserving or labeled heuristic; heuristic
omissions cannot support proof-grade completeness.

## Checker portfolio

| Mechanism | Role |
| --- | --- |
| Lean kernel proofs | Semantic and theorem authority |
| Exact exploration and checked certificates | Proof-carrying finite search |
| Native certificate producer | Replaceable scalable search with Lean-checked output |
| Lean temporal checking | Safety and liveness under explicit assumptions |
| Veil | Optional family-scoped Lean declarations and proof/search support |
| TLA+, TLC, Apalache | Quarantined historical experiments |

No remaining work creates a Veil generator, Veil-specific semantic IR, second Lake project, or new
TLA backend integration.

## Remaining roadmap

### Environment qualification

Demonstrate real Temporal behavior across deployment profiles:

- exercise Nexus, Workflow, Workflow Task, Activity, Update, Callback, ownership, routing, lineage,
  timeout, and cross-entity paths through each selected real mechanism;
- retain Umpire2 and independent Umpire3 root behaviors and require qualified semantic agreement
  while permitting implementation-specific mechanics;
- run one immutable semantic digest under local in-process, CI cluster, remote deployment, and
  public-gRPC-only profiles;
- obtain external build, configuration, isolation, observation, cleanup, and retention evidence;
- bind external evidence to reviewed Ed25519 authorities and exact candidate/result/environment
  identities;
- demonstrate public and internal evidence for every eligible claim or return precise unsupported
  results;
- rehearse authorized canary execution with allowlists, least authority, redaction, hard budgets,
  persisted recovery metadata, and resumable cleanup; and
- enforce hard bounds across prepare, execute, wait, observe, and cleanup.

Exit requires genuine retained evidence for every supported profile. Unavailable authority remains
unsupported rather than inferred.

### Release audit

Prove the system is coherent rather than relying on independently green pieces:

- audit every goal in `UMPIRE_VISION.md`, every required artifact, and every retained Umpire2
  behavioral contract against current source and fresh evidence;
- run semantic mutations through Lean proof, exact exploration, native certificates, temporal
  checking, live evidence, minimization, replay, and promotion;
- include the declaration and canonical binding in mutation audits for Veil-owning families;
- reject stale artifacts, cached-only tests, authored attestations, hidden omissions, and unavailable
  external runs; and
- move a release from `candidate` to `qualified` only after every mandatory input is present.

Flow-Next owns task decomposition and execution state for this roadmap. This document retains only
the architectural exit conditions.

## Result and trust model

| Result class | Meaning |
| --- | --- |
| `trace-witness` | Canonical replay accepts a violating finite trace |
| `tested-instance` | One bounded backend or live instance was exercised |
| `finite-exhaustive` | A collision-safe closure certificate was checked for the declared world |
| `bounded-safety` | Safety holds through an explicit depth or resource bound |
| `invariant-proved` | Lean accepted an invariant with an inventoried axiom set |
| `temporal-proved` | Lean accepted a temporal theorem under named fairness assumptions |
| `refinement-proved` | Lean accepted the feature/system simulation |
| `implementation-conforming` | One live run met a property under its profile and evidence set |
| `metadata-validated` | Artifact structure is valid; no semantic proof follows |
| `unknown` | Evidence or checking is incomplete, conflicting, unsupported, or exhausted |

Trust presentation distinguishes kernel proof, checked certificate, reconstructed solver proof,
trusted solver, external testing, live conformance, and metadata validation. Authored status text
cannot select a stronger class.

## Failure and operational semantics

- Unsupported capability and invalid scenario failures occur before allocation.
- Search exhaustion, timeout, cancellation, and resource pressure retain explicit termination
  reasons and never imply exhaustive success.
- Invalid, incomplete, or world-mismatched certificates fail closed.
- Backend crashes cannot transactionally publish final receipts.
- Evidence loss, open windows, causal ambiguity, and contradiction yield unknown or conflict.
- Cleanup runs under independent bounded authority; primary and cleanup failures are both retained.
- Production actions stop on digest drift, isolation uncertainty, evidence loss, budget exhaustion,
  or cleanup uncertainty.
- Worlds, descriptors, evidence, cardinalities, traces, output, and artifacts are bounded before
  allocation.
- Sensitive fields are omitted, redacted, or represented by explicit digests. Credentials never
  enter generated artifacts, command arguments, logs, replay bundles, or checker input.
- Canary subprocesses receive least authority, scoped resources, enforced limits, and killable
  process groups.

## Verification and completion evidence

Fresh checks must establish:

- Lean elaboration, soundness/completeness laws, refinement mutations, non-vacuity, composition,
  certificate validation, observations, theorem provenance, and named fairness assumptions;
- strict Go decoding, hostile-input bounds, digest/source binding, deterministic generation,
  scenario attempt/outcome separation, exploration, evidence, campaign, process, and layout tests;
- direct Veil elaboration, binding mutations, no-generated-source guards, and canonical replay for
  each family that uses Veil;
- focused real Temporal tests for every supported family and independent Umpire2/Umpire3 parity;
- local, remote, public-gRPC, and authorized canary qualification;
- `make umpire3-check`, relevant `go test -count=1 -tags test_dep` packages, formatting,
  `make lint-code`, and repository unit tests when feasible; and
- a release manifest linking every `UMPIRE_VISION.md` goal to current passing evidence.

Umpire3 is complete only when these claims are directly evidenced, every target omission is proved
or explicitly heuristic, every counterexample uses canonical replay, and no green result depends on
stale provenance or hidden trust.

## Source anchors

- Vision: `UMPIRE_VISION.md`.
- Semantic kernel and registrations: `tools/umpire3/model/Umpire3`.
- Feature, system, refinement, observation, composition, and targets:
  `tools/umpire3/model/Temporal`.
- Transport and trust vocabulary: `tools/umpire3/protocol`.
- Authoring and compilation: `tools/umpire3/scenario` and `tools/umpire3/umpire3test`.
- Checking: `tools/umpire3/model`, `tools/umpire3/explore`, and
  `tools/umpire3/model-checkers`.
- Live realization: `tools/umpire3/observation`, `tools/umpire3/execution`, and
  `tools/umpire3/temporal`.
- Campaign and operations: `tools/umpire3/campaign`, `tools/umpire3/replay`,
  `tools/umpire3/process`, and `tools/umpire3/canary`.
- Migration evidence: `tools/umpire3/migration/ledger.json`.

These are evidence and migration references. Except for ordinary Temporal infrastructure, they do
not authorize Umpire3 to depend on earlier Umpire implementations.
