---
satisfies: [R1, R3, R7]
---
# fn-43-deepen-ordinary-property-behavior-and.3 Deepen Query and Observation authoring

## Description
Complete the checked-authoring surface for Query and Observation, then migrate the ordinary Nexus walkthrough and the remaining Switch Query path. This task depends on the Property/Behavior facades because Nexus composes all three ordinary languages.

**Size:** M
**Files:** `model/Umpire/Query/Language.lean`, `model/Umpire/Query/Tests/Validation.lean`, `model/Umpire/Observation/Language.lean`, `model/Umpire/Observation/Tests/Compilation.lean`, `model/Umpire/Examples/Switch.lean`, `model/Temporal/Feature/Nexus/Operations/Internal.lean`, `model/Temporal/Feature/Nexus/Operations/AsyncStart.lean`, `model/Temporal/Feature/Nexus/Operations/Cancellation.lean`, `model/Temporal/Feature/Nexus/Operations/SuccessfulCompletion.lean`
**Touches:** [model/Umpire/Query/Language.lean, model/Umpire/Query/Tests/Validation.lean, model/Umpire/Observation/Language.lean, model/Umpire/Observation/Tests/Compilation.lean, model/Umpire/Examples/Switch.lean, model/Temporal/Feature/Nexus/Operations/Internal.lean, model/Temporal/Feature/Nexus/Operations/AsyncStart.lean, model/Temporal/Feature/Nexus/Operations/Cancellation.lean, model/Temporal/Feature/Nexus/Operations/SuccessfulCompletion.lean]

### Approach
- Add documented `checkedQuery` and `checkedObservation` facades with explicit proofs; keep raw typed checkers authoritative.
- Keep Query's dependent target re-ascription/materialization inside its language boundary so ordinary callers receive the correctly indexed checked query directly.
- Replace Query/Observation-local Definition ID and source-path plumbing with Task 1 primitives while retaining local error adapters.
- Add exact adapter-level Query fixtures for blank/malformed IDs, duplicate witness selection, source fallback, offending values, and related-ID ordering; preserve the existing Observation failure matrix.
- Migrate the three split Nexus operation walkthrough owners and the remaining Switch Query path away from `toOption.get`; preserve the Nexus facade explanation and diagnostic result values that existing tests intentionally inspect.
- Preserve the Nexus module explanation and every “Property, not Behavior” teaching comment while shrinking the surrounding literals.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Query/Language.lean:283-326` — local identity/source helpers and duplicate behavior.
- `model/Umpire/Query/Language.lean:496-530` — typed Query checker and dependent target result.
- `model/Umpire/Query/Tests/Validation.lean:1-48` — current kind-only validation coverage to deepen with exact payloads.
- `model/Umpire/Observation/Language.lean:416-503` — local identity/source helpers and error adaptation.
- `model/Umpire/Observation/Language.lean:1034-1075` — typed Observation checker.
- `model/Temporal/Feature/Nexus/Operations.lean:55-157` — first of three repeated ordinary authoring walkthroughs.
- `model/Umpire/Examples/Switch.lean:575-609` — remaining Query extraction and materialization ceremony.

**Optional** (reference as needed):
- `model/Temporal/Feature/Nexus/OperationsTests.lean:10-187` — public diagnostic and artifact-golden expectations that must remain valid.

### Key context
- fn-40 changes PlannerPolicy construction and canonical Query fixtures first; consume its final API and do not create a second policy helper.
- fn-39 will split Nexus Operations after this task; keep the refactor within the current file so the split moves already-simplified declarations.

### Quick commands
```bash
cd model && mise exec -- lake build Umpire.Query.Tests Umpire.Observation.Tests Temporal.Feature.Nexus.OperationsTests Umpire.Examples.Switch
```
## Acceptance
- [ ] Query and Observation expose documented explicit-proof checked facades through their existing public imports; raw checkers and typed diagnostics remain available.
- [ ] Checked Query owns extraction plus target-index re-ascription so ordinary Switch/Nexus callers do not need a separate materialization helper.
- [ ] Switch Query and all Nexus operation happy paths contain no `Except.toOption.get` ceremony; diagnostic result values inspected by existing tests remain intact where useful.
- [ ] Exact Query adapter fixtures pin blank/malformed-ID payloads, duplicate witness selection, source fallback, offending values, related-ID ordering, and canonical JSON; Observation's existing exact failure matrix remains green.
- [ ] Query/Observation invalid fixtures retain prior error kinds, offending values, related IDs, canonical ordering, and source-path fallback after shared identity utilities replace local mechanics.
- [ ] Existing teaching comments and authored documentation are preserved, and focused Query, Observation, Switch, and Nexus builds pass without semantic/fingerprint/artifact drift or new unapproved axiom dependencies.
## Done summary
Added explicit-proof checked Query and Observation facades, centralized their identity and source adapters, and migrated Switch plus the split Nexus operation walkthroughs to semantic checked authoring while preserving diagnostics, artifacts, fingerprints, and authored documentation. Exact adapter fixtures, focused and full model gates, lint, trust auditing, and implementation review are green.

stage: impl-review - ran [2026-09-03T01:34:07Z..2026-09-03T01:37:38Z]
stage: plan-sync - skipped(config: planSync.enabled != true)

## Evidence
- Commits: da89554df75ac331c2ebe744f9617b015884a90a
- Tests: baseline: green (cd model && mise exec -- lake build Umpire.Property.Tests Umpire.Behavior.Tests Umpire.Query.Tests Umpire.Observation.Tests Umpire.ImplementationLink.Tests UmpireTests; make umpire-build-model; make lint-model), TDD RED: cd model && mise exec -- lake build Umpire.Query.Tests.Validation Umpire.Observation.Tests.Compilation (missing checkedQuery and checkedObservation; exit 1), cd model && mise exec -- lake build Umpire.Query.Tests.Validation Umpire.Observation.Tests.Compilation, cd model && mise exec -- lake build Umpire.Query.Tests Umpire.Observation.Tests Temporal.Feature.Nexus.OperationsTests Umpire.Examples.Switch, cd model && mise exec -- lake build Umpire.Examples.SwitchTests, cd model && mise exec -- lake env lean ../.flow/tmp/Fn43Task3Trust.lean, cd model && mise exec -- lake build Umpire.Property.Tests Umpire.Behavior.Tests Umpire.Query.Tests Umpire.Observation.Tests Umpire.ImplementationLink.Tests UmpireTests, make umpire-build-model, make lint-model, git diff --check 5d2404284fcb14777d7aec95fa154857efcf5ef1..HEAD
- PRs:
