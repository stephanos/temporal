---
satisfies: [R6]
---
# fn-124-shrink-and-simplify-the-umpire-go.6 Make Model assessment the command-line judge and derive or retire the Evaluation Profile

## Description
Implements R6: umpire-run and umpire-assess run Model assessment (conformance) when given the Model/IR, beside the Contract Verdict; the Evaluation Profile's reason table is retired (host decision for the owner, 2026-10-05), not derived.

### Recommendation: retire the Profile as a separate judge; don't derive it.
1. **No Model declaration exists to derive it from.** `common/testing/testpilot/evaluation/profiles/local-ephemeral.json` and `tools/canary/assessment/profiles/production-canary.json` are frozen output of the archived Lean `Umpire.Evaluation` (`evaluation/profile.go:22,117`: "as Umpire.Evaluation spells them", "Lean's Profile.render"). The fields that differ between them (claim, trust, blocking gap kinds, how an unsupported rule is treated) describe a deployment, not behaviour. Declaring them in Scala would put deployment policy into the Models, in exactly the files fn-126 restructures.
2. **Most of the reason table repeats the verdict aggregation.** `evaluation.Admit` already requires `recordedrun.Agreement` (`evaluation/admission.go:162`), which makes `verdict-violated` equivalent to `disposition-stopped`, and `disposition-incomplete` imply `verdict-inconclusive`. So 4 of the 7 conditions in `assess.go:89-97` come down to task 4's `Conclude`.

### Approach
- **Evaluation:** `evaluation.Assess` (`assess.go:47-86`) becomes a fixed precedence:
  1. Verdict violated, or a nonconformant/violated Model assessment when one is supplied: rejected.
  2. Verdict inconclusive, cleanup not succeeded, a blocking known gap, or an unsupported rule: incomplete, unless the policy says an unsupported rule rejects.
  3. Otherwise accepted.
- The Profile shrinks to policy: `{version: 2, name, claim, trust, blockingKnownGaps, unsupportedRule}`. Delete `Reason`, `conditions`, `forcedDecisions` and the reason table from both JSON files. Receipt reasons become fixed ids. Bump the receipt version and re-render the goldens (`Makefile:585-624`, `evaluation/testdata/receipts/*`).
- **Canary:** `tools/canary/controller/controller.go:233-239` and `tools/canary/assessment/profile.go` keep loading their policy profile. The canary has no Model dependency (module map), so it supplies no Model assessment.
- **Shared limits:** add `conformance.DefaultLimits`, moved from `tests/testcore/testpilot/model_fixture.go:128-130`, so the live tests and both commands assess under the same ceilings.
- **`umpire-run`** (`tools/umpire/cmd/umpire-run/run.go`):
  - Optional `--model <dir>` (the `model/` root). When given, read `<dir>/cases/manifest.json` with `lower.DecodeManifest`, find the entry whose Case has the fixture's fingerprint, `umpiremodel.Load` `<dir>/ir/<entry.Model>`, and call `conformance.Prepare(model, entry.Query, source, DefaultLimits)`.
  - Run through a new `binding.Bound.RunAssessed(ctx, testpilot.AssessmentFactory)` (`common/testing/testpilot/temporal/binding/binding.go:251`), which uses `PreparedCase.WithAssessment(...).Run`.
  - `report` adds `conformance <status> <reason>` and `property <id> <status> <reason>` lines, plus `expected match|differs: …` from task 5's `ExpectedRun.Check`.
  - Exit code: the worse of the Verdict and the assessment (violated/nonconformant=1, inconclusive=2, assessment failure=3). Without `--model`, behaviour and output are unchanged.
- **`umpire-assess`** (`tools/umpire/cmd/umpire-assess/run.go`):
  - Optional `--model <dir>`. Prepare the admitted subject's Case offline (`binding.Prepare`, no I/O, as `umpire-run` already does at `run.go:84`), then `AssessedCase.Evaluate(ctx, run, nil)` (`common/testing/testpilot/assessment.go:301`).
  - Feed the Assessment into `Assess`; the receipt records `assessment.{model,query,conformance,properties}`.
  - `--profile` keeps naming the policy.
- **Docs:** update the Testpilot README "Running a Case from the command line" (`README.md:408-428`), `main.go` package docs, `.plans/UMPIRE4_SPEC.md` "Claim Assessment" (≈l.495) and the module map rows for Evaluation and Commands. Record the decision and its two reasons in the done summary.
## Acceptance
- [ ] Given `--model model`, `umpire-run` and `umpire-assess` run Model assessment for a generated Case. Both print or record conformance and every property with its reason id beside the Contract Verdict. Without `--model`, output and exit codes are byte-identical to today, and the existing `run_test.go` suites pass unchanged.
- [ ] `umpire-run`'s exit-code rule with an assessment is stated in the Testpilot README. A test covers each code: satisfied, a violated property under a satisfied Verdict, nonconformant, inconclusive, and assessment failure.
- [ ] `umpire-assess` on a recorded generated Run reproduces the Assessment the live test produced for the same Run (one `recordedrun` fixture). A recorded Run whose Model or Query differs is refused (`ErrForeignAssessment`).
- [ ] The Evaluation Profile has no reason or condition table. `Assess` decides by one fixed precedence that uses `testpilot.ConcludeVerdict`. Both profiles, the receipt goldens and the canary receipts are re-rendered, and the receipt version is bumped. The done summary records "retired, not derived" with its reasons.
- [ ] `TestUmpireRunImportsNeitherTheTestClusterNorAServerService` and `TestUmpireRunLinksNoTestClusterAndNoNewServerService` still pass. `tools/umpire/model/ownership_test.go` and the module map permit `umpire-run`/`umpire-assess` → `conformance`, `lower`, `model`.
- [ ] Go tooling suite, Testpilot and canary tests, `make lint-code-fast` and `make umpire-check-cases` pass with Case bytes unchanged. One live `umpire-run --model` against a dev server passes for `activity-completion-case.json`.
- [ ] `ExpectedRun.Check` also compares the conformance reason id: the IR `RunExpectation` and the manifest gain a declared conformance reason (deferred from fn-124.5's review), and the reader refuses an inconclusive or nonconformant expectation without one.
## Done summary
### Decision: retired, not derived
`Assess` now decides by one fixed precedence. A Profile keeps only the policy that differs between deployments. There were two reasons not to derive the table from the Model:
1. **Nothing in the Model could produce it.** The Profiles are frozen output of the archived Lean `Umpire.Evaluation`. What differs between them is claim, trust, blocking gap kinds and what an unsupported rule forces. That describes a deployment, not behaviour, and declaring it in Scala would put deployment policy into the Models.
2. **Most of the table repeated the verdict aggregation.** Admission already enforces `recordedrun.Agreement`. So `disposition-stopped` is the same as `verdict-violated`, and `disposition-incomplete` implies `verdict-inconclusive`. Four of the seven conditions reduce to `testpilot.ConcludeVerdict`.

### What changed
- **Evaluation** (`ae539a8fc0`, `d64e4b4cf7`):
  - The signature is now `Assess(subject, profile, assessment *testpilot.Assessment)`. The Verdict is read through `ConcludeVerdict`; when a hand-built subject disagrees with it, the worse of the two decides.
  - **Rejected:** a violated Verdict, a nonconformant Run, a violated property, or an unsupported rule when the Profile says it rejects.
  - **Incomplete:** an inconclusive Verdict, a cleanup that did not succeed, a blocking Known Gap, an unsupported rule, or an assessment that failed or left anything inconclusive.
  - **Accepted** otherwise.
  - Reasons are fixed ids (`evaluation.Reason`). `monitor-stopped` and `run-incomplete` are no longer separate reasons.
  - The Profile format is now version 2: `{version, name, claim, trust, blockingKnownGaps, unsupportedRule}`.
  - The receipt format is now version 2. Reasons are listed by id, and an optional `assessment` section records the model and query identities, the conformance and each property (status, reason id, supporting sequences) and the failure by code, with no prose.
- **Groundwork** (`ff8b2c3fd1`, `9460935a9b`):
  - `conformance.DefaultLimits()`, moved here from the live-test fixture.
  - `binding.Bound.RunAssessed`.
  - `lower.FindGeneratedCase`, which finds a Case's manifest entry by `CaseFingerprint`.
  - `ExpectedRun.Check` prints a missing reason as `none`.
- **`umpire-run --model <dir>`** (`ff914d7e89`, `726ccb9f71`):
  - It finds the Case, loads its IR and prepares the assessment before anything is opened, then runs the Case with the assessment.
  - It prints `conformance <status> [<reason>]` and one `property <id> <status> [<reason>]` line per property.
  - It prints `assessment failed <code> at event <n>` if the assessment failed, then `expected match`, or one `expected differs: …` line per difference.
- **`umpire-assess --model <dir>`** (same commits):
  - It prepares the Case offline, under the recorded Profile name, with every required setting and delivery control.
  - It replays the recorded events through `AssessedCase.Evaluate`, and refuses a Run that does not replay to its recorded Verdict.
  - The receipt records the assessment.
  - Two new statuses: `model-unassessable` and `assessment-unreproducible`.
- **Without `--model`**, both commands behave as before. The existing `run_test.go` tests of both pass unchanged, except one `umpire-assess` expectation, whose reason list loses `monitor-stopped` because of the retirement.
- **Declared conformance reason** (`dfca50683d`, `ef1d488284`):
  - IR field `RunExpectation.conformance_reason = 9`; in Scala, `RunExpectation(..., conformanceReason: Option[Reason] = None)`.
  - The reader, the manifest and generation all enforce one rule: an expected Run names a conformance reason exactly when its conformance is inconclusive or nonconformant, and the id must be known and spelled as `ExpectationID` spells it.
  - There are refusal tests for:
    - inconclusive without a reason;
    - nonconformant without a reason;
    - a reason on a conformant expectation;
    - an unknown id, and a mixed-case id.
  - `Check` always compares the reason; a conformant expectation names none.
  - forgedCompletion declares `incomplete`. The Run is stopped by its monitor, so it never closes complete; both the live and the offline judge report `incomplete`.
  - The lifter fixture `Capabilities.settles` pins the lift with `hole`.
- **Frozen archives** (`658b7b92f2`):
  - `golden.DeclaredRuns` gains `conformance_reasons`, closed both ways and tested. `original.json` maps `CONFORMANCE_INCONCLUSIVE` to `REASON_INCOMPLETE`; the archive's only such expected Run is forgedCompletion's.
  - The fn-112 migration inputs for `nexus-control.json` declare the reason in place.
  - 151 derived lowering artifacts were captured again the way `TestMigrationGoldens` derives them. They differ only by the reason and the digests and ids covering it: everything under `nexus-control`, plus the two generated manifests.
  - The original baseline's nexus-control exploration digest in `testdata/original/lower.json` was recaptured.
- **Docs:** the Testpilot README (command line, exit rule, `Assess` precedence), `tools/umpire/README.md`, both commands' `main.go`, `model/README.md`, `model/SEMANTICS.md`, the spec's Claim Assessment, and the module map rows for Evaluation, Lowering, Conformance, binding and Commands. `ownership_test.go` pins the commands' imports of conformance, lower and model.

### `umpire-run --model` exit codes (stated in the Testpilot README)
- `1`: a violated Verdict, a nonconformant Run, or a violated property, including one established before the assessment failed.
- Otherwise `3`: the assessment failed.
- Otherwise `2`: anything inconclusive.
- Otherwise `0`.

Whether the Run matches its expectation is printed but does not change the exit code. `TestRunWithAModelReportsTheAssessmentAndExitsByTheWorse` covers each code.

### Identities and goldens
- The receipt version is now 2. The `accepted`, `rejected` and `incomplete` goldens were re-rendered with `UMPIRE_RECEIPT_GOLDENS=write`, and an `assessed` golden was added.
- Profile identities:

| Profile | Before | After |
|---|---|---|
| `local-ephemeral` | `2803afa2…` | `05b2152d…` |
| `production-canary` | `3da213bc…` | `6265dc08…` |
| `canary-harness` | `cfb66759…` | `a5367ed7…` |

- `local-strict` was re-rendered.
- The canary provenance goldens were re-rendered: `released` is now `da77e7d2…` and `uncertain` is now `2c4f5e7b…`. The canary schema test now checks each document's own version.
- **What moved:**
  - `model/ir/nexus-control.json` gains the reason, and its Queries.scala positions shift by one line.
  - `model/cases/manifest.json` and the functional fixture manifest gain `"conformanceReason": "incomplete"`.
- **What did not:**
  - Every Case file and the canary pin are unchanged.
  - No Testpilot proto changed, so the catalog identity and the pinned Runs are unchanged.
  - `.plans/umpire-migration-manifest.json` keeps the old hashes, since it is a historical record.

### Deviations
- **Conformance reason: resolved.** forgedCompletion declares `conformanceReason = Some(Reason.incomplete)`, the reader and manifest require a reason on every inconclusive or nonconformant expectation, and `Check` always compares it.
- **`ErrForeignAssessment` acceptance case: nothing for `umpire-assess` to act on.**
  - A recorded Run carries no Assessment, so `umpire-assess` passes `nil` as the recorded one.
  - It refuses instead:
    - a Case the model does not lower (`model-unassessable`);
    - an assessment bound to another Case (`WithAssessment`);
    - a Run that does not replay to its recorded Verdict (`assessment-unreproducible`).
  - `ErrForeignAssessment` itself stays tested in testpilot and conformance.

### Merges
- `umpire` was merged twice. The first was `a766d10cba`, as `894c37cab0`.
- The second was `cea96f308f` (with fn-126.1), as `b265459b75`. One conflict, in the `Capabilities.scala` lifter fixture, was resolved by keeping both sides; its expected outputs were regenerated.
- After the second merge, `umpire-gen-model`, `umpire-gen-fixtures` and `canary-gen-case` produced no further diff.

### Gates on the re-merged tree (logs in `.flow/tmp/fn124-6/`)
| Gate | Result | Log |
|---|---|---|
| `make umpire-check-model` (`--skip-go-checks`) | exit 0 | `final-check-model.log` |
| `make lint-model` | exit 0 | `final-lint-model.log` |
| Full Go suite, `-json -p 2` | 6155 pass, 0 fail, 198 s wall. `model` and `export` were OOM-killed, then passed alone at `-p 1` (model 136 s, export 74 s). This covers the original-baseline check, `./tools/umpire/lower/...`, `./tools/umpire/model/...` and `./tools/umpire/lint` | `final-go-suite.json`, `final-go-model.json`, `final-go-export.json` |
| `make umpire-check-cases`, `umpire-check-fixtures`, `canary-check-case` | exit 0 | `final-check-*.log` |
| `make lint-code-fast` (origin/main, no fix) | exit 0 | `final-lint.log` |
| `buf breaking` against `umpire` `a766d10cba` (internal and chasm) | pass. Run by hand; this round did not touch the protos | `buf-breaking.log` |

**Live runs** (earlier round, before the reason was required):
- `make umpire-check-live-tests` exited 2 after 470 s (`live.log`). 25 top-level tests passed, including `TestTestpilotAssessRecordedRuns` and every canary harness test.
- These failed, all in the known ShutdownWorker-race family (INCONCLUSIVE with "the Run recorded no evidence", or expected 1 but got 3):
  - `activity-activityProtocol.cancelIsRequested`
  - `activity-pauseResume`
  - `activity-terminate`
  - `nexus-caller-scheduleToStartTimeout` (hsm and chasm)
  - `TestTestpilotNexusCallerScheduleToStartTimeout/hsm`
  - `TestTestpilotWorkerOutageCase`
- Live `umpire-run --model` on `activity-completion-case.json` against a sqlite dev server exited 0 with `conformance conformant`, `property completes satisfied`, `expected match`. Its record, assessed by `umpire-assess --model`, was accepted.
- Live forgedCompletion exited 1 with `conformance inconclusive incomplete`, which is the reason it now declares, and its offline assessment matched.
- The live suite was not rerun after the reason became required.

### For the owner
- `make protoc` does not work in this sandbox, because `.bin/goimports-v0.49.0` is a macOS binary. The subagent regenerated `ir.pb.go` with a Linux goimports it built in the scratchpad, and restored the `chasm/lib/*/gen` files the failed run had deleted; they now match HEAD.
- Both merge commits use git's default message, without the attribution trailer.

Subagents: 1 (Opus), which did the first round of conformance-reason plumbing.
Review: claude-opus-5-5 at high, fresh context (host-dispatched subagent). Writer and reviewer are the same family (Opus). Round 1: SHIP, no P1/P2. The reviewer reran `go vet` (integration tags) and `go list -deps` for both commands (no test cluster; only the three permitted server services), and spot-checked the frozen nexus-control input and two recaptured artifacts.
- P3s applied in f8b923e581:
  - `umpire-assess` refuses a receipt root inside the `--model` directory.
  - `Assess` treats a Run stopped by its Monitor as violated, so a hand-built stopped-but-satisfied subject is rejected as the retired table did.
  - Module map lists `umpire-assess` → `testpilot/temporal`.
  - SEMANTICS rewrap.
- Deviation accepted by the host for the owner: the acceptance's "a recorded Run whose Model or Query differs is refused (`ErrForeignAssessment`)" cannot arise in `umpire-assess`, because a recorded Run carries no Assessment. The command instead:
  - refuses a Case the model does not lower;
  - refuses an assessment bound to another Case;
  - refuses a Run that does not replay to its recorded Verdict.
  - The receipt's `assessment.model` identity records which Model judged it. Recording the Assessment identity beside the Run is a possible later improvement.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: ae539a8fc0, d64e4b4cf7, ff8b2c3fd1, dfca50683d, 9460935a9b, 30562d9da2, ff914d7e89, 894c37cab0, 726ccb9f71, ef1d488284, 658b7b92f2, b265459b75, f8b923e581
- Tests: make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks on the re-merged tree (exit 0; merge2-gen-model.log), make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks (exit 0; final-check-model.log), make lint-model (exit 0; final-lint-model.log), go test -count=1 -json -tags test_dep -p 2 -timeout 30m ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (6155 pass, 0 fail; model and export OOM-killed; final-go-suite.json), go test -tags test_dep -p 1 ./tools/umpire/model/ (pass, 136s; final-go-model.json) and ./tools/umpire/export/ (pass, 74s; final-go-export.json), make umpire-check-cases, umpire-check-fixtures, canary-check-case (exit 0), make GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main lint-code-fast (exit 0; final-lint.log), buf breaking vs umpire a766d10cba (pass; buf-breaking.log), make umpire-check-live-tests before the requirement (exit 2: known ShutdownWorker-race INCONCLUSIVEs only; live.log), live umpire-run --model activity-completion (exit 0, expected match) and umpire-assess --model on its record (accepted), go test -count=1 -tags test_dep ./common/testing/testpilot/evaluation/... ./tools/umpire/cmd/umpire-assess/... after the review P3s (pass; p3-focused.log), make GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main lint-code-fast after the review P3s (exit 0; p3-lint.log)
- PRs: