---
satisfies: [R15, R7, R14]
---
# fn-92-compose-entity-machines-into-one-system.7 Field-addressed Property requirements for composed and multi-field states

## Description
Let a same-step Property fix one state field: extend the predicate enumeration with a field-addressed requirement, lower it to the transition-contract clause both evaluators already match, and pin it on a fixture composition where the other member varies (R15, R7's caller claim, R14). Split from task .2 because it changes the `property` command's enumeration for every machine, not the `compose` command, and the caller composition (.5) is its first shipped use.

**Size:** M
**Files:** `model/Umpire/Command/Predicate.lean` (per-field fixing in `fixedRequirements`; the enumeration entry points take the field catalogs), `model/Umpire/Command/Authoring.lean` (`PropertyRequirement.stateFieldClause`, its lowering in `authoredProperty`), `model/Umpire/Command/Syntax.lean` (`requirementTerm` arm; the generated enumeration call in `sameStepCommand` passes the model's field metadata), `model/Umpire/Command/Instances.lean` (`liftedProperty` arm: located refusal), `model/Umpire/Command/Refinement.lean` (`refinedProperty` arm: located refusal), `model/Umpire/Command/Tests/Compose.lean` (nested-field fixture Property and its `verify` Query, positive and refused cases), `model/Umpire/Command/Tests/Authoring.lean` (a local synthetic multi-field machine: field fix, instances and refinement refusals, unchanged enumeration of every existing shape), `model/Umpire/Search/Tests/Differential.lean` (expected lines for the new fixture Query)
**Touches:** [model/Umpire/Command/Predicate.lean, model/Umpire/Command/Authoring.lean, model/Umpire/Command/Syntax.lean, model/Umpire/Command/Instances.lean, model/Umpire/Command/Refinement.lean, model/Umpire/Command/Tests/Compose.lean, model/Umpire/Command/Tests/Authoring.lean, model/Umpire/Search/Tests/Differential.lean]

### Approach
- `fixedRequirements` (`Predicate.lean:91-135`) fixes a whole state when every accepted step shares it and altering it to any other catalog state is rejected. Add the same rule per state field: read the field values through the model's `stateFieldValues` (`Authoring.lean:189-195`), and fix field `f` at value `v` when every accepted step carries `v` and every accepted step altered to any other value of `f`'s catalog, holding the other fields, is rejected; a whole-state fix wins when it exists, so every existing Property enumerates as before. The `carried` check (`:118-125`) includes field fixes so `notCarried` still names the distinguishing step; `fixesNothing` still fires when nothing is fixed.
- `PropertyRequirement.stateFieldClause label fieldName spelling` beside `stateClause` (`Authoring.lean:397-401`); `authoredProperty` (`:455-480`) lowers it to `.transitionContract … selected (PropertyPattern.exact .resultingState fieldId spelling)`, the field-addressed pattern `liftedProperty` already uses for slot fields (`Instances.lean:225`), with the field's own Definition ID from `stateFieldIds`. No Search change: `Observed.values .resultingState` (`Search/Product/Monitor.lean:196-212`) and the reference evaluator (`Property/Evaluate.lean:617, 966`) both offer a state's fields beside the state, from `query.target.stateFields` (`Product.lean:285`, `Search.lean:1049`); check `property.access.allows` (`Evaluate.lean:703`) admits the field.
- The three other exhaustive consumers of `PropertyRequirement` gain an arm: `requirementTerm` (`Syntax.lean:353`) renders the new constructor into the generated Property term, and the generated enumeration call in `sameStepCommand` (`Syntax.lean:396-440`) passes the declared model's field names and per-field catalogs so `fixedRequirements` can alter one field at a time; `liftedProperty` (`Instances.lean:222`) and `refinedProperty` (`Refinement.lean:164`) refuse a field clause with a located error in version one, because an `instances:` product exposes each slot's whole state as its fields and a refinement maps whole states to the abstract state, so neither has a field to address yet. Both refusals are pinned. The `Syntax.lean` edit overlaps task .4's, which is why this task follows .4.
- Fixture: a two-member composition where the claim fixes `worker.phase` while the other member varies; the positive `verify` Query verifies, a Property that varies the field is refused as `notCarried` with the step, and a Property over one whole member on a one-state member still reports `fixesNothing` (the one-member rule, `Predicate.lean:85-89`). Machine-side, in `Tests/Authoring.lean`: a local synthetic multi-field machine declared in the test module (an Umpire test module may not import `Temporal.Feature`, `ModelLint/ImportGraph.lean:315`) with a field fix, an `instances:` product and a refinement over it that pin the two refusals, plus the existing `#guard_msgs` pins; every shipped Property's enumeration is pinned unchanged by the Temporal differential block (`TemporalModelTests/SearchDifferential.lean:30-104`) and the Caller, Pair, Control, Outage, and Start `#guard_msgs` outputs.
- Add the fixture Query's lines to the Umpire sweep's expected block (`Differential.lean:570-582`), reading `veil default` with both backends agreeing.

### Investigation targets
**Required:**
- `model/Umpire/Command/Predicate.lean:14-30, 85-135, 140-210`
- `model/Umpire/Command/Authoring.lean:189-195, 397-410, 455-480`
- `model/Umpire/Command/Syntax.lean:353, 396-451`; `model/Umpire/Command/Instances.lean:218-240`; `model/Umpire/Command/Refinement.lean:160-182`
- `model/Umpire/Search/Product/Monitor.lean:190-215`; `model/Umpire/Property/Evaluate.lean:570-620, 692-712`
- `model/Umpire/Search/Tests/Differential.lean:534-582`; `model/ModelLint/ImportGraph.lean:305-330`

### Key context
- Whole-state fixing stays first so every shipped Property, whose clauses are pinned by the Temporal differential block, enumerates unchanged.
- The field's model value is `ModelValue.named fieldId spelling`, the shape `Instances.lean:101` and `:132` already emit for slot fields.

### Quick commands
```bash
cd model && lake build Umpire.Command.Tests.Compose Umpire.Command.Tests.Authoring Umpire.Search.Tests.Differential TemporalModelTests.SearchDifferential
LEAN_NUM_THREADS=1 make lint-model
```
## Acceptance
- [ ] A same-step Property on one member's field is fixed as a `stateFieldClause` while the other member varies; the fixture `verify` Query verifies and its differential lines read `veil default` with both backends agreeing
- [ ] A predicate that varies the field is refused as `notCarried` with the distinguishing step; the one-member rule still reports `fixesNothing`; `instances:` and `refines:` over a field clause refuse with pinned located errors; `requirementTerm` and the generated enumeration call carry the field metadata
- [ ] Every existing Property in `Temporal.Feature` and the Umpire fixtures enumerates to the same clauses as before; both differential expected blocks unchanged except the new lines
- [ ] `lake build UmpireTests TemporalModelTests` and `LEAN_NUM_THREADS=1 make lint-model` pass
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
