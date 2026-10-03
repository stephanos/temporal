---
satisfies: [R3, R4, R16]
---
# fn-112-make-the-standalone-activity-scala.4 Add typed composition members, sync references and reusable claims

Touches: [model/umpire/Compose.scala, model/umpire/Claims.scala, model/lifter/Compositions.scala, model/lifter/Claims.scala, model/lifter/test/**, model/lifter/testdata/**]

## Description
Replace composition/action/fact string keys with typed selectors and support one claim definition across member and composed states.

**Size:** M
**Files:** model/umpire/Compose.scala and Claims.scala; model/lifter/Compositions.scala and Claims.scala; focused fixtures.

### Approach
- Add typed member selectors for composition declaration, own, synced, records and withMember. Use `c.synced(_.member -> action)` as the disambiguated scenario form.
- Preserve sync/member order. `withMember` recomputes the selected member's `replaces` target from the new machine's direct refinement instead of copying the base target; reject missing/non-field selectors, incompatible members/refinements and zero/multiple sync matches at the selector.
- Add the minimum typed abstraction that lets the three shared properties be declared once and lifted for both composition families, with no new IR.
- Cover separator-bearing names and identically spelled domains so member/action identity is injective rather than string-joined.

## Acceptance
- [ ] All typed composition operations lower to existing member/sync/action keys with original order and identities.
- [ ] withMember fixtures cover ordinary providers replacing dispatchQueue and the lossy provider replacing dispatchQueueUnderStorageLoss, with exact original metadata for both.
- [ ] Ambiguous syncs require member qualification and every invalid selector/replacement has a located refusal fixture.
- [ ] One property definition can produce the original machine and composition Property rows without string keys.
- [ ] Task-1 equivalence and focused composition/claim tests pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
