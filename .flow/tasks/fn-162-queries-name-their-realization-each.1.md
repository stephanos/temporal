---
satisfies: [R1]
---
# fn-162-queries-name-their-realization-each.1 Trace how a Case's assessed Properties are selected today

## Description
Trace how a realized Case's assessed Properties are selected today, end to end, before changing it. Write the findings into the spec as `## Resolved via Codebase`. No code changes.

**Size:** S
**Files:** the spec only; read-only investigation
**Touches:** [.flow/specs/fn-162-queries-name-their-realization-each.md]

### Approach
- **Model:** how a `Realizes(<machine>)` / `DerivesFrom(...)` names its subject, and how a Query relates to its subject machine (`scenario.machine`, `property.machine`). Look at the activity's `system/Heartbeat.scala` subjects and their realizations (`HeartbeatThenCompletion`, `RetryAfterHeartbeat`, `ExhaustAfterHeartbeat`).
- **IR:** which Properties a realization, Query or Case carries in `model/ir/activity-standalone.json` (search for one subject's Properties and Queries). Which field says "assess against these".
- **Lowering:** `tools/umpire/lower` and `umpire-gen-cases`: what a lowered Case records about Properties.
- **Assessment:** `umpire-run --model` and `umpire-assess --model` (and the model-assessment code they link). Which Properties a Run is assessed against, and how capability-generated Properties enter.
- **The pin:** read `ActivitySubjectRegression`'s "focused subjects independently own the exact complete property truth sets" and say exactly what it guarantees.
- **Count the claim containers:** list every derivation in the models that changes nothing (`Derived(X.unmonitored)` / `Derived(X.rebind())` with no added rules, bindings or assumptions), with its Properties and Queries, and every predicate declared more than once.
- **Conclude:** confirm or correct the target design (a Query declares its assessment set). Flag anything that needs the user's decision before fn-162.2.

### Quick commands
```sh
grep -rn "extends Derived(.*\(unmonitored\|rebind()\))" model/temporal
grep -rn "Realizes(\|DerivesFrom(" model/temporal | head -40
```

## Acceptance
- [ ] The spec records, with code references, how a Case's assessed Properties are chosen today across model, IR, lowering and Go assessment, and what the subject regression test guarantees.
- [ ] It lists every claim-container derivation and every duplicated predicate.
- [ ] The target design is confirmed or corrected; open decisions are flagged before fn-162.2 starts.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
