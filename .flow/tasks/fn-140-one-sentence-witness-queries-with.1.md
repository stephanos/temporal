---
satisfies: [R1, R2]
---
# fn-140-one-sentence-witness-queries-with.1 Declare typed witness builders and their core Query values

Touches: [model/framework/Claims.scala, model/framework/Machine.scala, model/framework/Syntax.scala, model/framework/IrFile.scala, model/framework/Witness.test.scala, model/irgen/Switches.scala]

## Description
Build the authoring surface and its ordinary declaration values for R1/R2. Keep typed construction together before teaching the lifter the captured spelling.

**Size:** M
**Files:** `model/framework/{Claims,Machine,Syntax,IrFile}.scala`; new `model/framework/Witness.test.scala`.

### Approach
- Use the integrated `model/framework` paths left by fn-143. Begin implementation only after fn-156's Flow closure is committed; consume fn-155's actual mapping and fn-156's source ownership at that baseline without changing scope.
- Reuse `Declares[S]` and its Outcome/Fact members at `Machine.scala:57-83`, and the builders in `Claims.scala:20-75`. A completed witness stays usable wherever an existing Query value is accepted, including IR roots and explicitly named list items.
- Keep the public witness spelling in `Syntax.scala` under the existing sugar/core rule. Its documented core form is the existing pinned Scenario, same-step Property and find Query. Core declarations must not import sugar.
- Support machine, derived-machine and composition declarations, implicit val names and explicit names. Type machine facts against the declaring machine. For composition records, bind the member selector to that member's fact type and prove foreign facts fail compilation; do not copy the existing Any-typed convenience signature without checking R1.
- Derive steps/actions from path length, use one documented framework search default of 512 (the established live-query budget), and permit the existing Limits override. Keep computed Query totals internal; current eligible Models need neither a new starts nor total surface.
- The integrated baseline already replaced `.expect` with `.live` and added source-only expectation reasons. Keep that surface.
- The typed composition records selector is `_.member -> MemberMachine`, reusing composition declarations to retain the member machine's fact type. The conductor accepted this clarification on 2026-10-11.
- Repair inherited syntax findings in the touched Machine/Syntax files by moving the unwritten-phase sentinel into core Phasing. The conductor also authorized deleting only the unused `Switches.on` accessor so the syntax gate passes; no other lifter edits belong to this task.

### Investigation targets
**Required:**
- `model/framework/Machine.scala:57-83` - shared declarations and associated fact types.
- `model/framework/Claims.scala:20-116` - existing core builders and Query modifiers.
- `model/framework/Syntax.scala:208-220` - machine and composed records forms.
- `model/framework/Compose.scala` - member selectors and composed classes.
- `model/framework/IrFile.scala:61` - Query root discovery.
**Optional:**
- `model/check/SyntaxRule.scala:27` - sugar placement contract.

### Quick commands
```bash
mise exec -- scala-cli test model/project.scala model/framework --test-only '*Witness*'
make lint-model-syntax
```

## Acceptance
- [ ] Constructed witness values have the existing Query/triple shape, typed state predicates, ordered machine/composed classes and derived/overridden limits (R1/R2).
- [ ] Compile specimens cover foreign machine facts, foreign member facts and a reason-free satisfied value; positive fixtures include derived and composed owners and explicit names in lists.
- [ ] Query discovery still visits completed witnesses through machine sections and IR root values.
- [ ] Sugar definitions state their core form and the focused authoring tests and syntax gate pass.

## Done summary
Model authors can construct machine, derived-machine and composition witnesses as ordinary find Queries with an ordered pinned Scenario, a same-step recorded-fact Property, optional typed end conditions and path-derived Limits. Explicit names propagate to the Query and both core declarations; implicit names retain the existing blank-name source-capture convention for task .2.

The composition spelling is `.records(_.member -> MemberMachine, fact)`, approved by the conductor to preserve the machine's fact type through the selector. Its predicate compares the exact composed key and checks unique machine membership; task .2 must validate the source field-to-machine binding when lifting.

Witness.test.scala passes four focused tests covering the core triple, derived and composed owners, discovery through Query lists and machine sections, foreign facts, missing records/classes, typed ends, explicit names, custom limits, exploration and a reason-free satisfied live value. Effects passes eight tests and Phased passes five tests after moving the missing-writer sentinel into core Phasing. The syntax gate passes after that repair and the authorized removal of unused Switches.on.

baseline: red (make lint-model-syntax failed pre-edit with four inherited sugar findings, repaired under conductor authorization). The initial Witness filter exited 0 with no suite selected and counts as baseline: none. The TDD log records missing witness API compilation failures before implementation; fixture state annotations were subsequently corrected. An attempted combined test filter was inconclusive because scala-cli rejects repeated --test-only arguments; each selected suite was then run separately and passed.

stage: impl-review - skipped(config: REVIEW_MODE=none; owner defers review to the shared close)
Tier: session (jev-unavailable(no_key)); AGENTS implementer routing gpt-6.1-sol medium.

Verification remains Scala-only. The full model gate, Go checks, managed artifact regeneration, independent review and live equivalence remain deferred under the owner's milestone instructions. Task .2 owns source-name capture and witness source refusals. The integrated .live and expectation-reason surface remains in place.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: c95a18c8d1fe9d49c166b46eee7324ec88982648
- Tests: mise exec -- scala-cli test model/project.scala model/framework --test-only '*Witness*', mise exec -- scala-cli test model/project.scala model/framework --test-only '*Effects*', mise exec -- scala-cli test model/project.scala model/framework --test-only '*Phased*', make lint-model-syntax, mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check model/framework/Claims.scala model/framework/Machine.scala model/framework/Syntax.scala model/framework/Witness.test.scala model/irgen/Switches.scala, git diff --check
- PRs: