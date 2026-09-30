Injected filesystem ownership now covers volumes, engines, quarantine, checkpoint archive/restore and SST ingest. Validated configured capacity replaces restore statfs; paired discovery source rejects host discovery under gomad. Native restore preserves exclusive creation and permissions, and existing comments remain intact.

Regression tests caught corrupt-engine ID lookup, restore checksum ordering under memory VFS writes and capacity wrapper Unwrap behavior. Focused red/green evidence, native storage/replication suites, gomad-tag tests and lint pass. Independent three-axis review found one wrapper defect; the resumed review is SHIP with no surviving findings. Review receipt: .flow/artifacts/fn-107-gomad-finish-downstream-cell/task1-review.json.

This establishes the isolated storage contract. Task .2 owns propagation through actual localcluster/bimmer/IU-store consumers. No cell qualification is claimed. No staging or commit; commit ownership remains with the user.
