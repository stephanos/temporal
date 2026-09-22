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
- [x] Exactly one Prepare or Run is outstanding at any time; the bridge client refuses a second `next` before `observe` (a local guard, pinned by a test; `.6` owns the coordinator's state machine).
- [x] `prepare-rejected`, Run failure, cleanup failure and a decisive Verdict are distinguishable observations with the exact candidate identity.
- [x] `umpire-run` keeps its behavior and exit codes; the shared binding has no flag that widens a declared Limit.
- [x] The integration proof point, when a cluster is available, runs one candidate and credits its planned path; the receipt says whether it ran.
## Done summary
The serial path from an outstanding Case to a bridge observation. `tools/umpire/binding` lifts the deployment binding out of `umpire-run`: `Open` binds a deployment once (frontend connection, provisioning when asked, method catalog) and `Campaign.Bind` binds one Case (derived Profile under the identity named, `Prepare`, SDK client, composite Driver), each released in reverse order with a bounded budget per resource; `umpire-run` binds one Case through both and keeps its behavior, messages and exit codes, and its dependency pins still hold. `tools/umpire/campaign.Bridge` is the client of `umpire-explore`: `Initialize`, `Next`, `Observe`, `Finish`, every reply matched to its frame by sequence, set and profile, a second `Next` refused while a candidate is outstanding and a crossed `Observe` refused before any frame is written, `rejected` replies leaving the campaign untouched, frames capped in both directions. `RunCandidate` decodes the candidate's Case, binds it (a preparation rejection is observed as `prepare-rejected` before any Driver opens and creates no Run), runs it once, releases, and hands the closed Run with its observed cleanup back to the bridge, returning what the bridge credited; a binding failure that is not the Case's own and a Run that could not execute leave the candidate outstanding as `bind-failed`/`run-failed`. Tests drive the client against a scripted fake bridge and fake binders (guard, crossed, prepare-rejected, bind failure, run failure, cleanup-failed, decisive, whole campaign, rejected and mismatched replies, oversized frames, context deadline) and the real `umpire-explore` when built; the binding's tests pin preparation-before-Driver against an unreachable deployment. The integration proof (`-tags 'test_dep integration'`, cluster from `UMPIRE_FUZZ_GRPC`/`UMPIRE_FUZZ_HTTP`) did not run in this session: no cluster was available, and the test skips saying so.
## Evidence
- Commits: 494c2e3ab4d53a5b35978d68d189f4f12c36cdc1, 273c0b63518d3f7185fd48e250150467fc385e7f, 8485646a17d020a448f722fde196d1a730ef0e71, 55f58c75f0aaca4cd61adb7fe3c9c70070b6e2ef
- Tests: go test -count=1 -tags test_dep ./tools/umpire/campaign/... ./tools/umpire/binding/... ./tools/umpire/cmd/umpire-run/..., go vet -tags 'test_dep integration' ./tools/umpire/campaign/, GOLANGCI_LINT_BASE_REV=HEAD~1 make lint-code-fast, integration proof (UMPIRE_FUZZ_GRPC/UMPIRE_FUZZ_HTTP): not run, no development cluster in the session
- PRs: