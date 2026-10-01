---
satisfies: [R1, R9]
---
# fn-108-gomad-reduce-code-size-without-removing.1 Record baseline revision, size accounting and gate dispositions

## Description
Stage 0 of the fn-108 delivery order. Record the implementation baseline before any source edit, so the final task can prove a net production reduction (R1) and unchanged test dispositions (R9) with the same inventory and counting rule. This task edits no Go source.

**Size:** S
**Files:** new `.flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/baseline.md`, `size-count.sh`, `api-baseline/` (captured text).
**Touches:** [.flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/**]

### Approach

1. Record `git rev-parse HEAD`, branch, `git status --short` (must be clean; planning saw `d4d800fb47` on `stephanos/gomad`), `go version` for both `go` and `tools/gomad3/.toolchain/bin/go`, and `uname -sm`.
2. Write `size-count.sh`: a POSIX shell script over `git ls-files tools/gomad3 tools/gomad3sim tools/gomad3integration` (tracked files only, so `.toolchain/` and untracked output never count). It reports, per directory and in total, both physical lines and code lines (blank lines and comment-only lines excluded, so comment deletion and reformatting earn nothing) for these disjoint classes:
   - generated Go: files whose header matches `^// Code generated .* DO NOT EDIT` (19 files at planning time, including `toolchain/runtime/overlay/**` generated sources and `toolchain/version/generated.go`);
   - test Go: remaining `*_test.go` plus Go under `testdata/`;
   - runtime overlay Go: remaining `tools/gomad3/toolchain/runtime/overlay/**/*.go`;
   - authored production Go: every other `*.go`;
   - protocol/schema/template inputs: `*.tmpl` (12), `*.patch` (1), `*.json` schema/manifest files, `*.s`, `*.sh`, `*.mk`, `Makefile`.
   The script takes no arguments, is deterministic, and prints a table that can be diffed. State the exact classification rule in `baseline.md`.
3. Run it and store the output as the baseline table.
4. Capture the public surface for R8: `go doc -all` for each public package of `tools/gomad3` (`runner`, `target`, `record`, `artifact`, `choice`, `deterministicio`, `world`, `qualification/...`, `toolchain`, `upgrade`, `simulation/...`), `go doc -all ./tools/gomad3sim` from the repo root, and `--help` output of the `gomad` and `gomadtool` commands and their subcommands, into `api-baseline/`.
5. Record baseline dispositions (pass / fail with reason / not run) for: `make -C tools/gomad3 validate`, `make -C tools/gomad3 test-harness`, `make -C tools/gomad3 test-host`, `make -C tools/gomad3 world-test`, `go test -tags test_dep ./tools/gomad3sim/...` (repo root), `make gomad3-integration-test`. Record known Linux/Darwin replay findings (D12/D14 in `.plans/GOMAD_MILESTONES.md` "Open findings") as pre-existing, with their owners.

### Investigation targets

**Required:**
- `tools/gomad3/Makefile:127-174` — the gate recipes and the exact environment they use
- `Makefile:165-225` — root `gomad3-*` targets
- `.plans/GOMAD_MILESTONES.md` — "Open findings", "Constraints", "Code-size cleanup (fn-108)"

### Quick commands

```bash
git -C /Users/stephan/Workspace/temporal/gomad status --short
sh .flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/size-count.sh
make -C tools/gomad3 validate
make -C tools/gomad3 test-host
```

### Standing constraints (every fn-108 task)

- The user owns commits: no `git commit`, `git add`, `git stash`, and no worktrees. Leave changes in the working tree and report the paths.
- No new dependencies (Go modules or external tools). `tools/gomad3` is a nested module pinned to go1.27.1; `tools/gomad3sim` and `tools/gomad3integration` belong to the root module.
- Preserve existing comments: keep them with the logic they describe when code moves, and delete a comment only together with the dead code it documents. Do not compress formatting.
- Public Go names/signatures/fields/defaults, CLI commands/flags/exit statuses, schemas, canonical bytes, `HostError.Reason` values and failure precedence stay unchanged (spec "API Contracts", R8).
- Host is darwin/arm64. linux/amd64 gates cannot run here: list them as "not run (no host)" in the evidence, never as passed.
- Focused tests run from `tools/gomad3` as `env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -tags test_dep <packages>`. `.toolchain/bin/go` is the patched toolchain; `make -C tools/gomad3 toolchain` rebuilds it (needs go.dev access). New test assertions use `require` with whole-value equality.
- Evidence (commands, platform, results, remaining failures) goes under `.flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/`. A defect found on the way is recorded for its existing owner, not fixed here.

## Acceptance
- [ ] `baseline.md` records revision, clean-tree status, host platform, both Go versions, and the classification/counting rule.
- [ ] `size-count.sh` runs with no arguments from the repo root and reports production, test, generated, runtime-overlay and protocol/schema/template counts separately, in physical and code lines, for all three directories.
- [ ] Public Go surface and CLI help captures exist under `api-baseline/`.
- [ ] Each baseline gate has a recorded disposition; linux/amd64 gates are listed as not run (no host); pre-existing replay findings are named with their owners.
- [ ] No tracked source file changed; nothing staged or committed.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
