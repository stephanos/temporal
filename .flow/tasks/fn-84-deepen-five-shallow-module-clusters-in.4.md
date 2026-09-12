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
The offline Evidence structure module now returns verdicts: `EvidenceStructure.analyze` builds an opaque structure, and `orderingFault?` / `closureFault?` take an `Audience` (`raw` or `accepted`) and return the first fault with the related identities that audience reports. `factsInOrder` and `linkSupport` are the remaining projections. Raw and Admission only name their failure kind (Raw: one `OrderingFaultKind .raw → ObservationFailureKind` map; Admission: `missingOrderSupport` / `missingClosureSupport`). The finding enum, closure expectations, origin mode and both precedence tables are private and pinned as unexported in `ImportTests`. `Shared.Reachability.reaches` is the single reachability walker. The correlated projection's `reaches` became that walker; the structure (causal cycles, fault receipts) and Raw (record precedence, rule ordering via `rulePrecedes`, a two-line instantiation) call it; the three evaluator copies are deleted.

Equivalence pin: the mutation suite is unchanged and green. A differential corpus of 36000 diagnostics (random raw bundles over two plans, random accepted-envelope mutations) is byte-identical before and after. Two deliberate precedence mutations were detected (canonical instead of written parent order: 175 diffs; supplied instead of bundle record order: test red).

Precedence entries where the raw and accepted tables disagree (each pinned in `Tests/Structure.lean`, origin-mode matrix):
- Global ordering: raw checks fault receipts before every order fault (receipt on a gap record gives `misdirectedFaultReceipt`, accepted gives `sequenceGap`). Accepted has no receipts.
- Source ordering: raw judges a record's causal parents before its own receipt, and receipts after all gaps.
- Accepted only: conflicting duplicate identity, then `uncoveredEvidence` (facts differ from envelope identities), then order faults, then `inconsistentLinkOrder`. Raw reports any duplicate identity, conflicting or not.
- Mixed origins and duplicate sequence: same position, but raw names the records and accepted names none (mixed) or only the later record (duplicate sequence).
- Global closures: raw judges only required kinds in declaration order, then any duplicate closure. Accepted reports an uncovered fact of any kind first (fixture global-closure-1/2: raw `[K]`, accepted `[record-2, Aux]`). An unrequired closure without facts passes raw, and passes accepted only when `lastSequence = 0`.
- Source closures: raw judges duplicate closures, then supplied closures, then uncovered records in bundle order, then required kinds. Accepted judges link duplicates, then uncovered facts in canonical order, then closures, then required kinds (fixture source-closure-a/b: raw `[A, K]`, accepted `[b, B, K]`). Raw bundle order decides which record is named (a-0/a-1 fixture).
- Both faults at once: both audiences judge ordering before closure, and each verdict still reports its own fault.

Decisions taken autonomously:
- `OrderingFaultKind` is indexed by `Audience`, so the raw map has no dead branch.
- Admission keeps its `sourceClosed` check between the two verdicts. The diagnostic is identical to the old combined check.
- Raw passes each record's parents as written, because written order decides the reported parent. A new Evaluation test pins this.
- `analyze` takes `linked : Option LinkedSupport` (envelope identities plus links) and `faultReceipts`. These are beyond the spec sketch, because the accepted coverage check and raw receipts interleave with precedence.
- The end-to-end origin examples stay in `Tests/Evaluation.lean` as regression pins. The per-audience matrix lives in `Tests/Structure.lean`.
- Removed `Evidence.Internal` names are internal, so they are not added to the retired-vocabulary list. Prior fn-84 tasks followed the same practice.
- Touches extension: `model/Shared.lean` (facade import of `Shared.Reachability`), within `model/Shared/**`.

Review follow-up (P3, not landed): the four-constructor closure-mismatch matcher repeats in the raw and accepted closure tables and could share a private helper.

stage: impl-review - ran [2026-09-12..2026-09-12] (claude backend, SHIP first round)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 6b3ff2a92a2c290502cf6b8abba14f4b362fd41d
- Tests: cd model && mise exec -- lake build Umpire.Evidence.Tests Umpire.Evidence.Tests.Mutations UmpireTests Testpilot TestpilotTests (green; mutation suite unchanged), differential diagnostic corpus: 24000 raw bundles x 2 plans and 12000 accepted-envelope mutations, repr of every diagnostic byte-identical before/after (scratch Corpus.lean; sensitivity confirmed by two deliberate precedence mutations), make lint-model (163, baseline 163, all Temporal/API/Proto.lean; import graph, Shared, Umpire.Lint clean), make umpire-check-regression (exit 0; 576 Lean jobs; 9 passing live identities), go clean -cache && make lint-code GOLANGCI_LINT_FIX=false (161 issues, exit 2, inherited baseline 161; no Go files touched)
- PRs: