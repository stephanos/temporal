# Temporal Lean model

Lean owns behavioral meaning in this directory. Generated declarations describe API and dynamic
configuration structure; handwritten modules decide what that structure means. Runtime clients,
credentials, worker resources, callback authority, and live identifiers remain outside the model.

## Generated structural catalogs

`umpire-gen-lean-api` consumes serialized protobuf descriptor sets and exclusively owns
`Temporal/API.lean` and `Temporal/API/`. The generated modules describe messages, enums, maps,
oneofs, presence, recursion, and unary gRPC methods behind the stable `Temporal.API` facade. They do
not provide clients or assign product behavior.

`umpire-gen-lean-dynamic-config-catalog` snapshots Temporal's initialized production registry and
exclusively owns `Temporal/DynamicConfig.lean` and `Temporal/DynamicConfig/`. Handwritten
interpretation and validation live under `Temporal/System/Configuration/`; Callback and Matching
semantics remain in their owning consumer packages.

From the repository root, regenerate the catalogs through their owners:

```sh
make umpire-gen-lean-api
make umpire-gen-lean-dynamic-config-catalog
```

Each owner validates its complete candidate output before replacing managed files.

## Case production

The checked-in Testpilot `.proto` closure is the sole Case schema. `Testpilot.Protocol` exposes its
generated Lean declarations, `Testpilot.Authoring` constructs generated Cases through context-safe
helpers, and `Testpilot.ProtoJSON` delegates the one canonical codec policy to `Protobuf.Json`.
`Umpire.Provenance` retains only Umpire-owned definition bindings, fingerprints, sources, Known
Gaps, and Correlated Rule bindings, and lowers them into the Case's typed provenance rows, which the
runtime never reads.

`Temporal.Feature.Nexus.Caller` authors the Nexus caller-side operation, and its functional set
produces the seven caller Cases through `Temporal.Case.Realization.asyncNexus`: each Contract carries
no monitor Rule, only the Correlated capability the checked Property lowered into.
[AUTHORING.md](AUTHORING.md) walks that Model file from an empty file to a green live test, one
command at a time; its canary set is admitted through the same block and its exploratory set's
coverage targets are pinned by a golden. `Temporal.Testpilot` supplies the unrelated `GetSystemInfo` example, the
worker-outage fault Case, and the six public-facade conformance fixtures. `Temporal.Tool.Testpilot` forwards rendering
to `Testpilot.ProtoJSON`. The broader Nexus success Markdown sketches remain design material rather than executable
coverage. Lean is the first Producer, while the Case format and Go runtime remain independent of
Lean.

Umpire-backed Producers lower checked semantics into generated values and pass them to
`Umpire.Case.Compiler` for source-bound Property validation, the coverage the Case requests,
Case-local names and model value spellings (`Umpire.Case.LocalNames`), exact provenance rows, and
final generated Case assembly. The Testpilot-only synthetic Producer assembles its generated Case
directly.

Exact Case 1.0 is the only admitted format. A Program's symbolic binding graph is the set of binding
IDs its roles and expressions reference, derived at preparation; a resource-free Program references
none. Producers name stable namespace, task-queue, and named Nexus endpoint binding IDs on roles and
references; they do not embed the physical resource names. Producers also omit what preparation
derives: outcome fields, activation reservations, and instruction limits equal to the Profile's
instruction defaults. These IDs are resource references, not transport
addresses. Rebinding an unchanged Case does not change its canonical bytes, Contract, Behavior
Fingerprints, or producer provenance rows.

Testpilot terms have precise boundaries:

- Slots are immutable, single-assignment private execution state. They are not recorded
  automatically.
- Observations are declared typed Run Event fields available to Contracts. Declared response
  reads are ordinary data; Slot privacy does not imply general response secrecy.
- Contract expiry is checked before transitions on every event kind. Captures are bounded and
  isolated per rule and per Run.
- Run disposition, cleanup status, and Verdict remain independent. A proved violation survives
  later cleanup failure.

## Model authoring and Search

The retained model uses separate `Umpire.Model`, `Umpire.Property`, `Umpire.Scenario`,
`Umpire.Query`, `Umpire.Search`, `Umpire.Variations`, `Umpire.Exploration`, and `Umpire.Promotion`
APIs. A checked Model owns behavior; Properties state Trace claims; Scenarios constrain Trace
shape; Queries ask bounded questions and Search answers them; Variations and Exploration select
finite candidates. None of these performs runtime I/O.

For ordinary authoring, use `import Umpire` or the focused `Umpire.Model`, `Umpire.Property`,
`Umpire.Scenario`, and `Umpire.Query` facades. These retain the finite-table/machine helpers and
syntax-aware checkers.
When implementing evaluation or Search over an already checked Model, use
`Umpire.Model.Check`; it exposes the authoritative Machine and finite Search contracts without
loading the model elaborator. `Umpire.Search` consumes checked Queries and does not supply the
authoring conveniences of the other facades. `Search.admit` checks a Property, an optional
Scenario, Known Gaps, and a Query against one checked Model and returns an `AdmittedQuery`, whose
search view only its own `search`, `searchWithIntent`, and `analyzeBranches` use. The
[ownership guide](Umpire/ARCHITECTURE.md#model-ownership-and-semantic-imports) explains how pure
admission and serialization support both paths.

`Umpire.Promotion` remains scenario-neutral. It re-answers an unchanged admitted Query, validates
the complete Plan anchor and exact source bytes, and returns an opaque review-only source value.
It has no Case execution authority and imports no Temporal scenario.

The `umpire-inspect` executable exposes the caller Model's Queries and the Switch example
(`make umpire-list`, `make umpire-explain QUERY=<id>`) and emits deterministic Plan Artifacts. Generated Views remain navigation and test wrappers around that Plan data; they do not
execute a Case or determine a Verdict.

### Ordinary Nexus authoring

`Temporal.Feature.Nexus` is the compiled established walkthrough. Read
`Lifecycle.Semantics`, `Lifecycle.Model`, the three `Operations` modules, and `Observation` in that
order. The finite `Lifecycle.finiteMachine` is the ordinary Model seam: authors still provide the
ordered domains, encoders, enumerators, closure proofs, and Action-executability proof, while
`modelSpec` and `draftModel` remove repeated record and planning transport. Authors who
need an independently specified authoritative relation can use the expert `Machine` path.

Property, Scenario, Query, and Evidence inputs remain ordinary values. Call each language's
`check` operation to inspect its typed `Except` error. Property, Scenario, and Query `checked`
operations take checker-success evidence that defaults to `native_decide`, so a closed valid
declaration names no proof and an invalid one fails where it is written; Evidence's `checked` still
takes the evidence explicitly. To search, `Search.admit` runs the Property, Scenario, Known Gap,
Query, and search-view checks in order and reports the first that rejected. Stable `DefinitionId`
suffixes, source locations, providers/connectors, Model-owned outcomes, and stage-specific `Limits`
values are authored choices; declaration order and instance search choose none of them. Search
returns `Except KnownGapError PlanResult`.
An optional checked `authoredKnownGaps` set is composed with phase gaps before search or artifact
publication. Gaps describe limits and missing evidence; they cannot make a Property pass or imply
that an omitted limitation was detected.

Operation-correlated response Properties can use `correlated_response%` inside
`Property.correlatedRules`. For example, with Model-owned `request` and `response` values:

```lean
correlated_response% (family.id "property" "response") at source
  whenever (.selectedActionIs request) eventually (.outcomeIs response)
  within 1
  correlated [runField] by operationField closing .«partial»
```

This is the same `PropertyCorrelatedClause` as the typed record constructor. `property% spec against
context tracking [...]` resolves references and reports admission failures at the author expression;
`spec.check context` provides the ordinary `Except` path for parameterized inputs. Authors choose
the trigger, response, scope, operation key, natural bound, and endpoint. The checked projection and
`Umpire.Case.Correlated.lower` derive executable Contract data and correspondence evidence without a
separately authored monitor. A response at the trigger or inclusive deadline satisfies that obligation;
only admitted transitions of the same operation advance its clock. Incomplete execution never invents
a deadline, and an already proved violation survives cleanup failure.

The generated correlated corpus includes executable non-cancellation Cases qualified through public
Go `Prepare`/`Run`, including repeated/concurrent Runs and bounded tenfold loads. Its synthetic
source is a controlled qualification fixture, not a production Implementation Link. The existing
Nexus success integration remains the live Temporal demonstration. Nexus operation cancellation
Models, adapters, capabilities, and Cases are explicitly deferred to fn-79.

### Typed operation authoring

An author who needs to name a concrete API operation and relate its fields references the generated
declaration rather than a method string. `Temporal.API.bindUnary` admits a candidate schema only
against the generator's own selection for that method, so a wrong-method pairing, a forged request
or response closure, and an unsupported streaming shape each reject with their own diagnostic.
`Umpire.Operation.ActionTemplate` carries the admitted binding, and `ParameterDomain.check` admits
an explicit list of exact request values as parameterized Action instances. Selecting an Action
never selects its outcome: the authoritative Model still owns which result each instance admits.

Two claims are declared separately and reported separately.

- The **finite domain** is exactly the authored list, under a `ParameterCoverage` of `.sampled`
  (representative cases) or `.fixed` (one exact value). An `.abstracted` claim rejects, because no
  transition and Property preservation evidence exists for it. Ten samples are still ten samples;
  enlarging the list never turns a sample into an exhaustive claim.
- The **runtime admission scope** is a separate `RuntimeScope`. `.samplesOnly` admits nothing the
  finite domain did not list; `.schema bounds` admits any value inside its own declared depth, byte
  and collection bounds. A value past those bounds is reported `outOfScope`, which is a different
  answer from the checker exhausting its own resource ceiling — that one is owned by the value
  layer. Neither answer enlarges the finite claim.

Field operands read the exact schema-defined value, not a summary of it. Supported forms are nested
message access, implicit- and explicit-presence reads, oneof selection, enum values, repeated index
and cardinality, keyed map lookup, concrete bytes, and the integer kinds the descriptor declares.
Two byte strings of equal length stay distinguished, an absent map key is absent rather than an
empty default, and presence follows the descriptor rather than the author's expectation. Unsupported
forms reject with the responsible field and reason rather than substituting an approximation:
floating-point comparison rejects (`unsupported floating-point evaluation`) while its metadata stays
discoverable, a recursive value past its declared depth is incomplete rather than an empty subtree,
an integer outside its declared range rejects before protobuf lowering, and a closed enum value the
descriptor does not name rejects while an open enum retains its unknown number.

A Property relates those operands. Same-step clauses compare the selected Action's own immutable
request against the modeled result and resulting state; cross-step clauses bind a named earlier
occurrence under an explicit operation key and compare the captured value. Unmentioned fields are
unconstrained: a partial conjunction is not an exhaustive field contract. Missing evidence never
satisfies a comparison — it is rejected or left unresolved.

Lowering is checked in both directions before any Driver I/O. `Umpire.Case.Coverage` binds each
modeled input field to the exact request assignment that constructs it, so a Program that stopped
constructing a covered field rejects the whole Case; `Umpire.Case.Projection.lower` derives the
monitor rule from the checked Property -- the read path of each observed operand from the declared
Observation's own message, the comparison and the literal the Program assigns
-- so moving a coordinate in the Property moves the runtime read with it.

Two authored examples carry this end to end:

- [`Temporal/Feature/Workflow/Start/Model.lean`](Temporal/Feature/Workflow/Start/Model.lean) (the
  typed unary example, re-authored with the commands by fn-86 .2) binds `startWorkflow` to the
  generated `StartWorkflowExecution` request through `schema:` and writes one
  `relates: startWorkflow.input.workflow_type.name = workflowExecutionStarted.workflow_type.name`
  line, resolved against the generated descriptors while the file compiles; the recorded field is
  read through the generated `GetWorkflowExecutionHistory` response schema. Its derived rule
  reports all three answers: an agreeing recorded type is satisfied, a disagreeing one is violated,
  and an event that never establishes the field leaves the rule pending.
- [`Temporal/Feature/Nexus/Pair/Model.lean`](Temporal/Feature/Nexus/Pair/Model.lean) (the typed
  Nexus example, re-authored with the commands by fn-86 .3) runs two instances of the caller
  Model's operation in one workflow, `instances: 2` on its Scenario, and writes
  `relates: nexusOperationCompleted.scheduled_event_id = nexusOperationScheduled.event_id`: the
  scheduled event is an earlier step's, so each instance's rule captures its own -- selected by
  the operation name its schedule command assigned -- and matches the completion's reference
  against the retained event's id.

Environment binding stays outside the model. Namespaces, task queues and named Nexus endpoints are
symbolic in the Program and supplied by the Profile; a semantic relationship that involves one is
declared in the model and preserved under binding. Nexus operation cancellation commands,
confirmation, and resolution are not modeled here and remain deferred to fn-79.

Known Gaps disclose what a Case does not check; they never waive a requested clause. The
two-operation Nexus Query declares two: a completed history event carries no operation identity,
so the operation key of the lifted evidence is the scheduled event a completion references; and a
completion referencing another scheduled event leaves its rule pending rather than violated, because
separating the two would need a correlation condition the model never declared, while the model
Property still distinguishes them.

Lean syntax used by the walkthrough:

- `:=` defines a value; `{ base with field := value }` makes a record update.
- `.case` selects an inferred enum or structure constructor.
- `Except Error Value` is either `.error error` or `.ok value`; `do` and `←` stop on the first error.
- `by` starts a proof; `checked`'s proof argument defaults to `by native_decide`, the raw/check/checked
  seam.
- `#guard_msgs` compiles an expected elaboration failure; `#print axioms` reports transitive trust.

`Temporal.Feature.NexusTests` compiles the facade-only path: the caller Model's machines, a
Property, a Scenario, a Query, a set and a Case, reached through `Temporal.Feature.Nexus` alone.
The coverage record is [the caller Model's](Temporal/Feature/Nexus/Caller/COVERAGE.md). The
first-generation lifecycle, Operations, Observation, Experimental and race-prototype modules were
retired by fn-86 .5; the race behavior is recorded in fn-79's spec and the exploration inputs in
fn-33's.

## Runtime ownership

The Go runtime is the consumer of canonical Case data:

```text
Lean or another Producer
        │
        ▼
      Case { Program, Contract }
        │
        ▼
testpilot.Prepare(case, Profile) ──▶ immutable PreparedCase
        │
        ▼
PreparedCase.Run(ctx, Driver) ──▶ immutable Run + Verdict
```

The Testpilot `.proto` files own the Case protocol. `common/testing/testpilot` owns the public
Profile, Driver, and two-call facade. The Session, handle, coordinate, role-policy, and Opcode types
that facade exposes are aliases of its `common/testing/testpilot/contract` leaf, which private execution
shares, so no translated copy of the Driver contract exists.
Testpilot's private execution package owns scheduling, recording, effect lifecycle, private Slot state,
and bounded cleanup. Its private verification package owns Contract preparation, fresh Run-local
Monitors, and offline evaluation. Umpire owns its semantic model and the provenance rows it writes;
each Lean Producer owns its checked lowering, the shared compiler completes Umpire-backed Case
assembly, and Go `testpilot.Prepare` owns Case admission.

Temporal authority remains split:

- `common/testing/testpilot/temporal/server` supplies the authorized descriptor catalog and transports prepared
  unary method/request pairs, returning raw typed responses and protocol status.
- `common/testing/testpilot/temporal/worker` owns SDK workflow, activity, and Nexus-handler interpretation,
  reserved activation delivery, and activation-level cancellation.
- `common/testing/testpilot/temporal` composes server and SDK worker Drivers without interpreting scenario or Contract
  semantics.

Internal execution constructs typed requests and applies declared response reads to private
Slots and Run Observations.

`Prepare` snapshots Profile-owned physical binding values and resolves private prepared request and
role data without Driver I/O. Prepared/Driver identity includes a deterministic fingerprint over the
complete binding snapshot. `Run` checks that identity, calls the Driver's static no-I/O `Validate`,
creates the Monitor, and only then calls `Open`. The shared Temporal Driver uses prepared resources in
exact Case 1.0 resources as the sole source of namespaces, task queues, and named Nexus endpoints.
Transport targets, credentials, callback authority, SDK clients, and lifecycle configuration remain
physical Driver inputs.

## Generated artifacts

The checked semantic inventory is the generated navigation view
[`INVENTORY.md`](INVENTORY.md). Catalog consumers explicitly import
`Umpire.Inventory` or a focused inventory module; `import Umpire` does not include it.
Semantic owners publish classifiers through neutral `Umpire.OutcomeClassification` contracts and
carry mappings through `Umpire.KnownGap`. The inventory consumes their declarations without owning
stage behavior. `make lint-model` enforces this dependency direction, including transitive paths,
and the neutral module's minimal foundation. The
[ownership guide](Umpire/ARCHITECTURE.md#artifact-and-generated-view-boundaries) describes the
exhaustive classifiers, stage not-run marker, and exact versus lossy carry contracts.

Its owner commands are:

```sh
make umpire-gen-inventory
make umpire-check-inventory
```

The retained planning Generated Views are owned transactionally:

```sh
make umpire-gen-regression-views
make umpire-check-regression-views
```

The Testpilot conformance and example trees are independently owner-managed by the same tool:

```sh
make umpire-check-case-runtime-conformance
make umpire-gen-case-runtime-conformance  # separate reviewed promotion
model/.lake/build/bin/umpire-case nexusCallerTests-asyncCompletion
mise exec -- go test -count=1 -tags 'test_dep integration' ./tests -run '^TestTestpilotNexusCallerAsyncCompletion$'
```

The live Nexus success selector prepares the same canonical Case bytes against two Profiles, runs both
environments concurrently, verifies namespace isolation and correlated endpoint history, and obtains
the same satisfied Contract result. Its binding fingerprints and Driver identities differ because
their physical resources differ.

The check builds the Lean renderer, creates and validates both the complete twelve-file conformance
tree and the two-file `tests/testcore/testpilot/testdata` example tree under one physical temporary
root, and recursively diffs each independently owned root against the checkout. The promotion target
is a separate action. Ordinary Go tests only read checked-in fixtures; they invoke neither Lean nor
a rewrite mode.

The corpus contains exactly these facade proof classes:

1. satisfied;
2. violated;
3. inconclusive;
4. static preparation rejection;
5. cleanup failure after a proved violation; and
6. cross-Run isolation.

Lean-produced Cases compare byte-for-byte. Runtime results compare through one named closed stable
projection, while Run IDs, elapsed values, event/source identities, causal references, activation
identities, support references, and diagnostics are checked structurally. There is no generic
normalization or ignore mechanism.

## Build and regression

From the repository root:

```sh
make umpire-build-model
make umpire-check-regression
make fmt-imports
make lint-code
```

The aggregate regression check regenerates owner-managed artifacts into temporary roots, checks the
active vocabulary and semantic inventory, runs every package under `tools/umpire` with
`-tags test_dep`, builds the complete Lean roots including generic promotion and the Case renderer,
and runs the complete live selector with `-tags 'test_dep integration' -run '^TestTestpilot'`. The live
gate compares the entire inherited failure-identity set, so both additions and deletions fail.

`make lint-model` runs Lean declaration linting and validates the complete first-party import graph.
The regression boundary intentionally adds no broad generated-Lean API drift check and no new
GitHub Actions surface.

## Superseded runtime history

The pre-fn-64 portable-plan, resident-executor, caller-specific adapter, and separate Run Evaluation
interfaces were removed. Historical planning documents label those names explicitly as superseded;
they are not supported runtime entry points or compatibility targets.
