# Gomad: Smaller WASM guests

## Conversation Evidence

> user: "also file 7. Smaller WASM guests"

## Goal & Context

Reduce unnecessary dependencies in Temporal WASM guests and measure the resulting module size, startup cost and memory use. [paraphrase]

The existing optional-provider trimming idea names Elasticsearch as a candidate; actual dependency and registration analysis determines what can be excluded from the supported guest workload.

## Architecture & Data Models

Narrow the guest build closure at optional provider or registration boundaries. Keep required Temporal services, storage/network models and qualified workload behavior. The normal server and stock/native builds preserve their existing dependency selection and capabilities.

The qualified fn-178 candidate supplies the integrated baseline after fn-176 storage migration. Build-profile and dependency changes participate in preparation, provenance and cache identity; retained replay uses the recorded module rather than rebuilding it with a new dependency selection.

## API Contracts

Any guest-specific dependency selection must be explicit in the supported build profile and discoverable in retained provenance. An excluded provider selected by configuration produces a clear unsupported-configuration refusal, rather than silently using another provider. No new provider API or runtime scheduling behavior is introduced.

## Edge Cases & Constraints

An import that appears optional can register behavior through initialization or affect the test build closure. Confirm transitive dependencies and registration before removing it. Tests continue to exercise the same workload and failure witnesses; disabling a workload to shrink its binary is not a successful optimization.

Guest changes can rotate module, target-site or model identities. Measure repeatability and replay within each candidate's own identities, and compare functional outcomes across the baseline and candidate without assuming cross-build choice tapes are interchangeable.

## Acceptance Criteria

- **R1:** Record the dependency closure for representative qualified Temporal guests, identify removable optional providers and demonstrate which dependency edges the selected exclusions eliminate. Errors: any candidate needed by a supported workload remains included or is rejected as an optimization candidate.
- **R2:** Implement the justified guest-only exclusions and demonstrate a smaller module for the same supported workload selection; default production server and stock/native build behavior remain unchanged. Errors: explicitly requesting an excluded provider fails clearly without fallback; unsupported exclusions do not ship.
- **R3:** Preparation keys, provenance and compiled-cache admission bind the selected dependency profile and resulting module. Errors: different profiles cannot share incompatible cache entries, and retained replay with incompatible execution identities remains a classified refusal.
- **R4:** Retain matched baseline/candidate measurements on the same host for module bytes, compiled-cache bytes, cold compilation, warm preparation/load, fresh-instance startup, execution wall time and helper peak RSS; report repetitions, median/range, host/build identities and rerun commands. Errors: unavailable measurements or differences within measurement spread are labeled as such, with no assumed startup or memory gain.
- **R5:** The affected qualified workloads retain success and intentional-failure witnesses, deterministic repetition and fresh retained replay on the trimmed candidate; existing capacity and resource budgets still pass. Errors: behavior changes, missing canary failures or failed replay prevent adoption of the exclusion.

## Boundaries

- No GC, fuel, scheduler, engine or runtime-policy changes; no dependency upgrades or broad provider redesign.
- No removal of production-server features, stock/native support, differential models or test cases needed to retain workload coverage.
- No promise that smaller modules improve every phase; performance conclusions follow matched measurements.

## Decision Context

Optional dependency trimming targets compilation, cache and memory costs without changing simulated execution semantics. Elasticsearch is a concrete starting point, not a required removal if closure analysis shows it is needed. The implementation follows fn-178 so measurements include the settled cluster execution architecture.

Planning confirmed optional Elasticsearch references in the visibility factory and existing build-tag/source-closure provenance. Reuse those mechanisms; an explicit dependency selection does not require a new runtime profile framework. The accepted measurement contract follows the owner WASM measurement-host decision (any host; baseline and candidate on the same host). Tasks .1 through .3 are sequential.

## Early proof point

fn-182-gomad-smaller-wasm-guests.1 proves that a safe optional-provider exclusion shrinks a same-selection guest build and captures the baseline. If it fails, reconsider the exclusion rather than weakening workload coverage or proceeding to task .2.

## Quick commands

Use pinned tool versions and the documented helper inputs for actual guest gates.

```sh
cd tools/gomad && go test -tags test_dep ./internal/backends/wasm/backend
```

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Record the dependency closure for representative qualified Temporal guests, identify removable optional providers and demonstrate which dependency edges the selected exclusions eliminate. Errors: any candidate needed by a supported workload remains included or is rejected as an optimization candidate. | fn-182-gomad-smaller-wasm-guests.1 | — |
| R2 | Implement the justified guest-only exclusions and demonstrate a smaller module for the same supported workload selection; default production server and stock/native build behavior remain unchanged. Errors: explicitly requesting an excluded provider fails clearly without fallback; unsupported exclusions do not ship. | fn-182-gomad-smaller-wasm-guests.2 | — |
| R3 | Preparation keys, provenance and compiled-cache admission bind the selected dependency profile and resulting module. Errors: different profiles cannot share incompatible cache entries, and retained replay with incompatible execution identities remains a classified refusal. | fn-182-gomad-smaller-wasm-guests.2 | — |
| R4 | Retain matched baseline/candidate measurements on the same host for module bytes, compiled-cache bytes, cold compilation, warm preparation/load, fresh-instance startup, execution wall time and helper peak RSS; report repetitions, median/range, host/build identities and rerun commands. Errors: unavailable measurements or differences within measurement spread are labeled as such, with no assumed startup or memory gain. | fn-182-gomad-smaller-wasm-guests.1, fn-182-gomad-smaller-wasm-guests.3 | — |
| R5 | The affected qualified workloads retain success and intentional-failure witnesses, deterministic repetition and fresh retained replay on the trimmed candidate; existing capacity and resource budgets still pass. Errors: behavior changes, missing canary failures or failed replay prevent adoption of the exclusion. | fn-182-gomad-smaller-wasm-guests.3 | — |
