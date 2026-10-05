---
satisfies: [R3]
---
# fn-124-shrink-and-simplify-the-umpire-go.3 Declare activity attempts, start carriers, lost admissions, causal parents, timeouts and history event names in the realization

## Description
Implements R3. Cross-spec entry gate: start after fn-118 lands (it edits waits, timeouts and realization surfaces). Each Temporal fact listed in R3 moves to one declaration in the realization (model/temporal/realize or the feature's Realization.scala), is lowered into the Case, and the Go runtime, lowering and conformance read the declared value instead of their own rule; remove the Go copies. Case bytes may change only by the new declared fields; record each change. Run the live generated Cases once.
## Acceptance
- [ ] History event names: the lowering and Testpilot take a history kind's recorded message from the realization's own history read (the read method's response type at the read path, which the Case already carries) and its oneof from the attributes member it names. `historyEventMessage` in `tools/umpire/lower/descriptor.go` and `common/testing/testpilot/internal/execution/evidence.go`, and the literal `attributes` oneof name next to them, are gone. No Case byte changes for this.
- [ ] Activity attempts: the Temporal kit declares once (`temporalBehavior`) how the server numbers an activity's attempts (from 1, every attempt of one activity in its one run). The lowering writes it into each Case activity entrypoint; Testpilot's scheduler classifies a reservation outcome by the declared numbering (no `ordinal+1`, no unconditional one-run rule) and refuses an activity entrypoint that declares none; the Temporal Driver refuses, at Profile derivation, a numbering other than the one it routes (from 1, one run); conformance's one-operation-one-run rule reads the realization's declaration. `guard.go` and the attempt role in `conformance/evidence.go` already read declared values (AttemptOf.number, the Run protocol's typed attempt); recorded, not changed.
- [ ] Default instruction timeouts: the kit declares the limits of an instruction that writes none, with its reason. The lowering writes them into each Case's Program; Testpilot resolves an instruction's limits as its own, else the Program's declared defaults, else the Profile's (a generic Profile option no Temporal Profile sets). `temporal.DefaultInstructionLimits` and the canary's literal copy are gone; a reader of the default (canary controller) reads the Case's.
- [ ] Causal parents: the kit declares that one operation's evidence is ordered by the Run's record across sources. The lowering writes it into the Case; Testpilot names the previous evidence of the operation as a causal parent only when the Case declares it.
- [ ] Lost admission: Testpilot's rule for a lost admission response is kept, because it checks the Driver's outcome against the Testpilot IR's own definition of `FAULT_KIND_ADMISSION_RESPONSE_LOSS` (it replaces one committed admission response), not a server fact: the server fact is the Model's (`committedThenLost`/`failedThenLost`, `committedDespiteLostResponse`), and removing the check would report a Driver bug as a product violation. Its attempt check is a presence check (no "from 1"), and `run.proto` no longer restates the numbering.
- [ ] Start carriers and "a failed start starts nothing" stay in the Temporal Driver (owner decision, recorded in R3): the Driver may know Temporal, carriers are its own interception mechanism, and a failed start releasing reservations can only make a Run incomplete. The carrier table is defined once (`delivery.Carried`), read by Profile derivation, the ledger and the worker Driver's validation.
- [ ] Case bytes change only by the new declared fields; each change is recorded with a before/after projection under `.flow/tmp/fn124-3/`. The golden harness and `original.json` are extended only for differences this task introduces. The Driver catalog identity rotates with the Testpilot proto; the pinned Runs and receipts follow as fn-118.3 did.
- [ ] Docs (model/README.md, model/SEMANTICS.md, Testpilot README where they state these rules) describe the declarations.
- [ ] Gates pass: model gate, original-baseline check, full Go tooling suite + Testpilot + canary tests (`-json`, timed), `make lint-code-fast`, `make umpire-check-cases`, `make umpire-check-fixtures`, `make canary-check-case`; the live generated Cases run once (known INCONCLUSIVEs and the stale `TestTestpilotAssessRecordedRuns` expectation excepted).
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
