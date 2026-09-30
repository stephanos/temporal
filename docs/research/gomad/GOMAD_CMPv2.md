# What Gomad can learn from Loom and deterministic simulation testing

> Dated research snapshot. Local capability and support statements describe the
> assessment below. Use the [current README](../../../tools/gomad3/README.md) and
> [active milestones](../../../.plans/GOMAD_MILESTONES.md) for present behavior and delivery order.

Assessment date: 2026-09-27. Local basis: working tree at `5285983bb9`, including
in-progress changes. This is a source-based assessment, not a new qualification
run or a replacement work queue. External facts link to primary sources;
proposed changes and experiments are recommendations.

Loom offers lessons about controlling and reducing concurrency choices;
TigerBeetle offers stronger lessons about which failures to generate and how to
recognize incorrect behavior. Gomad already implements much of the execution
machinery. Its next gains should come from better semantic oracles, recovery
tests, workload distributions, and measured search policies.

Follow [GOMAD_MILESTONES.md](../../../.plans/GOMAD_MILESTONES.md) for current delivery order. Investigating
these ideas must not turn an unsupported or divergent functional test into a
passing expectation. The Flow-Next specs remain authoritative.

## The baseline has moved

The August comparison predates exact choice replay and the schedule frontier.
Its snapshot remains in Git history; the research below assumes those mechanisms.

| Area | Present in this checkout | Remaining opportunity |
| --- | --- | --- |
| Runtime control | Native Go, one P, virtual time, seeded scheduling, validated choice-tape replay | Scheduling points that current execution cannot expose; explicit limits on coverage |
| Search | Bounded breadth-first alternative-prefix exploration; combined runtime/scenario/network/storage/fault/crash exploration | Evidence that a different policy finds more bugs per compute-hour |
| Distributed faults | Partitions, directional disconnects, delay, crash/restart, partial persistence, process backend | Fault distributions and recovery obligations tied to a real workload |
| Oracles | State predicates, exact histories, duplicate/lost checks, snapshot convergence | Independent reference models, legal concurrent histories, bounded progress monitors |
| Guidance | Semantic/choice features and replay-verified seed corpus | Generating new semantic inputs and fault configurations, beyond reusing seeds |
| Reduction | Forced schedule suffix/range and fault-entry reduction | Typed scenario shrinking and durable minimizer resume |

Sources: [runtime and Runner contract](../../../tools/gomad3/README.md),
[choice frontier](../../../tools/gomad3/runner/internal/exploration/choice/engine.go),
[combined frontier](../../../tools/gomad3/runner/internal/exploration/simulation/frontier.go),
[fault vocabulary](../../../tools/gomad3sim/fault.go),
[oracles](../../../tools/gomad3sim/oracle.go), and
[minimizer](../../../tools/gomad3/runner/internal/minimizer/minimizer.go).
Implemented mechanisms do not establish that every target or platform qualifies;
the milestone reports retain the observed limitations.

Three distinctions matter when comparing competitors:

| System | Controlled boundary | Relevant lesson |
| --- | --- | --- |
| Loom | Small Rust tests using replacement concurrency primitives | Semantic scheduling points, dependency reduction, bounded search |
| TigerBeetle VOPR | Production database logic behind simulated network, storage, and time | Protocol-aware safety and recovery checks |
| FoundationDB | Cluster simulation integrated with its Flow execution model | Diverse workloads and faults, supplemented by real-system testing |
| Gomad | Supported Go execution behind a patched runtime and reviewed I/O boundary; explicit simulation backends | Preserve ordinary Go execution while improving what campaigns exercise and check |

These are different abstraction boundaries, not interchangeable completeness
claims. See [Loom's guide](https://docs.rs/loom/0.7.2/loom/),
[TigerBeetle's protocol-aware DST](https://tigerbeetle.com/blog/2026-08-20-protocol-aware-dst/),
and [FoundationDB testing](https://apple.github.io/foundationdb/testing.html).

## Loom: improve the choices before optimizing their search

Loom requires relevant synchronization to use its replacement types. Operations
hidden in uninstrumented dependencies are invisible. It explores modeled
executions with reduction, but large models still grow rapidly; its documented
relaxed-memory limitations also prevent treating it as a complete Rust memory
model verifier. [Loom limitations](https://docs.rs/loom/0.7.2/loom/#limitations-and-caveats)

### Expose missing interleavings deliberately

Gomad disables asynchronous preemption. Replaying every available runnable or
`select` choice therefore does not explore interleavings at every atomic or
uncontended synchronization operation. Its
[preemption fixture](../../../tools/gomad3/internal/gomadtool/conformance/testdata/preemption/main.go)
explicitly exercises a spin loop that cannot finish under this contract.

The useful Loom lesson is semantic visibility. Before adding a new scheduler,
build a small catalog of bug witnesses: check-then-act through separate atomic
loads/stores, competing cancellation/completion, lock handoff, and timer ties.
For each, record whether the failing execution is reachable at existing
scheduling points. A controller cannot select a transition the runtime never
offers.

If a representative bug requires a missing point, evaluate narrowly scoped
compiler/runtime checkpoints as a separate, identity-bound profile. Measure
overhead and qualification stability. Do not rewrite existing functional tests
to sprinkle yields into them; the milestone contract forbids that shortcut.

Go's atomics are sequentially consistent and data-race-free Go has a sequential
consistency guarantee. Porting Rust relaxed-memory machinery is therefore not
the first requirement. This still does not make single-P execution complete:
the offered scheduling points remain a restriction. Run the stock Go race
detector separately; Gomad currently excludes it.
[Go memory model](https://go.dev/ref/mem),
[Gomad execution contract](../../../tools/gomad3/README.md#contract)

### Try preemption bounds before general dependency reduction

Loom exposes separate limits for preemptions, branches, permutations, and time,
plus checkpoint controls. A preemption bound differs from Gomad's prefix-depth
bound: it limits switching away from an actor that could continue, allowing
long executions with few disruptive switches.
[Loom Builder](https://docs.rs/loom/0.7.2/loom/model/struct.Builder.html)

For Gomad, first establish reliable continuation/block/exit classification at
controlled decisions. Then compare bounds 0, 1, and 2 against seed sampling and
the existing frontier on small workloads. Blocking switches should not spend
the preemption budget. Report completion only for the instrumented transition
set and the other recorded limits. This is an experiment, not evidence that two
preemptions suffice for Temporal.

Dynamic partial-order reduction (DPOR) eliminates redundant orderings using
dependencies and happens-before relationships. Loom's execution implementation
uses operation dependencies and vector clocks to add backtracking points.
[Loom execution source](https://docs.rs/loom/0.7.2/src/loom/rt/execution.rs.html)

Gomad's rank prefixes and semantic hashes do not supply that dependency model.
A credible experiment needs stable actor/resource identities and operation
effects; unknown interactions must remain dependent. Start with a small explicit
mailbox or storage model and compare reduced exploration against the unreduced
frontier on fixtures with known outcomes. Pruning native Go runs solely because
their World hashes match would be unsound: the hash does not capture every
goroutine's stack, application heap, and future behavior.

## TigerBeetle: make successful runs prove something useful

### Recovery must preserve the failure conditions being tested

TigerBeetle's liveness mode first creates difficult state under faults, then
selects a healthy quorum, heals its internal connections, and makes failures
outside that core permanent. This caught failures that ordinary chaos concealed
by eventually rebooting or reconnecting the troublesome replica.
[Simulation Testing for Liveness](https://tigerbeetle.com/blog/2023-07-06-simulation-testing-for-liveness/)

This is directly applicable to Gomad's existing scenario and fault machinery.
Introduce a reusable recovery phase that records the healthy subset, remaining
faults, required progress, and logical-time budget. Test both full recovery and
recovery with an asymmetric partition or permanently unavailable participant.

Temporal needs its own availability assumptions. A one-box frontend/history/
matching/SQLite test is not a replicated VSR cluster, so a majority of its
services is not a meaningful quorum. Each scenario must declare the dependencies
that must be available: persistence, shard ownership, matching, and a polling
worker where appropriate. With those preconditions satisfied, require concrete
progress such as an acknowledged update completing or a timer advancing history.

The existing `EventualConvergence` function compares the byte slices supplied at
one instant; it does not wait, impose a deadline, or check progress. A progress
monitor should own those obligations, including a nonempty set of work to
complete. Distinguish unmet recovery preconditions, a bounded liveness failure,
and a wall-watchdog termination. A finite deadline is a tested recovery bound,
not a proof of eventual progress under all fair schedules.
[Current oracle implementation](../../../tools/gomad3sim/oracle.go)

### Check both public behavior and internal protocol invariants

TigerBeetle's protocol-aware simulator checks committed-log agreement and
storage consistency, and asks whether individual replicas recover when their
durable state permits it. Overall availability can hide a stuck participant.
Its byte-identical storage checks rely on TigerBeetle's physical determinism.
[Protocol-Aware DST](https://tigerbeetle.com/blog/2026-08-20-protocol-aware-dst/)

For Gomad, pair public API histories with narrow, read-only observations at
meaningful transitions. Candidate Temporal properties include legal workflow
state transitions, preservation of acknowledged durable history, correct update
admission order, and recovery of eligible work after ownership changes. Compare
logical state unless the component explicitly promises byte-identical storage;
do not impose TigerBeetle's disk-layout contract on SQLite or Temporal.

Keep the oracle separate from the production implementation. A small sequential
reference model can validate permitted outcomes without duplicating the same
algorithm and its bugs. `ExactHistory` is useful for a prescribed execution,
but equality with one history is not a linearizability checker. Concurrent
operations may admit several legal orders, and a timeout can leave an operation's
effect unknown. Model those cases explicitly and bound any history search.

The existing
[matching duplicate-delivery scenario](../../../tools/gomad3sim/temporal_scenario_toolchain_test.go)
is a TCP harness using `collection.SyncMap`, not a running Temporal matching
service. It proves a failure can be recorded and replayed. It does not establish
a production matching bug or an exactly-once transport guarantee. Place any
no-duplicate property at the semantic boundary that promises it, such as an
idempotent committed effect, rather than rejecting every retry delivery.

### Vary workloads independently of the oracle

TigerBeetle documented a query bug missed by its fuzzers because generated data
and queries had overly correlated structure. Less constrained inputs and an
independent, more detailed model made the bug reproducible.
[Fuzzer Blind Spots](https://tigerbeetle.com/blog/2025-06-06-fuzzer-blind-spots-meet-jepsen/)

Gomad should treat scenario generation as a search dimension distinct from
scheduling. Its guided corpus currently reuses realized seeds and transcripts;
it does not mutate scenario inputs or fault plans. More schedule seeds cannot
find a bug whose necessary input never appears.
[Guidance contract](../../../tools/gomad3/README.md)

Use versioned campaign configurations to vary operation mix, concurrency, key
skew, retries, payload sizes, cancellation timing, resource capacities, and fault
intensity. Include runs that disable selected operations or fault classes so
other behaviors can develop. Preserve some low-fault runs; constant failure can
prevent the system from reaching the interesting state at all.

Give generated input, runtime scheduling, and fault selection separate recorded
streams/configurations. Retain the realized input and fault plan, not just a
master seed. These additions should extend the current identities rather than
silently changing `GOMADSEED`, which currently controls scheduling independently
of deterministic I/O entropy.

TigerBeetle's swarm-testing example makes the mechanism concrete: select an
operation subset and weights once per run, then draw operations from that
distribution. Equal push/pop probabilities tend to keep queues small; changing
the distribution reaches different queue shapes.
[Swarm Testing Data Structures](https://tigerbeetle.com/blog/2025-04-23-swarm-testing-data-structures/)
For Temporal, compare mostly-enqueue, mostly-cancel, hot-key, and burst-retry
families with the current fixed workload. Retain both the sampled weights and
the realized operation sequence so replay does not depend on generator changes.

## Other projects with useful, specific lessons

### FoundationDB: adversarial scenarios and named reachability

FoundationDB documents cluster simulation, workloads with semantic invariants,
and staged connection disruption/recovery, alongside live performance and
hardware failure testing. It also reuses workload code across simulation and
performance tests. [FoundationDB testing](https://apple.github.io/foundationdb/testing.html)

For Gomad, compose staggered directional disconnects and reconnects through the
existing network model before adding more networking APIs. Require a campaign
to demonstrate the intended situation: a retry after a lost acknowledgement,
recovery from a selected partial-persistence state, or a cancellation overlapping
completion. Record planned and realized conditions separately. A fault action
present in configuration is not evidence that the intended failure path ran.

FoundationDB's client BUGGIFY separates whether an injection site is active
for a run from whether it fires on an encounter.
[Client Testing](https://apple.github.io/foundationdb/client-testing.html)
That is a useful distribution pattern for existing Gomad fault actions. New
application injection points should require evidence that a named error path
cannot be exercised efficiently through the existing model, with their semantics
and profile identity explicit. It is not a reason to alter unchanged functional
test bodies or inject errors that the real operation cannot produce.

### Antithesis: separate invariant checks from coverage obligations

Antithesis distinguishes always, sometimes, reachable, and unreachable
properties. It catalogs assertions so that never encountering a required
property can itself be reported, and uses assertions as exploration guidance.
[Assertion semantics](https://antithesis.com/docs/product/writing_tests/assertions/)

Gomad already has required probes and semantic coverage. Extend that mechanism
with declared workload properties rather than building another coverage store.
For example, separately check that every observed committed effect is legal,
that a retry under partition was reached, and that a recovered workload completed
at least one operation. Keep stable property IDs, witness counts, and the scope
of each requirement: one execution, one scenario family, or the entire campaign.
A reachability witness is useful coverage evidence, not universal liveness.

### Shuttle: a small portfolio of search strategies

Shuttle offers randomized scheduling, PCT, DFS, replay, and portfolio execution,
explicitly trading exhaustive exploration for larger tests.
[Shuttle guide](https://docs.rs/shuttle/0.9.4/shuttle/)

Gomad should retain seed sampling and its raw frontier as baselines. Benchmark
PCT or preemption bounding as additional policies behind the choice interface,
then allocate compute according to observed results. Do not infer PCT's
probabilistic guarantees from merely using randomized priorities: actor counts,
step definitions, priority-change sampling, and bug-depth assumptions need a
separate validated contract.

The existing roadmap reports that frontier and seed sampling both found the two
declared outcomes of a small fixture in sixteen executions. That is a neutral
result, not justification for general DPOR. Compare multiple representative
bugs and record both executions and CPU/wall cost, including preparation,
replay, reduction, and artifact storage.
[Search evidence](../../../.plans/GOMAD3_NEXT_BUG_FINDING.md#search-evidence)

### Gosim: test the simulator against the system it replaces

Gosim runs filesystem behavior tests in simulated and nonsimulated builds. It
also supports stopping in Delve at a recorded simulation step.
[Gosim development and debugging](https://github.com/jellevandenhooff/gosim)

Gomad already has conformance, boundary canaries, and cross-backend tests. Extend
shared behavior tests for the exact modeled contracts, especially TCP close/
deadline races and file/directory sync. Two simulation backends agreeing is not
independent validation if they share the same incorrect model. Compare permitted
outcomes against stock Go on each supported platform; do not require identical
host schedules or pretend a normal filesystem run validates all power-loss
states.

TigerBeetle makes the same distinction with Vortex, an intentionally
nondeterministic suite exercising compiled binaries, native client bindings,
and real integration boundaries that DST replaces.
[A Descent Into the Vortex](https://tigerbeetle.com/blog/2025-02-13-a-descent-into-the-vortex/)
Preserve a stock-toolchain, real-I/O testing lane for Gomad consumers and share
workload intent and logical checkers where possible. Simulation speed is not
a production throughput benchmark, and the native lane need not reproduce an
identical trace to expose a model or integration defect.

A later replay-to-decision debugger could make existing choice ordinals and
node/incarnation records easier to use. Keep debugger builds and instrumentation
explicit: rebuilding a binary with debug flags changes its identity and cannot
silently inherit an old artifact's exact-replay claim.

## Recommended experiments, in delivery order

These are candidates for later Flow-Next specs, not newly scheduled tasks.
The effort estimates are relative judgments, not measured implementation costs.

| Order | Experiment | Evidence needed to continue | Cost / main risk |
| --- | --- | --- | --- |
| Existing gate | Finish F4–F7 support and repeatability work | The milestone reports meet their existing criteria on the named platforms | Current priority; more search amplifies any uncontrolled nondeterminism |
| First extension | Independent oracle plus recovery phase for one real Temporal component | A deliberately broken implementation fails; the corrected implementation passes; unmet preconditions are classified separately; failure exactly replays | Medium; the oracle may encode an invalid guarantee |
| Next | Scenario distributions and required semantic witnesses | Reach named interactions absent from the fixed workload under equal compute; reproduce every retained witness | Medium; generator and oracle can share a blind spot |
| Next | Typed scenario shrinking and durable minimizer resume | Remove operations/data while preserving the declared failure; interrupted reduction resumes without losing accepted progress | Medium; reductions must preserve input validity and failure meaning |
| Evidence-gated | Preemption bounds or PCT | Improve known-bug discovery cost over seed/frontier baselines without weakening replay | Medium–high; current scheduling points may be the actual limitation |
| Evidence-gated | Targeted checkpoints, then narrow DPOR | Reach a previously inaccessible bug; reduced/unreduced finite models agree on outcomes and deadlocks | High; added instrumentation and unsound independence rules |

For the first extension, choose a component with a small public operation model
and an existing reliable test harness. The update registry is a candidate: the
[F6 task evidence](../../../.flow/tasks/fn-100-gomad-f6-a-package-level-functional.2.md) records a
same-timestamp admission-order bug, and
[the implementation](../../../service/history/workflow/update/registry.go) now uses an
admission sequence to break ties. Retaining that known failure as a benchmark is
more informative than adding another synthetic two-outcome `select` test.

Keep the new logic in deep modules: a workload model owns operation validity and
expected effects; a progress monitor owns recovery preconditions and deadlines;
a generator owns distributions and shrinkers; search policy consumes the existing
validated decisions. The Runner should continue owning execution, budgets, and
artifact publication, without knowing Temporal protocol semantics.

At ten times the scenario or history size, history validation and frontier growth
can dominate execution. Bound operations, outstanding requests, oracle work,
trace bytes, and reduction attempts independently. Publish capacity exhaustion
as incomplete evidence. Preserve immutable parent artifacts and record both the
failure predicate and every accepted reduction. Histories may contain payloads,
so new evidence must inherit the existing private artifact and retention rules.

Success means more distinct, replayable, semantically valid failures per unit of
compute, with smaller reproductions. Report simulator/model defects separately
from target defects, and report coverage obligations separately from passing
invariants. Do not interpret a large seed count, exact replay, or bounded frontier
completion as proof that all relevant Go or distributed-system executions were
tested.
