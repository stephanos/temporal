---
satisfies: [R6]
---
# fn-189-the-reify-plugin-keeps-only-the-body.2 Refuse curried helpers in lifted code; remove call-group proofs

Touches: model/framework/reify/body, model/framework/reify/plugin, model/framework/dsl/Reify.scala

## Description
**DSL completion phase (owner, 2026-10-17):** scheduled before the IR/plugin phase because it changes what authors write or can reach; the gate is the native checks plus the plugin compiling the six samples.

A lifted body may call a def with one term parameter list; using clauses do not count. Refuse a call to a def with two or more term parameter lists at the call with the exact text `<fullName> has two parameter lists: write it with one` (the count spelled as written: two, three). Delete CallGroupEvidence, retainCallGroups, retainedGroups, the ruleCallGroups annotation and its Carrier.saveCallGroups/readCallGroups, and the proof copies in namedCopy and function copies. Add a red-first refusal fixture and a using-clause control that is accepted.

**Size:** S

### Verification
- The six samples compile and lift unchanged.
- Focused reify and export suites pass; the new refusal fixture reports at the call.

## Acceptance
- [ ] A curried helper reached from a capture is refused with the exact text; a helper with a using clause is accepted.
- [ ] ruleCallGroups and every call-group proof path are gone.
- [ ] The six samples lift to the same IR as before.

## Done summary
`5f09c1816a`, merged as `b7d4b2bc13`. Lifted code refuses a call to a def with two or more term parameter lists with the text `<fullName> has two parameter lists: write it with one` (the count spelled out); using clauses don't count. The call-group proof machinery is deleted: CallGroupEvidence, retainCallGroups, the ruleCallGroups annotation, Carrier save/read and the chunk codec (about 210 lines). Using-clause helper calls now carry their explicit argument too.

The refusal fixtures for two and three lists plus the using-clause control are in ReifyBoundarySuite: red at the branch point, green after. No new export failures. The DSL's `CoreBindings.CurriedCall` decoding is now dead and goes with fn-190.15/.16. Its README mentions are around lines 1058 and 1146.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 5f09c1816a, b7d4b2bc13
- Tests: mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests, ReifyBoundarySuite (two-parameter-list fixtures), mise exec -- scala-cli test --server=false model/framework/check --require-tests --test-only framework.check.lint.*RuleSuite
- PRs: