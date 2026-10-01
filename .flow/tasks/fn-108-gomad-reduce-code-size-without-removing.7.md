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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
