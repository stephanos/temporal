---
satisfies: [R1, R3, R9]
---
# fn-155-gomad-syscall-level-io-boundary-from.2 Select the boundary and adapter exclusions per run and bind them into artifact identity

## Description
Add the per-run boundary choice and the per-run adapter exclusion, both bound into the recorded run and artifact identity, so replay refuses mismatches. With the boundary selected, the standard-library-level network hooks stand aside.

**Size:** M
**Files:** `deterministicio/profile.go`, `deterministicio/adapter_registry.go`, `record/validation.go`, `runner/runner.go`, `runner/campaign_shard_execution.go`, `runner/replay_operation.go`, `runner/resume.go`, `runner/internal/corpus/corpus.go`, `cmd/gomad/internal/cli/cli.go`, `qualification/set/set.go`, overlay `internal/gomadio/gomadio.go` (`NetworkEnabled`), runtime bootstrap reading; tests.
**Touches:** [tools/gomad3/deterministicio/**, tools/gomad3/record/**, tools/gomad3/runner/**, tools/gomad3/cmd/gomad/**, tools/gomad3/qualification/set/**, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/gomadio.go, tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go]

### Approach
- Model both options on the clock-tick policy: CLI flag (`cmd/gomad/internal/cli/cli.go:617`, `qualify.go:79`), recorded environment (`record/validation.go:511-517`), Runner wiring (`runner/runner.go:586-591`), shard check (`runner/campaign_shard_execution.go:115-130`), workload field (`qualification/set/set.go:75,719,792`).
- The boundary selects a second named profile in `deterministicio/profile.go` (Contract Name/Implementation/Inventory, L30-34) whose inventory differs, so bootstrap hashes and `MatchesRecorded` refuse mismatches. Route the ~15 callers of `Default()` (e.g. `runner/replay_operation.go:443-445`, `runner/resume.go:51`, `runner_local.go:80`, `corpus.go:372`) through the selected profile.
- Carry the selection to the target through the bootstrap frame/profile, not a raw environment variable (Gomad scrubs the environment).
- Own the single switch that makes std-level TCP hooks stand aside, for both the standalone and simulation networks (.4 consumes it). Split `gomadio.NetworkEnabled()` (`gomadio.go:54-60`): TCP hooks in `net/gomad.go` fall through to upstream `net` under the boundary, while the resolver (`gomadInterceptResolverLookupIPAddr`) and interface hooks (`net/gomad.go:34-52`) keep their current behavior.
- Startup check: selecting the boundary where generic dispatch is unsupported fails before the target runs; test it with an injected unsupported dispatch table, since both qualified platforms support it.
- Adapter exclusion filters `requireAdapterSums` (`adapter_registry.go:159-197`) and feeds the excluded set into the profile's implementation identity.

### Investigation targets
**Required:**
- `tools/gomad3/deterministicio/profile.go:30-214`
- `tools/gomad3/deterministicio/adapter_registry.go:159-197`
- `tools/gomad3/record/validation.go:511-517`
- `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go:124-158`
**Optional:**
- memory entries profile-adapter-changes-leave-libc-2026-10-01 and shard-merge-and-prepared-target-cache-2026-09-29 (bind identity to actual sources and adapter set)

## Acceptance
- [ ] `gomad` run/explore/qualify accept the boundary option and an adapter-exclusion option; both appear in the recorded environment and artifact identity.
- [ ] Excluding an unknown adapter is rejected before preparation; a workload needing an excluded adapter fails with the existing closure error.
- [ ] Replay of an artifact under a different boundary or adapter set is refused before execution (tests for each).
- [ ] With the boundary selected, std-level TCP hooks do not record; with it off, existing behavior and identities are unchanged.
- [ ] Under the boundary, `localhost` resolution still records at the gomadio level (test).
- [ ] Selecting the boundary with an injected unsupported dispatch table fails at startup with a clear error (unit test).
- [ ] `make -C tools/gomad3 test-host` passes.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
