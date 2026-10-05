---
satisfies: [R5]
---
# fn-124-shrink-and-simplify-the-umpire-go.5 Compare generated-Case outcomes by declared disposition, cleanup and reason ids

## Description
Implements R5: the generated-Case comparison checks what each Query's `expected_run` declares — the Contract verdict, the Run's disposition and cleanup, conformance, and each property's status and reason id — by equality, instead of inferring the disposition from "violated" and matching reason prose by suffix.

Approach:
- IR (`ir.proto` `RunExpectation`): enums `Disposition`, `Cleanup`, `Reason`; fields `disposition = 6`, `cleanup = 7`, `reason = 8` (enum), `MonitorExpectation.reason = 4` (enum); the string fields 2 and 3 reserved. The Umpire IR is outside buf's breaking scope, but fields are still replaced rather than retyped.
- Scala (`umpire.realize`): `RunExpectation` takes contract, disposition and cleanup without defaults (the lifter leaves a defaulted argument unset, so a framework default would reach Go as an unset field Go must interpret — the inference R5 removes); `reason: Option[Reason]`. the Temporal kit declares the common expectation once (`satisfied`, `inconclusive(reason)` in `Kit.scala`), replacing the three Models' copies; prose vals deleted.
- Go reader (`model/validate.go`): requires contract, disposition, cleanup; reason iff not satisfied.
- Lowering (`lower/generated.go`): manifest carries the ids; `validateExpectedRun` checks `(contract, disposition)` against `testpilot.ConcludeVerdict`; `ExpectedRun.Check` is the reusable comparison (task 6 reuses it).
- Judge: `testpilot.{Conformance,Property}Assessment.Reason`; `conformance/conclude.go` concludes an IR reason, maps it to prose once, and reports both.
- Live test: `fixture.Expected.Check(run, verdict, assessment)`.
- Golden original-baseline harness: the archive (old schema) is read with expected Runs declared as its reader read them (`golden.DeclaredRuns`); two frozen exploration digests recaptured.
## Acceptance
- [ ] The IR's `RunExpectation` declares, for every Query that has one, its Contract verdict (`contract`), the Run's `disposition` and `cleanup`, and each non-satisfied outcome's reason as an id (`RunExpectation.Reason`, the judge's reasons: no evidence, incomplete, hole, unexplained, explanations disagree, never evaluated, unreadable, every explanation violates). The string `reason` fields are reserved by number and replaced (`RunExpectation.reason = 8`, `MonitorExpectation.reason = 4`); no field changes type in place.
- [ ] Scala: `RunExpectation(conformance, property, contract, disposition, cleanup, reason = None, monitors = …)` with `Disposition`, `Cleanup` and `Reason` enums beside `Conformance`; contract, disposition and cleanup have no framework default (the lifter leaves a defaulted argument unset), so each is declared: once in the Temporal kit's `satisfied` / `inconclusive(reason)` (`model/temporal/realize/Kit.scala`), which replace the three Models' copies, and in the two hand-written expectations. The Models' reason prose vals are gone; `forgedCompletion` declares `contract = violated`, `disposition = stoppedByMonitor`.
- [ ] `model/ir/*.json` and `model/cases/manifest.json` (and the functional fixtures' manifest) change only by these fields (`contract`, `disposition`, `cleanup` added; `reason` prose replaced by the id) and the source positions of the edited Scala files. Every Case file is byte-identical: `git diff -- model/cases ':!model/cases/manifest.json'` is empty.
- [ ] Reader: an expected Run with an unset contract, disposition or cleanup, a satisfied outcome with a reason, or a non-satisfied outcome without one is refused at its Query's position (tests).
- [ ] Lowering: `lower.ExpectedRun` carries `contract`, `disposition`, `cleanup`, and each `ExpectedClaim.reason` is the id. The manifest reader refuses a `(contract, disposition)` pair `testpilot.ConcludeVerdict` cannot produce, an unknown disposition or cleanup, and a reason that is not a known id (tests). `ExpectedRun.Check(run, verdict, assessment)` compares disposition, cleanup, Contract verdict, conformance and every property's status and reason id by equality, and is unit-tested both ways. It compares conformance by status only for now: an expected Run declares no conformance reason, which fn-124.6 takes up (`conformance_reason`). Every id is spelled by `umpiremodel.ExpectationID`/`EnumID` alone (a mixed-case id is refused), and a value an enum does not name reads `unknown(N)`, never a panic.
- [ ] Judge: `testpilot.ConformanceAssessment` and `PropertyAssessment` gain `Reason`, a stable id beside the prose `Detail`; conformance sets it from the reason it concluded (one table in `conformance/conclude.go` maps each IR reason to its prose); Testpilot refuses an outcome with an invalid reason id and carries it through settle, bounding and replay. The Testpilot README's "How a Run is judged" names it.
- [ ] `tests/testpilot_generated_test.go` checks the Run through `fixture.Expected.Check`; no `HasSuffix`, no disposition or verdict inferred from "violated".
- [ ] A grep for the judge's reason sentences ("the executions that explain the evidence disagree", "an execution that explains the evidence never reaches the claim's evaluation point", "every modeled execution that explains the evidence violates it") over `model/`, `tools/` and `tests/` matches only `tools/umpire/conformance/conclude.go` and the original-baseline delta `tools/umpire/internal/golden/original.json`, which maps the frozen archive's prose to the ids (retired with the harness by R7).
- [ ] Docs (model/README.md, model/SEMANTICS.md "Generated Case expectations") describe the declared fields.
- [ ] Original-baseline harness: the archive, frozen in the old schema, is read with each expected Run declared as its reader read it (`golden.DeclaredRuns`, `declared_run_expectations` in `original.json`: unwritten contract satisfied, a violated contract stopped by the monitor, any other completed, cleanup succeeded, each prose reason its listed id; closed both ways, unit-tested). The two frozen exploration digests whose candidate Models carry expected Runs with reasons (`nexus-caller`, `nexus-control` in `testdata/original/lower.json`) are recaptured from that reading; nothing else in the archive changes.
- [ ] fn-112 migration harnesses: the frozen IR inputs (`model/testdata/migration/inputs`, `lower/testdata/migration/original/inputs`) are declared in the new schema once, by the same rule, each differing only in its expected Runs; the lowering artifacts derived from them are captured again and differ only by expected Runs and the digests covering them (checked by normalization); job oracles untouched. The schema rename test lists the replaced and added fields and compares wire bytes without expected Runs.
- [ ] Gates: model gate (`umpire-gen-model`, then check), `make lint-model`, original-baseline check, `make umpire-check-cases`, `make umpire-check-fixtures`, `make canary-check-case`, `make buf-breaking`, the full Go tooling suite + Testpilot + canary (`-json`, timed), `make lint-code-fast`; the live generated Cases run once (known ShutdownWorker-race INCONCLUSIVEs and `activity-pauseResume` excepted).
## Done summary
R5 is implemented and committed on branch `umpire-fn124-5` (HEAD `437430d6cf`, with `umpire` at `d3ee2ad1d5` merged in). It is not fully green: the live run has 3 failing Cases and the Go suite 2 failing tests. All five also fail on the unmodified `umpire` base, so they predate this task.

Writing `summary.md` was blocked for subagents, so the full summary is here. `.flow/tmp/fn124-5/evidence.json` was written. I did not run `flowctl done`.

### What changed
- **Testpilot (`2626d27e01`):** `ConformanceAssessment` and `PropertyAssessment` gain `Reason`, a stable id kept beside the prose `Detail`. Testpilot refuses an outcome whose reason isn't a valid id and carries the reason through unchanged. The README's "How a Run is judged" section documents it.
- **IR (`f6200f81cd`):** `RunExpectation` gains `Disposition`, `Cleanup` and `Reason` enums and the fields `disposition=6`, `cleanup=7` and `reason=8`. `MonitorExpectation` gains `reason=4`. The old string fields 2 and 3 are reserved rather than given a new type.
- **Scala:** `RunExpectation(conformance, property, contract, disposition, cleanup, reason: Option[Reason] = None, monitors)`.
  - The Temporal kit (`model/temporal/realize/Kit.scala`) now declares `satisfied` and `inconclusive(reason)` once, replacing the three Models' copies and their reason prose.
  - `forgedCompletion` declares a Run stopped by its monitor (`stoppedByMonitor`).
- **Reader:** an expected Run without a contract, disposition or cleanup is refused at its Query's line. So is a satisfied outcome that names a reason, or any other outcome that names none. `umpiremodel.ExpectationID` is the one place the ids are spelled.
- **Judge:** `conformance/conclude.go` holds the only wording of each reason and reports the id beside the prose.
- **Lowering:** the manifest carries the ids. `GenerateCases` and the manifest reader refuse a contract/disposition pair that `testpilot.ConcludeVerdict` can't produce. `lower.ExpectedRun.Check(run, verdict, assessment)` compares everything by equality; task 6 can reuse it.
- **Live test:** it now goes through `fixture.Expected.Check`. There is no suffix match on reason text and no disposition inferred from "violated".
- **Docs:** `model/README.md` and `model/SEMANTICS.md` updated, along with the lifter fixtures and the functional and canary manifests.
- **Original-baseline harness (`6068f3aced`, `13a287295b`):** the archive is frozen in the old schema, so it is now read with each expected Run filled in the way the old reader interpreted it. That rule is `golden.DeclaredRuns`, listed in `original.json` and unit-tested. Two frozen exploration digests (nexus-caller, nexus-control) were recaptured.
- **Migration archives (`aa853e93c6`):** the old fn-112 migration harnesses re-derive everything from frozen inputs and compare bytes, so these inputs no longer decoded. I rewrote them once, in place, in the new schema using the same rule; each differs only in its expected Runs.
  - I recaptured the 234 artifacts derived from them. A normalization check confirmed they differ only by expected Runs and the digests or ids that cover them. The job oracles are untouched.
  - The schema-rename test now lists the replaced and added fields, and compares wire bytes without expected Runs.
- **Flow (`312e6ef43c`, `437430d6cf`):** approach and final acceptance written into the task.

### IR and manifest changes
- **IR files:** each expected Run gains `contract`, `disposition` and `cleanup`, and the prose reasons become `REASON_*` values. Source positions shift in the edited Scala files; the golden harness ignores positions.
- **Manifests:** they gain the three new values, and the prose reasons become ids such as `explanations_disagree`.
- **Case files:** every Case file is byte-identical.
- **Reason prose:** a grep for the three reason sentences matches only `conclude.go` and `original.json`, which maps the frozen archive's prose to the ids.

### Gates (logs under `.flow/tmp/fn124-5/`)
| Gate | Result |
|---|---|
| Model gate check (`umpire-check-model --skip-go-checks`) | pass |
| `umpire-gen-model` | ran (`gen-model-1.log`); its output is committed |
| `make lint-model` | pass |
| Original-baseline check | pass |
| `umpire-check-cases`, `umpire-check-fixtures`, `canary-check-case` | pass |
| `make lint-code-fast` | pass |
| `buf-breaking` against the `umpire` base | pass, run by hand from a scratch clone, because the script fetches commits that aren't pushed |
| `make buf-breaking` (default) | fails on `routing/v1/extension.proto`, an upstream change already on the branch. The cause is the stale local `main`, not this task |
| Full Go suite (`-json`, `-p 2`, 196 s) | 45 of 51 packages pass, 3 skip; 2 tests fail, both on the base too (below) |
| `tools/umpire/model` alone at `-p 1` (139 s) | only `TestCapabilitiesGeneratedClaims` fails; it was OOM-killed at `-p 2` |
| `tools/umpire/export` alone at `-p 1` (75 s) | pass; it was OOM-killed at `-p 2` |

The slowest tests in the suite were `TestOriginalBaselineCases` (47 s), `TestOriginalBaselineModel` (46 s) and `TestMigrationProjectionPreservesSemantics` (42 s).

**Live generated Cases** (one run, 90 s): 15 of 18 pass, including `activity-pauseResume` and `forgedCompletion` (stopped by monitor, both Nexus implementations). Three fail:
- `activity-activityProtocol.cancelIsRequested`
- `activity-activityProtocol.terminateSettles`
- `activity-terminate`

Each ends with an INCONCLUSIVE Verdict and no recorded evidence, consistent with the known worker-stop (ShutdownWorker) race. I ran the same three on the unmodified `umpire` base and they fail the same way (`live-base.log`). The new `Check` reports every differing value, so the failure messages now say why.

### Decisions
- **No framework defaults** for contract, disposition or cleanup. The lifter leaves a defaulted argument unset, so a Scala default would reach Go as an empty field that Go would have to interpret, which is the inference R5 removes. The common expectation is written once in the kit instead.
- **Contract is declared too.** Go used to read an unset contract as "satisfied", which is the same kind of inference.
- **The new enum field keeps the name `reason`** under a new number, so the Scala parameter name is unchanged.
- **Frozen archives:** the original archive is converted as it is read; the migration archives are converted in place, because those harnesses compare bytes. R7 retires both harnesses.

### For the owner
- `lint TestCoverageSummaryIsPinned` and `model TestCapabilitiesGeneratedClaims` already fail on `umpire`, since fn-127.2 added the `rightNeverHeld` Query to the capabilities lifter fixture. I left them alone as out of scope.
- `make buf-breaking` needs either a fresh local `main` or a pushed base to pass as the script is written.
- No subagents were used.
Review: claude-opus-5-5 at high, fresh context (host-dispatched subagent). Writer and reviewer are the same family (Opus).
- Round 1: SHIP, no P1/P2. The reviewer normalized all 244 changed migration archives against base: every difference is an expected-Run field or a digest/id covering one.
- P3s applied in 2e024c9ad2 and 19828c47a9:
  - One id spelling rule (`ExpectationID`/`EnumID`); a mixed-case id is refused.
  - `unknown(N)` instead of a panic for an unknown enum number.
  - `Check` no longer prints `: []` when a Run has no diagnostics.
  - New refusal tests: monitor reasons, unknown cleanup, and an invalid nonconformance reason in Testpilot.
  - Module map entries for `ExpectationID`/`EnumID` and the kit's `satisfied`/`inconclusive`.
  - R5 now says the judge's `conclude.go` owns the reason prose.
- Deferred to fn-124.6 (recorded in its acceptance): declaring and comparing the conformance reason. `Check` compares conformance by status only for now.
- The two Go failures the first full run showed (`TestCoverageSummaryIsPinned`, `TestCapabilitiesGeneratedClaims`) were drift from fn-127.2's last fixture, fixed on `umpire` by 95fb3da27e and merged here (6a89f38c7d).
- Reruns after the fixes:
  - Focused packages (lower, testpilot, lint): 23 pass.
  - `tools/umpire/model` at `-p 1`: pass.
  - `umpire-check-cases`: pass.
  - lint-code-fast: pass.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 2626d27e01, f6200f81cd, 6068f3aced, 312e6ef43c, 13a287295b, aa853e93c6, 437430d6cf
- Tests:
- PRs: