---
satisfies: [R10]
---
# fn-112-gomad-determinism-assurance-and-test.9 Consolidate the change-detector tests with a retained mapping

## Description
Source-work resumption (2026-10-07). The owner requested unblocking and completing the source tasks on the current gomad branch. This task returns to todo for its retained source work, with all dependency/admission and acceptance requirements preserved except the expressly scoped owner decisions in [source-unblocking-20261007/owner-decisions.md](../artifacts/source-unblocking-20261007/owner-decisions.md). Historical Done summary and Evidence below retain their original provenance; current lifecycle status comes from flowctl. Native qualification remains deferred under fn-128/fn-149 and is not revived by this resumption.


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: Current-source full Darwin gates, semantic preservation/mapping, real built-CLI evidence and review. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.

Shrink tests that detect change more than defects into table-driven or generated form, with no loss of asserted behavior (R10).

**Size:** M
**Files:** `tools/gomad3/cmd/gomad/internal/cli/cli_test.go`, `tools/gomad3/runner/runner_test.go`, `tools/gomad3/runner/completion_characterization_test.go`, `tools/gomad3/deterministicio/*_adapter_test.go`, `tools/gomad3/deterministicio/filesystem_patch_test.go`, `network_patch_test.go`, `tools/gomad3/architecture_test.go`
**Touches:** [`tools/gomad3/architecture_test.go`, `tools/gomad3/deterministicio/adapter_rewrite_test.go`, `tools/gomad3/runner/completion_test.go`, `.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-9/source-acceptance-20261008/**`]

### Approach
- Depends on task 2, which edits `architecture_test.go`.
- Start after fn-109 task 6 has rewritten the executor-injection tests; a dependency is recorded.
- Behavior pin: before editing, record the list of `go test -json` test and subtest names per package with pass status. After, every recorded behavior maps to a named table row or generated case. Retain the mapping file under the spec's artifacts directory.
- CLI tests that assert fields forwarded to injected dependency structs: fold into one table per command.
- The adapter test family shares one template; drive it from one table of adapters.
- `completion_characterization_test.go` and `completion_test.go` pin the same error text twice; keep one.
- `architecture_test.go`: keep import and export rules; the deleted-file list, banned words, and required filenames are candidates to remove, each with a stated reason.
- Preserve existing comments that still apply to the merged code.

### Investigation targets
**Required** (read before coding):
- `tools/gomad3/cmd/gomad/internal/cli/cli_test.go` — 16 dependency injections
- `tools/gomad3/runner/runner_test.go:1909-2268` — fake preparer and executor variants
- `tools/gomad3/deterministicio/sprig_adapter_test.go` and `validator_adapter_test.go` — near-identical pair
- `tools/gomad3/architecture_test.go:58`, `:112`, `:138` — housekeeping checks

**Optional** (reference as needed):
- `tools/gomad3/runner/completion_characterization_test.go:242` — pinned error rows
- `.flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/final.md` — fn-108's counting script and baseline of 1140 tests

### Key context
- fn-113 task 2 regenerates adapters; coordinate edits to adapter tests with it.
- fn-108 measured test lines with one counting script; reuse it for the before and after report.
### Current source-acceptance admission (2026-10-08)

The two existing consolidation parts are integrated in the current gomad branch. Verify retained R10 source acceptance rather than repeating that implementation. Worker writes are limited to fresh task9 source-acceptance evidence; preserve all Go source, assertions, comments, fixtures, original mapping/checker/behavior/status/size evidence and every prior frozen source proof. Report any exact lost assertion, unmapped behavior, stale active contract or source gap to the conductor with original/current bytes and a meaningful failure before editing outside this evidence directory. No additional behavior waiver, guard bypass or lint exception is admitted.

Re-verify fn112.2 and fn109.6 actual Done before claim. Bind both original owner commits and their parents (deterministicio/architecture and CLI/Runner/review fix), original before/after test and subtest statuses, mapping rows and current integrated named assertion bodies. A retained name alone is not proof of behavior preservation. Each removed behavior must have a concrete current row/condition or the exact task-authorized housekeeping reason; preserve every unaffected assertion, comment and fixed-input result. Identify later owners and only already-approved fn114 trace/controller migrations and completed exact fn109 assertion restorations. Rebind the task3 final accepted four bodies, not superseded pre-restoration checkpoints. Source-supported host setup failures stay failed, not generic exceptions to a lost portable assertion.

Run meaningful current focused Go JSON suites for the four affected packages, record exact top/subtest identities and failing assertions, and retain actual missing-driver/unsupported-host/stock-runtime consequences separately from portable assertion failures. Verify the mapping checker and golden error cases against their exact current fixture, with negative controls for an unmapped behavior, a removed assertion/replacement, worse status and wrong golden error as appropriate. Retain real currently built CLI process evidence (argv, binary SHA, output/status/input identities); fake forwarding tests are not a substitute for built-process evidence, and host/runtime prerequisites are not green native runs. Do not repeat broad unbounded native test-host execution on this unsupported host.

Use the unmodified fn108 size-count.sh for current test-code accounting and retain both original part-specific before/after reports with their original source identities. Distinguish the historical -1398 lines/-88 tests from the current candidate and this evidence-only wave. Run generators, error-type checks, scoped stock vet, source-path formatting and make lint-code-fast plus actual unfiltered scoped lint; attribute inherited findings to exact source/functions/owners without suppression or global-green claims. Narrow static both-source-set/runtime/first-baseline bindings remain required and must not reuse changed inputs as old passes.

The actual host is Linux/aarch64 with stock Go1.27.1 and no patched runtime. Native Darwin/Linux qualification stays deferred under fn149/fn128. Partial portable coverage never establishes a native/full-host gate pass. The conductor owns Flow lifecycle, MILESTONES, local Git checkpoints, current formal review and final acceptance; the worker hands over frozen evidence and does not claim Done or SHIP. No PR, push, native qualification, Linux CI, history rewrite, new dependency or edits to user untracked documents are authorized. Current task status comes from flowctl; older Blocked/Done prose and development-harness passes retain historical provenance. All other acceptance and dependency criteria remain unchanged.

### Narrow preservation-restoration admission (2026-10-08)

Two exact lost R10 conditions are now demonstrated by executable controls. This admission supersedes the evidence-only restriction only for the two test files and restorations below. It authorizes no production change, behavior waiver, new dependency, guard bypass, lint suppression, unrelated assertion/comment edit or native execution.

1. The seven exact historical adapter identity bodies pass against current production without a populated module cache or patched driver (adapter-original-portable-control, exit0, seven PASS). The same bodies fail against the single cache-before-identity counterfactual (adapter-original-mutant-control, exit1, seven identity-mismatch assertions receive stat errors). The current shared populated-cache test fails before the assertion because its patched driver is absent; that control is inconclusive for mutant survival and must remain so. Restore the empty-cache condition before any populated-cache or download setup, using the existing adapter table and the exact changed sum/module/version inputs. Retain the current populated-cache case, download conditions, failure predicate, identity values and applicable comment. A named empty/populated cache table within TestRewrittenModulesRejectChangedIdentity is admitted; retain concrete before/after assertion and test/subtest mappings. No adapter implementation changes are admitted.

2. The original TestPublicPackagesDoNotExportTypeAliases exported-alias predicate rejects type PublicAlias = string, while current architecture.PublicSignatures accepts it (alias-actual-condition-control, actual executable exit1 with both observations). Restore the exact original blanket AST guard as an additive TestPublicPackagesDoNotExportForwardingAliases, retaining all ten original directories, production-file filters, parsing/error handling and exported-plus-alias predicate. Preserve the existing TestPublicPackagesDoNotExportTypeAliases signature-accessibility guard byte-exact. Do not replace or rename that existing guard or change the architecture API. Verify the actual new compiled guard against valid and exported-alias fixture inputs, retaining raw terminal results and input/binary identities; source-text comparisons alone are insufficient. An isolated fixture working directory for the compiled actual test binary is preferred to temporarily inserting an alias into product source.

Preserve both test files' original bytes before editing and prove that only these admitted function changes occur. Preserve all previous frozen proofs as historical; rebind affected current-source checks, accounting and final evidence after restoration. Retain the runner suite timeout and all host/runtime setup failures as failed/incomplete observations, never current native or full-suite passes. The unchanged acceptance criteria, approved fn114 migrations, fn128/fn149 native deferrals and root-owned Flow/Git/review lifecycle remain in force.

### Exact private-completion cause restoration (2026-10-08)

The conductor reviewed the exact original completion_test.go at 59ca3d17395be5501906586006e2539d33bea28c, current private test and production assessment function, both control producers and the actual three control receipts. Original private input conditions cannot be waived merely because shared public rows pin the same messages with different coverage modes.

An evidence-only production overlay changes only Err text for CoverageSemanticChoice choice failure and CoverageSemanticChoice watchdog semantic failure, retaining Reason and every triggering input. The exact original test file against unmodified current production passes one top/eleven subtests (completion-original-cause-baseline). It rejects both changed causes (completion-original-cause-mutant, exit1, two failed subtests), while the current private test accepts the same mutant (completion-current-cause-mutant, exit0, eleven passed subtests). The shared public operations' missing-driver failures are not mutant-survival evidence.

Restore only the original cause field, four exact original row cause literals and exact-cause comparison/error reporting in TestAssessCompletionProjectsCoverageInOrderAndClassifies in runner/completion_test.go. Keep the current explicit nil-Err guard in addition to the restored comparison. Preserve every current row, name, input/result/terminal/coverage mode, existing Reason assertion, successful-result whole comparison, applicable comment, the other World test, public completion observations and all accepted fn109 whole-stat assertions. Preserve pre-edit bytes and prove that no other function or product file changes. No production edit, new behavior waiver, suppression, new dependency or broader source scope is admitted.

Run the actual restored private test against unmodified current production and the exact same retained two-cause mutant: normal must pass, mutant must fail those exact original conditions. Retain all earlier final checkpoints as pre-cause-restoration evidence and rebind affected current-source gates, mappings, accounting and lint after this restoration. Native deferrals and the root-owned Flow/Git/review lifecycle remain unchanged. Earlier two-file-only language is superseded only by this third named test restoration.

### Exact private World seed-mismatch row (2026-10-08)

The conductor read world-seed-controls.mjs, original/current World tables and actual three receipts. The evidence-only overlay changes only the actual decoded-seed7/expected-seed8 mismatch Err text, leaving decode errors, reasons and other seed inputs unchanged. The exact original table passes six children against current production, rejects only seed_mismatch against the mutant, while the current four-row private table passes that same mutant. The public mismatch row uses decoded-seed8/expected-seed7 and its missing-driver failure is not survival evidence.

Add only the exact original seed-mismatch row to TestAssessWorldValidatesTheRecordAgainstItsSeed in the already-admitted runner/completion_test.go: record recording (seed7), expected seed8, limit1<<20, cause World record seed or schema does not match seed 8. Preserve all four current rows, their order, inputs, decoder-before-seed-mismatch precedence, valid-record detached-copy assertion, parsing/error checks, applicable comments and every other assertion. No malformed-record row restoration is admitted here: the public malformed-World row retains original7/7 inputs, while the private malformed-before-seed-mismatch row uses the same decode-first error branch that does not inspect seed until successful decoding. Retain that source/input distinction explicitly, without relabelling changed inputs as an old pass.

Verify the actual restored World table against unmodified current production and the exact retained seed-mismatch mutant: normal must pass and mutant must fail seed_mismatch. Rebind the current three-file source proof, accepted-body bridges, affected gates, mapping, size and lint under fresh names; older proofs remain historical. This supersedes the earlier other-World-test-unchanged restriction only for this one exact additive row. No production changes, new behavior waivers, suppressions, native qualification or scope beyond the three admitted test files are authorized.
## Acceptance


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.

- [ ] A mapping file lists every removed test and the table row or case that replaces it
- [ ] No recorded behavior is lost; removed housekeeping checks each carry a reason
- [ ] Test code lines and test counts before and after are reported with the fn-108 counting script
- [ ] `make -C tools/gomad3 test-host` and `validate` pass on darwin/arm64; linux status recorded
## Done summary
Blocked:
Blocked: only the native darwin/arm64 and linux/amd64 gates remain for task 9; both parts are implemented and reviewed.

Done: the deterministicio/architecture part (97f221c7ea) and the CLI/Runner part, now that fn-109 tasks 2 to 12 have merged (ebc5ea1e09, plus review fix 7ba322acc8 on gomad-fn112-9 over 59ca3d1739). The CLI forwarded-field and outcome tests are now one table per command. The runner preparation, resume, coordinator-response, replay-preflight and cancellation tests are now tables, and one canned coordinator helper replaces six. Seven validation tests are removed because golden rows pin their error text, and one golden row is added. completion_test.go no longer pins the same error text twice.

Local evidence (linux/arm64, development harness on, not native evidence):
- Behavior pin: `go test -json` on cli (361 -> 364 names) and runner (679 -> 654). `mapping-check.py` covers all four packages with exit 0 and problems: 0, and checks each validation mapping against the golden error. The failure sets match before and after.
- fn-108 size-count.sh: this part takes test-go code from 64394 to 64168 (-226) and top-level tests from 1605 to 1565 (-40). Both parts together: -1398 code lines and -88 tests.
- `make -C tools/gomad3 validate`: exit 0 (5 s).
- `GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host`: exit 2 (258 s, harness key 60e4051c...). The cli and runner failures are the baseline ones. The other failures are in packages this task does not touch and come from the harness or host (`test-host-cli-runner.txt`).
- Impl-review (claude:claude-fable-5-1:high): SHIP with three P3s. One is fixed; the other two (test shape) are left as is.

Remaining native gates: on darwin/arm64 and on linux/amd64, run `make -C tools/gomad3 validate` and `GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host`.
## Evidence
- Commits:
- Tests:
- PRs:

## Linux ownership blocker (2026-10-04)

Linux ownership amendment (2026-10-04): all native Linux execution obligations moved to fn-128. Missing transferred Linux evidence no longer blocks this task. Source-owned acceptance remains incomplete for Current-source full Darwin gates, semantic preservation/mapping, real built-CLI evidence and review. Keep the task blocked for those independent requirements, with current-source evidence required by its original acceptance. See the scoped Description/Acceptance and .flow/artifacts/linux-scope-transfer-2026-10-04.md.
