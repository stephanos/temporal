---
satisfies: [R6]
---
# fn-228-runner-and-run-the-one-go-host-loop.6 After .4 and .5: retire legacy Testpilot packages and verify replacement coverage and size

## Description
Remove internal/verification, evaluation, cel, predicate, the IR CEL bridge and cel-go, contract, casefile, campaign and replay; shrink internal/ir to instruction decoding and internal/execution to act-and-observe; delete generated Case trees and the seven legacy failing tests, replacing their coverage with Runner questions over the activity Model. Umbrella green. Report non-test lines per package before and after, excluding generated v1; targets at most 5,000 generic and 11,000 total, else owner exception in writing. Runs after .5 because manifests and integration wiring overlap.

### Required integration contract

Map every removed feature/test to the parent R6 replacement coverage before deleting; deleted tests cannot supply green-gate credit. Include the new generic runner in the generic-runtime total and its adapters in total counts to prevent relocation-only compliance.

**Touches:** common/testing/testpilot/**, tools/umpire/**, tools/canary/**, tests/testcore/testpilot/**, model/umpire/hosts/go/umpire/runner/**, go.{mod,sum}, model/umpire/hosts/go/go.{mod,sum}, Makefile and generated Case trees. Runs after .5; re-anchor imports/manifests after its handoff.

### Carried from fn-228.3 (2026-10-23)

See `.flow/tmp/fn-228.3/retired-coverage.md`. In addition to the original scope:
- Reduce `tools/canary` to `controller` lease and fencing plus `policy`, and have it publish Runs with `runner.Publish`.
- Retire `common/testing/testpilot/publish` and its eight listed importers.
- Remove stale prose that still names the deleted commands, the inert `UMPIRE_REPEAT_RUN_DIR` hook, the pinned control record and the `UMPIRE_CONTROL_RECORD` hook.
- Map the coverage of the retired Nexus control and the IR-bridge minimization, which have no Runner equivalent.
- Four live failures pre-exist at 0c7822bc43 and must be retired or mapped together with the seven legacy failures: `TestTestpilotExplorationDiscoversUnpinnedExecution` and `TestTestpilotGeneratedCases` (both read the retired Scala-era tree), `TestTestpilotNexusControlForgedCompletionIsViolated` (malformed contract) and `TestTestpilotNexusPairCase` (`observe_failed` dereference).

### Dependency change (2026-10-23)

With fn-228.5 deferred by the owner, this task depends on .4 only. Its coverage map and size counts exclude the Shim, which arrives with .5.
## Acceptance
- [ ] Listed packages gone; ir and execution shrunk; cel-go out of go.mod
- [ ] Case trees and seven legacy failures removed; umbrella green
- [ ] Per-package counts reported; targets met or exception recorded

## Done summary
Implemented R6. Retired the legacy Testpilot packages listed in the spec (verification, evaluation, CEL and predicate, the IR CEL bridge, contract, casefile, campaign, replay, publish), deleted `internal/ir` and `internal/execution` outright (the Temporal Driver needs neither), the generated Case trees, Case lowering, conformance, the IR bridge, `tools/umpire/lint` (its only input went with Case lowering) and cel-go/antlr from go.mod; no dependency added. The canary runs the committed activity Model through the Runner: each iteration fences a fresh activity on its lease, publishes with `runner.Publish`, and pins the guest hash in policy (a test asserts the pin equals the committed guest). The fence admits only start, cancel and describe calls and describe/poll reads on the fenced activity, unwrapping read instructions. The Temporal Driver gained `Options.Transport` for TLS and API-key dialing.

Coverage: the coverage map (.flow/artifacts/fn-228.6/coverage-map.md) maps all 25 retired live tests, the seven legacy failures and the four pre-existing live failures to retained Runner coverage, and states the gaps with no Runner equivalent (Nexus, Workflow, workflow-scheduled activities, heartbeats, attempt timeout, IR-bridge minimization). `TestTestpilotModelSlice` is kept as the only live coverage of CancelAfterCompletion, CancelBeforeWorkerStart and Annotations, now required by the slice gate.

Size (non-test Go, generated v1 excluded and reported at 5,734): generic runtime 23,117 -> 1,647 (target 5,000); total Testpilot plus Runner 33,851 -> 3,317 (target 11,000); 6,387 including canary and `cmd/umpire`. The Host the Runner imports (624) is not counted; targets hold with it.

Review: fresh-context round (Opus, same family) NEEDS_WORK (fence bypass via wrapped or SignalWithStart requests, unchecked guest-hash pin, stale docs, lost symlink tests, thin credential tests, slice gate list); all fixed in d06dba6af6 and 61a00542e2 with regression tests. Conductor decisions: published Runs may carry namespace and task queue (not secrets, private store), with a live scan that the gRPC target and credentials never reach the output; the two dropped live canary tests stay mapped. No second review round (owner wrap-up instruction).

Gates: `make umpire-check-live-tests` rc 0, `make umpire-check-model-slice` rc 0, default `make model-check` rc 0, `UMPIRE_BACKENDS=require make model-time` rc 0 at d23625a390 (gate 571.968 s against 60 s, inner loop 8.864 s against 10 s, quiet machine). After the review fixes: focused canary/act/testpilot tests, -race on canary, golangci-lint on changed packages (no findings in changed files), the slice gate with four newly required subtests and the live canary tests, all rc 0. Open: AGENTS.md still describes Case tooling (fn-244 updates it); the remaining `tools/umpire` IR reader (about 17k lines) has no live input and keeps CI's `go test ./tools/umpire/...` red, a follow-up for a later spec.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 96d635bc17, 7a2eb2458b, 166d864a6f, e689481277, da7c84d64d, d06dba6af6, 61a00542e2
- Tests: make umpire-check-live-tests, make umpire-check-model-slice, make model-check, UMPIRE_BACKENDS=require make model-time
- PRs: