---
satisfies: [R4]
---
# fn-228-runner-and-run-the-one-go-host-loop.4 After .3: Temporal Driver as act-and-observe; live activity slice through umpire explore

## Description
Reduce the retained temporal/* driver to a runner.Driver: send the proposed request over the generated API, collect Observations with poll interval, read timeout and backoff from instruction fields, capture and redaction as field paths, RPC completion and evidence well-formedness checks; surface SDK callbacks from the in-process Go worker. Run the live activity slice (start, retry then completion, cancellation, host refusal) through umpire explore with the Run as the only artifact; make umpire-check-model-slice invokes it. Read timeout without Model acceptance is a DriverFault event; malformed evidence refused before Judge.

## Acceptance
- [ ] Temporal Driver implements runner.Driver without interpreting Temporal messages
- [ ] Live slice passes through umpire explore; Run is the only artifact
- [ ] DriverFault and malformed-evidence controls

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
