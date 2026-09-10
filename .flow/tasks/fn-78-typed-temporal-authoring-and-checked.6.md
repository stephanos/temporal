---
satisfies: [R4, R5, R6, R9]
---
# fn-78-typed-temporal-authoring-and-checked.6 Implement shared scoped-obligation semantics and proofs

## Description
Nexus operation cancellation is deferred to fn-79. Implement and qualify this generic task with non-cancellation, multi-operation fixtures; no cancellation adapter or capability is a prerequisite.

Implement D4's single checked scoped-obligation semantic kernel and its correspondence to existing Property meaning. Model, incremental, and offline evaluation must share one passive `compile / consume / close` transition contract over admitted projected semantic steps.

**Size:** L
**Files:** `model/Umpire/{Property,Observation}.lean` facade imports, `model/Umpire/Property/{Language,Authoring,Check,Evaluation,Trace,Tests/**}.lean`, focused scoped-obligation modules/tests under `model/Umpire/Property/`, `model/Umpire/Observation/{Projection,Verdict,Evaluation/**}.lean`, `model/Temporal/Feature/Nexus3/{Testpilot,Tests}.lean` for the existing Producer rejection guard, `model/Umpire/Property/COMPATIBILITY.md` only if a semantic migration is required
**Touches:** [model/Umpire/Property.lean, model/Umpire/Observation.lean, model/Umpire/Property/**, model/Umpire/Observation/Projection.lean, model/Umpire/Observation/Verdict.lean, model/Temporal/Feature/Nexus3/Testpilot.lean, model/Temporal/Feature/Nexus3/Tests.lean, model/Umpire/Observation/Evaluation/**, model/Umpire/Property/COMPATIBILITY.md]

### Approach
- Add a checked clause with trigger/response predicates, immutable correlation key, semantic clock, natural bound, and endpoint policy.
- Compile supported clauses into independent obligations. At one coordinate, create triggers before evaluating responses so a matching response can discharge a bound-zero obligation immediately.
- Count only admitted labeled transitions in the captured scope; count labeled self-loops and ignore unrelated operations, polls, duplicate reads, and acknowledgements.
- Expose the admitted projector’s existing scope/key/initial-state bindings through read-only accessors. Until task 7 supplies portable lowering, reject scoped clauses at the existing Nexus3 Producer admission boundary with the responsible clause ID and no Case output; this does not add cancellation behavior.
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
Implemented checked scoped bounded obligations and a shared passive compile/consume/close runtime, with a generic projector adapter and non-cancellation multi-operation fixtures. Independent conductor review returned SHIP with no findings; no staging or commits were performed in the working checkout.

The actual per-operation Execution carries proofs that its retained obligations equal the kernel fold and that its coordinates equal the existing checked Property projection. Execution.closed_property composes these invariants with checked_eventuallyWithin_agrees to connect actual runtime execution to the existing eventuallyWithin evaluator. Each admitted step is projected once and appended by a proved map/append lemma; production admission does not rebuild the cumulative reference trace or compare final verdicts. Typed predicate-to-pattern agreement is checked once at compilation. The supported fragment includes selected-action triggers and outcome, resulting-state, and expectation-fact present/text-equality responses, including duplicate fact occurrences.

The kernel creates triggers before responses, includes the deadline coordinate, retains independent repeated obligations and irreversible violations, and ticks only the operation receiving an admitted labeled transition. Open runs always use prefix answers; deliberate finite close turns remaining obligations into violations. Missing causal evidence forces unresolved unless a violation is already established. Transition, retained-obligation, and work limits fail atomically; retained checked inputs and coordinates are bounded by the transition ceiling.

The Observation adapter consumes only newly emitted semantic steps. Its exact append theorem covers successful and failed chunked admission, while focused fixtures exercise all chunk boundaries of reordered partial evidence, duplicate satisfied/violated self-loops, stuttering reads, and interleaved operations. An independent enumerated-position differential oracle covers all five-coordinate Boolean streams across bounds zero through four. Existing default-empty Property canonical bytes remain unchanged; scoped declarations canonicalize clause order. Axiom guards audit production consume/chunk APIs and both correspondence theorems.

Until portable lowering is provided by task7, the actual Nexus3 Producer rejects any scoped clause before producing a Case, with the responsible clause ID and source. Case.Compiler has no Property input, so the guard belongs in Temporal/Feature/Nexus3/Testpilot.lean. No cancellation capability or adapter was implemented.

Extra owner paths beyond the Property subtree: model/Umpire/Observation/Projection.lean (read-only admitted scope/key/initial-state getters), model/Umpire/Observation/Evaluation/Scoped.lean (adapter), model/Umpire/{Property,Observation}.lean (facade imports), and model/Temporal/Feature/Nexus3/{Testpilot,Tests}.lean (whole-Case rejection and regression). Existing Flow changes and the conductor's System/Nexus/ImplementationLink.lean repair are outside this worker's implementation.

Baseline: green focused Property/Projection build (36 jobs), /tmp/fn78-task6-baseline.log and .exit. Final focused gate: green, 152 jobs, /tmp/fn78-task6-gate-focused3.log and .exit; this includes Property.Tests.Scoped.Evidence and Nexus3 producer tests. Earlier iterative red logs are superseded by this final focused receipt. Full build passed 502 jobs, exit0 (/tmp/fn78-task6-build2.log and .exit). Final LEAN_NUM_THREADS=1 make lint-model passed exit0 (/tmp/fn78-task6-lint-model2.log and .exit), including all rebuilt scoped regression/axiom modules. Final git diff --check passed exit0. The full build preceded only removal of redundant eq_comm simp arguments in three proof branches; the final lint gate verifies this final source. The initial full-build facade import-order error and initial lint proof warnings were corrected; their red receipts are superseded by build2 and lint-model2.

Inherited make lint-code GOLANGCI_LINT_FIX=false failure: exit2 with 1284 pre-existing Go issues, captured by conductor at /tmp/fn78-draft-repair-lint-code.log and .exit. No Go files changed and this baseline was not rerun.

stage: impl-review - ran (model: gpt-6-astra at medium)
Review receipt: /tmp/fn78-task6-impl-review.json; session 01a07fa7-394c-71f2-b75b-995c4ec3b59c; SHIP with no findings. Review used a scratch-clone commit range; the real workspace remains uncommitted.
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: BASELINE GREEN: cd model && LEAN_NUM_THREADS=1 lake build Umpire.Property.Tests Umpire.Observation.Tests.Projection; exit0,36 jobs; /tmp/fn78-task6-baseline.log and .exit, GREEN: cd model && LEAN_NUM_THREADS=1 lake build Umpire.Property.Tests Umpire.Observation.Tests.Projection Temporal.Feature.Nexus3.Tests; exit0,152 jobs; /tmp/fn78-task6-gate-focused3.log and .exit; includes Scoped.Evidence, GREEN: LEAN_NUM_THREADS=1 make umpire-build-model; exit0,502 jobs; /tmp/fn78-task6-build2.log and .exit; followed only by proof-simp cleanup checked by final lint, GREEN: cd model && LEAN_NUM_THREADS=1 lake --wfail lint Umpire.Property.Evaluate --builtin-only --lint-only=.all,.extra,-.missingDocs; exit0; /tmp/fn78-task6-lint-proof2.log and .exit, GREEN: LEAN_NUM_THREADS=1 make lint-model; exit0; /tmp/fn78-task6-lint-model2.log and .exit; includes rebuilt scoped regression and axiom guard modules on final source, GREEN: git diff --check; exit0; /tmp/fn78-task6-diff-check.log and .exit, INHERITED RED: make lint-code GOLANGCI_LINT_FIX=false; conductor pre-edit baseline exit2,1284 existing Go issues; /tmp/fn78-draft-repair-lint-code.log and .exit; not rerun because no Go sources changed, SHIP: independent codex implementation review, gpt-6-astra medium; /tmp/fn78-task6-impl-review.json; no findings; scratch clone review of exact implementation snapshot
- PRs: