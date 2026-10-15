# Gomad: WASM operator integration

## Conversation Evidence

> user: "yes use sub agent to file 1. WASM operator integration"

## Goal & Context

Operators should be able to install and run the qualified WASM storage, networking and multi-instance capabilities through the normal Gomad CLI, explore seeds, retain failures and replay them. This captures the operator integration item selected from the WASM follow-up proposals. [paraphrase]

Internal storage and cluster execution already have dedicated owners, while installation currently refuses network/storage configurations and the operator refuses simulation exploration. Retained multi-instance replay already exists. This work closes the operator workflow around those capabilities after fn-176 and fn-178 qualify their execution paths.

## Architecture & Data Models

The installation-selected WASM provider uses the existing shared campaign, artifact, inspection, recovery and replay owners. Cluster execution uses the qualified helper path from fn-178; storage execution and recovery images use fn-176. Input scenarios use the existing typed, retained lifecycle with declared boundaries. The operator must retain all configuration, module and input identities needed to reproduce each execution.

Dependencies are fn-176 (qualified helper-owned storage and recovery) and fn-178 (qualified cluster execution and evidence). This spec consumes those capabilities and does not redefine their model semantics.

## API Contracts

Use the existing installation, discovery, explore, inspect, replay, recover and resume commands and their shared result and exit-status conventions. Extend their WASM admission and scenario inputs only as needed for the qualified configurations. Discovery reports the supported profiles and operations. Invalid configurations fail before execution; unsupported capabilities remain explicit refusals with no backend fallback.

Seed-based exploration covers typed cluster scenarios. Existing standalone bounded-choice exploration keeps its current contract; introducing cluster-wide forced-choice exploration is outside this delivery. Retained cluster replay continues to validate per-incarnation evidence and declared lifecycle boundaries rather than claiming one complete global choice tape.

Use additive `explore --scenario FILE` with an ordinary prepared `go-run` or `go-test` target and the explicit cooperative WASM installation. The flag selects seed campaigns only; it does not enable the existing broad native `Simulation` feature, simulation-exploration, guidance, minimization, distributed plans or stock WASI scenarios. Discovery distinguishes typed cluster seed execution from those unsupported operations. Missing/malformed scenario input or conflicting selectors refuses before preparation launches a guest. Existing invocations without the flag retain their source/behavior contract.

## Implementation Design

### Prepared scenario and crash selections (.1)

`internal/backends/wasm/cluster/operator.go` owns one bounded, strict `gomad3.wasm-scenario/v1` executable input: topology/limits, boot bindings for one prepared module, per-boot arguments/environment/configuration, incarnation-specific boot bindings, declared output/storage/network boundaries and a closed ordered list of existing start/wait/crash/stop/restart actions. The campaign supplies each seed; reject embedded replay tapes, observed outcome records, arbitrary callbacks and conflicting per-boot seed overrides. Validate the entire script's node/incarnation references, bounded actions, boundary registration and boot profile compatibility before helper startup, using the existing pure cluster/broker validators. Execute by translating actions into the existing `gomad3sim.Cluster` lifecycle calls; observed transition evidence remains a separate output. Do not serialize callback-based `ScenarioStep`/`Choose` machinery or reconstruct an executable plan from a prior execution's observed transitions.

Prepare and authenticate the immutable scenario bytes, normalized semantic input, single module, every boot/config/captured mount and any explicit crash-selection context through existing target backend payload references. Keep host paths/report metrics outside semantic identity. fn-178's final `OpenCluster`/`ClusterSession.Start` and cluster provider supply execution: sealing a boot must not start it, never-started nodes remain unexecuted, and restarts use the owner's holds, fresh stores and recovery image. Do not copy the provisional lifecycle implementation or move its unfinished requirements into this spec.

Explicit crash selections consume fn-176's full pending-operation context and pending digest, not just selected operation IDs. Reuse its strict Go/Rust pre-start validation, dependency closure, per-volume matching, context bounds and chunked descriptor transport. Retain the same context in the prepared boot/scenario, request identity and replay payload so source-less replay/resume can reproduce the selection. A valid authenticated context that differs from actual pending operations at the crash boundary is execution divergence, not malformed operator input or a successful crash. Enumeration output may supply a later sealed request's context; this route adds no live crash-enumeration command or mid-run host mutation. Preserve the absent-context/default nil bytes and stock/native behavior; no synthesized empty context or re-derived operations from IDs.

### Campaign, installation and retained operations (.2–.4)

Keep CLI parsing in `cmd/gomad/internal/cli`; a small backend-neutral scenario payload reference on existing target/preparation/execution data is the only shared seam if required. Runner continues to import no concrete WASM implementation. The WASM backend adapts prepared scenario seeds into fresh fn-178 cluster executions and projects results through existing campaign commit, failure policy, bounded output and artifact publication. Share one prepared module across nodes while retaining incarnation-local evidence and deterministic campaign seed commit order. Result projection covers success, oracle/target failure, modeled storage interruption, replay divergence, capacity, cancellation and infrastructure distinctly; a storage crash prefix never becomes a complete exact tape.

Replace installation's blanket network/storage refusal only for configurations the existing cooperative validators admit and the installed owners support. Reuse `installation.{Build,Resolve,Provider}`, `backend.Describe/Execute/ValidateReplay` and capability tables; do not make executable advertisement a claim of qualification on every host/workload. Native simulation flags stay under their current capability. Standalone stock, default cooperative and no-scenario bytes/behavior have explicit controls.

Use existing `RetainedClusterReplay`, `VerifyRetained`, `ReplayBoundRetained`, `ReopenProvider` and artifact payload/inspection owners for newly produced success/failure artifacts. Authenticate declared scenario, module, boot, captured-input and crash-context payloads before execution; resolve them exclusively from retained bytes. Preserve shared lifecycle/World/network/storage evidence, per-incarnation complete versus interrupted tapes and any fn-180 inspection metadata without requiring instrumentation. Installed helper/model/runtime identities remain mandatory, while original source, compiler, mounts and preparation/compile caches may be unavailable.

Persist all prepared scenario/boot/input references in the current campaign envelope/log before admitting seeds. Recovery and resume use `loadPreparedBackend`, existing append-log recovery and immutable prepared target validation; they never reread a changed scenario or synthesize future boot inputs from observed transitions. Honor the campaign's actual admitted/terminal-receipt/committed outcome contract: preserve acknowledged results, rerun only work its durable receipts mark incomplete, and commit in original seed order. Do not redefine this as blindly rerunning every uncommitted seed. A missing reference, identity mismatch or corrupt complete record refuses without overwriting valid evidence; the existing recognized torn-tail recovery remains owned by the log. Kill tests quiesce the process group before reading its durable prefix and reap every descendant before resuming.

### Delivery and verification (.1–.5)

Develop .1–.4 as one coupled operator subsystem with one implementation owner; preserve their sequential integration and task/R-ID accounting. .5 runs the frozen installed-workflow qualification and documentation. Planning may precede fn-176/fn-178 completion; implementation admission/acceptance follows the milestone dependencies and their final strict crash-context/client contracts. No source changes or execution gates are part of this planning pass.

Focused implementation verification covers strict schema/reference/limit and capability refusals before guest start; default nil/no-scenario preservation; explicit crash selections with full valid context, invalid dependency closure, over-limit/truncated contexts and real pending-operation divergence; delayed starts/restarts; ordered seed commits, failure policy, capacity/cancellation and helper reaping. Built-CLI end-to-end gates cover clean installation, success, intentional failure, crash/reopen, partition/heal and multi-instance crash/restart, followed by detached inspection/replay and interrupted recovery/resume with originals/caches absent or changed. Run affected host-source/lint/architecture checks, protocol/identity validation (`make -C tools/gomad validate` when needed), full `test-wasm` and relevant installed operator/qualification selectors with pinned tools; verify actual selectors executed. Retain one candidate-bound handover for all six R-IDs with commands/exits, identities, resource budgets and independent review. Preserve historical evidence and native deferrals; no PR/push/CI authority follows.

## Edge Cases & Constraints

Retained replay and campaign resume must not reopen original captured-input mounts or require the original application sources and compiler for already prepared targets. Installed helper and model identity compatibility remains mandatory. Storage crash prefixes, modeled failures, replay divergence, capacity exhaustion and infrastructure failures keep their distinct classifications.

Interrupted campaign recovery preserves completed evidence and deterministic execution ordering. Unbound live lifecycle commands and live volume crash enumeration retain their existing unsupported or non-replayable status. Qualification claims name the actual profile, host and workload covered.

## Acceptance Criteria

- **R1:** Installation accepts qualified cooperative WASM storage/network configurations, and discovery advertises the resulting supported operations. Errors: malformed or unsupported configuration and incompatible helper/model identities fail before guest execution; no silent native fallback.
- **R2:** A user can run bounded seed exploration for a typed multi-instance scenario with modeled networking, storage and declared crash/restart boundaries through the normal CLI; results and retained artifact locations use shared campaign reporting. Errors: invalid topology, lifecycle boundaries or resource bounds fail preflight; execution failures retain their established classifications.
- **R3:** CLI inspection and replay consume artifacts produced by R2 and reproduce their retained outcome, lifecycle, network/storage evidence and per-incarnation replay contract after original source, compiler and input mounts are unavailable. Errors: missing/tampered payloads or incompatible installed identities fail closed; incomplete prefixes are never labeled complete exact tapes.
- **R4:** Recover and resume handle an interrupted R2 campaign using retained prepared inputs, preserve completed evidence and produce the same logical results as an uninterrupted control. Errors: incompatible installation, damaged retained state or missing required inputs fail without overwriting valid evidence or reporting an incomplete campaign as successful.
- **R5:** Clean-install end-to-end acceptance covers storage crash/reopen, network partition/heal and a multi-instance crash/restart scenario, with a successful run and an intentional replayable failure, plus interrupted recovery/resume. Errors: an unpassed workload or host gate remains explicitly unqualified.
- **R6:** Operator documentation supplies runnable installation, exploration, inspection, replay and recovery examples and reconciles the capability table with R1-R5. Errors: unsupported modes and qualification limits are stated alongside the supported workflow; no new runtime error surface beyond R1-R5.

## Boundaries

- Coverage-guided search, reusable search corpora, new consistency oracles and failure shrinking belong to separate work.
- Campaign planning, distributed shards and merging are outside this spec; existing local campaign recovery/resume is included.
- No new storage/network semantics, arbitrary scenario callbacks, cluster-global choice search, checkpointing or revival of deferred native qualification.

## Decision Context

Complete a single operator workflow over qualified execution owners. Reusing shared campaign and evidence contracts keeps CLI integration from becoming a second execution framework. Existing cluster replay is extended only where newly produced operator artifacts require it.

Planning confirmed installation admission, existing retained replay and shared campaign reuse against current source. The executable scenario input is distinct from observed replay transitions and preserves the existing single-module restriction. Public capabilities are enabled only when the campaign route executes; supported features and workload qualification remain separate. Tasks .1 through .5 are sequential.

Planning refinement (2026-10-15): retain executable scenario and all future boot inputs before seed admission; carry fn-176's authenticated crash-selection context through preparation, replay and resume; honor existing receipt-based recovery and fn-178 delayed-start/restart semantics. Keep typed cluster seed capabilities separate from the broad native simulation feature. These choices refine existing R1–R6 without adding another campaign or replay engine.

## Early proof point

fn-181-gomad-wasm-operator-integration.1 proves that a bounded typed scenario can be prepared and executed through the existing cluster owner. If it fails, resolve the scenario/provider seam before wiring campaigns in task .2.

## Quick commands

Use pinned tool versions and the documented helper inputs for actual guest gates.

```sh
cd tools/gomad && go test -tags test_dep ./internal/backends/wasm/installation ./internal/backends/wasm/backend ./cmd/gomad/internal/cli
```

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Installation accepts qualified cooperative WASM storage/network configurations, and discovery advertises the resulting supported operations. Errors: malformed or unsupported configuration and incompatible helper/model identities fail before guest execution; no silent native fallback. | fn-181-gomad-wasm-operator-integration.1, fn-181-gomad-wasm-operator-integration.2 | — |
| R2 | A user can run bounded seed exploration for a typed multi-instance scenario with modeled networking, storage and declared crash/restart boundaries through the normal CLI; results and retained artifact locations use shared campaign reporting. Errors: invalid topology, lifecycle boundaries or resource bounds fail preflight; execution failures retain their established classifications. | fn-181-gomad-wasm-operator-integration.1, fn-181-gomad-wasm-operator-integration.2 | — |
| R3 | CLI inspection and replay consume artifacts produced by R2 and reproduce their retained outcome, lifecycle, network/storage evidence and per-incarnation replay contract after original source, compiler and input mounts are unavailable. Errors: missing/tampered payloads or incompatible installed identities fail closed; incomplete prefixes are never labeled complete exact tapes. | fn-181-gomad-wasm-operator-integration.3 | — |
| R4 | Recover and resume handle an interrupted R2 campaign using retained prepared inputs, preserve completed evidence and produce the same logical results as an uninterrupted control. Errors: incompatible installation, damaged retained state or missing required inputs fail without overwriting valid evidence or reporting an incomplete campaign as successful. | fn-181-gomad-wasm-operator-integration.4 | — |
| R5 | Clean-install end-to-end acceptance covers storage crash/reopen, network partition/heal and a multi-instance crash/restart scenario, with a successful run and an intentional replayable failure, plus interrupted recovery/resume. Errors: an unpassed workload or host gate remains explicitly unqualified. | fn-181-gomad-wasm-operator-integration.5 | — |
| R6 | Operator documentation supplies runnable installation, exploration, inspection, replay and recovery examples and reconciles the capability table with R1-R5. Errors: unsupported modes and qualification limits are stated alongside the supported workflow; no new runtime error surface beyond R1-R5. | fn-181-gomad-wasm-operator-integration.5 | — |
