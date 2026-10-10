---
satisfies: [R7, R8, R10]
---
# fn-152-gomad-runner-storage-on-one-append-only.2 Gate artifact storage lineage and define owned reference cleanup

## Description
Implement the artifact portion of R7/R8 and its publication/reference seam.

**Size:** M
**Files:** Core files (4-5): artifact/publication.go, store.go, open.go and focused lineage/cleanup tests. Wider touches: target_pool.go only to expose/reuse existing ownership semantics, manifest fixture constructors and qualification artifact fixtures affected by the new required envelope.
**Touches:** [tools/gomad3/artifact/**, tools/gomad3/record/*_test.go, tools/gomad3/qualification/set/*_test.go]

### Approach

Introduce an artifact-owned storage-envelope discriminator that the publisher writes durably and OpenArtifact validates before decoding the existing execution manifest. Preserve record.SchemaVersion, RecordContract and runtime choice/World/I/O/simulation encodings unless a separately authorized owner changes them. Old/missing/unknown artifact envelope lineage gets an unsupported-format error without legacy decoding or migration. Keep existing artifact handles and payload APIs. Make publish-before-reference and reachability validation reusable for campaign and corpus callers, including diagnostics as execution-owned files. Cleanup validates every currently required reference first and only sweeps the caller's owned payload roots. Shared targets and source shard evidence require their existing owning pool/source authority; a campaign cannot sweep them.

### Investigation targets

**Required** (read before coding):

- `tools/gomad3/artifact/publication.go:33`
- `tools/gomad3/artifact/store.go:82`
- `tools/gomad3/artifact/open.go`
- `tools/gomad3/artifact/target_pool.go:103`
- `tools/gomad3/runner/internal/campaign/retained_evidence.go`
- `tools/gomad3/record/types.go:3`
- `tools/gomad3/qualification/set/execution.go:239`

### Verification

Focused command: go -C tools/gomad3 test -tags test_dep -count=1 ./artifact ./record.

Follow the parent spec's Delivery and verification section. Retain exact selectors and current command scope; do not substitute portable coverage for supported-host evidence. If deletion/fixture migration expands the surviving implementation beyond this cohesive owner, stop for conductor scope splitting before implementation.

## Acceptance
- [ ] R7: artifact payloads, manifest/envelope and containing publication directories are synced before a transaction can durably reference them; every publication/reference crash window has a failing control.
- [ ] R7: missing or corrupt required payloads/diagnostic sidecars report corruption before orphan deletion, and cleanup failures retain their error identity.
- [ ] R7: cleanup removes only unreferenced payloads in an exclusively owned root, preserves shared target pools and external source shards, and follows final live references after committed corpus eviction. Live durable pending-result receipts protect their payloads until an outcome transaction retires or adopts them; missing/corrupt pending evidence prevents cleanup.
- [ ] R8: old artifacts and malformed/unknown new envelope versions fail with the named unsupported format; no old artifact decoding/migration path remains reachable.
- [ ] R8/R10: current-build replay and immutable target/environment/toolchain/platform/profile bindings survive the lineage change with stdlib-only code.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
