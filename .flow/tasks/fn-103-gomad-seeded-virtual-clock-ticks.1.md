---
satisfies: [R1, R2, R4]
---

# fn-103-gomad-seeded-virtual-clock-ticks.1 Implement forward and strict tick policies in the runtime and profile
# fn-103-gomad-configurable-virtual-clock-tick.1 Implement the configurable clock tick in the runtime and profile

## Description
Scope cut 2026-09-29: seeded and fixed moved to fn-105-gomad-follow-ups-deferred-scope.6. Implement the forward policy (every draw at least 1 ns at the configured application point, read or wake) next to strict (today): runtime tick, profile and Campaign/Artifact identity fields, CLI and manifest plumbing, replay mismatch check, a runtime fixture and a core workload under forward, docs. strict reproduces today byte-for-byte.
## Acceptance
- tick off: all gates unchanged
- tick on: fixture and core workload repeat and replay exactly

## Done summary
Implemented the forward and strict tick policies (seeded and fixed moved to fn-105 D6). `--clock-tick=forward` on explore and qualify, `clock_tick` on qualify-set workloads and the ./tests generator, and `GOMAD3_CLOCK_TICK=forward` in direct mode advance what time.Now reports by 1 to 1024 ns per read, drawn from a splitmix stream derived from the seed and separate from scheduling. The advance lives in an offset only time.Now observes: the first version advanced faketime and broke the process-simulation time transport (runner failures in functional suites), fixed in the follow-up commit. The policy is a recorded environment entry, so it is part of Campaign, Artifact, plan, and evidence identity and replay, resume, and shards restore it; strict is its absence, so existing identities are unchanged; record validation, plan environments, the coordinator wire, and the runtime reject other values.

Evidence: runtime tier (strict reads tie; forward reads advance 1-1024 ns, repeat per seed, differ across seeds; invalid value exits 2) passed with the final implementation; the core corpus qualifies 7/7 under forward with exact replay; unit tests for config, manifest command, and record validation pass.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: e22b3bce6, 1fdc06069
- Tests: make validate test-toolchain test-runtime (darwin/arm64), gomad qualify-set core.json with clock_tick=forward: 7/7 qualified, go test ./runner ./qualification/set -run ClockTick
- PRs: