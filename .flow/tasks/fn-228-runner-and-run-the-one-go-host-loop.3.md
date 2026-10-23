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
Implemented R3: the Go `runner` package (`umpire.local/model/umpire/runner`) with Runner, Run (`umpire.run.v1`), the incremental Driver/Exchange contract and the five modes question, explore, reduce, replay and repeat; one `umpire` command (tools/umpire/cmd/umpire, exit 0 Explained, 1 Unexplained, 2 Refused/DriverFault, 3 could not run) replacing run, fuzz, replay, assess and repeat; never-overwrite publication keyed by model hash / seed / SHA-256 of the canonical Run.

One Host per Run, every Host call from one goroutine in receive order. Replay rebuilds a fresh Host, replays Reset/Propose/Judge/Answer/Annotate in order, compares every response, rejects calls after the terminal event, empty Runs and termination that does not fit the events; Explained records the budget that ended it. Reduce first replays the recorded Run, tries proposal budgets 1..n, and reports Refused (exit 2) when nothing reproduces. Driver.Start must return once sent; late or interrupted Starts are faulted and their exchanges joined. Every Runner wait goes through one deadline hook, so the tests drive timeouts deterministically.

Fake-Driver tests over fn-228.2's stateful control guest cover every mode, callbacks between polls, blocked/canceled sends, premature close, correlation failures, 18+ replay tamper cases, two distinct repeated Runs, concurrent publication, collisions and progress redaction; stock Kettle is the refusal control.

Decisions: canary reduction to lease/policy and retirement of testpilot/publish deferred to .6; four live tests of the deleted commands retired with their coverage carried into .4 and .6 (retired-coverage.md); four further live failures pre-exist at 0c7822bc43 and are carried into .6; a guest trap still exits 3 without a partial Run (reviewer accepted).

Review: three fresh-context rounds (Opus, same family): NEEDS_WORK (live-gate blocker, replay laundering, budget off-by-one, refused Reset, reduce exit, Start bound) -> NEEDS_WORK (unjoined late Start, empty Run) -> SHIP.

Gate hygiene found on the way: TestKettleMutants now bounds each probe instead of one shared 30 s budget; the Apalache per-check limit was raised from 120 s to 600 s because concurrent bound-63 checks overran it on a loaded machine (a timeout only refuses).

Gates: default `make model-check` rc 0 (422 s at 70f0013527); `UMPIRE_BACKENDS=require make model-time` rc 0 at 059dd53bc1, gate 872.085 s against the 60 s soft target, inner loop 6.687 s against 10 s. Runner tests -count=50 under load, -race, lint clean. `make umpire-check-live-tests` still fails only on the four pre-existing identities. Evidence in .flow/artifacts/fn-228.3/.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 8118630a45, 6b137fb3fb, e98ec44df6, 5694e9732c, 47e2d38822, 32ab1cf162, 70f0013527, 059dd53bc1
- Tests: UMPIRE_BACKENDS=require make model-time, make model-check, go test ./umpire/runner/... -count=50 -race, make umpire-check-live-tests (four pre-existing failures)
- PRs: