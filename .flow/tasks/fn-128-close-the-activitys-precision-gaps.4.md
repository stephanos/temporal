---
satisfies: [R4]
---
# fn-128-close-the-activitys-precision-gaps.4 Stutter facts checked: visible on ActivitySystem's refinement

## Description
**Batch:** ACTIVE approved Activity Model batch (MILESTONES.md, fn-128 → fn-138 → fn-129). Do not run `make umpire-gen-model`, regenerate production fixtures or Cases, or run the full gates in this task; production IR/answer/Case comparison is checked at the shared batch's single regeneration against ee32b5fa6023c4ab8dfcc928f07d79afe5587186, not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own. Fresh independent review and the single live run remain mandatory at the shared fn-128.6/fn-129.5 boundary.

R4 (comparison P2-8). `ActivitySystem`'s `refinement` declares exhaustive public-status-fact `visible`, including each typed `statusTimedOut`; exclude only `attemptCount`, which the Product omits. Decide each refinement rejection this surfaces (Product row, System fact, or a recorded exception) and record the decision in the done summary. Preserve the System mapping/facts and strict checker. The Product carries creation, held pause and pause withdrawal observations while retaining its existing abstract acceptance/rejection alternatives. Runs after tasks 1-3 because they change the facts.

### Touches
- `model/temporal/features/activity/standalone/{product/Product.scala,system/System.scala}`
- Focused Activity/refinement tests in `model/temporal/features/activity/standalone/` and only directly required tests in `tools/umpire/check/`.
- `MILESTONES.md`: own fn-128.4 row and As-of date only.
- Ignored current-source scratch lift/admission/refinement proof and task evidence under `.flow/tmp/activity-batch/`; no generic engine/DSL edits or production generated artifacts.
## Acceptance
- [ ] `visible` is declared and the refinement check passes.
- [ ] Every rejection it surfaced is listed with its decision.
- [ ] Changes listed for R6; the spec's Verification gates pass.

## Done summary
# fn128.4 integrated source verification

ActivitySystem now makes every public status fact visible, including typed
statusTimedOut values, and excludes only attemptCount because Product omits it.
ActivityProduct carries creation, held pause and pending-pause withdrawal while
preserving its existing acceptance/rejection alternatives. System mapping,
facts and transition rows remain unchanged. No checker or DSL weakening.

Original isolated source handoff and complete evidence/provenance are retained in
.flow/tmp/activity-batch/fn1284-worker-summary.md and fn1284-worker-evidence.json.
Original pre-edit base d5e3cc45c6a5a42c619ffd9da8f94df1ba5a6816; isolated/integrated
base802167f21295260ac59d94d8e04cfb53b42e3aed; implementation21206f853a5f010d07a82c6c6b68118da2933490
is reachable on target branch umpire through an original-commit fast-forward.

Exactly three public stutter families surfaced:48creation classes/10368rows,
held pause216rows and withdrawal216rows. All are carried; no status was hidden.
Remaining3096stutters are factless stop2376/startDelay360/backoff360.
Product states/classes/rows/results changed9/11/49/51→9/59/97/101. System and
TimeoutRetry remain2376/63/25488/25488 with917reachable and0unknown. Carried
refinement rows11592→22392; stutters13896→3096. Existing15authored Query totals
remain unchanged; Product's three free capability totals become1593 each.
Production answers, receipts, fingerprints and Case deltas await the shared batch.

Scoped checks: Activity22, framework45, refinements3Activity+2Nexus, package/full
scratch lift, DefaultEndsSuite5, focused Go visibility/Boolean checks and scoped
format passed. Conductor confirmed all five transferred source SHA256 values
against the tested originals, artifact copies with cmp, and unchanged executable
inputs. Those scope-valid tests are reused under MILESTONES verification rules.
Conductor additionally reran the actual-current-source refinement/mutant proof
on integrated target21206f853a:exit0, log fn1284-integrated-proof.log. All five
mutants refuse and reachable witnesses replay: three missing carriers and an
injected visible dispatch fact→visible-stutter; exposing count→unmatched.

Checkout collision handling preserved external fn149/fn150/MILESTONES content.
Only the five verified duplicate task-source files were saved in recoverable
stash ced97742aae6aafd1298e1cb4af9ca8226095a61 before fast-forward. Keep that stash
and isolated worktree until later cleanup is explicitly appropriate. No history
rewrite, manual conflict resolution or external edit deletion occurred.

Tier: project implementer gpt-6.1-sol/high; judge unavailable/no_key. No actual
model provenance is asserted without host execution metadata. One isolated
source worker; conductor owns integration and lifecycle.
stage: impl-review - deferred(approved shared activity batch)

ACTIVE source batch128.1–.5→138.1–.3→129.1–.4 retains ONE production regeneration,
full Model/Go/lint/fixture/canary gates, fresh independent review and live run at
shared128.6/129.5. These are NOT claimed passed here; neither parent spec nor goal
closes. Existing cancellation-failure disagreement remains unaltered.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 21206f853a5f010d07a82c6c6b68118da2933490
- Tests: /usr/bin/flock /tmp/umpire-heavy-gates.lock timeout 600s mise exec -- scala-cli test --server=false --suppress-outdated-dependency-warning model/project.scala model/umpire model/temporal --test-only '*Activity*' --require-tests, /usr/bin/flock /tmp/umpire-heavy-gates.lock timeout 600s mise exec -- scala-cli test --server=false --suppress-outdated-dependency-warning model/project.scala model/umpire --require-tests, /usr/bin/flock /tmp/umpire-heavy-gates.lock timeout 600s mise exec -- scala-cli test --server=false --suppress-outdated-dependency-warning model/project.scala model/umpire model/temporal --test-only '*RoleRefinements' --require-tests, /usr/bin/flock /tmp/umpire-heavy-gates.lock timeout 600s mise exec -- scala-cli --power package --server=false --suppress-outdated-dependency-warning --library model/project.scala model/umpire model/temporal -f -o model/build/model-scala.jar, /usr/bin/flock /tmp/umpire-heavy-gates.lock timeout 600s mise exec -- scala-cli run --server=false --suppress-outdated-dependency-warning model/irgen --main-class umpire.irgen.lift -- --ir model/build/model-scala.jar=model/ model/build/model-scala.classpath .flow/tmp/activity-batch/fn1284-ir activity-standalone, /usr/bin/flock /tmp/umpire-heavy-gates.lock timeout 600s mise exec -- scala-cli test --server=false --suppress-outdated-dependency-warning model/irgen --test-only umpire.irgen.DefaultEndsSuite --require-tests, /usr/bin/flock /tmp/umpire-heavy-gates.lock timeout 600s mise exec -- go test -tags test_dep ./tools/umpire/check -run 'TestVisibleProjectionOfARefinement|TestEndsAndVisibleMustReturnABoolean' -count=1 -v, /usr/bin/flock /tmp/umpire-heavy-gates.lock timeout 600s mise exec -- go run -tags test_dep .flow/tmp/activity-batch/fn1284-proof.go, /usr/bin/flock /tmp/umpire-heavy-gates.lock timeout 600s mise exec -- scala-cli fmt --check --scalafmt-conf model/.scalafmt.conf model/temporal/features/activity/standalone/system/System.scala model/temporal/features/activity/standalone/product/Product.scala model/temporal/features/activity/standalone/StandaloneActivityPins.test.scala model/temporal/features/activity/standalone/ActivityRejectionRegression.test.scala model/temporal/features/activity/standalone/ActivityVisibilityRegression.test.scala, git diff --check, python3 /home/agent/.codex/scripts/flowctl.py gate classify --base 802167f21295260ac59d94d8e04cfb53b42e3aed (FULL, expected exit 1; full gates deferred to approved shared batch boundary), DEFERRED: one production regeneration, exact production IR/answer/Case comparison, full Model/Go/lint/fixture/canary gates, free Query execution, fresh independent review and live run at fn128.6/fn129.5; not claimed passed, Integrated target: /usr/bin/flock /tmp/umpire-heavy-gates.lock timeout 600s mise exec -- go run -tags test_dep .flow/tmp/activity-batch/fn1284-proof.go (exit 0; fn1284-integrated-proof.log)
- PRs: