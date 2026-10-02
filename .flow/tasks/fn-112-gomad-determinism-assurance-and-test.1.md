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
## Acceptance
- [ ] Each failing job's cause is identified from its log and recorded in the done summary
- [ ] A workflow run on the fixed commit passes every job on linux/amd64 and darwin/arm64; run URL retained
- [ ] A failure whose cause stays unknown is recorded as a finding in `MILESTONES.md` and keeps this task open
- [ ] D12 allowances are unchanged
- [ ] `make -C tools/gomad3 validate` passes locally
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
