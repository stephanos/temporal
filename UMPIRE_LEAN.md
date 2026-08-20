# Umpire3 roadmap to the full Umpire vision

Status: implemented through the Umpire3 1.2 candidate; external deployment qualification remains.

This document replaced the original Umpire3 bootstrap plan. Its milestones have now been implemented
as the independent `tests/umpire3` candidate. The plan remains the architectural contract; the
implementation boundary and remaining qualification work are recorded in `UMPIRE_LEAN2.md` and the
candidate release manifest.

The end state is an independent Umpire3 in which:

- Lean is the single semantic authority for product behavior, selected distributed-system behavior,
  executable exploration, observation predicates, and refinement;
- ordinary Go test authors use a small, typed, source-located API and never edit experiment JSON,
  hashes, capabilities, or generated model files;
- one sparse regression description runs unchanged against local, CI, remote deployment, and
  production-canary profiles when the selected profile can supply the required authority and
  evidence;
- protobuf wire structures are imported automatically and reproducibly, while their product meaning
  remains explicit and proved rather than guessed by a generator;
- known regressions, model-guided exploration, faults, fuzzing, minimization, replay, and promotion
  all consume the same model family;
- white-box and gRPC-only black-box observations produce evidence-qualified claims without assuming
  a shared clock;
- the existing top-level Umpire tests continue to run as Umpire2 tests and have independent Umpire3
  counterparts; and
- Umpire3 remains independent of Umpire1, Umpire2, and `common/testing/umpire`. Shared code is
  extracted only after two independent implementations prove a genuinely stable seam.

## 1. Decisions and non-goals

### 1.1 “Single model” means one authoritative model family

The vision does not require one enormous state machine. It requires one source of semantic truth.
Umpire3 should use a compositional Lean model family with explicit providers, consumers, assumptions,
guarantees, refinements, and target projections. Product contracts, system mechanisms, experiment
generation, runtime monitors, and coverage identities must all trace back to that family.

Go may interpret generated, versioned programs and realize actions. It must not restate Temporal
state machines, property rules, or action meaning in handwritten registries.

### 1.2 Full independence remains mandatory

Umpire3 may use ordinary Temporal test infrastructure, generated Temporal protobuf packages, the SDK,
and public service clients. It must not import or wrap:

- `common/testing/umpire`;
- `tests/umpire1`;
- `tests/umpire2`;
- Umpire2's Go semantic IR or runtime rule registry; or
- Umpire2's TLA+, Apalache, P, PEx, Ivy, or FizzBee exporters as a semantic shortcut.

Umpire2 is a behavioral reference and migration oracle, not an implementation dependency. The
existing import guard in `tests/umpire3/layout_test.go` remains a release gate.

### 1.3 Lean remains the authority; it does not become the live-system driver

Lean owns meaning and proves executable projections. The Go side owns bounded orchestration, Temporal
API calls, participant lifecycles, observation collection, evidence normalization, cleanup, and
process isolation. Their boundary is versioned data with hashes and manifests, not FFI.

### 1.4 Protobuf structure is generated; protobuf meaning is not

Temporal request, response, event, and failure message shapes are useful model inputs. Importing them
by hand is error-prone and will drift. Umpire3 will therefore generate selected Lean wire types from
protobuf descriptors.

It will not infer that a field named `run_id` establishes identity, that a response means an operation
completed, or that absence has product meaning. Those are semantic interpretation decisions and must
be written in Lean, associated with evidence requirements, and proved against the product model.

### 1.5 Umpire3 does not claim support it cannot enforce

An unsupported action, unavailable capability, incomplete exploration, missing identity lineage,
ambiguous ordering, lost evidence, cleanup failure, or exhausted budget cannot produce a conforming
claim. Production time limits are called hard only when a process boundary lets the controller stop a
non-cooperative driver.

### 1.6 The existing Umpire2 tests are retained

Migration is additive. The current root-level files are renamed to make ownership explicit, then
copied and independently rewritten:

| Current file | Retained Umpire2 file | New independent Umpire3 file |
| --- | --- | --- |
| `tests/umpire_test.go` | `tests/umpire2_test.go` | `tests/umpire3_test.go` |
| `tests/umpire_probe_test.go` | `tests/umpire2_probe_test.go` | `tests/umpire3_probe_test.go` |
| `tests/umpire_regress_test.go` | `tests/umpire2_regress_test.go` | `tests/umpire3_regress_test.go` |

The Umpire2 files remain executable baselines. The Umpire3 files must not call Umpire2 or compare
against Umpire2 at runtime.

## 2. Baseline at plan creation

Umpire3 1.0 established several valuable foundations:

- `Umpire3.Transition` defines runs and reachability;
- `Umpire3.Executable` requires a proof that executable `next` is equivalent to relational `Step`;
- `Umpire3.Refinement` supports relational refinement and stuttering;
- product and system models are separated;
- Nexus cancellation and Workflow Update share a current-completion delivery guarantee;
- checked experiments carry semantic and proof provenance;
- experiment decoding, capability preflight, runtime orchestration, cleanup, minimization, replay, and
  artifact encoding are fail-closed and independently testable;
- the Nexus slice reaches a real Temporal matching/frontend task exchange; and
- generation drift and forbidden dependency directions are checked.

That was the semantic kernel from which M13–M27 began.

### 2.0 Implementation outcome

Umpire3 1.2 now has the generated semantic catalog and protobuf projection, v2 experiment protocol,
composition/parity/monitor ledgers, sparse compiler, typed domain authoring facade, causal evidence
graph, SDK participant, target-specific public-history observer, scoped fault realizer contract,
guided exploration, deterministic campaign/minimizer/promotion pipeline, portable runner, process
isolated canary controller, strict replay bundle, unified CLI, qualification receipts, and the
side-by-side Umpire2/Umpire3 root file layout. The checked migration catalog describes 28 behaviors.

The release intentionally remains `candidate`. No local implementation can create the required
remote-deployment, gRPC-only, or production-canary receipts, and several broad parity contracts still
use the nearest public SDK mechanism rather than a dedicated Temporal mechanism. Those are explicit
qualification gaps, not hidden skips or inferred passes.

### 2.1 Gap summary at plan creation

| Area | Umpire3 now | Full-vision requirement |
| --- | --- | --- |
| Semantic breadth | Nexus cancellation and a small Update lifecycle | Temporal model family covering the Umpire2 entity, relation, property, and verification inventory |
| Experiment format | Actions have IDs, string kinds, and checkpoints; exporters assemble JSON strings | Typed values, arguments, symbols, relations, policies, faults, partial order, and generated canonical encoding |
| Go vocabulary | Handwritten known-action and capability tables | Lean-generated catalog consumed by generic Go interpreters |
| Protobuf awareness | One selected cancellation request, limited generated Lean shape, descriptor hash | Recursive selected descriptor closure with presence, enums, oneofs, maps, repeated fields, nested messages, and well-known types |
| API interpretation | Narrow handwritten Lean and Go interpretations | Explicit Lean interpretations with coverage obligations and generated cross-language conformance vectors |
| Authoring UX | Model authors export JSON; test authors coordinate experiment, factory, and runtime directly | One typed `umpire3test.RequireRegression` facade with source diagnostics and profile-independent intent |
| Planning | Fixed serial action list | Deterministic sparse-plan completion, partial orders, bounded alternatives, and runtime identity binding |
| Runtime realism | Real Nexus task exchange; Update is largely adapter-local state | Real Temporal realizations for supported product actions and programmable SDK participants |
| Evidence | Narrow per-action observations and checkpoints | Multi-source causal evidence graph with identity lineage, clock domains, omissions, and generated property requirements |
| White/black box | Controlled local and CI profiles | In-process, telemetry/history, and public-gRPC-only evidence profiles |
| Faults | Controlled stale completion in one slice | Typed fault algebra, scopes, occurrence policies, learned footprint, and deterministic scheduling |
| Exploration | Bounded Lean frontier for selected models | Templates, holes, constraints, symmetry reduction, coverage, campaigns, and promoted regressions |
| Fuzzing | None | Descriptor/model-guided input, schedule, fault, and topology mutation prioritized by semantic novelty |
| Environments | Local/CI | Local, CI, remote deployment, CI/CD cloud, and separately authorized canary |
| Canary | Explicitly unsupported | Isolated, allowlisted, audited, redacted, recovery-safe execution with enforceable hard budgets |
| Migrated functional tests | One Umpire3 integration slice | Independent Umpire3 equivalents for every retained root-level Umpire2 functional/probe/regression behavior |

### 2.2 Vision traceability

| Vision goal | What is missing | Terminal evidence |
| --- | --- | --- |
| One model for software behavior | Composition/catalog and generated monitor semantics | Every action, property, observation predicate, and coverage point resolves to one Lean catalog identity |
| Specify and verify known regressions | Typed scenario compiler and property-complete models | Each retained Umpire2 regression has a Umpire3 scenario, model proof, live execution, and qualified result |
| Deterministic regression plans | Sparse deterministic compiler | Same catalog, scenario, seed, profile capabilities, and bounds produce byte-identical compiled intent and digest |
| Same tests local through canary | Portable profiles and authority separation | One scenario digest executes unchanged in all supported profiles; only realization/evidence manifests differ |
| White-box and gRPC-only black-box | Evidence graph and public observation adapters | The same eligible claim is established independently under both profiles, or explicitly unsupported when public evidence is insufficient |
| Developer-friendly API | Generated typed DSL and one facade | Ordinary regression needs no generated-file knowledge and meets the UX budget in section 7 |
| Find unknown bugs | Guided exploration and campaigns | A campaign finds, minimizes, replays, and promotes at least one previously unknown real defect or seeded cross-layer mutation |
| Faults first class | Fault algebra and realizers | Fault identity appears in model, compiled scenario, evidence, coverage, artifact, minimizer, and replay |
| Non-linear steps and unknown IDs | Typed symbols, projections, and binding graph | A run ID learned after start is bound and used across unordered later actions without weakening identity claims |
| Programmable SDK workers | Umpire3-owned kitchensink program | Model actions compile exhaustively to SDK participant programs and unsupported commands fail before allocation |
| Guided parameter/template exploration | Typed holes and constrained domains | Bounded exploration reports candidates, symmetry reductions, omissions, and completeness status |
| Distributed clocks and skew | Clock-domain-aware partial order | Cross-source judgments use causal references or declared order guarantees, never raw timestamp comparison |
| Guided automatic fuzzing | Semantic novelty queue and corpus | New action/property/relation/fault/input coverage is prioritized deterministically and retained for replay |

## 3. Chosen approach

Three implementation strategies were considered.

### 3.1 Mechanical Umpire2 port

This would quickly recover breadth, but it would make Go a second semantic authority and import the
architecture Umpire3 was created to replace. It is rejected.

### 3.2 Authoring facade over the current narrow runtime

This would improve the first impression but leave most scenarios unrealizable and most claims
unobservable. It would freeze an API before the catalog, identity, fault, and evidence types are
known. It is rejected as the main sequence.

### 3.3 Capability ladder rooted in generated semantics

This is the chosen approach:

1. make Lean emit a complete typed catalog and protobuf projections;
2. make composition, monitor programs, and scenario compilation consume that catalog;
3. expose the resulting types through a small generated author facade;
4. recover live evidence, participants, and faults as Umpire3-owned adapters;
5. add guided exploration, fuzzing, portable profiles, and canary authority; and
6. finish with side-by-side functional test parity and an end-to-end release audit.

This sequence is slower than wrapping Umpire2, but every increment strengthens the desired final
architecture and can be accepted as a complete vertical slice.

## 4. Target architecture

```text
selected protobuf descriptors
          |
          v
generated Lean wire types -------- descriptor manifest/hash
          |
          v
explicit Lean interpretation + proofs
          |
          v
product models <--- system modules ---> refinements
          |               |
          +-------+-------+
                  v
       authoritative Lean catalog
       - types and typed values
       - actions and parameters
       - entities and relations
       - properties and evidence requirements
       - faults and capabilities
       - planner and monitor programs with proofs
                  |
       generated, versioned artifacts
                  |
       +----------+-----------+
       |                      |
       v                      v
typed Go scenario API   generic Go interpreters
       |                      |
       +----------+-----------+
                  v
       deterministic compiled experiment
                  |
                  v
        environment/session adapter
                  |
                  v
          causal evidence graph
                  |
                  v
       qualified result + replay artifact
```

### 4.1 Semantic layers

1. **Wire layer** — generated representations of selected protobuf messages and presence semantics.
2. **Interpretation layer** — explicit functions from wire values/evidence to product commands,
   identities, and observations.
3. **Product layer** — user-visible lifecycle and cross-entity contracts.
4. **System layer** — task delivery, ownership, persistence, routing, retry, and failure mechanisms.
5. **Refinement layer** — proofs that system behavior implements product behavior.
6. **Experiment layer** — bounded, typed projections suitable for planning and execution.
7. **Observation layer** — proved monitor programs and evidence prerequisites for live claims.

### 4.2 Runtime layers

1. **Author facade** — typed sparse intent only.
2. **Compiler** — validates intent, closes dependencies, completes bounded paths, and emits one
   canonical experiment.
3. **Profile** — declares authority, realizations, observations, ordering guarantees, retention,
   isolation, and budgets.
4. **Session** — allocates scoped resources and realizes actions.
5. **Evidence graph** — normalizes independent facts, identity lineage, causal edges, omissions, and
   clock domains.
6. **Monitor interpreter** — evaluates only generated proved programs.
7. **Qualifier** — decides established, violated, unsupported, inconclusive, or evidence failure.
8. **Artifact/corpus** — stores redacted replay material under semantic and realization digests.

### 4.3 Deep public seams

The intended stable operations are:

```go
umpire3test.RequireRegression(t, scenario, options...)
runtime.Run(ctx, experiment, factory, options)
campaign.Run(ctx, template, executor, options)
canary.Run(ctx, approvedExperiment, controller)
```

Everything else should be an implementation detail or generated vocabulary. `runtime.Run` remains
the low-level conformance seam. Most Temporal test authors should use only
`umpire3test.RequireRegression` plus generated domain packages.

### 4.4 Versioned artifacts

Each compiled or executed artifact must identify:

