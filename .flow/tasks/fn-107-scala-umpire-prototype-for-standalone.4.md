---
satisfies: [R1, R4, R9]
---
# fn-107-scala-umpire-prototype-for-standalone.4 Check activity parity, admission race, and scoped queue refinement

Touches: [model/scalav2/scala/temporal/standaloneactivity/**, model/scalav2/ir/activity*.json, model/scalav2/goir/activity*_test.go, model/scalav2/run.sh]

## Description
Deliver the executable activity proof point using the existing Go activity baseline and the reviewed provider/race controls.

**Size:** M
**Files:** model/scalav2/scala/temporal/standaloneactivity/Model.scala and Claims.scala, proposed System.scala beside them, generated activity IR, differential fixture/test.

### Approach
- Lift the existing activity behavior and compare all states/actions/results/claims in the explicit intersection domain, including disabled actions. Keep the baseline Go policy unchanged.
- Add independently identified operation/attempt/delivery state, current-eligibility admission, and the deliberately faulty stale-eligibility control.
- Replace the opaque durable queue with the detailed provider from the sketch; explore crash/commit/ack cuts and competing timers. Include a faulty provider.
- Author ordinary process crash and committed-storage loss as separate transitions with distinct fault IDs and receipt entries. Ordinary crash preserves committed queue state; enable destructive storage loss only through its explicitly selected fault assumption. Include a provider mutation that wrongly loses committed state on an ordinary crash.
- Pin expected witnesses and report source-specific coverage/exclusions. Stop downstream integration if parity or corrected refinement fails.

### Investigation targets
**Required:** model/go/standaloneactivity/model.go:301; model/go/standaloneactivity/claims.go:24; model/go/standaloneactivity/pins_test.go:122; model/scalav2/scala/temporal/standaloneactivity/Model.scala; chasm/lib/activity/tasks.go:75.
**Optional:** tests/activity_parity_test.go:544; chasm/lib/activity/statemachine.go:423.

### Quick commands
`make umpire-check-scala`; `make lint-scala`; `mise exec -- go test -tags test_dep ./model/scalav2/... ./model/go/standaloneactivity/...`.

## Acceptance
- [ ] Exhaustive correspondence within the declared baseline domain covers disabled behavior and results.
- [ ] Generic search finds the stale-delivery control and excludes it for the current-eligibility design.
- [ ] Scoped queue substitution passes and the violating provider fails with a replayable witness.
- [ ] Duplicate delivery, pre-pause admission, timer ordering, and crash cuts match the trace oracles; ordinary crash preserves committed records and separately enabled storage loss has its own transition/receipt/control.

- [ ] Legacy model/scala files remain unchanged; v2 generation, lifting, native tests and lint keep passing with that tree absent.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
