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

## Joined checkpoints and review

`9e0d09b5e6` checkpoints the retirement/cache source, actual generated choice/volume identities and milestone/task revival records. `5d4a6c88ef` integrates the network corpus. Starting the ordinary WASM suite also detected stale volume projections; `go generate ./volume` refreshed them before that suite was relaunched. The intermediate WASI model pin is `sha256:08855eac1e1ea29b93f16df3fd0c23a61c2b9e57a80d6fc13b5b9abf6a38e5ea`; final layout/oracle source changes will rotate it again.

Independent GPT-family source reviews are separate from final acceptance. The retirement/cache reviewer identified malformed cache bounds classified as capacity instead of named infrastructure corruption, and a prototype workflow reset witness that counted only a named fault. Both corrections pass focused tests and the independent re-review reports source readiness YES; fresh workflow execution remains a final guest gate. The corpus reviewer identified retained mutable domain pointers in revoke/restart scripts; snapshotting inputs and proving every serialized script replays are required before task completion. A passing currency test alone did not establish executable-script consistency.

Fresh root `TestNetworkCorpus|TestNetworkV2` tests passed on the joined pre-correction corpus (exit 0). Preserve that observation without treating it as evidence that the review-found script defect is absent. The ordinary WASM suite finished with exit 0 (session `17998`, log `/tmp/wasm-rotation-fast-20261013.log`): all ten WASM package suites pass. It checked pre-review-fix source and the matching copied helper, not the final layout/oracle candidate. Fresh root correction tests passed: the reset-witness positive/negative control, seven Rust cache unit tests and six cache integration tests. The manual throughput benchmark is ignored by ordinary tests.

Canonical affected lint first failed with exit 2 because the checkout's installed golangci-lint is a macOS executable. It then passed with exit 0 using the existing `.tmp/linuxbin/golangci-lint-v2.13.0` and `.tmp/linuxbin/errortype` command-line overrides, base a1f0be7e96 and fixes disabled (log `/tmp/wasm-rotation-lint-linux-20261013.log`). Both source sets report zero new issues.

Actual fn179 cluster execution exposed the sealed-plan guard refusing automatically derived soft-assertion oracles. The exact retained admission/test paths are now admitted for output/instance/detector-bound automatic evidence only; generic/mixed oracles still require their sealed user plan.

The corpus correction joined as `5a2fe5b544`: only the two revoke/restart inputs changed, with expected outcomes preserved. All 114 serialized scripts now replay against the owner; independent domain/request/payload alias controls failed on the original and pass after detachment. Fresh root owner/generator and exact `TestNetworkV2RustOwnerPinIsCurrent` tests pass, and independent re-review reports source readiness YES. An earlier mistyped drift-pin selector selected no tests and supplies no pin evidence. The correction's isolated canonical Linux lint also passes both source sets (log `/tmp/gomad-fn177-corpus.84vutL/lint-correction-isolated.log`). Root completed fn177.1 through `flowctl done` and verified status `done`; the Rust owner is still unimplemented, and layout currency/pin resynchronization remains required.

## Layout joined

Root fast-forwarded to `0d9dd53f23341f6ae885619fc7d95e212498c51c`, preserving the reviewed retirement/cache and corrected corpus commits in ancestry. `internal/` now has exactly archcheck, backends, evidence, host, model and runner. Cluster codecs depend downward on internal history DTOs; the server collection dependency and public host facades are retired. Runner exports are 75 against the measured original 151, satisfying the required halving. The module/tree name is still gomad3; fn166.11 remains pending.

The isolated layout worker reports fresh passing validation, all-platform architecture, affected lint, portable model/evidence/Runner tests, native source checks and relocated publication tests. These do not supply full guest qualification or supported native execution. Its WASI pin is `sha256:61b211fc51d484b6eac4010dfea441b24288508d394a212db0db71be3b653a4b`; its network pin is `sha256:9743e4c65faed7ba77a98a55ba182ca09b32ffb38fccad1106543f84b6ad282b`. The corpus was regenerated with corrected inputs and rerun after relocation. Root must verify the combined oracle/layout candidate and freeze final identities before qualification.

fn179's old-tree worker reports real single-process and two-instance oracle gates passing, including late assertions, two-seed signature deduplication, opt-out, retained replay and capacity/inconclusive handling. Preserve underlying capacity and incomplete-trace errors; automatic oracles are rederived from authenticated retained output. Its source has not yet joined the final layout and still needs final lint/commit, serialized integration and independent review. The old copied helper at `tools/gomad3/backends/wasm/wasmhost/target/release/gomad3-wasmhost` is temporary local state used by that live lane; move it aside only after the lane releases it.

Root's fresh relocated network/corpus tests and WASI implementation/drift pins pass. The full architecture run initially failed only `TestToolPackageLayout`: local old helper output and an empty old Runner fixture directory kept physical `backends` and `runner` roots present. After the oracle lane copied/pinned its own helper and released the path, root moved both local output trees recoverably under `tools/gomad3/.toolchain/pre-layout-output/`. The exact layout test then passed; the failed full run is retained at `/tmp/wasm-layout-arch-root-20261013.jsonl` and is not relabelled as a pass. Final combined-source architecture still reruns after the oracle joins.

R3 report-only observations for successful discarded executions now use the existing committed event/receipt owners, with omitted fields on normal runs and no extra artifact retention. The old-tree worker reports real choice-campaign and receipt-resume reporting controls passing; final source integration/review remains required.
