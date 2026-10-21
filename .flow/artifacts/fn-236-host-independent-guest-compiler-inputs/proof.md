# Self-contained native Linux/macOS validation (fn-236)

All twelve actual byte comparisons passed for stock and Temporal, across two
independent source paths on each native OS. This validates the permanent compiler
input fix without reference platform hash maps, copied compiler outputs or binary
rewriting. fn-232's final certification still requires a user-created commit
containing the fix and another run from clean committed checkouts.

## Source and isolation

Base commit: `77d2e4e90eeea14711ebfd050f41f0ef3b1d0b1e`. Seven reviewed overlays
are listed in `overlay-manifest.json`; every overlay matches all four source
snapshots, including the new helper and test. Other agents' working changes were
excluded by starting from that commit. No commit or worktree was created.

`prepare-snapshots.py` materialized independent clones under
`.flow/tmp/fn236-snapshot-proof/{linux,macos}/source-{a,b}`. Their Git statuses
explicitly show the uncommitted overlay. These are frozen validation snapshots,
not clean committed-checkout certification receipts. Initially absent, independent
`target-{a,b}` directories were checked before building. Downloaded dependencies
were shared; compiled outputs were not. `comparisons.json` records distinct target
device/inode identities.

The tracked-file plus helper build-input manifest has SHA-256
`078f72231143d5243581673b9fa5a33b32e3897d07aab73dc853b612ecb899ee`.
Per-path manifests compare unchanged before and after building; the final host
checks also compare source-a again. The overlay manifest additionally covers the
untracked regression test. Generated ignored build outputs are excluded. Frozen,
before/after and final manifests are retained losslessly as `.json.gz` archives.

## Executors and results

Native Linux 7.0.14/aarch64 ran through `sbx ... dev-build`; native macOS
25.6.0/arm64 ran directly. Both used Rust 1.99.1, commit
`58cf8f9c35b355199727627d07ae34a4e2846d67`, LLVM 23.1.1, Cargo 1.99.1,
Go 1.27.0 and protoc 29.5. Complete platform/toolchain transcripts are in
`linux-snapshots.log` and `macos-snapshots.log`. Each build used two Cargo jobs,
the locked dependency graph, the existing guest profile and rebuilt unwind std.

| Guest | Bytes in each of four builds | SHA-256 in each build |
| --- | ---: | --- |
| Stock | 192271 | `f3da11cbe34e75e24ec698979058a6f300aad016e5215a63fe2d392951341219` |
| Temporal | 3330006 | `5bd641822d7a124f10b247d022d8f6d296fe13632f418acfa23bd85152a6285e` |

`compare.py` executes all six pairwise `cmp` calls for each guest. All twelve
returned 0; `comparisons.json` records each invocation's paths, status and output.
Both within-OS path comparisons and all cross-OS comparisons passed. Each Temporal
output also matches its source snapshot's refreshed embedded asset.

## Verification

Sixteen focused script tests passed, including malformed/missing dependency
identity refusal, variant separation, compiler failure, metadata pipelining,
inherited jobserver descriptors and real macOS loader-environment preservation.
The focused log and independent review are retained alongside this report.

Stock/control and Temporal Go host suites passed on both OSes, including caught
panic/overflow parity, repeated calls and Reset, uncaught traps and import denial
controls. Linux host packages took 0.532 s and 25.040 s; macOS packages took
1.145 s and 28.431 s. Both native runner exit statuses are 0. The full
required-backend model gate is still running.

## Exact commands

From `/Users/stephan/Workspace/skunkworks/umpire/temporal`:

```sh
python3 .flow/artifacts/fn-236-host-independent-guest-compiler-inputs/prepare-snapshots.py "$PWD"
sbx exec -w "$PWD" dev-build bash -c 'bash .flow/artifacts/fn-236-host-independent-guest-compiler-inputs/run-snapshots.sh "$PWD" linux'
bash .flow/artifacts/fn-236-host-independent-guest-compiler-inputs/run-snapshots.sh "$PWD" macos
python3 .flow/artifacts/fn-236-host-independent-guest-compiler-inputs/compare.py "$PWD"
python3 -B -m unittest discover -s model/scripts/tests -p 'test_*.py'
bash -n model/scripts/guest.sh .flow/artifacts/fn-236-host-independent-guest-compiler-inputs/run-snapshots.sh
```

The full gate runs from `.flow/tmp/fn236-candidate`, the same base plus overlay,
with the checksum-pinned TLC jar copied into its ignored cache:

```sh
mise exec -- env GOBIN="$PWD/.bin" CGO_ENABLED=0 go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.6
cp .bin/protoc-gen-go .bin/protoc-gen-go-v1.36.6
CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn236-builds/gate-macos CARGO_BUILD_JOBS=2 UMPIRE_BACKENDS=require mise exec -- make model-check
```

The first gate attempt stopped at protobuf freshness because the copied shared
`protoc-gen-go` was a Linux ELF binary. The isolated checkout now has its own native
macOS v1.36.6 binary. The original failed log/status are retained as
`model-check-macos-attempt-1.{log,rc}`; the rerun uses the unchanged model gate.
