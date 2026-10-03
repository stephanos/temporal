# [PRODUCT] Gomad v3 Product Specification

This specification describes the current high-level behavior of Gomad v3. A statement containing **must** is a product requirement. Each section has a stable semantic identifier; code, tests, and verification reports may cite that identifier without depending on document line numbers.

Identifiers use complete uppercase words separated by dots. They describe capabilities rather than implementation components, so they should remain stable when the code is reorganized.

## [PRODUCT.PURPOSE] Purpose

Gomad must make supported Go programs repeatable under controlled runtime choices and modeled external interactions. It must explore those choices, preserve trustworthy evidence, reproduce retained observations, and distinguish target behavior from failures of the Gomad infrastructure.

## [PRODUCT.SCOPE] Product Boundary

The product includes:

- the pinned Go toolchain and deterministic runtime contract;
- target review and preparation;
- deterministic execution, exploration, simulation, evidence, replay, recovery, and qualification;
- the application-facing World and simulation contracts;
- the user-facing `gomad` command;
- the maintainer-facing `gomadtool` command and release gates.

This specification covers implemented behavior, not roadmap proposals. It specifies observable capabilities and invariants, not package layout, internal algorithms, file names, or schema versions.

## [PRODUCT.VOCABULARY] Ubiquitous Language

The capitalized terms below are the canonical product language. They describe product concepts rather than commands, packages, or stored formats. Use the specific term instead of the ambiguous **run**, **result**, or **batch**. Alternative names are identified explicitly where existing documentation or interfaces use them.

The Runner reviews and prepares a Target once for a Campaign, then launches isolated Executions with selected Seeds and control decisions. Each Execution uses the Gomad Toolchain and Deterministic I/O Contract, and may also use World or Simulation. Validated observations form a Record; retained evidence is published as an Artifact. Replay checks retained behavior, a Corpus can guide later Campaigns, and Qualification measures support for a specific toolchain and platform.

### [PRODUCT.VOCABULARY.EXECUTION] Execution and Exploration

**Target**: A user-selected Go program, package test, or provenance-backed executable that Gomad is asked to evaluate.

**Prepared Target**: The immutable executable and bound execution inputs produced by reviewing and preparing a Target for a Campaign.

**Runner**: The host orchestrator that prepares Targets, starts, controls, observes, and terminates isolated Executions, enforces wall-time limits, and publishes Evidence.

**Execution**: One isolated attempt to execute a Prepared Target with fixed inputs and control decisions. An Execution is not a Campaign.

**Seed**: A reproducible root value from which Gomad selects deterministic alternatives for one Execution. Application entropy remains a separate derived input.

**Choice**: One Gomad-controlled selection among logically eligible runtime alternatives, such as runnable goroutines or ready selection cases.

**Choice Trace**: The bounded, validated runtime records observed during an Execution when choice recording is enabled. It includes logical Choices and observations such as selection outcomes; it is not itself a replay control plan.

**Decision Tape**: The ordered branching logical Choices derived from a complete Choice Trace, bound to execution identities, and used to force and validate exact runtime replay. Each decision identifies the eligible alternatives and the selected alternative.

**Choice Replay Plan**: Identity-bound runtime replay controls. Exact replay uses a complete Decision Tape; exploration may force a finite prefix and then continue selecting Choices from the Seed.

**Campaign**: One bounded exploration effort over a Prepared Target, comprising selected Executions, a failure policy, limits, and retained evidence. A Campaign is not its plan: a Portable Plan immutably describes selected work, while a Campaign executes work and retains its Evidence.

**Choice Exploration**: Bounded exploration of alternative runtime Choice prefixes from one base Seed.

**Combined Exploration**: Bounded exploration of alternatives across runtime Choices and modeled scenario, network, storage, fault, and crash-state dimensions from one base Seed.

**Frontier**: The remaining candidate work in a Choice Exploration or Combined Exploration. Equal observed Outcomes do not make distinct forced prefixes interchangeable.

**Corpus**: A private bounded collection of replay-verified, semantically novel Artifacts that may guide later Campaigns.

### [PRODUCT.VOCABULARY.EVIDENCE] Evidence and Reproduction

**Evidence**: Canonical, bounded information that supports a product claim about an Execution, Campaign, replay, or qualification result.

**Outcome**: The semantic classification of an Execution, distinct from command status and infrastructure health.

**Record** (also called **Execution Record**): The canonical, versioned semantic description of one completed Execution and its Outcome, identities, inputs, limits, and retained Evidence.

**Artifact**: An immutable, validated, content-addressed package containing a Record and the inputs required to inspect or replay it.

**Observation**: Detached, bounded application behavior collected for comparison or evaluation, including simulation histories and terminal state.

**Transcript**: The ordered record of modeled interactions between a Target and its declared external environment.

**Identity**: A canonical digest that binds all inputs relevant to the sameness of a product object or claim. Equal names or paths do not imply equal identity.

**Provenance**: Trusted evidence that binds a Prepared Target to its binary, build origin, dependency closure, and capability review.

**Coverage**: Versioned semantic or runtime-Choice features observed across Executions. It does not mean source-line coverage.

**Probe**: A named semantic event whose observation contributes to Coverage and may be required by a Campaign or qualification workload.

**Replay**: Validation of an Artifact followed, unless verification-only, by re-execution of its retained Prepared Target and comparison with its Record.

**Exact Replay**: Replay that forces the retained branching runtime Choices and applicable modeled-environment decisions, validates complete consumption of the replay controls, and compares the recorded Outcome and terminal evidence. An Artifact replays on its recorded platform and, when applicable, its recorded Backend.

**Replay Divergence**: The first point at which replayed behavior, identity, or decision consumption differs from the retained evidence. It is not a Target failure.

### [PRODUCT.VOCABULARY.MODELING] External and Distributed Modeling

**Interaction Boundary**: The explicit reviewed contract through which a Target accesses supported host capabilities and modeled external input.

**Deterministic I/O Contract**: The versioned identity, supported operations, Adapters, limits, and Transcript rules for transparent deterministic input/output through the Interaction Boundary.

**Adapter**: A versioned deterministic replacement that brings a specific dependency's external operations inside the Interaction Boundary.

**Captured Read-Only Input**: Host file or directory entries imported on demand through declared read-only mounts, retained in an Artifact, and reused by Replay without reopening the original host path.

**World** (also called **World Model**): The deterministic in-memory event model for application-declared external requests, readiness, cancellation, logical time, delivery, snapshots, and replay. World owns neither host input/output nor application state.

**Simulation**: A bounded execution model for application-declared nodes, networks, durable storage, scenarios, faults, observations, and correctness checks.

**Cluster**: The application-facing Simulation contract for Node lifecycle, topology, faults, Observations, histories, Oracle evidence, and bounded crash-state enumeration.

**Backend** (also called **Simulation Backend**): The selected Simulation execution mechanism: in-process execution of logical Node incarnations or process execution with separate Node processes.

**Fidelity**: The guarantee claimed by a Simulation, declared and recorded separately from the selected Backend. Both Backends support Model Fidelity; only the process Backend supports Hard Isolation.

**Model Fidelity** (also called **Simulation Model Fidelity**): A claim that detached modeled transitions and Outcomes satisfy the Simulation contract. It does not claim fresh arbitrary package globals or hard cleanup of crashed work.

**Hard Isolation** (also called **Hard Isolation Fidelity**): A process-Backend guarantee that each Node Incarnation has fresh runtime and package state and can be terminated and reaped at process level.

**Node**: A named simulated participant with a stable identity, configured address, Boot Identity, volume attachments, and restartable lifecycle.

**Incarnation**: One monotonically identified lifetime of a Node between boot and crash or termination. Restart retains the Node identity and creates a new Incarnation so stale handles and model access can be rejected.

**Boot Identity**: A stable identifier registered to an application boot function. Simulation inputs bind the identifier rather than a process-local function pointer.

**Scenario**: A bounded composition of application and environment actions to be realized during a Simulation.

**Fault Plan**: A deterministic specification for selecting and applying eligible partitions, healing, crashes, restarts, or crash-persistence outcomes.

**Oracle**: A named correctness rule evaluated against detached Simulation Observations.

### [PRODUCT.VOCABULARY.DURABILITY] Durability and Distribution

**Portable Plan**: A path-independent, identity-bound description of all inputs and selected work needed to execute a supported Campaign elsewhere.

**Shard**: A deterministic disjoint subset of a Portable Plan's globally ordered work.

**Aggregate**: A newly published immutable Campaign result formed by validating and merging Shards from one Portable Plan.

**Resume**: Continue only unfinished logical work from a validated interrupted Campaign. Resume may execute the Target.

**Recovery**: Repair a recognized interrupted publication state without executing unfinished Target work. Recovery is not Resume.

### [PRODUCT.VOCABULARY.SUPPORT] Support and Maintenance

**Gomad Toolchain**: The pinned Go toolchain whose deterministic runtime changes and source overlays make supported runtime Choices, time, and host input/output repeatable.

