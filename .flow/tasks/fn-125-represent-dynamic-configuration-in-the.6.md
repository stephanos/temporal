---
satisfies: [R7, R9]
---
# fn-125-represent-dynamic-configuration-in-the.6 Declare API preconditions, derive each Case's required settings with origin, and build clusters from them

## Description
**Partly absorbed by fn-144** (2026-10-06): the required-settings part of this task (required settings with relation and origin, the union of origins, and the refusal of conflicting origins) is delivered by fn-144 R17. The API preconditions, the harness and Q1 stay here.

Implements R7, the harness half of R9, and the owner's Q1 decision. A typed method, a performed command kind or a Testpilot fault kind declares its preconditions in the kit, beside fn-118's behavior hints; lowering derives each Case's `Program.required_settings` from what its Program uses, with origin and source; the live harness builds each cluster from exactly those settings.

**Cross-spec entry gate:** fn-118.2 done (`ApiBehavior` targets). Not concurrent with fn-124.8. Depends on task 5. This task unblocks fn-121.3's green CI evidence (the ShutdownWorker race); tell the conductor when it lands.

**Size:** L
**Files:** `model/temporal/realize/**` (preconditions on fn-118's `ApiBehavior`); `model/irgen/**`; `ir.proto` (`ApiBehavior` preconditions: target method, cause kind or fault kind plus repeated `RequiredSetting`; reserve `Realization.required_settings = 15`); `proto/internal/temporal/server/api/testpilot/v1/program.proto` (`RequiredSetting` gains `relation = 3`, `origin = 4`, `SourceLocation source = 5`); `tools/umpire/lower/{realization,lower}.go`, `lower/internal/producer/**`; `common/testing/testpilot/internal/execution/prepare.go`; `model/temporal/features/nexus/standalone/Realization.scala` (its realization-level setting moves to methods); `tests/testpilot_generated_test.go`; regenerated `model/cases/**`.
**Touches:** [model/temporal/realize/**, model/temporal/features/**/Realization.scala, model/irgen/**, model/ir/**, model/cases/**, proto/internal/temporal/server/api/umpire/v1/**, proto/internal/temporal/server/api/testpilot/v1/**, api/umpire/v1/**, api/testpilot/v1/**, tools/umpire/lower/**, tools/umpire/model/**, common/testing/testpilot/**, tests/testpilot_generated_test.go, tests/testcore/testpilot/**]

### Approach
- Declare: `activity.enableStandalone` on the standalone activity methods; `activity.enableStandalone` plus the operator-commands flag on Pause and Unpause (`chasm/lib/activity/frontend.go:460-555`); `history.enableChasm` and `nexusoperation.enableStandalone` on the standalone Nexus methods (`N/frontend.go:323`); `frontend.enableCancelWorkerPollsOnShutdown=true` on the `workerStop` fault kind. Confirm whether standalone activity needs `history.enableChasm` (the harness sets it; the research says the frontend does not read it) and declare it only if a server path does.
- Owner Q1 (decided 2026-10-04): declare `frontend.enableMatchingFanOutForPollCancellation=false` on the `workerStop` fault kind, `because` citing the ShutdownWorker early return (`service/matching/matching_engine.go:1311-1324`, upstream #9424), to be removed when the server fix lands; never a Profile value. Draft the upstream report (repro: `MILESTONES.md:92-96`, fn-121.1's diagnosis, 20/20 SATISFIED with the flag off) in the done summary for the owner to post.
- Lowering: required settings = union of the preconditions of every method, performed command kind and fault kind the Program uses, each with relation, origin (`PauseActivityExecution`, `workerStop`) and source. Two origins requiring one key differently are refused, naming both; nothing is resolved by "last wins". A key the kit does not declare is refused at its line. Retire the realization-level `requiredSettings` (fn-122.4) and reserve field 15.
- Preparation keeps fn-122.4's equality check (`prepare.go:95-118`); its refusal names the origin and source.
- Harness (R9, first half): build each cluster from exactly the Case's required settings through a lookup derived from the registry (no hand table); delete `requiredSettingKinds` and the blanket `activity.Enabled` / operator-commands / `EnableChasm` settings (`testpilot_generated_test.go:168-172`). A required key the registry lacks fails the Case, naming it. The switch stays for workflow-Nexus Cases until task 7 encodes it, so those Cases do not fall back to server defaults.
- Record the Case-byte delta: required settings with origins on activity, pause, worker-stop and standalone Nexus Cases. Regenerate Cases and receipts; nothing else moves.

### Investigation targets
**Required:**
- `tools/umpire/lower/realization.go:100-120`; `common/testing/testpilot/internal/execution/prepare.go:90-120`
- `model/temporal/features/nexus/standalone/Realization.scala` (fn-122.4's `RequiredSetting`); `model/temporal/realize/Realize.scala:31`, `Kit.scala:138,151`
- fn-118.2's `ApiBehavior` IR and Scala surface
- `tests/testpilot_generated_test.go:160-230`; `MILESTONES.md:90-97`
**Optional:**
- `.flow/tasks/fn-121-shard-generated-cases-per-case-in-ci.1.md` (race diagnosis); `.plans/DYNAMIC_CONFIG.md` section 1 (task queue / worker)

### Quick commands
```bash
make protoc && make umpire-gen-model && git diff --stat model/ir model/cases
make umpire-check-cases
go test -count=1 -tags test_dep -p 2 ./tools/umpire/... ./common/testing/testpilot/...
go test -count=1 -tags 'test_dep integration' ./tests -run TestTestpilotGeneratedCases
```

### Execution constraints
- Case-byte changes are only the recorded required-settings delta; Programs, expectations and Contracts are otherwise unchanged.
- Workflow-Nexus Cases keep the (fixed) switch until task 7.
## Acceptance
- [ ] Methods, performed command kinds and fault kinds declare preconditions in the kit: standalone activity, Pause/Unpause operator commands, standalone Nexus (`history.enableChasm`, `nexusoperation.enableStandalone`), and on `workerStop` both `enableCancelWorkerPollsOnShutdown=true` and `enableMatchingFanOutForPollCancellation=false` with a reason citing upstream #9424.
- [ ] Lowering derives each Case's required settings from its Program's uses, with relation, origin and source; two origins requiring one key differently are refused naming both; an undeclared key is refused at its line.
- [ ] The realization-level `requiredSettings` is gone and field 15 is reserved.
- [ ] The harness builds clusters from exactly the required settings via a registry-derived lookup; `requiredSettingKinds` and the blanket activity/CHASM settings are deleted; a key the registry lacks fails the Case naming it.
- [ ] The Case-byte delta is recorded; the four ShutdownWorker Cases (activity-terminate, activity-pauseResume, activityProtocol.{terminateSettles,cancelIsRequested}) pass live; the upstream report is drafted for the owner.
- [ ] `make umpire-check-cases`, tooling and Testpilot tests, the live generated Cases and `make lint-code-fast` pass.


## Done summary
Blocked:
Blocked: deferred by the owner on 2026-10-05 together with the whole of fn-125 (dynamic configuration in the Models). Task 1 (the HSM/CHASM switch fixes) is done and merged; revive the spec to continue.
## Evidence
- Commits:
- Tests:
- PRs:
