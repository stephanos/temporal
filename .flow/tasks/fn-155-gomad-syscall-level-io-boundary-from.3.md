---
satisfies: [R2, R3]
---
# fn-155-gomad-syscall-level-io-boundary-from.3 Record virtual-descriptor I/O and prove same-seed determinism for a single-process gRPC workload

## Description
The early proof point. Record network events at the descriptor layer, wake readiness waiters directly on the seeded path (never through a netpoll batch), and show that a single-process gRPC + HTTP workload with the boundary on and its network adapters excluded produces byte-identical transcripts for the same seed and replays exactly.

**Size:** M
**Files:** overlay descriptor layer (from .1), `internal/gomadtrace` use, `deterministicio/transcript.go`, a new qualification workload (gRPC/HTTP demo adapted from the spike `demo/main.go`) under `qualification/corpus/` with boundary-specific probes, `qualification/core.json` (or a new suite file); tests.
**Touches:** [tools/gomad3/toolchain/runtime/overlay/src/internal/**, tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go, tools/gomad3/deterministicio/transcript.go, tools/gomad3/qualification/corpus/**, tools/gomad3/qualification/*.json]

### Approach
- Emit descriptor-layer events through `gomadtrace.Record` (`internal/gomadtrace/trace.go:91-142`) with new op names; replay compares them like existing `net.*` records (`internal/gomadio/transcript.go:18-23`).
- Verify that readiness wakeups never pass through a netpoll batch (host-timed in Gomad, `overlay/src/runtime/gomad.go:724-737, 789-792`) and that virtual waiters are excluded from quiescence's `netpollAnyWaiters()` (`:1580`); fix in the descriptor layer or runtime hook if not.
- Add a corpus workload: gRPC unary, concurrent, streaming, deadline, shutdown and HTTP GETs (spike demo). Existing `stdlib.net.*` probes will not fire under the boundary; declare boundary-specific required probes instead.
- Run with the boundary on and the workload's network adapters excluded: gRPC, plus the ones its dependencies pull in (`x/net`, and `sockaddr` if selected). Same seed twice → identical choice trace and transcript; then exact replay of the retained artifact.

### Investigation targets
**Required:**
- `tools/gomad3/internal/gomadtrace/trace.go:91-142`
- `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go:725-811`
- `tools/gomad3/qualification/core.json` — `loopback-tcp-roundtrip` suite shape
- spike `demo/main.go`

### Key context
If determinism fails here and cannot be fixed within the descriptor layer or readiness ordering, stop and record the failure for .7 (R8) before .4–.6.

## Acceptance
- [ ] The gRPC/HTTP workload passes with the boundary on and its network adapters (gRPC, `x/net`, `sockaddr` if selected) excluded, with zero host sockets.
- [ ] Two runs with the same seed produce identical choice traces and I/O transcripts (diagnostic diff clean) on the platform in use.
- [ ] The retained artifact replays exactly; a deliberately altered transcript fails replay at the first differing event.
- [ ] `make -C tools/gomad3 overlay-test test-simulation test-host` pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
