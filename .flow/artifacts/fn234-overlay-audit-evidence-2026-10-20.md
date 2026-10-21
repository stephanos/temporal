# fn-234 overlay audit evidence

This index makes the retained execution evidence available to both task reviews.
It is not a review verdict or final Rust qualification. The open spec owns
`/home/agent/wt/evidence/fn234-overlay`; paths below are relative to that root.
The complete commands, environments, failures and custody notes are in
`HANDOVER.md`. Do not infer a runtime pass from declaration equality alone.

## Source and deletion custody

Pre-split source is `pre/source/`, archived from `0cd13c3b95`, before embedded
source edits or cache refresh. `pre/source.sha256` covers 9,452 tracked files;
its SHA-256 is `810cec76b8cc1c0a96ff9bd18cbcda001c3a197575c1b5320592b8ea70a2ab86`.
Post source is `post/source/`, with `post/source.patch` and
`post/changed-source.sha256`. Production checkpoints `469f92ca38` and
`d1a89a44ce` integrate as `c19e8af5ba` and `05e148b9de` respectively.

Only `tools/gomad/expected-intercepts-go1.27.1.txt` was deleted. The exact and
native-era proving searches are retained in `task1-required-native-era-sweep.log`
and the handover. Source/build/generator/fixture/operator-document searches
found no reader; the two filename references were historical/current Flow tasks.
All other scoped files have live consumers and remain. The DTrace executable
`clock_audit.d` is native qualification tooling, outside the overlay/toolchain
data-file cleanup boundary; the explicit retention and searches are in
`task1-clock-audit-{search,filename-search}.log`. No WASM-used native-named
source was deleted. Deletion alone changes none of the overlay identity inputs.

## Actual before/after execution inputs

Each side's `retained/` directory contains `identity.json`, the published
overlay, `runtime-list.json`, the sorted 3,997-entry `runtime-declarations.txt`,
and `choices/` and `temporal/` workload directories. Each workload retains
`module.wasm`, `installation.json`, `provenance.json`, `prepared.json`, and
`request-{0,7}.json` with complete configurations, limits, arguments and seeds.
`request-{0,7}-executor.json` binds the actual executor/helper/module digests.

| Input | Before SHA-256 | After SHA-256 |
| --- | --- | --- |
| Choices module | e013e6d82bc3d288846134398bfa3206c676f41d62863d49cf4959c29b853fe3 | eb4bb1dae18f0812497efb98938c6050234acead8db36812c6586f0909b9270c |
| Temporal TestFrontendSystemInfo module | 29f90f0aca2dedcf1aabe262ed50744edaaaa0d31119be4413772e8c11a366be | 3dc50be0b24f1e2c91f11b4ac590daab501c9db7e72cf496e5962ea1f28c5e19 |

The fixed primary executor `bin/executor` has SHA-256
`6448324f872201ad93fc83ab5255ebc983e0f563b0162056fe31254be7832894`.
Both sides use the same Go 1.27.1 compiler, SHA-256
`1675694ef690db0f18fbe7046a886170904bede1d9db6ec96ae27945c1705c64`,
and ordinary helper, SHA-256
`079b9bc2b510546292fac0c3ded2f927643c555eabcbcb76280b154f536c43fc`.
The runtime implementation changes from
`4ac4238ff22066e38a7e67356f1388bb8d09995a3fc9f2bea2c57fce8c284ac4`
to `b836473a142a30d55d2a5a4493665bc26cdd3d6e01df5c53228371e0c6df9937`.
Actual identity differences are retained, not normalized.

## Fresh observations and replay evidence

For each workload and seed 0/7, both sides retain fresh
`request-SEED-record.json`, `request-SEED-replay-request.json` and
`request-SEED-replay.json`. All eight originals and eight own full-tape
replays exited 0. Choices retain four records per seed; Temporal retains
3,943 and 3,926 records. The Temporal module was prepared through the actual
Temporal installation with captured SQLite inputs, not a stand-in guest.
Full profile/bounds and numeric command outcomes are in the handover and
`pre/{choices,temporal}-*.log` / `post/{choices,temporal}-*.log`.

Helper feature evidence is `pre/feature-runs-fixed/` and `post/feature-runs/`:
11 cases, 30 admitted case/profile combinations, two observations each side.
Each case retains raw frames/stderr, semantic views and source/guest/tool
identities. Four fixed helpers and their source/build manifests are in
`pre/helpers/`; `final-frozen-tools.sha256` binds them. The fixed feature
executor `bin/features-fixed.test` has SHA-256
`312d30964fe5bd9755138ab82432f9d5be25a7daf8caf7c576f035a0d600ea79`.
These helper-only observations supplement, rather than replace, the actual
Go-overlay executions; they are not forced-choice replay claims.

Run `env -u BASH_ENV -u SANDBOX_START_DIR /bin/bash /home/agent/wt/evidence/fn234-overlay/compare.sh`.
The root rerun exited 0. `comparison/report.json` has SHA-256
`bbae096ec245d78836897e970eba32122299453d8b2e9d609b0f154898fa19f6`:
68 paired labels (eight primary original/replay labels and 60 helper labels),
matching logical inputs, outcome/termination, first-divergence ordinal/reason,
four failure-group partitions, declaration lists and unchanged feature corpus.
The primary successful pairs have null divergence. Raw inputs/results remain
alongside the comparison; a summary alone is not its evidence.

## Verification and limits

Worker build, vet, lint, toolchain/runtimesource, backend/installation,
determinism/architecture and live `TestCooperativeInspectionEveryDecision`
checks exited 0; exact logs and commands are indexed in `HANDOVER.md`.
`task2-validate-green.log` and `task2-validate-green-exit.txt` record full
`make -C tools/gomad validate` exit 0 in 14 seconds on `d1a89a44ce`.
The earlier target-symlink inventory failure remains in `task2-validate.log`
and `task2-validate-exit.txt`; no inventory rule was relaxed.

Root's integrated `05e148b9de` focused checks and full validation also exited 0:
`root-05e148b9de-focused.log` and `root-05e148b9de-validate.log`. The disposable
private `go-cache/` was subsequently cleared with `go clean -cache`; it is not
evidence and can be regenerated. No required input/result or shared cache was
removed. The declaration-test portability repair `1755d2df0f` (root
`c62a9d43ef`) changes only the test to reuse `stockgo.Root(t)`. Its actual
downloaded-toolchain regression changes from exit 1 to exit 0; the direct
control still produces Go's module-cache overlay rejection. Exact driver,
RED/GREEN logs and numeric results are under `repair-downloaded-toolchain/`,
including `results.json`. Focused tests, vet and lint exit 0 with unchanged
runtime/overlay identities. This test-only repair does not invalidate the
retained runtime observations. Source custody was also independently rechecked
at root with `sha256sum -c` for both source manifests; logs are
`root-pre-source-verification.log` and `root-post-source-verification.log`.

The frozen fn-232 Go baseline and its authenticated extracted source are
untouched. This dedicated old/new overlay comparison does not substitute for
Go/Rust host parity or qualify the final integrated Rust candidate. The complete
`test-wasm`/runtime/workload/retained-replay qualification remains with fn-232.13.
