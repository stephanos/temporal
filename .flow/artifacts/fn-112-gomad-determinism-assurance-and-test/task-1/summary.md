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
