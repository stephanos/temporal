---
satisfies: [R1, R2, R3, R4]
---
# fn-147-migrate-elapsed-time-fields-to-protobuf.3 Migrate Testpilot duration consumers and read presence

## Description
**Note 2026-10-18 (one-language series):** Depends on fn-202 (Testpilot protos move under the framework) and then fn-203's single Case regeneration.

Migrate admission, execution, verification and Run recording to the checked Duration fields for the shared format 4.0 contract; public format activation remains fn-148.6. Apply the defined one-read/poll policy and scalar singleton presence cleanup.

**Size:** M
**Files:** Testpilot execution bounds and evidence policies, verification deadlines, Run elapsed recording, schema presence fields and generated consumers
**Touches:** [common/testing/testpilot/**, proto/internal/temporal/server/api/testpilot/v1/*.proto, api/testpilot/v1/*.go, tools/umpire/lower/**, tools/umpire/conformance/**]

### Approach
- Reuse Task 1's validation and Task 2's generated fields.
- Absent interval performs one read; present positive interval polls. Implement only current-format policy; reject retired formats and keep no old decoder.
- Replace scalar-only singleton oneofs with explicit presence, including an exact absent/zero/default test matrix.
- Preserve monotonic elapsed recording and reject precision or narrowing overflow before execution.
- Restore compilation across realization admission, all runtime consumers and test builders. Update focused hand-authored test builders here without activating public format emission/admission; managed generated trees and recorded companions remain fn-148.7's single regeneration responsibility.
- Add focused named Duration, polling, elapsed and deadline tests. Run only these intermediate tests; `TestGeneratedCasesAreCheckedIn` and other managed-artifact checks run after fn-148.7 regeneration.

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
- [ ] R1-R4 pass at all successor-format consumers before the single public 4.0 activation at fn-148.6.
- [ ] Absent, zero, positive, default and invalid Duration tests cover every field family.
- [ ] Singleton-oneof cleanup belongs exclusively to this task and preserves presence.
- [ ] Consumer packages compile and focused current-format Duration/replay tests pass; old formats reject explicitly. Managed-artifact gates wait for fn-148.7.
## Done summary
Blocked:
OWNER HOLD (2026-10-16): Umpire framework DSL is the top priority. This integration task is not admitted while fn-188.3 completes final native public-client verification and formal batch review. Preserve its existing source checkpoints, evidence, dependencies and unfinished acceptance obligations; this scheduling hold claims no implementation or proof completion. Completion of fn-188.3 does not automatically release this task: select later IR/exporter/schema/Go/backend work explicitly. Any model/temporal work remains independently excluded entirely, including disabled source contents, until the owner restores that scope.
## Evidence
- Commits:
- Tests:
- PRs:
