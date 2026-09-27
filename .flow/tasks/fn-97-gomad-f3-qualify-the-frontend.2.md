---
satisfies: [R3, R4]
---
# fn-97-gomad-f3-qualify-the-frontend.2 Update temporal.json and the CI assertion for the probe

## Description
Set the probe's capability mode and darwin expectation `qualified`; keep linux expectation honest; update the CI Temporal assertion counts.

## Acceptance
- `make gomad3-qualification` reports the probe supported on darwin
- CI assertion matches

## Done summary
temporal.json now expects `frontend-system-info` to be `qualified` on darwin/arm64. Its capability mode stays `guarded`, the mode that qualifies it, and linux/amd64 stays `unrepeatable` because this host cannot measure linux. `make gomad3-qualification` on darwin with the committed manifest met every expectation: 6 supported, 11 unsupported, 1 failed, 0 infrastructure errors, 18/18 completed. The probe qualified on both seeds with replay_match and choice_replay_exact. The darwin CI assertion in .github/workflows/gomad3.yml now requires supported/failed of 7/0 or 6/1 and requires the probe to be `qualified`; it evaluates true against this report. The tools/gomad3integration README states the new darwin expectation.

Probe numbers, from the set report and its two per-seed qualification reports:
- Seed 11: 249 transcript records, 31872 transcript bytes. 3350 choice decisions in 4627 records (444192 tape bytes of the 8 MiB limit). Evidence sha256:3c129ec0463ddb1f6310870dea995a23035edb0db9803096382ad645ec736bfa, trace_bytes 888384.
- Seed 17: 253 transcript records, 32384 transcript bytes. 3301 choice decisions in 4572 records (438912 tape bytes). Evidence sha256:04d22b1ad82f5f1cc9d451053981c9a8152e8f209e4d11d375a747e7eca9efc7, trace_bytes 877824.

Transcript bytes are records × 128, the same accounting task .1 used. Evidence digests differ from task .1's standalone `gomad qualify` run because the qualify-set run has a different configuration; the choice counts match task .1 exactly.

`user-timers-workflow` was `nondeterministic` on both seeds (replay_match false), so its darwin expectation stays `intermittent`. It was not tightened.

Copies of the report are at scratchpad/temporal-qualification-set-run1.json and scratchpad/f3t2/. The retained artifacts in tools/gomad3/.toolchain/temporal-qualification were deleted afterward.

stage: impl-review - ran (codex three-draw fan-out, all SHIP, finalized SHIP)
## Evidence
- Commits: 403b1d36ecd69f4fc806676332e9e7fda91bdd11
- Tests: baseline: none (spec lists no Quick commands), make gomad3 (toolchain key b59e8d7a5b0e9693b8c7ef97c7977979972983093a2c44e2cd94280b6b25498c), make gomad3-qualification (darwin/arm64, with the committed temporal.json): expectations-met=true supported=6 unsupported=11 failed=1 infrastructure-errors=0 completed=18/18, updated darwin CI jq predicate evaluated against that report: true, make gomad3-integration-test: ok
- PRs: