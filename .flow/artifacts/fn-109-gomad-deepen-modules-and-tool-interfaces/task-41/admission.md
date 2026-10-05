# Canonical JSON exhaustive correction admission

Task41 advances the remaining R18/R19 lint gate without changing canonical JSON behavior. Root's actual BASE package tests pass and pinned unfiltered package lint reports one exhaustive finding. The correction owns the switch header and seven selectors only, with additive literal characterization before production edits.

## BASE evidence

- Source revision `a683e64af560322014e14f3a1ef3953b27cad96a`; native execution host `linux/arm64`; stock Go 1.27.1. This is developmental source evidence, not qualified runtime execution.
- Collection ran from 2026-10-05T12:50:27Z through 12:50:28Z. The [actual command](baseline-command.sh) pins offline resolution and `GOMAXPROCS=2`, clears seed controls and retains raw output. It does not capture inherited environment values.
- [Whole-package JSON output](baseline-tests.jsonl) records six top-level passes, two nested passes, no failures and no skips; process exit 0. Existing test source remains unchanged.
- [Unfiltered package lint](baseline-lint.log) exits 1 with exactly one exhaustive finding at `canonical.go:120`. No exclusions or enabled-analyzer changes were applied.
- Before/after checks cover 995 entries selected by the command's exact tracked-path inventory plus four executable pins. All entries match; stability exit 0. The complete local manifest `/tmp/fn109-canonical-base.aMBIU3D3/source.sha256` has SHA-256 `35139285c61f78e99e826e833c0e1703939571c450ae129ac7887d99a00ce575`. This selector is not a whole-repository snapshot.

| Input | SHA-256 |
| --- | --- |
| canonical.go | 879038120b0fbdc7e88a2de2a4f8644718ab521c8b512afdf68899ce949e3c31 |
| canonical_test.go | 840517c357c1c0f24c81d8061764c814402cc443c9ff29ad727fa5f56cd41846 |
| Nested go.mod | 5fbae20edbc0cde0c1d6b8a0e974ed7791a87404d5ec91b0ea13eb5f97ecce9f |
| Nested go.sum | d438e0b5972c523df1a73e37a6b3d6a7523019e1ec18c2fd6cbb503d1c196960 |
| Lint config | 2abff492b6a1aeaaded801bccc2d8ab85ed366608862b0b9a5dd5311ae0fed43 |
| Stock Go executable | 1675694ef690db0f18fbe7046a886170904bede1d9db6ec96ae27945c1705c64 |
| golangci-lint v2.13.0 | acacd04faf1d17489a890a4589ae36a62484c1076cf5fbcb7a33274cbc6a6bdc |
| errortype | db481b4086fb85962e98ae625984f2560a883dce91f8b7d8c705414c207be8cc |

## Preservation and evidence boundaries

Repository research found the visited-slice key omits length. The short-then-long alias can skip validation of a longer invalid UTF-8 suffix. Task41 must freeze actual BASE bytes, reverse-order rejection and differently named aliases, and retain this behavior without claiming universal UTF-8 rejection. A semantic repair is outside this mechanical correction.

Canonical JSON serializes generated pack requests, packs and generation state. Check-only generator validation remains required even though canonical.go is absent from explicit protocol/toolchain identity inputs. No regeneration or pin refresh is admitted. The real serializer analyzer fixture, record/World/pack consumers and target canonical/digest/projection controls retain their own measured outcomes.

The previously proposed Runner-fixture scope was rejected by actual BASE execution. Its eight controls produced one top-level pass and seven failures; the preparation validation guard rejects linux/arm64 before the fake executor can run. The unfiltered Runner lint scope retains 17 findings, including its two fixture exhaustive findings. No Runner source edit or corrective owner was admitted. Raw local output is `/tmp/fn109-runner-fixture.sMWwoRVI/baseline-controls.jsonl` (SHA-256 `81a9844272c5e50fdc04f98e4601a829ac19f951ccce57baec9d44be5bee206b`) and `baseline-lint.log` (SHA-256 `eee0855435b069a8b33f390280739f939e147f015da598c1d5e4fa1edc1c8765`). These failures change the next action; they establish no fixture behavior or native qualification.

Parallel read-only repo/spec/gap scouts identified no competing canonical source owner. Gap findings require explicit marshaler callback counts and deterministic alias traversal; the task incorporates both. Memory search and its keyword-refining fallback found no relevant entries (`bm25`, `jev-unavailable(no_key)`). Task21 gains a direct dependency on task41 and consumes its evidence; no old dependency or acceptance is removed.

Original first-baseline, preservation, predecessor, full/default/functional/affected-consumer/formal and native Darwin requirements remain required wherever unproved. Native Linux execution remains with fn128. The last actual integrated lint result has 325 residual findings and an unreached full errortype stage; scoped lint success cannot infer a new aggregate count or close acceptance. Root will retain a new actual frozen-candidate integrated run and independent source-progress review before committing implementation progress.
