# fn-104-comparative-go-implementation-of-the.8 T7 Port the standalone activity Model with pins

## Description
TBD

## Acceptance
- [ ] TBD

## Done summary
standaloneactivity/ ports .plans/cmp/lean/StandaloneActivity.lean in its order and with its comments: product (9 states) and protocol (288 states) machines, the refinement, activityWorker, the standaloneActivity composition, 10 Properties, 8 Scenarios, 10 Queries plus the cross-entity verify, and the three sets. Every translated ActivityPins.lean pin passes, including "stoppedWorkerStartsNothing is exercised" (Answer.Exercised added to the search).

Lean parity is possible only for the product machine, which matches exactly (table and IDs). Elaborating the Lean sample (lean/ActivityDump.lean, a copy) found three blockers, two of them not in the sample's README:
- no activity RPC message is reachable from the schema roots in model/Temporal/Case/Schema.lean, so every action with a `schema:` line is rejected;
- ProtocolState has 288 members and Umpire.Command.elaborationBound is a hard 256, so Lean refuses the protocol machine and everything that names it;
- Lean rejects pausedIsNotDispatched as "claims nothing": its lowering cannot express "not started". Go verifies it.

Also fixed: go-check-sumtype only checks interfaces carrying a `//sumtype:decl` doc-comment line, which the models lacked, so earlier "sumtype clean" results were vacuous. Markers added; run.sh runs both exhaustiveness linters with -default-signifies-exhaustive=false, and a probe confirms a dropped variant or enum case is reported even behind a default arm.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: experiments/umpire-go/run.sh (9s warm, all checks passed), experiments/umpire-go/lean/activity-dump.sh
- PRs: