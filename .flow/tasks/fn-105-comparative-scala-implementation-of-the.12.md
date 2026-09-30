# fn-105-comparative-scala-implementation-of-the.12 S11 Error corpus scala edits

## Description
TBD

## Acceptance
- [ ] TBD

## Done summary
Scala edits for all twelve corpus cases and a scala side in model/go/corpus/run.py (compile, test, prove stages); results in results-scala.json and results-scala-prove.json. Found and fixed scala-cli exiting 0 on -Werror failures through Bloop (model/scala/scala.sh).

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: python3 model/go/corpus/run.py scala
- PRs: