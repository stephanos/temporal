# Gomad v3 capability roadmap

This roadmap collects candidate investments beyond committed work.
[MILESTONES.md](../MILESTONES.md) owns delivery constraints, remaining
functional-test dispositions, open findings, and deferred follow-ups with revival
triggers. Flow-Next specs own tasks and acceptance criteria.

Use the [README](../tools/gomad3/README.md) for implemented capabilities and platform
limits, the [specification](../tools/gomad3/SPEC.md) for requirements, and the
[architecture](../tools/gomad3/ARCHITECTURE.md) for module ownership and replay
contracts. [Downstream findings](GOMAD_CLOUD.md) identify seams consuming modules
must supply; qualification manifests and reports own workload support.

## Investment rules

- Start a capability spec when a named consumer and measurable exit criteria justify it.
- Account for the milestone findings before broadening functional-test support claims.
- Judge search by distinct replayable failures and semantic outcomes per compute-hour
  against equal-budget seed sampling; report neutral or worse results too.
- New models need declared semantics, hard bounds, exact replay, host-escape canaries,
  positive/negative/capacity tests, and performance evidence.
- Preserve separate runtime-choice, model, and execution-profile identities. Replay
  validates the target, platform, inputs, and those identities before visible effects.
- Keep model semantics in deep modules and retain native integration and race testing.
  Expected rejections and bounded search completion do not prove correctness.

## Bug finding

### Search evidence

Benchmark prefix and combined-frontier search on representative Temporal failures.
The existing two-outcome fixture shows no advantage over seed sampling. Compare
equal execution and wall-time budgets, failure signatures, semantic outcomes,
artifact bytes, frontier growth, and recovery executions. Keep raw prefix search
as the bounded diagnostic baseline until a richer policy demonstrates an improvement.

### BUG-5: Failure minimization

Crash-resumable minimizer state is delivered through `gomad minimize --resume`,
with persisted attempt order, budgets, accepted reductions, and parent identity.
Add typed scenario-input shrinkers to the existing combined-simulation reducer.
Accepted candidates preserve the normalized
failure, outcome, input validity, and exact choice/model replay; publish reductions
with lineage while keeping the parent immutable. Scenario owners supply validity
rules and shrinkers. Input shrinking does not establish causal minimality.

### BUG-6: Deterministic fault plans

Extend fault control to additional explicit World adapters when a workload needs
error, readiness-delay, cancellation, delivery-drop, or capacity-exhaustion actions.
Retain stable resource/operation/occurrence matching, separate planned and realized
fault identities, and replay rejection before mutation. Measure discovery against
the same seed budget without fault control.

### BUG-7: Later research extensions

| Candidate | Required evidence or constraint |
| --- | --- |
| PCT and preemption bounds | Stable actors, enabled sets, and preemption accounting |
| DPOR | Reviewed semantic dependencies and causality; unknown transitions stay dependent; agreement with unreduced finite-fixture search |
| Visited-state pruning | Complete future-relevant model state; World or evidence hashes cannot summarize native Go heap, stacks, timers, and runtime state |
| Compiler checkpoints and code coverage | Separate profile identities, overhead evidence, disabled-mode tests, and replay qualification |
| Deterministic GC | A demonstrated collector channel and reviewed patch policy; see milestone findings |
| Multi-P record/replay | Separate qualification; single-P exploration does not cover racy reads or weak-memory behavior |

### Extension gates

Choice protocols own validation and projection; search consumes validated records.
Preserve explicit run/depth/byte/time budgets, durable remaining work, and typed
capacity/divergence outcomes. Review generators and oracles independently.

## Compatibility

### COMPAT-5: Targeted deterministic adapters and I/O models

Choose operations and exact adapters from retained analyzer findings and workloads
they unlock. Candidates include explicit hosts records, filesystem metadata,
in-memory pipes, Unix-domain streams, and dependency adapters. Each model declares
its differences from host behavior and meets the investment rules, including error
and deadline tests. Broad UDP, raw descriptors, and subprocess support require
their own model and containment design.

Packs approve exact source/ABI facts; adapters supply modeled behavior. Exceptions
bind module version, sums, source hashes, platform, owner, workload, and adapter
identity. Preserve discovery, review, exact-digest approval, generation, checking,
and qualification. Generic forbidden-package access remains closed.

The tiered corpus and closure/linked/guarded capability handling are implemented.
Expand them only for a new invariant or live blocker. Downstream closure-mode
support belongs to [GOMAD_CLOUD.md](GOMAD_CLOUD.md) and its linked implementation spec.

### COMPAT-7: Platform bundles

Additional platforms, including linux/arm64, need their own patch/overlay identity,
boundary inventory, adapters, packs, containment and clock audits, and core/Temporal
qualification. Cross-platform conformance compares declared semantics and support;
artifacts replay on their recorded platform. Linux clock auditing remains D11 in
the milestones; downstream Linux qualification follows [GOMAD_CLOUD.md](GOMAD_CLOUD.md).

### COMPAT-8: Dependency and Go upgrade impact reports

[fn-113](../.flow/specs/fn-113-gomad-reduce-version-pin-maintenance.md) delivered
the dependency side: `gomadtool pin-impact` reports every pin a candidate `go.mod`
invalidates (unknown, never unaffected), `adapter-regenerate` re-derives adapter
anchors behind an approval digest, and `compatibility-pack refresh` re-reviews
invalidated packs up to approval. Both-platform qualification of that work is
still owed. Remaining: extend the upgrade dossier with workload support/behavior
differences, changed pack/adapter identities, and an addressable qualified
rollback bundle, and accept a Go-version candidate as impact input. Releases
require reviewed boundary differences and qualification; uncertainty and unavailable
audits remain unqualified.

Fn-113 supplies dependency pin-impact reports, reviewed exact adapter
regeneration, and per-request compatibility-pack refresh. Workload behavior
diffs and qualified rollback bundles remain future COMPAT-8 work.

## Productionization

### PROD-3: Artifact lifecycle and data policy

Add store-wide age/quota, reachability, sensitivity, capture, and export policies
beyond per-run bounds and qualification pruning. Pruning needs dry-run and lineage
checks; export inventories validated payloads. Inputs omitted under a data policy
must be identified and resupplied on replay, with artifacts marked accordingly.
Self-contained replay requires permitted retention of every input.

### PROD-4: Deterministic campaign plans, sharding, and merge

Extend static seed and qualification sharding to dynamic choice/combined frontiers.
A round coordinator must prove unique candidate ownership, durable round commits,
complete global ordinals, and resume without duplicate logical work. Remote
scheduling consumes these protocols without changing evidence or Runner semantics.

### PROD-5: Immutable release and installation bundles

Build on installation resolution, doctor, and upgrade dossiers with published
qualified bundles, attestations/SBOM/notices, host requirements, and rollback/uninstall
metadata. Verify before atomic activation and retain a qualified rollback bundle.
Declare schema-reader windows and CLI compatibility; migrations publish new evidence
without rewriting artifacts. Include the nested module in lint, vet, vulnerability,
and license checks, with clean-host and storage-fault evidence in release gates.

### PROD-6: CI integration

Provide reusable campaign orchestration beyond the existing repository qualification
jobs, using CLI plans, shards, caches, and merge. Compare support, divergence,
runtime, artifact cost, and failure signatures with a declared baseline. Preserve
typed failure classifications and export artifacts under the data policy.

### PROD-7: Observability and reporting

Extend existing qualification and merged-campaign reports with cross-campaign
projections and missing phase/resource metrics. Human/JSON reports and metrics
adapters share validated evidence; trend storage stays outside immutable artifacts.

### PROD-8: Resource control and performance

Measure preparation, frontier, minimizer, process, memory, descriptor, disk, and
network costs on clean-host and multi-host soaks. Compare overhead, growth, and
cleanup at ten times representative scale before widening concurrency or quotas.
Preserve cache identity validation, fresh execution processes, and private run dirs.

### Ownership

Keep lifecycle/recovery, retention/export policy, work identity, aggregate validation,
release tooling, and report projections in separately testable modules. The Runner
composes their interfaces; data policy and external services stay outside execution
semantics.

## Simulation

### SIM-6: Controlled schedule and fault exploration

Benchmark the existing combined frontier using [search evidence](#search-evidence).
Minimizer resume and typed input shrinking belong to [BUG-5](#bug-5-failure-minimization).
Preserve distinct candidate prefixes even when evidence deduplicates; a semantic
hash is insufficient for native-execution visited-state pruning.

### SIM-7: Evidence-driven expansion beyond v2

Choose the smallest model justified by a named consumer and retained blocker or
failure. Declare supported operations, simplifications, bounds, and backend fidelity.

| Area | Candidate extensions |
| --- | --- |
| Network | Transport loss/duplication/reordering, bandwidth/backpressure, half-open connections, discovery, Unix streams, UDP, IPv6 |
| Storage | Injected space/inode/I/O faults, torn/short writes, corruption, permissions enforcement, shared-volume or external-store adapters |
| Node/environment | Pause, boot-failure plans, clock offset/drift, resource quotas, rolling upgrades |
| Workloads/oracles | Generated Temporal actors, typed shrinkers, independent reference models, bounded linearizability/serializability, differential scenarios |

TCP packet loss needs transport semantics; arbitrary byte deletion cannot model it.

### Oracles and recovery

Prioritize independent sequential reference models and bounded progress monitors.
Exercise fault, heal, then required progress under declared availability/fairness
assumptions and logical deadlines. Coverage witnesses establish that an event was
reached; assertions must still check its result. Compare model semantics with stock
Go or a local cluster where a meaningful contract exists.

### Verification and fidelity

Cross-backend tests compare detached model state, histories, oracles, and normalized
outcomes; each artifact replays on its recorded backend. Hard-crash claims require
the process backend, including fresh initialization, reap, and unambiguous operation
commit during IPC failure. Qualify small service pairs before Temporal retry/timeout,
persistence recovery, partition/heal convergence, and crash-after-ack durability.
Retain malformed-input, capacity, replay-mutation, interruption, and host-load checks.

## Research

The dated [Loom and simulation-testing assessment](../docs/research/gomad/GOMAD_CMPv2.md)
covers workload, oracle, and search-policy candidates. The
[system-level determinism study](../docs/research/gomad/GOMAD3_OS.md) proposes a
linux/arm64 escape firewall and a separate mixed-language machine-replay experiment.
