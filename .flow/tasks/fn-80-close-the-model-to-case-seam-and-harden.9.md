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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
