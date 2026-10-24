---
satisfies: [R1]
---
# fn-241-gomad-a-race-profile-for-wasip1-deferred.1 Lift the wasip1 race gate and rebuild the pinned go and compile tools for the race profile

## Description
Touches: the pinned patched Go tree (internal/platform/supported.go only) and the preparation step under tools/gomad/internal/backends/wasm/toolchain/** plus rust/gomad/src/install/toolchain.rs. Add wasip1/wasm to internal/platform.RaceDetectorSupported; both cmd/go and cmd/compile consult it, so no other gate edit is allowed. Add a race-profile preparation step that rebuilds the go and compile tools from the patched tree, pins them like the existing toolchain inventory (toolchain::pin_before) and records their digests as race-profile identity inputs. Stock version stays go1.27.1. No runtime/race yet: this task ends when 'go build -race' for wasip1/wasm reaches the link step and fails only on the missing runtime/race package. Blocked on the spec-level revival trigger.
## Acceptance
Only internal/platform/supported.go changes in the patched tree; stock version remains go1.27.1; rebuilt go/compile digests are recorded as race-profile inputs and absent from the cooperative identity; 'go build -race' of the choices fixture for wasip1/wasm fails only on the missing runtime/race; make -C tools/gomad validate green.
## Done summary
Blocked:
Deferred by owner decision on 2026-10-23 (GOMAD_RACES.md §9: for memory races Gomad is the fourth-best detector; the fault matrix and scheduling bugs rank higher as proof of value). Revival trigger: the persistence fault matrix has saturated, the known-bug benchmark table exists, and it shows a data-race or lock-discipline bug that a workload under Gomad reaches but Gomad does not report. No implementation, PR, push or CI run until the owner revives this spec.
## Evidence
- Commits:
- Tests:
- PRs:
