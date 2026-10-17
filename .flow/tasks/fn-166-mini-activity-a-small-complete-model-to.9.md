---
satisfies: [R6, R7]
---
# fn-166-mini-activity-a-small-complete-model-to.9 Mini-guided author documentation and migration convention

## Description
Completes R7 and supplies R6's rule source and isolated mini proof. Integration into the executable Make/Scalafix route remains in held task 10.

**Size:** M
**Files:** shared custom-rule source and focused fixtures under `model/migrate/`, `model/README.md`, `model/framework/SEMANTICS.md`, convention sentence in open Scala DSL batch specs.
**Touches:** [model/migrate/**, model/README.md, model/framework/SEMANTICS.md, .flow/specs/*.md]

### Approach
- Replace the hypothetical job in `model/framework/SEMANTICS.md:13` with the complete committed mini feature from task 8. Cite actual files and source declarations for each example; do not invent a declaration for a guide snippet.
- Update README Writing a Model with the complete three-level mini feature and its named hard instances.
- Establish one shared custom-rule source location/naming under `model/migrate/` and author the worked rename as an actual Scalafix rule with isolated compile/transformation fixtures. Reuse the repository's existing Scalafix dependency; inspect its API before coding. A direct in-process source test is source proof, not proof of the production loader. Do not add a competing general-purpose rewrite engine or repeat fn-167.4's gap transformation.
- Select a concrete rename that preserves semantic declaration identities and source positions so the specified byte-identical lift is demonstrable. A rule that renames an exported definition cannot silently substitute an expected identity change for that proof.
- Record the mini-first migration policy and the actual source-test route. `Makefile:689` currently loads built-ins only; its custom-rule loading/configuration integration belongs to task 10.
- Carry the required convention sentence into each open DSL spec through Flow setters. Preserve unrelated owner edits and completed histories. No Make/config/production-loader change or real-feature edit belongs here.

### Verification
Check cited paths/declarations, run the mini-focused compile covering them, and test the actual rule source against isolated copied-mini input with exact lift comparison and rename/refusal controls. Report commands requiring held infrastructure rather than claim they passed. R6's executable-route integration remains incomplete until task 10.

## Acceptance
- [ ] SEMANTICS uses the committed mini feature as its running example and every snippet cites actual source; README names its complete hard-instance inventory.
- [ ] One shared migration source location/naming and mini-first convention is documented consistently in README and every open Scala DSL batch spec.
- [ ] The worked example is actual compiled rule source with isolated copied-mini transformation and the required byte-identical lift proof; no alternative migration engine is introduced.
- [ ] Docs state current versus held behavior accurately; no production-loader, real-feature migration, full-set or review credit is claimed.


## Done summary
Model authors now read the complete committed mini feature throughout SEMANTICS, with 30 source-checked Scala excerpts; README names its three levels and all hard instances. A compiled task-named Scalafix rule and isolated copied-mini proof establish the migration convention, repeated through Flow setters in the 15 open Scala DSL batch specs.

baseline: green. The fresh selected mini/shared compile exited 0 in baseline.log before edits. The persisted base is 00d22597d29c74c03dea13511162035f6e55d8e0. The owner override replaced the held Make/full-suite Quick commands with focused source verification for this step.

Focused evidence
- focused-final.log records exit 0 and four executed tests. QueueAliasTest transforms both committed mini queue files exactly, preserves length, converges on a second application, refuses target collisions, shadow bindings, bare uses and conflicting imports without a partial rewrite, and retains unrelated scopes, comments and literals. GuideSourcesTest checks every Scala excerpt against its cited file and original source line.
- proof-fifth.log records exit 0 for the actual compiled Scalafix rule on copied complete-mini sources. The proof directory is .flow/tmp/mini-migration.tWPfvN/. Both packages compile and both complete mini lifts succeed; raw cmp passes for activity-mini.json and activity-mini.waivers.json. The before and after IR SHA-256 is b38f5fe304ee9d6c0de6aad646e39a68c36cc5ecad3b071cf169e57c801d92d9. No identity map, source-position stripping, normalization or expected-change projection is used.
- Fn166Task9QueueAlias renames only a local import alias, foundationQueue to taskQueueSource, and qualified uses. Both have fifteen characters and resolve to the same foundation queue declaration. The proof's corresponding frozen jars replace mutable classes on the model classpaths; it copies only mini and shared authoring sources.
- The shared rule source location is model/migrate/rules/ and naming is Fn<spec>Task<task><Change>.scala. README and all 15 open batch specs carry the same mini-first convention sentence. set-plan changed each spec's own updated_at only; the commit stages only the new convention Markdown hunks and those setter timestamps. Removing the inserted sentence restores every preexisting spec Markdown byte. The checked index patch is .flow/tmp/fn166/task9/conventions.patch.
- git diff --check and git diff --cached --check passed. No source outside the task Touches was edited.

Red and setup observations
- rule-red.log executed the three rule tests and failed for the intended no-op rule, before the real patch implementation. rule-green.log then passed all three.
- docs-first.log executed the citation guard and failed on an incorrect framework/Machine.scala line 13 citation; correcting it to line 10 restored the guard. focused-final.log is the final four-test green observation.
- proof-first.log, proof-second.log and proof-fourth.log are inconclusive setup attempts. This host's child shell could not resolve the relative script path. proof-third.log used the absolute script path, compiled/lifted the before Model, then failed to find RewriteMini because ScalaCLI classifies test/ as test-only. Moving the source-proof helper to proof/ and invoking /bin/bash with the absolute path produced the fifth successful proof. These failed attempts receive no proof credit.

Authored protocolCompleted.live(satisfied) and retryCompleted.live(inconclusive(Reason.explanationsDisagree, because = ...)) remain expectations rather than observed live Verdicts. Only those two Queries belong to MiniActivityRun. The existing native limitation for an imported bare action remains visible in SEMANTICS with Rules.test.scala:707; fn-159 owns its later runtime contract.

R6 production loading/configuration and the executable Make route remain held in fn-166.10. Mini Case lowering/live proof remain held in .4, canonical loop/full-set closure in .5, Case subtree in .6 and target rename in .7. Make linked-protobuf prerequisites, Go/proto/runtime, source gap transformation, real-feature migration/artifacts, production custom-rule loading and checked-in generated outputs were not changed. The root-captured native mini baseline and task8 frozen jar were not recaptured or edited. Lint checks and manual acceptance were NOT RUN under the owner override.

stage: impl-review - skipped(policy: owner-approved Scala batch-end; REVIEW_MODE=none)
Broad validation and independent review remain deferred to the Scala batch-end gate. gate classify returned FULL on unrelated IDEAS.md; no docs-only skip, full-suite receipt or SHIP is claimed.
stage: plan-sync - skipped(config: planSync.enabled=false)
Tracker sync: n/a (bridge inactive)

Tier: session (jev-unavailable(no_key)); project implementer routing retained

External owner scopes in all 17 preexisting Spec Markdown files, IDEAS.md and MILESTONES.md, plus untracked fn168 MD/JSON, remain outside this task's commits.
## Evidence
- Commits: 50043145a07a24e6f5f43e67811bd9ea22fe3dfa
- Tests: timeout 600 mise exec -- scala-cli compile --server=false --suppress-outdated-dependency-warning model/project.scala model/framework model/temporal/Bounds.scala model/temporal/foundations model/temporal/actors model/temporal/capabilities model/temporal/realize model/temporal/features/activity/Activity.scala model/temporal/features/activity_mini, timeout 600 mise exec -- scala-cli test --server=false --suppress-outdated-dependency-warning model/migrate --test-only 'umpire.migrate.*Test', timeout 600 /bin/bash /Users/stephan/Workspace/skunkworks/umpire/temporal/model/migrate/prove-mini.sh, git diff --check, git diff --cached --check, RED: no-op Scalafix rule (rule-red.log); incorrect framework citation (docs-first.log), INCONCLUSIVE SETUP: relative script paths (proof-first.log, proof-second.log, proof-fourth.log); test-only RewriteMini main excluded (proof-third.log), corrected in proof-fifth.log, NOT RUN: lint/manual acceptance; broad suite and independent review deferred by owner to Scala batch end; production Make/custom-rule loader, Case/live, Go/proto and full-set gates held
- PRs: