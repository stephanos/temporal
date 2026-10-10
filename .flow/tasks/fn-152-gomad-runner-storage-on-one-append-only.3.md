---
satisfies: [R1, R2, R7, R8]
---
# fn-152-gomad-runner-storage-on-one-append-only.3 Replace campaign lifecycle and recovery files with replayed log state

## Description
Implement R2's campaign storage owner and the campaign portions of R1/R7/R8. Keep Runner policy integration in the next task.

**Size:** M
**Files:** Core rewrites (4-5): campaign_journal.go, recovery.go, resume_plan.go, retained_evidence.go and one campaign-log replay owner. Wider touches are substantial but deletion/fixture focused: lifecycle.go, open_campaign.go, resume_journal.go, segmented_journal.go, filesystem.go and obsolete lifecycle/segment/recovery fixtures. Their surviving semantic validators must move into the new owner before deletion. This task is M at the upper end and cannot include Runner policy or round migration.
**Touches:** [tools/gomad3/runner/internal/campaign/**]

### Approach

Convert the campaign storage owner while preserving its public operation shape for Runner callers. Store initialization/frozen plan, prepared target references, lifecycle, seed outcomes, capacity metadata and publication as campaign log records. Replay semantic validators into a detached state and validate artifacts before repairing or cleaning owned files. Keep existing semantic execution/artifact/frontier capacities. SegmentRecords and MaximumSegments are retired storage-only compatibility values with documented disposition, never hidden limits on new round transactions. SegmentBytes preserves the per-execution payload bound or a read-batch bound, while a whole-round frame derives its checked bound from the existing allowed component envelope rather than inheriting the old segment cap. No new aggregate/round cap is introduced. Retain public detached inspection fields needed by the external caller fixture and mark their retired segment meanings explicitly in current reports/docs. No segment-limit CLI flag is registered in the inspected source; existing command grammar remains unchanged. Keep prepared targets and ephemeral execution work as payloads; they cannot become alternate state authorities. Recover under the writer lock, use log tail repair, delete only owned unreachable files and remain idempotent. Preserve changed-input refusal, interrupted-preparation refusal, published authority and recovery classifications. Reuse existing pure ordinal/identity/retained-evidence validation; remove old state parsing.

Expose the validated typed transaction and live-reference seam consumed by the subsequent admission/receipt owner. Its durable pending receipts contribute live payload references before recovery cleanup; the later owner defines that state machine rather than duplicating it in this lifecycle conversion. No temporary payload file is a second campaign metadata authority.

### Investigation targets

**Required** (read before coding):

- `tools/gomad3/runner/internal/campaign/campaign_journal.go:153`
- `tools/gomad3/runner/internal/campaign/lifecycle.go:53`
- `tools/gomad3/runner/internal/campaign/recovery.go:31`
- `tools/gomad3/runner/internal/campaign/resume_journal.go:129`
- `tools/gomad3/runner/internal/campaign/resume_plan.go:99`
- `tools/gomad3/runner/internal/campaign/segmented_journal.go:87`
- `tools/gomad3/runner/internal/campaign/retained_evidence.go`

### Verification

Focused command: go -C tools/gomad3 test -tags test_dep -count=1 ./runner/internal/campaign.

Follow the parent spec's Delivery and verification section. Retain exact selectors and current command scope; do not substitute portable coverage for supported-host evidence. If deletion/fixture migration expands the surviving implementation beyond this cohesive owner, stop for conductor scope splitting before implementation.

## Acceptance
- [ ] R2: campaign initialization, prepared plan, lifecycle, outcomes and final publication have one log authority; campaign JSON, segment/index, staging and resume-archive implementations are deleted with their storage-only fixtures.
- [ ] R2/R8: old campaign layouts and unknown log lineage fail before mutation, while current recorded plan/target/environment/profile identities and existing input-error precedence are retained.
- [ ] R7: replay validates committed retained artifacts, prepared inputs, diagnostics and live pending-receipt references before owned-root cleanup; shared roots remain untouched. The admission/receipt owner consumes one validated transaction/reachability seam without adding metadata side files.
- [ ] R1/R2: lock/repair/publication crash controls and repeated recovery cover every lifecycle transition, including acknowledged publication with interrupted cleanup.
- [ ] R2: existing execution/artifact/frontier/partial-attempt capacities retain their semantics; checked frame bounds preserve the allowed component envelope. Retired SegmentRecords/MaximumSegments/public inspection fields have tested and documented compatibility disposition; SegmentBytes cannot silently become a new aggregate/whole-round cap.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
