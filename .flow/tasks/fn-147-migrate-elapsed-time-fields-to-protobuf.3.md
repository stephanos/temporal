---
satisfies: [R1, R2, R3, R4]
---
# fn-147-migrate-elapsed-time-fields-to-protobuf.3 Migrate Testpilot duration consumers and read presence

## Description
Switch format 3.0 admission, execution, verification and Run recording to the checked Duration fields. Apply the defined one-read/poll policy and scalar singleton presence cleanup.

**Size:** M
**Files:** Testpilot execution bounds and evidence policies, verification deadlines, Run elapsed recording, schema presence fields and generated consumers
**Touches:** [common/testing/testpilot/**, proto/internal/temporal/server/api/testpilot/v1/*.proto, api/testpilot/v1/*.go, tools/umpire/lower/**, tools/umpire/conformance/**]

### Approach
- Reuse Task 1's validation and Task 2's generated fields.
- Absent interval performs one read; present positive interval polls. Implement only current-format policy; reject retired formats and keep no old decoder.
- Replace scalar-only singleton oneofs with explicit presence, including an exact absent/zero/default test matrix.
- Preserve monotonic elapsed recording and reject precision or narrowing overflow before execution.
- Restore compilation across realization admission, all runtime consumers and test builders. Convert hand-authored current-format fixtures and Case/Run pairs here; managed generated trees remain Task 4's responsibility.
- Add focused named Duration, polling, elapsed and deadline tests. Run only these intermediate tests; `TestGeneratedCasesAreCheckedIn` and other managed-artifact checks run after Task 4 regeneration.

### Investigation targets
**Required** (read before coding):
- `common/testing/testpilot/internal/execution/dataflow.go:245-329` - bounds and hints
- `common/testing/testpilot/internal/execution/evidence.go:292-350` - read policy
- `proto/internal/temporal/server/api/testpilot/v1/contract.proto:118-130` - deadline clocks
- `proto/internal/temporal/server/api/testpilot/v1/run.proto:24-38` - elapsed coordinate
- `common/testing/testpilot/recordedrun/recordedrun.go:40-90` - current Case and Run pairing

### Quick commands

```bash
go test -tags test_dep -run 'Test.*(Duration|Polling|Elapsed|Deadline)' ./common/testing/testpilot/... ./tools/umpire/realization/... ./tools/umpire/conformance/... ./tools/umpire/lower/...
```
## Acceptance
- [ ] R1-R4 pass at all format 3.0 consumers.
- [ ] Absent, zero, positive, default and invalid Duration tests cover every field family.
- [ ] Singleton-oneof cleanup belongs exclusively to this task and preserves presence.
- [ ] Consumer packages compile and focused current-format Duration/replay tests pass; old formats reject explicitly. Managed-artifact gates wait for Task 4.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
