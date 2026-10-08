---
satisfies: [R1, R2]
---
# fn-140-one-sentence-witness-queries-with.1 Declare typed witness builders and their core Query values

Touches: [model/umpire/Claims.scala, model/umpire/Machine.scala, model/umpire/Syntax.scala, model/umpire/IrFile.scala, model/umpire/Witness.test.scala]

## Description
Build the authoring surface and its ordinary declaration values for R1/R2. Keep typed construction together before teaching the lifter the captured spelling.

**Size:** M
**Files:** `model/umpire/{Claims,Machine,Syntax,IrFile}.scala`; new `model/umpire/Witness.test.scala`.

### Approach
- Re-anchor these paths and package names to the integrated fn-142/fn-143 preparation, if it has landed; the current `model/umpire` paths become `model/framework` without changing this task's ownership or scope.
- Reuse `Declares[S]` and its Outcome/Fact members at `Machine.scala:60-88`, and the builders in `Claims.scala:20-75`. A completed witness stays usable wherever an existing Query value is accepted, including IR roots and explicitly named list items.
- Keep the public witness spelling in `Syntax.scala` under the existing sugar/core rule. Its documented core form is the existing pinned Scenario, same-step Property and find Query. Core declarations must not import sugar.
- Support machine, derived-machine and composition declarations, implicit val names and explicit names. Type machine facts against the declaring machine. For composition records, bind the member selector to that member's fact type and prove foreign facts fail compilation; do not copy the existing Any-typed convenience signature without checking R1.
- Derive steps/actions from path length, use one documented framework search default of 512 (the established live-query budget), and permit the existing Limits override. Keep computed Query totals internal; current eligible Models need neither a new starts nor total surface.
- Retain `.expect` while existing callers still require it. Task .3 removes it atomically with the live migration. No additional compatibility method is added.

### Investigation targets
**Required:**
- `model/umpire/Machine.scala:60-88` - shared declarations and associated fact types.
- `model/umpire/Claims.scala:20-116` - existing core builders and Query modifiers.
- `model/umpire/Syntax.scala:208-220` - machine and composed records forms.
- `model/umpire/Compose.scala` - member selectors and composed classes.
- `model/umpire/IrFile.scala:61` - Query root discovery.
**Optional:**
- `model/check/SyntaxRule.scala:27` - sugar placement contract.

### Quick commands
```bash
mise exec -- scala-cli test model/project.scala model/umpire --test-only '*Witness*'
make lint-model-syntax
```

## Acceptance
- [ ] Constructed witness values have the existing Query/triple shape, typed state predicates, ordered machine/composed classes and derived/overridden limits (R1/R2).
- [ ] Compile specimens cover foreign machine facts, foreign member facts and a reason-free satisfied value; positive fixtures include derived and composed owners and explicit names in lists.
- [ ] Query discovery still visits completed witnesses through machine sections and IR root values.
- [ ] Sugar definitions state their core form and the focused authoring tests and syntax gate pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
