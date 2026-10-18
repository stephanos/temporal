---
satisfies: [R1, R2, R3]
---
# fn-196-the-scala-checker-is-the-oracle.2 DSL completion: replay, find with a horizon, seeded sample, refuted, vacuity and the gate budget

## Description
**DSL completion phase (owner, 2026-10-17).** Complete the Scala checker's executable questions and error contracts for fn-195's sections: positioned pinned replay, horizon-bounded find, seeded random-walk sampling, negative controls, complete-only vacuity and gate-owned default/per-suite budgets. Keep one execution engine and one common Result/Report vocabulary; the module move in .1 alone did not finish R1.

**Size:** L

**Files:** native checker entrypoints, results and existing execution primitives plus focused Replay/Spaces/Sampling/Vacuity/budget helpers and tests; gate budget validation/transport and its tests; focused sample-owned public-question controls and separate-library clients.

**Touches:** `model/framework/checker/**`, `model/framework/check/**`, `model/framework/samples/**`, `model/framework/REFERENCE.md` (new public checker type inventory only)

**Evidence-driven scope addition (conductor, 2026-10-17).** The full boundary gate reproduced ReferenceSurfaceTest refusal for the ten newly public checker types. Add those names with concise user-facing execution/evidence descriptions to the reference's closing inventory; no Scala-file references, no unrelated reference cleanup, and no oracle-direction changes reserved for .5. Preserve the coverage gate unchanged. `refuted` also forwards Monitor claims to the existing monitor execution boundary, preserving ordinary verdicts, policy budgets, starts, caller naming and the existing cache negative-control path; no assertion inversion or second execution engine.

### Contracts and verification
- `replay(instance, path)` consumes pinned operations with positioned expectations. A failing second expectation names its own line/text/state. Disabled first steps and disabled later suffixes refute with the step/state, never vacuously verify. Success retains the concrete executed path. Holes and state-budget cuts are unknown under the shared incompleteness contract, not false success. Preserve the old pinned-query semantics until their separate fn-195 retirement; add the new contract rather than changing old callers to fit it.
- `find(space, horizon)(target)` stops on the first witness and returns found, notFound or unknown with actual evidence. Finishing the requested horizon without a witness is notFound; a budget cutoff is unknown; decisive evidence before a cut stands. Refuse absent targets and horizons below one. Cover target at the boundary, one beyond, holes/cuts, position-sensitive/repeated-state spaces and no effect execution beyond the horizon.
- `sample(space, count, seed)` judges the selected seeded walks against the instance's claims, not an unrelated exhaustive verifyAll. Preserve ordered walks, seed and failing index in common evidence; repeated seed reproduces paths. Refuse counts below one. Report finite experiment completion honestly, without implying successful finite walks prove exhaustive safety or unbounded progress. A Space's finite step bound is explicit, matching fn-195's `explore(steps = 6)`; do not invent a hidden default10. Keep that bound distinct from the state budget and the find horizon; refuse a requested find horizon above the declared Space bound rather than silently search less and call it notFound. Cover that refusal and carry the actual step bound in evidence.
- `refuted(machine, claim)` returns the underlying ordinary claim Result; fn-195 owns the assertion adapter requiring refuted. Verified and unknown do not count as negative-control success. Keep munit out of the execution module. Reproduce the Leaky Bank control and preserve its original mathematical failure.
- A model effect throwing becomes a refuted Result carrying its message and pre-step state/action. Convert at the execution boundary, not with a broad catch that changes existing independent monitor/progress predicate exception behavior. Preserve holes as unknown and metadata/catalog/refusal boundaries.
- VerifyAll retains R15 ordinary/scoped safety and complete-only vacuity: admitted actions never fired and after(action) claims never triggered. Preserve exact instance/claim ownership and suppress completeness-based warnings on holes/cuts.
- Every public answer retains actual budget, completion and concrete witness/counterexample; replay failures include position evidence and sampling retains its experiment coordinates. Public macros preserve first checked stable-holder naming at the external caller, never an internal adapter parameter's name. Finite-only class callers and object/derived/composition controls remain valid.

### Gate budgets and task ownership

The gate owns a positive default plus fully qualified suite overrides, never Model configuration or ordinary authored test-local policy. Choose one minimal documented configuration format using established dependencies; no new library is authorized. Reject malformed, duplicate, nonpositive/overflowing and unknown-suite entries before any compiler/test command. Validate keys against all actual source-discovered suites, not only selected or unheld ones; no second hand-maintained suite catalog. Use existing Tools.withEnvironment transport for the validated policy. Two suites in one child JVM must resolve distinct budgets and every resulting question must use the resolved value. No active-suite global/thread-local that loses asynchronous context. Keep explicit small-budget engine contract tests, not a backdoor authored policy surface.

FrameworkLoop already discovers suites before tool execution; ChangeCheck needs the same policy and preflight for its framework route. Do not execute its held Temporal route. The checker owns reusable execution/configuration contracts; fn-195 owns ModelSuite construction/registration, four section builders, expect macro, syntax, suite assertions and removal of old Query/Scenario APIs. Supply and prove the policy resolver/transport/question boundary now; carry actual future ModelSuite wiring/discovery into fn-195 explicitly rather than claim that a nonexistent suite adapter is implemented. Source discovery currently handles direct FunSuite classes; future ModelSuite discovery must follow actual declarations, not a sample-name list.

No production DSL edit is pre-authorized: execution inputs can live in the checker and reuse existing DSL declarations/private framework hooks. If a narrow declaration seam is genuinely required, report the exact file/change to the conductor before editing; preserve declaration-only DSL+realize compilation without checker dependencies. No prototype suite, second explorer, duplicated stepping engine or new unrelated author API.

### Baseline and gates

Run an own focused pre-edit checker baseline. Integrated fn-194 is now4/4 native tasks done; last source/docs checkpoint `f431b252da`, closure `f8d5d0f39a`, root milestone `c118d42f9d`. The current native baseline is635 tests, with seven class-lint, 31 retained RuleSuite, 46 boundary/loop controls and166 classified suites. Preserve registrations/assertion purposes and all sample mathematical answers. Bank retains160 input classes over -1..supply+1 and its exact Leaky trace; Voting retains ballots0/1/sentinel-1, majority behavior, alternate unanimous251 graph and unknown cross-refinement; TwoPhaseCommit retains bound3/longest3/depth10 and EagerCommit negative.

Use real red tests before supporting fixes, then focused and full affected native DSL/realize/checker/samples, class-lint/RuleSuite/boundary/loop/source-corpus checks. Fresh declaration-only/core/plugin and six-sample+Inputs realization-mode source/separate-library clients stay valid under unchanged strict flags. New public questions get fresh separate-library client execution/typing/refusal checks, not only implementation-unit macro proof. Format before final exact-byte verification; verify the actual committed fixture/snippet bytes, not formatter equivalence. Record actual gate classify/check/receipt outcomes; a FULL result is not a docs-only skip. Every command must reach a terminal exit; preserve failed probes separately and never restart solely for an observation timeout.

### Preparation and holds

Read `.flow/tmp/fn1962-prep/{contracts.md,budget-gate.md,admission-scope.md}` as source-backed preparation, not current acceptance evidence. Reconcile line numbers and actual APIs after fn-194 integration. Recent recipes/evidence live under `.flow/tmp/fn1943/` and `.flow/tmp/fn1944/`; no baseline handoff after intervening root MILESTONES changes.

All222 disabled Temporal files, broad GuideSourcesTest and both real-Go sample bodies remain held before execution. No Temporal, IR/export/testdata, schema/Go/backend/generated/lifting/regeneration, no Make green claim if model-check-changed still routes to held Temporal. No measurement/hash-key optimization or Go deletion: those belong to .3 and held Phase3 .4. Root owns MILESTONES; documentation/oracle direction is .5. Review remains phase-batch deferred, plan-sync false, tracker inactive, spec OPEN, no publishing or history rewriting. Independent helpers may own disjoint implementation/scratch surfaces, but one owner coordinates source freeze, sole compiler and all staging/commits.
## Acceptance
- [ ] replay, find, sample and refuted exist with the Result shapes and refusals of the spec
- [ ] Vacuity reported in R15's shapes
- [ ] Budget from the gate's configuration with per-suite overrides; configuration error for an unknown suite

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
