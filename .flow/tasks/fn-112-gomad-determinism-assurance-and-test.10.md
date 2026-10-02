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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
