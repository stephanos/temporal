---
satisfies: [R4, R6, R7]
---
# fn-84-deepen-five-shallow-module-clusters-in.4 Evidence structure module returns a verdict

## Description
Turn the offline Evidence structure module (R4) from a bag of findings into a verdict module with an `Audience` parameter, so the raw-bundle and accepted-trace callers stop holding their own finding-to-diagnostic matcher families and precedence orders, and the four copies of causal reachability become one generic walker owned on the `Shared` side.

**Size:** M
**Files:** the structure, raw and admission modules of the offline evaluator under `model/Umpire/Evidence/` (post-fn-82 home; today `Umpire/Observation/Evaluation/{Structure,Raw,Admission}.lean`), the evaluator facade under `Umpire.Evidence`, `model/Shared/CorrelatedProjection.lean` (its `reaches` copy becomes the shared generic walker), the Evidence test modules (`Structure`, `Check`, `Evaluation`, `Mutations`), `model/UmpireTests.lean`, `model/Umpire/ARCHITECTURE.md` (Evidence facade row), `tools/umpire/CONTEXT.md`
**Touches:** [model/Umpire/Evidence/**, model/Umpire/Evidence.lean, model/Shared/CorrelatedProjection.lean, model/Shared/**, model/UmpireTests.lean, model/Umpire/ARCHITECTURE.md, tools/umpire/CONTEXT.md]

### Approach
- Read `.plans/LEAN_GUIDELINES.md` first.
- Baseline: run the mutation suite and the evaluation and check suites and record every diagnostic; the memory entry on the earlier Admission extraction that silently strengthened validation is the reason this is the first step and the last check.
- Before collapsing precedence, prove on every fixture that the raw and accepted orders never disagree on a reachable input. Where they can, make precedence an audience-specific table inside the module so no diagnostic changes; list each such input in the task summary. Both faults firing at once is answered by the same table.
- Interface: `analyze` returns an opaque `EvidenceStructure`; `orderingFault?` and `closureFault?` take the structure and an `Audience` (`raw` or `accepted`) and return the first fault in canonical precedence; `factsInOrder` and `linkSupport` are the two projections the callers still need. Move the finding enum, closure expectations and the origin-mode branch behind the seam.
- Raw and Admission shrink to naming their failure kind for the fault the module identified; the lossy Evidence-Gap admission mapping (code plus optional subject) stays where it is.
- Reachability: the four copies walk different node types (ordinal edges, causal parents, rule ordering), so the shared definition is one generic walker. It lives on the `Shared` side, because MOD-09 forbids `Shared` importing Umpire, and the Evidence structure module imports it; the three evaluator copies are deleted and the correlated projection's `reaches` becomes the walker itself.
- Tests: rewrite the structure test module from asserting finding lists to asserting `orderingFault?`/`closureFault?` per audience; fold the origin-mode matrix from the check and evaluation tests into it; keep the mutation suite unchanged as the regression net.
- Docs: the `Umpire.Evidence` facade row names the verdict interface; add an `Evidence structure` glossary entry that states it belongs to offline Evidence, not to the Testpilot Observation.

### Investigation targets
**Required** (pre-fn-82 lines at HEAD ebb94a44e; fn-82 .6 moves these files):
- `model/Umpire/Observation/Evaluation/Structure.lean:30-72, 96-104, 144-154, 258-262, 344-391` — findings, analysis record, reachability, closure findings, analyzeStructure
- `model/Umpire/Observation/Evaluation/Raw.lean:142-162, 490-605, 609-692, 795-796` — reachability copies, matcher families, validators, call site
- `model/Umpire/Observation/Evaluation/Admission.lean:84-166, 170-249, 563` — mirror matchers, ordering and closure validators, call site
- `model/Umpire/Observation/Tests/Structure.lean` and `Tests/Mutations.lean` — the 20 direct calls to replace and the 571-line regression net
- `model/Shared/ScopedProjection.lean:148-163` (post-fn-82 `CorrelatedProjection.lean`) — the fourth reachability copy, future home of the generic walker

**Optional:**
- `model/Umpire/Case/Correlated.lean` `lower` — one-entry admission pattern
- `model/ModelLint/ImportGraph.lean` — the import policy that allows Umpire to import Shared
- `.flow/memory/bug/integration/behavior-neutral-refactors-must-not-2026-09-04.md`

### Key context
- fn-82 .6 renames the offline evaluator to `Umpire.Evidence`; write the new identifiers in that vocabulary and avoid `Umpire.Observation*` names.
- MOD-09: `Shared.*` imports neither `Umpire.*` nor `Temporal.*`.
## Acceptance
- [ ] the structure module exports `analyze`, `orderingFault?`, `closureFault?`, `factsInOrder`, `linkSupport` and `Audience`; the finding enum, closure expectations and precedence are not exported
- [ ] the raw and accepted modules contain no finding-matcher family and no reachability definition; one generic reachability walker on the `Shared` side serves the correlated projection and the Evidence structure module; `make lint-model` confirms Shared imports no Umpire module
- [ ] every diagnostic in the mutation suite is byte-identical to the baseline; the origin-mode matrix passes through the new interface; inputs on which the two old precedence orders differed are listed in the summary with their audience-table entries
- [ ] focused: `lake build Umpire.Evidence.Tests Umpire.Evidence.Tests.Mutations` (fn-82's names) and `lake build UmpireTests` green; `make lint-model` green
- [ ] facade row and `CONTEXT.md` entry updated; documentation gate passes; `make umpire-check-regression` green
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
