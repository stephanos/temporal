---
satisfies: [R2, R3]
---
# fn-33-run-serial-bounded-semantic-exploration.3 Prepare and run one candidate through the Case Runtime against a bound deployment

## Description
Implement the serial path from an outstanding canonical Case through `testpilot.Prepare`, one fresh `PreparedCase.Run` against the deployment binding `umpire-run` performs, cleanup, closed Verdict, and bridge observation. Preserve exact identities and separate preparation rejection from Run outcomes.

**Size:** L
**Files:** `tools/umpire/campaign/bridge.go`, `tools/umpire/campaign/run.go`, `tools/umpire/campaign/run_test.go`
**Touches:** [tools/umpire/campaign/bridge.go, tools/umpire/campaign/run.go, tools/umpire/campaign/run_test.go, tools/umpire/cmd/umpire-run/run.go]

### Approach
- `bridge.go`: the Go side of the frames (spawn `umpire-explore`, one request outstanding, exact identity echo, byte caps on frames).
- Lift `umpire-run`'s deployment binding (`openSession`: dial, namespace, task queue, handler queue, Nexus endpoint, release) into a package both commands use, without changing `umpire-run`'s behavior or exit codes.
- `run.go`: prepare the Case with the Profile it implies; `prepare-rejected` creates no Run and is observed as such; one Run; cleanup observed before `observe`; a Run that is not closed is reported as such and credits nothing.
- Tests use the facade's external-driver test doubles for Prepare/Run/cleanup outcomes; no test cluster in unit tests; one integration test under `-tags 'test_dep integration'` runs the early proof point end to end.

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
- [ ] Exactly one Prepare or Run is outstanding at any time; a second `next` before `observe` is impossible by construction and pinned by a test.
- [ ] `prepare-rejected`, Run failure, cleanup failure and a decisive Verdict are distinguishable observations with the exact candidate identity.
- [ ] `umpire-run` keeps its behavior and exit codes; the shared binding has no flag that widens a declared Limit.
- [ ] The integration proof point runs one candidate against the development cluster and credits its targets.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
