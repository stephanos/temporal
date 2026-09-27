---
satisfies: [R9, R10]
---
# fn-89-one-contract-rule-per-entity.6 GOV-02 drafts, docs and closing gates

## Description
Draft the Rule glossary Amendment and the SEM-17 Restatement and update the docs that describe Contract rules (R9), and run the closing gates (R10). One finalization task: the gates need everything else landed.

**Size:** S/M
**Files:** `.plans/UMPIRE4_SPEC.md`, `.plans/UMPIRE4_ORDER.md`, `.plans/UMPIRE_CASE_RUNTIME_DESIGN.md`, `common/testing/testpilot/README.md`, `common/testing/testpilot/internal/verification/README.md`, `model/Umpire/ARCHITECTURE.md`, `model/README.md`
**Touches:** [.plans/UMPIRE4_SPEC.md, .plans/UMPIRE4_ORDER.md, .plans/UMPIRE_CASE_RUNTIME_DESIGN.md, common/testing/testpilot/README.md, common/testing/testpilot/internal/verification/README.md, model/Umpire/ARCHITECTURE.md, model/README.md]

## Approach
- GOV-02 drafts in `.plans/UMPIRE4_SPEC.md`, in the existing marker style (indented line under the unchanged original): after the Rule entry (`:78-90`) `*Amendment (drafted by fn-89; awaiting GOV-02 approval.)*` with the spec's Rule text; after SEM-17 (`:166-169`) `*Restatement (drafted by fn-89; awaiting GOV-02 approval.)*` with the full rule text. Do not edit EVD-12/13/21 or the Verdict entry (the Amendment's clause covers them; EVD-21 is already approved).
- Docs (docs-gap findings): Testpilot README intro (`:19-24`, "per violated Rule instance"), `### Every extension` (`:117-154`) a short note on Contract Rule instances, `### A new expression reference` (`:226-242`) the new arm's checklist; verification README (`:3-6,13,21-33,44-48,68`) bind once, per-instance state, Deadline counters, ceilings, Verdict order; `UMPIRE_CASE_RUNTIME_DESIGN.md` `## Contract IR` (`:271-283`); `model/Umpire/ARCHITECTURE.md` `### Typed field lowering` (`:281-296`) the fold and `relation.instance-shape`; `model/README.md:207-212` the Pair example.
- `.plans/UMPIRE4_ORDER.md`: fn-89 moves from the queue to the delivered list at close and later items renumber. The file carries concurrent edits from other sessions; re-read and merge, never overwrite.
- Gates last: `make umpire-check-regression` (physical `TMPDIR`), `LEAN_NUM_THREADS=1 make lint-model` (no new findings over the baseline recorded in ORDER's Gate baselines), `make lint-code-fast`, and `make umpire-check-plan-index` if `.plans/index.json` was touched.

## Investigation targets
**Required**:
- `.plans/UMPIRE4_SPEC.md:56,78-101,159-175,204` — glossary entries, SEM-16/17/19, marker examples
- `common/testing/testpilot/README.md`, `common/testing/testpilot/internal/verification/README.md`

### Carried from fn-89.5 (2026-09-27)
- Replace the duplicated has-rule-instances check in `execution` and `verification` with one shared `ir.HasRuleInstances`.
- Record in the docs that the corpus does not pin the expanded Case-size charge (a fixture large enough exceeds the corpus's 16 MiB small-fixture rule); a Go unit test in `execution` covers it.
- `make umpire-check-regression` was red at fn-89.5 only because of fn-88.5's in-progress Veil pin check (TestUmpireCIWorkflowRunsSeparatedUnitAndLiveProofs); run the closing gates after fn-88.5 lands its fix.

## Acceptance
- [ ] the Amendment and Restatement are drafted with GOV-02 markers; no approved rule text is edited
- [ ] every doc listed describes Rule instances in the spec's vocabulary (no "monitor" for a Rule)
- [ ] `make umpire-check-regression`, `LEAN_NUM_THREADS=1 make lint-model` (no new findings) and `make lint-code-fast` pass

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
