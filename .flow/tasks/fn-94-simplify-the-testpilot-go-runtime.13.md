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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
