---
satisfies: [R1]
---
# fn-112-gomad-determinism-assurance-and-test.1 Reproduce and fix the failing gomad3 workflow jobs

## Description
Get `gomad3.yml` and `gomad3-smoke.yml` green on both platforms (R1). Split from the gate changes in task 2 so a red baseline is not confused with newly enabled tests.

**Size:** M
**Files:** `.github/workflows/gomad3.yml`, `.github/workflows/gomad3-smoke.yml`, `tools/gomad3/Makefile`, plus whatever source the diagnosed failures name
**Touches:** [.github/workflows/gomad3.yml, .github/workflows/gomad3-smoke.yml, tools/gomad3/Makefile]

### Approach
- Read the failing job logs first: `gh run list --repo stephanos/temporal --workflow gomad3.yml -L 10`, then `gh run view <id> --log-failed`. The latest run failed in `make validate`, the host tier, and linux smoke.
- `make -C tools/gomad3 validate` passes locally on darwin/arm64 in under two seconds, so the `validate` failure is specific to linux or to an older commit. Reproduce on the commit CI ran.
- CI runs on a committed tree. The working tree has uncommitted changes, so ask the owner to commit or push a branch before judging a run; do not commit unprompted.
- Leave the D12 `nondeterministic` and `replay_divergence` allowances as they are (fn-105 R12 owns removing them).
- The declared write surface is the workflows and the Makefile. If a diagnosed failure needs a source fix elsewhere, name the files in the done summary and check that no in-progress task of this spec is editing them.
- Share linux gate evidence with fn-108 task 8, which runs the same gates.

### Investigation targets
**Required** (read before coding):
- `.github/workflows/gomad3.yml:33-58` — host-tools job that failed in `make validate`
- `.github/workflows/gomad3.yml:59-95` — core-linux tiers
- `.github/workflows/gomad3-smoke.yml:58-107` — linux smoke job and its evidence check
- `tools/gomad3/Makefile:138-153` — `validate` composition

**Optional** (reference as needed):
- `.flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/final.md` — linux commands already listed

### Key context
- The toolchain builder downloads Go source from go.dev; cloud sessions cannot reach it, GitHub CI can.
- All 16 recent pull-request runs failed or were cancelled within about 6 minutes, which points at an early step.

### Current verification evidence (2026-10-02 UTC)
Task 1 review scope is the committed baseline repair, not task 2 or the whole branch. Compare against failed CI commit `38957053f1ce342a8797af1803f5f8f6bb53fcad`; the fix was qualified at `8789deab055d1b72ac6bc86711d74f3fd7313fa2`. Later gate and diagnostics additions are separate tasks and are not claimed qualified by that run.

Read `.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-1/summary.md` and its adjacent evidence files: `ci-run.json` records all six successful jobs and the green run URL; `previous-failures.log` retains failed-job excerpts; `previous-smoke-summary.json` retains denied boundaries from the downloaded failed report; `source-bindings.json` binds inspected source; `validate-darwin-arm64.log` records local `make -C tools/gomad3 validate` exit 0. Green run: https://github.com/stephanos/temporal/actions/runs/36968858553 . Failed run: https://github.com/stephanos/temporal/actions/runs/36945812832 .

The actual repairs to review are `tools/gomad3/runner/retention_characterization_test.go` (derive fixture ranks from platform-bound candidate identities) and `tools/gomad3/internal/compatibilitypack/{requests,packs}/modernc-libc-xsys-v047-linux-amd64.json` with its authoring report and generation approval (rebind the stale Linux deterministic-I/O profile). Host-tools validation and smoke blockers came from the stale pack; core-linux also failed platform-specific retention fixture assumptions. Workflow changes up to the green SHA only update milestone references; D12 nondeterministic/replay_divergence allowances remain unchanged. Read the matching MILESTONES baseline evidence paragraph. No unknown baseline failure remains. No new commits, staging, push, or dispatch were made for this evidence task.

The previous review used HEAD as the base, yielding an empty committed diff and missing the untracked evidence; this corrected task description exposes the retained evidence and corrected implementation base without changing acceptance or claiming completion. Review only R1 and the named task-1 repairs/evidence; do not review task 2 implementation, task 2 findings, or later diagnostics changes.
## Acceptance
- [ ] Each failing job's cause is identified from its log and recorded in the done summary
- [ ] A workflow run on the fixed commit passes every job on linux/amd64 and darwin/arm64; run URL retained
- [ ] A failure whose cause stays unknown is recorded as a finding in `MILESTONES.md` and keeps this task open
- [ ] D12 allowances are unchanged
- [ ] `make -C tools/gomad3 validate` passes locally
## Done summary
Task 1 baseline evidence, reviewed 2026-10-02 UTC.

All six jobs passed in https://github.com/stephanos/temporal/actions/runs/36968858553 on committed tree 8789deab055d1b72ac6bc86711d74f3fd7313fa2: core (darwin/arm64), core-linux, host-tools-linux, both functional-smoke platform jobs, and temporal-integration. This qualifies the baseline fix, not subsequent task-2 gate additions. ci-run.json binds the result and each job URL to that SHA.

The previous failed run https://github.com/stephanos/temporal/actions/runs/36945812832 tested 38957053f1ce342a8797af1803f5f8f6bb53fcad. Its three failures are identified:

- host-tools-linux / validate-compatibility: modernc-libc-xsys-v047-linux-amd64 still bound libc and memory adapters to profile 964874354d1ad2f46f00e3d1b1109f89b0d1747b399a97ea867ff29c48eda8e1 instead of current Linux profile 84b27e6227508fda72c8be6b0ae588e4f4f052a07bc35a27d6feff729b22d50b. The request, generated pack, authoring report, and generation approval were updated on the fixed commit.
- core-linux / test-host: the same stale pack failed TestHostPacksBindCurrentProfile; retention characterization fixtures also assumed alternatives had the same rank on both platforms. Candidate identities bind the platform and sort differently on Linux. retention_characterization_test.go now derives alternative ranks from a campaign with distinct probes before injecting rank-specific novelty/failure scenarios. Production candidate ordering remains unchanged.
- functional-smoke-linux: all four workloads were unsupported because the stale modernc pack did not activate. Downloaded report evidence shows denied bigfft linknames, x/sys syscall and assembly, and libc/memory boundaries. The fixed run passes the smoke verification without broadening admission or D12 allowances.

previous-failures.log retains exact failed-job excerpts; previous-smoke-summary.json projects the failed smoke report. source-bindings.json hashes the inspected source. make -C tools/gomad3 validate passed again on darwin/arm64 (validate-darwin-arm64.log).

No unknown baseline failure remains. D12 allowances are unchanged: workflow differences from the failed SHA to the green SHA only repair MILESTONES.md references; intermittent Linux qualifications still accept nondeterministic and replay_divergence. No commit, staging, push, or dispatch was performed by this worker. The parent independently verified the completed run through gh; implementation review returned SHIP with R1 met and no findings. The review covers this baseline repair only.

Review evidence: review-receipt.json and review.md. No commit created; the tested commit was already supplied by the owner.

stage: impl-review - ran (model: gpt-6-astra at high)
stage: plan-sync - skipped(config: planSync.enabled != true)
Tracker sync: n/a (bridge inactive)
## Evidence
- Commits: 8789deab055d1b72ac6bc86711d74f3fd7313fa2
- Tests: make -C tools/gomad3 validate, gh run view 36968858553 --repo stephanos/temporal --json status,conclusion,url,headSha,jobs
- PRs: