---
satisfies: [R1]
---
# fn-241-gomad-a-race-profile-for-wasip1-deferred.1 Lift the wasip1 race gate, rebuild the pinned toolchain, stub runtime/race

## Description
Touches: the pinned patched Go tree (internal/platform/supported.go only), tools/gomad/internal/backends/wasm/toolchain/** (preparation step that rebuilds cmd/go and cmd/compile from the patched tree for the race profile), a new overlay source race_wasm.go.txt and a wasm runtime/race package as stubs.
Add wasip1/wasm to internal/platform.RaceDetectorSupported; both cmd/go and cmd/compile consult it, so no other gate edit is allowed. Rebuild the go and compile tools from the patched tree as a preparation step bound to the race profile's identity. Land a stub runtime/race for wasm that implements the roughly 65 runtime race entry points as no-ops with a real raceinit/racefini, so the compiler's instrumented calls link. Build the toolchain choices fixture with -race and run it under the cooperative overlay; its outcome class and choice records must match the uninstrumented run. Record the rebuilt tool identities. Cross-spec ordering: does not depend on other specs; blocked on the spec-level revival trigger.

## Acceptance
Only internal/platform/supported.go changes in the patched tree; stock version remains go1.27.1; the choices fixture builds with -race for wasip1/wasm and runs to the same outcome class and choice records as the uninstrumented build; rebuilt go/compile identities are recorded as race-profile inputs; make -C tools/gomad validate green; cooperative profile identity unchanged.


## Done summary
Blocked:
Deferred by owner decision on 2026-10-23 (GOMAD_RACES.md §9: for memory races Gomad is the fourth-best detector; the fault matrix and scheduling bugs rank higher as proof of value). Revival trigger: the persistence fault matrix has saturated, the known-bug benchmark table exists, and it shows a data-race or lock-discipline bug that a workload under Gomad reaches but Gomad does not report. No implementation, PR, push or CI run until the owner revives this spec.
## Evidence
- Commits:
- Tests:
- PRs:
