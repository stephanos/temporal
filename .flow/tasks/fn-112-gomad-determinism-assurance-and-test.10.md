---
satisfies: [R6, R7, R11]
---
# fn-112-gomad-determinism-assurance-and-test.10 Add the scheduled determinism soak gate and update the docs to the delivered state

## Description
The soak gate (R6) and final documentation (R11). Combined because the docs report the bound the soak measures.

**Size:** M
**Files:** `Makefile` (root), `tools/gomad3integration/qualification/` (a soak manifest), `.github/workflows/gomad3.yml`, `tools/gomad3integration/README.md`, `tools/gomad3/README.md`, `SPEC.md`, `ARCHITECTURE.md`, `TUTORIAL.md`, `MILESTONES.md`
**Touches:** [Makefile, tools/gomad3integration/**, .github/workflows/gomad3.yml, tools/gomad3/*.md, MILESTONES.md, AGENTS.md, tools/gomad3/qualification/**]

### Approach
- Soak selection: the smoke suites plus one guarded-mode workload, seeds 11 and 17, choice tracing and diagnostics on, with unrelated CPU load from the existing load helper or an equivalent host-side loader.
- First measure per-run cost, then choose the largest N per seed that fits one scheduled job by looping `qualify` at its 32-repetition bound. Record N and the reasoning.
- Each `qualify` invocation compares only against its own first execution. The soak therefore compares the evidence baseline across batches within one cohort (workload, seed, platform, execution identity) and fails when two batches disagree. Cumulative counts across scheduled runs are aggregated per cohort only, with the cohort's baseline evidence retained, and a changed toolchain identity starts a new cohort.
- Test the aggregation with a batch sequence A, A then B, B: it must be a divergence.
- The gate accepts zero divergences. A trace overflow or infrastructure failure is reported separately and is not a pass.
- On a divergence, upload both diagnostic traces and the differ output.
- The report states repetitions, seeds, load, platform, toolchain identity, and a cumulative count across retained scheduled runs.
- linux/amd64 stays informational until fn-105 R12 closes D12, and the report says so. Coordinate the workflow edit with fn-105 task 12, which removes the D12 allowances.
- Docs: take the flag and subcommand names, the gate name, the inventory location, the declared-differences table, and the drafted contract sentences from the done summaries of tasks 2, 4, 5, 6, and 7. Update the milestones quality-assessment section, README Contract and Development, SPEC, ARCHITECTURE choice-trace section, and TUTORIAL determinism section to the delivered state, including the measured bound. Rerun fn-111's manual link and command-inventory checks; they have no Make target. Keep the `MILESTONES.md` headings `Quality assessment (2026-10-01)`, `Maintenance cost`, and `Constraints`, which specs link to.

### Investigation targets
**Required** (read before coding):
- `Makefile:192-222` — root qualification targets
- `.github/workflows/gomad3.yml:23-24`, `:136-161`, `:245-282` — cron and schedule-gated steps
- `tools/gomad3/internal/gomadtool/conformance/runtime_repeatability.go:295` — CPU load helper
- `tools/gomad3/cmd/gomad/internal/cli/qualify.go:20-105` — repeat bound
- `tools/gomad3integration/qualification/smoke.json` — selection to copy

**Optional** (reference as needed):
- `tools/gomad3/TUTORIAL.md:588-623` — determinism and qualification text
- `MILESTONES.md` section `Quality assessment (2026-10-01)`

### Key context
- The bound is measured with diagnostics on; say so wherever it is quoted.
- Spec Open Question 1 (target bound and runners) is unanswered; deliver the cumulative report and state the count reached.
## Acceptance
- [ ] A scheduled job on each platform runs the soak with stated N, seeds, load, and identities, and fails on any divergence on darwin/arm64
- [ ] The linux job reports its result as informational while D12 is open
- [ ] A divergence retains both traces and the differ output as workflow artifacts
- [ ] Overflow and infrastructure failures are reported separately from divergence
- [ ] Batches of one cohort are compared against each other; an A, A then B, B sequence is reported as a divergence by a test
- [ ] Cumulative counts are per cohort, and a new toolchain identity starts a new cohort
- [ ] One completed scheduled or dispatched run per platform is retained with its report
- [ ] Milestones, README, SPEC, CLI.md, ARCHITECTURE, TUTORIAL, and `AGENTS.md` describe the delivered gate, localiser flag and differ, inventory, fixtures, closure-mode limit, declared differences, and measured bound; links and command inventories checked
- [ ] `make -C tools/gomad3 validate` passes
## Done summary
Blocked:
Blocked: the soak gate and contract docs are delivered and reviewed (SHIP, claude-fable-5-1, round 2). Two items remain, and both need native CI: "one completed scheduled or dispatched run per platform is retained with its report", and the native measured bound that run supplies.

Done (commits bae373d14, 204732c33, 8ecc7e584, 4efa98213 on gomad-fn112; base 331b75bb6):
- `gomadtool soak` (tools/gomad3/qualification/soak) runs `qualify --diagnostics` batches of 32 under 2 busy host threads.
  - Per-cohort comparison: workload, seed, platform, and execution identity (a new toolchain build key starts a new cohort). The ledger is carried between runs.
  - Overflow, target, and infrastructure failures are reported apart from divergence and never count as a pass.
  - A divergence retains both traces, both evidence records, and the differ output.
  - The bound counts clean-batch repetitions only.
  - Rounds: always at least 2, at most 16, gated by the measured round cost against a 300m budget.
  - An A,A then B,B test reports a divergence.
- tools/gomad3integration/qualification/soak.json: the 4 smoke suites plus the guarded frontend-system-info, seeds 11 and 17, with the sizing reasoning recorded. linux/amd64 is informational under fn-105 D12.
- Root `make gomad3-soak`. gomad3.yml gains `determinism-soak-darwin` (strict) and `determinism-soak-linux` (informational): schedule/dispatch only, a workload x seed matrix, the ledger restored via `gh run download`, and uploads that use the existing pinned SHAs.
- Docs updated: README, SPEC (new RUNTIME.CHOICES.DIAGNOSTICS, RUNTIME.STREAMS, RUNTIME.TRUST.EXCLUSIONS, QUALIFICATION.SOAK, and gomadtool DIAGNOSTIC.DIFF/SOAK rows), CLI, ARCHITECTURE, TUTORIAL, AGENTS.md, MILESTONES, and the integration README.

Local evidence (linux/arm64, developmental only):
- `go test -tags test_dep ./qualification/soak`: 16 pass. Mutation check: removing the cross-batch divergence assignment fails 3 tests.
- With stock Go 1.27.1, `. ./cmd/gomadtool ./qualification/soak ./qualification/set`: pass, including TestPackageArchitecture and the vocabulary test.
- `make -C tools/gomad3 validate`: exit 0 (35s). actionlint: clean. go vet and gofmt: clean.
- fn-111 guide audit: no new errors relative to the pre-edit baseline.
- The soak was exercised end to end against a stand-in `gomad qualify`, because the patched toolchain refuses linux/arm64:
  - 2 clean runs, exit 0, ledger accumulating to 24 repetitions per cohort.
  - 1 injected divergence, exit 1, with traces retained and the differ reporting runtime_cheap_rand_draws.
  - No Gomad bound was measured.
- Evidence: .flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-10/handover.md.

Remaining native gates:
1. Dispatch or schedule gomad3.yml and retain one completed run of the determinism-soak-darwin and determinism-soak-linux jobs, with their reports and ledgers. Quote their per-cohort counts as the first measured bound (diagnostics on), and re-check the round sizing from execution_wall_nanos.
2. Run `make -C tools/gomad3 test-host` on darwin/arm64 and linux/amd64 covering the new packages.

Blocked:
Blocked: the soak gate and contract docs are delivered and reviewed (SHIP, claude-fable-5-1, round 2). Two items remain, and both need native CI: "one completed scheduled or dispatched run per platform is retained with its report", and the native measured bound that run supplies.

Done (commits bae373d14, 204732c33, 8ecc7e584, 4efa98213, 9bc745d1e on gomad-fn112; base 331b75bb6):
- `gomadtool soak` (tools/gomad3/qualification/soak) runs `qualify --diagnostics` batches of 32 under 2 busy host threads.
  - Per-cohort comparison: workload, seed, platform, and execution identity (a new toolchain build key starts a new cohort). The ledger is carried between runs.
  - Overflow, target, and infrastructure failures are reported apart from divergence and never count as a pass.
  - A divergence retains both traces, both evidence records, and the differ output.
  - The bound counts clean-batch repetitions only.
  - Rounds: always at least 2, at most 4 (N 64 to 128 per seed), gated by the measured round cost against a 120m per-job budget (cost cap, commit 9bc745d1e; delta review SHIP).
  - An A,A then B,B test reports a divergence.
- tools/gomad3integration/qualification/soak.json: the 4 smoke suites plus the guarded frontend-system-info, seeds 11 and 17, with the sizing reasoning recorded. linux/amd64 is informational under fn-105 D12.
- Root `make gomad3-soak`. gomad3.yml gains `determinism-soak-darwin` (strict) and `determinism-soak-linux` (informational): schedule/dispatch only, a workload x seed matrix, the ledger restored via `gh run download`, and uploads that use the existing pinned SHAs.
- Docs updated: README, SPEC (new RUNTIME.CHOICES.DIAGNOSTICS, RUNTIME.STREAMS, RUNTIME.TRUST.EXCLUSIONS, QUALIFICATION.SOAK, and gomadtool DIAGNOSTIC.DIFF/SOAK rows), CLI, ARCHITECTURE, TUTORIAL, AGENTS.md, MILESTONES, and the integration README.

Local evidence (linux/arm64, developmental only):
- `go test -tags test_dep ./qualification/soak`: 16 pass. Mutation check: removing the cross-batch divergence assignment fails 3 tests.
- With stock Go 1.27.1, `. ./cmd/gomadtool ./qualification/soak ./qualification/set`: pass, including TestPackageArchitecture and the vocabulary test.
- `make -C tools/gomad3 validate`: exit 0 (35s). actionlint: clean. go vet and gofmt: clean.
- fn-111 guide audit: no new errors relative to the pre-edit baseline.
- The soak was exercised end to end against a stand-in `gomad qualify`, because the patched toolchain refuses linux/arm64:
  - 2 clean runs, exit 0, ledger accumulating to 24 repetitions per cohort.
  - 1 injected divergence, exit 1, with traces retained and the differ reporting runtime_cheap_rand_draws.
  - No Gomad bound was measured.
- Evidence: .flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-10/handover.md.

Remaining native gates:
1. Dispatch or schedule gomad3.yml and retain one completed run of the determinism-soak-darwin and determinism-soak-linux jobs, with their reports and ledgers. Quote their per-cohort counts as the first measured bound (diagnostics on), and re-check the round sizing from execution_wall_nanos.
2. Run `make -C tools/gomad3 test-host` on darwin/arm64 and linux/amd64 covering the new packages.
## Evidence
- Commits:
- Tests:
- PRs:
