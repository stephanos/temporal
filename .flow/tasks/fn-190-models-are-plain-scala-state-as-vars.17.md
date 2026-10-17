# fn-190-models-are-plain-scala-state-as-vars.17 Remove the Native prefix from current framework sample names

## Description
User follow-up on 2026-10-17: remove the historical Native prefix from framework/samples names. Rename NativeInputs.scala and its object/test to Inputs; NativeHistory.test.scala cases and helpers into the existing Kettle.test.scala and TwoLayeredCache.test.scala suites; NativeVoting.test.scala/class to VotingBehavior because Voting.test.scala already exists. Update live runner and documentation references. Preserve historical Flow receipts and frozen retained source bytes. This naming follow-up grants no closure credit for deferred IR obligations. Root separately owns ignored comparison layout and MILESTONES.md.

## Acceptance
No current sample filenames or suite/model names start with Native; live references follow the renamed files; renamed suites and framework-loop discovery tests pass with unchanged behavior. History.test.scala and HistoryTest are removed; all former history cases and assertions remain in their model-specific suites.

## Done summary
Removed Native from the current Inputs and VotingBehavior sample names and their live references. Split all 21 former NativeHistory cases and their helpers into KettleTest (12) and TwoLayeredCacheTest (9), preserving their test bodies modulo whitespace; removed the history suite and adjusted discovery without changing behavior.

baseline: green — pre-edit Native*Test (54 passing entries) and FrameworkLoopSuite (8 tests); no parent Quick commands section. The first baseline command repeated --test-only, which Scala CLI rejected before compilation; the corrected command passed.

Verification: final full native DSL/realize/samples suite exited 0 (644 passing entries), FrameworkLoopSuite exited 0 (8 tests, framework mode), RuleSuite lint exited 0 (31 tests; real block-form scan 155 files, 0 findings), scoped scalafmt check and git diff --check passed. Logs are .flow/tmp/fn19017/verify-split-{samples,loop,lint}.log and format-split-check.log.

An exploratory broader framework.check.* run exited 1 in GuideSourcesTest because held Temporal activity_mini/protocol/Protocol.scala is absent; its log is verify-check.log. No changes were made to that gate. Owner-authorized native scope excludes model-check-changed because it reaches held IR/export and Temporal checks. Native results grant no held IR, exporter, schema, Go or Temporal closure credit.

Frozen migration source bytes and historical Flow receipts were preserved. Only live inventory pointers changed. No .flow/features routes changed. The user concurrently committed all task source changes in 4996481ded292b4754b5430fc35e9a6ce355724c before worker staging, including unrelated user/root work in MILESTONES.md and ir-breakdown.md; the worker did not edit or stage those unrelated files. No source commit was rewritten.

Tier: session (jev-unavailable(no_key))
stage: impl-review - skipped(config: REVIEW_MODE=none; independent review deferred to owner batch)

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 4996481ded292b4754b5430fc35e9a6ce355724c
- Tests: mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests, UMPIRE_FRAMEWORK_BUILD=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn1893-verify mise exec -- scala-cli test --server=false model/framework/check --require-tests --test-only framework.check.FrameworkLoopSuite, mise exec -- scala-cli test --server=false model/framework/check --require-tests --test-only framework.check.lint.*RuleSuite, mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check model/framework/samples/Inputs.scala model/framework/samples/Inputs.test.scala model/framework/samples/Kettle.test.scala model/framework/samples/TwoLayeredCache.test.scala model/framework/samples/VotingBehavior.test.scala model/framework/check/FrameworkLoop.test.scala, git diff --check
- PRs: