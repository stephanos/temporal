---
satisfies: [R3, R4, R11]
---
# fn-107-gomad-finish-downstream-cell.1 Inject storage filesystems and deterministic capacity through volume and restore paths

## Description
Work in ../saas-temporal. Reuse VolumeOptions.WithFS and Pebble VFS across volume, engine, replica quarantine, checkpoint and restore paths. Add the minimal injectable capacity mechanism needed for in-memory storage; isolate host volume discovery under gomad. Preserve production defaults and existing comments. Write failing isolated tests before implementation. Files/Touches: walker/storage/**, walker/replication/checkpoint_store*.go. Quick: mise run test -t 3m ./walker/storage; mise run test -t 3m ./walker/replication; relevant lint. Storage and replication are shared providers for the later in-process profile.

## Acceptance
Injected storage owns the complete volume and checkpoint tree; write/read, reopen, lock lifecycle, isolation, missing-file and capacity failures are exercised. Native defaults keep behavior. Explicit configured capacity replaces host statfs in the injected path. Required tests and focused lint pass, with commands recorded.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
