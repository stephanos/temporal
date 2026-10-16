# fn-182 smaller WASM guests: handover (2026-10-15)

Branch `fn182` (worktree `/home/agent/wt/fn182`), based on `gomad` `01c419c40a` (pre-fn-178,
provisional under batch-first). No Flow task is marked done. Nothing is merged and
MILESTONES.md is not edited.

| Commit | Task | Content |
| --- | --- | --- |
| `9335ea7c32` | .1 | Closure/candidate analysis `.flow/artifacts/fn182-guest-dependency-closure-2026-10-15.md` plus scripts and the removed-package list |
| `52a9413803` | .2 | `gomad_wasm_slim` tag on the existing archiver/membership seams, new `tests/testcore/flag_sql_wasm_slim.go`, refusal tests, `backend.SlimGuestBuildTag`/`TemporalGuestBuildTags()`, adoption in the one-box and multi-instance cells, backend integration tests |
| (this commit) | .3 partial | CLI/README/QUALIFICATION docs, `.flow/artifacts/fn182-guest-qualification-2026-10-15.md` with drivers, this handover |

## Per task / R-ID

- **R1 (.1) done provisionally.**
  - *Closures and seams.* Closures were recorded for one-box `./tests` (1466 packages) and the
    multi-instance fixture (1446). Three seams were selected: cloud archivers, ringpop and
    MySQL/PostgreSQL test drivers. Together they remove 126 packages from both guests, and the
    predicted removed set equals the actual one.
  - *Rejected candidates.* Elasticsearch is rejected because the scheduler's legacy query converter
    needs `olivere/elastic`. ES AWS signing would remove 0 packages. WCI is an always-enabled
    runtime component. Cassandra gains little.
  - *Init review.* The `init`/`Register*` calls in removed packages were reviewed.
- **R2 (.2) implemented.**
  - *Size.* Disposable stock builds shrink one-box from 337,877,939 to 306,274,379 B (−9.35%) and
    multi-instance from 240,188,710 to 208,975,675 B (−13.0%).
  - *Default builds unchanged.* Default/native file selection is logically unchanged, and vet passes
    under the tag sets {none, gomad, gomad_wasm_slim, both}.
  - *Refusals.* Tests cover gs/s3 → `errCloudArchiverUnavailable`; ringpop → `fx.Error`; mysql8/postgres12(_pgx)
    → SQL visibility retained plus `ErrPluginNotSupported` with `supported plugins: [sqlite]`. Commands:
    `go test -tags gomad_wasm_slim,test_dep -run '<3 tests>' ./common/archiver/provider ./temporal ./tests/testcore` (exit 0).
- **R3 (.2) implemented.** `TestBuildSelectionChangesInvalidateCompilationCache` covers four
  properties:
  - *Changes key separate entries.* A tag closure change, a tag-only change and a go.mod language
    version change each get a different build key.
  - *Provenance.* Provenance names the actual sources.
  - *Relabeling refused.* A relabeled BuildTags target is rejected.
  - *Reuse.* An unchanged selection reuses its entry.

  `TestSlimGuestClosureCutsOnlyOptionalProviderSeams` checks three more properties:
  - *Exact closure.* The actual slim closure equals the default closure minus the 5 seam edges.
  - *Retained packages.* Required packages remain.
  - *Module inputs.* Removed modules' go.mod files leave the provenance inputs.

  A mutation run (also trimming WCI) fails it as expected. Both tests passed (`GOMAD3_STOCK_GO=<go1.27.1> go test -tags test_dep -run ... ./backend`).
  The compiled-cache identity already binds the module SHA (`wasmhost/src/cache.rs` `identity`).
- **R4 (.1/.3) OPEN.** No timed baseline or candidate measurement has run. The prepared driver is
  `.flow/artifacts/fn182-guest-qualification-2026-10-15/measure-r4.sh` with `summarize.py`, which
  was dry-run checked on candidate data. Module and compiled-cache bytes are deterministic. The
  provisional candidate one-box cooperative module is 306,370,445 B with a 628,984,402 B compiled
  cache entry.
- **R5 (.3) partial.** All four candidate cells exited 0 (provisional, loaded host):
  - *Success cells.* One-box and multi-instance success are qualified and deterministic.
  - *Canaries.* Both canaries are deterministic expected failures with 1 replayable bug each.
  - *Replays and budgets.* Fresh replays passed, with 0 budget violations.

## Open items / exact next steps

1. After fn-178 lands, rebase `fn182` onto the joined candidate. Move `/home/agent/wt/fn182-base`
   to that commit (`git -C /home/agent/wt/fn182-base checkout --detach <joined>`) and rebuild both
   helpers (`CARGO_BUILD_JOBS=4 make -C tools/gomad wasm-helper`).
2. Re-run `TestSlimGuestClosureCutsOnlyOptionalProviderSeams`, `TestBuildSelectionChangesInvalidateCompilationCache` and the three refusal tests on the joined candidate.
3. Then run `make -C tools/gomad test-wasm GOMAD3_STOCK_GO=...` and lint (`make lint-code-fast`, `make lint-code-gomad-wasm`), which were not run on this branch.
4. Run the functional cells on both sides:
   `.flow/artifacts/fn182-guest-qualification-2026-10-15/run-cells.sh <wt> <root>` for the candidate
   and for the baseline (same-source A/B). On the joined candidate, use fresh fn-178 cluster inputs
   (`TestTemporalIndependentClustersPartitionHealWholeClusterCrash` producer) instead of the fn-176
   ones.
5. QUIET HOST only (GO MEASURE): `measure-r4.sh /home/agent/wt/fn182-base /home/agent/wt/fn182 /home/agent/wt/fn182-tmp/measure/r4-<date> 5`.
   Record `summary.md` in the .3 evidence and report within-spread results honestly.
6. Get an independent impl review: `flowctl claude impl-review ... --spec claude:claude-opus-5-5:high`. It has **not been run yet**.
7. Then complete .1/.2/.3 only on the conductor's instruction. `flowctl done` overwrites Evidence,
   so re-append the details afterwards.

Local scratch (not committed): `/home/agent/wt/fn182-tmp` (disposable modules, go list JSON,
qualification evidence `qual/candidate-1`).
