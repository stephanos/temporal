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
## Acceptance
- [ ] Listed packages gone; ir and execution shrunk; cel-go out of go.mod
- [ ] Case trees and seven legacy failures removed; umbrella green
- [ ] Per-package counts reported; targets met or exception recorded

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
