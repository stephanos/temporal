# Gomad: ringpop membership on WASM within reason

## Conversation Evidence

> user: "why was ringpop stubbed out? are we addressing that?"
> user: "it would be nice to support it WITHIN reason. for example when we have a multi-cluster setup to verify it works and has no bugs. but we don't want to build the ocean."
> user: "yes file"

## Goal & Context

Ringpop membership is stubbed out of the `gomad_wasm_slim` guest build by `temporal/membership_gomad.go`, which replaces the default membership module with an error telling the operator to supply static service hosts. The comment on the stub says ringpop "reaches syscall through tchannel and thrift"; that was written for the native interception boundary. For WASM it is not load-bearing, verified on 2026-10-19:

- `GOOS=wasip1 GOARCH=wasm go build ./common/membership/ringpop/` and `go build -tags disable_grpc_modules,test_dep ./temporal/` both exit 0 with stock Go 1.27.1 and without the slim tag.
- tchannel-go v1.22.1 uses `net.Listen("tcp", ...)`, `net.Dial("tcp", ...)` and `net.Dialer`; the cooperative overlay (`internal/backends/wasm/toolchain/net_stream_wasm.go.txt`) models listen, connect, accept, read, write, deadlines and ephemeral port allocation on node addresses.
- tchannel's only raw-socket use, `getSysConn` in `connection.go`, is optional: a missing `syscall.RawConn` is logged and the connection keeps a nil handle.
- Ringpop gossip is timers plus seeded random peer selection, both already virtual under gomad. No UDP, DNS or `os/user`.

The owner wants ringpop supported within reason: enough to run a multi-node membership scenario under gomad and find real bugs in it, without modeling tchannel internals or changing how testcore boots. testcore keeps its static host table; this spec changes nothing on the functional-suite path.

## Architecture & Data Models

Three gates, no new models:

1. **Unstub.** Remove ringpop from the `gomad_wasm_slim` exclusion: `temporal/membership_gomad.go` and its test keep the `gomad` tag only, so the native build is untouched. Cloud archivers (`common/archiver/provider/provider_cloud_gomad.go`) and the SQL driver trim (`tests/testcore/flag_sql_wasm_slim.go`) stay excluded. `backend.TemporalGuestBuildTags()` is unchanged. Record the slim module byte delta before and after from the prepared module file lengths, with compiler, source and module SHA-256 identities bound to each preparation. Module bytes alone establish no performance gain.
2. **Single guest.** Prepare `common/membership/ringpop` as a `go-test` target under the cooperative profile with the Temporal guest build tags and run its eleven existing tests (`factory_test.go`, `monitor_test.go`) over a bounded seed set. Those tests start several ringpop instances in one process over loopback, so this exercises tchannel on the modeled network without scenario machinery. Every outcome is classified with the fn-221 vocabulary: pass, refused import, capacity, timeout, guest failure. A stall is the expected failure mode: time advances only when every goroutine is idle, so a harness that waits on wall time spins instead. A stall is fixed in the test harness or with a declared `ClockPolicy.ReadStepNanos`, never in gomad's clock.
3. **Multi-node scenario.** One typed `gomad-wasm.scenario/v1` fixture under `internal/backends/wasm/command/testdata/scenario/` with three or four nodes, each booting a small guest program that joins a ringpop ring over the modeled network, a partition between two nodes, a timed heal, and an output marker when every node's membership view has converged. The guest program lives under `tools/gomad/internal/backends/wasm/testdata/` beside the existing fixtures and is the only new Go code outside gomad's tests. Explore it with the existing `explore --scenario` grammar over a bounded seed set; retained failures replay and inspect through the existing verbs.

## API Contracts

- `gomad explore ... go-test ./common/membership/ringpop -- -test.run=^Test...$` under the cooperative profile prepares and executes without a refusal attributable to the former stub.
- Selecting ringpop membership in a slim guest no longer returns `ringpop membership is not built under the gomad or gomad_wasm_slim build tags`; the native `gomad` build keeps that error.
- The scenario fixture declares its nodes, links, boot arguments and crash/stop markers in the existing schema; no new scenario fields. The convergence marker is an ordinary output-bound marker.
- No change to `wasi.Config`, the helper protocol, the network model, the clock model or `testcore`.

## Edge Cases & Constraints

