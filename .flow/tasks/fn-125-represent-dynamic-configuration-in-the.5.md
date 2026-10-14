---
satisfies: [R6]
---
# fn-125-represent-dynamic-configuration-in-the.5 Declare each dynamic-config key once in the kit and pin it to the server registry

## Description
**Absorbed by fn-144** (2026-10-06): fn-144 R17 delivers this task whole: the typed, scoped key declarations in `model/temporal/foundations`, carried in the foundations IR file, and the registry pin test. The keys fn-125's remaining tasks need are declared through that mechanism when those tasks run.

Implements R6 (spec Part B). The Temporal kit declares each dynamic-config key a Model uses once, in `model/temporal/foundations`, typed (bool, int, duration, string) and scoped, with no citation of its server definition; a Temporal-side Go test pins the declarations to the server registry, and that test is the link.

**Cross-spec entry gate:** fn-118.2 done (`ApiBehavior` IR and the kit's behavior declarations, which these keys sit beside); fn-114 closed. Not concurrent with fn-124.8. Tasks 2 and 5 both edit `ir.proto`; the second takes the next free field numbers.

**Size:** M
**Files:** `model/temporal/foundations/` (new key declarations: `DynamicConfig.bool/int/duration/string`, `Scope`); `model/irgen/**` (keys travel in the IR so Go can read them); `ir.proto` (a key catalog; `RequiredSetting` gains `relation`, `position`, `because` with `Relation {EQUAL = 0, AT_MOST, AT_LEAST}`) and generated Go; the registry pin test (location per the spec's parked unknown: `tests/` or the Temporal Driver package; decide and record).
**Touches:** [model/temporal/foundations/**, model/irgen/**, model/ir/**, proto/internal/temporal/server/api/umpire/v1/**, api/umpire/v1/**, tools/umpire/model/**, common/testing/testpilot/temporal/**, tests/testcore/testpilot/**]

### Approach
- Declare the keys fn-125 needs: `activity.enableStandalone`, `history.enableStandaloneActivityOperatorCommands`, `history.enableChasm`, `nexusoperation.enableStandalone`, the six CHASM/HSM implementation keys of `tests/nexus_workflow_test.go:82-94` (rollout percent included), `frontend.enableCancelWorkerPollsOnShutdown`, `frontend.enableMatchingFanOutForPollCancellation`, `history.timerProcessorMaxTimeShift`, and the Nexus `retryPolicy.initialInterval`. No declaration cites its server definition; the pin test links each to the registry.
- The pin test checks every declared key, and every required value in every lowered Case, against `common/dynamicconfig/registry.go` / `RegisteredSettingMetadata` (`metadata.go`): the key exists, the codec matches the declared type, the scope matches, and the value parses in the registry codec's text form (`true`, `100`, `1s`). Failures name the key, the value and the kit line.
- `RequiredSetting`'s new fields are default-empty, so existing IR and Cases keep their bytes. The realization-level `requiredSettings` stays until task 6.

### Investigation targets
**Required:**
- `common/dynamicconfig/registry.go`, `metadata.go`, `constants.go` (`DC`); `chasm/lib/activity/config.go`; `chasm/lib/nexusoperation/config.go`; `service/history/hsm/nexusoperations/config.go`
- `model/temporal/realize/{Kit,Realize}.scala`; fn-118.2's behavior declarations
- `proto/internal/temporal/server/api/umpire/v1/ir.proto:675-690` (`RequiredSetting`)
**Optional:**
- `.plans/DYNAMIC_CONFIG.md` sections 1 and 4

### Quick commands
```bash
make protoc && make umpire-gen-model && git diff --stat model/ir model/cases
go test -count=1 -tags test_dep ./tools/umpire/model/... ./common/testing/testpilot/...
```

### Execution constraints
- IR and Case bytes unchanged except the new default-empty catalog.
## Acceptance
- [ ] Each key fn-125 needs is declared once in `model/temporal/foundations`, typed and scoped, with no citation of its server definition, and reaches Go through the IR.
- [ ] A Go test checks every declared key and every required value in every lowered Case against the server registry (exists, codec, scope, parsable value); a failure names the key, the value and the kit line.
- [ ] `RequiredSetting` has `relation`, `position` and `because`; existing IR and Case bytes are unchanged; tests and `make lint-code-fast` pass.
## Done summary
Blocked:
Blocked: deferred by the owner on 2026-10-05 together with the whole of fn-125 (dynamic configuration in the Models). Task 1 (the HSM/CHASM switch fixes) is done and merged; revive the spec to continue.
## Evidence
- Commits:
- Tests:
- PRs:
