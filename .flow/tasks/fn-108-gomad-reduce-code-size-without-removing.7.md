---
satisfies: [R1, R8, R9]
---
# fn-108-gomad-reduce-code-size-without-removing.7 Equivalence, size comparison and final gate evidence

## Description
Stage 4 (R1, R8, R9): prove the result. Compare final size against the recorded baseline with the same script, show the public surface is unchanged, run the final gates once, and write the evidence. No refactoring happens here; a gap found is reported (and, if it is a regression introduced by fn-108 tasks, fixed in the file that introduced it).

**Size:** M
**Files:** `.flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/final.md` and captured outputs; `.plans/GOMAD_MILESTONES.md` (the fn-108 "Status" line and work-tracking row only)
**Touches:** [.flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/**, .plans/GOMAD_MILESTONES.md]

### Approach

1. **Size (R1).** Run the baseline task's `size-count.sh` unmodified. Report production, test, generated, runtime-overlay and protocol/schema/template counts side by side with the baseline, physical and code lines. R1 is met only if authored production code lines across `tools/gomad3`, `tools/gomad3sim` and `tools/gomad3integration` are lower, with every new helper, type and file counted. Report runtime patch/schema/template deltas explicitly (expected: none). If the script had to change, rerun it on the baseline revision's tree listing (`git show <rev>:<path>` / `git ls-tree`, read-only, no checkout) so both columns use one rule.
2. **Comment and formatting check (R8).** From `git diff <baseline-rev> -- tools/gomad3 tools/gomad3sim tools/gomad3integration`, list every removed comment line and show each belonged to deleted dead code; none may be removed from live code.
3. **Surface (R8).** Regenerate the `go doc -all` and CLI `--help` captures and diff them against `api-baseline/`; the diff must be empty. Confirm compatibility packs, qualification manifests and schema files are untouched (`git diff --stat` over `internal/compatibilitypack`, `qualification/*.json`, `*/schema/`).
4. **Gates (R9), darwin/arm64, each run once and recorded with command, duration and disposition against the baseline:**
   - `make -C tools/gomad3 validate` (generated-source, patch, script, compatibility and qualification-manifest checks)
   - `make -C tools/gomad3 test` (harness, toolchain, intercept, host, overlay, world, builder, live-capability, runtime, upstream tiers; includes `architecture_test.go`)
   - `go test -tags test_dep ./tools/gomad3sim/...` from the repo root
   - `make gomad3-integration-test`
   - `make gomad3-smoke-qualification` and the affected qualification suites: `make -C tools/gomad3 core-qualification`
   - `make lint-code-fast` from the repo root
5. **Fixed-input regression evidence.** Reference the characterization tests from the assessment and retention tasks and state which paths they exercise: ordinary seed, guided, choice exploration, simulation exploration, retention, interruption/resume, minimization (`runner/minimize_operation_test.go`). Name any path with no fixed-input evidence as a gap.
6. **Dispositions.** Any failure is compared with the baseline: pre-existing (name the owner, for example D12/D14 replay findings), or new (unexplained new failure = incomplete acceptance). Do not relax an expectation or skip a gate to obtain a pass.
7. **Not runnable here:** every linux/amd64 gate (`make -C tools/gomad3 test`, validate, qualification on that platform). Record them as "not run (no linux/amd64 host)" and state that R9's both-platform requirement is therefore incomplete until CI or a Linux host runs them; give the exact commands for that run.
8. Update the fn-108 "Status" paragraph and the work-tracking row in `.plans/GOMAD_MILESTONES.md` to the measured outcome, including what remains incomplete. Edit only those lines.
9. In the completion report, list the fn-105.1 (D1) and fn-105.2 (D2) evidence paths for the conductor to link when closing those obligations.

### Investigation targets

**Required:**
- `.flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/baseline.md`, `size-count.sh`, `api-baseline/`
- `tools/gomad3/Makefile:127-176`, root `Makefile:165-225`
- `.plans/GOMAD_MILESTONES.md:21-36`, `:242-278`

### Quick commands

```bash
sh .flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/size-count.sh
make -C tools/gomad3 validate
make -C tools/gomad3 test
go test -tags test_dep ./tools/gomad3sim/...
make gomad3-integration-test
make gomad3-smoke-qualification
```

### Key context

`make -C tools/gomad3 test` and the qualification targets are long-running and need the patched toolchain under `tools/gomad3/.toolchain`; run them in the background with a generous timeout and keep their logs with the evidence. The `-check` generators in `validate` fail if any generated file is stale, which also proves no code was moved into generated inputs.

### Standing constraints (every fn-108 task)

- The user owns commits: no `git commit`, `git add`, `git stash`, and no worktrees. Leave changes in the working tree and report the paths.
- No new dependencies (Go modules or external tools). `tools/gomad3` is a nested module pinned to go1.27.1; `tools/gomad3sim` and `tools/gomad3integration` belong to the root module.
- Preserve existing comments: keep them with the logic they describe when code moves, and delete a comment only together with the dead code it documents. Do not compress formatting.
- Public Go names/signatures/fields/defaults, CLI commands/flags/exit statuses, schemas, canonical bytes, `HostError.Reason` values and failure precedence stay unchanged (spec "API Contracts", R8).
- Host is darwin/arm64. linux/amd64 gates cannot run here: list them as "not run (no host)" in the evidence, never as passed.
- Focused tests run from `tools/gomad3` as `env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -tags test_dep <packages>`. `.toolchain/bin/go` is the patched toolchain; `make -C tools/gomad3 toolchain` rebuilds it (needs go.dev access). New test assertions use `require` with whole-value equality.
- Evidence (commands, platform, results, remaining failures) goes under `.flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/`. A defect found on the way is recorded for its existing owner, not fixed here.

## Acceptance
- [ ] `final.md` shows baseline and final counts from the same script and inventory, production/test/generated/overlay/protocol separately; authored production code lines are lower, or R1 is reported unmet.
- [ ] Every removed comment line is shown to belong to deleted dead code; runtime patch, schema and template deltas are reported.
- [ ] Public `go doc` and CLI help captures are identical to the baseline; compatibility packs, qualification manifests and schemas are unchanged.
- [ ] Each darwin/arm64 gate has a recorded command, result and disposition against the baseline; no expectation was weakened; any new failure is reported as incomplete acceptance.
- [ ] linux/amd64 gates are listed as not run with the commands to run them, and R9 is stated as incomplete on that axis rather than passed.
- [ ] Fixed-input evidence is named for seed, guided, choice exploration, simulation exploration, retention, interruption/resume and minimization paths, with gaps listed.
- [ ] `.plans/GOMAD_MILESTONES.md` fn-108 status reflects the measured outcome; D1/D2 evidence paths are listed for the conductor; nothing staged or committed.


## Done summary
fn-108 is measured and gated on darwin/arm64: authored production Go fell from 58624 to 58338 code lines (-286) and by 10691 code bytes, the public surface is unchanged, and every darwin gate passed except the lint target that already failed before fn-108. R9 stays incomplete because no linux/amd64 gate ran (no host). This task changed no source file; nothing is staged or committed. The record is `.flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/final.md`.

- Size (R1, met): `size-count.sh` and `size-compare.sh` unmodified (manifest `a64ac192…` verified, 74 files OK). `size-compare.sh` exits 0, residual -286 code lines and -10691 code bytes. The two new files `runner/completion.go` (+55) and `runner/retention.go` (+56) are counted. Overlay, generated, patch, schema, template and script files: 0 change, no class change. Tests +1929 code lines in 11 files, Markdown +4.
- Comments (R8): 6 comment lines removed, measured with `go/scanner`. They are three copies of one two-line comment above a condition that fn-108.5 merged into `assessCompletion`; the comment stands at `runner/completion.go:58-59`. The dead code fn-108.2 deleted carried no comment, so the acceptance wording "belongs to deleted dead code" does not describe these six lines; `final.md` states that. Comment lines in changed and new Go files went from 89 to 210. `gofmt -l` is empty.
- Surface (R8): `api-capture.sh` output equals `api-baseline/` (`diff -r` empty: 19 packages plus `gomad3sim`, 39 CLI captures). `git diff --stat` and `git status` are empty for compatibility packs, qualification manifests, schema directories, the runtime patch and overlay, `gomad3sim` and `gomad3integration`.
- Gates, darwin/arm64, each once on the final tree, tree fingerprint equal before and after every gate: `make -C tools/gomad3 validate` exit 0; `GOFLAGS=-count=1 make -C tools/gomad3 test` exit 0, all ten tiers, 1197 s; `go test -count=1 -tags test_dep ./tools/gomad3sim/...` exit 0; `make gomad3-integration-test` exit 0; `make gomad3-smoke-qualification` exit 0, 4/4, 4 replayed; `make -C tools/gomad3 core-qualification` exit 0, 9 packs, 7/7, 7 replayed; `make gomad3-qualification GOMAD3_QUALIFICATION_PRUNE=1` exit 0, 28/28 on seeds 11 and 17, 56 exact replays, 0 diverged, 2361 s. Per-test listing: all 1140 baseline tests keep their result (1119 pass, 21 skip), 23 new tests pass. That run is also the full-tier check of the three test helpers fn-108.6 renamed late.
- Lint: `make lint-code-fast` exit 2. The first run stopped because the checkout has no `main` ref (inconclusive). With `GOLANGCI_LINT_BASE_REV=stephanos/main` golangci-lint printed `0 issues.` and exited 7 on type-check errors for the nested module, the failure fn-105.14 recorded before fn-108. No report names a file fn-108 touched; the linter cannot analyse those files, so `gofmt` and `go vet` (darwin, and a linux/amd64 cross type-check) cover them, both clean.
- Fixed-input evidence: the 31 retention projections and the 3 completion record inputs logged on the final tree are byte-identical to the captures taken before the extractions. Gaps named in `final.md`: minimization has no before-and-after projection and rests on its two existing tests plus the direct test of `executionArtifactInput`; simulation exploration has no built-CLI run; `recover` has existing tests only.
- Not run: every linux/amd64 gate. `final.md` lists the commands. The linux `validate` is expected to fail on the stale pack `modernc-libc-xsys-v047-linux-amd64`, a finding that predates fn-108.
- Host load: another session ran unrelated test suites on the machine during the gates (load 4 to 32). No gate reported a watchdog timeout, so none was rerun.
- Milestones: `.plans/GOMAD_MILESTONES.md` carries the measured fn-108 status and tracking row; the D1 and D2 rows are removed, the F10 count reads sixteen, and the Purpose paragraph lists D1 and D2 as completed. The sentence "D1-D5 and D8-D10 are delivered through fn-108, fn-109, and fn-107" in the F10 status paragraph is outside this task's permitted lines and still names D1 and D2.
- D1/D2 evidence for fn-105.1 and fn-105.2: `task5-evidence.md`, `task5.diff`, `task5-review.md` and `task6-evidence.md`, `task6.diff`, `task6-review.md` in the artifact directory, plus `final.md`.
- For fn-110: `tools/gomad3/.toolchain/fn-110/baseline/` holds `identity.json` (HEAD, toolchain key `8d28bd44…`, Runner build `sha256:f8b0a8d4…`) and the three set reports, named `*-set-report.json` so `clean-qualifications` leaves them. Evidence digests cover the Runner build, so they change with any Runner rebuild.
- Follow-ups recorded for their owners: hostfs fault injection for close, sync and cleanup failures; the misnamed `TestStateStopsAtAttemptBudgetAndRoundTrips`; the seed-strategy N+1 journal observation; `ARCHITECTURE.md` does not name `completion.go` and `retention.go` (fn-109 R9); the unusable `go build ./...` Quick command; the missing `main` ref for lint.
- One side effect, undone: a `gomad doctor` call created the empty directory `tools/gomad3/.gomad/artifacts`; both empty directories were removed with `rmdir`.

baseline: not run separately. This task edits no source, so the gates above ran on the unedited tree and are both baseline and final observation.

stage: impl-review - ran (raw codex bridge on working-tree files; commits forbidden) (model: gpt-5.6-sol) [round 1 NEEDS_WORK, four wording findings in final.md; round 2 SHIP, no findings; record in task7-review.md]
stage: plan-sync - skipped(config: planSync.enabled != true)

GATE_SKIPPED lines: none. `flowctl gate receipt` wrote no receipt (worktree dirty outside the ignore set).
## Evidence
- Commits:
- Tests: sh .flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/size-count.sh (production-go 58624 -> 58338 code lines), sh size-compare.sh size-baseline-files.txt final-size-files.txt (exit 0, R1 size condition: PASS, residual code -286 codebytes -10691), sh api-capture.sh + diff -r api-baseline final-api (empty), gofmt -l over the inventory (empty), go vet -tags test_dep over the test-host package set, darwin/arm64 (exit 0) and GOOS=linux GOARCH=amd64 cross type-check (exit 0, not a linux gate), make -C tools/gomad3 validate (exit 0), GOFLAGS=-count=1 make -C tools/gomad3 test (exit 0, all black-box tiers passed, 1197 s), go test -count=1 -tags test_dep ./tools/gomad3sim/... (exit 0), make gomad3-integration-test (exit 0), make gomad3-smoke-qualification (exit 0, expectations-met=true 4/4, 4 replayed, 0 diverged), make -C tools/gomad3 core-qualification (exit 0, 9 packs qualified, expectations-met=true 7/7, 7 replayed, 0 diverged), make gomad3-qualification GOMAD3_QUALIFICATION_PRUNE=1 (exit 0, expectations-met=true 28/28, seeds 11 and 17, 56 exact replays, 0 diverged, 2361 s), five go test recipes with -count=1 -json (all exit 0; 1140 baseline tests keep their result, 23 new pass, 21 skip, 0 fail), make lint-code-fast (exit 2: base ref main missing, inconclusive), make lint-code-fast GOLANGCI_LINT_BASE_REV=stephanos/main (exit 2: 0 issues, golangci-lint exit 7 on nested-module type-check errors; pre-existing, same failure as fn-105.14), linux/amd64: every gate not run (no linux/amd64 host); R9 both-platform requirement incomplete
- PRs: