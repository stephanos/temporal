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
- Kill test: start `explore` with `--parallel 1`, wait until the journal shows a fixed number of completed executions, send SIGKILL to the coordinator, run `resume`, and compare with an uninterrupted run of the same seeds. Compare decoded execution records (seed, selection ordinal, outcome, output hashes, transcript and World identities) and semantic summary counts. Normalize Campaign IDs and artifact references. Exclude wall-time fields (including the recorded remaining overall timeout), the record hash derived from that timeout, journal segment counts, and journal hashes, which legitimately differ after recovery. Physical retained byte counts can differ because timestamp and deadline text has variable length; validate each Campaign's byte accounting and storage integrity separately with `inspect` instead of requiring those byte totals to match. Keep execution/retention counts, non-wall resource limits, and all semantic record fields equal.
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
Built-CLI explore/replay and coordinator recovery tests are implemented and independently reviewed SHIP. They exercise a known failing seed, successful and failed replay, invalid replay, published-campaign resume rejection, and SIGKILL after exactly two or zero journaled executions. Each store passes inspect; full decoded execution evidence and semantic campaign fields match the uninterrupted run under the stated wall-time/storage exclusions. Journal schema, record count and non-wall capacity limits remain compared.

Review round 1 found that the initial normalization dropped journal limits. The repair adds a real-baseline MaximumBytes negative control, demonstrated red before the correction and green afterward. Round 2 returned SHIP with R9 met and no surviving findings (gpt-6-astra high, session 01a0fc88-8ceb-77a1-a106-4644b2c68f23).

Final checks: twenty consecutive CLI runs passed in 235.732s, including twenty capacity controls, forty exact-boundary coordinator kills and 120 execution comparisons. Full test-host passed on darwin/arm64 with explicit pinned stock Go 1.27.1; focused vet and formatting passed. The retained twelve-command CLI sample has independently verified statuses and all twenty-one payload hashes. Owned subprocess cleanup passed. No production code or toolchain identity changed.

Limits: native Linux is unverified; root lint fails on missing main and nested-module discovery. The fixture compares the valid None World identity and a nonempty filesystem transcript. The normal host-test compiler-selection gap is tracked by fn-112.12, and missing I/O terminal handling after a watchdog kill by fn-112.11; neither is claimed fixed here.

Evidence: review-fix-1 contains the final full patch, incremental correction, frozen sources, red/green logs and handover. Original round-1 evidence remains immutable. All 60 protected prior source files remain unchanged. User owns commits; no staging, commit, push, stash or worktree was used.

stage: impl-review - ran (model: gpt-6-astra at high)
stage: plan-sync - skipped(config: planSync.enabled=false)
stage: wave - skipped(policy: shared dirty checkout and user forbids worktrees)
Tracker sync: n/a (bridge inactive)
## Evidence
- Commits:
- Tests: env -u GOROOT -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -tags test_dep ./cmd/gomad -run ^TestCLI -count=20 -v: PASS, 235.732s, make -C tools/gomad3 test-host with GOMAD3_STOCK_GO=/Users/stephan/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin/go and cleared GOROOT/Gomad seeds: PASS, 45 package lines, patched go vet -tags test_dep ./cmd/gomad ./cmd/gomad/testdata/campaign: PASS, stock Go 1.27.1 gofmt -l on both task code files: PASS, make lint-code-fast GOLANGCI_LINT_FIX=false: FAIL, missing main; HEAD override: FAIL, nested-module discovery
- PRs: