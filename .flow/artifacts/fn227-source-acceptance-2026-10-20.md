# fn-227 source verification — 2026-10-20

## Candidate and scope

- Source checkpoint: `04b6eb08cfd32a3831e1ea5796b57814969635d2` (`04b6eb08cf`).
- Conductor-reported integrated candidate: `04e06bc95f`.
- Implementation base: `ea45f94e59`.
- Worker branch: `fn-227-gomad-close-wasm-map-order-determinism`.
- Worker worktree: `/home/agent/wt/fn227-map-order`.
- R1–R3 source implementation and focused verification passed as described below.
- R4 remains pending: the conductor owns runtime, workload and retained-replay qualification on the frozen integrated candidate. No result below qualifies that integrated candidate or establishes a new determinism bound.
- No Flow task completion, push, PR or CI action was performed by this worker.

Evidence for these commands consists of the worker's command-tool outputs in the session. Separate raw log files were not retained; no raw-log paths are claimed here. The commands below ran before the source checkpoint, against the source later committed in it, except where explicitly identified as RED or a temporary mutation. The integrated SHA is supplied by the conductor, not a candidate tested in this worker worktree.

## Requirements and behavior

| Requirement | Implementation and evidence |
| --- | --- |
| R1 | `ClusterSession.Shutdown` collects live handles under the existing lock, then sorts by node ID and incarnation before stop frames and waits. `TestClusterShutdownControlOrder` checks exact emitted controls, incarnation ties and exclusion of completed nodes over 32 insertion variations. `TestClusterShutdownLiveNodesEvidenceOrder` checks World and network equality across 16 fresh same-seed helper sessions with two paused live nodes and varied map insertion order. |
| R2 | `ValidateStorageCrashSelection` collects and lexically sorts volume IDs separately for selection and context validation, preserving selection-before-context precedence. `TestStorageCrashValidationVolumeOrder` tests both validation and pre-sealing cluster-boot admission over 64 insertion variations per map, with multiple invalid volumes that produce different refusal dimensions without sorting. Existing refusal classification and messages are preserved. |
| R3 | The three escape rows become controlled and name behavioral tests. Historical ID/site pairs and the accepted-prefix anchor are preserved; one shutdown `sort.Slice live` identity is appended. The former escape-specific source assertion is replaced by controlled-row evidence checks. The generic escape-anchor test still validates any future escape rows but no longer demands that an escape exist. README caveats were replaced after behavioral tests passed. |
| R4 | Pending frozen integrated qualification. The changed Go model identity and fixture pins are recorded below; historical qualification is not relabeled. |

## RED and GREEN regression evidence

The following command exited **1** before production edits:

```sh
cd /home/agent/wt/fn227-map-order/tools/gomad && env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off GOMAXPROCS=2 /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go test -p=1 -tags test_dep -count=1 -timeout=3m -run '^(TestStorageCrashValidationVolumeOrder|TestClusterShutdownControlOrder)$' ./internal/backends/wasm/wasi
```

Observed failures were the intended defects: selection reported `duplicate operation` instead of `zero operation`; context reported `operation order` instead of `inventory`; stop frames appeared in varying orders, including incarnation 2 before incarnation 1 for node `a`.

After the minimal sorting changes, the following command exited **0**, with the WASI package reporting `0.007s`:

```sh
cd /home/agent/wt/fn227-map-order/tools/gomad && env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off GOMAXPROCS=2 /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go test -p=1 -tags test_dep -count=1 -timeout=3m -run '^(TestStorageCrashValidationVolumeOrder|TestClusterShutdownControlOrder|TestStorageCrashContextAdmission|TestClusterBootAdmitsStorageCrashSelectionBeforeSealing)$' ./internal/backends/wasm/wasi
```

The real-helper test initially exited **1** because it read `CompletedResult`, which intentionally exposes metrics rather than World/network evidence. The test was corrected to retrieve the already-finished result through `Finish`; this was a test-harness error, not a product defect. The following command then exited **0**, reporting the named test passed in `6.53s`:

```sh
cd /home/agent/wt/fn227-map-order/tools/gomad && env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off GOMAXPROCS=2 GOMAD_STOCK_GO=/Users/stephan/Workspace/skunkworks/gomad/temporal/tools/gomad/.toolchain/stock/2198cf19890232fc/go/bin/go GOMAD_WASM_TEST_HELPER=/home/agent/wt/evidence/qualification-final/early-onebox-ea45f94e59/gomad-wasmhost /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go test -p=1 -v -tags test_dep -count=1 -timeout=10m -run '^TestClusterShutdownLiveNodesEvidenceOrder$' ./internal/backends/wasm/wasi
```

### Mutation experiment and limit

The shutdown sort alone was temporarily removed, and the exact real-helper command above was rerun. It exited **0**, with the test reporting `3.44s`: these paused live guests have not accumulated pending execution effects, so their World/network evidence remains equal even when stop order changes. The sort was restored before final verification and commit.

Consequently, this real-helper test is evidence preservation coverage for the paused-live scenario, not an independent red witness for sorting. The complementary direct control-order regression supplies the observed RED/GREEN witness for the ordering defect. No active-network/pending-World shutdown mutation claim is made.

## Final focused and source gates

After identity pins were refreshed, the following final focused command exited **0**; the WASI package reported `4.856s`:

```sh
cd /home/agent/wt/fn227-map-order/tools/gomad && env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off GOMAXPROCS=2 GOMAD_STOCK_GO=/Users/stephan/Workspace/skunkworks/gomad/temporal/tools/gomad/.toolchain/stock/2198cf19890232fc/go/bin/go GOMAD_WASM_TEST_HELPER=/home/agent/wt/evidence/qualification-final/early-onebox-ea45f94e59/gomad-wasmhost /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go test -p=1 -tags test_dep -count=1 -timeout=10m -run '^TestDeterminism|^TestStorageCrashValidationVolumeOrder$|^TestStorageCrashContextAdmission$|^TestClusterBootAdmitsStorageCrashSelectionBeforeSealing$|^TestClusterShutdown|^TestTransportInputValidation$|^TestImplementationIdentity|^TestClusterTransport|^TestBrokerLifecycleCorpus' ./internal/backends/wasm/wasi
```

The following source inventory and architecture command exited **0**. Reported package times were determinism `4.794s`, archcheck `15.313s`, and wasm `2.811s`:

```sh
cd /home/agent/wt/fn227-map-order/tools/gomad && env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off GOMAXPROCS=2 /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go test -p=1 -tags test_dep -count=1 -timeout=10m ./internal/backends/wasm/determinism ./internal/archcheck ./internal/backends/wasm
```

Its earlier invocation exited **1** because `TestInventoryEscapeAnchorsResolve` demanded at least one remaining escape. The final change removes only that mandatory-existence assertion; anchor resolution for actual escape rows, inventory completeness, pinned evidence and identity-history enforcement remain active. The final passing source-gate invocation preceded identity-only corpus regeneration and the identity-test expectation update; those later changes were covered by the final focused command above.

The final required lint invocation exited **0**, reporting **0 issues**, and also ran the target's errortype vet check:

```sh
cd /home/agent/wt/fn227-map-order && env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off GOMAXPROCS=2 GOFLAGS=-p=1 PATH=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:/usr/local/bin:/usr/bin:/bin make lint-code-fast GOLANGCI_LINT_BASE_REV=ea45f94e59 GOLANGCI_LINT=/home/agent/wt/evidence/fn219/qualification-inputs/host-tools/golangci-lint-v2.13.0 ERRORTYPE=/home/agent/wt/evidence/fn219/qualification-inputs/host-tools/errortype GOLANGCI_LINT_FIX=false
```

`git -C /home/agent/wt/fn227-map-order diff --check` exited **0** before staging. After the implementation commit, `git -C /home/agent/wt/fn227-map-order status --short` exited **0** with no output.

## Identity rotation and corpus preservation

- Go compiled model before: `sha256:f9607ca9acb77614c5b247563bd586a5b1c5a3333bffead91392848fdcb14f29`.
- Go compiled model after: `sha256:73c1a4f3f64dedd4c18dad47849f6dcc8bc04d08e27067dedd096acde05e2e60`.
- `cluster.go` transport pin before: `7f4f65592b47ea9518c0df509e42ab3c60d43e8a147ee6c362219e08eafae7c8`.
- `cluster.go` transport pin after: `dc88360dd465bf7439a426fc2d1e5bf464ec7f6e4551015d4ecba91a76373585`.
- Exact reused helper executable: `/home/agent/wt/evidence/qualification-final/early-onebox-ea45f94e59/gomad-wasmhost`.
- `sha256sum` of that helper: `8956496039ed213d2c1d0e88266bb2f7a9f2c9916e299ae8b734387f2a6dbe5f` (command exit **0**).
- The helper Rust `src`, `Cargo.toml` and `Cargo.lock` have no changes from the implementation base. The helper tree does contain changed JSON fixtures below. No claim is made that a source digest covering the entire helper tree remains unchanged, or that the reused executable supplies qualification for changed model sources.

The pin-focused command below first exited **1**, identifying stale broker lifecycle and transport fixtures and the old compiled-model expectation:

```sh
cd /home/agent/wt/fn227-map-order/tools/gomad && env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off GOMAXPROCS=2 /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go test -p=1 -tags test_dep -count=1 -timeout=3m -run '^TestImplementationIdentity|^TestClusterTransport|^TestBrokerLifecycleCorpus' ./internal/backends/wasm/wasi
```

The existing generators refreshed 13 JSON fixtures: ten `wasmhost/tests/testdata/broker_lifecycle_cases/*.json`, two `wasi/testdata/cluster-transport-stage-conformance/*.json`, and one `wasmhost/tests/testdata/cluster_transport_codec/restarted-a-incarnation-two-rejects-old-prefix.json`. Generation exited **0**:

```sh
cd /home/agent/wt/fn227-map-order/tools/gomad && env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off GOMAXPROCS=2 GOMAD_BROKER_LIFECYCLE_CORPUS_REGENERATE=1 GOMAD_CLUSTER_TRANSPORT_STAGE_REGENERATE=1 GOMAD_CLUSTER_TRANSPORT_CODEC_REGENERATE=1 /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go test -p=1 -tags test_dep -count=1 -timeout=3m -run '^TestClusterTransport(Stage|Codec)CorpusIsCurrent$|^TestBrokerLifecycleCorpusIsCurrent$' ./internal/backends/wasm/wasi
```

The following comparison exited **0**, with no output. For every changed fixture, base and candidate JSON are identical after deleting only the top-level implementation identity fields. Inputs, expected outcomes, refusal classifications, evidence and all other JSON values were preserved:

```sh
cd /home/agent/wt/fn227-map-order && while IFS= read -r fixture; do diff -q <(git show "ea45f94e59:$fixture" | jq -S 'del(.implementation_sha256,.go_implementation_sha256)') <(jq -S 'del(.implementation_sha256,.go_implementation_sha256)' "$fixture") || exit 1; done < <(git diff --name-only -- tools/gomad/internal/backends/wasm/wasi/testdata tools/gomad/internal/backends/wasm/wasmhost/tests/testdata)
```

## Environment corrections

These unsuccessful setup attempts do not count as regression failures or passing gates:

- The first unit-test invocation used `/usr/bin/go` with the same RED selector and flags. It exited **1** because that binary is Go 1.26.0 while the module requires Go 1.27.1. All reported tests above used the explicit existing Go 1.27.1 distribution.
- The first lint invocation supplied `/Users/stephan/Workspace/skunkworks/gomad/temporal/.bin/golangci-lint-2.10.1`; it exited **2** after its tool installation/move failed. The next invocation supplied `/Users/stephan/Workspace/skunkworks/gomad/temporal/.bin/golangci-lint-v2.13.0`; it exited **2** with `Exec format error`. The final passing invocation uses the existing Linux host tools under the fn-219 qualification inputs, as shown above.

## Remaining acceptance

R4 is explicitly unfinished. The conductor must retain the frozen integrated candidate's actual qualification commands, exits and source/helper/runtime/module/engine identities. Source verification here does not replace the full runtime, workload, retained-replay or integrated Rust corpus gates. The worker worktree remains available for review fixes; the conductor owns integration, independent review, final acceptance and Flow completion.
