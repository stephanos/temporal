---
satisfies: [R10]
---
# fn-112-gomad-determinism-assurance-and-test.9 Consolidate the change-detector tests with a retained mapping

## Description
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
- [ ] A mapping file lists every removed test and the table row or case that replaces it
- [ ] No recorded behavior is lost; removed housekeeping checks each carry a reason
- [ ] Test code lines and test counts before and after are reported with the fn-108 counting script
- [ ] `make -C tools/gomad3 test-host` and `validate` pass on darwin/arm64; linux status recorded
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
