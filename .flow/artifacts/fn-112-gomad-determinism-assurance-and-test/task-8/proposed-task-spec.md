---
satisfies: [R9]
---
# fn-112-gomad-determinism-assurance-and-test.8 Drive explore, replay, and kill-then-resume through the built CLI

## Description
End-to-end CLI tests (R9) through real processes: `explore` then `replay`, and a coordinator killed mid-campaign, resumed, and compared with an uninterrupted run.

**Size:** M
**Files:** new `tools/gomad3/cmd/gomad/e2e_test.go` and a small fixture target under `tools/gomad3/cmd/gomad/testdata/`
**Touches:** [tools/gomad3/cmd/gomad/e2e_test.go, tools/gomad3/cmd/gomad/testdata/**]

### Approach
- Build `cmd/gomad` once per test binary and invoke it as a subprocess against a small deterministic fixture target with the patched toolchain.
- `explore` over a seed range with one known-failing seed, then `replay` the retained artifact and assert the exit status table from the README.
- Kill test: start `explore` with `--parallel 1`, wait until the journal shows a fixed number of completed executions, send SIGKILL to the coordinator, run `resume`, and compare with an uninterrupted run of the same seeds. Compare decoded execution records (seed, selection ordinal, outcome, output hashes, transcript and World identities) and semantic summary counts. Normalize Campaign IDs and artifact references. Exclude wall-time fields, journal segment counts, and journal hashes, which legitimately differ after recovery. Validate each Campaign's storage integrity separately with `inspect`.
- Cover the edges: kill before the first completed execution, and resume of an already published Campaign (must fail closed).
- Poll journal state with bounded ticker/context polling and the nested module's existing plain `testing` style; no `time.Sleep` or new dependencies. The nested module does not include testify, so `require.Eventually` is unavailable.

### Investigation targets
**Required** (read before coding):
- `tools/gomad3/runner/coordinator_transport_test.go:27-31`, `:178-250` — real-subprocess seam
- `tools/gomad3/cmd/gomad/internal/cli/resume.go:14-23` — resume entry
- `tools/gomad3/cmd/gomad/internal/cli/cli.go:125` — command dispatch
- `tools/gomad3/runner/internal/campaign/filesystem_fault_test.go:220` — existing fault-injected resume test

**Optional** (reference as needed):
- `tools/gomad3/README.md:436-455` — resume contract and status table

### Key context
- fn-109 tasks 2 to 5 change option handling and CLI construction. These tests use only the public CLI, so they double as that migration's behavior pin; check overlap before starting.
- `resume` is on the list of features whose fate is undecided (spec Open Questions 4).
- A missing `.toolchain` is fatal in existing tests; follow that convention.
## Acceptance
- [ ] `explore` then `replay` run through the built binary with asserted exit statuses and retained artifact
- [ ] A coordinator killed after a fixed number of journaled executions resumes to a Campaign whose decoded execution records and summary counts equal the uninterrupted run under the stated normalization; both Campaigns pass `inspect` validation
- [ ] Kill before the first completed execution and resume of a published Campaign are covered
- [ ] No wall-clock sleep; the test passes 20 consecutive runs locally
- [ ] `make -C tools/gomad3 test-host` passes on darwin/arm64; linux status recorded
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