**Capability Review**: A non-executing assessment of whether a Target's reachable external requirements fit a selected platform and compatibility policy.

**Supported Target**: A Target whose active requirements are covered by the selected qualified platform, Interaction Boundary, and Adapters.

**Unsupported Target**: A Target with an active requirement outside that reviewed support contract. Unsupported does not mean the Target itself failed.

**Qualification**: Independent repeated execution and comparison of canonical evidence against declared expectations.

**Qualification Set**: A versioned collection of workloads analyzed, qualified where supported, and reported together as one support claim.

**Support Comparison**: Classification of the difference between two validated Qualification Set reports.

**Qualified Platform**: A specific operating-system and architecture bundle that has passed the complete declared product conformance gate.

**Compatibility Pack**: A reviewed, version-pinned policy and deterministic adaptation contract for one third-party dependency version.

**Conformance Tier**: A named bounded group of checks for one layer of the Gomad product contract.

**Release Gate**: The required set of conformance and qualification evidence that permits a support or release claim.

**Upgrade Dossier**: The retained bounded evidence for accepting or rejecting a Go or product-boundary upgrade.

### [PRODUCT.VOCABULARY.FAILURE] Failure Language

**Target Failure**: A completed Execution whose application-level Outcome violates the Target's success contract.

**Watchdog Observation**: Evidence that an Execution exceeded its wall-time bound and was terminated; it is not proof of a deterministic deadlock.

**Infrastructure Failure**: A failure of Gomad, the Runner, publication, or the host to complete its own contract. It is not evidence that the Target failed.

**Capacity Exhaustion**: A declared product bound was reached before a complete claim could be produced. It must remain distinct from success and Target Failure.

## [PLATFORM] Platform and Toolchain

### [PLATFORM.SUPPORT] Supported Platform

The complete Runner and deterministic-interaction contract must be available only on explicitly qualified platform bundles; the current qualified bundles are `darwin/arm64` and `linux/amd64`. An unsupported host must be rejected before Gomad claims deterministic execution or begins a platform-specific toolchain build.

### [PLATFORM.TOOLCHAIN] Pinned Toolchain

Gomad must use a version-pinned Go toolchain whose source, deterministic runtime changes, overlays, build environment, and platform form one verified identity. Builds with the same identity may be reused; conflicting or incomplete builds must not be published as valid toolchains.

When Gomad is not activated, the toolchain must retain upstream Go behavior within the verified compatibility contract.

### [PLATFORM.INSTALLATION] Installation Resolution

Commands that execute or verify Targets must resolve one explicit, valid toolchain installation from the command line, environment, installation bundle, or adjacent installation state. Malformed or ambiguous installation metadata must fail closed rather than fall through to an unrelated toolchain.

### [PLATFORM.IDENTITY] Product Identity

Plans, campaigns, Records, Artifacts, qualification reports, and replays must bind every execution-relevant product identity, including the toolchain, platform, Runner, deterministic runtime controller, interaction boundary, selected adapters, and relevant protocol contracts.

## [TARGET] Targets

### [TARGET.KINDS] Target Kinds

Gomad must accept Go package execution, Go package tests, and prepared executable Targets with verified provenance. Target arguments must cross an argument-safe boundary without shell reinterpretation.

### [TARGET.PREPARATION] Preparation

Gomad must resolve, review, build, and validate a Target before its Campaign begins. A Campaign must execute one immutable Prepared Target rather than rebuilding independently for each Execution.

### [TARGET.CAPABILITY] Capability Review

Gomad must review the Target's reachable dependencies and host-capability boundaries against the selected platform and compatibility policy. It must report supported, blocked, guarded, and eliminated findings with stable evidence and must not execute a Target whose active requirements are unsupported.

Capability analysis must support source-closure review and linked-program review without launching the Target. Linked review must fail closed if its build identity or reachability evidence cannot be validated.

### [TARGET.PROVENANCE] Executable Provenance

A prebuilt executable must carry trusted provenance that binds the binary, package policy, dependency closure, build information, and compatibility review. Runtime arguments must be bound separately into Campaign and Artifact identity. Arbitrary or changed binaries must be rejected.

## [RUNTIME] Deterministic Runtime

### [RUNTIME.ACTIVATION] Activation

Deterministic runtime behavior must activate only through an explicit direct seed or an identity-bound Runner bootstrap. Invalid activation must fail before user initialization; absent activation must use upstream runtime behavior.

### [RUNTIME.SCHEDULING] Scheduling

For a fixed Target, toolchain, platform, deterministic inputs, and seed, supported runtime-controlled choices must repeat across fresh processes. These choices include supported goroutine scheduling, selection polling, map randomization, and equal-deadline timer ordering. Different seeds must be able to select different alternatives where alternatives exist.

The local run queue orders runtime-owned goroutines by a fixed rule and offers only user goroutines as alternatives. The rule applies while deterministic runtime behavior is active and `GOMAXPROCS` is one; with more than one P the upstream pick applies. When the scheduler takes from a local run queue holding two or more goroutines, it runs the first runtime-owned goroutine in queue order, without a seeded draw or a choice record. Runtime-owned means the runtime's own system-goroutine test (`isSystemGoroutine` with `fixed` false): a goroutine whose start function is in package `runtime`, other than `runtime.main`, the coroutine start `runtime.corostart` that `iter.Pull` uses, and the `runtime.handleAsyncEvent` start of wasm event handlers, which count as user goroutines. The finalizer goroutine and the cleanup goroutines count as user goroutines while they run user finalizers or cleanups, and as runtime-owned while they wait for or fetch work. Only a queue of user goroutines is a scheduling choice: it draws from the seeded stream and, with a Choice Trace, records one Runnable decision whose alternatives are exactly those goroutines. A queue with at most one user goroutine records nothing, and its pick takes no seeded draw. The rule depends only on the queue contents and that classification, so traced and untraced runs schedule alike. Neither class is starved: a picked goroutine leaves the queue and returns only when readied again, so runtime-owned goroutines delay user goroutines only while they keep being readied, and user goroutines never delay them. Replay and forced prefixes apply the same rule, and a Decision Tape recorded under another rule is rejected by controller identity. The rule covers the local run queue only. It leaves out the `runnext` slot that a readied goroutine takes and its seeded demotion to the queue tail, the global run queue taken in batches when the local queue is empty and its seeded batch shuffle, collector mark and idle workers that the collector picks outside the run queue once started, the execution-trace reader, and goroutines readied by timers or the network poller until they reach the local queue.

### [RUNTIME.TIME] Virtual Time

Enabled Targets must observe a process virtual clock with a fixed versioned epoch for supported time operations. When application work cannot proceed, time must advance to the next deterministic deadline without waiting for equivalent wall time. Runnable work must not be skipped merely to deliver a future timer.

### [RUNTIME.CHOICES] Choice Evidence

Gomad must optionally retain a bounded Choice Trace of logical scheduling and selection records. Exact runtime replay must derive a Decision Tape from complete trace evidence, force the recorded branching alternatives, validate each decision before applying it, consume the complete tape, and reject a tape whose Prepared Target, toolchain, platform, or controller identity differs.

### [RUNTIME.ISOLATION] Execution Isolation

Each Execution must run in a fresh contained process and working directory with controlled environment, bounded output capture, a wall-time watchdog, and complete process-group termination. Parallelism must come from independent processes rather than multiple deterministic processors inside one Target.

### [RUNTIME.TRUST] Trust Boundary

Deterministic mode is for trusted tests, not production workloads. Gomad must not claim to be an operating-system sandbox. Direct raw system calls, native code, plugins, foreign threads, or other unmodeled host interactions remain outside the supported contract unless explicitly covered by a qualified adapter.

## [INTERACTION] Deterministic Interactions

### [INTERACTION.BOUNDARY] Reviewed Boundary

Runner-managed Targets must use a versioned, reviewed interaction boundary for supported filesystem, loopback network, hostname, entropy, time, and other declared operations. The supported operation inventory must be explicit, and an unsupported call that enters the boundary must fail before using the host operation.

### [INTERACTION.ADAPTERS] Dependency Adapters

Gomad may adapt explicitly versioned third-party dependencies to the same deterministic boundaries. Adapter selection must be derived from the Prepared Target and bound into its identity; resume and replay must reject unavailable or changed adapters.

### [INTERACTION.TRANSCRIPT] Transcript

Modeled interactions must produce a bounded, ordered transcript. Replay must use the retained transcript, stop at the first mismatching operation, and never substitute live host input for missing recorded input.

### [INTERACTION.ENTROPY] Entropy

Modeled entropy must be repeatable and bound to execution identity without reusing the runtime scheduling seed as application input.

### [INTERACTION.MOUNTS] Read-Only Mounts

A Campaign may expose declared host directories as lazy read-only Target mounts. Gomad must capture stable observed entries within explicit limits, reject unsafe or ambiguous filesystem objects, and replay entirely from the captured Artifact without reopening the original host directory.

### [INTERACTION.FAIL.CLOSED] Fail-Closed Behavior

Unsupported operations, malformed protocol data, identity mismatches, capacity exhaustion, and replay divergence must remain explicit outcomes. Gomad must not silently fall back to host time, host readiness, live files, approximate schemas, or unbounded storage.

## [WORLD] Explicit Event Modeling

### [WORLD.MODEL] Event Model

World must provide a deterministic in-memory model for application-declared external requests, readiness, cancellation, logical time, and delivery. It must not perform host input/output or own application state.

### [WORLD.ORDERING] Event Ordering

World must order ready events from stable semantic fields and a versioned seeded choice where declared alternatives are equivalent. Host timestamps, pointers, goroutine identities, map iteration, and callback arrival order must not determine delivery.

### [WORLD.LIFECYCLE] Request Lifecycle

Requests and events must have stable non-reused identities and explicit state transitions. Invalid, duplicate, unknown, late, or capacity-exceeding operations must fail without partially mutating World state.

### [WORLD.QUIESCENCE] Quiescence

When the application declares quiescence, World must either deliver the earliest ready events, report a World deadlock when requests cannot become ready, or report idle when no work remains. These outcomes must remain distinct from runtime deadlock and wall-time watchdog expiration.

### [WORLD.REPLAY] Snapshot and Replay

World must support bounded canonical snapshots, validated restoration, semantic transition recording, and exact replay. Replay must report the first incompatible or missing transition without advancing past the divergence.

## [SIMULATION] Distributed-System Simulation

### [SIMULATION.HARNESS] Application Harness

Gomad must provide a bounded application-facing harness for named nodes, stable node identities, monotonically increasing incarnations, registered boot functions, topology, scenarios, observations, and oracles.

### [SIMULATION.BACKENDS] Backend and Fidelity

The Simulation contract must distinguish the selected Backend from its Fidelity claim. Both Backends must support Model Fidelity; only the process Backend may claim Hard Isolation. The process Backend must add fresh package initialization, hard crash and reap behavior, and host-owned shared models without attributing those guarantees to the in-process Backend.

### [SIMULATION.NETWORK] Network Model

The simulation must support deterministic multi-node addressing, directional links, listeners, connections, delivery delay, partitions, healing, and incarnation-aware connection outcomes. Stale traffic from a crashed incarnation must not be delivered after restart.

### [SIMULATION.STORAGE] Durable Storage Model

The simulation must model durable volumes, file and directory synchronization, operation dependencies, persisted-only crash state, restart, and bounded resumable enumeration of valid crash states.

### [SIMULATION.SCENARIOS] Scenarios and Faults

Scenarios must compose bounded sequential, repeated, chosen, and parallel actions. Fault plans must select stable eligible targets for partition, heal, crash, restart, and crash-persistence outcomes, and must record the realized action independently from the plan.

### [SIMULATION.ORACLES] Observations and Oracles

The simulation must produce detached bounded observations and support named correctness checks including invariants, exact histories, duplicate-or-loss checks, and convergence checks. Oracle failures must be retained as part of simulation evidence.

### [SIMULATION.REPLAY] Simulation Replay

Scenario, fault, network, storage, runtime-choice, and crash-state decisions must retain separate identities. Repeating the same specification, backend, seed, and inputs must reproduce equal semantic evidence; exact replay must report the first dimension and operation that diverges.

## [CAMPAIGN] Campaigns and Exploration

### [CAMPAIGN.SELECTION] Selection

A Campaign must select a finite ordered set of Seeds or bounded work from a Choice Exploration or Combined Exploration before or during execution. Seed ranges and counts must have stable ordinals, and resuming a Campaign must not reselect already planned work.

### [CAMPAIGN.EXECUTION] Execution

Gomad must prepare once, run selected work in isolated processes up to a declared parallelism bound, track progress, and produce one terminal Campaign result. Completion order must not change deterministic selection or exploration commitment.

### [CAMPAIGN.FAILURE] Failure Policy

A Campaign must support stopping after the first failure, after a distinct-failure budget, or after all selected work. Failure identity must group equivalent observations independently of their seed while preserving each Execution's evidence.

### [CAMPAIGN.COVERAGE] Coverage

Gomad must support versioned semantic-probe coverage, runtime-choice coverage, or both. A Campaign may require named probes and must fail visibly when a required probe is absent.

### [CAMPAIGN.SUCCESS] Successful Evidence

Successful Executions must be discarded by default. A Campaign may retain all successes or only successes that add declared coverage, but retention must be explicitly bounded by count and bytes and must produce replayable Artifacts.

### [CAMPAIGN.GUIDANCE] Guided Exploration

