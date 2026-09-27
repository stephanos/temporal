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
Drafted fn-88 GOV-02 amendments, under the `drafted by fn-88; awaiting GOV-02 approval` marker, in the five plan documents. Old text is kept and each amendment names what it supersedes. UMPIRE4_SPEC.md gains a new MOD-17 (search-backend isolation, planned under fn-88; a new ID per GOV-01) and amendments to MOD-05, MOD-11, Exhaustive Search, Search (which covers SearchStats and PlanResult), `Umpire.Verify.Veil`, VER-05, and VER-06. MODEL_ARCH §2.7, §3 tree and MOD list, §9 (two sentences, a new diagram, Generic Veil mechanics, the last two rule bullets), §10, §11, §13.5, and two §14 non-goals are amended too. So are the DSL principle bullet, "Optional Veil checking", the verification contract, and the non-goals. SPEC_COMPS §3.9, §6.4 rows and paragraph, the §6.5 row, §11, the parallel track, and §15 non-goals are amended. COMPONENTS gets a C11 row, a new fn-88 snapshot row after fn-25, the C11 toolchain paragraph, and the rejected-designs line. The drafts carry the R22 amendments. The model's toolchain is Veil's declared Lean 4.32.0 and moves with the pinned commit. Node and npm are model build prerequisites. The checker entry is `IO` and runs during command elaboration. Witnesses are kernel-replayed. Absence answers are trusted from the checker, with the differential test as oracle and 64-bit state-hash dedup as the stated trust assumption. Mapping notes: the spec's "SPEC_COMPS rejected-designs line" exists only in COMPONENTS (amended there), and "SPEC_COMPS §10 non-goal" is §15 Non-goals. Follow-up: fn-23 re-scoping stays a separate user-approved edit.

stage: impl-review - ran [2026-09-27..2026-09-27] triage_skip SHIP (docs-only)
## Evidence
- Commits: 0bf2aa7f1917c459673b5f89f99789144d872b76
- Tests: baseline: green (make umpire-check-plan-index; go test -tags test_dep ./tools/umpire/vocabulary/; make umpire-check-retired-vocabulary), make umpire-check-plan-index, go test -count=1 -tags test_dep ./tools/umpire/vocabulary/, make umpire-check-retired-vocabulary, Lean Quick commands not run: docs-only .plans diff, per conductor scope (no builds beyond plan-index/vocabulary)
- PRs: