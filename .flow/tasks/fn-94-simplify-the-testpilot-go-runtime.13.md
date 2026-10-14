---
satisfies: [R6]
---
# fn-94-simplify-the-testpilot-go-runtime.13 One WorkflowBinding and one admit path in delivery and the worker

## Description
Lane D2's binding and admit items: one comparable, JSON-tagged `delivery.WorkflowBinding` replaces five representations, one extraction function replaces the two ways the binding is read from a request, and the admit paths share one tail and one fan-out loop. The Session admission caches stay (replay after `Session.Close` and once-only completion live there).

**Size:** M
**Files:** `common/testing/testpilot/temporal/internal/delivery/{codec,ledger,carrier}.go`, `temporal/worker/{api,carrier,routing,session}.go`, `temporal/driver.go`, their tests
**Touches:** [common/testing/testpilot/temporal/internal/delivery/**, common/testing/testpilot/temporal/worker/api.go, common/testing/testpilot/temporal/worker/carrier.go, common/testing/testpilot/temporal/worker/routing.go, common/testing/testpilot/temporal/worker/session.go, common/testing/testpilot/temporal/worker/*_test.go, common/testing/testpilot/temporal/driver.go, common/testing/testpilot/temporal/driver_test.go]

### Approach
- Unify `codec.go:29` `binding`, `ledger.go:37` `WorkflowBinding`, `worker/api.go:48` `WorkflowBinding`, `routing.go:16` `workflowRouteIndex`, and the binding half of `carrier.go:41` `startRequestFields` (the header stays beside it) into one `delivery.WorkflowBinding` with today's `codec.go` field order and JSON tags; it is the route key.
- Extraction: requests are `*dynamicpb.Message` values built by `ir.BuildRequest`, so a generated-type assertion would reject them. Keep one exported `delivery` function that reads the binding from a StartWorkflow request by descriptor (today's `startFields`, `delivery/carrier.go:328`, generalized to return `WorkflowBinding`), and make the composite Driver (`temporal/driver.go:228-244`) call it instead of its Marshal/Unmarshal round trip. Keep the call order: `CreateCarrier` needs the binding before `PrepareRPC` receives the bundle. Delete the type conversions (`worker/carrier.go:48`, `ledger.go:215,244`).
- Admit: `AdmitWorkflow`/`AdmitNexus` (`delivery/carrier.go:111,241`) share one consume-and-admit tail, keeping each one's stop/terminal/replay check order; `Driver.admitWorkflow`/`admitNexus` (`routing.go:103,259`) share one fan-out loop. The Session caches (`session.go:53-54`) and `workflowAdmissionLocked`'s replay-before-shutdown order stay; `TestAdmittedWorkflowUsesImmutableNexusDispatchAfterStop` must pass unchanged. Before touching the admit paths, add a `-race` test pinning concurrent replay and completion of one admission (completion happens once).
- The fn-94.2 route golden must pass unedited.

### Investigation targets
**Required:**
- `common/testing/testpilot/temporal/internal/delivery/codec.go:20-130`
- `common/testing/testpilot/temporal/internal/delivery/carrier.go:30-60,100-130,235-340`
- `common/testing/testpilot/temporal/worker/routing.go:10-30,95-300`
- `common/testing/testpilot/temporal/driver.go:220-250`

### Quick commands
```sh
go test -race -count=3 -tags test_dep ./common/testing/testpilot/temporal/...
make umpire-check-live-tests
make lint-code-fast
```

## Acceptance
- [ ] One `delivery.WorkflowBinding`; one extraction function serves every caller; no Marshal/Unmarshal round trip of the binding remains.
- [ ] The route golden passes unedited; admit paths share one tail and one fan-out loop.
- [ ] Replay after `Session.Close` still succeeds (existing test unchanged), and the new concurrent replay/completion test passes under `-race`.
- [ ] `-race -count=3` tests pass; live tests pass or are reported not run with the reason; lint passes.


## Done summary
One comparable, JSON-tagged `delivery.WorkflowBinding` now replaces the codec's `binding`, the ledger's and the worker's `WorkflowBinding`, the worker's `workflowRouteIndex`, and the binding half of `startRequestFields`, and it serves as the route key. The route golden (`codec_test.go`) is byte-identical to dd3b7bf278; a test-only alias `binding = WorkflowBinding` in `delivery_test.go` keeps it compiling. `delivery.StartBinding` reads the binding and header by descriptor, and both `PrepareRPC` and the composite Driver call it. The composite Driver's Marshal/Unmarshal round trip is gone. Its `carrierBinding` keeps the old rejection of empty binding fields with the composite `ErrInvalid`: the review found that this check had been dropped, and it is now restored and pinned by `TestCarrierBindingRejectsIncompleteStartRequests`. `CreateCarrier` still runs before `PrepareRPC`.

`AdmitWorkflow` and `AdmitNexus` share `consumeLocked`, and `Driver.admitWorkflow` and `admitNexus` share `admitFirst`. The Session caches are unchanged, and `TestAdmittedWorkflowUsesImmutableNexusDispatchAfterStop` passes unedited. The new `TestConcurrentReplayAndCompletionOfOneAdmission` landed before the admit changes and was checked against two mutations. Bypassing the Session cache made it fail with ErrRouteStale, and removing the completion lock made `-race` fail. `TestStartBindingReadsDynamicStartRequestsOnly` pins R6's extraction rejections, and it was run red first.

The worker Driver (`worker/driver.go`, outside this task's Touches and being edited by fn-94.12) still names `workflowRouteIndex`, so that name stays as an alias of `delivery.WorkflowBinding`. Follow-up: drop the alias once that file is free.

baseline: green (race suite, run with -overlay because other workers' uncommitted edits broke the shared build)
Live tests ran at b85202b97d, before the review fix, with 45 passing identities and 0 failing. That equals the fn-94.2 count. They ran with -overlay and the existing model/.lake binaries, without the lake prelude.

stage: impl-review - ran [codex fan-out rid 8d9caab4c5e74f3580eff1707f919315: correctness SHIP, contracts NEEDS_WORK, integration SHIP -> fix 940a020b0a -> re-review SHIP]
## Evidence
- Commits: b85202b97d920c681d57cf51b37ab2bf45d00c79, 940a020b0aade27117c9e2823d32afd59751dce6
- Tests: go test -race -count=3 -tags test_dep ./common/testing/testpilot/temporal/... (with -overlay pinning other workers' uncommitted files to HEAD; green pre-edit and at 940a020b0a), make umpire-check-live-tests (Go step only, with -overlay, existing model/.lake binaries; run at b85202b97d: 45 passing identities, 0 failing), make lint-code-fast (on an exported HEAD copy plus this task's diff, GOLANGCI_LINT_FIX=false: 0 issues)
- PRs: