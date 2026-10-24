---
satisfies: [R2, R3c]
---
# fn-209-spike-the-scala-checker-as-a-native.2 GraalVM native-image build of the checker with one sample

## Description
GraalVM native-image is the owner's live-deployment target for functional tests, OMES, canary and live fuzz, not merely a fallback. After committed .1, build the real checker and unchanged Bank sample with its proved compile-time claim mechanism. Record JVM/native verdict parity, binary size, startup and Bank verifyAll time against the stated 83-second reference, or the evidenced blocking failure. Preserve a matched fresh JVM comparison with exact world/options rather than claiming the historical number is a matched local measurement.

The latest snapshot revision also admits this target as a synchronous simulation sidecar. Prove restoring the oracle to the simulation's prior point through snapshot ids over its actual immutable state values or deterministic observation-prefix replay. After divergent continuation and repeated restore, answers must equal a fresh replay of the same prefix; an unknown snapshot id refuses. A divergence is a defect, not a target-host limitation. Measure actual synchronous per-step round-trip latency over 100,000 calls with task-owned scratch transport and compare to .5's actual in-host measurement when available, or explicitly retain the unavailable comparison and its blocker. Report exploration-scale cost rather than assume a sidecar is too slow. Same-process/store co-snapshot is not required for correctness.

**Size:** M, bounded feasibility, no production adoption.
**Files:** task-owned scratch `.flow/tmp/fn209-graal/` and `model/framework/design/native-checker-graal-spike.md` (new companion note linked by the final decision).
**Touches:** [model/framework/design/native-checker-graal-spike.md, .flow/tmp/fn209-graal/**]

Pin the native-image toolchain through mise or task-owned framework/scratch paths; never download into the user's home. Consume .1's immutable prototype/evidence without changing it. Supply explicit reflection reachability metadata where needed and keep the throwing runtime-rewriter sentinel/no-reload boundary. Preserve unchanged Bank assertions, strict flags, complete attempt history, exact source/input/artifact hashes and fresh build workspaces. Package success is not executable parity. Do not replace the checker with a toy oracle or hand-coded Bank logic.

Run in an isolated worktree alongside .5's independent preparation. The conductor grants one coordinated compiler/toolchain block at a time; do not overlap .21 or another target's block. All shared Flow lifecycle, final integration and independent audits are conductor-owned. Return proof and the scoped companion note, not a target-wide go decision or a production replacement. The note distinguishes actual native measurements from JVM controls and from unproved launch/pinning/per-step protocol costs.

**Quick commands:** record and execute the complete versioned task-owned recipe, with terminal statuses and positive sample counts. The normal repository gate is not authorized: use the retained immutable dependency in scratch and leave repository model/build/ir-scalapb.jar absent.

Admission permits only this spike's native-image packaging/execution and necessary scratch toolchain setup. No Go/Temporal/export-module/Make/lifting/IR/schema/backend/generated/regeneration/server/Case work, production target, dependent implementation or publishing. If an actual target/toolchain incompatibility blocks the build, preserve the concrete failing command and official upstream status; missing tooling alone is not a feasibility verdict. .2 decides live-sidecar evidence independently of simulation .5; Scala Native .3 is conditional fallback.

## Acceptance
- [ ] A pinned, versioned recipe builds and runs the actual checker/Bank with .1's compile-time mechanism and no runtime rewrite fallback, with JVM-equivalent verdicts and measured size/startup/verifyAll; otherwise the exact blocking attempt, diagnostic and upstream/toolchain status are recorded without fabricated measurements.
- [ ] Actual oracle snapshot/restore or deterministic prefix replay meets R3c's fresh-prefix equivalence, divergent/repeated restore and unknown-id refusal controls. Record 100,000 synchronous per-step round trips and compare with actual in-host cost when available, or preserve the exact blocking experiment/measurement dependency without claiming success. Scratch protocol/host code proves only this experiment, not production adapter implementation.
- [ ] The companion note and preserved scratch evidence retain strict flags, unchanged Bank bytes/assertions, exact worlds, matched JVM controls, all inconclusive/red attempts, source/artifact hashes, terminal exits and nonzero executed counts. No production adoption or unmeasured simulation feasibility follows from this bounded result.

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: