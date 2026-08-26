---
satisfies: [R8]
---
# fn-17-bounded-semantic-exploration-and.7 Wire root exploration ergonomics and synchronize documentation

## Description
Complete R8 with root-only command wiring, public facades, user documentation, and precise C8 status updates.

**Size:** M
**Files:** `model/Umpire.lean`, `model/Temporal.lean`, `Makefile`, `model/README.md`, `model/Umpire/ARCHITECTURE.md`, `.plans/UMPIRE4_COMPONENTS.md`
**Touches:** [model/Umpire.lean, model/Temporal.lean, Makefile, model/README.md, model/Umpire/ARCHITECTURE.md, .plans/UMPIRE4_COMPONENTS.md]

### Approach
- Export the vertical Exploration facade and exact Temporal adapter without flattening implementation modules.
- Add only the root `umpire-explore` Make target with required SPACE/STRATEGY/BUDGET and optional STRENGTH/SEED forwarding.
- Document the checked-space-to-selected-spec/report user path, strategy/budget grammar, semantic-versus-case coverage, pinned precedence, pure model-only scope, and fresh versus persisted resume boundary.
- Update C8 implementation status only for delivered algorithms/state/report/command; keep Milestone C live replay/minimization/promotion incomplete.
- Coordinate fn-5's generated glossary terms without adding a separate glossary or list/explain implementation.

### Investigation targets
**Required** (read before coding):
- `Makefile:988-1032,1254` — root-only model targets and phony registration
- `model/README.md:130-165` — current model use walkthrough
- `model/Umpire/ARCHITECTURE.md:145-235` — Query/search/planning/artifact documentation
- `.plans/UMPIRE4_COMPONENTS.md:23-54,362-390,641-660` — status and milestone boundaries
- `model/Umpire.lean`, `model/Temporal.lean` — public facade pattern after fn-16

### Acceptance
- [ ] Root and direct executable produce identical bytes/status for the unpinned coverage-guided Nexus fault-matrix invocation; pinned precedence remains covered by the pure engine suite because the command accepts no pinned input.
- [ ] Missing/invalid Make variables fail concisely before Lake execution.
- [ ] Documentation never calls model coverage execution evidence or Milestone C complete.
- [ ] No model-local Makefile, extra glossary, persisted reader, CI file, or Umpire3 reference/use is added.

## Acceptance
- [ ] Public facades, root command, docs, and roadmap satisfy R8.
- [ ] All focused suites plus `UmpireTests`, `TemporalModelTests`, and root smoke command pass.
- [ ] Existing comments are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
