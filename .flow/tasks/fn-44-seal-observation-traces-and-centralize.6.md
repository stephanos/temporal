---
satisfies: [R5, R6]
---
# fn-44-seal-observation-traces-and-centralize.6 Document and verify the accepted trace boundary

## Description
Update the public model documentation for R5/R6 and run the complete final verification once all semantic migrations have landed.

**Size:** S
**Files:** `model/README.md`, `model/ARCHITECTURE.md`, `model/Umpire/ARCHITECTURE.md`
**Touches:** [model/README.md, model/ARCHITECTURE.md, model/Umpire/ARCHITECTURE.md]

### Approach
- Describe Core coordinate ownership, strict one-based semantics, and the single Observation admission handoff in the learner-facing reading paths.
- Correct cross-altitude pipeline wording so Property, Implementation Link, and Run Evaluation consume an already-admitted trace while retaining their own checks.
- Audit changed public declarations, comments, imports, warnings, and axiom dependencies before the aggregate gates.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/ARCHITECTURE.md:149-271` — Property, Observation, and Implementation Link public lifecycle.
- `model/ARCHITECTURE.md:185-248` — cross-library Observation/Link pipeline.
- `model/README.md:232-272` — user-facing offline Observation path.
- `model/Umpire/Observation/Evaluation.lean:160-250` — final public docstrings to cross-check.
- `model/Umpire/Observation/Verdict.lean:285-287` — verdict boundary wording.
- `model/Umpire/ImplementationLink/Application.lean:754-764` — application boundary wording.

### Quick commands
```bash
cd model && mise exec -- lake build Umpire.CoreTests Umpire.Observation.Tests Umpire.ImplementationLink.Tests UmpireTests TemporalModelTests
make umpire-build-model
make umpire-check-regression
make lint-model
make lint-code
```

## Acceptance
- [ ] Public docs explain one Core coordinate authority, strict one-based positions, and the opaque accepted Observation handoff without introducing another language.
- [ ] Observation, Property, Implementation Link, and Run Evaluation documentation assigns validation to the correct stage and retains the no-partial-result guarantees.
- [ ] All changed comments are preserved or accurately revised at their owning invariant; no unrelated documentation is changed.
- [ ] Focused and aggregate Lean builds, regression checks, model lint, and repository code lint pass with no new warning or axiom/trust dependency.
- [ ] No generated file, artifact schema, runtime behavior, persisted byte, fingerprint, or checksum changes.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
