---
satisfies: [R2, R3, R5]
---
# fn-31-deepen-umpire-target-and-simplify.3 Migrate the domain-neutral Switch teaching target

## Description
Move the minimum Umpire teaching example to the ordinary Target interface and prove compatibility (R2, R3, R5).

**Size:** S
**Files:** `model/Umpire/Examples/Switch.lean`, `model/Umpire/Examples/SwitchTests.lean`
**Touches:** [model/Umpire/Examples/Switch.lean, model/Umpire/Examples/SwitchTests.lean]

### Approach
- Replace routine provider/connector/extraction/planner assembly with the public Target path and opt the target into Target-owned finite planning once.
- Preserve the existing target-owned transition kernel and all fixtures.
- Preserve the existing `switch-role-domain/v1` and `switch-action-domain/v1` compatibility tokens verbatim at the Target finite-planning declaration.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Examples/Switch.lean:237-259` — current declaration, `composeTarget`, and checked extraction
- `model/Umpire/Examples/Switch.lean:409-548` — finite completeness, ordering, and planning plumbing
- `model/Umpire/Examples/SwitchTests.lean` — reference behavior

### Acceptance
- [ ] The example demonstrates ordinary authoring rather than framework plumbing.
- [ ] Existing semantic identities, plans, artifacts, and invalid cases are unchanged.
- [ ] Query derivation copies the existing role/action-domain tokens verbatim; ordinary query/planner code no longer threads them.

## Acceptance
- [ ] R2/R3 are demonstrated by the domain-neutral example.
- [ ] R5 whole-value and byte fixtures pass.
- [ ] Switch remains independent of Temporal.

## Done summary
Migrated the domain-neutral Switch teaching target onto the ordinary Target→Query→Planning path. Switch now declares its finite action domain and exact compatibility tokens once at Target, derives Query completeness and indexed Planning through the public seams, and preserves semantic identities, planner outcomes, canonical Query bytes, canonical artifact bytes, and Temporal independence.

Codex review found that the first version still exposed checked-result extraction at the teaching call site. The fix added `checkedTarget` to `model/Umpire/Target/Language.lean`, keeping extraction and exact kernel/planning re-ascription inside the deep Target boundary; this is the only implementation file outside the task's declared Switch surface. The unrelated dirty `.plans/UMPIRE4_SPEC.md` and `.flow/memory/declined/generated-api-drift-verification.md` were preserved and excluded. Project memory capture was attempted after NEEDS_WORK→SHIP but skipped because memory is not initialized.

baseline: red (`cd model && mise exec -- lake build Umpire.TargetTests Umpire.Query.Tests Umpire.Planning.Tests` hit a transient `Umpire.TargetTests.olean` ENOENT; its exact retry passed, and the other four Quick commands passed pre-edit)
stage: impl-review - ran [2026-08-27T04:19:59Z..2026-08-27T04:23:35Z]
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 5e5f59c2085fe9dbd209df22728a047e722ead21, 7189a5aa77e91c6bf705d04d518dcb1c75684fc2
- Tests: baseline: red (cd model && mise exec -- lake build Umpire.TargetTests Umpire.Query.Tests Umpire.Planning.Tests hit transient Umpire.TargetTests.olean ENOENT; exact retry passed; remaining Quick commands passed), cd model && mise exec -- lake build Umpire.Examples.SwitchTests, cd model && mise exec -- lake build Umpire.TargetTests Umpire.Examples.SwitchTests, cd model && mise exec -- lake build Umpire.TargetTests Umpire.Query.Tests Umpire.Planning.Tests, cd model && mise exec -- lake build Temporal.Feature.Nexus.LifecycleTests Temporal.Feature.Nexus.OperationsTests Temporal.Feature.Nexus.Experimental.CallerClosureTests, cd model && mise exec -- lake build UmpireTests TemporalModelTests, make umpire-check-regression (first verification hit transient TemporalExperimentalTests.olean ENOENT; exact retry passed), make lint-model, rg -n '^import Temporal' model/Umpire/Examples/Switch.lean model/Umpire/Examples/SwitchTests.lean (no matches), git diff --check
- PRs:
