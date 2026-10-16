---
satisfies: [R6]
---
# fn-189-the-reify-plugin-keeps-only-the-body.2 Refuse curried helpers in lifted code; remove call-group proofs

Touches: model/framework/reify/body, model/framework/reify/plugin, model/framework/dsl/Reify.scala

## Description
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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
