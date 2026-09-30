# fn-104-comparative-go-implementation-of-the.1 T0 Baseline the Lean side (measure.sh, results/lean-<commit>.json)

## Description
TBD

## Acceptance
- [ ] TBD

## Done summary
measure.sh for lean, go and scala; results in model/go/results/*-d08d20140.json. Lean cold rebuild after the relayout: 20 min; edit loop 191 s (Model) and about 590 s (pins, from the corpus).

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: model/go/measure.sh go, model/go/measure.sh lean, model/go/measure.sh scala
- PRs: