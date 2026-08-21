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

## [PRODUCT.VOCABULARY] Vocabulary

- A **Target** is the exact program or test binary to execute.
- An **Execution** is one isolated attempt to run a Target with fixed inputs and control decisions.
- A **Campaign** is a bounded collection of Executions selected and managed together.
- A **Record** is the canonical semantic description of one Execution.
- An **Artifact** is a durable, validated package containing a Record and its replay inputs.
- **Replay** is validation and optional re-execution of an Artifact.
- **Qualification** compares repeated or grouped observations against declared expectations.

## [PLATFORM] Platform and Toolchain

### [PLATFORM.SUPPORT] Supported Platform

The complete Runner and deterministic-interaction contract must be available only on explicitly qualified platform bundles; the current qualified bundle is `darwin/arm64`. An unsupported host must be rejected before Gomad claims deterministic execution or begins a platform-specific toolchain build.

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

Gomad must resolve, review, build, and validate a Target before its Campaign begins. A Campaign must execute one immutable prepared Target rather than rebuilding independently for each Execution.

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

### [RUNTIME.TIME] Virtual Time

Enabled Targets must observe a process virtual clock with a fixed versioned epoch for supported time operations. When application work cannot proceed, time must advance to the next deterministic deadline without waiting for equivalent wall time. Runnable work must not be skipped merely to deliver a future timer.

### [RUNTIME.CHOICES] Choice Evidence

Gomad must optionally record bounded logical scheduling and selection choices. Exact replay must force the recorded logical alternatives, validate each decision, consume the complete decision tape, and reject a tape whose Target or controller identity differs.

### [RUNTIME.ISOLATION] Execution Isolation

Each Execution must run in a fresh contained process and working directory with controlled environment, bounded output capture, a wall-time watchdog, and complete process-group termination. Parallelism must come from independent processes rather than multiple deterministic processors inside one Target.

### [RUNTIME.TRUST] Trust Boundary

Deterministic mode is for trusted tests, not production workloads. Gomad must not claim to be an operating-system sandbox. Direct raw system calls, native code, plugins, foreign threads, or other unmodeled host interactions remain outside the supported contract unless explicitly covered by a qualified adapter.

## [INTERACTION] Deterministic Interactions

### [INTERACTION.BOUNDARY] Reviewed Boundary

Runner-managed Targets must use a versioned, reviewed interaction boundary for supported filesystem, loopback network, hostname, entropy, time, and other declared operations. The supported operation inventory must be explicit, and an unsupported call that enters the boundary must fail before using the host operation.

### [INTERACTION.ADAPTERS] Dependency Adapters

Gomad may adapt explicitly versioned third-party dependencies to the same deterministic boundaries. Adapter selection must be derived from the prepared Target and bound into its identity; resume and replay must reject unavailable or changed adapters.

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

### [SIMULATION.BACKENDS] Backend Fidelity

The simulation contract must distinguish in-process model fidelity from process-backed hard isolation. The process backend must add fresh package initialization, hard crash and reap behavior, and host-owned shared models without attributing those guarantees to the in-process backend.

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

A Campaign must select a finite ordered set of seeds or one bounded exploration frontier before or during execution. Seed ranges and counts must have stable ordinals, and resuming a Campaign must not reselect already planned work.

### [CAMPAIGN.EXECUTION] Execution

Gomad must prepare once, run selected work in isolated processes up to a declared parallelism bound, track progress, and produce one terminal Campaign result. Completion order must not change deterministic selection or frontier commitment.

### [CAMPAIGN.FAILURE] Failure Policy

A Campaign must support stopping after the first failure, after a distinct-failure budget, or after all selected work. Failure identity must group equivalent observations independently of their seed while preserving each Execution's evidence.

### [CAMPAIGN.COVERAGE] Coverage

Gomad must support versioned semantic-probe coverage, runtime-choice coverage, or both. A Campaign may require named probes and must fail visibly when a required probe is absent.

### [CAMPAIGN.SUCCESS] Successful Evidence

Successful Executions must be discarded by default. A Campaign may retain all successes or only successes that add declared coverage, but retention must be explicitly bounded by count and bytes and must produce replayable Artifacts.

### [CAMPAIGN.GUIDANCE] Guided Exploration

Gomad may guide seed selection from a private bounded corpus of replay-verified, semantically novel Artifacts. Each Campaign must bind one immutable corpus snapshot, preserve a portion of unguided requested seeds, and publish corpus changes atomically only after exact replay succeeds.

### [CAMPAIGN.CHOICE.FRONTIER] Choice Frontier

Choice-frontier exploration must expand observed alternative runtime choices in deterministic bounded rounds. It must preserve every distinct forced prefix within the declared depth, execution, and memory limits, even when outcomes deduplicate to the same evidence.

### [CAMPAIGN.COMBINED.FRONTIER] Combined Frontier

Combined-frontier exploration must coordinate bounded alternatives across runtime, scenario, network, storage, fault, and crash-state dimensions from one base seed. It must preserve deterministic candidate order, separate logical from recovery executions, and retain enough evidence to resume or minimize a failure.

## [EVIDENCE] Records, Artifacts, and Replay

### [EVIDENCE.RECORD] Execution Record

Every completed Execution must produce a canonical, versioned, bounded Record containing the execution-relevant identities, outcome, full output hashes, retained output metadata, and applicable runtime, World, interaction, and simulation evidence.

### [EVIDENCE.ARTIFACT] Artifact Publication

Retained failures and retained successes must be published as immutable content-addressed Artifacts. Partial or interrupted publication must never appear complete, and existing content may be reused only after full validation.

### [EVIDENCE.INSPECTION] Inspection

Inspection must validate a plan, Campaign, merged Campaign, or Artifact before reporting its identity, lifecycle, outcome, bounds, retained evidence, replayability, and exact replay command. Optional choice inspection must validate and summarize the retained logical choice trace.

### [EVIDENCE.REPLAY] Replay

Replay must validate every identity and required payload before starting the stored Target. It must execute the retained binary rather than rebuild from current source, compare the new semantic outcome with the Record, and distinguish exact reproduction from divergence. Verification-only replay must perform validation without executing the Target.

### [EVIDENCE.MINIMIZATION] Failure Minimization

Gomad must minimize supported combined-simulation target failures through fresh-process, bounded candidate attempts. An accepted reduction must preserve the normalized failure, outcome, exact runtime-choice replay, and exact simulation replay. The source Artifact must remain immutable and the minimized result must retain its parent and reduction evidence.

## [DURABILITY] Campaign Durability

### [DURABILITY.LIFECYCLE] Lifecycle

A Campaign must expose explicit planned, prepared, running, committing, published, and recoverable-failure states. A validated published manifest must remain authoritative when private interrupted state is also present.

### [DURABILITY.JOURNAL] Journal

Completed Execution records and exploration rounds must be stored in bounded integrity-checked segments or equivalent immutable units. A partial terminal write must not be mistaken for a completed Execution.

### [DURABILITY.RESUME] Resume

Resume must lock and validate the interrupted Campaign, its identities, prepared Target, completed records, retained Artifacts, limits, and strategy state. It must schedule only unfinished logical work and fail closed for published, changed, incompatible, or concurrently resumed Campaigns.

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
| `[COMMAND.GOMAD.EXPLORE]` | `explore` | Run a bounded seed, choice-frontier, or combined-frontier Campaign and report progress, result classification, and retained Artifacts. |
| `[COMMAND.GOMAD.QUALIFY]` | `qualify` | Repeat one Target under the same controls and compare its canonical evidence. |
| `[COMMAND.GOMAD.QUALIFY.SET]` | `qualify-set` | Validate or execute a declared workload set and publish its aggregate qualification report. |
| `[COMMAND.GOMAD.COMPARE.SUPPORT]` | `compare-support` | Compare two qualification-set reports and require exact approval for reviewed-boundary changes. |
| `[COMMAND.GOMAD.PLAN]` | `plan` | Publish a portable Campaign plan and its verified execution bundle. |
| `[COMMAND.GOMAD.RUN.SHARD]` | `run-shard` | Execute one deterministic shard of a portable plan. |
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
| `[COMMAND.GOMADTOOL.TEST.MODE]` | `test-mode` | Resolve the declared conformance tiers and success contract for a named test mode. |
| `[COMMAND.GOMADTOOL.TEST]` | `test` | Execute a selected bounded conformance campaign and report the first failing evidence. |
| `[COMMAND.GOMADTOOL.UPGRADE.DOSSIER]` | `upgrade-dossier` | Run upgrade gates and publish the complete qualification dossier even when a completed gate rejects the upgrade. |

## [FAILURE] Failure and Safety Semantics

### [FAILURE.CLASSIFICATION] Failure Classification

Gomad must distinguish target failure, watchdog observation, replay divergence, capacity or invalid input, unsupported Target, and Runner or host infrastructure failure. A Campaign result must not present infrastructure failure as evidence about the Target.

### [FAILURE.BOUNDS] Bounded Operation

Every untrusted or potentially growing product surface must have an explicit bound, including process time, output, transcripts, choices, World transitions, simulation state, frontier state, journals, retained Artifacts, corpora, protocol frames, and qualification reports. Capacity exhaustion must be visible and must not produce a claim of completeness.

### [FAILURE.INTEGRITY] Integrity

Canonical inputs and durable outputs must be validated before use. Publication must be atomic in its claims, immutable after completion, and recoverable only through recognized states. Hash or identity mismatches must fail before Target execution where possible.

### [FAILURE.CANCELLATION] Cancellation and Cleanup

Cancellation, deadlines, and crashes must terminate owned processes and close owned resources within bounded cleanup work. Cleanup failure must remain an infrastructure error and must not erase already durable evidence.

## [VERIFICATION] Requirement Mapping

### [VERIFICATION.TRACEABILITY] Traceability

Verification work must map code and tests to the most specific identifier in this document. A section is verified only when its normal behavior, failure behavior, limits, and relevant compatibility behavior are covered.

### [VERIFICATION.CHANGE] Specification Change

A product behavior change must update the affected identifier or add a new nested identifier. Existing identifiers should be renamed or removed only when the product concept itself is replaced, so historical verification results remain interpretable.