- schema version;
- Lean source and toolchain digest;
- semantic catalog digest;
- selected protobuf descriptor digest;
- proof-manifest digest;
- sparse scenario digest and deterministic seed;
- compiled experiment digest;
- profile and capability manifest digest;
- concrete realization/build/configuration identity;
- observation/evidence profile digest;
- retention/redaction policy; and
- omissions, limits, cleanup outcome, and replay classification.

Semantic identity must not include concrete namespace, run ID, endpoint, worker identity, timestamp,
or secret. Those belong to a separately redacted realization record.

## 5. Protobuf-aware modeling

### 5.1 Current answer

Umpire3 is aware of a protobuf message today, but only narrowly. The current generator under
`tests/umpire3/cmd/umpire3-api` selects the cancellation request descriptor, emits a limited Lean
shape, and records a descriptor hash. The product interpretation is still handwritten. This proves
the seam, but it is not broad enough for the messages used by the existing Umpire tests.

Automatic import is useful and should be expanded. Hand-copying protobuf structures into Lean would
create drift, lose field presence and oneof semantics, and make fuzz input generation incomplete.
Automatically importing all Temporal APIs would create an enormous accidental model surface and
confuse wire compatibility with product meaning. Selection must therefore be explicit and closed over
dependencies.

### 5.2 Descriptor selection

Add a checked-in selection manifest under `tests/umpire3/model/Temporal/API` containing:

- fully qualified request, response, event, failure, and payload message names;
- the product action, observation, or fuzz domain that needs each message;
- inclusion or explicit exclusion of fields;
- sensitivity/redaction classification;
- required presence behavior; and
- an owner for each semantic interpretation obligation.

The generator resolves the recursive `FileDescriptorSet` closure. It must support:

- all scalar kinds;
- enums, including unknown numeric values;
- nested messages;
- proto3 optional and explicit presence;
- oneofs;
- repeated fields and maps;
- `google.protobuf.Duration`, `Timestamp`, and selected other well-known types;
- bytes without leaking them into diagnostics; and
- recursive types through bounded views rather than unbounded Lean values.

Unsupported descriptor features fail generation with the fully qualified field path. They are never
silently dropped.

### 5.3 Generated outputs

For each selection, generation produces:

- Lean wire types and validation helpers;
- field/presence/oneof metadata;
- the closed descriptor manifest and digest;
- typed value domains usable by exploration and fuzzing;
- coverage obligations showing interpreted, transport-only, sensitive, and intentionally unmodeled
  fields; and
- canonical cross-language fixtures.

Go continues to use Temporal's existing generated protobuf types. It reads descriptors and fixtures;
Umpire3 must not generate a competing Go message implementation.

### 5.4 Explicit semantic interpretation

For every selected message, a model author supplies a Lean interpretation such as:

```lean
interpretCancelRequest : CancelRequestWire -> Interpretation CancelCommand
```

The result distinguishes accepted, rejected, irrelevant, ambiguous, and unsupported input. Proofs
connect accepted interpretations to product preconditions and record which fields establish identity,
causality, or other evidence. Generated coverage fails if a selected non-transport field has no
disposition.

Go realization code may contain transport mappings, but it cannot independently decide product
meaning. Generated fixtures and monitor/action programs test its behavior against the Lean
interpretation.

### 5.5 Protobuf acceptance tests

- descriptor output is byte-for-byte deterministic;
- a selected descriptor change fails the generated-diff gate;
- optional absent and present-default values remain distinct;
- unknown enum values remain representable and reach an explicit interpretation result;
- oneof replacement, map ordering, repeated ordering, and nested presence round-trip correctly;
- negative and overflow `Duration` cases are retained;
- sensitive bytes are hashed/redacted in failures and artifacts;
- every selected field has one explicit disposition;
- Lean-generated fixtures and Go transport interpretation agree; and
- an unselected descriptor change does not perturb the Umpire3 catalog digest.

## 6. Umpire2 feature disposition

Umpire2 already demonstrates many behaviors required by the vision. Umpire3 should reproduce their
contracts independently, improve weak guarantees, and decline architecture that conflicts with Lean
authority.

### 6.1 Must be recreated in Umpire3

| Umpire2 capability | Umpire3 disposition |
| --- | --- |
| Typed sparse regression DSL | Recreate from the generated Lean catalog with typed entities, actions, outcomes, relations, policies, symbols, and projections |
| `OnePath`, `AllPaths`, `AnyOrder`, `Before`, `During`, `Repeat`, `Require` | Recreate as structural scenario constraints compiled deterministically against proved transition metadata |
| Runtime binding of unknown IDs | Recreate with typed symbols, evidence-backed projections, single-assignment bindings, and lineage validation |
| Deterministic completion of sparse intent | Recreate with stable ordering, explicit bounds, and no silent truncation |
| Environment-independent intent | Recreate; profiles bind realizations and evidence only after semantic compilation |
| Capability preflight before allocation | Retain and broaden |
| Entity/relation/identity model | Recreate in Lean, including Namespace, TaskQueue, Workflow, WorkflowRun, WorkflowTask, Activity/ActivityExecution, NexusOperation, and Callback |
| Live assurance rules | Recreate as Lean properties compiled to proved generic monitor programs, not handwritten Go rules |
| Causal trace with source, clock domain, sequence, and references | Recreate as a Umpire3 evidence graph |
| Public API, history, telemetry, and in-process evidence profiles | Recreate with explicit qualification requirements and black-box limitations |
| Kitchensink/programmed participants | Recreate as a versioned Umpire3 participant program with exhaustive SDK mappings |
| First-class scoped faults | Recreate in Lean and in profile-specific realizers |
| Learned RPC/HTTP footprint | Recreate as observed realization metadata used by deterministic fault selection |
| Coverage denominator and pairwise matrix | Recreate from the Lean catalog and supported profile dimensions |
| Risk/novelty-guided campaign | Recreate around Umpire3 experiments, evidence, and corpus |
| Minimization, replay, and promotion | Extend current Umpire3 support to sparse intent, faults, bindings, parameters, and exact qualified-violation identity |
| Canary safety envelope | Recreate with process-enforceable deadlines and independent recovery control |
| Explicit unsupported action gaps | Recreate as catalog entries with owner, reason, affected goals, and exit criterion |

### 6.2 Semantic inventory to recover

The Umpire2 inventory is the minimum comparison set, not automatically the final design.

Entities and relations to disposition:

- Namespace and TaskQueue;
- Workflow and WorkflowRun;
- WorkflowTask;
- Activity and ActivityExecution;
- NexusOperation;
- Callback;
- ownership and current-epoch relations;
- workflow/Nexus endpoint and callback references;
- Nexus/Activity bidirectional links;
- queue, routing, hosting, and persistence relations; and
- parent/child or continuation lineage needed by current root tests.

Named assurance properties to disposition:

- SpeculativeTaskCreation;
- NexusOperationClosure;
- NexusActivityLinkConsistency;
- NexusOperationTimeoutSemantics;
- CallbackReferenceConsistency;
- CallbackResponseConsistency;
- WorkflowTaskStarvation; and
- EntityProgress.

Verification targets to disposition:

- `feature-nexus`;
- `feature-workflow-speculative-delivery`;
- `foundation-backlog-ack`;
- `foundation-delivery-safety`;
- `foundation-ownership-fencing`;
- `foundation-routing-isolation`;
- `integration-activity-delivery`;
- `integration-callback-nexus`;
- `integration-callback-workflow`;
- `integration-nexus-activity`;
- `integration-workflow-delivery`; and
- `protocol-atomic`.

For each item, the parity ledger must record `equivalent`, `replaced`, `intentionally unsupported`, or
`not yet implemented`, with evidence and an owner. “Present in a catalog” is not parity: an item needs
the relevant theorem, executable projection, observation contract, and at least one negative control.

### 6.3 Do not port

- Go-owned product state machines or property evaluators;
- backend-specific semantic IR as a second source of truth;
- handwritten action-kind and capability registries;
- raw snapshots or timestamps as sufficient evidence;
- promotion of the original broad template instead of the minimized sparse reproducer;
- “same violation” comparisons based only on a status/property label without evidence identity;
- success results with no qualified claim;
- cooperative in-process timeouts described as hard budgets; or
- a compatibility wrapper that makes Umpire3 execution depend on Umpire2.

Multiple formal backends are not a release requirement. Lean is the initial and authoritative prover
and explorer. A later backend is justified only by a demonstrated class of properties or scale that
Lean cannot handle, and it must consume a proved projection with differential conformance tests.

## 7. Developer experience

### 7.1 Honest comparison

| Task | Umpire2 | Umpire3 now | Umpire3 target |
| --- | --- | --- | --- |
| Write an ordinary regression | Concise typed Go DSL | Manually coordinate exported experiment, JSON, adapter, and runtime | Concise typed Go DSL backed by Lean-generated vocabulary |
| Express partial intent | Yes | No; fixed serial actions | Yes, with deterministic completion and explain output |
| Use an ID learned at runtime | Typed bind/project | Narrow string bindings only | Typed single-assignment symbols and lineage-checked projections |
| Add a fault | Scoped policy/action | Bespoke controlled fault | Typed fault constructors with scope and occurrence policy |
| Change profile | Same intent can compile for several profiles | Local/CI only and adapter-facing | One test option; unsupported evidence fails before allocation |
| Understand failure | Source-bearing compiler and rich artifacts | Runtime/protocol errors are lower-level | One diagnostic from source intent through evidence and cleanup |
| Add model semantics | Go declarations and generators | Lean, proofs, exporter, and Go adapter | Lean plus scaffolding, generated catalogs, and explicit adapter obligations |

Today Umpire3 is better at semantic authority and proof discipline but materially worse for an
ordinary test author. The roadmap must not call the UX complete until it is at least as concise as
Umpire2 and more explanatory when something fails.

### 7.2 Target authoring API

The following is a design target, not a currently implemented API:

```go
func TestNexusCancellationRetryUmpire3(t *testing.T) {
    op := nexus.Operation("op")

    umpire3test.RequireRegression(t, regress.OnePath(
        op.Reaches(nexus.Started),
        regress.During(
            nexus.DropNext(nexus.CancelNexusOperation),
            op.CancelWithRetry(),
        ),
        op.Reaches(nexus.Canceled),
    ))
}
```

An unknown identity remains typed:

```go
run := workflow.Run("run")
start := workflow.Start("workflow")

umpire3test.RequireRegression(t, regress.AllPaths(
    start,
    regress.Bind(run.ID(), start.ObservedRunID()),
    regress.AnyOrder(
        workflow.Signal(run, "finish"),
        workflow.Query(run, "status"),
    ),
    run.Reaches(workflow.Completed),
))
```

The exact fluent spelling may change before the M18 API freeze. The important constraints are:

- domain values are typed; arbitrary action/property strings are not accepted;
- the author declares behavior, not cluster setup or observation polling;
- profiles are options, not different scenario definitions;
- model-required capabilities are inferred;
- source locations survive normalization and generated expansion;
- unsupported profile/action/evidence combinations fail before allocation;
- the failure message shows the sparse intent, selected completed path, grounded bindings, first
  divergent observation, missing evidence, cleanup outcome, artifact path, and one replay command;
- `t.Helper()` points failures at the author's constructor or test line; and
- payloads and credentials are redacted by construction.

### 7.3 UX budgets

The authoring milestone is not complete unless:

- a new contributor can write and run the documented first regression in ten minutes or less from an
  already built repository;
- an ordinary one-entity regression is at most 20 non-blank lines and normally uses no more than
  three Umpire3 imports;
- no ordinary test touches Lean exporter commands, JSON, hashes, manifests, capabilities, environment
  sessions, polling loops, or cleanup;
- the same scenario source runs against every eligible profile;
- `go test` provides the normal path; a separate CLI is optional explanation/replay tooling;
- generated constructors have Go documentation derived from Lean catalog descriptions;
- editor completion exposes only valid typed options where practical; and
- every compile failure names a stable category, source location, expected type/capability, and
  suggested resolution.

### 7.4 Model-author UX

Model authors need a different but equally explicit workflow:

1. scaffold a product/system/refinement/experiment slice;
2. select needed protobuf descriptors;
3. implement interpretations and proofs;
4. run one generation command;
5. see field, action, property, proof, adapter, observation, and negative-control obligations in one
   report; and
6. run one focused check before the repository-wide gate.

Generation must be idempotent and diff checked. Checked-in generated files are reviewable semantic
artifacts and are never hand edited.

The Lean-side API should provide typed constructors or syntax for lifecycle states, parameterized
actions, guards/effects, observations, properties, modules, and refinements. It may generate routine
encoders, catalogs, finite-domain enumerators, and proof statements; it must not synthesize a trusted
proof by hiding an unproved assumption. A small product lifecycle should be declared once, and the
tool should list the remaining executable-equivalence, refinement, evidence, adapter, protobuf-field,
and negative-control obligations. A semantic-only addition must not require a handwritten Go
registry change.

Model-author acceptance includes a documented small lifecycle built from scratch, a cross-module
refinement example, an intentionally incomplete model whose exact obligations are reported, editor
navigation from generated Go vocabulary back to Lean source, and focused checks that finish without
running a live cluster.

## 8. Milestone roadmap

Milestones are dependency ordered, not calendar estimates. Each milestone must ship a vertical slice
with tests, generated-diff checks, documentation, and explicit unsupported cases. A later milestone
may start discovery work early, but it cannot claim completion before its dependencies.

### M13 — Authoritative semantic catalog and experiment v2

**Goal:** remove handwritten cross-language vocabulary and give every later tool one typed model
catalog.

**Deliverables**

- Add Lean catalog declarations for types, values, entities, relations, actions, parameters,
  observations, properties, evidence requirements, faults, capabilities, modules, and targets.
- Extend `SemanticExperiment` to typed arguments, typed bindings, policies, partial-order constraints,
  faults, and retention requirements.
- Replace experiment-specific JSON string concatenation with one canonical Lean encoder/exporter.
- Generate a versioned `catalog.json`, experiment schema, proof manifest, and Go typed identifiers.
- Replace `knownActionKinds` and `knownCapabilities` with catalog-backed validation.
- Preserve v1 decoding only long enough to regenerate all checked Umpire3 artifacts; do not create an
  indefinite compatibility layer.

