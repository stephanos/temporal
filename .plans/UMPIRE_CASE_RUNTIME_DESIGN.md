# Umpire Case Runtime design

Status: approved design, pending implementation planning

This document defines the replacement for the current `PortableTestPlan`, portable-evaluation, and
caller-closure execution path. The replacement keeps Umpire as the umbrella while separating the
portable instruction language, execution, verification, and Temporal-specific runtimes into deep,
independently testable modules.

The terminology in [`tools/umpire/CONTEXT.md`](../tools/umpire/CONTEXT.md) applies throughout this
design. Implementation must update the normative Umpire specification and architecture documents
as part of the cutover; this document does not create a second permanent source of truth.

## Problem

The current vertical slice encodes scenario-specific behavior in Go. The Nexus adapter knows the
caller-closure scenario, portable evaluation implements property operators in Go, and generated
tests bind those pieces together. Consequently, adding model behavior often requires changing both
Lean and Go, even when the Go runtime should only be interpreting a plan produced by Lean.

The desired boundary is different:

- a Producer creates a bounded Case containing all execution and verification semantics;
- the Executor interprets the Program without knowing the scenario;
- the Evaluator interprets the Contract without reimplementing its properties;
- a Host supplies authorized side effects and environment bindings;
- Temporal server and worker integrations implement different Host capabilities;
- the resulting Run contains only generic lifecycle data and explicitly declared Observations.

Lean is the first Producer, but the format and runtime do not depend on Lean. A conforming client
may construct the same IR directly and use Umpire without the model toolchain.

## Goals

1. Express bounded Temporal interactions as typed data rather than scenario-specific Go code.
2. Invoke any authorized unary protobuf RPC through one dynamic instruction.
3. Keep workflow, activity, and Nexus-handler execution on Temporal SDK APIs only.
4. Compile safety and bounded-liveness properties into deterministic monitor machines.
5. Use the same Contract semantics during execution and for offline evaluation.
6. Validate a Case once, then reuse its immutable prepared form for many sequential or concurrent
   Runs.
7. Abort ordinary execution immediately and unconditionally on any safety violation while still
   performing bounded cleanup.
8. Keep execution, verification, and Temporal server/worker integrations independently testable.

## Non-goals for the first version

- Caller-closure behavior or compatibility with its plans, adapters, fixtures, or tests.
- A compatibility reader for `PortableTestPlan`.
- Streaming RPCs.
- Unbounded loops, implicit retries, or general-purpose scripting.
- General JSONPath, filters, functions, or arbitrary expression evaluation.
- SDK-side self-verification or a workflow event-reporting protocol.
- Durable recovery of an active Run after the Umpire process crashes.
- Replay/audit payload digests.
- Cross-process transport for private worker-to-controller Slots.

## Domain model

The umbrella and public namespace remain **Umpire**.

- A **Producer** creates a **Case**. The Lean Producer is a Compiler.
- A **Case** is exactly one **Program** and one **Contract**.
- A **Program** is a bounded acyclic graph of typed instructions.
- A **Contract** is a set of deterministic safety and bounded-liveness monitors.
- An **Executor** interprets a Program through a **Host**.
- A **Host** binds symbolic roles and performs concrete primitive interactions.
- One execution attempt produces an append-only **Run** of immutable **Run Events**.
- An **Evaluator** applies a Contract incrementally or offline and returns a **Verdict**.

The vocabulary deliberately distinguishes operational data from evidence:

- A **Slot** is immutable, single-assignment typed data consumed by later instructions. Slots are
  private execution state and are not recorded automatically.
- An **Observation** is a declared typed field on a Run Event. Contracts may inspect Observations;
  they cannot inspect arbitrary Slots or raw request and response payloads.

## Module boundaries

The implementation should converge on this package topology:

```text
api/umpire/v1
  value.proto       typed values, expressions, paths, Slots, Observations
  program.proto     Program, entrypoints, nodes, instructions, limits
  contract.proto    deterministic monitor machines
  run.proto         Run, RunEvent, disposition, Verdict support
  case.proto        Case and version envelope

tools/umpire
  Profile and Host  public adapter contract
  PrepareCase       thin Host-agnostic facade
  PreparedCase.Run  execution plus authoritative Contract evaluation facade

tools/umpire/internal/execution
  admission, PreparedProgram, Executor, private driver and Monitor interfaces,
  DAG scheduler, Slot store, recorder, abort/drain/cleanup

tools/umpire/verification
  PreparedContract, per-Run Evaluator, live Monitor implementation,
  offline evaluation

tools/umpire/temporal
  composite Temporal Host and Profile

tools/umpire/temporal/server
  protobuf descriptor catalog, symbolic server endpoints, authorization,
  channels, credentials, dynamic unary transport, Nexus completion client

tools/umpire/temporal/worker
  generic worker registration and lifecycle, workflow/activity/Nexus-handler
  interpreters, SDK-only effects, private Slot publication

tools/umpire/internal/ir
  shared private type, expression, and path machinery
```

Internal `execution` must not import `verification`, the root facade, or a Temporal package.
`verification` implements execution's private Monitor contract. The root facade owns the public
Profile, Host, and effect-handle contract, translates that adapter to execution's private driver,
and composes execution with verification. Temporal packages implement the public Host contract and
are never imported by the root. This dependency direction avoids an import cycle while Go's
`internal` visibility prevents ordinary external callers from assembling the scheduler, recorder,
Slot store, or Monitor machinery themselves. Monitor injection remains an internal composition and
test seam.
There is no public third Nexus runtime: Nexus handlers share worker registration and lifecycle, so
they belong to `temporal/worker`. Controller-side Nexus completion is a server-side transport
capability.

## Case and Program IR

### Case

A Case contains:

- one schema/version envelope;
- one Program;
- one Contract;
- stable Case metadata needed for diagnostics and provenance;
- no addresses, credentials, live clients, mutable runtime state, or callback implementation.

The Case format is independently usable. Model provenance may be supplied and validated by a
trusted host policy, but neither Lean nor model provenance is required for plan-local execution.

### Program structure

A Program declares:

- symbolic endpoint, worker, task-queue, and participant roles;
- typed Slot and Observation schemas;
- global bounds and policy requirements;
- multiple named entrypoint DAGs tagged with their execution context;
- a separate always-run cleanup DAG.

Supported execution contexts are controller, workflow, activity, and Nexus handler. Each node has:

- a stable node ID;
- dependency node IDs;
- an optional typed Boolean guard;
- exactly one closed instruction variant;
- typed output assignments;
- applicable per-instruction bounds.

The graph is acyclic. Repetition is either unrolled into explicit nodes or represented by a closed
bounded primitive whose attempts are recorded. There are no hidden loops or implicit retries.

One Program can therefore describe the controller actions that start a Workflow, the Workflow code
run by a generic SDK worker, and the Nexus handler code run by that worker without conflating their
capabilities.

### Capability matrix

Admission rejects an instruction used in the wrong execution context.

| Instruction | Controller | Workflow | Activity | Nexus handler |
| --- | ---: | ---: | ---: | ---: |
| `InvokeRPC` | yes | no | no | no |
| `AwaitSlot` | yes | no | no | no |
| `CompleteNexusOperation` | yes | no | no | no |
| `StartNexusOperation` | no | yes | no | no |
| `Await` | no | yes | no | no |
| `Finish` | no | yes | no | no |
| `RespondNexus` | no | no | no | yes |

The initial implementation should add only instructions required by the async Nexus success Case.
Future SDK effects extend the closed instruction union and the relevant context interpreter only
when a concrete use case requires them.

### Dynamic unary RPC

`InvokeRPC` mechanically supports every unary protobuf method in the Host's pinned descriptor
catalog. The instruction specifies:

- a symbolic endpoint role;
- the fully qualified service and method;
- typed request assignments;
- declared response projections;
- explicit call bounds.

The Host resolves the role to an endpoint, channel, credentials, and authorization policy. Program
data cannot name an address or transport credential. Supporting every method mechanically does not
grant authority: preparation checks the Host Profile's allowed endpoint roles and method patterns.

Streaming methods are rejected in the first version. Pagination and long polling are expressed as
explicit bounded nodes rather than hidden in the transport.

`CompleteNexusOperation` is separate because Nexus operation completion uses the Nexus completion
protocol rather than an ordinary protobuf RPC. It remains controller-side and is implemented by the
server runtime's Nexus client, never by a workflow SDK effect.

