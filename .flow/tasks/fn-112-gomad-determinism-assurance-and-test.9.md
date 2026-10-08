---
satisfies: [R10]
---
# fn-112-gomad-determinism-assurance-and-test.9 Consolidate the change-detector tests with a retained mapping

## Description


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: Current-source full Darwin gates, semantic preservation/mapping, real built-CLI evidence and review. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.

Shrink tests that detect change more than defects into table-driven or generated form, with no loss of asserted behavior (R10).

**Size:** M
**Files:** `tools/gomad3/cmd/gomad/internal/cli/cli_test.go`, `tools/gomad3/runner/runner_test.go`, `tools/gomad3/runner/completion_characterization_test.go`, `tools/gomad3/deterministicio/*_adapter_test.go`, `tools/gomad3/deterministicio/filesystem_patch_test.go`, `network_patch_test.go`, `tools/gomad3/architecture_test.go`
**Touches:** [tools/gomad3/cmd/gomad/internal/cli/cli_test.go, tools/gomad3/runner/runner_test.go, tools/gomad3/runner/completion_characterization_test.go, tools/gomad3/runner/completion_test.go, tools/gomad3/deterministicio/*_adapter_test.go, tools/gomad3/deterministicio/filesystem_patch_test.go, tools/gomad3/deterministicio/network_patch_test.go, tools/gomad3/architecture_test.go, .flow/artifacts/fn-112-gomad-determinism-assurance-and-test/**]

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
