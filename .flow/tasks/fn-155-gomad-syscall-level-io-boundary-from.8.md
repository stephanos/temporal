---
satisfies: [R11]
---
# fn-155-gomad-syscall-level-io-boundary-from.8 Admit the syscall edge's socket entry points in the capability guard and closure policy under the boundary

## Description
Admit the socket entry points the syscall edge models, under the boundary profile only, in both of Gomad's capability layers: the compile-time guard prologue on exported `syscall`/`golang.org/x/sys` functions (guarded mode) and the closure policy that rejects third-party packages importing them (closure mode). Without this, upstream `net` and unadapted gRPC throw `GOMAD_CAPABILITY_DENIED` or fail preparation before reaching the edge. Split from .1/.2 because it changes capability policy and generated protocol code, a separate review surface.

**Size:** M
**Files:** `internal/gomadtool/generation/protocol/protocol.go` (source of the guard exemptions and forbidden imports), regenerated `toolchain/runtime/overlay/src/cmd/internal/gomadcap/protocol_generated.go`, `toolchain/runtime/overlay/src/cmd/compile/internal/gomadguard/guard.go`, `toolchain/runtime/overlay/src/runtime/gomad.go` (`gomadCapabilityGuard`), `target/internal/capabilitypolicy/policy.go`, `target/target.go`, `deterministicio/profile.go` (bind admission into the boundary profile's identity); tests.
**Touches:** [tools/gomad3/internal/gomadtool/generation/protocol/**, tools/gomad3/toolchain/runtime/overlay/src/cmd/**, tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go, tools/gomad3/target/**, tools/gomad3/deterministicio/profile.go]

### Approach
- Define the admitted set from what the edge models (socket, bind, listen, accept, connect, read, write, writev, close, shutdown, fcntl, get/setsockopt, getsockname/getpeername, plus the generic `Syscall*`/`syscall6` entries reaching them). Everything else in `syscall`/`x/sys` stays denied.
- Guarded mode: `gomadCapabilityGuard` (`runtime/gomad.go:217-221`) throws unconditionally when Gomad is enabled. Make the admitted entry points pass only when the run selected the boundary (runtime flag from .2), and only on the paths the edge serves; keep `IsGuardExempt` (`protocol_generated.go:120-144`) generated from the protocol source, not hand-edited.
- Closure mode: `capabilitypolicy/policy.go:131-133` flags non-standard packages importing `syscall`/`x/sys`; `target.go:703-704` makes that a preparation failure. Under the boundary profile, admit those imports when the reachable calls are in the admitted set, and record the admission in the prepared target's evidence and the profile identity (memory: profile-adapter-changes-leave-libc-2026-10-01 — identity must change with policy).
- Keep the boundary-off path unchanged in behavior: the same denials, findings and errors.

### Investigation targets
**Required:**
- `tools/gomad3/toolchain/runtime/overlay/src/cmd/compile/internal/gomadguard/guard.go:25-37`
- `tools/gomad3/toolchain/runtime/overlay/src/cmd/internal/gomadcap/protocol_generated.go:99-144`
- `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go:217-221`
- `tools/gomad3/target/internal/capabilitypolicy/policy.go:131-133`
- `tools/gomad3/target/target.go:703-728`
**Optional:**
- `tools/gomad3/deterministicio/grpc_adapter.go:62-78` — what the adapter strips today

## Acceptance
- [ ] Boundary off: guarded-mode and closure-mode tests show unchanged denials for `syscall`/`x/sys` entry points and imports.
- [ ] Boundary on, guarded mode: upstream `net` TCP and unadapted gRPC's keepalive `x/sys` socket-option call run without `GOMAD_CAPABILITY_DENIED`; a non-modeled entry point (e.g. `syscall.Kill`) still throws.
- [ ] Boundary on, closure mode: preparing the gRPC workload with its network adapters excluded succeeds, and the admission appears in the prepared target's evidence and profile identity; a package importing non-modeled `x/sys` calls is still rejected.
- [ ] Generated protocol code is regenerated, not hand-edited; `make -C tools/gomad3 validate-toolchain test-toolchain intercept-test test-host` pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
