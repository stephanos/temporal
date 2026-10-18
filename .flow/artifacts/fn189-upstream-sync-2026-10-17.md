# fn-189 upstream integration

## Current-state note — 2026-10-18

The owner has committed the integration as
`0077477965a5b766b95a667247a9f562c100941b`, with parents
`d8f936d2cea34f87acfad8f355702e59c9ed17e8` and upstream
`48a1e5ff025e55b3d4d0f44247423d4c279ff636`. The subsequent repair is
`bad3c401b5e7dbdf8bc8afaca3d08cad4da154fe`.
At inspected HEAD `95eed5c83a26eb1e93f2b1ba2f971e33bc1d51bc`,
`git merge-base --is-ancestor 48a1e5ff025e55b3d4d0f44247423d4c279ff636 HEAD`
exits 0 and `git rev-parse -q --verify MERGE_HEAD` exits 1: the upstream ancestor
is retained and no merge is in progress. This is ancestry evidence, not a new
review or qualification result.

The sections below preserve the earlier Darwin checkpoint, including its
then-pending merge, failures and pause boundary. Later repairs are recorded in
the [repair report](fn189-wasm-gate-fixes-2026-10-17.md); subsequent Linux/arm64
source checks are recorded in the
[source-gate checkpoint](fn189-linux-source-gates-2026-10-17.md) and
[pack reconciliation](fn189-pack-source-reconciliation-2026-10-17.md).
Their evidence is not interchangeable across candidates or hosts. Current
runtime/workload/replay/identity/resource qualification and remaining host gates
are still required; fn-189.1 remains in progress. Performance/resource-sensitive
work stays held, native execution stays deferred, and no publishing is authorized.

## Historical integration checkpoint

The working tree integrates Temporal upstream `main` at
`48a1e5ff025e55b3d4d0f44247423d4c279ff636` with Gomad
`d8f936d2cea34f87acfad8f355702e59c9ed17e8`. The owner requires leaving commits
to them. `MERGE_HEAD` retains the upstream parent and the merge remains pending.
This report does not claim that branch ancestry or fn-189 acceptance is complete.

## Conflict resolutions

The merge had 15 conflicting paths. The resolutions preserve Gomad's SQLite,
provider and deterministic matching seams while accepting upstream server changes.

| Paths | Resolution |
| --- | --- |
| `.github/.golangci.yml` | Keep precise Gomad exclusions and upstream naming rules. |
| `AGENTS.md` | Keep Gomad, Flow and model-routing instructions and the upstream review pointer. |
| `common/persistence/sql/sqlplugin/{mysql,postgresql,sqlite}/plugin.go`, `sqlplugin/plugin_names.go` | Use upstream driver-neutral plugin constants; retain SQLite platform selection. Migrate four Gomad callers. |
| `common/persistence/visibility/defs.go`, `visibility/store/tests/query_converter_test.go` | Accept upstream visibility behavior and canonical plugin constants. |
| `common/persistence/visibility/store/sql/query_converter_legacy_factory.go` | Accept upstream deletion with its removed legacy implementation. |
| `go.sum` | Combine entries and remove duplicates. |
| `service/matching/db.go` | Keep the explicit dirty flag alongside upstream `lastChange`, which upstream test setup still sets. |
| `tests/mixedbrain/go.mod`, `go.sum` | Accept upstream API/genproto upgrades and retain newer x/sys and x/text. |
| `tests/priority_fairness_test.go` | Keep activity readiness synchronization. |
| `tests/testcore/test_env.go` | Keep in-memory SQLite selection and upstream options. |

Upstream split `TestScheduleV1` into suites and moved workflow-pause cases into
`TestScheduleV1Suite`. It removed the legacy advanced-visibility suite. The
qualification generator now names the current suite and its generated inventory
comes from the actual upstream test declarations. Historical qualification
reasons remain historical; the regenerated inventory supplies no native pass.

## Verified checks

Host is darwin/arm64 with stock Go 1.27.1, Rust 1.94.1 and locked Wasmtime 47.0.3.
Raw logs live in `.flow/tmp/fn189/` and are local build evidence.

| Command or check | Exit | Evidence |
| --- | ---: | --- |
| `git fetch --no-prune https://github.com/temporalio/temporal.git main` | 0 | Exact upstream SHA above. The first fetch with an explicit remote ref failed because global prune removed that ref; fetching to FETCH_HEAD succeeded. |
| `make -C tools/gomad wasm-helper` | 0 | `helper-build.log` |
| `go build ./...` in `tools/gomad`, Go 1.27.1 | 0 | `build.log` |
| `go vet ./...` in `tools/gomad`, Go 1.27.1 | 0 | `vet.log` |
| Focused SQLite `TestDriverDSNPlatformSelection` and SQL `TestSQLQueryConverter` | 0 | Merge worker command results. |
| `go test -tags test_dep -count=1 -run '^TestBacklogManager.*$/^TestSyncState_UnloadsOnOwnershipLoss$' ./service/matching` | 0 | `matching-sync.log` |
| `make -C tools/gomad tests-qualification-generate` | 0 | `qualification-generate.log` |
| `make -C tools/gomad test-wasm-slim-refusals` with pinned stock compiler | 0 | `slim-refusals.log`; every named refusal passed. |
| `cargo +1.94.1 test --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked` after fixture repair | 0 | `rust-tests-repaired.log`; 418 passed, zero failed, two existing ignored benchmarks. |

The first Rust run and its serial repeat failed two cluster fixtures before
their intended execution phase. A one-page guest with active data needs 16,386
fuel even with an empty start function. The fixtures supplied 2,000 or 10,000.
The repaired budgets let the inspection guest reach its boundary and the fuel
guest reach its post-I/O exhaustion. Existing evidence comparisons remain and
the inspection fixture additionally requires successful recorded node results.
Production Rust sources and helper behavior are unchanged. The isolated probe
and red/green logs retain this diagnosis.

Initial `validate` exited 2 because an override named removed `TestScheduleV1`.
Its toolchain, boundary, script and compatibility constituents passed first.
The qualification inventory repair requires a final integrated rerun.
An overlapping lint invocation failed its process lock and requires a serialized
rerun; it supplies no lint result.

## Dependency reconciliation and repairs

The reviewed adapter regeneration upgraded OTel SDK 1.44 to 1.45 and Temporal
SDK 1.48 to 1.49. OTel retains its refusal rewrites; its source diff removes
canonical import comments and updates the semconv import. SDK's exact
`InterruptCh` anchor and replacement adopt upstream's `any` spelling.
[The adapter receipt](fn189-upstream-sync-2026-10-17/adapters.json) retains
approval digests, source review, failed atomic publication, retry and source checks.
The doctor and portable adapter preservation/regeneration checks passed.
Native runtime execution remains deferred.

Final pin-impact reports zero invalidated adapters, zero stale pins and zero
unknowns. It still exits 1 for 40 historical pack-rule invalidations.
[The baseline comparison](fn189-upstream-sync-2026-10-17/pack-baseline-comparison.json)
proves all 54 pack-rule rows match HEAD exactly, including the 40 invalidations
and 14 unaffected rows. The historical invalidations keep pin-impact red.

The architecture checker now canonicalizes its scan root so `/var` and
`/private/var` package metadata agree. Discovery retains that root, and the loader
uses the same normalization. Individual source symlinks still refuse. Portable
root-symlink regressions and the full architecture suite pass. Its existing
structural bound stays 700 lines (candidate 699); the host structural check took
3.570 seconds against its unchanged 5-second limit. Full suite time was 149.016
seconds, including both-platform inventory and vet.

The diagnostics identity fixture now reads the authenticated campaign journal
through `ReadResumePlan`. The old `.prepared/plan.json` authority was removed by
the journal migration. Four current snapshots separately cover portable and
selected native I/O identities; both original historical snapshots remain
unchanged. Reviewed differences are the existing portable envelope/storage,
I/O fixture or native implementation inventory, namespace-derived choice identity,
dependent hashes and the current capacity calculation. Both selected and default
tests passed two fresh repetitions. This changes no production format.

The merged functional-test compile found one unused `await` import after upstream
converted its uses to suite helpers. Removing that import restores
`go test -tags test_dep -run '^$' ./tests` (exit 0), without executing the full
functional suite. World race tests also passed (exit 0).

## Independent source review

Fresh-context `gpt-6.1-sol` at high reviewed the merge seams, exact adapters,
qualification inventory, canonical roots, fuel fixtures and both current identity
variants. It returned source-ready with no actionable findings. Reviewer and
writer are from the same GPT family. The review explicitly leaves runtime and
workload acceptance unclaimed. The only `git diff HEAD --check` findings are
two whitespace lines imported unchanged from upstream (`constants.go:3273` and
`testing.md:275`).

## Final checkpoint gates

[The machine-readable check receipt](fn189-upstream-sync-2026-10-17/checks.json)
records commands and exits. Logs remain under `.flow/tmp/fn189/`.

| Gate | Exit | Result |
| --- | ---: | --- |
| `make -C tools/gomad validate` | 0 | Final generated/toolchain/pack/inventory validation. |
| `make lint-code-fast GOLANGCI_LINT_BASE_REV=d8f936d2ce GOLANGCI_LINT_FIX=false` | 0 | Zero reported issues in 112 root and 52 default Gomad host packages. |
| `make -C tools/gomad test-harness` | 0 | Full default harness; repaired Runner and architecture suites pass. |
| `make -C tools/gomad world-test` | 0 | World, mailbox and process race suites. |
| Canonical `qualify-runtime`, using the already-built pinned release helper | 0 | 32 seeds, 25 fresh recordings each, 32 exact replays and aggregate pass. |
| `make -C tools/gomad test-wasm` | 2 | Full run retains the failures below; conformance and slim refusals pass. |

[The runtime receipt](fn189-upstream-sync-2026-10-17/runtime-qualification.json)
retains host/compiler/engine/module/helper/model identities, all 32 seed report
hashes and the rerun command. Pre/post verification found zero changed files in
the 6,014-file frozen manifest, SHA-256
`be8aa6e7eb54f4e6b3caa6c659263b88aa9df09b6c0e991f6716fd5f6d076c2a`.
Helper is `f543978ac9d2f913f5b044049b7f45a4553cf2fc1533c3a16bb1b3b23cfed1f6`;
model remains `sha256:9482b9c5e810464896c29d569c269d9b7e6d841872949ffa6cd20cfd5a7b0744`.
This qualifies the runtime-probe corpus only.

The initial full WASM run included the architecture fixture failure repaired
above; its focused rerun passes. Seven other top-level tests remain unresolved.
No baseline attribution or passing rerun is claimed for them.

| Test | Observed failure |
| --- | --- |
| `TestBuildSelectionChangesInvalidateCompilationCache` | Full provenance assertion does not find the optional provider closure. |
| `TestIntegratedCooperativeFailureArtifactDetachedReplay` | Cooperative canary campaign does not discover the required artifact. |
| `TestInstalledOperatorDiscoversInspectsAndReplaysWithoutBuildInputs` | Cooperative profile refuses incomplete runtime imports. Stock profile passes. |
| `TestInstalledOperatorTypedScenarioDetachedReplay` | Simulation finish reports unfinished helper runtimes. |
| `TestWarmPreparationAdmissionControls` | Expected admitted fixtures fail the old Linux/arm64 host binding. |
| `TestWarmPreparationComparisonControls` | Expected matching fixtures fail the old Linux/arm64 host binding. |
| `TestWarmPreparationSummaryControls` | Historical summary fixture fails the old Linux/arm64 host binding. |

The mandatory current-candidate Temporal/onebox/cluster/SQLite workload ladder
and resource budgets were not launched before the owner's stop request. Selected
native lint on the final adapter candidate remains unrun; portable adapter source
checks passed as recorded above. fn-189 R1 still needs the owner-created merge
commit, R2 retains these source-gate and inherited-pin disclosures, and R3 remains
open for the failed and unrun WASM gates. R4 has this source checkpoint; final
acceptance and changed workload identities remain unproved.

## Pause boundary

The owner requested finishing the in-flight batch and then stopping. Later
fn-187 fixture/module work and fn-185 profiling remain unstarted. fn-189.1 stays
in progress because acceptance remains incomplete. Native execution stays
deferred under fn-128/fn-149. No commit, push, PR or CI action occurred.

All commands and workers launched for this batch have finished. No worktree was
created. `git worktree list` also contains inherited prunable records for
`/home/agent/wt/fn184`, `fn184-base`, `fn185`, `fn187` and `fn189`; their directories
are absent on this host. This batch left those historical records and branches
untouched. Build caches, qualified runtime payloads and raw logs stay in their
existing ignored locations so the owner can inspect or resume the checkpoint.
