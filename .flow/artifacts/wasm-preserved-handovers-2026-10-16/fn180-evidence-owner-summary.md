# fn-180 retained inspection owner handover

Status: in_progress; stopped at the user's request. Workspace: /tmp/gomad-fn180-evidence, branch fn180-evidence. Worktree is clean. No tests, compilers, helper processes, or writers remain active from this worker. No review or Flow completion was performed.

Tier: IMPLEMENTER gpt-6.1-sol medium (dispatch; actual execution model not independently exposed).

Commits, oldest first:

- 578e35cd73418487ac2af14051c81a53c2932936
- a457f7c92a7f347053468dc2f5762b04279069c4
- 2a19f0ebce4f5425f084ab295152ba5640a0ae5d
- 8d03a6ab2a7f34276926b3d84444712f59f1fcde
- 012d434b06c2c11c69a469313cb3e2a07d464de1

## Source delivered

Model owner: `internal/model/cluster/{wasm,wasm_inspection,implementation,record}.go` now owns bounded versioned retained metadata, compulsory 128-byte runtime witnesses with individual seeded counters, closed registration descriptors, deep cloning, strict decoding, exact shared coordinates, and prefix comparison across executed incarnations. Cuts include explicit `choice_bytes`, pending folded-clock material, `network_state`, storage/recovery state, output, and World/lifecycle streams. Complete metadata cross-binds canonical streams and outputs; incomplete metadata remains retainable but cannot authorize inspection. The source identity inventory includes the new owner.

Backend owner: `internal/backends/wasm/cluster/{inspection,replay_inspection_search,provider,retained,implementation}.go` owns exact choice/trace mapping, retained preflight, metadata retention, and bounded scan/conditional-bisect orchestration. Search moved here from Runner after the architecture gate caught forbidden imports; its coverage moved with it. The evaluator contract requires a fresh exact replay and cleanup under the supplied whole-search context. Global bound exhaustion returns incomplete; local predicate timeout remains inconclusive.

Neutral/Runner/CLI: `internal/backends/inspection.go`, `internal/runner/replay_operation.go`, `cmd/gomad/internal/cli/{cli,replay_inspection}.go`, and the WASM command hook route optional inspection requests through `InspectionProvider.Inspect`. Requests carry stops, target, dump/probe selections, bounds and search options. Inspection reports carry requested/resolved coordinates and use dedicated JSON status rather than reproduction. Provider preflight errors map to CLI exit 2. Ordinary replay paths retain their prior behavior when inspection is absent.

Additive raw metadata fields were added to `wasi.Result` and `wasi.ClusterResult`. No Rust helper, runtime handoff, or guest library implementation was written by this worker.

## Verification observed

baseline: none (conductor batch policy; no per-task Quick command baseline required).

All commands below ran from `/tmp/gomad-fn180-evidence/tools/gomad`, using `/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go`.

- `go test -tags test_dep ./internal/model/cluster -count=1`: exit 0; `/tmp/fn180-evidence-model.log`.
- `go test -tags test_dep ./internal/model/cluster ./internal/backends/wasm/cluster -run '^(TestWASMInspection|TestInspection|TestRetainedCanonical)' -count=1`: exit 0; `/tmp/fn180-evidence-owners.log`.
- `go test -tags test_dep ./internal/model/cluster ./internal/archcheck -count=1`: initial exit 1 for forbidden owner edges, then exit 0 after the ownership refactor; `/tmp/fn180-evidence-model-architecture.log` and `/tmp/fn180-evidence-model-architecture-final.log`. The successful architecture run took 74.649 seconds and completed before the shared measurement stand-down.
- Focused red reproductions were observed before implementation; logs `/tmp/fn180-evidence-red*.log` and `/tmp/fn180-search-red.log` preserve numeric `suite_rc=1` observations.

These observations do not qualify final HEAD. The final guards requiring nonempty shared cuts and partial network/storage state, the global-search-bound adjustment, neutral/Runner/CLI plumbing, and newly queued Runner/parser/report tests have not been compiled or run. The measurement stand-down and subsequent user stop explicitly prohibited further tests/builds. No test, gate, or architecture rule was weakened.

## Remaining integration

1. Actual `wasm/backend.Provider.Inspect` is NOT implemented. Current plumbing refuses providers without that optional interface. No provider implementation was started after the stop request.
2. The standalone modeled-evidence v2 envelope, raw helper transport, Rust metadata producer, runtime handoff and library are owned by the parent/other workers. Their final source is not integrated into this worktree.
3. `ValidateReplay` must admit retained cluster artifacts before Runner dispatches inspection. Standalone execution should reuse existing `wasi.Run` request construction and exact tapes. Cluster execution must stop its existing lifecycle driver at the dedicated terminal, avoid normal Finish/oracles/full-suffix checks, freeze all peers and hard-cancel/reap fresh stores without model advancement.
4. The parent added `wasi.Result.DebugOutput` outside identity after this worktree's raw metadata seam. Add `bytes.Clone(result.DebugOutput)` to `Provider.CompletedResults` when that field is integrated. No clone line referring to an absent field was committed here.
5. Build the helper control JSON from authenticated retained cuts: target, decision, expected boundary, World/lifecycle prefix hashes, six canonical stream prefix hashes aligned with cuts, exact registry prefixes/closure, probe IDs and dump/check commands. A concrete standalone/cluster inspection-plan adapter is still needed; it was discussed but not written.
6. Run the queued host unit tests and architecture checks after integration, then the required current-candidate Rust/helper/WASM/runtime/pprof/Temporal gates and independent review. No current-candidate execution qualification is claimed.

stage: impl-review - skipped(policy: conductor owns review; user stopped before dispatch)
