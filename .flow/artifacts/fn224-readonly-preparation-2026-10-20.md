# Ringpop preparation — source observations, not execution evidence

Read-only scout on `fe1f3d81ac`; no source edits, builds, guest runs or task completion. fn-224 still follows the fn-221 sweep. Reconcile these source facts with its spec before implementation; no model-admission change is authorized here.

## Unstub surfaces

Change the three selectors together: `temporal/membership.go` currently selects `!gomad && !gomad_wasm_slim`, while `membership_gomad.go` and `membership_gomad_test.go` select `gomad || gomad_wasm_slim`. Real implementation must select `!gomad`, with the stub/test retaining `gomad`. Preserve cloud/SQL exclusions and `backend.TemporalGuestBuildTags()`.

The `test-wasm-slim-refusals` Make target explicitly requires the membership stub test under slim tags, including missing-test detection. `TestSlimGuestClosureCutsOnlyOptionalProviderSeams` currently asserts ringpop's dependency edge/module are absent. Both gates need corresponding positive-retention assertions; do not simply remove their coverage. Update exclusion descriptions in the provider and operator documentation. Capture matched current before/after module sizes and hashes, separately from fn-182's historical slim one-box 306,428,662 B/full 338,040,129 B and multi-instance slim 209,124,931 B.

## Actual test inventory

The stated eleven tests are two registration wrappers plus nine executable suite methods. Preserve all functions' coverage through these exact leaf selections, over at least three seeds:

```text
^TestRingpopSuite$/^TestHostsMode$
^TestRingpopSuite$/^TestInvalidBroadcastAddress$
^TestRingpopSuite$/^TestRingpopMutualTLS$
^TestRingpopSuite$/^TestRingpopServerTLS$
^TestRingpopSuite$/^TestRingpopInvalidTLS$
^TestRpoSuite$/^TestMonitor$
^TestRpoSuite$/^TestScheduledUpdates$
^TestRpoSuite$/^TestCompareMembers$
^TestRpoSuite$/^TestCompareMembersWithDraining$
```

The spec's `^TestRingpopMonitor` quick selection matches no top-level test and cannot prove execution. `TestRpoSuite/TestCompareMembers` is a smaller early proof than factory tests, whose `SetupTest` always generates certificates. Follow it with a real network test. Require actual named verdicts; replay every pass. Retain build/infrastructure/replay-divergence/not-run distinctions from the current sweep vocabulary, not only the original five-class shorthand.

## Networking and scenario constraints

Factory TLS tests use fixed ports 7600/7601; monitor helpers use ephemeral loopback ports. NetworkV2 admission in `wasi/environment.go`, `wasi/cluster.go` and `cluster/operator.go` requires `ReadStepNanos == 0`. The spec's proposed read-step stall workaround is not available for these networked runs. Diagnose named harness sites and preserve model/config admission. If the necessary harness fix lies in production-named `test_cluster.go`, reconcile that write surface with task .2 before changing it.

Reuse the existing typed scenario schema and `command/testdata/scenario/{config,network,durable}.json`, its guest, and existing preflight/detached-replay tests. Three fully connected nodes need six directed links; four need twelve. Existing `network_faults` express partition/heal. Output convergence stops use existing `OutputStops` markers (base64 bytes in JSON); every marker must be consumed and every node must have a terminal action. Do not accept pre-heal membership as the final convergence witness or stop peers before all required observations.

Run the new nested-module guest from `tools/gomad` as `go-run ./internal/backends/wasm/testdata/ringpop`, not from the Temporal root across the module boundary. Standalone tests use explicit choice/success retention and watchdog bounds; scenarios reject `--choices`/`--diagnostics` and report `choice-replay=none`, while retaining lifecycle, model and per-incarnation replay evidence. Keep `tests/testcore`, `wasi.Config`, helper protocol, network and clock models byte-identical under fn-224, with baseline/final restricted diffs proving that boundary.
