---
satisfies: [R3, R4, R11]
---
# fn-107-gomad-finish-downstream-cell.1 Inject storage filesystems and deterministic capacity through volume and restore paths

## Description
Work in ../downstream. Reuse VolumeOptions.WithFS and Pebble VFS across volume, engine, replica quarantine, checkpoint and restore paths. Add the minimal injectable capacity mechanism needed for in-memory storage; isolate host volume discovery under gomad. Preserve production defaults and existing comments. Write failing isolated tests before implementation. Files/Touches: storage/storage/**, storage/replication/checkpoint_store*.go. Quick: mise run test -t 3m ./storage/storage; mise run test -t 3m ./storage/replication; relevant lint. Storage and replication are shared providers for the later in-process profile.

## Acceptance
Injected storage owns the complete volume and checkpoint tree; write/read, reopen, lock lifecycle, isolation, missing-file and capacity failures are exercised. Native defaults keep behavior. Explicit configured capacity replaces host statfs in the injected path. Required tests and focused lint pass, with commands recorded.

## Done summary
Injected filesystem ownership now covers volumes, engines, quarantine, checkpoint archive/restore and SST ingest. Validated configured capacity replaces restore statfs; paired discovery source rejects host discovery under gomad. Native restore preserves exclusive creation and permissions, and existing comments remain intact.

Regression tests caught corrupt-engine ID lookup, restore checksum ordering under memory VFS writes and capacity wrapper Unwrap behavior. Focused red/green evidence, native storage/replication suites, gomad-tag tests and lint pass. Independent three-axis review found one wrapper defect; the resumed review is SHIP with no surviving findings. Review receipt: .flow/artifacts/fn-107-gomad-finish-downstream-cell/task1-review.json.

This establishes the isolated storage contract. Task .2 owns propagation through actual localcluster/bimmer/IU-store consumers. No cell qualification is claimed. No staging or commit; commit ownership remains with the user.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: mise run test -t 3m ./storage/storage ./storage/replication (292 tests, two existing skips; /private/tmp/fn107-storage-replication-native.log), mise run test --tags test_dep,gomad -t 3m -r "^TestInjectedFS_|^TestCapacityFS_|^TestDiscoverVolumes_RequiresConfigurationUnderGomad$|^TestCheckpointBlobStoreSuite/TestInjectedFSUploadDownload$" ./storage/storage ./storage/replication (11 tests; /private/tmp/fn107-storage-replication-gomad-tag.log), mise run test -t 3m -r "^TestCapacityFS_Root$" ./storage/storage (verified red then green; /private/tmp/fn107-capacity-root-red.log and -green.log), mise run test -t 3m ./storage/storage (72 tests, two existing skips after review fix; /private/tmp/fn107-capacity-root-storage-quick.log), mise run test -t 3m ./storage/replication (223 tests after review fix; /private/tmp/fn107-post-review-replication.log), mise exec -- golangci-lint run --timeout 2m ./storage/... ./replication/... (storage cwd; zero issues; /private/tmp/fn107-capacity-root-lint.log), git diff --check, Flow Codex three-axis review and resumed review: SHIP
- PRs: