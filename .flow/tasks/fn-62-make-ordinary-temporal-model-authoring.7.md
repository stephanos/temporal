---
satisfies: [R1, R8, R9]
---
# fn-62-make-ordinary-temporal-model-authoring.7 Publish and verify the complete established authoring journey

## Description
Integrate `.1`–`.6` and `.8` into a compiled established Nexus reader path, exact compatibility evidence, public documentation, and final gates.

**Size:** M
**Files:** model/owner docs, established Nexus public tutorial/tests, import tests and aggregate roots
**Touches:** [model/README.md, model/Umpire/ARCHITECTURE.md, model/Umpire/Target/ImportTests.lean, model/Umpire/Property/ImportTests.lean, model/Umpire/Behavior/ImportTests.lean, model/Umpire/Query/Tests/Visibility.lean, model/Umpire/Observation/ImportTests.lean, model/Umpire/Planning/VisibilityTests.lean, model/Temporal/Feature/Nexus.lean, model/Temporal/Feature/Nexus/**, model/TemporalModelTests.lean]

## Approach
- Make the established Nexus journey itself the compiled public-facade walkthrough: finite Target, Properties, Behaviors, Queries, existing planner, Observation, and authored gaps. Add a concise reader-order guide and locally relevant Lean syntax key, not an unrelated duplicate example family.
- Include representative malformed ID/reference, missing author proof, invalid raw Target/transition and Observation examples in runnable tests.
- Document ordinary versus expert authoring, explicit check/checked seams, stable IDs, composition, outcomes, Limits and nonbehavioral gaps. Keep Nexus2's exception and unmeasured usability boundaries explicit; no broader rule change.
- Verify exact Nexus compatibility and named source/gap deltas. Audit all changed load-bearing declarations, public imports and complete first-party module boundaries.
- Freeze source/tests before final serial gates. Recording results in documentation does not itself justify rerunning unchanged-source gates.

## Investigation targets
**Required:**
- `model/README.md:51` — public reader path.
- `model/Umpire/ARCHITECTURE.md:52` — authoring boundaries.
- `model/Temporal/Feature/Nexus.lean:8` — public model entrypoint.
- `model/Temporal/Feature/Nexus/Operations/PlanningTests.lean` — exact artifacts.
- `model/Temporal/Feature/Nexus/ObservationTests.lean` — Observation compatibility.
- `model/Temporal/Feature/Nexus2/EVIDENCE.md:57` — completed prototype comparison, not established migration evidence.

## Acceptance
- [ ] Public compiled walkthrough and negative specimens cover R1, including actual Observation and Known Gap paths, without internal/Experimental/runtime/verification imports.
- [ ] Exact established compatibility, full error payloads, source/gap deltas and trust inventory are recorded as executable evidence; docs accurately distinguish prototype and production.
- [ ] Serial aggregate `cd model && mise exec -- lake build UmpireTests TemporalModelTests TemporalExperimentalTests` and `make umpire-build-model` pass.
- [ ] Full `make umpire-check-regression` uses physical canonical TMPDIR and passes; `make lint-model` passes; `make lint-code GOLANGCI_LINT_FIX=false` has no new diagnostics against the verified baseline. Go tests use `-tags test_dep`.
- [ ] All task evidence is recorded before task review; fn-62 still requires whole-spec completion review before closure.
- [ ] Consolidate the named constructor/composition complexity audits from `.1`, `.3`, `.5`, `.6`, `.8`, checking every added call/traversal and the stated 1×/10× pass conditions; timing alone or repeated cached admission is insufficient.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
