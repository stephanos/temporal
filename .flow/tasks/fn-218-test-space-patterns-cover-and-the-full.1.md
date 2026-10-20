---
satisfies: [R1, R3]
---
# fn-218-test-space-patterns-cover-and-the-full.1 Full path! grammar: actions, inputs, outcomes, alternatives, repetition, permutations; alternative enumeration

## Description
Extend Path/path! in umpire-dsl with actions, structured inputs (ranges, sets, exact values, `_`/`..`), actor wildcard, typed outcomes and outcome alternatives, alternatives, exact/bounded repetition, unordered permutations, and exclusion gaps. Enumerate every authored finite alternative and count before search with checked arithmetic and an overridable default permutation limit of six. Preserve state-only paths, closure inference and two-state progress claims; malformed grammar and test-only claim forms have pinned trybuild diagnostics. Implement the normal Action payload surface using required finite #[inputs(...)] field domains, full-value equality/hash/clone/formatting, and typed actor projection; never narrow Model catalogs to requested paths. Supply the supporting typed outcome/fault catalog surface for R3: generated stock/unit outcomes and explicitly marked fault actors, borrowed classifiers, and honest undeclared metadata for manual Machines. The spec's Implementation decisions and .flow/tmp/fn218-design-handoff.md explain the grounded design; API details may improve while preserving the requirements. **Touches:** `model/umpire/dsl/**`. **Quick:** `cd model && mise exec -- cargo test -p umpire-dsl -p umpire-dsl-macros`; full gates `make model-check` and `make model-time`.

## Acceptance
- [ ] All requested grammar forms construct typed paths and enumerate finite alternatives in declared order, with checked counts and limit controls.
- [ ] Structured derived actions have explicit complete finite input domains, payload-sensitive identity/formatting and typed actor conversion.
- [ ] Outcome/fault metadata supports later complete coverage without imposing Clone/Eq on generic outcomes or treating missing metadata as declared empty.
- [ ] Malformed grammar and every test-only form in a progress claim are pinned by trybuild; existing state-only paths and claim behavior remain valid.
- [ ] Focused tests, make model-check and make model-time pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
