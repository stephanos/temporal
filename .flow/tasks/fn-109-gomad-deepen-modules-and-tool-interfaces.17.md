---
satisfies: [R12]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.17 Select backend-specific network listener and connection implementations at creation

## Description
Stage 5, first half of R12 (F11): the network handle family. `Listener` and `Conn` each hold fields for three backends (process handle, in-process simulation endpoint, standalone state) and every method branches on which is set. Choose the implementation once at creation and let each implementation own its valid state. Filesystem handles follow in the next task; do one family at a time.

**External coordination:** overlay edit; same fn-110 and toolchain-rebuild rules as the simulation-time task.

**Size:** M
**Files:** `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/{network.go,process_network.go,simulation_network.go}`, new per-backend files, tests, `toolchain/version/version.json` for new overlay files.
**Touches:** [tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/**, tools/gomad3/toolchain/runtime/overlay/src/net/**, tools/gomad3/toolchain/version/**, tools/gomad3/version_generated.mk, tools/gomad3sim/*_test.go]

### Approach
- Current shape: `Listener` (`network.go:41-52`: `processHandle`, `owner`, `network`, local `pending`/`closed`/`deadline`) and `Conn` (`:54-68`). Process dispatch branches at `network.go:214`, `:254`, `:288` (listener) and `:330`, `:409`, `:485`, `:523`, `:542`, `:569`, `:586`, `:602` (connection), forwarding to `processNetworkAccept` / `ListenerClose` / `ListenerSetDeadline` / `ConnRead` / `ConnWrite` / `ConnOperation` (`process_network.go:55-108`). Simulation versus standalone is a second branch on `network`/`owner`.
- Target: creation (`ListenTCP` `network.go:102`, `DialTCP` `:145`, accept, `processNetworkConn` `process_network.go:110`) picks a private implementation of a small internal interface; `Listener` and `Conn` keep their exported methods and the patched `net` callers do not change. Each implementation holds only its own fields, so an impossible combination cannot be constructed.
- Not a registry: three concrete implementations selected by the existing backend conditions, no plugin mechanism, no generic dispatch helper that just relocates the `if`.
- Domain model stays shared: `simulation_network.go` (1,102 lines) keeps the semantics for both simulation backends; do not duplicate it per implementation. Host-side registration (`registerProcessNetworkConn` `:259`, `revokeProcessNetworkResources` `:300`) keeps incarnation-bound revocation.
- Preserve: local-model lock ownership and ordering, deadlines on accept/read/write, close and reset semantics (EOF on graceful stop, reset on crash), duplicate bind rejection, partial I/O results, capacity errors, stale-incarnation rejection before model mutation, and transcript recording.
- Shared operation tests run the same cases against standalone, in-process simulation and process backends; backend-specific tests keep the hard-isolation distinctions that only the process backend provides.

### Investigation targets
**Required:**
- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/network.go` (654 lines)
- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/process_network.go:23-125,259-310`
- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/simulation_network.go` (outline first)
- `tools/gomad3sim/network_toolchain_test.go`, `tools/gomad3/runner/internal/execution/io_network_toolchain_test.go`, `io_net_bind_toolchain_test.go`
**Optional:**
- `tools/gomad3/deterministicio/network_patch_test.go`

### Quick commands
```bash
cd tools/gomad3
make generate && make validate
.toolchain/bin/go test -count=1 -tags test_dep internal/gomadio
make toolchain && make overlay-test
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./runner/internal/execution -run 'Network|NetBind'
cd ../.. && tools/gomad3/.toolchain/bin/go test -count=1 -tags test_dep,gomad3_toolchain ./tools/gomad3sim -run 'Network|Process|Backend'
```

### Constraints
- No `git add`, commit, stash or worktree: the user owns commits. Record `"commits": []` in the `flowctl done` evidence and say so in the summary.
- No new third-party dependency. `tools/gomad3/go.mod` requires only `golang.org/x/mod`, so testify is unavailable inside `tools/gomad3`: follow the existing `t.Fatalf` style with whole-value comparisons there. In the root module (`tools/gomad3sim`, `tools/gomad3integration`) use `require` with `Equal`/`EqualValues`.
- Preserve existing comments with their owning code, CLI grammar/defaults, canonical bytes for fixed supplied identities, and error precedence/classification.
- This host is `darwin/arm64`. `linux/amd64` gates cannot run here: list them as incomplete in the done summary, never claim them.
- fn-105 D12/D14 replay-divergence dispositions stay unchanged. Attribute a failure to those owners with retained evidence instead of relaxing an expectation.
- Run tests with `-tags test_dep`. Baseline the Quick commands before editing so a pre-existing failure is not attributed to this task.
- Evidence and decision records go under `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/`.
## Acceptance
- [ ] Network listener and connection creation selects one private backend implementation; exported handle methods no longer branch on `processHandle` or backend fields.
- [ ] Each implementation owns only its valid state; the simulation network model is shared, not duplicated, and no generic backend registry exists.
- [ ] Shared operation tests pass for standalone, in-process and process backends; backend-specific tests retain hard-isolation distinctions.
- [ ] Duplicate bind, deadline, close/reset, partial I/O, capacity, stale incarnation and replay divergence behave as before, with validation before mutation.
- [ ] Overlay inventories validate, the toolchain rebuilds, and gomad3sim network/process tests pass on darwin/arm64; linux/amd64 is recorded as incomplete.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
