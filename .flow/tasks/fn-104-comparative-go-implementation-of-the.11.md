# fn-104-comparative-go-implementation-of-the.11 T10 Error corpus on both sides

## Description
TBD

## Acceptance
- [ ] TBD

## Done summary
Twelve-case corpus (cases.json with lean, go and scala edits) and run.py harness; all three sides run; Go gap (duplicate Property names) found and fixed in umpire.Check.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: python3 model/go/corpus/run.py go, python3 model/go/corpus/run.py lean, python3 model/go/corpus/run.py scala
- PRs: