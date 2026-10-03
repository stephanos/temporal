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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