### Typed assignments and paths

Assignments use typed expressions made from literals, immutable references, and path projections.
Paths address protobuf payload fields, for example `foo.bar.baz`; they are not filesystem paths.

The first path grammar supports:

- singular field traversal;
- `[*]` fan-out over a repeated field;
- a literal map key;
- explicit presence and oneof selection.

It excludes filters, computed indexes, functions, and general JSONPath. Preparation resolves each
path against the exact request or response descriptor and validates presence, cardinality, and type
compatibility before any I/O.

A response projection may assign a value to a Slot, emit it as a declared Observation, or do both.
`EmitEach(response.history.events[*])` emits one Run Event per repeated element and applies declared
relative projections to that element. No full response or response digest is retained by default.

## Contract IR

Lean compiles rich model properties into bounded deterministic monitor machines. Go implements one
generic transition interpreter; it does not contain property-specific operators or Temporal/Nexus
checks.

Each Contract rule declares:

- a stable rule ID;
- safety or bounded-liveness kind;
- a finite state set and initial state;
- ordered transition cases over Run Event metadata and typed Observations;
- terminal satisfied and violated states;
- an explicit horizon for bounded liveness;
- references to supporting Run Events.

An unmatched event self-loops. Transition predicates use the same finite typed expression layer as
the rest of the IR. Preparation type-checks every transition and proves that state and work bounds
fit Host policy ceilings.

The Contract may see generic Run metadata, stable execution coordinates, and declared
Observations. It may not inspect undeclared raw payload fields or private Slots. Thus a Producer must
make every fact required by the Contract explicit at the Program boundary.

Safety rules can produce a finite bad prefix. Their first violation returns `Stop` synchronously and
unconditionally. A bounded-liveness rule becomes violated only when its declared horizon closes
without a witness; until then it remains pending. The same transition function and closure rules are
used for live monitoring and offline evaluation.

## Preparation and reuse

`PrepareCase` performs all static work once, before target-side I/O:

- envelope, schema, and version checks;
- size, node, edge, activation, attempt, event, and time bounds;
- stable and unique identifiers;
- DAG acyclicity and dependency validation;
- Slot single-writer, dataflow, guard, and type validation;
- instruction/context capability checks;
- descriptor, method, streaming, path, presence, oneof, map, and repeated-field validation;
- Contract state-machine, expression, horizon, and work-bound validation;
- Host capability and authorization policy checks;
- compilation of descriptor accessors, scheduler indexes, and monitor transition indexes.

Success returns an immutable, concurrency-safe `PreparedCase` bound to the exact immutable Host
Profile and descriptor catalog used for preparation. It contains no Run ID, live client, credential,
Slot value, recorder, or mutable evaluator state.

`PrepareCase(case, profile)` performs static work only; it does not accept a live Host. It prepares
and binds the Contract's immutable default Monitor factory. Each `PreparedCase.Run(ctx, host)`
validates the live Host identity and creates a fresh binding set, logical Host session, Slot store,
recorder, and Contract Monitor before Run creation. Failure to instantiate that already-prepared
Monitor is an internal pre-Run invariant failure with no target effects. Long-lived clients,
channels, and registered workers may be shared by the Host across Runs. The same PreparedCase can be
used sequentially or concurrently for canaries without data crossing Run boundaries.

A different Host Profile or descriptor catalog requires a new preparation. Endpoint unavailability,
credential expiry, and target-state changes after preparation are runtime outcomes, not reasons to
repeat static validation.

An admission error returns no Run and must cause no target-side effects. Once a Run is opened,
failures are represented by the Run, its disposition, diagnostics, cleanup outcome, and Verdict.

## Executor, Host, and Monitor boundary

The Executor owns Program semantics:

- dependency scheduling and graph bounds;
- instruction dispatch by context;
- typed request construction and response projection;
- Slot single assignment;
- Run Event recording and source-ID deduplication;
- synchronous Monitor calls;
- ordinary cancellation, bounded drain, and cleanup scheduling.

The Host owns concrete resources and side effects:

- binding symbolic roles to a Temporal environment;
- channels, SDK clients, workers, task queues, and Nexus handlers;
- credentials, transport metadata, descriptor catalogs, and method authorization;
- actual primitive execution and resource shutdown.

Host dispatch must return a Host-owned effect handle within its context bound. The Executor waits
and cancels through that handle; it does not wrap a potentially blocking synchronous Host call in an
unbounded goroutine. If an effect does not finish during drain, the Executor stops waiting and the
Host quarantines the handle under a Profile-wide concurrency ceiling until it terminates. Exhausting
that ceiling rejects new dependent work. A Host method that itself ignores its context violates the
Host contract; Umpire cannot guarantee bounded closure for a non-conforming Host.

The internal execution module defines narrow MonitorFactory and Monitor callbacks. The immutable
factory is bound to the prepared Program view and creates one Monitor per Run. A Monitor receives
immutable appended Run Events and returns only `Continue` or `Stop`. On Run closure it returns a
Verdict or an error. It cannot mutate the Program or Run, schedule instructions, change evidence, or
suppress cleanup. `verification.Evaluator` supplies the only production factory. Internal fake
factories exercise scheduling and failure paths without creating a caller-visible way to replace a
Case's Contract.

Binding and validating the Monitor against the prepared Program occurs before I/O. `Observe` is
called synchronously after an event is atomically appended to the in-memory Run and before the next
ordinary dispatch decision. Monitor implementations are runtime configuration and are never
serialized inside the Program.

Every Monitor method accepts an Executor-bounded context and a conforming Monitor must return when
that context is cancelled. The Executor does not wrap a synchronous Monitor callback in a goroutine
to manufacture a timeout. If a Monitor returns an error or times out through its context, ordinary
execution stops, cleanup runs, the Run is incomplete, and the Verdict is inconclusive unless an
earlier violation was already proven. A test fake that ignores cancellation violates the internal
Monitor contract and has no bounded-closure guarantee.

## Run and Verdict

A Run is append-only. Every Run Event carries:

- a monotonic observation sequence;
- Executor-recorded monotonic elapsed time since Run start;
- stable coordinates: entrypoint, activation, instruction, and attempt;
- a generic event kind and lifecycle data;
- explicit causal references where applicable;
- a stable source ID for deduplicating at-least-once publications;
- only the Observations declared by the Program.

Instruction timeouts and Run closure append explicit events with their elapsed coordinate. A time
horizon is evaluated only from those recorded coordinates; the Contract does not arm an invisible
timer. Offline evaluation therefore consumes the same timeout/closure facts as live evaluation and
never reconstructs elapsed time from target timestamps.

Late outcomes accepted during the bounded drain are appended normally. Events arriving after Run
closure are rejected and diagnosed; they cannot mutate the closed Run.

Run disposition is one of:

- `completed`: ordinary execution and closure completed;
- `stopped_by_monitor`: a Monitor requested a stop after proving a violation;
- `incomplete`: the harness could not complete an authoritative execution.

Cleanup has an independent outcome and diagnostics. It never replaces the root Run disposition or
an already established violation.

A Verdict is:

- `satisfied` when all rules terminate successfully and ordinary execution completes;
- `violated` when any rule proves a violation;
- `inconclusive` when no violation was proved and the Run or evaluation is incomplete.

A proven violation dominates later monitor, harness, or cleanup failures. Umpire must not erase a
finite bad prefix because subsequent shutdown was imperfect.

| Ordinary/monitor outcome | Run disposition | Cleanup/Host close | Verdict |
| --- | --- | --- | --- |
| all instructions and rules complete | `completed` | succeeded | `satisfied` |
| target or protocol non-success accepted by the Contract | `completed` | any recorded outcome | Contract result |
| safety or closed-horizon liveness violation | `stopped_by_monitor` | any recorded outcome | `violated` |
| execution/recorder/invariant failure before a violation | `incomplete` | any recorded outcome | `inconclusive` |
| Monitor error or timeout before a violation | `incomplete` | any recorded outcome | `inconclusive` |
| drain expiry after a proved violation | `stopped_by_monitor` | any recorded outcome | `violated` |
| cleanup or Host close fails after the ordinary outcome is fixed | unchanged | failed with diagnostics | unchanged |

## Abort and cleanup semantics

On a safety violation, the Executor:

