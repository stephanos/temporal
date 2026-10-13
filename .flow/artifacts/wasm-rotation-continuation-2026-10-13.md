# WASM rotation continuation — 2026-10-13

## Resumed source

The continuation starts from `a1f0be7e96` on `gomad`, preserving the staged planning changes and existing isolated source worktrees. The executing host is Linux ARM64 with stock Go 1.27.1 available; it cannot supply Darwin performance acceptance or patched native execution.

The following fresh checks passed before the final layout joins:

| Command (from tools/gomad3; stock Go 1.27.1, tags test_dep, count=1) | Exit |
| --- | --- |
| `go test ./internal/strictjson ./internal/hostfs ./world/... ./simulation/network/... ./runner/internal/campaign/...` | 0 |
| `GOMAD3_ARCH_ALL_PLATFORMS=1 go test ./internal/archcheck ./internal/puritycheck` | 0 |

These are source checks for the starting candidate, not final WASM qualification. `make validate` found stale choice-wire generated output; `gomadtool protocol-generate` refreshed the host and overlay copies successfully. A fresh `make -C tools/gomad3 validate` then passed (exit 0), including patch/overlay, script ownership, compatibility packs and qualification manifests.

The focused cache-publication, ordinary-provenance and retained-bundle tests subsequently passed in `./backends/wasm/backend ./backends/wasm/simulation` (exit 0) with explicit `GOMAD3_STOCK_GO=.../.toolchain/stock/878c9b2935e2c9e6/go/bin/go`. An initial invocation omitted the required stock compiler and was refused; that invocation was not a pass. These results precede the layout, oracle and final helper/model identity freeze.

## Integration lanes

- fn166.8 retires WASM network v1 and choice trace v1 while retaining diagnostic-profile, NetworkV2 and broker networking.
- fn173.2–.4 provisionally implement cache-hit module-read avoidance, miss-only digest verification, Go per-run module-hash removal and the CRC cache envelope. The fn173.1 Darwin prerequisite and final Darwin exit/throughput/preparation measurements remain open.
- fn166.6/.7/.10 implement the final six internal groups, facade retirement and lower cluster codec ownership in `/tmp/gomad-final-layout-20261013`.
- fn177.1 resumes the existing operation-corpus worktree `/tmp/gomad-fn177-corpus.84vutL/worktree` against post-fn154 source.
- fn179.1 resumes the existing soft-assertion worktree `/tmp/gomad-fn179-oracle.pq7aav/worktree`; exact provider provenance and replay-option propagation is admitted after the fn153 corrections joined.

Root integrates helper/cache, oracle and corpus changes before the layout candidate and then freezes all generated and implementation identities. Historical fn151/fn156 evidence remains historical. Independent review, affected lint, final source/WASM gates, explicit new native requirements and performance measurements remain mandatory; none of these lanes is complete on the strength of this checkpoint.
