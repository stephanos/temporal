---
satisfies: [R4, R5, R6, R9]
---
# fn-78-typed-temporal-authoring-and-checked.6 Implement shared scoped-obligation semantics and proofs

## Description
Implement D4's single checked scoped-obligation semantic kernel and its correspondence to existing Property meaning. Model, incremental, and offline evaluation must share one passive `compile / consume / close` transition contract over admitted projected semantic steps.

**Size:** L
**Files:** `model/Umpire/Property/{Language,Authoring,Check,Evaluation,Trace,Tests/**}.lean`, focused scoped-obligation modules/tests under `model/Umpire/Property/`, `model/Umpire/Observation/{Verdict,Evaluation/**}.lean`, `model/Umpire/Property/COMPATIBILITY.md` only if a semantic migration is required
**Touches:** [model/Umpire/Property/**, model/Umpire/Observation/Verdict.lean, model/Umpire/Observation/Evaluation/**, model/Umpire/Property/COMPATIBILITY.md]

### Approach
- Add a checked clause with trigger/response predicates, immutable correlation key, semantic clock, natural bound, and endpoint policy.
- Compile supported clauses into independent obligations. At one coordinate, create triggers before evaluating responses so a matching response can discharge a bound-zero obligation immediately.
- Count only admitted labeled transitions in the captured scope; count labeled self-loops and ignore unrelated operations, polls, duplicate reads, and acknowledgements.
- Reuse the same transition function for whole-stream, incremental, and offline evaluation; distinguish deliberate finite close from incomplete runtime-prefix close.
- Prove agreement with existing `eventuallyWithin` closed-trace semantics for the supported projection and state the incremental/prefix correspondence explicitly.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Property/Language.lean` — existing checked clause vocabulary
- `model/Umpire/Property/Authoring.lean:65-109` — typed constructors
- `model/Umpire/Property/Check.lean:505-599` — checked admission and diagnostics
- `model/Umpire/Property/Evaluation.lean:572-655` — existing bounded temporal reference evaluator
- `model/Umpire/Observation/Verdict.lean` — evidence-backed verdict mapping

### Key context
- Closing a selected finite trace with a live obligation violates; closing an incomplete runtime prefix without deadline evidence remains inconclusive.
- Never use runtime timeout or search-budget exhaustion as a semantic transition.
## Acceptance
- [ ] One checked scoped clause records typed trigger/response, correlation key, semantic clock, bound, endpoint policy, source, and stable clause ID.
- [ ] Independent obligations cover repeated triggers, one response discharging every matching in-bound obligation, two interleaved operations, and counted labeled self-loops without cross-scope ticks.
- [ ] Bounds zero, one, and larger pass at the trigger coordinate and inclusive deadline and violate only after an unanswered deadline step; later response cannot repair violation.
- [ ] Invalid model traces reject before evaluation; deliberate finite close and incomplete runtime-prefix close produce their specified different outcomes.
- [ ] Whole-stream, incremental, and offline evaluation agree across every chunk boundary, including partial projected evidence.
- [ ] Checked proofs connect existing Property closed-trace meaning, scoped projection, obligation transitions, and prefix/close outcomes; bounded differential tests use an independent reference.
- [ ] Unsupported predicates, keys, clocks, scopes, endpoint combinations, or numeric bounds reject the whole requested lowering with the responsible clause ID and no partial Case output.
- [ ] Existing Property APIs/operators and default-empty canonical bytes/fingerprints remain unchanged; changed declarations pass transitive axiom audits.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
