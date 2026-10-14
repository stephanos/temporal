---
satisfies: [R6, R11]
---
# fn-122-capabilities-and-their-laws.4 Model the standalone Nexus operation with Close, Terminate, Cancel and Describe, and run its generated Cases live

## Description
The second instantiating entity. A minimal Model of the standalone Nexus operation (`chasm/lib/nexusoperation`): start, describe, request-cancel, terminate and handler completion; no retry, no deadline. It declares `Closable`, `Terminable`, `Cancelable` and `Describable`, and its generated `terminateSettles` and `cancelIsRequested` lower to Cases that run live. `rejected` is the parameter where it differs from the activity; where the server answers a mutation of a closed operation differently from `closedIsRejectedUniformly`, it overrides with a recorded reason (the vision's acceptance test). Depends on task 3 for the harness's new-IR-file and new-Case allow-list categories.

**Size:** M
**Files:** `model/temporal/nexusoperation/{Model,Properties,Queries,Realization}.scala` (new folder, fn-112's layout, captured-name forms from the start so fn-114.7 need not touch it); its IR-file declaration or gate root; `model/ir/nexus-operation.json` + `.laws.json`; `model/cases/**` (its own Cases); `model/temporal/realize/**` only for the dynamic-configuration flag that enables standalone Nexus operations, as a new value; `tools/umpire/internal/golden/original.json` (the new IR file and Cases by name).
**Touches:** [model/temporal/nexusoperation/**, model/temporal/realize/**, model/gate/Roots.scala, model/ir/nexus-operation.json, model/ir/nexus-operation.laws.json, model/cases/**, tools/umpire/internal/golden/original.json, .plans/UMPIRE_MODULES.md]

### Approach
- Vocabulary: a status object with `status`, `terminal`, `running`, `cancelRequested` as named defs; phases scheduled/started/completed/failed/canceled/terminated; facts named after the status the step lands in (the Describe law).
- Server grounding, cited per step: `chasm/lib/nexusoperation/operation.go` (`ErrOperationAlreadyCompleted`, `ErrCancellationAlreadyRequested` with request id), `handler.go` (`StartNexusOperation`, `DescribeNexusOperation`, `RequestCancelNexusOperation`, `TerminateNexusOperation`), `frontend.go` (`isStandaloneNexusOperationEnabled`, the feature flag the Case's dynamic configuration must enable).
- Realization with the shared kit (`temporal/realize`, fn-112.9) against fn-117's typed `WorkflowServiceGrpc.METHOD_*NEXUS_OPERATION_EXECUTION` methods; the handler's completion is answered by the Driver's Nexus handler role as the Nexus caller realization already does. No literal wait (the kit's one interval until fn-118).
- Declare the capabilities with `limits`; `rejected` is a parameter, cited; an override carries its reason. Record in the done summary which laws now have two instantiating machines.
- Keep it to a few screens; a construct the DSL lacks is a finding appended to `.flow/tmp/fn122-4/findings.md`, never a Go workaround.

### Investigation targets
**Required:**
- `chasm/lib/nexusoperation/{operation.go,handler.go,frontend.go}`
- `tests/nexus_standalone_test.go` - the live calls and the flag they enable
- `model/temporal/standaloneactivity/` (post task 3) - the capability declarations to mirror
- `model/temporal/realize/` - the kit
**Optional:**
- `model/temporal/nexuscaller/Realization.scala` - the Nexus handler role

### Quick commands
```bash
make umpire-gen-model && make umpire-check-model
make umpire-check-live-tests
```

### Execution constraints
- Existing IR files and Cases are untouched; this task adds one IR file, its sidecar and its Cases, each allow-listed by name.
- fn-114 owns `model/temporal/nexuscaller/**`; this task does not edit it. `model/gate/Roots.scala` may be edited by fn-114.1 at the same time; whichever lands second rebases.
## Acceptance
- [ ] `model/temporal/nexusoperation` has Model, Properties, Queries and Realization in fn-112's forms, declares the four capabilities with cited parameters and `limits`, and passes the model gate.
- [ ] Generated `terminateSettles` and `cancelIsRequested` lower to Cases that run live and replayed with a satisfied Verdict; the feature flag off fails preparation naming the flag.
- [ ] `rejected` differs from the activity's as a parameter; any override carries its reason in the sidecar; the done summary lists the laws that now have two instantiating machines.
- [ ] Existing IR and Cases are byte-identical; the new IR file, sidecar and Cases are allow-listed by name; model gate and the live runner pass.
## Done summary
Modeled the standalone Nexus operation as the laws' second instantiating entity, with its generated Cases running live.

**What changed**
- **`model/temporal/nexusoperation/`** (fn-112 layout, captured-name forms):
  - `Model.scala`: the machine `nexusOperation`.
    - Actions: start, requestCancel, terminate, handlerReply (sync success, sync failure, async) and complete (succeeded, failed, canceled). No retry, no deadline.
    - Phases: unstarted, scheduled, started, succeeded, failed, canceled, terminated, with a cancel-requested flag.
    - Facts are named after the status each step lands in. Each step cites operation.go / operation_statemachine.go.
  - `Properties.scala`: `operationCapabilities` (limits `three`).
    - Closable: `rejected = Outcome.alreadyCompleted`, the parameter where it differs from the activity's NotFound (ErrOperationAlreadyCompleted).
    - Terminable and Cancelable: reach `Seq(start)`, expect `inconclusive(explanationsDisagree)`.
    - Describable: `operationStatus`.
    - `.overriding(closedIsRejectedUniformly -> closedRejectsOrRepeats, because = …)`: the server answers a repeated request id OK after close (operation.go RequestCancel and Terminate). The sidecar records the override and its reason.
  - `Queries.scala`: limits and expectations.
  - `Realization.scala`: built on the kit, against `METHOD_{START,REQUEST_CANCEL,TERMINATE,DESCRIBE}_NEXUS_OPERATION_EXECUTION`. It has a Describe status table, an await on terminated, and the endpoint role. No handler answers, so the operation stays running until a control lands.
  - `IrFiles.scala`: `irFile("nexus-operation")`.
- **Kit:** `nexusEndpointName`, appended at the end so no Model's positions move.
- **Harness:**
  - `original.json`: `new_ir_files: ["ir/nexus-operation.json"]` and its two new Cases.
  - `config.json`: `later_inventory`.
  - The lowering comparison now lowers listed new IR files with the rest.
- **Live suite:** applies each generated Case's declared required settings (see below), beside the blanket `activity.Enabled`.
- **Docs:** `.plans/UMPIRE_MODULES.md` (module row), `.plans/SEMANTIC_PROTOCOLS.md` (the second entity, plus the laws with two machines).
- **Catalog test:** now counts the declared machines.
- **IR and Cases:** existing IR files and Cases are byte-identical. Only the manifest gains the four nexus-operation entries.

**Laws that now have two checked-in instantiating machines** (model/ir/*.laws.json):
- `terminalStatesAreFinal`: activityProduct, currentAdmission, nexusOperation.
- `closedIsRejectedUniformly`: the same three; overridden on nexusOperation, waived on the admission designs.
- `pausedIsNotDispatched`: activityProduct, currentAdmission.
- `terminateSettles`: activityProtocol, nexusOperation.
- `cancelIsRequested`: activityProtocol, nexusOperation.

**Verdicts:** `nexusOperation.terminalStatesAreFinal` and `.closedIsRejectedUniformly` (the override) are verified; `.terminateSettles` and `.cancelIsRequested` are found and lowered.

**Live** (live.md; under the lock with nothing else heavy):
- Both nexus-operation Cases passed in 20 of 20 Runs (5 runs × hsm/chasm × 2 Cases), with a SATISFIED Verdict and the expected Property assessment.
- The activity's Cases that stop the worker failed most runs at the stop-worker instruction's 10 s limit, with an INCONCLUSIVE Verdict:
  - generated terminateSettles: 1/5 hsm, 0/5 chasm;
  - generated cancelIsRequested: 2/5 and 2/5;
  - authored terminate: 1/5 and 1/5.
- That is fn-121's signature: matching's ShutdownWorker returns early when the root partition is not loaded. It is recorded for fn-118/fn-121. No window was widened, and no server or Profile change was made.

**Required settings** (the conductor chose option (a) on the NEEDS_HUMAN, commit 6872629c51):
- **Protos:** two new generic fields, `Realization.required_settings = 15` in the IR and `Program.required_settings = 8` in Testpilot, each a repeated `RequiredSetting{key, value}`.
  - The IR field is listed as an inert schema addition (original.json and config.json `inert_fields`, plus schema_test's added fields and messages), the way fn-112.11 and fn-120.1 recorded theirs.
  - The Testpilot proto has no captured-schema test.
- **Scala:**
  - `umpire.realize.RequiredSetting` and the kit's `temporalRealization(…, requiredSettings = …)`. The kit's later line numbers are kept, so existing IR and Cases are byte-identical.
  - The Nexus operation realization requires `nexusoperation.enableStandalone = "true"`, which `make umpire-gen-model` lifts into its IR.
- **Lowering:** carries the settings into the Case's Program, and the inventory accounts for them. Only the two nexus-operation Cases change.
- **Prepare:** refuses a Profile whose configuration lacks a required setting or sets it otherwise. It answers PreparationUnavailable at `program.required_settings[i]`, for example: `required setting "nexusoperation.enableStandalone" must be "true", and the Profile's configuration leaves it unset`. Malformed entries are refused as PreparationMalformed.
- **Tests:**
  - Prepare unit tests.
  - A negative test on the real nexus-operation Case fixtures: unset and false are refused naming the flag, true is admitted.
  - A lowering inventory test and IR validator cases.
- **Live harness:** applies the union of the lowered Cases' declared settings through a small key→setting table, failing on an unknown key. It records them in `binding.DynamicConfig` and now passes that into the Profile, so Prepare checks what the server runs with. The blanket `nexusoperation.Enabled` is gone.
- **Catalog rotation:** the proto field changes the Driver catalog identity. `catalog_test.go` takes the new literal, and `make umpire-rerecord-pinned-runs` re-recorded the pinned Runs and receipt goldens. The replay Case fixture had drifted from its generated Case (pre-existing old source paths), so it was refreshed to the generated one first.

**Findings** (findings.md):
1. **The flag-off criterion is resolved** by the required settings above.
2. **No handler carrier:** the Driver has no Nexus-handler carrier for a standalone operation. Handler paths are modeled and verified but do not lower. This is recorded in `.plans/SEMANTIC_PROTOCOLS.md`.
3. **Request ids:** a Case sends the run as every request id, so the Model folds the "different request id" errors into its repeated-request stutters.

**Review:** claude-opus-5-5 at high via `--spec claude:claude-opus-5-5:high`. Writer and reviewer are the same family (Opus).
- **Round 1:** NEEDS_HUMAN on the flag-off criterion. The conductor chose (a), expanding the task with the required settings above. Both P3s were fixed in e335f4d8dc.
- **Round 2:** after `review-rounds reset`, the same base and receipt. All three findings were fixed or withdrawn, and the verdict is SHIP.
- **FYI:** the realization's header pointed at a scratch file; it now points at SEMANTIC_PROTOCOLS.md (45b5018f49, a comment on the same line).

**Gates after the expansion:** all pass (evidence.md): lint-protos, umpire-check-model, the full Go suite, lint-model, lint-code-fast, tests/testcore/testpilot and the live nexus-operation Cases 4/4.

**Shared files for the merge:**
- Protos and bindings: `proto/internal/temporal/server/api/{umpire,testpilot}/v1/`, `api/{umpire,testpilot}/v1/`.
- `common/testing/testpilot/{prepare.go,prepare_test.go,internal/execution/{prepare,program}.go,temporal/catalog_test.go}`.
- Re-recorded testdata: `common/testing/testpilot/{evaluation,replay}/testdata` and `tools/canary/assessment/testdata`.
- `tests/testpilot_generated_test.go`, `tests/testcore/testpilot/model_fixture_test.go`.
- `tools/umpire/lower/{lower,realization}.go`, `tools/umpire/lower/internal/producer/*`, `tools/umpire/lower/{inventory,original_migration}_test.go`.
- `tools/umpire/model/{validate_realization.go,realization_test.go,schema_test.go,capabilities_test.go}`.
- `tools/umpire/internal/golden/{original.go,original.json,golden.go,config.json}`.
- `model/umpire/realize/Realize.scala`, `model/temporal/realize/Kit.scala`, `model/README.md`, `.plans/{UMPIRE_MODULES,SEMANTIC_PROTOCOLS}.md`.
- After merging: regenerate model/ir and model/cases, and re-record the pinned Runs if the catalog moves again.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: cc0957e498, e335f4d8dc, 6872629c51, e3f6432ea4, 45b5018f49
- Tests: make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks (exit 0), go test -json -tags test_dep -count=1 -p 2 -timeout 40m ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (exit 0), make lint-model (exit 0), make lint-protos (exit 0), GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main make lint-code-fast (exit 0), go test -tags test_dep ./tests/testcore/testpilot/ (exit 0), go test -tags 'test_dep integration' ./tests -run TestTestpilotGeneratedCases/.../nexus-operation (4/4 PASS; 20/20 over 5 runs before the expansion), flowctl claude impl-review --spec claude:claude-opus-5-5:high (round 2 SHIP)
- PRs: