---
satisfies: [R3, R4]
---
# fn-101-gomad-f7-any-functional-test-and-ci.6 Make core-linux pass: platform-neutral required probes, measured linux expectations, platform-aware host tier

## Description
From fn-101.5's fork CI findings (run 36485805062; per-seed linux reports in run 36470738437): (1) remove `stdlib.os.getwd` from the functional suites' required_probes in temporal.json and tests.generator.json (it fires only from darwin's os init, not from anything the suites depend on); keep only probes for modeled operations the suites actually use on both platforms, verified against both platforms' reports; (2) set frontend-system-info's linux/amd64 expectation to `intermittent` with a finding naming F3 and the observed nondeterministic seed; (3) make the linux host tier's darwin-assuming tests (TestGenerateRendersDescriptorConsumers, upgrade Run* tests) select or skip by platform correctly (no weakening on darwin); (4) push, dispatch the fork workflow, and from the linux Temporal requalification tighten every linux expectation that qualified on both seeds in the run (e.g. user-timers, activity-batch-cancel, the F6 slice) from `intermittent` to `qualified` only where the evidence shows both seeds qualified; anything still diverging on linux stays with a finding and is reported to the conductor as a determinism finding; (5) iterate until core-linux passes.

## Acceptance
- fork gomad3 workflow: all four jobs pass
- every linux expectation is either qualified (measured) or intermittent/unrepeatable with a finding

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
