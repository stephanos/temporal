---
satisfies: [R1, R2]
---
# fn-241-one-workflowservice-classification-for.1 Split classify into one service table and per-feature claims

## Description
After fn-228.4 (same Temporal realization files), before fn-228.5. Move the WorkflowService resource-family classification out of model/temporal/feature/activity/api.rs into one service table in model/temporal outside feature/, owning exhaustiveness; the activity feature declares only its eight claimed RPCs; update the catalog syntax/build parsing (system-agnostic) and the services.rs controls; add duplicate-claim and unclassified-RPC negative controls. Goldens byte-identical except listed reason text.

## Acceptance
- [ ] Service table outside feature/ classifies every RPC and owns exhaustiveness
- [ ] Activity feature declares only its eight RPCs; reasons no longer name the activity Model
- [ ] Duplicate claim and unclassified RPC fail the build with named diagnostics
- [ ] Default and required model gates pass; guest refreshed and slice/gaps proof if guest bytes change

## Done summary
Split WorkflowService classification (R1, R2). `model/temporal/foundation/service.rs` holds one exhaustive `family` table over all 123 RPCs in descriptor order (11 resource families) and owns exhaustiveness: a new upstream RPC fails descriptor admission and the compiled match. The activity feature's `classify` claims only its eight RPCs and defers the rest to `family`. The generic catalog syntax and build (still parameterized by service and method enum) read tables and claim lists with named diagnostics: duplicate feature claim, unclassified claimed RPC, duplicate table, claims without a table, complete classification duplicating the table, claim arms that are not Action or Observed, and an annotated RPC resolving out of scope (so a dropped Observed claim or a newly annotated unclaimed RPC fails the build). Kettle's single exhaustive classify is unchanged.

Generated dispatch, binding order, roles and the bind macro are byte-identical with reason strings masked (services-before.rs / services-after.rs). Only out-of-scope reason text changed, now family names; one realization test assertion updated; the guest was rebuilt for the reason strings only (slice and gaps proofs passed). No behavioral or semantic golden changed.

Review: fresh-context round (Opus, same family) SHIP with three minors and five nits; minors 1-2 and all nits fixed in 427231772b; minor 3 (reason wording) accepted as within spec. Known limit recorded for a future second feature: all features' Action claims merge into one bind macro, so two features' action types would collide there (untested; the spec defers any sharing rule).

Gates: worktree default and required `make model-check`, slice and gaps all rc 0; post-fix default rc 0. Integrated by merge c3c95eb00b. `UMPIRE_BACKENDS=require make model-time` rc 0 at c3c95eb00b: gate 2054.281 s against 60 s, inner loop 22.546 s against 10 s, at load average 37-41. Ran as parallel lane C in `.worktrees/fn2411-classify`.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 481017ad95, 427231772b, c3c95eb00b
- Tests: UMPIRE_BACKENDS=require make model-time, make model-check, make umpire-check-model-slice, make umpire-check-model-gaps
- PRs: