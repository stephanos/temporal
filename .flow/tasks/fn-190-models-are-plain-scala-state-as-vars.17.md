# fn-190-models-are-plain-scala-state-as-vars.17 Remove the Native prefix from current framework sample names

## Description
User follow-up on 2026-10-17: remove the historical Native prefix from framework/samples names. Rename NativeInputs.scala and its object/test to Inputs; NativeHistory.test.scala cases and helpers into the existing Kettle.test.scala and TwoLayeredCache.test.scala suites; NativeVoting.test.scala/class to VotingBehavior because Voting.test.scala already exists. Update live runner and documentation references. Preserve historical Flow receipts and frozen retained source bytes. This naming follow-up grants no closure credit for deferred IR obligations. Root separately owns ignored comparison layout and MILESTONES.md.

## Acceptance
No current sample filenames or suite/model names start with Native; live references follow the renamed files; renamed suites and framework-loop discovery tests pass with unchanged behavior. History.test.scala and HistoryTest are removed; all former history cases and assertions remain in their model-specific suites.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
