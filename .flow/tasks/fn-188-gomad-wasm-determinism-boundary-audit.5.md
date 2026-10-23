---
satisfies: [R3]
---
# fn-188-gomad-wasm-determinism-boundary-audit.5 Same-seed stress producer with host perturbation axes and retained report

## Description
**Touches:** tools/gomad/internal/backends/wasm/internal/measure/**, tools/gomad/internal/backends/wasm/wasmhost/src/**, tools/gomad/internal/backends/wasm/cmd/wasmmeasure/** and fn-188 retained stress reports (existing Go producer carry-over only under the recorded ordering decision).

Add the `stress` producer as a subcommand of the fn-187.2 measurement command if it has landed, otherwise as an env-gated producer in the same shape (explicit host and identity flags, never a test target). For each cell (workload x seed x axis) run K fresh executions holding declared inputs fixed and perturbing one undeclared host condition: CPU load (busy threads, reuse the native conformance/soak pattern), concurrent instances (fan-out), process layout (helper restarted per execution with varied cache path length and environment size, kept outside the recorded environment), and handoff timing via a new `stress-jitter` helper Cargo feature that delays baton handoffs and remote replies by bounded random amounts and is compiled out of the release helper; the Go side refuses a `stress-jitter` helper for any qualification role. Compare every repetition to the unperturbed reference using the first-divergence report. Workloads: cooperative runtime qualification seeds, the fn-178 two-cluster scenario (seeds 7, 19, 42), single-guest workflow and activity selections (seeds 0, 1000003, 2000006). Retain one artifact under `.flow/artifacts/` with the matrix, per-cell pass/divergence/skipped (fewer repetitions than requested is skipped), every divergence report, identities and rerun commands; a cell that changes an identity or uses forced replay is rejected by the producer. Run the full matrix on a quiet host and retain the report; any divergence is a finding routed to task .6.

**Size:** L
**Files:** tools/gomad/internal/backends/wasm/cmd/wasmmeasure/stress.go (or determinism/cmd), tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml (feature), tools/gomad/internal/backends/wasm/wasmhost/src/cluster/baton.rs, tools/gomad/internal/backends/wasm/wasmhost/src/namespace_transport.rs, tools/gomad/internal/backends/wasm/wasi/process.go (feature refusal), .flow/artifacts/wasm-determinism-stress/**

**Ordering (conductor decision 2026-10-21, owner asked to avoid double work):** do not build the stress producer against the Go Runner/measurement host. Resume after fn-232.10 (qualify in Rust) and implement the producer to drive the Rust binary, so fn-232.13's Go host retirement does not require porting it. Existing Go-producer receipts remain history. Revival trigger: fn-232.10 done.

## Acceptance
- [ ] Producer runs the workload x seed x axis matrix with fresh seeded executions and compares via the divergence report
- [ ] `stress-jitter` feature exists, is absent from the release helper, and is refused for qualification roles
- [ ] Forced-replay cells and identity-changing cells rejected; short cells marked skipped
- [ ] Retained report with host evidence, identities and rerun commands; every cell pass or each divergence carried as a finding

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

### Postponed by owner (2026-10-23)

The owner said: "can we postpone the Stress testing? we won't have a calm system for a while". The full stress matrix is not part of the fn-232.13 one-hour cycle and has no scheduled window.

**Revival trigger:** the owner declares a quiet-host window. It then runs on the candidate current at that time, using worker B's stress producer and assembler: `gomad-measure --stress-inputs` and `groups_r4.py stress()` in /home/agent/evidence/fn232-composition-v1/.

Nothing about this acceptance is waived.

### Reinstated as a correctness check (2026-10-23)

This supersedes the postponement above. The owner approved running the full matrix right after the freeze as a determinism check, even though the host is loaded. A divergence is a finding regardless of load. Wall time and timing results are recorded, and judged only in the quiet-host window.
