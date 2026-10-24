---
satisfies: [R6]
---
# fn-166-mini-activity-a-small-complete-model-to.10 Held sweep: runnable migration rules and worked application

## Description
Held sweep. Completes R6's executable infrastructure and mini worked application after the owner opens the required tooling work.

**Size:** M
**Files:** shared custom migration rule source/build/loading configuration under model, `model/.scalafix.conf`, existing Make fix route, worked-example fixtures and proof records.
**Touches:** [model/migrate/**, model/.scalafix.conf, model/README.md, model/framework/SEMANTICS.md, Makefile, .flow/specs/*.md]

### Approach
- Re-anchor the convention authored in task 9 and establish the actual custom Scalafix rule compilation/loading route at `Makefile:689`. Use one canonical location, not two competing migration modules. Reconcile fn-167's pure transformation location before wiring its adapter.
- Implement the worked example as an actual rule, not a separately tested string-rewriting engine described as Scalafix. Select a concrete source rename whose required byte-identical lift is demonstrable; an exported identity rename cannot silently stand in for that proof.
- Apply the rule through the documented route to isolated mini input, compile/lift and compare the required IR. The current spec's owner Scope forbids application to a real feature; that later sweep is outside this task's admission and completion. Preserve historical payloads, pins and equivalence baselines.
- Record executable configuration, commands, positive/refusal fixtures and committed closure. Fn-167.7 consumes this infrastructure closure; do not make this task await fn-166.5's complete loop or fn-167.7's gap retirement.
- No owner-held authority is granted by this plan. Admit only after the owner lifts the relevant tooling restriction.

### Verification
Execute actual rule loading, the mini example proof, exact semantic/identity comparisons and failed-load/unplaceable-input controls at the authorized sweep. Broad validation/review use the batch-end cadence.

## Acceptance
- [ ] Owner admission is recorded and an actual custom rule compiles, loads and runs through the documented model fix route.
- [ ] The mini worked example produces the required byte-identical IR; no real feature is edited or migrated under the current owner Scope.
- [ ] One committed infrastructure closure supplies fn-167.7's adapter prerequisite without depending on final loop or gap retirement.


## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: