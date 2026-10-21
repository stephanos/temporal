# fn-224.3 — three-node ringpop partition/heal scenario

Branch `fn224-ringpop`. Raw evidence: `/home/agent/wt/evidence/fn224/scenario/` (runs, bisect variants, `DIAGNOSIS.md`).

## Fixture

- Guest: nested module `tools/gomad/internal/backends/wasm/testdata/ringpop` (`main.go`, `go.mod`, `go.sum`). It uses `ringpop-go` v0.1.0 and `tchannel-go` v1.22.1 directly; `go.mod` requires the Temporal module through `replace ../../../../../../..` so MVS selects exactly Temporal's versions. All 172 `go.sum` lines also appear in the root `go.sum`; `GOOS=wasip1 GOARCH=wasm go build -mod=readonly .` exits 0 offline.
- Scenario: `command/testdata/scenario/ringpop.json` — nodes a/b/c at 10.0.0.1-3:7946, six directed links, `c` partitioned from `a` and `b` at 10 s (symmetric) and healed at 30 s. Boot args: self, ring hosts, view 25 s, heal 30 s, deadline 90 s. Existing schema fields only. `observation_bytes` is 64 MiB, the `MaximumObservationBytes` cap.
- Installation config: `command/testdata/scenario/ringpop-config.json` (the scenario `config.json` with larger call, transcript, descriptor and NetworkV2 capacities; same clock, `ReadStepNanos=0`).
- `TestOperatorRingpopScenarioPreflight` normalizes the scenario (3 nodes, 6 links, 4 faults) and admits the config (exit 0, `evidence/fn224/scenario/preflight-test.log`).
- Operator guide: `CLI.md`, "Explore a typed cluster scenario".

Convergence witness: after the heal each node needs three consecutive one-second polls with all three members reachable, then prints `converged <node> after heal <d> checksum <n>` and runs to the shared 90 s deadline, so no peer stops before every node has observed the full ring. A missing convergence panics and is retained.

## Results

Identities: gomad `sha256:407263f9be00…`, helper `sha256:b25439140f0c…`, compiler `sha256:1675694ef690…`, model `sha256:73c1a4f3f64d…`, guest module `sha256:8249a1869116…`.

| Seed | Outcome | Wall | Partitioned view (a / b / c) | Converged | Replay |
| --- | --- | --- | --- | --- | --- |
| 7 | success | 456 s | {a,b} / {a,b} / {a,c} | all three 3 s after heal, checksum 990001539 | `reproduced=true result=success choice-replay=none`, 453 s |
| 19 | success | 448 s | {a,b} / {a,b} / {a,c} | all three 3 s after heal, checksum 990001539 | `reproduced=true result=success choice-replay=none`, 456 s |

Cluster replay checks the retained module, lifecycle, network evidence and per-incarnation choices; `choice-replay=none` is the documented cluster replay mode. The two seeds produced identical observable output; ringpop's member selection did not diverge between them. During the partition `c` still listed `a` as reachable (suspicion not yet escalated) while `a` and `b` had dropped `c`; this is ringpop SWIM behavior on the modeled network and is not asserted.

Seed 19's first attempt failed before execution with `gomad-wasmhost has unexpected link count 2`: a diagnostic `perf record` had hard-linked the shared helper into `~/.debug`. The link was removed and seed 19 rerun; the failure is infrastructure, not a scenario outcome.

## Findings filed

The first fixture used a 1 MiB `observation_bytes`. The cluster broker charges 64 B of World payload per brokered operation for the life of the run, and three gossiping nodes use about 940 operations per virtual second, so the budget ran out at 17.4 s virtual. The helper then misclassified the error as infrastructure and lost the terminal result: helper and Go both blocked until the wall watchdog fired, with empty streams. A 300 s timeline with 64 MiB was too slow for a 15 min watchdog (about 3-8 wall s per virtual s, growing). Both are gomad defects, filed as fn-235 (.1 fast classified failure, .2 superlinear World commitment cost). They are not ringpop or tchannel faults: the same three-member ring in one guest process ran 40 s virtual in 27 s wall. Whether the run-lifetime 64 B/operation charge against `observation_bytes` is intended is an owner question recorded in fn-235.

## Boundary (R4)

No diff under `tests/testcore`, `wasm/wasi`, `internal/model`, `wasm/toolchain` or `wasm/wasmhost` on this branch.

## Rerun

`/home/agent/wt/evidence/fn224/scenario/explore.sh <label> 7,19` (installation `evidence/fn224/scenario/gomad-wasm.json` from `ringpop-config.json`, cwd = guest module), then `gomad replay <artifact>`.
