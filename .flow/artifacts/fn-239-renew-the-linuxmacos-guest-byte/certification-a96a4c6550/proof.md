# fn-239 Linux/macOS guest certification

Four clean builds at committed source
`a96a4c6550b468c5d7c3011a7961d4e118dacd93` produced byte-identical stock and
Temporal guests on 2026-10-22. All twelve actual pairwise comparisons passed.
Both stock/control and embedded Temporal host suites passed on Linux and native
macOS. Every Temporal output matches the committed asset.

This renews fn-232 after fn-238.8 added per-crate lockfile closure identities.
The source pin descends from the fn-238.8 merge `cbe2c6fcb7` and includes the
front-end integration. The proof uses the committed build recipe, dependencies
and Wasm asset. `run-proof.sh` and `compare.py` differ from the retained fn-232
certification only in their source pin and task output paths.

## Source and target guards

Each OS used two detached clones under
`.flow/tmp/fn239-certified-{linux,macos}-a96a4c6550/source-{a,b}`. The runner
required an initially absent proof root, source directories and independent
`target-{a,b}` directories. It checked the exact source revision and empty
`git status --porcelain --untracked-files=all` before and after each build, and
again after the host suites. No inherited compiler wrapper or flags were set.
Cargo compiled each target independently, sharing only downloaded dependencies.

`comparisons.json` repeats the four clean-source and revision checks, records the
identical source tree and four distinct target device/inode pairs, and retains
artifact sizes, SHA-256 hashes and all twelve `cmp` exit statuses.
`linux-manifest.json` and `macos-manifest.json` retain each runner's manifest.
The clean clones, build targets and proof binaries remain under the task's
`.flow/tmp` roots. They are outside the tracked proof files.

## Closure-map precheck

Before starting the builds, both native executors ran the following from the
repository root at the pinned revision, substituting their OS for `<os>`.

```sh
mkdir -p .flow/tmp/fn239-precheck-a96a4c6550/<os>
mise exec -- cargo metadata --manifest-path model/Cargo.toml --format-version 1 --locked | python3 model/scripts/guest-lock.py model/Cargo.lock .flow/tmp/fn239-precheck-a96a4c6550/<os>
```

Both produced `umpire-lock-88d30b801c701567.json`. The actual `cmp` returned 0;
`closure-precheck.json` records its paths, pin and SHA-256. `closure-map.json`
retains the common map. `closure-build-comparisons.json` records four further
successful `cmp` calls against each clean build's actual closure map.

## Native executors and tools

| Executor | OS | Architecture | Rust host |
| --- | --- | --- | --- |
| `sbx exec dev-build` | Linux 7.0.14 | aarch64 | aarch64-unknown-linux-gnu |
| Local Mac | Darwin 25.6.0 | arm64 | aarch64-apple-darwin |

Every clone used Rust 1.99.1, commit
`58cf8f9c35b355199727627d07ae34a4e2846d67`, LLVM 23.1.1, Cargo 1.99.1
(`5f94df478`), Go 1.27.0 and protoc 29.5. The runner recorded installed
components, `rust-src` and the `wasm32-wasip1` target. Each build used two Cargo
jobs, locked dependencies and the committed guest profile with rebuilt unwind std.

## Results

| Guest | Bytes in every clone | SHA-256 in every clone |
| --- | ---: | --- |
| Stock | 192489 | `bae203f82ecea5183d592232c62b746f6ccabda9944df53e46dd442482d1bc63` |
| Temporal | 3332314 | `d8127a2bf8c72de75cfafde655e35b5b1aa6355a023628708dd8ec6d22f83370` |

`compare.py` executed all six pairwise `cmp` calls per guest. All twelve returned
0. Both native runner exit statuses and `comparisons.rc` are 0. On each OS, the
runner also compared the gate-built stock and Temporal guests with the first
clean build and compared Temporal with its committed embedded asset. The twelve
comparisons establish that every clone's Temporal output matches that asset.

Each runner built stateful, caught-panic and overflow controls, generated native
panic reports, and exercised the full stock/control Go host suite. Controls
covered native/guest full-response and golden parity, repeated calls and Reset,
uncaught traps and exact import/descriptor denial. Temporal's separate Go module
used its committed embedded asset. Expected panic-hook output and Wasm
exception-handling feature warnings remain in the successful transcripts.

| Executor | Stock/control package | Temporal package |
| --- | ---: | ---: |
| Linux | 0.325 s | 9.692 s |
| macOS | 0.873 s | 10.484 s |

## Exact certification invocations

From `/Users/stephan/Workspace/skunkworks/umpire/temporal`:

```sh
sbx exec -w "$PWD" dev-build bash -c 'bash .flow/artifacts/fn-239-renew-the-linuxmacos-guest-byte/certification-a96a4c6550/run-proof.sh "$PWD" "$PWD/.flow/tmp/fn239-certified-linux-a96a4c6550"'
bash .flow/artifacts/fn-239-renew-the-linuxmacos-guest-byte/certification-a96a4c6550/run-proof.sh "$PWD" "$PWD/.flow/tmp/fn239-certified-macos-a96a4c6550"
python3 .flow/artifacts/fn-239-renew-the-linuxmacos-guest-byte/certification-a96a4c6550/compare.py "$PWD"
```

Each runner's stdout and stderr are retained in `<os>.log`, with its exit status
in `<os>.rc`. Comparator stdout is `comparisons.json`, with its exit status in
`comparisons.rc`. The runners log every executed command through `set -x`.
`SHA256SUMS` covers all files in this certification directory except itself.

## Repository gate

`make model-check` also passed on the pinned Linux `source-a` clone after its
certification runner completed, reusing that clone's target. The exact invocation
from the repository root was:

```sh
sbx exec -w "$PWD/.flow/tmp/fn239-certified-linux-a96a4c6550/source-a" -e CARGO_BUILD_JOBS=2 -e CARGO_TARGET_DIR="$PWD/.flow/tmp/fn239-certified-linux-a96a4c6550/target-a" dev-build make model-check
```

`model-check.log` retains the complete output and `model-check.rc` is 0. The gate
covered guest freshness, script and capability controls, source boundaries,
protobuf freshness, formatting, Clippy, docs, workspace tests, Wasm builds and both
Go host modules. Default mode skipped real TLC execution and agreement; this
certification makes no TLC claim. After the gate, the comparator repeated all
four source checks and twelve byte comparisons successfully, and the gate-built
guests still matched the certified outputs.