Gomad may guide seed selection from a private bounded corpus of replay-verified, semantically novel Artifacts. Each Campaign must bind one immutable corpus snapshot, preserve a portion of unguided requested seeds, and publish corpus changes atomically only after exact replay succeeds.

### [CAMPAIGN.CHOICE.FRONTIER] Choice Exploration

Choice Exploration must expand observed alternative runtime Choices in deterministic bounded rounds. It must preserve every distinct forced prefix within the declared depth, execution, and memory limits, even when Outcomes deduplicate to the same Evidence.

### [CAMPAIGN.COMBINED.FRONTIER] Combined Exploration

Combined Exploration must coordinate bounded alternatives across runtime, scenario, network, storage, fault, and crash-state dimensions from one base Seed. It must preserve deterministic Frontier order, separate logical from recovery Executions, and retain enough Evidence to resume or minimize a failure.

## [EVIDENCE] Records, Artifacts, and Replay

### [EVIDENCE.RECORD] Execution Record

Every completed Execution must produce a canonical, versioned, bounded Record containing the execution-relevant identities, outcome, full output hashes, retained output metadata, and applicable runtime, World, interaction, and simulation evidence.

### [EVIDENCE.ARTIFACT] Artifact Publication

Retained failures and retained successes must be published as immutable content-addressed Artifacts. Partial or interrupted publication must never appear complete, and existing content may be reused only after full validation.

### [EVIDENCE.INSPECTION] Inspection

Inspection must validate a plan, Campaign, merged Campaign, or Artifact before reporting its identity, lifecycle, outcome, bounds, retained evidence, replayability, and exact replay command. Optional choice inspection must validate and summarize the retained logical choice trace.

### [EVIDENCE.REPLAY] Replay

Replay must validate every identity and required payload before starting the retained Prepared Target. It must execute the retained binary on its recorded platform and, when applicable, Backend rather than rebuild from current source, compare the new semantic Outcome with the Record, and distinguish exact reproduction from divergence. Verification-only replay must perform validation without executing the Target.

### [EVIDENCE.MINIMIZATION] Failure Minimization

Gomad must minimize supported combined-simulation target failures through fresh-process, bounded candidate attempts. An accepted reduction must preserve the normalized failure, outcome, exact runtime-choice replay, and exact simulation replay. The source Artifact must remain immutable and the minimized result must retain its parent and reduction evidence.

## [DURABILITY] Campaign Durability

### [DURABILITY.LIFECYCLE] Lifecycle

A Campaign must expose explicit planned, prepared, running, committing, published, and recoverable-failure states. A validated published manifest must remain authoritative when private interrupted state is also present.

### [DURABILITY.JOURNAL] Journal

Completed Execution records and exploration rounds must be stored in bounded integrity-checked segments or equivalent immutable units. A partial terminal write must not be mistaken for a completed Execution.

### [DURABILITY.RESUME] Resume

Resume must lock and validate the interrupted Campaign, its identities, Prepared Target, completed records, retained Artifacts, limits, and strategy state. It must schedule only unfinished logical work and fail closed for published, changed, incompatible, or concurrently resumed Campaigns.

### [DURABILITY.RECOVERY] Recovery

Recovery must repair only recognized interrupted publication states under lock. It must either complete safe private cleanup, normalize to a validated resumable state, or report that no safe repair exists without altering the Campaign.

### [DURABILITY.COMPATIBILITY] Stored Compatibility

Readers may retain explicit compatibility for prior stored formats. Unsupported or ambiguous historical data must be rejected rather than silently migrated or interpreted under current semantics.

## [DISTRIBUTION] Portable and Distributed Campaigns

### [DISTRIBUTION.PLAN] Portable Plan

Gomad must be able to publish a path-independent portable plan for supported seed Campaigns. The plan must bind the complete selection, Target, product identities, bounds, environment, and captured read-only inputs required for independent execution.

### [DISTRIBUTION.SHARD] Shard Execution

A shard must receive a deterministic disjoint subset of global selection ordinals, revalidate the complete plan bundle, and record global rather than local ordinals in its Campaign evidence.

### [DISTRIBUTION.MERGE] Merge

Merge must accept only validated shards from one plan, reject overlaps and unexplained gaps unless a partial result is explicitly requested, deduplicate evidence by content identity, enforce aggregate bounds, and publish a new immutable aggregate without changing its source Campaigns.

## [QUALIFICATION] Qualification and Support

### [QUALIFICATION.REPETITION] Repeated Qualification

Qualification must prepare and execute one Target independently at least twice with the same seed and compare bounded canonical evidence. It must distinguish deterministic agreement, target failure, missing required coverage, replay divergence, unsupported capability, invalid input, and infrastructure failure.

### [QUALIFICATION.SET] Qualification Sets

A qualification set must validate a versioned manifest, analyze all workloads before executing supported ones, checkpoint completed phases, retain unsupported analysis as evidence, compare actual results with expectations, and publish one bounded path-independent report.

### [QUALIFICATION.COMPARISON] Support Comparison

Gomad must compare validated baseline and candidate qualification reports as clean, improved, regressed, review-required, or incomparable. A changed reviewed boundary must require approval of the exact reported difference identity.

### [QUALIFICATION.CONFORMANCE] Product Conformance

Maintainers must be able to run bounded conformance tiers for the builder, live capability review, deterministic runtime, and disabled upstream behavior. A release claim must require the complete platform-specific gate, not only platform-neutral host tests.

## [COMMAND] Command-Line Products

### [COMMAND.GOMAD] User Command

The `gomad` command must expose the following user workflows. Each row is normative and may be cited by its identifier.

| Identifier | Command | Required behavior |
|---|---|---|
| `[COMMAND.GOMAD.DOCTOR]` | `doctor` | Validate the resolved installation, platform, product identities, adapters, and Artifact location; report an actionable repair when unavailable. |
| `[COMMAND.GOMAD.ANALYZE]` | `analyze` | Review a Go Target's capabilities without launching it and emit human-readable or stable machine-readable evidence. |
| `[COMMAND.GOMAD.EXPLORE]` | `explore` | Run a bounded seed, choice-exploration, or simulation-exploration Campaign and report progress, result classification, and retained Artifacts. |
| `[COMMAND.GOMAD.QUALIFY]` | `qualify` | Repeat one Target under the same controls and compare its canonical evidence. |
| `[COMMAND.GOMAD.QUALIFY.SET]` | `qualify-set` | Validate or execute a declared workload set, whole or as one ordinal-modulo shard of it, and publish its aggregate qualification report. |
| `[COMMAND.GOMAD.MERGE.SET]` | `merge-set` | Validate that shard reports of one workload set cover it exactly once under one run configuration and publish the whole set's aggregate qualification report. |
| `[COMMAND.GOMAD.COMPARE.SUPPORT]` | `compare-support` | Compare two qualification-set reports and require exact approval for reviewed-boundary changes. |
| `[COMMAND.GOMAD.PLAN]` | `plan` | Publish a portable Campaign plan and its verified execution bundle. |
| `[COMMAND.GOMAD.EXECUTE.SHARD]` | `execute-shard` | Execute one deterministic shard of a portable plan. |
| `[COMMAND.GOMAD.MERGE]` | `merge` | Validate and combine shard Campaigns into one immutable complete or explicitly partial aggregate. |
| `[COMMAND.GOMAD.INSPECT]` | `inspect` | Validate and describe a plan, Campaign, aggregate, or Artifact. |
| `[COMMAND.GOMAD.REPLAY]` | `replay` | Validate and optionally re-execute a retained Artifact against its recorded observation. |
| `[COMMAND.GOMAD.MINIMIZE]` | `minimize` | Produce a smaller replay-preserving Artifact for a supported simulation failure. |
| `[COMMAND.GOMAD.RECOVER]` | `recover` | Repair a recognized interrupted Campaign publication state without executing unfinished work. |
| `[COMMAND.GOMAD.RESUME]` | `resume` | Validate and continue only the unfinished work of a resumable Campaign. |

### [COMMAND.OUTPUT] Output Contracts

Commands intended for automation must provide stable machine-readable output where declared. Routine progress and final results must remain separable, retained Artifact locations must be discoverable, and machine output must not be mixed with incidental diagnostics.

### [COMMAND.STATUS] Status Contracts

Exit statuses must keep these classes distinct: success, retained target-level mismatch or failure, invalid or unsupported input, and Gomad infrastructure failure. Commands with specialized outcomes may refine those classes but must document the mapping.

## [MAINTENANCE] Maintainer Product

### [MAINTENANCE.GOVERNANCE] Generated and Reviewed Inputs

The release descriptor, runtime changes, source overlay, interaction inventory, protocol declarations, compatibility packs, and generated consumers must have explicit owners. Generation must be reproducible, and validation must reject drift between canonical inputs and checked-in outputs.

### [MAINTENANCE.COMPATIBILITY] Compatibility Packs

Compatibility-pack development must follow discovery, human review, exact approval, generation, validation, and qualification. A pack must bind an exact dependency version, source inventories, platform scope, governance, and any approved deterministic adapter replacement.

### [MAINTENANCE.UPGRADE] Upgrade Evidence

A Go or product-boundary upgrade must produce a bounded dossier covering source and boundary differences, runtime changes, overlay collisions, generated evidence, mandatory probes, disabled upstream behavior, conformance, and platform qualification. The dossier must be retained even when a gate fails, and a boundary change must require approval of its exact identity.

### [COMMAND.GOMADTOOL] Maintainer Command

The `gomadtool` command must expose the following maintainer workflows. Each row is normative and may be cited by its identifier.

| Identifier | Command | Required behavior |
|---|---|---|
| `[COMMAND.GOMADTOOL.TOOLCHAIN.BUILD]` | `toolchain-build` | Verify, build, cache, and publish the pinned toolchain for a qualified platform. |
| `[COMMAND.GOMADTOOL.BUILD.KEY]` | `build-key` | Derive the canonical toolchain build identity from all relevant source, platform, and build-environment inputs. |
| `[COMMAND.GOMADTOOL.PATCH.MATERIALIZE]` | `patch-materialize` | Apply the exact reviewed runtime patch to a verified Go source tree. |
| `[COMMAND.GOMADTOOL.PATCH.REGENERATE]` | `patch-regenerate` | Recreate the canonical runtime patch from a reviewed candidate source tree. |
| `[COMMAND.GOMADTOOL.PATCH.VALIDATE]` | `patch-validate` | Validate the runtime patch and overlay as complete governed inputs. |
| `[COMMAND.GOMADTOOL.VERSION.GENERATE]` | `version-generate` | Generate or verify consumers of the canonical release descriptor. |
| `[COMMAND.GOMADTOOL.BOUNDARY.GENERATE]` | `boundary-generate` | Discover, qualify, generate, refresh, or verify the reviewed host-capability boundary and its compiler conformance inputs. |
| `[COMMAND.GOMADTOOL.PROTOCOL.GENERATE]` | `protocol-generate` | Generate or verify both endpoints of each declared cross-process protocol. |
| `[COMMAND.GOMADTOOL.COMPATIBILITY.PACK]` | `compatibility-pack` | Discover, review, generate from exact approval, check, and qualify version-pinned compatibility packs. |
| `[COMMAND.GOMADTOOL.SCRIPT.VALIDATE]` | `script-validate` | Enforce the approved ownership and policy boundary for repository scripts. |
| `[COMMAND.GOMADTOOL.CHECKED.RUN]` | `checked-run` | Run a bounded external command, classify timeout and exit status, and retain bounded diagnostic output. |
| `[COMMAND.GOMADTOOL.TEST]` | `test` | Execute a selected bounded conformance campaign and report the first failing evidence. |
| `[COMMAND.GOMADTOOL.UPGRADE.DOSSIER]` | `upgrade-dossier` | Run upgrade gates and publish the complete qualification dossier even when a completed gate rejects the upgrade. |

## [FAILURE] Failure and Safety Semantics

### [FAILURE.CLASSIFICATION] Failure Classification

Gomad must distinguish target failure, watchdog observation, replay divergence, capacity or invalid input, unsupported Target, and Runner or host infrastructure failure. A Campaign result must not present infrastructure failure as evidence about the Target.

### [FAILURE.BOUNDS] Bounded Operation

Every untrusted or potentially growing product surface must have an explicit bound, including process time, output, transcripts, choices, World transitions, simulation state, exploration state, journals, retained Artifacts, corpora, protocol frames, and qualification reports. Capacity exhaustion must be visible and must not produce a claim of completeness.

### [FAILURE.INTEGRITY] Integrity

Canonical inputs and durable outputs must be validated before use. Publication must be atomic in its claims, immutable after completion, and recoverable only through recognized states. Hash or identity mismatches must fail before Target execution where possible.

### [FAILURE.CANCELLATION] Cancellation and Cleanup

Cancellation, deadlines, and crashes must terminate owned processes and close owned resources within bounded cleanup work. Cleanup failure must remain an infrastructure error and must not erase already durable evidence.

## [VERIFICATION] Requirement Mapping

### [VERIFICATION.TRACEABILITY] Traceability

Verification work must map code and tests to the most specific identifier in this document. A section is verified only when its normal behavior, failure behavior, limits, and relevant compatibility behavior are covered.

### [VERIFICATION.CHANGE] Specification Change

A product behavior change must update the affected identifier or add a new nested identifier. Existing identifiers should be renamed or removed only when the product concept itself is replaced, so historical verification results remain interpretable.
