---
satisfies: [R7]
---
# fn-105-gomad-follow-ups-deferred-scope.7 D7: add required macOS functional smoke CI

## Description
Origin: fn-101.4 (F7 R4). The user activates D7 as required work on 2026-09-30. Add a darwin/arm64 job to the existing functional smoke workflow using a standard GitHub-hosted macOS runner, following the repository's current macOS runner pattern. Run the same selected functional tests and relevant triggers as the Linux job. Retain platform-specific qualification evidence with uniquely named artifacts. Standard macOS runner compute is free for this public repository under [GitHub's runner policy](https://docs.github.com/en/actions/reference/runners/github-hosted-runners), checked on 2026-09-30. Preserve Linux coverage and existing workflow comments.

## Acceptance
- The workflow verifies darwin/arm64 and checks that the qualification manifest is current before running the same smoke selection as Linux.
- Qualification reports identify the Darwin platform, selected seeds, complete workload counts, and met expectations. Require zero unsupported, failed, and infrastructure_errors outcomes.
- Keep the representative smoke replay gate explicitly traced under R13. Every selected workload qualifies with verified exact choice-tape replay. An affected D12/D14 divergence remains an open required fix and cannot be accepted as a completed smoke gate.
- Upload Darwin evidence with a platform-specific artifact name that cannot collide with Linux uploads. Preserve the existing Linux job's selection and coverage.
- Verify workflow configuration and a real GitHub Actions Darwin run. Retain the run link and qualification report. A workflow edit without passing runner evidence does not close the task.

## Done summary
Verified the existing macOS smoke implementation against GitHub Actions job 110647570973 in stephanos/temporal run 36945812832 at 38957053f1ce342a8797af1803f5f8f6bb53fcad. All steps passed on macos-15; the retained report identifies darwin/arm64 and seed 11, with all four workloads qualified and exact choice-tape replay matched, and zero unsupported, failed, infrastructure-error, or replay-diverged outcomes. The workflow and smoke manifest are byte-identical to that run. Retained job metadata, source hashes, and report in .flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d7-github-*.json. The overall workflow remains failed in Linux jobs; this closes only the D7 macOS job requirement. No new commit was made.
## Evidence
- Commits: 38957053f1ce342a8797af1803f5f8f6bb53fcad
- Tests: GitHub Actions: https://github.com/stephanos/temporal/actions/runs/36945812832/job/110647570973 (success), Downloaded macOS smoke artifact and reran the workflow exact-replay evidence predicate (true), Compared current workflow and smoke manifest bytes with CI head (identical)
- PRs: