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
