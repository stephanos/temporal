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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
