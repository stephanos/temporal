---
satisfies: [R2, R3, R4, R5, R6, R7]
---
# fn-14-milestone-a-pilot-baseline-and-lean.5 Compose the strict evidence bundle and pilot decision commands

## Description
Compose provider-free and Agentworkflow evidence into one strict recomputable receipt and observable root command surface for R2-R7.

**Size:** M
**Files:** `tools/umpire/pilot/run.go`, `tools/umpire/pilot/run_test.go`, `tools/umpire/pilot/progress.go`, `tools/umpire/pilot/progress_test.go`, `tools/umpire/pilot/decision/receipt.go`, `tools/umpire/pilot/decision/receipt_test.go`, `tools/umpire/cmd/umpire-pilot/main.go`, `Makefile`
**Touches:** [tools/umpire/pilot/run.go, tools/umpire/pilot/run_test.go, tools/umpire/pilot/progress.go, tools/umpire/pilot/progress_test.go, tools/umpire/pilot/decision/receipt.go, tools/umpire/pilot/decision/receipt_test.go, tools/umpire/cmd/umpire-pilot/main.go, Makefile]

### Approach

- Validate and publish the canonical evidence bundle as one lock-protected, all-or-nothing set; bind every member through the payload digest.
- Implement strict `ReadBundle`, recompute metrics/gates/outcome, and expose qualification authorization only for `LEAN_FIRST_GO`.
- Apply the exact decision precedence without caller override: invalid evidence, then core gates, then ergonomics gates.
- Emit structured `umpire-pilot-progress/v1` JSON Lines on stderr for phases, mutation samples, trial attempts, terminal transitions, and a heartbeat at least every 30 seconds while a child is active. Keep progress outside measured child streams and canonical evidence.
- Add root-only run/check/verify targets. Check and verify consume retained evidence without provider calls or mutation; run refuses an existing v1 evidence directory.
- Reject unsafe paths, symlinks, unsupported schemas, unknown/duplicate fields, digest drift, mixed source/config identities, partial concurrent publication, and declared/derived gate disagreement.

### Investigation targets

**Required:**
- `tools/common/artifactio/set.go:16-103` — transactional set/path/lock pattern.
- `tools/agentworkflow/internal/agentworkflow/evidence.go` — strict trial export input.
- `tools/umpire/regression/projection.go:30-63` — strict metadata-only boundary language.
- `Makefile:988-1032` — root Umpire targets and propagation style.

### Quick commands

`go test -count=1 -tags test_dep ./tools/umpire/pilot/... && make umpire-pilot-check`

## Acceptance

- [ ] Strict read/write round trips preserve the exact canonical bundle and reject every schema/path/hash/member-set corruption class.
- [ ] Gate and outcome recomputation cannot be overridden; only `LEAN_FIRST_GO` returns qualification authorization.
- [ ] Concurrent/interrupted publication never exposes a partial or mixed bundle and v1 evidence is not overwritten.
- [ ] Fake-clock/process tests prove every phase/unit transition and the at-most-30-second heartbeat appear on stderr without contaminating canonical evidence or measured command output.
- [ ] Root Make run/check/verify commands validate required variables and propagate all failures; no model-local Makefile or CI workflow is added.
- [ ] Check/verify make no provider call, run no mutation, and do not alter retained evidence.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
