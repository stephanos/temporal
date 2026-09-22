---
satisfies: [R2, R3]
---
# fn-33-run-serial-bounded-semantic-exploration.3 Prepare and run one candidate through the Case Runtime against a bound deployment

## Description
Implement the serial path from an outstanding canonical Case through `testpilot.Prepare`, one fresh `PreparedCase.Run` against the deployment binding `umpire-run` performs, cleanup, closed Verdict, and bridge observation. Preserve exact identities and separate preparation rejection from Run outcomes.

**Size:** L
**Files:** `tools/umpire/binding/**`, `tools/umpire/campaign/bridge.go`, `tools/umpire/campaign/run.go`, `tools/umpire/campaign/run_test.go`
**Touches:** [tools/umpire/binding/**, tools/umpire/campaign/bridge.go, tools/umpire/campaign/run.go, tools/umpire/campaign/run_test.go, tools/umpire/cmd/umpire-run/run.go]

### Approach
- `bridge.go`: the Go side of the frames (spawn `umpire-explore`, one request outstanding, exact identity echo, byte caps on frames).
- Split `umpire-run`'s `openSession` (today one closure that dials, provisions, derives the Profile, prepares and opens the Driver) into campaign-scoped binding (dial, provision, catalog, release once) and candidate-scoped work (Profile, `Prepare`, Driver open, worker, release per candidate), in a neutral package `tools/umpire/binding` that both `umpire-run` and the campaign consume (the campaign package is never imported by `umpire-run`), without changing `umpire-run`'s behavior or exit codes.
- `run.go`: `prepare-rejected` is decided before the Driver opens and creates no Run; one Run; cleanup observed before `observe`; a Run that is not closed is reported as such and credits nothing.
- Tests use the facade's external-driver test doubles for Prepare/Run/cleanup outcomes; no test cluster in unit tests. One integration test under `-tags 'test_dep integration'` runs the proof point end to end against the development cluster; it is non-gating for this task's receipt and reported as run or not run.

### Investigation targets
**Required** (read before coding):
- `common/testing/testpilot/prepare.go:25`, `prepared_case.go:11`, `driver.go:120` — the public facade.
- `tools/umpire/cmd/umpire-run/run.go:60-260` — the binding to lift.
- `common/testing/testpilot/facade_external_test.go` — cleanup-failure semantics a Run preserves.

### Quick commands
`go test -count=1 -tags test_dep ./tools/umpire/campaign/... ./tools/umpire/cmd/umpire-run/... && GOLANGCI_LINT_BASE_REV=<base> make lint-code-fast`

### Re-plan note (2026-09-21)
Re-planned on fn-85's exploratory set after fn-86 R6 deleted the variation Space this task was first written against; see the spec's **Re-plan on fn-85** section. Start only after the spec's fresh plan review.
## Acceptance
- [ ] Exactly one Prepare or Run is outstanding at any time; the bridge client refuses a second `next` before `observe` (a local guard, pinned by a test; `.6` owns the coordinator's state machine).
- [ ] `prepare-rejected`, Run failure, cleanup failure and a decisive Verdict are distinguishable observations with the exact candidate identity.
- [ ] `umpire-run` keeps its behavior and exit codes; the shared binding has no flag that widens a declared Limit.
- [ ] The integration proof point, when a cluster is available, runs one candidate and credits its planned path; the receipt says whether it ran.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