1. appends the triggering Run Event;
2. receives `Stop` from the Monitor;
3. dispatches no new ordinary nodes;
4. cancels Host-owned handles for in-flight ordinary effects;
5. performs a bounded drain and records accepted late outcomes;
6. runs the always-run cleanup DAG with a fresh bounded context;
7. closes the Host session and Run.

Already committed external effects remain committed. Cleanup is compensating behavior, not a
transaction rollback. A second Monitor `Stop` cannot suppress cleanup. Cleanup nodes are bounded and
their failure is reported independently.

## Temporal runtime boundaries

### Server runtime

`temporal/server` implements controller-side network capabilities. It owns the pinned descriptor
catalog, symbolic endpoint resolution, authorized gRPC channels, credentials, transport metadata,
and dynamic unary invocation. Its primitive result is generic protobuf output plus protocol status;
it does not interpret scenario meaning.

The same runtime owns controller-side Nexus completion through a Nexus client. It does not import
the worker runtime or SDK workflow APIs.

### Worker runtime

`temporal/worker` owns generic SDK workers and the interpreters for workflow, activity, and Nexus
handler entrypoints. Workflow interpretation calls replay-safe workflow SDK APIs only. It never
invokes server RPCs, opens arbitrary network clients, reads wall-clock time directly, or emits a
private execution log for verification.

Nexus handlers belong here because they share registration, routing, and lifecycle with workers.
`RespondNexus` may return a synchronous result, asynchronous operation information, or an error.

### Private Slot bridge

The async Nexus Case requires the handler to transfer completion authority to the controller. The
worker Host stores the callback URL, headers, and token behind an opaque Run/activation-scoped
capability Slot. Program expressions cannot read, traverse, copy, project, or observe the capability;
`AwaitSlot` can observe readiness and `CompleteNexusOperation` is its only consumer. The initial Host
may implement the bridge in-process, but its interface must permit a transport-backed implementation
later.

Capability Slots are operational authority, not evidence, and are destroyed at session closure.
The Contract sees only separately declared Observations derived from authoritative server results.

### Verification evidence

The first version verifies SDK-triggered behavior from authoritative server-side history, state,
and RPC outcomes. The workflow and handler do not self-report internal events. SDK-only behavior
that leaves no server-visible effect is outside the first-version verification boundary; a future
observation protocol may add it without changing Program execution semantics.

## First vertical slice: async Nexus success

The first Lean-produced Case exercises every important boundary without embedding scenario logic in
Go:

1. The controller entrypoint uses `InvokeRPC` to call `StartWorkflowExecution`.
2. A generic worker dispatches the Program's workflow entrypoint.
3. The workflow uses `StartNexusOperation` and `Await` through the workflow SDK.
4. The generic Nexus handler uses `RespondNexus(async)` and publishes an opaque completion
   capability to a private Run-scoped Slot.
5. The controller uses `AwaitSlot`, then `CompleteNexusOperation(success)` through the server-side
   Nexus client.
6. The controller performs explicitly bounded `GetWorkflowExecutionHistory` calls with
   `InvokeRPC` and uses `EmitEach` to record declared server-history Observations.
7. The workflow finishes through the SDK.
8. The Contract recognizes the expected lifecycle and correlations from server-history
   Observations and returns a Verdict.

The server and worker sides use logically distinct task queues and endpoints even if the initial
integration test runs them in one process. No Nexus lifecycle checker or callback-closure adapter is
implemented in Go.

## Failure semantics

Protocol and application outcomes are facts for the Contract, not implicit test failures:

| Condition | Classification | Run effect | Contract effect |
| --- | --- | --- | --- |
| Any gRPC status, including non-OK | instruction outcome | record and follow the explicit graph | Monitor decides |
| SDK application failure or Nexus response | instruction outcome | record server-visible outcome | Monitor decides |
| Bounded `Await` or `AwaitSlot` timeout | instruction outcome | record and follow the explicit graph | Monitor decides |
| Missing required Slot at an unguarded consumer | execution failure | incomplete; stop ordinary graph | inconclusive unless already violated |
| Worker startup, recorder, invariant, or global-limit failure | harness failure | incomplete; cleanup | inconclusive unless already violated |
| Monitor error or timeout | verification failure | incomplete; abort and cleanup | inconclusive unless already violated |

Invalid Cases, unsupported instruction/context pairs, authorization failures, and Host capability
mismatches fail preparation before Run creation or target-side I/O.

## Security and operational limits

- The Host Profile grants endpoint roles and method patterns; the Program grants itself no
  authority.
- Transport credentials and metadata are Host-owned and unaddressable by Program expressions.
- Request/response sizes, emitted events, path fan-out, nodes, activations, attempts, calls, waits,
  cleanup, and monitor work all have admission-checked ceilings.
- Raw payloads and ordinary Slots are excluded from the Run unless a declared projection emits a
  typed Observation. Opaque capability Slots can never be projected.
- Prepared descriptor accessors and transition indexes avoid reflection and rule scans on every
  event.
- Channels and workers may be shared, while Slots, Run recording, Host sessions, and Evaluator state
  remain per Run.
- Monitor transition cases should be indexed by event kind to keep incremental work predictable.

At ten times the Run rate, the design scales through PreparedCase reuse and shared Host resources;
per-Run memory remains bounded by declared limits. Host policy provides backpressure through
concurrency ceilings rather than allowing unbounded goroutines or event queues.

In the first version, a process crash can lose an active in-memory Run. The canary supervisor must
report the iteration as lost and must not fabricate a Verdict. Durable recording, resume, replay
audit, and payload normalization are deferred.

## Replacement sequence

Temporary coexistence keeps intermediate commits buildable; it is not supported compatibility.
The final repository has one Case path.

1. Add the new split protobuf IR and generate Go and Lean types. Implement pure type, path, and
   admission machinery.
2. Implement Contract preparation, deterministic monitor evaluation, and exact live/offline parity.
3. Implement the generic execution core against a fake Host, including Slots, recording,
   Monitor-driven abort, bounded drain, cleanup, and concurrent PreparedCase reuse.
4. Implement the Temporal server runtime's dynamic unary transport, authorization, and Nexus
   completion; shared execution/IR code remains responsible for request construction and response
   projection.
5. Implement the Temporal worker runtime's generic workflow and Nexus-handler interpreters and
   strict context capabilities.
6. Compile and prepare an orthogonal `GetSystemInfo` Case, then compile and run the async Nexus
   success Case from Lean through a real Temporal test environment. The preparation-only
   `GetSystemInfo` proof uses an empty request, a typed `server_version` projection, a different
   Contract topology, and the exact authorized WorkflowService descriptor without adding an
   instruction or live scenario.
7. Switch generators, CLI commands, documentation, and regression gates to the Case path. Remove
   `PortableTestPlan`, property-specific `portableevaluation`, the scenario-specific
   `temporal/nexus` package, caller-closure model/fixtures/generated tests, and obsolete adapters.
   Update `UMPIRE4_SPEC.md` so the final normative rules describe only the new architecture.

The removal step must be based on ownership rather than a blind text deletion. General model,
artifact, or regression functionality that remains relevant is moved behind the new boundaries;
caller-closure-specific behavior is deleted. A reviewed cutover ledger accounts for every deleted
top-level legacy Test/Fuzz and inherited failure identity as `preserved`, `replaced`, or
`intentionally-retired`, names its Case Runtime replacement where applicable, and gives an owner and
reason for retirement. An unaccounted row blocks deletion. Scenario-neutral fn-5 checked-promotion
types and validation remain buildable without caller-closure imports; the fixed caller-closure
candidate, command, and binding are removed.

## Case Runtime conformance corpus

Fn-64 establishes a small closed test-only corpus at the public facade rather than performing the
broad test consolidation proposed by fn-63. It contains exactly six proof classes: satisfied,
violated, inconclusive, static preparation rejection, cleanup failure after a proved violation, and
cross-Run isolation. Concurrency, cancellation, descriptor/path grammar, fuzzing, cardinality, and
resource-lifecycle invariants remain focused tests rather than static goldens.

Lean-produced deterministic Case/Contract data compares byte-for-byte. Run-assigned identities,
elapsed coordinates, and other intentional runtime values use a closed named stable projection while
their dynamic fields are validated structurally; there is no generic ignore or normalization
facility. Expected values come from Lean or fixed hand-authored oracle tables that do not invoke the
Go runtime under test. Ordinary Go tests neither invoke Lean nor update expected output.

Fixture generation writes a complete tree to a temporary root, validates it, and diffs it against
the checkout. Interruption or failure cannot partially update checked-in fixtures, and promotion of
new expected output is a separate reviewed action. The complete tagged live gate remains
`-run '^TestUmpire'` and retains the inherited failure-identity policy.

## Verification strategy

### IR and preparation tests

- schema/version, duplicate ID, cycle, missing dependency, and every global bound;
- Slot single writer, missing reference, unguarded unavailable reference, guard typing, and output
  typing;
- every accepted and rejected instruction/context pair;
- unknown, streaming, and unauthorized methods;
- request and response path traversal, presence, oneof, repeated fan-out, literal map key, and type
  mismatch;
- Host Profile and descriptor-catalog binding.

### Executor tests with a fake Host

- dependency order and allowed concurrency;
- generic non-OK instruction outcomes;
- immutable Slot assignment and missing-Slot incompleteness;
- source-ID deduplication for at-least-once outcomes;
- unconditional safety stop, no new ordinary dispatch, in-flight cancellation, bounded drain, fresh
  cleanup context, and independent cleanup failure;
- Monitor errors and timeouts;
- violation precedence over later failures;
- sequential and concurrent reuse of one PreparedCase, including race-enabled tests.

### Evaluator tests

- exact first safety-violation prefix;
- bounded-liveness witness and horizon violation;
- pending rule plus incomplete Run produces inconclusive;
- live and offline evaluation produce identical transitions, supporting-event references, and
  Verdicts;
- event-kind indexing and declared work bounds.

### Temporal server tests

- dynamic invocation against in-process unary gRPC services with multiple request/response shapes;
- transport of prepared requests and raw response/status results for execution-owned Slot and
  Observation projection;
- non-OK status recording;
- method authorization and absence of credentials or injected metadata from Run data.

### Temporal worker tests

- generic workflow interpretation of `StartNexusOperation`, `Await`, and `Finish` using the Temporal
  SDK test environment;
- async `RespondNexus` and opaque capability Slot publication;
- rejection of `InvokeRPC` and other controller instructions in SDK entrypoints;
- no workflow-side verification event channel.

### Producer and integration tests

- Lean compilation emits a reproducible Case and valid monitor machines;
- an unrelated `GetSystemInfo` Case with a different Contract topology compiles and prepares against
  the exact authorized descriptor with zero Host I/O and no runtime specialization;
- Go admission accepts the Lean output and agrees on typed values and paths;
- the async Nexus success integration executes through server and worker runtimes;
- Contract evidence comes only from declared server-history Observations;
- one preparation drives many isolated sequential and concurrent canary Runs.

Focused Go tests use `-tags test_dep`; only cluster integration tests add `integration`. Generated
artifacts must regenerate cleanly. Final gates include the relevant Lean builds, the new Umpire
regression target, `make fmt-imports`, and `make lint-code`.

## Trade-offs

- **Performance:** preparation does more up-front descriptor and graph work, but repeated Runs avoid
  reparsing paths and Contracts. Declared limits trade unrestricted programs for predictable cost.
- **Scalability:** in-memory bounded Run state and reusable Host resources support canaries, but the
  first version cannot resume across process crashes.
- **Complexity:** a typed multi-context IR and monitor machine are more foundational work than a
  bespoke adapter. They remove scenario growth from Go and concentrate complexity behind stable
  preparation, execution, and verification interfaces.
- **Security:** dynamic RPC broadens mechanical reach, so Host authorization and immutable Profiles
  are mandatory admission boundaries. Programs never control transport credentials or addresses.
- **Diagnostics:** recording only declared projections improves privacy and stability but means a
  Producer must anticipate evidence needs. Host logs may aid operations, but they are not Contract
  evidence.

## Deferred extensions

- streaming RPC instructions;
- durable Run storage and crash recovery;
- normalized replay/audit digests;
- cross-process private Slot transport;
- an optional SDK observation protocol for behavior with no authoritative server-visible effect;
- additional workflow, activity, and Nexus SDK instructions justified by concrete Cases.
