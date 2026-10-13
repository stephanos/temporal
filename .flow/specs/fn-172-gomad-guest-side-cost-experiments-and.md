# Guest-side cost experiments and baselines

## Conversation Evidence

> user (turn 1): "are there more levers for us that would bring those numbers for wasm execution down (ignore building wasm; pure execution)"
> user (turn 2): "yes, let's profile"
> user (turn 13): "actually expand flow spec to the other recommended parts too"
> user (turn 14, selected): "split-as-proposed"

## Goal & Context

<!-- Goal & Context: 30% [paraphrase], 70% [inferred] (profiling evidence) -->

After the host-side fixes, the guest itself dominates. The diagnostic profile attributes ≈ 60–70% of authorized execution to time outside host calls: guest code plus Wasmtime trampolines (≈ 0.43 s of 0.6 s for workflow, ≈ 1.3–1.5 s of 2.2–2.4 s for activity on darwin/arm64). What is inside that share is unknown: Go GC (single-threaded on wasip1), goroutine switching (stack unwinding and rewinding on Go/Wasm), fuel metering, or test code. Network-v2 and multi-instance runs have never been profiled, but they bypass the local model entirely.

The goal is to measure, not to change behaviour: produce the guest-side attribution and the A/B results needed to decide on guest-level changes, and the baselines the network and cluster specs measure against.

## Architecture & Data Models

- A diagnostic-only guest profiler (Wasmtime's guest profiling support) samples guest stacks during execution, in diagnostic helper builds only, and writes a profile viewable as a flame graph.
- An A/B harness runs the same selection and seed under variants and records execute time and evidence equality:
  - guest GC percent raised, or GC off under a memory limit;
  - guest built with `GOWASM=satconv,signext`;
  - fuel metering off (diagnostic only).
- Baseline profiles are recorded for one network-v2 Temporal selection and one multi-instance cluster selection.

## API Contracts

- None for normal builds; diagnostic outputs only. [inferred]

## Edge Cases & Constraints

- Release helpers must contain no profiler, the same rule as the existing diagnostic-profile feature.
- Variants that change guest builds or engine configuration change identities; results are diagnostic and never qualification evidence.
- Any variant that changes evidence (choice traces, transcripts) for the same seed is reported as non-deterministic and not recommended.

## Acceptance Criteria

- **R1:** A diagnostic helper build can produce a guest flame profile for the Temporal activity selection, attributing guest time at least to GC, scheduler/goroutine switching and test code. Errors: requesting a profile from a release build → refused; profile output over its size bound → truncated with an explicit marker. [paraphrase]
- **R2:** An A/B report covers GC-percent, `GOWASM=satconv,signext` and fuel-off variants against the baseline at fixed seeds, recording execute time and whether evidence is identical, and ends with a written recommendation per variant. No error surface beyond R1. [paraphrase]
- **R3:** Baseline diagnostic profiles (phase, ABI and nested buckets) exist for one network-v2 selection and one multi-instance cluster selection, recorded where the network and cluster specs can reference them. No error surface beyond the existing diagnostic profile. [inferred]

## Boundaries

- No production behaviour changes; adopting a variant is a follow-up decision. [inferred]
- Host-side changes belong to the sibling specs. [paraphrase]

## Decision Context

The user asked for levers on pure execution and approved profiling. Profiling showed the guest share is the largest remaining cost, and that network and cluster runs are unmeasured. [paraphrase]

Sibling specs: WASM helper per-run preparation fast path; WASM helper local host-call fast path; Filesystem namespace in the Rust helper; Durable volume crash model in the Rust helper; Network models in the Rust helper; Multi-instance clusters in one helper process.