**Proposed seams**

- Lean: `model/Umpire3/Catalog.lean`, `model/Umpire3/Value.lean`, and the existing
  `model/Umpire3/Experiment.lean`.
- Go: `protocol/catalog.go`, `protocol/value.go`, and `protocol/experiment.go`.
- Generation: one Umpire3 export command used by `make umpire3-gen` and `make umpire3-check-generated`.

**Acceptance**

- adding or renaming a Lean action/property changes the catalog and fails drift checks until
  regenerated;
- no handwritten Go allowlist owns semantic vocabulary;
- decode/encode is canonical, bounded, unknown-field rejecting, and round-trips typed values;
- malformed types, dangling identities, unbound symbols, duplicate IDs, cyclic order constraints,
  and unsupported schema versions fail before resource allocation; and
- both current experiments migrate to v2 with unchanged proved semantic traces.

### M14 — Recursive protobuf projection and interpretation obligations

**Goal:** make the model structurally aware of every selected Temporal message without hand-copying
wire definitions.

**Deliverables**

- Add the explicit descriptor selection manifest described in section 5.
- Expand the generator to the full supported descriptor closure and well-known types.
- Generate Lean wire types, field metadata, typed fuzz domains, descriptor hashes, and conformance
  fixtures.
- Add a field-disposition report and fail generation on silent omissions.
- Move existing API meanings into explicit Lean interpretation modules with proofs.
- Validate Go transport mappings against generated fixtures.

**Acceptance**

- Nexus cancellation, Nexus start, Workflow start/signal/query/update, relevant history events, and
  failure messages used by the retained tests are selected or explicitly deferred;
- presence, oneof, enum, repeated, map, nested, bytes, and `Duration` cases have positive and negative
  tests;
- a protobuf drift produces a focused semantic review diff; and
- adding unrelated Temporal protobuf fields does not expand the model accidentally.

### M15 — Compositional model family and proved monitor programs

**Goal:** scale from two slices to one compositional authority without building a monolith or a
handwritten Go rule engine.

**Deliverables**

- Define module ownership, imports, assumptions, guarantees, interference, refinement, and target
  projection in Lean.
- Add typed Lean declaration helpers for lifecycles, actions, properties, modules, and targets, plus a
  generated obligation report for unfinished model slices.
- Generalize the current task-delivery guarantee into versioned provider/consumer contracts.
- Reject dependency cycles, missing providers, conflicting owners, unsatisfied assumptions, and
  vacuous target projections.
- Define a small generic monitor-program IR in Lean for decidable observation properties.
- Prove each exported monitor program equivalent to its Lean property over normalized observations.
- Implement one generic Go interpreter for the generated monitor IR.
- Generate coverage identities and evidence prerequisites from the same declarations.

**Acceptance**

- Nexus and Update compose through a shared delivery provider rather than duplicate mechanics;
- a small product lifecycle is declared once without parallel catalog or Go-registry edits;
- an intentionally weakened provider or observation program fails a proof or a negative control;
- Go contains no product-specific property switch; and
- target projection retains interfering environment actions and records bounded omissions.

### M16 — Umpire2 semantic inventory parity

**Goal:** express or explicitly disposition the useful Umpire2 semantic surface in the Lean model
family.

**Deliverables**

- Add the entities, relations, properties, and verification targets listed in section 6.2.
- Model Workflow Task speculative delivery, ownership fencing, routing isolation, backlog/ack,
  Activity delivery, Nexus/Activity linking, callbacks, timeout behavior, continuation/reset lineage,
  and cross-entity atomicity needed by current tests.
- For each target, provide bounds, assumptions, executable equivalence, refinement, permitted traces,
  forbidden traces, and at least one mutation/negative control.
- Maintain a checked parity ledger with semantic identity and evidence links.
- Carry explicit action/property gaps rather than omitting them from generated inventory.

**Acceptance**

- all eight named assurance properties and twelve named targets have an explicit disposition;
- every `equivalent` item has proof, executable exploration, monitor program where observable, and
  negative control;
- no property exists once in Lean and again as independent Go logic; and
- bounded exploration reports complete, incomplete, or resource-limited truthfully.

### M17 — Deterministic sparse compiler and non-linear identity

**Goal:** compile concise behavioral intent into complete, deterministic, model-valid experiments.

**Deliverables**

- Support `OnePath`, `AllPaths`, `AnyOrder`, `Before`, `During`, bounded `Repeat`, and `Require`.
- Normalize scenario structure into a partial-order constraint graph.
- Use proved catalog transition metadata for generic path completion; do not add a Go product planner.
- Add typed symbols, single-assignment binds, observation projections, dependency chains, and grounded
  identity records.
- Distinguish user order, semantic dependency, same-source order, and runtime causal order.
- Deduplicate and sort completed paths deterministically.
- Make path, state, memory, and time limits explicit; never silently truncate `AllPaths`.
- Produce an explain artifact before allocation.

**Acceptance**

- identical inputs produce byte-identical suites and digests across repeated runs;
- unordered actions produce all and only valid bounded linearizations;
- cycles, ambiguous producers, missing projections, rebinds, type mismatches, and incomplete all-path
  enumeration fail with source-bearing errors;
- a workflow run ID unknown at author time is learned, grounded, and reused with verified lineage; and
- sparse Umpire2 regression shapes can be expressed with equal or fewer semantic statements.

### M18 — Generated typed author facade

**Goal:** make the normal test-writing experience better than Umpire2 without weakening semantic
ownership.

**Deliverables**

- Add Umpire3-owned `regress`, `regress/nexus`, `regress/workflow`, `regress/activity`,
  `regress/callback`, `regress/fault`, `regress/capability`, and `umpire3test` packages.
- Generate typed IDs, enums, entity handles, action constructors, outcome constructors, relations,
  property checks, and capability options from the catalog.
- Keep ergonomic handwritten combinators structural only; they may compose generated terms but may
  not introduce product semantics.
- Add `RequireRegression` as the ordinary deep facade over the M17 compiler.
- Add source capture, stable error categories, explain output, replay output, and generated Go docs.
- Publish a first-test tutorial and model-author workflow.

**Acceptance**

- the cancellation retry and ordinary completion examples meet the UX budgets in section 7.3;
- invalid action/entity combinations fail at Go compile time where representable and otherwise at
  scenario compilation with source location;
- changing catalog vocabulary regenerates the facade deterministically; and
- the facade has no imports of prior Umpire implementations.

### M19 — Causal evidence graph and real white/black-box observation

**Goal:** qualify model claims from independent live evidence in distributed systems with clock skew.

**Deliverables**

- Replace narrow checkpoint matching with a normalized evidence graph of facts, transitions,
  relations, actions, omissions, and claims.
- Record source identity, clock domain, source-local sequence, authoritative event time, observation
  time, causal references, entity identity, lineage, and payload digest.
- Add explicit evidence profiles for public gRPC only, public gRPC plus history, telemetry, and
  in-process hooks.
- Drive black-box observation exclusively through public service APIs allowed by the profile.
- Ingest history, API responses, task protocol evidence, OpenTelemetry, and in-process facts through
  source adapters into one bounded normalization seam.
- Connect property evidence requirements from the catalog to the generic monitor interpreter.
- Stop observation immediately on established safety violations while preserving cleanup.

**Acceptance**

- cross-clock timestamps alone can never establish `Before`;
- causal references or declared same-source ordering can establish it;
- missing source, identity, lineage, order, or retention evidence yields unsupported/inconclusive;
- public-gRPC and white-box profiles establish the same eligible semantic claim from independent
  evidence paths;
- evidence contradiction produces evidence failure, not an arbitrary winner; and
- real Nexus and Workflow Update tests use server evidence rather than adapter-local booleans.

### M20 — Real action adapters and programmable SDK participants

**Goal:** realize the modeled behavior against Temporal without bespoke workflow code per regression.

**Deliverables**

- Replace remaining adapter-local simulations with real Temporal API/task/history interactions.
- Define a versioned Umpire3 participant program for workflow, activity, Nexus, update, signal, query,
  timer, child, cancellation, retry, and failure responses.
- Compile model actions into Omes kitchensink where it is already sufficient and into a narrow
  Umpire3-owned participant only where a proven gap exists.
- Support synchronous/asynchronous/deferred responses, blocking, controlled failure, cancellation,
  worker crash/restart, and payload/result selection.
- Validate the entire program and all action mappings before starting a worker.
- Expose participant capabilities through environment preflight.

**Acceptance**

- every supported program command has positive, malformed, unsupported, and cleanup tests;
- command-to-SDK mapping is exhaustive and drift checked;
- a regression author does not register a bespoke workflow for modeled behavior;
- participant teardown is idempotent and independently bounded; and
- current kitchensink workflow and Nexus functional tests run through Umpire3-owned compilation.

### M21 — First-class fault algebra and deterministic realization

**Goal:** represent faults as semantic, scoped, replayable behavior rather than hidden timing tricks.

**Deliverables**

- Define Lean fault terms for drop, delay, duplicate, reorder, hold/release, rejection, process crash,
  restart, partition, failover, clock skew, and selected persistence errors.
- Define scope by namespace, endpoint, task queue, service, RPC/HTTP route, participant, attempt,
  occurrence, and bounded interval.
- Model installation, activation, observation, release, and cleanup.
- Implement profile-specific realizers with capability and safety classification.
- Learn actual RPC/HTTP causal footprints from a baseline run and use them as candidates.
- Select fault drives deterministically by semantic novelty and risk under explicit budgets.
- Retain fault identity/effect in coverage, evidence, minimization, replay, and artifacts.

**Acceptance**

- faults cannot escape declared isolation scope;
- unsupported or unsafe faults fail in preflight;
- cleanup removes installed faults after success, violation, cancellation, panic, or timeout;
- a learned footprint is realization evidence, not silently promoted to product semantics;
- the cancellation retry and HTTP fault tests replay deterministically; and
- clock-skew tests prove the qualifier does not depend on wall-clock order.

### M22 — Guided model exploration

**Goal:** let users provide goals, templates, parameters, and constraints while Lean explores the
remaining bounded state space.

**Deliverables**

- Add typed holes and finite domains for actions, entity counts, topology, parameters, schedules,
  faults, and checkpoints.
- Support reachability goals, safety-property challenges, transition/relation coverage goals, and
  required/forbidden template fragments.
- Use Lean executable models to enumerate valid candidates with deterministic ordering.
- Add symmetry reduction for interchangeable entities and partial-order reduction for independent
  actions, with checked preservation conditions.
- Report explored, pruned, omitted, incomplete, and resource-limited candidates separately.
- Export every selected candidate as an ordinary v2 experiment.

**Acceptance**

- the same catalog/template/seed/bounds produce the same ordered candidates;
- a small model is exhaustively compared with and without reductions;
- invalid template constraints fail before search;
- no incomplete search is described as exhaustive; and
- selected candidates pass the same runtime/evidence path as handwritten regressions.

### M23 — Coverage-guided fuzzing, campaign, minimization, and promotion

**Goal:** prioritize new behavior, find unknown failures, and turn them into small normal regressions.

**Deliverables**

- Define semantic coverage over product/system transitions, properties, relations, refinements,
  evidence alternatives, protobuf field classes, faults, schedules, topologies, and profile dimensions.
- Mutate typed protobuf values, scenario parameters, schedules, fault scopes, worker responses, and
  bounded topology using catalog and descriptor domains.
- Rank candidates by new semantic coverage, risk focus, rare evidence paths, and corpus distance.
- Keep generation deterministic for a seed and make parallel result merge order-independent.
- Extend minimization across actions, order edges, resources, bindings, parameters, payload fields,
  participant commands, and faults.
- Require preservation of the same qualified violation, grounded identity/lineage, and evidence
  predicate—not merely the same status/property name.
- Replay under controlled realization and evidence profiles before promotion.
- Promote the minimized sparse intent, never the original broad template.
- Emit compilable Umpire3 regression source plus a stable artifact and replay command.

**Acceptance**

- seeded mutations are rediscovered and minimized;
- a real unknown defect or an approved cross-layer mutation is found through novelty-guided search;
- repeated and parallel campaigns produce the same selected corpus for the same inputs;
- every dropped candidate has a budget/duplicate/unsupported reason;
- exactly exhausted but complete minimization is reported complete; and
- promoted source passes the normal `RequireRegression` path without campaign-only semantics.

### M24 — Portable local, CI, and remote-deployment profiles

**Goal:** execute one semantic scenario unchanged across non-production environments.

**Deliverables**

- Define profiles for local in-process, CI test cluster, remote CI/CD deployment, and gRPC-only
  black-box deployment.
- Separate driving authority, observation authority, fault authority, isolation, clock guarantees,
  retention, and cleanup capabilities.
- Add remote endpoint/auth configuration, build/configuration attestation, unique namespace/task-queue
  allocation, and secret-safe diagnostics.
- Bind adapters after semantic compilation and reject unsupported combinations before allocation.
- Add deterministic capability-filtered pairwise matrices.
- Run drivers that can block behind a killable process boundary when a profile promises a hard
  execution budget.
- Keep cleanup on an independent bounded context and preserve primary plus cleanup failures.

**Acceptance**

- one checked scenario digest runs unchanged in local, CI, and eligible remote profiles;
- gRPC-only tests import no server-internal observation path;
- profile differences change realization/evidence digests but not semantic intent;
- unavailable actions or evidence produce explicit unsupported results;
- credentials and payloads cannot appear in logs or artifacts; and
- blocking-driver tests demonstrate enforceable termination where the profile claims a hard budget.

### M25 — Side-by-side rewrite of all existing root Umpire tests

**Goal:** retain every existing Umpire2 functional behavior and add an independently executing
Umpire3 equivalent.

**Deliverables**

**Inventory at plan creation**

- `tests/umpire_test.go`: the suite entry point plus four plan/drive/judge and kitchensink/Nexus
  behaviors;
- `tests/umpire_probe_test.go`: seventeen Nexus/Workflow generation, rejection, reflected-input,
  fault, resilience, degradation, exploration, reset/continuation, learned-footprint, coverage, and
  randomized behaviors;
- `tests/umpire_regress_test.go`: four direct sparse regressions plus three shared regression helpers
  called from `tests/nexus_workflow_test.go` for timeout, callback-after-caller-completion, and
  bidirectional Nexus/Activity linking.

The implementation must regenerate this inventory mechanically from root test files that import or
call an Umpire implementation. New qualifying tests added before M25 are included automatically.

**File migration**

1. Rename the three original files to their `umpire2_*` names with history preserved.
2. Preserve their behavior and existing comments. Make only the naming changes needed to coexist.
3. Qualify Umpire2 suite, test, and shared-helper names where necessary to prevent duplicate Go test
   symbols.
4. Update the three `tests/nexus_workflow_test.go` helper call sites to the retained Umpire2 helper
   names.
5. Copy each retained file to its `umpire3_*` counterpart.
6. Rewrite the copy to use `tests/umpire3` packages, the typed facade, real Umpire3 profiles, and
   Umpire3 evidence. Do not overwrite or route through the Umpire2 original.
7. Give Umpire3 suite/test/helper symbols explicit Umpire3 names so both versions appear separately in
   test output.
8. Add explicit Umpire3 test entries for the three helper-only regressions and the same CHASM variants
   exercised from `tests/nexus_workflow_test.go`; do not leave copied Umpire3 helpers uncalled.
9. Keep the two suites executing side by side until a separate future decision retires Umpire2.

**Behavioral migration rule**

The Umpire3 copy preserves user-visible intent and expected semantic outcome, not Umpire2 internal
structure. A Umpire2 monitor assertion may become an Umpire3 property/checkpoint claim; a bespoke
driver may become a participant program; a randomized probe may become a seeded campaign. Every
mapping is recorded in a checked migration ledger with:

- Umpire2 test and source location;
- Umpire3 test and source location;
- model target and properties;
- scenario and profile;
- required participant/fault/evidence capabilities;
- expected verdict and negative control; and
- artifact/replay coverage.

Package-internal tests under `tests/umpire2/**` continue testing Umpire2. They are not mechanically
copied into Umpire3 because the architectures differ. Every behavioral contract they expose is,
however, covered by the semantic parity ledger and Umpire3-native unit/integration tests. This avoids
copying implementation assumptions while still requiring behavioral completeness.

**Acceptance**

- all six named root files exist in the final layout;
- every inventoried behavior has an independent, enabled, non-skipped Umpire3 test;
- Umpire2 and Umpire3 tests can run in the same package without symbol, global instrumentation, or
  environment collisions;
- Umpire3 copies have no prior-Umpire dependency;
- comments from retained Umpire2 files are preserved, and copied comments are updated only where the
  Umpire3 behavior truly differs;
- each pair agrees on the user-visible semantic outcome while Umpire3 independently qualifies its
  evidence; and
- missing Umpire3 capability blocks milestone completion rather than producing a skipped parity test.

### M26 — Production canary with enforceable safety

**Goal:** run approved Umpire3 experiments in production without expanding test authority implicitly.

**Deliverables**

- Add a separate canary controller and profile; local capability never implies production authority.
- Require immutable approved experiment/catalog/profile digests, tenant and namespace isolation,
  action/fault allowlists, destructive-operation gates, rate/concurrency/count/resource/evidence
  budgets, redaction, audit records, and recovery metadata.
- Begin with read-only/shadow observations, then explicitly approved safe writes, then a separately
  approved minimal fault set.
- Execute nontrivial drivers in killable workers so execution duration is enforceable even if an
  adapter ignores context.
- Keep cleanup/recovery under independent controller authority and persist enough metadata to recover
  after controller or worker crash.
- Stop on evidence loss, isolation uncertainty, configuration drift, budget exhaustion, or cleanup
  uncertainty.

**Acceptance**

- blocking prepare, execute, worker-wait, observation, and cleanup tests cannot exceed the documented
  safety envelope;
- a killed controller can resume cleanup from persisted recovery metadata;
- unauthorized action/fault/profile/digest combinations fail before any production change;
- all emitted diagnostics are secret-safe and auditable;
- evidence loss cannot yield conformance; and
- canary execution uses the same semantic scenario digest as non-production execution.

### M27 — Full-vision convergence and extraction decision

**Goal:** prove Umpire3 satisfies the vision as a coherent product rather than a collection of
features.

**Deliverables**

- Run the end-state acceptance matrix in section 12.
- Close or explicitly reject every Umpire2 parity-ledger item and every vision gap.
- Run model mutations through proof, exploration, live monitor, replay, and promotion paths.
- Exercise local, CI, deployment, black-box, and authorized canary profiles.
- Publish support, limits, authoring, modeling, operations, security, and incident-recovery docs.
- Audit public seams for depth, generated/internal packages for ownership, and forbidden dependency
  directions.
- Consider extraction only for responsibilities implemented independently by Umpire2 and Umpire3
  with demonstrably symmetric tests and stable inputs/outputs.

**Acceptance**

- every goal in `UMPIRE_VISION.md` has passing evidence linked from the release manifest;
- every current root Umpire behavior executes independently in both retained Umpire2 and new Umpire3
  form;
- a known regression, a guided exploration candidate, and a fuzz-discovered violation all converge
  on the same scenario/runtime/evidence/artifact path;
- no green result lacks a qualified claim and supporting evidence;
- the dependency guard is clean; and
- any shared extraction is optional. Umpire3 is complete while still fully independent.

## 9. Continuous migration lane

M25 is the final parity gate, but test migration should drive each capability milestone rather than
wait until the end.

After M18, select one existing root test per milestone as a private or temporary vertical acceptance
case. Add its public `umpire3_*` copy only when it passes without skip or Umpire2 dependency. Suggested
order:

1. ordinary Nexus completion and cancellation retry for catalog/DSL/compiler;
2. reflected invalid inputs for protobuf projection;
3. Workflow start/complete and Update for live evidence;
4. kitchensink Workflow and Nexus for participants;
5. held/dropped HTTP and cancellation faults for the fault algebra;
6. exploration, randomized, learned-footprint, and coverage-guided probes for campaigns;
7. continuation, reset, callbacks, Activity links, and timeout for model-family breadth; and
8. all remaining plan/drive/judge cases before the M25 inventory gate.

This order keeps each milestone honest while avoiding checked-in placeholders. Umpire2 originals are
never removed as part of this roadmap.

## 10. Failure semantics and safety

### 10.1 Stable result classes

Every operation returns one of these semantic classes, with orthogonal cleanup and budget data:

- `established` — the property holds and every declared evidence requirement is met;
- `violated` — the same property is falsified with normalized replayable evidence;
- `unsupported` — the model/profile lacks a required action, authority, observation, or semantic
  interpretation;
- `inconclusive` — execution occurred but bounded search or available evidence cannot decide;
- `resource_limited` — an explicit state/time/memory/action/evidence budget prevented completion;
- `evidence_failure` — sources are missing, contradictory, corrupt, unretained, or cannot establish
  required identity/causality; and
- `infrastructure_failure` — allocation, driver, participant, transport, artifact, or cleanup failed.

Syntax-only generation is not semantic success and does not require a fabricated property claim.
Semantic success always does.

### 10.2 Cleanup and crash handling

- validate semantics and capabilities before allocation;
- allocate every resource under a unique scope and register recovery metadata immediately;
- unwind in reverse order under an independent bounded context;
- make cleanup idempotent and retain both primary and cleanup errors;
- do not let caller cancellation erase cleanup authority;
- use a process/control-plane boundary for any promised hard deadline;
- persist canary recovery state before the corresponding external action; and
- fail closed after crash when realization or evidence identity cannot be reconstructed.

### 10.3 Distributed ordering

Wall-clock time is descriptive only across sources. Ordering claims require one of:

- an explicit causal reference;
- a declared authoritative source-local sequence;
- a request/response relation with verified identity;
- a persisted event ordering contract; or
- another model-declared evidence rule proved sufficient for the property.

Clock offsets and skew can be generated as faults. They cannot change semantic verdicts unless the
product contract itself depends on an authoritative time source.

## 11. Performance, scalability, complexity, and security

### 11.1 Performance

- Compile catalogs and sparse scenarios once per digest and use immutable concurrent-read indexes.
- Keep model exploration bounded and separate from live execution.
- Batch evidence ingestion but preserve source-local order and causal references.
- Use deterministic incremental coverage/corpus updates rather than rescanning all artifacts.
- Do not trade evidence completeness for an unqualified fast pass.

### 11.2 Ten-times load

At 10x candidate or evidence volume:

- shard candidates by semantic digest and merge results deterministically;
- apply backpressure before evidence retention limits are exceeded;
- stop scheduling new work when cleanup or observation falls behind;
- bound per-scope entities, graph edges, payload bytes, artifacts, and retained corpus entries;
- deduplicate before live allocation;
- preserve explicit omission counts and reasons; and
- degrade to `resource_limited` or `inconclusive`, never to a weaker success definition.

### 11.3 Complexity

The main complexity cost is the proved catalog/generator boundary. It is justified because it removes
several more dangerous duplications: handwritten Go semantics, action registries, monitor rules,
coverage inventories, and fuzz domains. Public APIs remain small; generated and domain-specific
details stay behind them.

### 11.4 Security

- descriptors classify sensitive fields before generation;
- arbitrary descriptors, Lean source, participant programs, or experiments are not accepted from an
  untrusted canary caller;
- artifacts store hashes or redacted values for payloads, credentials, and concrete identities;
- production authority is signed/approved and digest bound;
- fault scopes and destructive capabilities are allowlisted separately;
- subprocesses receive least authority and bounded resources; and
- corpus inputs are validated as hostile before decoding or replay.

## 12. End-state acceptance matrix

Umpire3 is complete against this roadmap only when all rows pass.

| End-state invariant | Required demonstration |
| --- | --- |
| One semantic authority | Generated catalog traceability shows no handwritten Go action/property/state-machine ownership |
| Compositional model | All accepted targets close provider/consumer obligations and refinements without cycles or vacuity |
| Protobuf awareness | Selected descriptor closure is generated, drift checked, fully dispositioned, and interpretation-tested |
| Known regression support | Every retained root Umpire2 behavior has an enabled independent Umpire3 counterpart |
| Deterministic plans | Repeated and parallel compilation yields identical paths, ordering, omissions, and digests |
| Great author UX | UX budgets pass with documented first test and representative complex scenarios |
| Non-linear identity | Runtime-learned IDs remain typed and lineage-qualified across partial orders |
| Real implementation conformance | Nexus, Workflow, Workflow Task, Activity, Update, Callback, and relevant cross-entity paths use real Temporal evidence |
| White-box/black-box | Same eligible claims are established through independent in-process and gRPC-only profiles |
| Clock-skew safety | Adversarial skew cannot change causal verdicts |
| Programmable workers | Participant schema maps exhaustively to SDK behavior and cleans up after every outcome |
| First-class faults | Faults are modeled, scoped, covered, minimized, replayed, and safely removed |
| Guided exploration | Templates and bounds yield deterministic candidates plus honest completeness/omission data |
| Guided fuzzing | Novel semantic behavior is prioritized and a violation is minimized and promoted |
| Environment portability | One scenario source/digest runs in local, CI, remote deployment, and eligible canary profiles |
| Production safety | Blocking/crashing drivers cannot exceed enforceable budgets or strand untracked resources |
| Evidence integrity | No established/violated claim lacks identity, causality, source, retention, and cleanup qualification |
| Independence | Layout guard reports no Umpire1/Umpire2/common-Umpire dependency |

## 13. Verification gates

Each implementation change starts with the smallest affected tests and ends with the focused Umpire3
gate. Code changes follow repository requirements, including `-tags test_dep`, import formatting, and
linting.

The release ladder is:

1. Lean unit proofs, executable equivalence, and negative compile/mutation controls;
2. generator unit tests and generated-diff checks;
3. protocol/catalog/compiler property and fuzz tests;
4. generic runtime, evidence, cleanup, replay, and minimizer tests;
5. fake-adapter failure tests, including non-cooperative and contradictory-evidence cases;
6. real local Temporal integration tests;
7. retained Umpire2 plus independent Umpire3 root functional tests;
8. remote black-box tests;
9. authorized canary safety tests in a non-production rehearsal environment;
10. `make umpire3-check`, relevant `go test -tags test_dep` packages, formatting, and
    `make lint-code`.

Repository-wide `make unit-test` remains the final integration gate when feasible. A resource-limited
local run is reported as such; cached results or unrelated passing packages are not evidence for an
untested change.

## 14. Source anchors for this plan

The roadmap is grounded in these existing seams and behavioral references:

- Umpire3 semantic kernel: `tests/umpire3/model/Umpire3/{Transition,Executable,Refinement}.lean`;
- current experiment shape/export: `tests/umpire3/model/Umpire3/Experiment.lean` and
  `tests/umpire3/model/Temporal/Experiments`;
- strict protocol/runtime boundaries: `tests/umpire3/protocol` and `tests/umpire3/runtime`;
- current environment and Temporal adapters: `tests/umpire3/environment` and
  `tests/umpire3/temporal`;
- current protobuf slice: `tests/umpire3/cmd/umpire3-api` and
  `tests/umpire3/model/Temporal/API`;
- independence/support policy: `tests/umpire3/layout_test.go` and `tests/umpire3/SUPPORT.md`;
- Umpire2 sparse authoring and completion: `common/testing/umpire/regress` and
  `tests/umpire2/regress`;
- Umpire2 evidence, trace, campaign, canary, and verification references:
  `common/testing/umpire/{trace.go,environment_profile.go,campaign,canary,verify}`;
- Umpire2 Temporal realization and semantic inventory: `tests/umpire2/internal` and
  `tests/umpire2/umpiretest`; and
- root migration inventory: `tests/umpire2_test.go`, `tests/umpire2_probe_test.go`,
  `tests/umpire2_regress_test.go`, the independent `tests/umpire3_test.go`,
  `tests/umpire3_probe_test.go`, `tests/umpire3_regress_test.go`, and the retained Umpire2 helper
  call sites in `tests/nexus_workflow_test.go`.

These are evidence for requirements and patterns. Except for ordinary Temporal infrastructure, they
are not permission for Umpire3 to depend on earlier Umpire implementations.
