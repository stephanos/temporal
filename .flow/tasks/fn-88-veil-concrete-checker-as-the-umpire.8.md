# fn-88-veil-concrete-checker-as-the-umpire.8 GOV-02 amendment drafts for every rule the Veil dependency contradicts

## Description
Draft, under the `*(drafted by fn-88; awaiting GOV-02 approval.)*` marker, every rule amendment the spec's Decision Context lists, so no rule text contradicts the Veil dependency when task .5 lands it. Old sentences are marked, never deleted. Adopt mode only.

**Size:** S
**Files:** `.plans/UMPIRE4_SPEC.md`, `.plans/UMPIRE4_SPEC_MODEL_ARCH.md`, `.plans/UMPIRE4_DSL.md`, `.plans/UMPIRE4_SPEC_COMPS.md`, `.plans/UMPIRE4_COMPONENTS.md`
**Touches:** [.plans/UMPIRE4_SPEC.md, .plans/UMPIRE4_SPEC_MODEL_ARCH.md, .plans/UMPIRE4_DSL.md, .plans/UMPIRE4_SPEC_COMPS.md, .plans/UMPIRE4_COMPONENTS.md]

### Approach
- `UMPIRE4_SPEC_MODEL_ARCH.md`: §2 principle 7, §3 module tree and MOD list (add `Umpire/Search/Product`, `Selection`, `Backend/Veil`, and `search-backend-isolation`), §9 (the two sentences, the diagram, the "Generic Veil mechanics" paragraph), §10 diagnostic, §11 build-gate sentence, §13 criterion 5, §14 non-goal.
- `UMPIRE4_DSL.md` "Optional Veil checking"; `UMPIRE4_SPEC_COMPS.md` §3 principle 9, the rejected-designs line, §6.4 `Umpire.Search` row and paragraph, §10 non-goal; `UMPIRE4_COMPONENTS.md` C11 and fn-23/24/25 rows gain a fn-88 row.
- `UMPIRE4_SPEC.md`: glossary entries for Search, SearchStats, PlanResult, Exhaustive Search, and `Umpire.Verify.Veil`; MOD-05 and MOD-11 amendment naming `search-backend-isolation`; VER-05 and VER-06 co-attribution to fn-88. Every dotted Lean name added must exist or carry `*(planned: fn-88-…)*` for the MOD-15 test in `tools/umpire/vocabulary/spec_names_test.go`.
- Follow the MOD-11 and MOD-16 amendment blocks as the pattern.

### Investigation targets
**Required:**
- `.plans/UMPIRE4_SPEC.md` MOD-11 and MOD-16 blocks; `.plans/UMPIRE4_SPEC_MODEL_ARCH.md:332-382`
- `tools/umpire/vocabulary/spec_names_test.go`

### Key context
- Drafts only; a human approves under GOV-02.

## Acceptance
- [ ] Every listed location carries a drafted amendment with the fn-88 marker; old text marked, not removed
- [ ] MOD-15 spec-names test and `make umpire-check-plan-index` pass
- [ ] Defer mode: closed as not applicable citing the R1 receipt identity, nothing added

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
