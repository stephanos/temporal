---
satisfies: [R1, R3]
---
# fn-227-parameterized-models-declared-worlds.1 Declared World, world access and computed starts in #[machine]; parameterized_world fixture with TLC

## Description
Add #[derive(World)], the optional trailing &World parameter on handlers and claim methods (arity-detected like before), and the #[starts] impl block for computed starts, keeping field #[starts(range)] and refusing both at once. Absent world keeps struct World; and byte-identical expansions. Lift declared worlds as TLA+ CONSTANTs with .cfg values and computed starts as the enumerated Init set. Add shared fixture parameterized_world (two-parameter world, computed starts, a world-bounded claim) with report/rows/states/paths and tlc goldens; trybuild pins the three refusals. Verify make model-check, UMPIRE_BACKENDS=require make model-check, make model-time; every pre-existing golden and expansion snapshot byte-identical.

## Acceptance
- [ ] #[derive(World)], &World access and #[starts] impl block work; misuse refused by trybuild
- [ ] parameterized_world fixture goldens present and TLC agrees
- [ ] All pre-existing goldens and expansion snapshots byte-identical
- [ ] make model-check and UMPIRE_BACKENDS=require pass; model-time pasted

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
