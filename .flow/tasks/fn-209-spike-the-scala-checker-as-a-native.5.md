---
satisfies: [R3a, R3b, R3c]
---
# fn-209-spike-the-scala-checker-as-a-native.5 WebAssembly build of the checker loaded from a Go host

## Description
WebAssembly is the preferred simulation target for latency and packaging, not a correctness prerequisite under the latest owner snapshot revision. After committed .1, attempt the actual checker and unchanged Bank with the owner's selected standalone scala-wasm fork, pinned to an exact commit. First verify the current public/open-source and standalone-wasmtime status of GraalVM's own WasmGC backend. If it is available and demonstrably loads through wasmtime-go, use it instead and record both attempted routes. Stock Scala.js's JavaScript-host output is not standalone proof. TeaVM remains a fallback only after the selected route's concrete blocker is established; Scala Native to wasm32 is not attempted.

**Size:** M, bounded portability feasibility and host proof; concrete failures may close the spike requirement, not count as target success.
**Files:** task-owned `.flow/tmp/fn209-wasm/` and `model/framework/design/native-checker-spike.md`.
**Touches:** [model/framework/design/native-checker-spike.md, .flow/tmp/fn209-wasm/**]

Consume .1's frozen compile-time claim-capture evidence without mutating it. Pin all compiler/fork/ScalaPB/wasmtime/wasmtime-go dependencies in task-owned paths, never the user's home. Verify the fork's actual build instructions, JavaScript-backed library calls, runtime/reflection constraints and ScalaPB compatibility against primary sources and real link attempts. Preserve strict flags and unchanged Bank source/assertions. No non-Scala replacement, toy oracle or hand-coded verdict earns checker portability credit.

Build a task-owned Go host using wasmtime-go with gc, function-references and exceptions enabled. Load the real checker module beside a second distinct module in one wasmtime Store, call rows(state), and compare the admitted sample answers to a matched JVM control. Record module bytes, cold load time and in-host per-call latency across 100,000 calls, workload, warmup and host/module versions. Keep the historical 83-second Bank verifyAll reference separate from matched local measurements.

R3c requires the oracle to return to the simulation's checkpoint, not mandatory whole-store cloning. Prove either actual host-store snapshot/restoration (including reachable WasmGC objects and relevant host state if this branch is used), or oracle snapshot ids over actual immutable state values, or deterministic observation-prefix replay as the owner permits. After divergent continuation and repeated restore, answers must equal a fresh replay of the same prefix; an unknown id refuses. A divergence is a defect to fix, not a hosting limitation. Linear-memory-only copying earns no whole-GC-store credit but lack of a whole-store API is not itself a no-go when the oracle restore branch works. A trivial module is host setup evidence only, never real checker or restore credit. Preserve unsupported APIs/upstream status separately from the chosen proven restore mechanism.

Coordinate the sole compiler/toolchain lane with the conductor and .21/.2. Independent read-only investigation and source/host preparation may proceed in the isolated worktree, but no compiler/test/native packaging or toolchain launch before its explicit lane grant. Return a scoped note plus immutable recipes/inputs/artifacts/logs/statuses and all handles joined; conductor owns integration, audits and Flow completion.

**Quick commands:** complete fresh versioned build/load/parity/snapshot/latency recipes recorded in the note; retain all failed/inconclusive attempts and terminal statuses. The repository dependency remains absent; use only the retained immutable scratch dependency where applicable. No actual repository Make, held models, full export-module compilation, lifting/IR/schema/backend/generated/regeneration, production Go changes, server/Case execution, dependent implementation or publishing. Task-owned scratch Go host builds/runs and isolated target-toolchain setup are specifically admitted for .5 only.

## Acceptance
- [ ] Record the official GraalVM WASM availability/standalone-host check and the chosen pinned toolchain/fork commit. The actual checker/sample module answers from wasmtime-go with JVM-equivalent verdicts, or a concrete attempted build/link/load blocker is retained with exact diagnostics and upstream status. No stock JS-host, stub or package-only success is mislabeled standalone checker execution.
- [ ] Demonstrate same-store second-module loading for R3b and the chosen R3c restore branch with fresh-prefix-equivalent answers after divergent/repeated restore and unknown-id refusal. If whole-store snapshot is chosen, include actual reachable GC/host state, not linear memory alone; oracle immutable-state ids or deterministic prefix replay are valid owner-approved alternatives, not host-snapshot claims. Otherwise record the exact failed/unsupported experiment without treating an answer divergence as a mere hosting limitation.
- [ ] Record module size/load time and 100,000-call in-host latency on the real module when reachable, with exact workload/versions/warmup and matched JVM answers; otherwise explicitly identify which prior blocker prevents each measurement. Preserve unchanged Bank/strict flags, immutable evidence, all attempts and terminal handles. Report simulation and live-sidecar outcomes independently; no production adoption or final go decision here.

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: