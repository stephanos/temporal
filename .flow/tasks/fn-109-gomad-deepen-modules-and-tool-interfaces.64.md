---
satisfies: [R18, R19]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.64 Check watchdog fixture readiness writes without masking setup failure

## Description
Correct the last observed unchecked child-fixture readiness write without changing production execution or successful watchdog/cancellation behavior. See [the bounded admission](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-64/admission.md).

**Touches:** [tools/gomad3/runner/internal/execution/watchdog_io_test.go, tools/gomad3/runner/internal/execution/watchdog_fixture_output_test.go, .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-64/**]

Use a checked readiness write and immediate fixture exit 3 on failure; retain terminal write/close, signal handling and healthy wait ordering. Test actual read-only-stdout child execution in both watchdog and cancelled modes. Existing parent assertions and default production execution are untouched. This corrective owner is independent of further local Runner redesign and supports fn-112.10/fn-109.63 acceptance without relaxing it. Root owns review/lifecycle/integration. Shared execution gates are serialized; no native/CI/PR/push authority.

Quick commands (pinned documented Go environment; all tests include `test_dep`): focused `go test -tags test_dep -count=1 ./runner/internal/execution -run '^(TestWatchdogFixtureReadinessWriteFailure|TestRunIOTerminalAfterTermination)$'`, configured unfiltered package lint, affected vet/errortype, formatting and `make lint-code-fast`. Retain the original failing regression and actual source/tool/command bindings. Aggregate original-base lint is measured once at the frozen integration batch, not claimed green by a scoped command.

## Acceptance
- [ ] A real subprocess regression first fails on the uncorrected readiness write for the intended undelivered-readiness behavior, then passes with prompt child exit status 3 for both watchdog and cancelled modes on read-only stdout; actual EBADF, unchanged input and no supplemental output are covered.
- [ ] Successful readiness bytes, terminal-frame write/close ordering, SIGTERM-ignore placement, wait behavior and all existing TestRunIOTerminalAfterTermination assertions remain unchanged and pass.
- [ ] Only the admitted unchecked fixture write is corrected; no production/public API, parent assertion, lint configuration, suppression, global test hook or unrelated helper change.
- [ ] Focused tests, configured unfiltered affected-package lint, affected vet/errortype, formatting and repository fast lint pass on the frozen candidate. Historical aggregate failures remain accurately reported and are not converted into qualification by these scoped gates.
- [ ] Fresh independent review accepts the bounded correction after green task-owned source gates; integration preserves other candidate inputs and unrelated user changes. Original-base aggregate lint comparison records actual removals/additions without waiving remaining findings or altering fn-112.10/fn-109.63 acceptance.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
