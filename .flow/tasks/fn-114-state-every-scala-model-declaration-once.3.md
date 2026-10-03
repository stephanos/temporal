---
satisfies: [R1, R4, R6, R11]
---
# fn-114-state-every-scala-model-declaration-once.3 Rewrite the Nexus caller realization by value with the shared script helpers

## Description
Finish the Nexus realization that fn-112.9 migrated only far enough to consume the shared kit: every reference to its own roles, learned values, observations, evidence kinds, controls and commands becomes a value reference, and its private string constants go.

**Size:** M
**Files:** `model/temporal/nexuscaller/Realization.scala` (687 lines on 2026-10-03, 28 private string constants at :42-74, :311, :362; evidence ids re-spelled at :106, :114, :134-166, :256-260 - re-locate after fn-112.9), `model/temporal/realize/**` only if a missing generic helper is found (else record it as a finding for fn-112 or a later spec), focused lifter refusal fixture.
**Touches:** [model/temporal/nexuscaller/Realization.scala, model/lifter/testdata/realizationRefusals/**, model/lifter/test/Fixtures.test.scala, model/ir/nexus-caller.json, model/ir/nexus-control.json, model/cases/**]

### Approach
- Mirror the finished `standaloneactivity/Realization.scala` from fn-112.9: `script`/`perform`/`onPath`/`always`, no `Item(` constructor, facts named by value, no `Control as _` import.
- An id the IR needs as text (e.g. `temporal.nexus.caller.evidence.started`) is written once on its declaration; every other mention references that declaration. The evidence-id vector order must stay `started, completed, failed, canceled, timedOut` (module map, migration goldens).
- Leave the literal waits (250 ms polls, `timeoutMs = 5000`) untouched: fn-118 owns them. Budget literals stay as data.
- Add a refusal fixture proving a reference to a non-existent declaration fails to compile or is refused by the lifter at its line (R4 errors).

### Investigation targets
**Required:**
- `model/temporal/nexuscaller/Realization.scala`
- `model/temporal/standaloneactivity/Realization.scala` (post fn-112.9)
- `model/temporal/realize/` (fn-112.9 kit)
**Optional:**
- `model/lifter/Realizations.scala` - reference resolution
- `tools/umpire/lower/testdata/migration/oracles/nexus`

### Quick commands
```bash
make umpire-gen-model && git diff --stat model/ir model/cases
make umpire-check-model
```

### Execution constraints
- Realization IDs, evidence catalogs, script order/modes and every Case byte equal the baseline; no wait/interval/timeout change (fn-118 boundary).
## Acceptance
- [ ] `Realization.scala` has no private string constant referenced by spelling elsewhere; each IR text id is written once on its declaration.
- [ ] No `Item(` constructor or `Control as _` import remains; the shared helpers are the only form, or exceptions are listed with reasons.
- [ ] A refusal fixture proves a reference to a missing declaration fails at its source.
- [ ] Literal waits are untouched and listed as fn-118's; all IR and Case bytes match the baseline; model gate passes.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
