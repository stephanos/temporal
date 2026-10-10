---
satisfies: [R2, R5, R7, R8]
---
# fn-152-gomad-runner-storage-on-one-append-only.7 Concatenate source-scoped shard logs after complete merge preflight

## Description
Implement R5 and the aggregate/portable-plan portions of R7/R8.

**Size:** M
**Files:** Core files (4-5): internal/campaign/merge.go, runner/campaign_merge.go, campaign_plan.go, campaign_shard_execution.go and merge tests. Wider touches: campaign_shard.go, campaign_merge capacity/ordinal fixtures and portable-plan publication/inspection helpers; no exploration shard support is added. Serialize this task after shared round schema stabilization because both own campaign record interpretation.
**Touches:** [tools/gomad3/runner/internal/campaign/merge*.go, tools/gomad3/runner/internal/campaign/resume_plan.go, tools/gomad3/runner/campaign_merge*.go, tools/gomad3/runner/campaign_plan*.go, tools/gomad3/runner/campaign_shard*.go]

### Approach

Keep the current seed-only on-failure=all shard protocol and plan identity checks. Introduce a current portable-plan storage lineage because the embedded journal contract changes. Validate every source shard/log and required artifact before creating an output directory or file. Concatenate validated records within aggregate source scopes so shard initialization/publication records cannot alter another shard's lifecycle. Preserve global ordinal partitioning, source identities, no duplicate shard or ordinal, missing ranges with --partial, content-evidence dedup and aggregate count/byte bounds. Record immutable source provenance and external artifact ownership rather than copying or sweeping shard files. The aggregate is one replayable campaign log with an aggregate envelope.

### Investigation targets

**Required** (read before coding):

- `tools/gomad3/runner/internal/campaign/merge.go:153`
- `tools/gomad3/runner/campaign_plan.go`
- `tools/gomad3/runner/campaign_shard_execution.go`
- `tools/gomad3/runner/campaign_merge.go`
- `tools/gomad3/runner/internal/campaign/merge_capacity_test.go`
- `.flow/memory/bug/integration/shard-merge-and-prepared-target-cache-2026-09-29.md`

### Verification

Focused command: go -C tools/gomad3 test -tags test_dep -count=1 ./runner/internal/campaign; run existing scripted shard/merge Runner tests with the exact retained selector.

Follow the parent spec's Delivery and verification section. Retain exact selectors and current command scope; do not substitute portable coverage for supported-host evidence. If deletion/fixture migration expands the surviving implementation beyond this cohesive owner, stop for conductor scope splitting before implementation.

## Acceptance
- [ ] R5: merge creates one campaign log through source-scoped concatenation and replay reports the combined shard totals with the unchanged seed-only shard protocol.
- [ ] R5: wrong plan/source identity, duplicate shard/ordinal, malformed/torn source, missing required evidence and invalid partial/bounds cases fail during full preflight before any output is created.
- [ ] R5/R7: aggregate provenance resolves external shard references and preserves source artifacts/diagnostics/target pools through merge, inspect and recovery; cleanup never sweeps them.
- [ ] R5: strict global ordinal coverage, --partial missing ranges, distinct evidence accounting and resource error classifications survive interruption and reopen.
- [ ] R8: new portable-plan lineage carries the replacement log limits; old plans are rejected explicitly with no legacy decoder, and current prepared-bundle revalidation still runs before target execution.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
