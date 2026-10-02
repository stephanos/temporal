---
satisfies: [R3, R7]
---
# fn-112-gomad-determinism-assurance-and-test.6 Add conformance fixtures for unverified channels and state the closure-mode limit

## Description
One seeded black-box fixture with a positive control per unverified channel, and the contract sentence on closure mode (R7).

**Size:** M
**Files:** new fixture directories under `tools/gomad3/internal/gomadtool/conformance/testdata/`, `runtime_campaign.go` and the `runtime_*.go` file owning each behavior
**Touches:** [tools/gomad3/internal/gomadtool/conformance/runtime_*.go, tools/gomad3/internal/gomadtool/conformance/testdata/netpoll/**, tools/gomad3/internal/gomadtool/conformance/testdata/sigprof/**, tools/gomad3/internal/gomadtool/conformance/testdata/profile_sampling/**, tools/gomad3/internal/gomadtool/conformance/testdata/numcpu/**, tools/gomad3/internal/gomadtool/conformance/testdata/timer_ties/**, tools/gomad3/internal/gomadtool/conformance/testdata/runq_shuffle/**]

### Approach
- First step (R3): re-anchor assessment findings Q7 and Q8.
- Channels: netpoll readiness, SIGPROF, block and mutex profile sampling, `runtime.NumCPU`, timer-tie draws, run-queue shuffle draws.
- Follow the `runqueue` fixture: a `main.go` under the shared testdata module, a build registration, and listing in the repeatability and load tables.
- Each fixture needs a positive control showing the detector can fail (disabled mode varies, or different seeds diverge).
- A channel that stays outside the contract gets a fixture proving it fails closed, or a contract sentence naming it. `NumCPU` cannot be varied on a CI host; if no control is possible, take the contract-sentence route and say why.
- Confirm from `target.go` that closure mode adds no `-gomadguard` flag, and write the exact contract sentences (closure-mode limit, and any channel placed outside the contract) into the done summary. Task 10 puts them into README and SPEC.
- Depends on task 2, which edits the same campaign registration file.
- Fixture files must match the `runtime_*.go` grouping rule in `architecture_test.go`.

### Investigation targets
**Required** (read before coding):
- `tools/gomad3/internal/gomadtool/conformance/testdata/runqueue/main.go` — fixture shape
- `tools/gomad3/internal/gomadtool/conformance/runtime_campaign.go:298` — build registration
- `tools/gomad3/internal/gomadtool/conformance/runtime_repeatability.go:17-104` — repeat and positive-control tables
- `tools/gomad3/target/target.go:711-716` — guard flags by capability mode

**Optional** (reference as needed):
- `tools/gomad3/internal/gomadtool/conformance/runtime_load.go:59-106` — load and seed-divergence checks
- `tools/gomad3/README.md:166-176`, `tools/gomad3/SPEC.md:204` — closure and guarded text

### Key context
- fn-105 task 12 Q4 also names `NumCPU`; share the finding.
- fn-114 E3 and E4 change which select and system-goroutine decisions are recorded; if they have landed, write timer and run-queue fixtures against that behavior.
## Acceptance
- [ ] Findings Q7 and Q8 re-anchored and marked confirmed, changed, or refuted
- [ ] Each named channel has a seeded fixture with a positive control, or a drafted contract sentence placing it outside the contract with the reason
- [ ] The closure-mode sentence and any out-of-contract channel sentences are in the done summary for task 10
- [ ] `make -C tools/gomad3 test-runtime` passes on darwin/arm64 with the new fixtures; linux status recorded
- [ ] A fixture that is nondeterministic on first run is recorded as a finding, not weakened
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
