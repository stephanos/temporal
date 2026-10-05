---
satisfies: [R9, R10, R11, R18, R19]
---
# fn-126-read-each-feature-top-to-bottom-one.6 Rename in one batch: Product and System, the history record, actions and designs; close

## Description
The spec's last task: one batch of renames (R18), the levels named Product and System and the system contract renamed the history record (R19), retired names checked (R10), and the spec closed with its evidence (R11). This task accepts new Definition IDs, re-captures the golden baseline once and regenerates once.

**Cross-spec entry gate:**
- Task 5 is done.
- Never alongside fn-124.8.
- Before fn-124.7: its `original.json` re-capture is the last one the harness takes.
- fn-125 may resume only after this task.

**Size:** L
**Files:**
- every feature file and `Realization.scala`;
- `model/ir/**` (with `activity-system.*` → `activity-record.*`), `model/cases/**` and `manifest.json`;
- `tests/testcore/testpilot/testdata/{generated-case-names.txt,generated/**}`;
- `tools/umpire/{model,lower,lint,export,conformance,cmd,internal}/**` tests;
- `tools/canary/{casebinding,assessment,preflight}/**`;
- `tools/umpire/internal/golden/{config,original}.json`;
- `tools/umpire/model/layout_test.go`;
- the R9 docs.

**Touches:** [model/**, tests/testcore/testpilot/**, tools/umpire/**, tools/canary/**, .plans/**, AGENTS.md]

### Approach
- Apply R18's table in one change:
  - level objects and machines (`ActivityProtocol` → `ActivitySystem`, `NexusProtocol` → `NexusSystem`) and their state and fact types;
  - record designs, members and compositions;
  - the action renames inside the actor objects;
  - `SystemFamily` → `RecordFamily` with its family string, the `activity-system` IR file and `activitySystemFile`.

  The `…System$package$` pin strings stay, with a comment that they name the former file.
- Regenerate the IR, sidecars, lint acceptance files (rewrite the `owner` and `subjects` keys; reasons unchanged), Cases, manifest, generated fixtures, the Case-name golden and the canary Case. Update the Go tests and canary bindings that name these values. Re-capture `original.json` once.
- R10: the layout test also fails on a live file naming a retired machine, action or folder name from R18's table. Archives and `.flow/` are exempt.
- R19: rewrite "protocol machine"/"protocol" for this level as "System", and "system contract" as "history record", in the R9 docs. The reserved party `system` stays.
- Close (R11): measure against task 1's baseline. List every recorded R5 delta and every renamed Query, Case, law and lint key. Run the full gates once, including the live generated Cases (the known ShutdownWorker INCONCLUSIVE cases excepted).

### Investigation targets
**Required:**
- `model/ir/*.lint.json`, `model/ir/*.laws.json`, `model/cases/manifest.json`
- `tests/testcore/testpilot/testdata/generated-case-names.txt`
- `grep -rln "activityProtocol\|nexusProtocol\|currentAdmission\|staleAdmission\|heldAdmission\|admissionResponseLoss\|currentOverQueue\|attemptStart\|attemptResult\|activity-system\|standalone.system" --include=*.go --include=*.json --include=*.txt --include=*.scala --include=*.md . | grep -v "^./.flow/\|archive"`
**Optional:**
- `.plans/DSL_SIMPLIFICATION.md` section 4b (the full action table)

### Quick commands
```bash
make umpire-check-model && make lint-model && make lint-code-fast
go test -count=1 -tags test_dep -p 2 -timeout 30m ./tools/umpire/... ./tools/canary/... ./common/testing/testpilot/...
make umpire-check-cases && make umpire-check-fixtures && make canary-check-case
```

### Execution constraints
- Only names change. A table, answer, verdict or table-fingerprint difference beyond the renamed names stops the task.

## Acceptance
- [ ] Every rename in R18's table is applied in one change. The Models, IR, sidecars, lint acceptance keys, Cases, manifest, Case-name golden, generated fixtures, canary bindings, Go tests and golden baseline all carry the new names, and `original.json` was re-captured once.
- [ ] Tables, answers, verdicts and table fingerprints are unchanged apart from names. The done summary lists every changed Query, Case, law and lint-key name.
- [ ] The layout test fails on a live file naming a retired R18 name. The R9 docs use Product and System and "history record", and the prose grep of R19 is empty.
- [ ] The done summary gives R11's before/after evidence per folder and every recorded R5 delta.
- [ ] All gates of the spec's Verification pass, and the live generated Cases run with only the known ShutdownWorker INCONCLUSIVE outcomes.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
