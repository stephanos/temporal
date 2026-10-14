# Guest-side cost experiments and baselines

## Conversation Evidence

> user (turn 1): "are there more levers for us that would bring those numbers for wasm execution down (ignore building wasm; pure execution)"
> user (turn 2): "yes, let's profile"
> user (turn 13): "actually expand flow spec to the other recommended parts too"
> user (turn 14, selected): "split-as-proposed"

## Goal & Context

Current WASM performance acceptance follows the [owner's Linux/arm64 host decision](../../MILESTONES.md#wasm-performance-acceptance-host).

<!-- Goal & Context: 30% [paraphrase], 70% [inferred] (profiling evidence) -->

After the host-side fixes, the guest itself dominates. The diagnostic profile attributes ≈ 60–70% of authorized execution to time outside host calls: guest code plus Wasmtime trampolines (≈ 0.43 s of 0.6 s for workflow, ≈ 1.3–1.5 s of 2.2–2.4 s for activity on darwin/arm64). What is inside that share is unknown: Go GC (single-threaded on wasip1), goroutine switching (stack unwinding and rewinding on Go/Wasm), fuel metering, or test code. Network-v2 and multi-instance runs have never been profiled, but they bypass the local model entirely.

The goal is to measure, not to change behaviour: produce the guest-side attribution and the A/B results needed to decide on guest-level changes, and the baselines the network and cluster specs measure against.

## Architecture & Data Models

- A diagnostic-only guest stack sampler samples guest stacks during execution, in a separate diagnostic helper build only, and writes a bounded profile in the Firefox Profiler format (viewable at profiler.firefox.com, call tree and flame graph). It uses Wasmtime's `GuestProfiler` (the `profiling` feature, enabled only by the diagnostic Cargo feature) driven by an epoch-deadline callback, rather than a hand-written sampler: Wasmtime then owns stack capture, symbolization and the output format, at the cost of diagnostic-only crates in the lockfile. [owner preference: fewer lines of code]
- Sampling never feeds back into the guest: all guest clocks are virtual, and a profiled run's evidence equals an unprofiled run's at the same seed. Profiled timings are never used as A/B or baseline numbers; those come from `diagnostic-profile` builds without the sampler.
- An A/B harness runs the same selection and seeds under variants and records execute time and evidence equality:
  - guest GC percent raised, or GC off under a memory limit (at most ≈ 75% of the guest memory bound);
  - guest built with `GOWASM=satconv,signext` (other build settings, including `GOEXPERIMENT=nogreenteagc`, unchanged);
  - fuel metering off (diagnostic only).
- A baseline profile is recorded for one multi-instance cluster selection, with a committed, re-runnable driver. (A separate network-v2 baseline was dropped on 2026-10-13: network-v2 runs always attach a broker, so fn-178's cluster baseline covers them.)

## API Contracts

- None for normal builds; diagnostic outputs only.
- Diagnostic helper builds accept a guest-profile request field naming an absolute output path, sampling interval and sample-count bound. Finish a valid profile and record truncation rather than cutting JSON at a byte boundary. Release helpers refuse the field through strict request decoding.
- Sampler and fuel-off builds change the engine configuration, so they report distinct diagnostic configuration revisions. Go accepts those revisions only through a test-only request switch, never in the release path.

## Edge Cases & Constraints

- Release helpers must contain no profiler, the same rule as the existing diagnostic-profile feature.
- Variants that change guest builds or engine configuration change identities; results are diagnostic and never qualification evidence.
- "Evidence identical" compares behaviour, not identity: choice trace bytes, network and WASI transcripts (environment block masked for the GC variant, whose GC setting enters through the guest environment), stdout/stderr, outcome and virtual time. Module, environment, helper and engine identities are reported separately.
- Any variant that changes evidence for the same seed is reported as evidence-changing and not recommended. A variant that fails (memory trap, timeout) is reported as failed, not as non-deterministic.
- Ticks that elapse during a host call are dropped, not charged to the resuming guest frame. Goroutine unwind/rewind overhead and fuel checks are spread across ordinary code; the report says so and quantifies them only through the scheduler-frame share and the fuel-off A/B.
- The cluster baseline is taken on the post-fn-154 integrated candidate, because fn-177 and fn-178 port post-fn-154 network semantics. It records each instance and the summed buckets plus the parent Go profile, because fn-178 merges instances into one helper.

## Acceptance Criteria

- **R1:** A diagnostic helper build can produce a guest flame profile for the Temporal activity selection, attributing guest time at least to GC, scheduler/goroutine switching and test code. Errors: a guest-profile request sent to a release helper → refused as an invalid request before execution; sample count over its bound → sampling stops and the profile records the truncation; a profile write failure → diagnostic warning only, run outcome unchanged. [paraphrase]
- **R2:** An A/B report covers GC-percent, `GOWASM=satconv,signext` and fuel-off variants against the baseline at fixed seeds (at least 3 seeds × 5 warm repetitions, median and range), recording execute time and whether evidence is identical, and ends with a written recommendation per variant. No error surface beyond R1. [paraphrase]
- **R3:** Baseline diagnostic profiles (phase, ABI and nested buckets) exist for one multi-instance cluster selection, recorded in a `.flow/artifacts` report with commit, helper and module digests, seeds, host and the re-run command, so the cluster spec can reference them. No error surface beyond the existing diagnostic profile.

## Early proof point

Task fn-172-gomad-guest-side-cost-experiments-and.1 validates the core approach (an epoch-callback sampler captures symbolized guest stacks without perturbing evidence). If it fails, re-evaluate Wasmtime's `profiling` feature with an explicit owner approval of its added crates before continuing with fn-172-gomad-guest-side-cost-experiments-and.2+.

## Boundaries

- No production behaviour changes; adopting a variant is a follow-up decision.
- Host-side changes belong to the sibling specs. [paraphrase]
- No change to the existing 16 KiB diagnostic stderr report contract.
- Required measurements use the host decision linked above; historical results keep their original host and identities.

## Decision Context

The user asked for levers on pure execution and approved profiling. Profiling showed the guest share is the largest remaining cost, and that network and cluster runs are unmeasured. [paraphrase]

### Planning decisions (2026-10-13, autonomous planning pass)

- **Wasmtime GuestProfiler (owner-approved 2026-10-13).** Use the pinned Wasmtime `profiling` feature rather than a custom sampler. The owner explicitly approved its diagnostic-only `fxprof-processed-profile`/`ittapi` dependency closure after the zero-new-package conflict was identified. Cargo.lock may add that required closure; default release builds must compile none of the profiling-feature crates. No unrelated dependency or release-path widening is approved.
- **Profile channel is a file.** The helper runs with an empty environment and a 64 KiB stderr capture, so the profile goes to an absolute path named in the diagnostic request field.
- **Distinct diagnostic configuration revisions.** Enabling epoch interruption or disabling fuel changes compiled code; faking the release revision would mislabel identity.
- **fn-166 R8 amended (owner decision, 2026-10-13).** fn-166 keeps the `diagnostic-profile` feature, so this spec and the later measurements in fn-174 and fn-178 can rely on it.
- **Scratch driver.** `wasi/zz_scratch_profile_test.go` (committed, marked temporary in its header) shows the driver shape. The committed driver replaces it functionally; deleting the scratch file is left to the owner.

Sibling specs: WASM helper per-run preparation fast path; WASM helper local host-call fast path; Filesystem namespace in the Rust helper; Durable volume crash model in the Rust helper; Network models in the Rust helper; Multi-instance clusters in one helper process.

## Quick commands

```bash
cargo +1.94.1 test --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked --features diagnostic-profile --target-dir /tmp/gomad-wasmhost-diag
cargo +1.94.1 test --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked
(cd tools/gomad && env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off go test -tags test_dep -count=1 -run '^TestDiagnostic' ./internal/backends/wasm/wasi)
```

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | A diagnostic helper build can produce a guest flame profile for the Temporal activity selection, attributing guest time at least to GC, scheduler/goroutine switching and test code. Errors: a guest-profile request sent to a release helper → refused as an invalid request before execution; sample count over its bound → sampling stops and the profile records the truncation; a profile write failure → diagnostic warning only, run outcome unchanged. | fn-172-gomad-guest-side-cost-experiments-and.1, fn-172-gomad-guest-side-cost-experiments-and.2 | — |
| R2 | An A/B report covers GC-percent, `GOWASM=satconv,signext` and fuel-off variants against the baseline at fixed seeds (at least 3 seeds × 5 warm repetitions, median and range), recording execute time and whether evidence is identical, and ends with a written recommendation per variant. No error surface beyond R1. | fn-172-gomad-guest-side-cost-experiments-and.1, fn-172-gomad-guest-side-cost-experiments-and.3 | — |
| R3 | Baseline diagnostic profiles (phase, ABI and nested buckets) exist for one multi-instance cluster selection, recorded in a `.flow/artifacts` report with commit, helper and module digests, seeds, host and the re-run command, so the cluster spec can reference them. No error surface beyond the existing diagnostic profile. | fn-172-gomad-guest-side-cost-experiments-and.4 | — |
