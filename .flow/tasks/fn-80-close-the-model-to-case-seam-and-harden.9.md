---
satisfies: [R7]
---
# fn-80-close-the-model-to-case-seam-and-harden.9 Update docs, draft new spec rules, and add the roadmap entry

## Description
Implements R7 (spec §R7). Reconciles every document the docs-gap scan found false after fn-80, drafts three new UMPIRE4_SPEC rules under fresh IDs for human approval (GOV-01, GOV-02), adds the fn-80 roadmap entry, and runs the full gate set as the final check.

**Size:** S
**Files:** `.plans/UMPIRE4_SPEC.md`, `.plans/UMPIRE4_SPEC_COMPS.md`, `.plans/UMPIRE4_ORDER.md`, `model/ARCHITECTURE.md`, `model/README.md`, `model/Umpire/ARCHITECTURE.md`, `model/Temporal/Feature/Nexus3/Nexus.md`, `model/Temporal/Feature/Nexus3/Integration.md`, `common/testing/testpilot/README.md`, `common/testing/testpilot/internal/execution/README.md`, `common/testing/testpilot/internal/verification/README.md`, `common/testing/testpilot/temporal/worker/README.md`, `common/testing/testpilot/temporal/server/README.md`, `tools/umpire/CONTEXT.md`
**Touches:** [.plans/UMPIRE4_SPEC.md, .plans/UMPIRE4_SPEC_COMPS.md, .plans/UMPIRE4_ORDER.md, model/ARCHITECTURE.md, model/README.md, model/Umpire/ARCHITECTURE.md, model/Temporal/Feature/Nexus3/*.md, common/testing/testpilot/**/README.md, tools/umpire/CONTEXT.md]

### Approach
- New rules, each with a new ID (never renumber): (a) Driver-realized faults — declared instruction, Profile capability gate, one `FAULT_INJECTED` Run Event per instruction, a requested fault proves nothing until that event exists (EVD-17 currently allows only unary RPC transport and SDK entrypoints); (b) horizon units — two admitted bounds, `rule_events` semantics, both paths tick one counter, `elapsed_milliseconds` retained but host-clock dependent (ties to EVD-07); (c) AUT-08 clarification that macro-derived ordered domains and enumerators count as author-provided. Mark all three as pending human approval in the task receipt.
- Statements to correct: `model/Umpire/ARCHITECTURE.md` "request-only faults" and the closed instruction list; `internal/verification/README.md` elapsed-only expiry; `Nexus3/Integration.md` "no raw Run Event counts" and the Producer description; `Nexus3/Nexus.md` "success-only forms compile"; `internal/execution/README.md` `Run` closure paragraph (R6); `temporal/worker/README.md` registration lifetime; `model/README.md` fixture counts and facade classes; `instruction.proto:60` closed-table comment (verify task .2 edited it).
- `tools/umpire/CONTEXT.md`: add Fault, Horizon, Profile, Capability entries in the Definition/Avoid shape.
- `.plans/UMPIRE4_ORDER.md`: fn-80 section under Current work after fn-77, noting R1/R5 serialize behind fn-77.10 and that fn-80 R4 owns the vision's "inject one fault" acceptance criterion; adjust the fn-67 and fn-70 rows' next actions.
- Final gates: `make lint-model`, `make umpire-check-regression`, `make lint-code`.

### Investigation targets
**Required** (read before coding):
- `.plans/UMPIRE4_SPEC.md` — GOV-01/02, EVD-07, EVD-12, EVD-17, EVD-18, AUT-08, MOD-12
- `.plans/UMPIRE4_ORDER.md` — Current work section shape

**Optional** (reference as needed):
- `tools/umpire/internal/retiredvocabulary/check.go:264` — banned tokens the new doc text must avoid

## Acceptance
- [ ] Every doc listed in Files is updated and no doc asserts faults are intent-only, horizons are elapsed-only, or the Nexus3 syntax is success-only
- [ ] Three new rule IDs drafted in `.plans/UMPIRE4_SPEC.md` with no renumbering, flagged for human approval in the receipt
- [ ] `.plans/UMPIRE4_ORDER.md` has an fn-80 entry and the fn-67/fn-70 next actions reference it
- [ ] `tools/umpire/CONTEXT.md` gains Fault, Horizon, Profile, Capability entries; `make umpire-check-retired-vocabulary` passes
- [ ] `make lint-model`, `make umpire-check-regression`, and `make lint-code` pass; axiom inventory matches baseline

## Done summary
Every document this spec falsified now says what the tree does.

Faults: `Umpire.Space` authors the intent and lowers it; the Driver realizes it and records one
`FAULT_INJECTED` event per realized outage, and a requested fault still proves nothing until that
event exists. The worker README describes the dedicated per-Run group, the suppressed fatal path
during the stop window, and the cleanup that resumes before releasing the hold.

Horizons: a bounded-liveness rule declares exactly one bound. The verification README and
`Nexus3/Integration.md` now say which unit means what, that both tick through one helper so the
online and offline evaluation of the same Run answer identically, and that a scoped clause counts
operation transitions and falls back to neither classic bound.

The async-nexus Case: `model/README.md`, `Nexus3/Integration.md` and `Nexus3/Nexus.md` now describe
`produce`, a Contract that carries no monitor rule, the operation-scoped clauses the checked Property
lowers into, and the `ScopedEvidence` lift. `Nexus.md` no longer says the Producer binds Facts by
spelling; it does not, so a rename is a fixture regeneration and no Producer edit at all.

Also corrected: the execution README's `Run` closure paragraph (R6 returns the recorder close error
beside an unchanged Run and Verdict), and the facade and execution READMEs gained the evidence lift
and the fault instruction. `tools/umpire/CONTEXT.md` gained Horizon, Fault, Profile and Capability
in the Definition/Avoid shape.

Three rules are drafted under new IDs and are **pending human approval under GOV-02**; each carries
that marker in its own text and none is approved:
- **EVD-20 — Driver-realized faults.** A declared instruction, a Profile capability, one recorded
  event per realized outage, none for a refused one, and resources no other Run shares.
- **EVD-21 — Horizon units.** One positive bound; `rule_events` counts what the Run recorded and is
  the bound a conclusion may rest on; `elapsed_milliseconds` stays admitted and host-clock dependent;
  one shared helper ticks both; a scoped clause counts operation transitions and never falls back.
- **AUT-09 — Macro-derived finite domains.** A domain a command macro elaborates from the
  constructors of an inductive the author named is author-provided under AUT-08; the macro must not
  admit an undeclared spelling or weaken the obligations AUT-08 names.

`.plans/UMPIRE4_ORDER.md` records what fn-80 delivered after its completion review — .14's evidence
path and the closed `bounded-completion-is-model-only` gap, .4's scoped async-nexus Case, .8's fault
Case with its two recorded deferrals, and .9 itself — and adjusts the fn-82, fn-70 and fn-67 rows.

One reading recorded rather than silently changed: `UMPIRE4_COMPONENTS.md` and `UMPIRE4_DSL.md` still
say `Umpire.Space` authors "request-only faults". That stays true of the authoring surface, which
declares intent; realization belongs to Testpilot, which is what EVD-20 now states. Neither file is
in this task's list and neither was edited.

Two files on the Files list were searched and left alone because nothing in them was falsified:
`model/ARCHITECTURE.md` and `common/testing/testpilot/temporal/server/README.md` assert nothing
about faults, horizons or the Nexus3 syntax. `.plans/UMPIRE4_SPEC_COMPS.md` did carry one stale
claim — `Umpire.Space` marked "Planned" — and now reads Delivered, with the fault-intent lowering
named.

Reviewer findings addressed in a second commit: the fn-80 section's count is 14 of 15 and its
pre-`.14` narrative sits under a History heading in past tense, so a reader is not handed three
states for one spec in one screen; and the `CONTEXT.md` entry is **Profile capability**, saying
explicitly how it differs from the spec's Capability Contract and from a Contract's scoped
capability, rather than pinning one of three live senses of a bare word.

The three drafted rules — **EVD-20**, **EVD-21** and **AUT-09** — are **pending human approval under
GOV-02** and are marked as such in the spec text itself. None is approved.

A third round addressed the reviewer's accuracy slips: fn-70's next action names
`temporal.DeriveProfile` and the test-local `bindCase`/`runCase` helpers where they actually live,
the fn-82 dependency lists the four tasks it means rather than a range that includes the superseded
`.5`, and the fn-77 delivered row records that `.14` closed `bounded-completion-is-model-only` and
what replaced it, rather than leaving the roadmap asserting a gap the tree no longer has.

stage: impl-review - ran [836dbd43..7de8ecc3] SHIP
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: c49b2b56b5db9873d748bd08bd8858dc422cd0ae, 9ac0b1789835eeaf97f6130481eed34b4e5bdc84, 7de8ecc3190397815442ac6fc0d006d4428d9b17
- Tests: make lint-model (169 errors, unchanged baseline, all in generated Temporal/API; import-graph clean), make umpire-check-regression (includes umpire-check-retired-vocabulary and umpire-check-live-tests), make umpire-check-live-tests (empty failure set across 6 passing identities), CGO_ENABLED=0 go test -tags test_dep ./common/testing/testpilot/... ./tests/testcore/testpilot/... ./tools/umpire/..., go test -tags 'test_dep integration' ./tests -run '^TestTestpilot', make lint-code GOLANGCI_LINT_FIX=false (128: errcheck 1, govet 4, revive 106, staticcheck 17 - unchanged baseline), go vet -tags test_dep ./... (15 pre-existing diagnostics, unchanged), go run ./tools/planindex (45 findings, all .flow spec-dependency drift and unregistered .plans documents, none from this change)
- PRs:
stage: plan-sync - skipped(config: planSync.enabled != true)
