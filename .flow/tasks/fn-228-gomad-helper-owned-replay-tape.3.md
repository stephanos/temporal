---
satisfies: [R1, R4, R5, R6]
---
# fn-228-gomad-helper-owned-replay-tape.3 Runner stops projecting: pass trace and expectations, compare helper terminal counts

Touches: tools/gomad/internal/runner/replay_operation.go, tools/gomad/internal/backends/wasm/backend/inspection.go, backend/lifecycle.go, tools/gomad/internal/backends/wasm/wasi/helper_model.go, internal/backends/choice_request.go

## Description
Change `ChoiceRequest` to carry trace bytes, trace digest, expected tape digest and expected decision count instead of a `ReplayPlan`. `choiceCapabilityForArtifact` reads the trace payload and manifest expectations and no longer calls `DecodeStoredTrace`/`ProjectReplayPlan`; `inspection.go`'s request builder does the same. Extend the helper exit terminal with `trace_sha256`, `records`, `branching_records`; in `collectHelper` take those from the terminal and drop the exit-path `DecodeStoredTrace`, keeping the count and digest checks. `choiceTraceDivergence` compares manifest values against the terminal. Keep the storage-crash path: use the helper's reported cursor for the unconsumed-tape divergence and `ProjectInterruptedReplayPlan` as today. Inspection, publication and minimization decoders are untouched.

## Acceptance
Replay of every artifact in the runner replay test corpus yields the same match/divergence string and status as before; a storage-crash replay yields the same unconsumed-tape divergence; `go test ./internal/runner/... ./internal/backends/wasm/...` passes; no call to `ProjectReplayPlan` remains on the replay execution path (grep in a test).

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
