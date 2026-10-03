# Umpire 4: what we can learn from distributed-system verification

Research date: 2026-09-30. This is an analysis and recommendation document. It does not amend the
[Umpire specification](UMPIRE4_SPEC.md) or approve its pending changes.

## Recommendation

Umpire should concentrate its next work on the connection between a checked Model and a real
Temporal execution. We already have useful boundaries for authoring, deterministic Case production,
execution, and evidence. The systems examined here show how to make that connection more precise
through executable interface contracts, controllable interruption points, observations of committed
effects, and specifications that expose the concurrency relevant to a particular question.

Six patterns should guide Umpire's next design decisions: small abstract models, explicit mappings,
model-first exploration, a complementary implementation-simulation search, several abstraction
levels, and independent properties with strong oracles. The detailed recommendations below apply
those patterns to Temporal's admission, queue, retention, and ownership boundaries.

Welder adds a condition for scaling this work across components. Each component needs explicit
limits on interference and named progress dependencies. Checking the components separately becomes
useful for composition when their guarantees satisfy one another's assumptions.

These techniques address different failure modes. Adding a stronger model checker cannot reveal a
race omitted by an atomic model step. Running more Cases cannot expose an input relation the
generator never produces. Replaying a recorded Verdict cannot show that a fresh execution followed
the same internal schedule. Umpire needs evidence about each of these boundaries separately.

## Scope and evidence

“SandTable” refers to the EuroSys 2024 tool. “Mocket” refers to the EuroSys 2023 model-checking-guided
tester. “plang” is interpreted as the [P language](https://p-org.github.io/P/). The ZooKeeper comparison
focuses on the EuroSys 2025 multi-grained specification work and its Remix artifact.

The [Tianyin Xu post](https://x.com/tianyin_xu/status/2101806075955794395) points to
[Welder, SOSP 2026](https://cathy-cai.page/pubs/welder26.pdf), which extends Anvil with compositional
controller verification. The comparison below uses the paper and its
[SOSP 2026 artifact revision](https://github.com/anvil-verifier/anvil/tree/acca2bb454a0c8fcc1175ecb3c8b6c396d04aa41).

MongoDB, TigerBeetle, and FoundationDB provide experience from production engineering. SandTable,
Mocket, and Remix provide research implementations and evaluations. P provides a language and several
checking workflows. Their bug counts and published speedups measure different subjects, fault
models, versions, and budgets, so this document does not rank them by those numbers.

Industry descriptions cite primary papers, official documentation, first-party accounts, and source
repositories. Umpire descriptions come from the current working tree, including work in progress.
Existing result reports are identified as reports; their experiments were not rerun for this
document. Recommendations below are our synthesis, rather than capabilities claimed by the cited
systems.

## Six patterns to incorporate

These are the design recommendations adopted for this document. They apply to authoring, Search,
the implementation connection, and evaluation together. They do not introduce a new runtime or
silently amend the specification.

### 1. Keep the abstract model much smaller than the implementation

Model the decisions, identities, state, and interruption points needed for a Property. Omit parsing,
transport mechanics, database layout, and unrelated feature behavior unless they affect that claim.
SandTable shows the benefit of moving exploration into a cheap specification; MongoDB's storage
work shows how a narrow contract makes implementation checks practical.
[SandTable](https://xudongs.com/pub/eurosys24_sandtable.pdf),
[MongoDB storage verification](https://www.mongodb.com/company/blog/engineering/towards-model-based-verification-key-value-storage-engine).

For Umpire, the activity-admission model should capture stale dispatch, eligibility, attempt identity,
and admission commit without reproducing matching or history service internals. Nexus ownership
should capture irreversible effect, retained outcome, and delivery obligation without reproducing
the handler's business logic. The Model can remain small while exposing several transitions inside
one implementation operation.

The success criterion is a compact model that retains the bad interleaving and finds the negative
control. Source line count alone is insufficient. An abstraction that removes the admission race
has made Search cheaper by removing the behavior we need to check.

### 2. Make the Model-to-implementation mapping explicit

Declare how a modeled action reaches an implementation action, how identities and input classes
map, which event establishes a committed effect, and how observed state projects into Model state.
The concrete mechanisms differ across Mocket, MongoDB, and Remix; each supplies a mapping rather
than expecting generic logs to discover one.
[Mocket annotations](https://github.com/tcse-iscas/Mocket),
[MongoDB generated tests](https://github.com/mongodb-labs/vldb25-dist-txns/blob/main/README.md),
[Remix trace scheduling](https://github.com/Lingzhi-Ouyang/Remix/blob/master/checker/server/src/main/java/org/disalg/remix/server/scheduler/ExternalModelStrategy.java).

Umpire's Realization supplies action/instruction and observation/record bindings; Implementation
Links connect product and implementation descriptions. Build on those existing concepts. Each
mapping should expose the required control, evidence, initial state, identity relationships, and
unobservable behavior. Validate structural compatibility before execution. Retain a mismatch at
its source location rather than adding adapter normalization that invents correspondence.

The success criterion is that a developer can inspect why one real admission or acknowledgment
counts as one modeled step. A trace containing the right words is insufficient evidence.

### 3. Explore mostly in the cheap Model and execute selected paths in the expensive implementation

Allocate broad state-space work to Model Search. Execute witnesses chosen for a violation, a
coverage target, a representative input challenge, or a regression. SandTable's division of work
and MongoDB's path-cover generation provide concrete examples.
[SandTable workflow](https://xudongs.com/pub/eurosys24_sandtable.pdf),
[MongoDB path covering](https://www.vldb.org/pvldb/vol18/p5045-schultz.pdf).

For Umpire, retain deterministic selection, typed Limits, and model-owned coverage goals. Add
implementation evidence for selected paths without requiring every reachable model path to become
a live distributed test. A small component seam may permit systematic graph coverage; a full
Temporal deployment needs a more selective budget. Preserve the distinction between model coverage
and runtime-realized coverage.

The success criterion is useful counterexamples or new realized conditions per unit of execution
cost. A large generated suite that repeatedly misses its intended internal ordering is poor coverage.

### 4. Keep deterministic implementation simulation as a second search mechanism

Model Search finds errors in the authored behavior. A simulator that runs actual implementation
code can find coding and ordering failures within its controlled boundaries, including details
omitted from the abstract Model. FoundationDB and TigerBeetle demonstrate this complementary route.
[FoundationDB simulation](https://apple.github.io/foundationdb/testing.html),
[TigerBeetle VOPR](https://github.com/tigerbeetle/tigerbeetle/blob/6f8e6b58d1811bb21cd6ea4fb93cb2e9d77abd81/docs/internals/vopr.md).

For Umpire, prototype this with real admission or retention code behind a controlled dependency
boundary. Let that environment explore its own permitted schedules and faults, then map its
observations back to the Model and Properties. It must have enough freedom to produce behavior
beyond replaying the paths Model Search already selected. A mock Driver returning model answers
would not supply this evidence.

The success criterion is an implementation failure that the abstract search alone would not expose,
with repeatable execution tied to the source revision and controlled schedule or seed. The simulator
still has a fault and scheduling scope. It does not automatically cover arbitrary goroutine races,
real transport code, or every dependency; specialized tests remain necessary.

### 5. Use several abstraction levels

Give product understanding, finite exploration, and implementation conformance views appropriate
detail. ZooKeeper's work provides concrete evidence that detail can be selected per module while
preserving relevant interactions. Its mixed-grained selection is distinct from a universal
coarse-to-fine refinement hierarchy.
[ZooKeeper multi-grained specifications](https://tianyin.github.io/pub/mspec.pdf).

For Umpire, retain the product promise, use a detailed implementation description for admission or
retention, and choose coarse providers for unrelated components. Keep one authoritative behavior
model organized into explicitly connected views and modules. The “single model” vision should mean
shared declarations and checked relationships, rather than one monolithic transition system that
serves every audience equally.

The success criterion is that adding queue or commit detail exposes a meaningful counterexample
without forcing the whole feature model to expand. Each check must retain its module variants,
interface obligations, assumptions, projection, and bounds. An informal connection between two
models cannot justify carrying a Property from one into the other.

Welder supplies a complementary requirement for composition. State what each module may change,
what it requires others to preserve, and whose progress it depends on. Check that the other modules'
guarantees meet those requirements before reusing local results. Selected abstraction levels and
checked interference conditions address different parts of modular verification.
[Welder, sections 3 and 4.4](https://cathy-cai.page/pubs/welder26.pdf).

### 6. Make invariants and Properties independent and explicit

Declare correctness separately from execution generation. Protocol-aware checks and application
workload invariants show why reaching many executions needs a strong oracle.
[TigerBeetle protocol-aware checking](https://tigerbeetle.com/blog/2026-08-20-protocol-aware-dst/),
[FoundationDB workloads and oracles](https://www.foundationdb.org/files/fdb-paper.pdf).

For Umpire, the Scenario selects allowed behavior; the Property judges it. Model monitors remain
passive. Realizations and Drivers must not define correctness through the result they expect to
produce. Keep conformance, safety, bounded progress, and operational disposition separate. Reuse
Properties across supported search and runtime consumers while preserving each consumer's evidence
and trust basis.

The success criterion is an oracle that rejects a deliberately faulty design and an independently
faulty implementation or evidence bundle. Include violations that a convenient generator misses,
such as crossed identities or an acknowledged but unretained outcome. Independent properties do
not require duplicated definitions; they require a meaning that does not derive from the execution
generator's assumptions or copy the implementation's decision procedure.

Keep a component's safety guarantee active even when an environmental assumption fails. Report
whether the premises of conditional progress held and whether the obligation activated. Otherwise,
excluding an interfering execution can make a progress result appear successful without exercising
the promise. Welder's CORE deliberately separates the unconditional guarantee from conditional
reconciliation. [Welder, section 3.3 and figure 5](https://cathy-cai.page/pubs/welder26.pdf).

## Where Umpire stands today

The [vision](UMPIRE4_VISION.md) asks for one behavior model used for regressions, exploration,
white-box functional tests, and black-box canaries. It also asks for faults, learned identities,
programmable SDK workers, and evidence that works across processes with clock skew.

The architecture already distinguishes the Model from the executable artifact:

```text
Model + Property + Scenario + Query
                 |
              Search
                 |
        witness + Realization
                 |
              Producer
                 |
        Case { Program, Contract }
                 |
       Prepare(Case, Profile)
                 |
       PreparedCase.Run(Driver)
                 |
           Run + Verdict
```

The Case is authoritative for its bounded execution and Contract. The Driver supplies authorized
effects. The Contract evaluates declared Observations. Model behavior remains above that execution
boundary. This separation is valuable because it lets Umpire improve model checking and runtime
control independently. See the [module map](UMPIRE_MODULES.md) and
[Testpilot facade](../common/testing/testpilot/README.md).

| Area | Evidence in this tree | Qualification for this comparison |
| --- | --- | --- |
| Finite models and model-to-model refinement | [Scala results](../model0/scala/RESULTS.md), [Go results](../model0/go/RESULTS.md) | Existing reports show Nexus table, Query, refinement, and Case parity. This establishes agreement between representations over their shared domain. |
| Proofs over model behavior | [Scala results, Stainless lemmas](../model0/scala/RESULTS.md#what-stainless-proves) | Reported lemmas concern the model kernel, including bounded attempts and refinement. They do not prove the Temporal server implementation. |
| Scala authoring, serializable IR, Go interpretation | [IR README](../model/README.md), [IR semantics](../model/SEMANTICS.md) | The interpreter builds the lifted Nexus tables. The full authored semantic surface exceeds the implemented interpreter surface. |
| Channels, monitors, assumptions, holes, scoped replacement, progress | [IR semantics, implementation status](../model/SEMANTICS.md#what-the-reader-implements) | These have declarations and written semantics in ongoing work. The documented interpreter still omits several of their meanings; declaration support is not checking support. |
| Executable Cases and live/offline Contract evaluation | [Testpilot](../common/testing/testpilot/README.md), [Evaluator](../common/testing/testpilot/internal/verification/README.md) | The runtime implements bounded execution, immutable records, correlated evaluation, and semantic replay. |
| Runtime faults | [Instruction schema](../proto/internal/temporal/server/api/testpilot/v1/instruction.proto), [worker fault tests](../common/testing/testpilot/temporal/worker/fault_test.go) | The current fault enum supplies worker stop/resume. It does not supply arbitrary network delivery, server crash, or storage-commit scheduling. |
| Replay, reduction, promotion | [Specification](UMPIRE4_SPEC.md#exploration-replay-and-promotion) | The design separates offline Verdict replay, fresh reruns, and diagnostic SDK history replay. Fresh reruns are required for promotion. |
| General model conformance from partial observations | [fn-107 prototype specification](../.flow/specs/fn-107-scala-umpire-prototype-for-standalone.md) | The prototype calls for retaining compatible model executions and separating conformance from Property results. This is a planned extension beyond the existing Contract path. |

One immediate concern precedes all external inspiration. The IR semantics explicitly records that
the reader (`tools/umpire/model`) still reads some declared constructs without applying their semantics, although conforming
readers must implement or refuse them. A successful load therefore cannot stand in for a successful
check of every declared obligation. Close that admission gap before treating a new backend as an
independent verifier. This is already identified by the ongoing prototype, not a new feature request
from this research. See [implementation status](../model/SEMANTICS.md#what-the-reader-implements) and
the [current validator](../tools/umpire/model/validate.go).

### Four meanings of determinism

| Meaning | What is repeatable | What it does not establish |
| --- | --- | --- |
| Deterministic generation | Model inputs, strategy, bounds, and seed produce the same Case bytes | That the target executes those instructions with the same internal interleaving |
| Deterministic evaluation | The same recorded Run and prepared Contract produce the same Verdict | That a fresh target execution reproduces the failure |
| Controlled schedule replay | Selected implementation events occur in a retained order | That every uninstrumented thread, dependency, or event is controlled |
| Deterministic implementation simulation | The controlled implementation and its simulated dependencies repeat an execution | That the simulated dependency contracts match every real deployment |

Umpire implements important parts of the first two. Its Driver controls some worker behavior and
execution dependencies. TigerBeetle and FoundationDB invest throughout their implementations to
achieve the fourth. Mocket and Remix expose selected implementation events for the third.

This distinction matters to the vision's deterministic regressions. We can promise stable Case
artifacts today. A promise of exact internal execution requires an additional, explicitly bounded
control surface. The [scheduler](../common/testing/testpilot/internal/execution/scheduler.go) coordinates
real asynchronous effects; it does not replace Temporal's server scheduler, transport, or storage.

## MongoDB: make a component boundary executable

### Goal and mechanism

MongoDB wants both trustworthy protocol designs and evidence that production components implement
their specified behavior. Its experience gives two complementary routes.

The June 2025 retrospective describes an unsuccessful server trace-checking experiment. Obtaining
consistent multithreaded snapshots was costly, and the chosen model combined election transitions
that the implementation performed separately. Trace post-processing did not resolve that mismatch.
The team stopped that project when the expected cost of extending it outweighed its value. The
author recommends small evolving specifications, early conformance checks, and easily observed
events. The same account distinguishes generating model behaviors for execution from checking
observed implementation behaviors against a model. [MongoDB conformance retrospective](https://www.mongodb.com/company/blog/engineering/conformance-checking-at-mongodb-testing-our-code-matches-our-tla-specs).

The February 2026 WiredTiger work takes a narrower boundary. A compositional TLA+ specification
describes storage semantics relied on by distributed transactions. Modified TLC emits the reachable
graph for finite parameters; path coverings become sequences of storage API calls. The reported
two-key, two-transaction configuration generates and executes 87,143 tests in approximately
40 minutes. The authors explicitly describe a subset of the storage API and possible future
sampling strategies. [WiredTiger model-based verification](https://www.mongodb.com/company/blog/engineering/towards-model-based-verification-key-value-storage-engine).

The engineering advantage is a seam with manageable inputs, observable results, and a contract that
the upper layer actually relies on. A generated test can exercise that seam without reconstructing
the entire distributed system's internal state.

The storage test generator adds useful detail to this picture. The paper describes a greedy path
cover over reachable states, optionally accounting for symmetry, rather than enumeration of every
path. Action labels and arguments become API calls, with assertions for transaction outcomes and
reads. The artifact exposes model parameters, coverage percentage, and splitting controls, and
emits Python tests for WiredTiger's existing suite.
[Distributed transactions paper, section 5](https://www.vldb.org/pvldb/vol18/p5045-schultz.pdf),
[storage test artifact](https://github.com/mongodb-labs/vldb25-dist-txns/blob/main/README.md).

For our graph-based generation, the comparable output should state whether the objective covers
states, edges, obligations, or selected paths. Symmetry that exchanges two model identities must
preserve every equality relationship the runtime Property reads. The useful artifact for a test
owner is the concrete operation sequence, expected outcomes, and provenance; operating the model
checker should not be a prerequisite for running a retained regression.

### Detailed comparison to Umpire

Umpire's Realization similarly maps authored behavior into executable interactions. Our Case and
Profile separation makes the mapping portable between environments. The question to adopt from
WiredTiger is whether a particular interface has a complete enough modeled contract and observable
enough result to support systematic conformance tests.

For standalone activities, that seam could be authoritative attempt admission. Model the current
eligibility, attempt identity, dispatch identity, and admission result. Generate histories that
queue work, pause the activity, and then deliver the stale task. The product promise stays at the
feature level; the implementation boundary supplies the race and its evidence. The
[activity specimen](../model/specimens/activity.md) already identifies this route and a
deliberately faulty admission design.

For Nexus, the corresponding seam could be acceptance and durable retention of an operation result
across caller close/reset. An API acknowledgment alone is insufficient if the promise requires
retained outcome knowledge and a transferable delivery obligation. The
[Nexus specimen](../model/specimens/nexus.md) distinguishes these states already.

We should keep two conclusions separate. A generated Case demonstrates that one modeled behavior
can be realized. An observed Run can demonstrate that one implementation behavior is admitted by the
Model. Neither sampled direction proves universal implementation refinement. Optional behavior in
a specification also need not be exercised by every correct implementation.

### What to learn

Choose the smallest interface that carries a real Temporal promise. Make its Observation mapping
reviewable beside the Model. Treat a mismatch as a question about the implementation, Model, or
mapping, and preserve the point of divergence instead of normalizing it away. Record the marginal
cost of adding the second feature, because a successful one-off adapter can still be too expensive
to maintain.

## SandTable: search cheaply, reproduce precisely

### Goal and mechanism

SandTable targets the cost of exploring real distributed implementations. Its workflow builds a
specification of implementation behavior, samples model traces against code to improve correspondence,
explores the specification, and concretely replays bug witnesses. A shared-library preload layer
controls POSIX-level events such as message delivery, timeout, crash, and client request. Its
node-level models omit thread interleavings and serialization detail. Sampled conformance cannot
eliminate every model/code discrepancy. [SandTable paper, sections 3.1–3.4](https://xudongs.com/pub/eurosys24_sandtable.pdf).

The artifact makes the reproduction path visible. It supplies specifications, a containerized
environment, and named replay targets, including an Xraft election-safety witness. The published
evaluation covers eight consensus implementations and reports 18 newly discovered bugs, with
17 confirmed and 13 fixed. Those results describe the evaluated systems and versions.
[SandTable artifact](https://github.com/tangruize/SandTable).

The cost saving comes from allocating work differently. The model engine can visit and deduplicate
many states without repeatedly starting servers or waiting for disk and network activity. The real
implementation is used to check correspondence and confirm selected witnesses. The reproduction
mechanism must control the events whose order made the witness fail.

The source illustrates the control and observation costs. Its clock interceptor returns controlled
logical time for supported clock kinds and delegates unintercepted calls to the real clock; an
unsupported intercepted kind fails. A separate state collector forwards intercepted log writes
through a pipe to an extractor and retains returned `key=value` state observations.
[Clock interceptor](https://github.com/tangruize/SandTable/blob/main/src/interceptor/myclock_gettime.c),
[state collector](https://github.com/tangruize/SandTable/blob/main/src/interceptor/state_collector.cpp).

Transparent interception therefore still needs an explicit extraction contract. A retained value
from the last log message is trustworthy only for a checkpoint whose meaning makes that value
current. For Umpire, collection, ordering, and effect interpretation should be independently
testable. The implementation should never gain an implicit “state matched” result because the
extractor omitted a field or had not yet observed its commit.

### Detailed comparison to Umpire

Umpire already does inexpensive Model Search before lowering a witness into a Case. This part of
SandTable supports our architecture. The additional requirement is a realistic model of the
implementation's failure-producing transitions and a Driver that can realize those transitions at
the relevant boundary.

A product Model that says “pause prevents later admission” is useful as a promise. It does not by
itself represent queue validation, a delayed dispatch message, a later pause, and authoritative
admission. Search can expose that race only after those transitions exist in the implementation
description. Keep that description separate from the normative product promise. Aligning a
descriptive implementation model with a buggy server must not redefine the promised behavior.

The concrete witness then needs a way to hold the dispatch after validation and release it after
pause. Sending three RPCs in order may leave their internal transitions in a different order.
Testpilot's existing activation reservations and typed worker scripts help at worker boundaries;
server interruption points need their own narrowly scoped controls.

SandTable's POSIX interception is a design reference, not an integration recommendation for Temporal.
Go scheduling, gRPC streams, retries, and storage drivers introduce behavior beyond a node-level
network schedule. A local white-box actuator at a known Temporal seam may be cheaper and clearer
than introducing general process interception.

### What to learn

Make a model witness carry the controls needed to reproduce it. Admission should name a missing
control before target I/O. After execution, preserve expected events, actual events, and the first
unrealizable transition. Extend Umpire's existing replay/promotion path so that a Model violation
and a reproduced implementation violation remain distinct results.

The adoption experiment should compare model states explored per second, concrete reproduction
rate, and adapter effort. More states are useful only when the witnesses correspond to real behavior.

## TigerBeetle: design the implementation for adversarial execution

### Goal and mechanism

TigerBeetle's VOPR executes production consensus and storage code with replacements for clock,
network, and disk. The seed chooses workloads and fault parameters; the seed together with the Git
commit reproduces the simulated execution. Local assertions and simulator-only checkers inspect
invariants. The same infrastructure also supports selected handcrafted scenarios.
[VOPR documentation](https://github.com/tigerbeetle/tigerbeetle/blob/6f8e6b58d1811bb21cd6ea4fb93cb2e9d77abd81/docs/internals/vopr.md).

Recovery testing has a specific environmental contract. After fault-heavy exploration, VOPR chooses
a healthy core, restores its members and connectivity, and freezes failures outside that core.
The core must progress despite permanently unavailable or partitioned neighbors. This catches
failures that repeated healing or restart could accidentally repair.
[Simulation testing for liveness](https://tigerbeetle.com/blog/2023-07-06-simulation-testing-for-liveness/).

Protocol-aware testing adds checks inside each replica. TigerBeetle compares committed log data,
storage structure, and recovery conditions; physical determinism lets caught-up replicas' storage
be compared directly. It also checks whether local or distributed durable information suffices for
recovery. [Protocol-aware DST](https://tigerbeetle.com/blog/2026-08-20-protocol-aware-dst/).

This is a sustained implementation design choice. Simulation works because the real algorithm runs
through dependencies the simulator can control. Replacing the system under test with a simplified
model would provide different evidence.

### Detailed comparison to Umpire

Our Driver boundary makes test effects replaceable. It does not make the implementation's internal
dependencies replaceable. A simulated Driver returning expected results would test Testpilot and
the Contract; it would not expose a Temporal admission bug. To gain VOPR-like evidence, real Temporal
component code must execute behind the controlled queue, clock, or storage boundary.

Start with a component such as activity admission or operation-result retention. Give the test
control over its inputs and completion boundaries while preserving the implementation's decisions.
Keep admission policy in the Model and server implementation, rather than introducing a second
policy in the Driver. A stable interface could expose pending actions and apply one selected action;
the complicated queue, clock, and restart representation stays inside the test environment.

We should also adapt recovery conditions to Temporal. A recovering worker set, usable matching
route, and available persistence may form a healthy core. Declare which worker or route remains
unavailable, which retries are permitted, and which logical operation owes progress. A deadline
without these assumptions conflates a stuck implementation with an unavailable environment.

Physical byte equality is particular to TigerBeetle's design. Temporal's replicated or reset-related
records can differ legitimately. Our analogous check should use a declared semantic projection, such
as retained operation outcome and owner, rather than adopting byte equality indiscriminately.

### Workload and simulator blind spots

TigerBeetle describes a query bug missed by several fuzzers because their generated data constrained
matching records into convenient index arrangements. Less structured inputs and a more detailed
reference oracle exposed it. [Fuzzer blind spots](https://tigerbeetle.com/blog/2025-06-06-fuzzer-blind-spots-meet-jepsen/).

It also adds Vortex, a nondeterministic harness using production binaries, real language clients,
and process/network faults. This covers integration and I/O code outside the deterministic
simulation. [Vortex architecture](https://tigerbeetle.com/blog/2025-02-13-a-descent-into-the-vortex/).

For Umpire, the corresponding lesson is to challenge our Abstraction Claims and independently review
input relationships. Two operations, duplicate deliveries, changing run owners, overlapping
deadlines, and structurally valid but unusual protobuf values can expose behavior absent from a
single representative example. Sharing a Model across consumers helps consistency; sharing every
generator assumption can also share a blind spot. Retain ordinary integration and race tests as
the specification already requires under SCP-04.

## FoundationDB: make rare conditions common and measurable

### Goal and mechanism

FoundationDB's Flow runtime executes real database actors in a deterministic discrete-event
simulation. Network, disk, time, and randomness are controlled. Workloads check application
invariants and internal assertions, while recovery tests restore an environment in which progress
should be possible. The paper describes tuned fault rates, randomized configurations, and conditional
coverage for rare situations. It also identifies limits around external dependencies, real operating
system contracts, and performance. [FoundationDB paper, section 4](https://www.foundationdb.org/files/fdb-paper.pdf).

The official testing documentation describes a single-process simulated cluster and reuse of
workload code in live performance testing. The client-testing documentation explains that some
workloads are environment-specific and that end-to-end API tests complement simulation for
multithreaded client behavior. [Simulation and testing](https://apple.github.io/foundationdb/testing.html),
[client testing](https://apple.github.io/foundationdb/client-testing.html).

The source exposes two useful mechanisms. `Buggify.h` selects active fault sites once and separately
decides whether an active site fires on a visit. Both decisions use deterministic randomness, and
site activation is traced. `DeterministicRandom.h` chooses a generator intended to keep its output
consistent across compiler/library implementations.
[Fault-site selection](https://github.com/apple/foundationdb/blob/d0f1c6795fd1d06b7184fa49e6926582a5e65cb6/flow/include/flow/Buggify.h),
[deterministic randomness](https://github.com/apple/foundationdb/blob/d0f1c6795fd1d06b7184fa49e6926582a5e65cb6/flow/include/flow/DeterministicRandom.h).

These mechanisms vary entire executions as well as individual fault decisions. Different runs
activate different opportunities for delay or unusual behavior, so one perpetually disruptive
condition need not dominate every run.

### Detailed comparison to Umpire

Umpire's Exploration enumerates model-owned targets and credits coverage from decisive Runs along
their witness paths. FoundationDB suggests extending the kinds of conditions we target. “Visited
pause” is weaker than “delivered a pre-pause task after pause committed.” “Visited retry” is weaker
than “retried after an effect committed and its acknowledgment was lost.”

Those conditions should be declared in the Model and supported by Observations. A server test hook
can hold a transition or produce an admitted transient failure, but it should not choose an
unmodeled outcome. The Producer owns scenario selection; the actuator supplies only the requested
effect. This keeps EXP-02 and SEM-16 intact.

Similarly, configuration variation should change behavior only through modeled switches and
interpretations. We can vary queue capacity, retry class, worker topology, operation count, and
fault budget when the Model declares those dimensions. A Profile supplies permissions and physical
resources; it should not silently add scenario semantics.

A useful reporting change would show the rare conditions that were actually observed, targets that
were attempted but never reached, and targets whose evidence remained inconclusive. Run count and
model state count answer different questions. Neither alone tells us whether the important crash
window was exercised.

### What to learn

Adopt contract-preserving perturbations at a few Temporal boundaries and retain their identities in
failure artifacts. Vary the enabled subset between exploratory runs. Measure coverage of the
specific competing events. Keep seed, source revision, runtime versions, configuration, and selected
fault sites together; a seed cannot compensate for an uncontrolled clock or a changed implementation.

A whole-Temporal simulator would require control over services, goroutines, SDK workers, databases,
and transport dependencies. The evidence here supports a narrow component experiment first. It
does not establish that a full retrofit would pay for itself.

## Mocket: use the model graph as a stepwise execution oracle

### Goal and mechanism

Mocket turns model-checking results into tests of distributed implementations. TLC supplies a state
graph; generated paths provide actions and expected states. The runtime schedules corresponding
implementation actions and compares mapped state. Its paper distinguishes inconsistent states,
scheduled actions that cannot execute, and unexpected actions. Disagreement can originate in code,
the specification, or the mapping. The paper also warns about incomplete models, omitted paths, and
model reductions whose commutativity need not hold in implementation code.
[Mocket paper](https://gaoyu-cn.github.io/paper/2023-eurosys-mocket.pdf).

Its Java artifact makes the integration concrete. Developers annotate variables with
`MocketVariable` and actions with `MocketAction`, supply message parameters, export an action-labeled
TLC graph, and run a path generator. A Java agent instruments the system, while a controller receives
guidance, cluster settings, fault choices, and launch scripts.
[Mocket artifact](https://github.com/tcse-iscas/Mocket).

The expected state after each selected action provides a more localized oracle than an end-of-test
invariant. The same generated path supplies both execution guidance and checkpoints.

The controller's source reveals a handshake rather than a generic action invocation. A notification
contains a server, action identity/type, and parameters. Unmatched notifications wait; a matching
notification receives permission to continue. The controller waits for the associated state-check
notification, reconstructs mapped global state, compares it, and then advances the model path.
Separate waits diagnose a missing action and a missing state checkpoint. Remaining notifications
can diagnose unexpected actions.
[Mocket scheduler](https://github.com/tcse-iscas/Mocket/blob/main/Mocket/src/main/java/mocket/runtime/testbed/ActionScheduler.java).

The downloadable demo has narrower support than a general production integration. Its scheduler
includes system-specific mappings, an incomplete ZooKeeper branch, and an initialization handshake
that counts initialized nodes without checking every initial value. We should borrow the scheduling
protocol while evaluating the adapter completeness ourselves. In Umpire, a useful control contract
would cover enabled-action notification, identity/argument matching, release, committed observation,
and closure. Initial-state correspondence deserves the same scrutiny as transition correspondence.

### Detailed comparison to Umpire

A Testpilot Contract can already evaluate intermediate events. That does not automatically establish
full Model transition conformance. A Contract may check a particular promise while ignoring state
that a detailed Model would track. Mocket motivates making that distinction visible to the author.

For example, a final “activity completed” Observation might satisfy a simple completion Rule even
if a stale dispatch briefly admitted an illegal attempt. A stepwise model assessment could show the
first admission result that no expected transition explains. Conversely, a state can be legal while
a history-dependent obligation remains unresolved. Both transition conformance and passive monitor
state are needed for those questions.

In a local white-box environment, a committed state Observation can tightly constrain the expected
state. A black-box canary may expose only RPC results and history records. There can then be several
compatible executions, including invisible stutters. The proposed fn-107 assessment should retain
those possibilities rather than arbitrarily selecting the planned witness. A failed schedule and
an impossible observed transition also need different diagnostics.

For learned identities, bind an operation/run/attempt/delivery relationship when it is observed and
retain that relationship through subsequent checkpoints. Replacing every UUID with one generic
placeholder would erase exactly the correlation errors we need to detect.

### What to learn

Produce a diagnostic containing the source Model step, expected projected state or compatible
states, actual Observations, relevant identities, and first divergence. Reuse the Case and Run
artifacts, with a separately qualified model-assessment result. Do not let an adapter turn private
Slots into undeclared evidence.

Treat path or edge coverage as coverage of the declared Model graph. Before applying partial-order
reduction to concrete execution, check that the Model preserves the implementation dependencies
that make the two actions commute. A queue acknowledgment or commit boundary can invalidate that
assumption.

## P: give obligations explicit state and checking scope

### Goal and mechanism

P models distributed behavior as communicating state machines. Its module system composes machines
and permits interface bindings that substitute one machine for another in a selected composition.
The documentation connects this structure to compositional reasoning.
[P module system](https://p-org.github.io/P/manual/modulesystem/).

P's specification machines are passive observers. They update synchronously on selected events but
cannot send messages, create machines, or affect system behavior. Hot/cold states represent pending
liveness obligations. [P monitors](https://p-org.github.io/P/manual/monitors/).

The current toolchain has distinct assurance routes. PEx provides exhaustive stateful exploration
for supported finite configurations. PVerifier instead checks user-supplied assumptions and
inductive invariants through UCLID5 and an SMT solver. The latter proves properties of the formal
design under its assumptions; it does not automatically prove arbitrary production code.
[PEx](https://p-org.github.io/P/advanced/pex/),
[PVerifier](https://p-org.github.io/P/advanced/PVerifierLanguageExtensions/announcement/).

PObserve supplies an implementation connection. A parser converts service logs into timestamped
events; a sequencer orders them and a demultiplexer routes them to property monitors. Reusing the
monitor checks observed event properties without requiring execution of the whole P design.
[PObserve architecture](https://p-org.github.io/P/advanced/pobserve/pobserve/).

P's event semantics also determines which failures a model can express. Ordinary `send` is reliable,
buffered, nonblocking, and FIFO; loss and reordering need explicit modeling. Machine schedules and
`choose` supply nondeterministic alternatives. The ordinary checker exposes schedule counts and
replay traces, separately from the exhaustive backend.
[P execution semantics](https://p-org.github.io/P/advanced/psemantics/),
[checker workflow](https://p-org.github.io/P/getstarted/usingP/).

This makes an important review question concrete for our channel declarations. A Query using a
reliable FIFO channel has not tested acknowledgment loss, duplicate delivery, or arbitrary network
ordering merely because it explores many schedules. The selected channel semantics and fault
alternatives belong in the result's scope beside the entity counts and step bounds.

### Detailed comparison to Umpire

Umpire now has analogous monitor and assumption declarations in its IR semantics. The useful next
step is completing their execution and authoring workflow. Adding P-shaped syntax alone would not
close the gap recorded by the reader's implementation status.

For Nexus result delivery, a monitor can remember the outstanding obligation independently of the
caller's current state. Close/reset changes ownership; acknowledgment, retention, and successor
commit affect what remains owed. Two paths can reach the same caller state while owing different
results. Model Search must include monitor state in its explored-state identity to avoid merging
those histories. Runtime correlation must preserve the same logical distinctions.

Progress also needs a clear difference between a design-level claim and a finite Run. A fair
non-progress cycle is model evidence under a declared fairness condition. A runtime deadline miss
is evidence about the observed bounded Run. An infrastructure timeout with incomplete observations
establishes neither of those failures by itself. Umpire's SEM-09 and EVD-21 already insist on bounded,
qualified progress; a backend adapter must preserve those limits.

PObserve's timestamp ordering requires particular care for our clock-skew goal. Umpire should use
source record order and causal links, or explicitly declared timing assumptions. Globally sorting
distributed timestamps can invent an order that did not occur. If several orders remain compatible
with the evidence, the assessment should retain them and return inconclusive when they disagree
about the Property. EVD-07 already requires this discipline.

### What to learn

Make passive monitors convenient enough that developers express obligations directly, rather than
encoding all history in a product state enum. Keep safety, bounded progress, conformance, and proof
results separate. Give each backend an explicit support declaration and reject unsupported
constructs. A second backend becomes valuable when it checks the same declared meaning and can
disagree usefully; translating a narrower subset without accounting for the loss is weaker evidence.

Use P's modularity as inspiration for the opaque-queue replacement experiment already proposed in
fn-107. Machine substitution needs an interface obligation and a checked correspondence; the act of
selecting a replacement is not itself a proof that the replacement is safe.

## ZooKeeper / Remix: choose detail per module and per question

### Goal and mechanism

The multi-grained specification work addresses gaps caused by false atomicity, omitted local
concurrency, and missing transitions. Authors provide fine and coarse module descriptions, then
compose a mixed-grained specification that retains target-module detail while preserving relevant
cross-module interactions. Fine and coarse descriptions need not form a refinement hierarchy; the
paper explicitly avoids requiring that relation. Its interaction-preservation argument applies
under stated conditions. Remix replays model traces in instrumented ZooKeeper and compares states.
The paper explicitly acknowledges that model-to-code replay can miss implementation behaviors absent
from the model, and evaluates safety rather than liveness.
[Multi-grained specifications paper, sections 2–3 and 7](https://tianyin.github.io/pub/mspec.pdf).

The artifact separates TLC generation, parsed JSON traces, implementation replay, and per-step
checking reports. Its demo contains ZooKeeper 3.9.1 source and a checker. A match report describes
the replayed traces; it is not a universal implementation conformance certificate.
[Remix artifact](https://github.com/Lingzhi-Ouyang/Remix).

Its scheduling vocabulary reaches inside the application. The external-model strategy matches
enabled events by model action, server endpoints, and transaction zxid, distinguishing network
events from local log/commit processing. AspectJ interception of request-processor queue operations
offers those local events to the controller before execution continues. This supplies a fine
boundary that a socket-only interceptor cannot expose.
[Trace scheduler](https://github.com/Lingzhi-Ouyang/Remix/blob/master/checker/server/src/main/java/org/disalg/remix/server/scheduler/ExternalModelStrategy.java),
[request-processor interception](https://github.com/Lingzhi-Ouyang/Remix/blob/master/checker/zookeeper-wrapper/src/main/java/org/apache/zookeeper/server/SyncRequestProcessorAspect.aj).

The artifact's verifier records property passing separately from trace matching, including an
unknown matching result. A successful check requires both.
[Trace verifier](https://github.com/Lingzhi-Ouyang/Remix/blob/master/checker/server/src/main/java/org/disalg/remix/server/checker/TraceVerifier.java).
For our white-box mode, this favors an application-level admission or commit hook whose semantics
we can name. Its usefulness comes from the boundary it exposes, not from the instrumentation language.

### Why this is different from hierarchy

A hierarchy arranges abstractions at levels. Mixed granularity chooses which modules deserve detail
in one particular check. Consider this Temporal decomposition:

| Module | Coarse view | Fine view |
| --- | --- | --- |
| Activity lifecycle | scheduled, paused, started, terminal | eligibility check, attempt stamp, admission commit, result commit |
| Dispatch queue | interface behavior for queued work | enqueue commit, pending delivery, admission response, acknowledgment, redelivery |
| Worker | permitted attempt result | activation reservation, start, cancellation, execution, response |
| Nexus result retention | outcome retained and owed to an owner | handler effect, report, durable retention, reply, reset transfer, successor commit |

An activity-admission Query might use a fine lifecycle and queue with a coarse worker. A
worker-cancellation Query might reverse that choice. A Nexus ownership Query might detail retention
and reset while leaving handler business logic coarse. Each arrangement must preserve the interface
events the target Property depends on.

The value is avoiding a product of every subsystem's detailed states. The danger is hiding an
interaction that carries the bug. “Coarse” should therefore state which actions, shared values,
identities, and ordering constraints it retains. Merely making a state type smaller does not answer
that question.

### Detailed comparison to Umpire

Umpire's Model Refinement is a checked state mapping with stuttering forward simulation. The common
spec currently locates that relation between product machines and keeps Implementation Links
separate. The prototype also explores scoped provider replacement and visible-result projection.
Those mechanisms are related to selecting detail, but Remix's arbitrary fine/coarse descriptions
must not silently become instances of our existing Refinement relation.

For a concrete first experiment, our opaque durable-queue provider is a good starting point. Replace
it with a detailed provider in one composition and keep the same activity promise. Check the
provider's interface behavior and declare the assumptions each version uses. Include a provider
that loses committed work as a negative control. This narrower, checked replacement can demonstrate
value before proposing a general mixed-granularity selection language.

Visible stutters deserve particular attention. A detailed step that leaves the abstract state
unchanged can still emit an acknowledgment or externally visible result. Our new IR semantics
restricts this when a refinement names its visible facts; legacy refinement leaves some such
stutters admitted. The [specimen findings](../model/specimens/README.md#findings-for-later-tasks)
record that difference. Classifying a step as a stutter must account for the promise's visible output,
not just state equality.

### What to learn

Split atomic steps where a real interruption changes the result. For a retained Nexus completion,
the important sequence may include effect, report receipt, durable retention, acknowledgment,
ownership transfer, and successor commit. Collapsing it into “completion delivered” prevents Search
from considering a crash after acknowledgment but before retention.

Select detail by the Property and the failure boundary. Keep interfaces and assumptions explicit.
Report which module variants and bounds a Query actually checked. Continue using separate
implementation trace assessment to challenge behavior missing from every model variant.

## Welder / CORE: make local progress safe to compose

### Goal and mechanism

Welder verifies interacting Kubernetes controllers incrementally. Its CORE specification combines
Eventually Stable Reconciliation (ESR) with request-based rely-guarantee conditions and named
liveness dependencies. ESR requires eventual convergence and continued agreement once desired
state stabilizes. A compact reading of CORE is:

```text
Under the declared environment Model:
    Guarantee holds on every execution.
    If Rely and Depend hold, Success holds.
```

Composition requires each side's Guarantee to imply the other's relevant Rely. A provider's Success
must also discharge the consumer's Depend. Adding a controller can then reuse established results
while checking the new compatibility obligations. Merely placing locally correct controllers
together does not establish their collective progress.
[Welder, sections 3 and 4.4](https://cathy-cai.page/pubs/welder26.pdf).

The request conditions constrain owned objects, fields, names, and ownership changes. Welder derives
a state-preservation invariant from these conditions before reasoning about progress. Requiring
interference to cease eventually can be too weak. A single conflicting creation can leave an object
that permanently blocks a controller. The rely condition restricts requests throughout execution.
[Welder, sections 3.2 and 4.3](https://cathy-cai.page/pubs/welder26.pdf).

Welder also addresses two progress subtleties. Iterative dependencies use a nonincreasing measure
with eventual decreases to justify progress while an upper controller changes a lower controller's
desired state. Its Compose-Dep rule handles acyclic dependencies; mutual waiting needs additional
reasoning. For
shared-object updates, fair request processing still permits endless version conflicts. Welder
implements an atomic conditional-update abstraction with retry-on-conflict and explicitly assumes
that the retry loop terminates, possibly with an error.
[Welder, sections 4.1, 4.2, 4.4, and 8](https://cathy-cai.page/pubs/welder26.pdf).

The authors implemented and verified three core controllers and one custom controller using
Verus. These are verified implementations built from upstream references. The scope includes
trusted environment and client models, libraries, and tooling. Liveness assumes weak fairness and
eventual cessation of specified faults in an asynchronous environment with unbounded delays. CORE's
feature coverage depends on the authored `match` predicate; it supplies neither complete behavioral
conformance nor a finite latency guarantee.
[Welder, sections 5, 6, and 9](https://cathy-cai.page/pubs/welder26.pdf).

### Detailed comparison to Umpire

Umpire's Composition builds a reachable joint Model. Its Capability laws, assumptions, Properties,
and proposed provider replacement offer places to express component obligations. Those mechanisms
do not yet establish Welder's composition theorem. Any general rely-guarantee reuse rule needs a
separate specification decision and implemented checking semantics.

Use Nexus retention and reset as a bounded experiment. The retention provider guarantees that an
acknowledged outcome remains durably recoverable while its delivery obligation is outstanding.
The ownership provider guarantees that a reset preserves the logical operation and its delivery
obligation while changing the authorized owner.
Each relies on the other's steps preserving its relevant state and identity. Delivery progress
additionally depends on an available owner, retained outcome access, and successful retry conditions.
These are proposed Temporal obligations, rather than claims made by the Kubernetes paper.
Keep no-premature-acknowledgment as a separate safety Property. Eventual convergence alone permits
intermediate mistakes that violate that promise.

Check the local guarantees and their cross-component implications, then check the composed
Property under explicit Limits. Keep unreplaced providers as named assumptions. Retain the exact
module variants, Behavior Fingerprints, remaining assumptions, and explored bounds with the result.
An assumption can be discharged only by evidence covering the obligation the consumer actually uses.

A runtime assessment should distinguish a component guarantee violation, an observed rely violation,
and missing evidence about a premise. Keep unconditional safety checks active and qualify the
conditional progress result. Do not discard interfering records to manufacture a passing Run.
Sampling and bounded checks can challenge these obligations; they cannot establish an unlimited
composition theorem.

### What to learn

Start with one dependency edge and explicit preservation obligations. Expose transfer of ownership,
retained knowledge, and acknowledgment at their real commit boundaries. For iterative handoffs,
identify what decreases and which provider result permits the next step. For persistence retries,
distinguish being scheduled from making a successful commit or returning an allowed error.

Umpire's current weak-fair action declarations do not by themselves express Welder's retry-loop
termination assumption. Keep conflict and retry detail visible until a justified abstraction and its
progress conditions are supported. Preserve SEM-09's bounded meaning for finite checks and Runs.

## Comparison across the complete workflow

This table compares mechanisms described above with Umpire's current or proposed counterpart.
An external mechanism is an existence example, not a requirement to import that tool.

| Question | What the other systems make concrete | Umpire counterpart | Lesson for us |
| --- | --- | --- | --- |
| What behavior do we promise? | MongoDB interface contracts; P event specifications | Property and product Model | Keep a promise independently readable from the implementation machinery. |
| Where can a race occur? | SandTable implementation descriptions; Remix fine transitions | Detailed machines, channels, Implementation Links | Include interruption boundaries before expecting Search to find races. |
| What is explored? | Finite state graphs, selected schedules, or real simulated executions | Query Limits and exploratory targets | State the explored object and bounds in every result. |
| How does a witness reach code? | Storage API sequences, interception, Java agents, controlled dependencies | Realization, Program, Driver | Require the control that realizes the causally relevant order. |
| What confirms the expected transition? | Mocket/Remix state comparisons; protocol-aware checks | Observation mapping and proposed conformance assessment | Observe committed effects and retain the first divergence. |
| What remembers an obligation? | P specification machines | Model monitors and correlated Contract Rules | Keep monitor state in search identity and per-operation runtime state. |
| When should recovery succeed? | Healthy-core recovery testing | Assumptions, progress claims, Deadlines | Declare the usable environment and permanent failures. |
| How do rare paths become reachable? | FoundationDB perturbations and configuration variation | Model-owned variations, faults, Abstraction Claims | Generate relational corner cases and measure realized conditions. |
| How does detail remain tractable? | Narrow storage seams; mixed-grained module composition | Small machines, compositions, provider replacement | Detail the target interaction without expanding every subsystem. |
| When do local results compose? | Welder's compatible rely-guarantee conditions and discharged liveness dependencies | Capability laws, assumptions, Composition, scoped replacement | Check preservation and dependency obligations before reusing a local result. |
| Did a conditional promise apply? | CORE's unconditional guarantee and conditional ESR | Property activation, assumption evidence, qualified progress results | Expose a failed or unobserved premise while continuing to check safety. |
| What repeats a failure? | Versioned seeds or retained controlled schedules | Case identity, Profile identity, Run, semantic replay, fresh reruns | Retain the implementation version and scheduling basis as well as the artifact. |
| What covers omitted dependencies? | Live API/integration testing and Vortex | Functional tests, SDK workers, black-box canary, specialized tests | Keep tests that challenge the modeled dependency contracts. |
| What has actually been established? | Distinct model, test, monitoring, and proof workflows | Independent stage statuses, Known Gaps, Claim Assessment | A satisfied Contract and a conforming execution are different claims. |

### The two directions of correspondence

Let `M` be the allowed observable behaviors of the selected Model and `I` the observable behaviors
of the implementation under the selected environment. A correctness-oriented refinement claim asks
whether `I` is contained in `M`, under a declared observation mapping.

A generated witness starts in `M` and asks the implementation to realize it. This is valuable for
exercise, regression, and exposing an overly permissive or inaccurate implementation description.
It does not sample extra behaviors that occur only in `I`.

An implementation-driven Run starts in `I` and asks whether the Model explains its evidence. This
can find an unmodeled transition even when all generated witnesses pass. With partial observations,
the check ranges over compatible model executions rather than a fabricated full state snapshot.

We should support both directions through the existing execution artifacts. The assessment result
should record transition conformance separately from each Property result. An execution may conform
to a descriptive implementation Model that violates a product promise. An execution may also satisfy
one Property while containing an unexplained transition elsewhere. Neither result should overwrite
the other.

### Coverage needs several denominators

| Coverage measure | Useful question | Common mistaken conclusion |
| --- | --- | --- |
| Model state/edge coverage | Which declared behavior did Search reach? | Every implementation schedule was exercised. |
| Target realization | Which intended witness paths did decisive Runs confirm? | Every attempted Case reached its intended internal state. |
| Observation coverage | Which committed effects and correlations were visible? | Missing records establish that an effect never occurred. |
| Input-class exploration | Which members and relationships challenged the Abstraction Claim? | One convenient example represents every value in the class. |
| Control coverage | Which interruption points and fault combinations occurred? | Authorizing a fault means it happened at the required point. |
| Property activation | Which obligations were triggered and evaluated? | A never-triggered obligation provides a passing regression. |

Use the existing Inventory and exploration ledger where possible. This recommendation is about
making their denominators explicit, rather than introducing another independent reporting system.

## Recommended priorities

### First: finish the meaning already declared

Complete or refuse every supported IR construct. Include monitor state in search identity,
preserve visible-result constraints, and qualify holes, assumptions, and progress outcomes. Keep
compiler acceptance, IR admission, model checking, and Case execution as separate statuses.

This work improves the trustworthiness of every later experiment. It also keeps a secondary backend
from appearing to confirm behavior the primary interpreter never evaluated.

### Second: establish one useful implementation seam

Use activity admission as the initial controlled race. The Model supplies the correct promise and
the faulty design control. A white-box Driver holds dispatch at the named boundary, commits pause,
and releases the stale delivery. Observations identify authoritative admission and its attempt.

Measure whether an author can modify the admission rule, obtain a source-located counterexample,
reproduce it through Testpilot, and distinguish the corrected implementation without changing
framework policy code. That is more informative than adding another large feature model first.

### Third: assess observations against model behavior

Implement the compatible-execution assessment described by fn-107. Keep it passive, bounded, and
separate from the built-in Contract Verdict. Reuse the same interpretation for live assessment and
offline replay. Preserve logical operation, attempt, delivery, and run identity through projection.

Use synthetic evidence to exercise ambiguity, missing commit evidence, duplicate delivery, and
crossed identities before connecting another live consumer. A small evidence module should hide
the compatible-state representation behind admission, event consumption, closure, and receipt APIs.

### Fourth: demonstrate scoped detail and recovery

Replace the opaque queue with a detailed queue/persistence provider for the same activity Query.
Then use the Nexus ownership example to test retention and recovery under explicit reporting and
availability assumptions. Keep the negative controls in the experiment so we can show which
meaningful mistakes each mechanism detects.

Apply Welder's interference discipline to the same providers. Check preservation of retained outcome
and delivery obligation across reset, and record which provider result discharges each dependency.
Include an interfering owner transfer and a contention path that repeatedly schedules a retry
without committing. The report should distinguish failed guarantees, unmet or unobserved assumptions,
and bounded progress failures. This extends the scoped experiments rather than requiring a
control-plane proof framework first.

This can inform a later proposal for mixed granularity. Any change to Umpire's existing Refinement
or Implementation Link meaning needs its own specification decision under GOV-02; this document
does not enact one.

Alongside model-first exploration, evaluate one implementation-simulation environment as a distinct
search mechanism. It should vary controlled real-code schedules independently of selected model
witnesses, reuse the explicit observation mapping, and assess the same Properties. This makes the
fourth pattern an actual experiment rather than treating deterministic replay as its substitute.

### Fifth: improve exploration diversity with measured conditions

Add model-owned variations that create stale messages, acknowledgment loss, concurrent operations,
and owner changes. Vary fault subsets and configuration dimensions within declared bounds. Credit
the observed condition, rather than merely recording that a Case was submitted.

Consider P, Quint, or another backend after the same bounded slice has complete semantics and
negative controls. Consider broader deterministic implementation simulation only after a narrow
component shows that its controlled dependency boundary catches failures other tests miss.

## Concrete experiments and decision criteria

These are proposed evaluations, not task tracking or claims that the experiments have been run.

| Experiment | Mechanism to evaluate | Positive and negative controls | Evidence that would justify adoption |
| --- | --- | --- | --- |
| Activity admission | MongoDB-style seam plus SandTable/Mocket-style witness realization | Current-eligibility admission; admission that trusts a stale dispatch | Model finds the bad design, a controlled Run realizes the ordering, committed admission identifies the same violation, corrected behavior passes. |
| Queue replacement | Scoped provider detail inspired by P and Remix | Queue that preserves committed work; provider that loses committed work | Same product Property survives valid replacement; bad replacement fails its interface obligation; receipt shows changed assumptions and exact variant. |
| Retention/reset composition | Welder-inspired preservation and dependency checks over a finite Model | Compatible retention and ownership providers; transfer that drops an obligation; acknowledgment followed by lost retention | Local guarantees and cross-component implications are checked within declared bounds; composed Property catches interference; replaced assumptions are discharged and remaining assumptions stay visible. |
| Retry contention | Detailed persistence retries with explicit progress premises | Retry that commits or returns an allowed error; repeated conflicts despite fair request processing | Checker distinguishes scheduling fairness from retry termination, detects bounded starvation, and keeps safety active when progress premises fail. |
| Nexus close/reset | Stateful obligation monitoring and explicit commit boundaries | Retained outcome with transferable obligation; premature acknowledgment; original-run acknowledgment after reset | Checker distinguishes handler effect from durable knowledge, retains the obligation across ownership change, and reports the first incorrect acknowledgment or lost obligation. |
| Recovery phase | TigerBeetle-style healthy core | Usable workers/route/storage; permanently unavailable irrelevant worker; implementation that stalls despite usable resources | Progress conclusions name assumptions and bounds; permanent outside failures do not get healed accidentally; unavailable required resources remain distinct from a semantic failure. |
| Input abstraction challenge | Independent, less structured generation | Same/distinct operation IDs, attempt stamps, duplicate deliveries, varied ordering | Divergence splits an Abstraction Claim; crossed evidence cannot satisfy another operation; no hidden equality assumption survives normalization. |
| White-box and canary evidence | Shared Model with different controls and observations | Full commit evidence; only public history/RPC evidence; skewed source timestamps | Unsupported controls reject before I/O; incomplete evidence returns inconclusive; unrelated timestamp changes do not change a causally identical result. |
| Backend agreement | One finite slice evaluated by two independent algorithms | Correct model; output-producing stutter; monitor-history merge; reachable hole | Results agree on complete supported inputs and reject or qualify unsupported ones; negative controls demonstrate that agreement is substantive. |
| Independent implementation simulation | Actual admission or retention code with controlled dependencies | Correct code; stale-state admission; acknowledgment before retention commit | Simulator exploration finds and repeats a code failure independently of model-selected paths; mapped evidence identifies the violated Property and the controlled scheduling scope. |

Record author edit-to-answer time, explored states, concrete reproduction rate, missing-control
diagnostics, evidence volume, and effort to add the second feature. Record the Model scope and target
revision with those measurements. Evaluate adoption on useful failure detection and maintainable
seams, rather than published bug totals from unrelated systems.

## Design commitments worth keeping

The shared Model, versioned Case, thin Driver, immutable Profile, append-only Run, deterministic
Evaluator, correlated evidence, and explicit Known Gaps are useful foundations. The industry lessons
mostly deepen their connections.

Keep semantic policy in authored Model Definitions. Let controls expose real choices without
inventing product behavior. Keep normative promises separate from descriptive implementation
behavior. Give every observation a precise effect and identity. Preserve uncertainty when evidence
admits several meanings. Make regression promotion retain the exact Model and runtime basis that
reproduced the failure.

The next convincing demonstration is a small Temporal promise whose faulty design produces a model
witness, whose real execution exposes the same failure at a controlled boundary, and whose corrected
design and implementation both satisfy the unchanged promise. That would combine the most useful
parts of these approaches within Umpire's existing architecture.
