---
satisfies: [R3]
---
# fn-228-runner-and-run-the-one-go-host-loop.3 After .2: Go runner package: Runner, Run, Driver interface, five modes, one umpire command, publish

## Description
Create umpire.local/model/umpire/runner with Runner, the Driver interface (Act and Callbacks), the Run schema umpire.run.v1 with Event oneof and Verdict, and the loop Reset / Propose / Act / Judge / Answer until quiescent, budget or Unexplained; one Host per Run, callbacks serialized. Modes question, explore, reduce, replay, repeat as arguments. A fake Driver over the stateful control guest supplied by .2 (stock Kettle is a refusal control) pins the loop and each mode's ending and exit code. The umpire command replaces run, fuzz, replay, assess and repeat (deleted). publish stores Runs keyed by model hash / seed / canonical Run content hash, never overwriting; canary keeps lease and policy only.

### Required integration contract

Implement the parent incremental Driver/Exchange, ordered replay, mode budgets, publication identity and progress contracts. Use the positive control guest from .2; stock Kettle is a refusal control only. Include fake-driver callback-between-polls, blocked/canceled sends, premature stream close and correlation failures. Replay includes Reset/Propose/Answer/Judge and Annotate, compares responses, and never invokes a Driver. Test two distinct repeated Runs sharing model/seed and concurrent never-overwrite publication.

## Acceptance
- [ ] runner package with Runner, Run, Driver and the five modes; fake-Driver tests per mode
- [ ] One Host per Run; callbacks serialized
- [ ] umpire command present; old commands gone; exit codes as specified
- [ ] publish never overwrites

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