- Frozen clock: gossip periods and suspicion timeouts drive on virtual timers and advance at idle. A busy-waiting harness stalls; the Runner wall watchdog bounds it and the outcome is classified timeout with the stall site named.
- Ephemeral ports: ringpop instances bind `127.0.0.1:0` in the unit tests; the model allocates deterministic ports. In the scenario each node binds its configured node address.
- `syscall.RawConn` absence produces an error log line per connection in tchannel; that is noise, not a failure, and the expected output filters must tolerate it.
- Seeded exploration of gossip may expose real nondeterminism or ordering bugs in ringpop or tchannel; those are findings to retain and report, not gomad defects, unless the first-divergence report points at the model.
- Module size: unstubbing grows the slim guest. If the delta pushes the one-box cell past its recorded capacities, the unstub is gated behind a per-workload tag rather than reverted.
- No CI run, PR or push is authorized by this spec.

## Acceptance Criteria

- **R1:** Ringpop membership builds into the `gomad_wasm_slim` guest; the native `gomad` build still stubs it; cloud archivers and SQL driver trim remain excluded; the slim module byte delta is recorded. Errors: a native build that links ringpop, or an unrecorded size delta, is a finding.
- **R2:** All eleven `common/membership/ringpop` tests run under the cooperative profile over at least three seeds with every outcome classified; passes replay exactly; any stall is diagnosed to a named site and resolved in the harness or by a declared read-step, with the choice recorded. Errors: an unclassified outcome or a clock change inside gomad is a finding.
- **R3:** A retained multi-node scenario with partition and heal converges on every explored seed or retains a replayable failure with its first-divergence report; the fixture, guest program and exact commands are checked in and documented in the WASM operator guide. Errors: a convergence claim without retained replay is a finding.
- **R4:** testcore, `wasi.Config`, the helper protocol and the network and clock models are byte-for-byte unchanged by this spec. Errors: any diff under those owners is a finding.

## Boundaries

- No deterministic model of tchannel internals, TCP keepalive, socket options, DNS or UDP.
- No change to how testcore boots; static service hosts stay the functional-suite path.
- No gossip tuning or ringpop source changes beyond its own test harness if a stall needs one.
- No new scenario schema fields, helper imports or model types.
- No CI run, PR or push.

## Decision Context

Owner decision (2026-10-19): support ringpop within reason, for a multi-node setup that verifies membership works and has no bugs, without building the ocean. The stub was kept from the native build where tchannel crossed the syscall interception boundary; under WASM every network call goes through `net` onto the modeled stream, so the stub only removes coverage. Rejected: modeling tchannel's framing in the helper (unneeded, the modeled TCP stream carries it); changing testcore to boot with ringpop (the functional suite does not use it and would gain nothing); leaving the stub and filing ringpop as permanently unsupported (loses the one membership mechanism a real multi-node deployment uses).

Sequencing: after the fn-221 triage sweep, which reports whether any functional tests are blocked only by the stub; gate 1 may land earlier if the sweep wants it.

## Early proof point

Gate 2's first run: prepare `common/membership/ringpop` under the cooperative profile and run one test at one seed. If preparation refuses an import, stop and report the import before any scenario work.

## Quick commands

```sh
cd tools/gomad && GOOS=wasip1 GOARCH=wasm go build ../../common/membership/ringpop/   # expect exit 0
gomad explore --json --seeds=1,2,3 --parallel=1 --on-failure=all --artifacts /absolute/ringpop-artifacts \
  go-test ./common/membership/ringpop -- -test.run=^TestRingpopMonitor
gomad explore --json --scenario tools/gomad/internal/backends/wasm/command/testdata/scenario/ringpop.json --seeds=7,19 \
  --artifacts /absolute/ringpop-scenario go-run ./tools/gomad/internal/backends/wasm/testdata/ringpop
```

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Unstub in the slim guest, native unchanged, size delta recorded. | fn-224-gomad-ringpop-membership-on-wasm-within.1 | — |
| R2 | Ringpop unit tests classified and replaying under the cooperative profile; stalls resolved outside gomad's clock. | fn-224-gomad-ringpop-membership-on-wasm-within.2 | — |
| R3 | Multi-node partition/heal scenario converges or retains a replayable failure; documented. | fn-224-gomad-ringpop-membership-on-wasm-within.3 | — |
| R4 | testcore, config, protocol and models unchanged. | fn-224-gomad-ringpop-membership-on-wasm-within.1, fn-224-gomad-ringpop-membership-on-wasm-within.2, fn-224-gomad-ringpop-membership-on-wasm-within.3 | — |
