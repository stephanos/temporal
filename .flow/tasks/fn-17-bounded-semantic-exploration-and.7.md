---
satisfies: [R8]
---
# fn-17-bounded-semantic-exploration-and.7 Publish Exploration facades and protocol documentation

## Description
Publish only the `Umpire.Exploration` and Temporal semantic facades plus protocol/coverage documentation. Command ergonomics, durable resume, and runtime claims belong to fn-33.

**Size:** M
**Files:** `model/Umpire.lean`, `model/Temporal.lean`, `model/README.md`, `model/Umpire/ARCHITECTURE.md`, `.plans/UMPIRE4_SPEC_COMPS.md`
**Touches:** [model/Umpire.lean, model/Temporal.lean, model/README.md, model/Umpire/ARCHITECTURE.md, .plans/UMPIRE4_SPEC_COMPS.md]

### Approach
- Export the vertical Exploration facade and exact Temporal adapters without flattening implementation modules.
- Document the checked-source-to-selected-spec/report API path, strategy/budget grammar, semantic-versus-case coverage, pinned precedence, exact-certificate boundary, pure protocol equations, and in-memory resume compatibility.
- State explicitly that closed v1 has no mutation language, adaptive corpus, priority feedback, persisted reader, or command surface.
- Update implementation status only for delivered algorithms/state/report/protocol; keep runtime campaign, replay, minimization, promotion, and qualification milestones separate.
- Coordinate fn-5's generated glossary terms without adding a separate glossary or list/explain implementation.
- Preserve existing comments and point command users to fn-33's eventual `umpire-fuzz` surface rather than creating `temporal-model-explore` or `umpire-explore`.

### Investigation targets
**Required** (read before coding):
- `model/README.md:130-165` — current model use walkthrough
- `model/Umpire/ARCHITECTURE.md:145-235` — Query/search/planning/artifact documentation
- `.plans/UMPIRE4_SPEC_COMPS.md` — current component and milestone ownership
- `model/Umpire.lean`, `model/Temporal.lean` — public facade pattern after fn-16

## Acceptance
- [ ] Public facades expose the pure checked source, selection, report, and protocol APIs without internal-module leakage.
- [ ] Documentation distinguishes model coverage from execution evidence, selection termination from protocol drainage, and in-memory state from fn-33 durable campaign checkpoints.
- [ ] Documentation states the exact-certificate trust boundary and that projection fixture equality remains fn-5's separate check.
- [ ] No Makefile, executable, model-local Makefile, extra glossary, persisted reader, CI file, or Umpire3 reference/use is added.
- [ ] Focused suites plus `UmpireTests` and `TemporalModelTests` pass with existing comments preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
