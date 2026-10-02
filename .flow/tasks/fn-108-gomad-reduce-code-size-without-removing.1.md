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
5. Record baseline dispositions (pass / fail with reason / not run) for: `make -C tools/gomad3 validate`, `make -C tools/gomad3 test-harness`, `make -C tools/gomad3 test-host`, `make -C tools/gomad3 world-test`, `go test -tags test_dep ./tools/gomad3sim/...` (repo root), `make gomad3-integration-test`. Record known Linux/Darwin replay findings (D12/D14 in `MILESTONES.md` "Open findings") as pre-existing, with their owners.

### Investigation targets

**Required:**
- `tools/gomad3/Makefile:127-174` — the gate recipes and the exact environment they use
- `Makefile:165-225` — root `gomad3-*` targets
- `MILESTONES.md` — "Open findings", "Constraints", "Code-size cleanup (fn-108)"

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
Recorded the fn-108 implementation baseline at `6782b55f49a0317b230e827ea2a63a37d116d502` on darwin/arm64 under `.flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/`. No Go source changed, nothing is staged or committed. Review ended at SHIP in the fourth round, which the conductor authorized after the three-round cap.

Baseline (counting rule v2, `size-count.sh`): authored production Go across the three directories is 252 files, 63187 physical lines, 58624 code lines, 1970324 code bytes. Test Go 279 files / 42849 code lines, runtime-overlay Go 35 / 11462, generated Go 19 / 4124, protocol/schema/template inputs 56 / 13926, other 77 / 5918. The inventory (718 paths) equals the HEAD tree; `git diff --stat HEAD` and the untracked listing for the three directories are empty.

Deviations from the task text, each stated in `baseline.md`:
- Baseline revision is HEAD `6782b55f4` with 80 uncommitted docs/evidence paths (conductor decision), recorded in `git-status-baseline.txt`.
- The inventory is tracked plus untracked-not-ignored files, because fn-108 forbids commits and a tracked-only count would miss new helper files. Any ignored path other than `.toolchain/` and `.bin/` fails the script.
- A sixth class `other` makes the classes cover the whole inventory. A fourth unit `codebytes` (non-whitespace bytes outside comments, commas and semicolons excluded in Go) is the guard against line joining.
- `size-compare.sh` (not in the task's file list) decides the R1 size condition: production change plus per-file growth in overlay, generated, protocol-input and non-Markdown other files must be negative in code lines and code bytes, and no file may change class.
- `api-capture.sh` (not in the task's file list) reproduces `api-baseline/`: `go doc -all` for 19 public packages plus `gomad3sim`, and 39 CLI usage/help captures with exit status and separate streams. `simulation/...` has no Go package and the module root package is test-only; neither has a capture.

Gates on darwin/arm64, go1.27.1 on PATH, toolchain key `8d28bd44…`, all exit 0: `make -C tools/gomad3 validate`, `test-harness` (50 tests pass), `test-host` (45 packages ok, 975 pass, 21 skip), `world-test` (36 pass), `go test -tags test_dep ./tools/gomad3sim/...` (55 pass), `make gomad3-integration-test` (3 pass). linux/amd64: all six not run (host unavailable); `validate` is expected to fail there on the stale pack. Not run on either platform and without a baseline here: the other `make -C tools/gomad3 test` tiers, clock-audit, compatibility-pack and core qualification, smoke, temporal and tests qualification.

Pre-existing findings recorded with owners: D12 linux replay divergence (open, fn-105.12); D14 darwin Chasm divergence (fixed in the baseline, fn-105.14); stale linux pack `modernc-libc-xsys-v047-linux-amd64` (open, recorded by fn-105.26, no open task owns it, needs a linux/amd64 host).

Integrity: `SHA256SUMS` digest `a64ac192433c03e4c7813027dd8a9832790a6fd34af0217468231ab6a1a64327` covers 74 files; `baseline.md` digest `54e10ab35ba98f2b67ac4f178f36f3fd0deecdca294358e1d62fcef26715190c`. Round 4 reviewed `baseline.md` at `b3a8c0dd…`; the one change since names `task1-review.md` as outside the manifest and is applied and unreviewed. `flowctl gate classify` reports FULL (uncommitted `MILESTONES.md` from earlier tasks); the six gates ran in full and no gate receipt was written.

Session restart: the scratchpad was cleared mid-task. The six gate runs had finished before it (`gate-logs/results.txt` ends with ALL_DONE) and no gate was in flight, so none was re-run. The go/scanner cross-check program, the fixture repositories, the extra capture runs and the round-1 and round-2 review files are gone from disk; `baseline.md` reports the check results and `task1-review.md` reconstructs those two rounds.

baseline: green (the six gates above, run before any file was written)

stage: impl-review - ran (raw codex bridge on working-tree files; commits forbidden; 4 rounds, 4th authorized by conductor) (model: gpt-5.6-sol) [round 1 NEEDS_WORK 5 findings, round 2 NEEDS_WORK 3, round 3 NEEDS_WORK 1, round 4 SHIP; record in task1-review.md]
stage: plan-sync - skipped(config: planSync.enabled != true)

GATE_SKIPPED lines: none.
## Evidence
- Commits:
- Tests: make -C tools/gomad3 validate (darwin/arm64, exit 0), make -C tools/gomad3 test-harness (darwin/arm64, exit 0), make -C tools/gomad3 test-host (darwin/arm64, exit 0), make -C tools/gomad3 world-test (darwin/arm64, exit 0), go test -tags test_dep ./tools/gomad3sim/... (darwin/arm64, exit 0), make gomad3-integration-test (darwin/arm64, exit 0), sh .flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/size-count.sh (exit 0, two runs byte-identical), shasum -a 256 -c SHA256SUMS (74 files OK), linux/amd64: all six gates not run (host unavailable)
- PRs: