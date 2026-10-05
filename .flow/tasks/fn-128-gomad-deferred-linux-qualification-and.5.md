---
satisfies: [R5]
---
# fn-128-gomad-deferred-linux-qualification-and.5 Run and retain the actual Linux determinism soak

## Description
Implements R5. Transfer source: fn-112.10 / R6/R11 and Linux test-host requirement.

**Size:** M
**Files:** .github/workflows/gomad3.yml, tools/gomad3/qualification/soak/**, tools/gomad3/cmd/gomadtool/soak.go, Linux soak reports/ledgers
**Touches:** [.github/workflows/gomad3.yml, tools/gomad3/qualification/soak/**, tools/gomad3/cmd/gomadtool/soak.go, Linux soak reports/ledgers]

### Approach
Execute the scheduled/dispatched workflow on native Linux with the original task-10 cohort, load, diagnostics, overflow and repetition settings. Retain the report and ledger and derive the measured Linux bound only from completed cohorts. Preserve informational Linux disposition until task 2 passes; refresh invalidated source-bound evidence for the final matrix.

### Investigation targets
**Required:**
- `.flow/tasks/fn-112-gomad-determinism-assurance-and-test.10.md`
- `.github/workflows/gomad3.yml`
- `tools/gomad3/qualification/soak/soak.go`

### Revival prerequisites
Owner requests qualification and supplies a native linux/amd64 host or CI and a pinned supported toolchain/source candidate. This task stays deferred until those prerequisites exist. Retain original policy and exact command/evidence requirements from the transfer manifest and source clauses at commit 10d884c6f9d97681d08aaf2636f5850407f1586a. Commit verified task progress separately; do not convert an unavailable gate into a pass.

## Acceptance
- [ ] An actual completed native Linux soak report and ledger retain all required cohort counts, source/build identities and the measured bound. Stand-in repetitions and source checks supply no bound. Missing runs, overflow omissions or stale identities leave acceptance open.
- [ ] Retain exact commands, exit codes, source/build identities and an independent review for accepted changes; commit task progress before completing it.

## Done summary
Blocked:
Deferred by owner-authorized Linux scope transfer on 2026-10-04. No native linux/amd64 host or CI is available. Revival requires an owner request, native execution, pinned supported toolchain/profile and a frozen source candidate. This task owns R5; Linux evidence is incomplete and no pass or waiver is claimed.
## Evidence
- Commits:
- Tests:
- PRs:
