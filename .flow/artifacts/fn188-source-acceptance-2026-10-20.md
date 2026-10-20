# fn-188 source acceptance — 2026-10-20

Tasks .1–.4 are supported by the integrated source at
`5dcd37ceaea62e95ba7e33703e247094fea6d158`. This is not full-spec completion,
stress acceptance or qualification of the full candidate.

## Requirements and proof

| Task | Requirements | Evidence |
| --- | --- | --- |
| .1 | R1: strict inventory, derived imports/streams, evidence names, seeded-draw decision and stable identities | Go/Rust readers and source-derived completeness; all 195 original identities preserved in the accepted 263-pair history. Missing, renamed, reassigned, swapped and prefix-mutated identities fail. |
| .2 | R1/R2/R6: remaining site classes and seed decisions | 68 exact runtime rows (17 draw/clock patches, 22 host-clock reads, 29 other patches), source-derived and checked bidirectionally; helper scans, Go host effects and engine pins have negative checks. Seed findings retain decisions or follow-up referrals, not unsupported claims of fixes. |
| .3 | R2: strict request/reply, remote and cluster refusals and engine pins | Rust engine refusal tests, helper corpus, and Go standalone/remote/cluster boundaries. Unknown valid live instance-tagged kinds now return `unsupported` naming the kind; malformed, stale and terminal identities remain `invalid`. |
| .4 | R4: first divergence, identities, actual coordinates, values and usable replay prefix | Four real Runner fixtures retain draw, clock, handoff and output-before-clock divergences; provider and literal external CLI replay verify each preceding prefix and witness. Actual broker-generated controller events are separately accepted without admitting guest incarnation zero. |

Inventory repairs are `d5725379e0`, `f512576f88`, `dd5c726e64` and
`10d83505fd`. They close invented-retirement and retired-allowlist bypasses while
preserving legitimate accepted retirement and new active appends. Go/Rust RED
exits were 1/101 before the relevant fixes. Exact commands and raw logs remain in
`/home/agent/wt/evidence/fn188/inventory-contract-repair/`:
`REPORT.txt`, `RETIREMENT-PREFIX-REPORT.txt`,
`OFFICIAL-REVIEW-RETIREMENT-REPORT.txt` and
`ACTIVE-APPEND-RETIREMENT-REPORT.txt`.

Boundary repair `696a6ecc3e` has a real ten-case `clusterStream.frame` regression.
Its RED run failed only the two valid tagged cases; GREEN passed all ten. Source
pins in three codec/stage fixtures were regenerated; stale attribution and
malformed JSON semantics did not change. The report is
`/home/agent/wt/evidence/fn188/cluster-unknown-frame-repair/REPORT.txt`.

Controller repair `bc3d00b5d6` permits the reserved `controller/0` event identity,
not a guest inspection boundary. Its Go regression first failed on a complete
stream emitted by the actual native `Broker.command(Revoke)` path. A Rust test
subprocess compares that producer's bytes to the checked-in parser fixture.
Guest/0, misspelled/empty identities and controller inspection boundaries still
fail. Full Go determinism tests/vet, affected lint, the native producer test,
focused Rust Clippy and formatting passed. No WASM store was used for this
producer test. Exact commands, exits and hashes are in
`/home/agent/wt/evidence/fn188/controller-order-repair-e0cd113841/verification.json`.

The four actual execution/replay proofs remain at
`/home/agent/wt/evidence/fn188/fixtures/TestDeterminismExecutedDivergenceArtifacts/`
under `draw`, `clock`, `handoff` and `output-clock`. Each `prefix-receipt.json`
records `diverged`, then `inspected_prefix`, `prefix_verified=true` and
`witness_match=true` for both provider and external CLI inspection. Earliest
helper events are respectively 102, 87, 449 and 84; choice stops are 7 except
handoff's b/1 choice 23. Those executed proofs retain their original helper,
module and source identities; they are not relabeled as current qualification.

## Integrated checks and independent review

On the clean integrated source above, stock Go 1.27.1 with `GOMAXPROCS=2`,
`CARGO_BUILD_JOBS=2` and `CARGO_INCREMENTAL=0` ran:

- `make -C tools/gomad validate`: exit 0. Rust inventory 12/12, helper corpus
  26/26 and engine refusals 3/3; Go determinism 6.267 s, observed 0.004 s,
  archcheck 41.124 s, both-platform WASM architecture 3.330 s and WASI 0.029 s.
  Log: `/home/agent/wt/evidence/integration-2026-10-20/validate-fn188-controller-tagged-fixes.log`.
- `make lint-code-gomad` with retained Linux lint/errortype binaries,
  `GOLANGCI_LINT_FIX=false` and base `e0cd113841`: exit 0, zero issues.
  Log: `/home/agent/wt/evidence/integration-2026-10-20/lint-fn188-controller-tagged-fixes.log`.
- The integrated focused WASI boundary/codec/stage suite: exit 0. Both
  helper-backed execute refusal subtests passed; no skips.
  Log: `/home/agent/wt/evidence/integration-2026-10-20/tagged-frame-integrated.log`.

The official first three-draw review found the two scoped R2/R4 gaps above.
After their fixes, the actual SINGLE re-review returned SHIP at
2026-10-20T18:15:58Z, with both findings fixed and no unaddressed requirements
within tasks .1–.4. Receipt `.flow/tmp/impl-review-fn188-source-batch.json`
binds base `ea6ed346a12830bc47d07c849c374c3da0d5dadc` to this source head in
`findings.headSha`, session `01a15a74-37ad-7020-b5d4-830052a6a1a3`, model
`codex:gpt-6.1-sol:high`. Reviewer and writer are the same model family.

## Remaining acceptance

Tasks .5 and .6 remain open. The first public stress reference on historical
`f046982309` failed before any guest because retention staged the requested root;
the failure and linked source repair `05ed193659` remain recorded in the existing
integration checkpoint and external preflight logs. No completed reference or
full matrix is inferred from that failure or from a helper build. The 96-stage
candidate qualification, full stress matrix, full functional sweep, and final
contract/gates still require their own current evidence. No seed, repetition,
budget, oracle, classification or one-hour qualification limit is relaxed.
