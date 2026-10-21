# fn-232 Linux/macOS guest certification

Certification passed on 2026-10-21 at committed source
`3c314fb5b85427471a1525d6f190fe5d1fb9f3b2`. Four fresh, clean clones produced
byte-identical stock and Temporal guests. All twelve actual pairwise comparisons
passed. Both native stock/control and embedded Temporal host suites passed.

The separately tracked fn-236 implementation supplied the permanent compiler-input
fix. This certification uses the committed builder, without reference hash maps
or binary rewriting. The earlier failure at `8d429e97409b1537ecb347c359129bffafa63f78`
remains in the parent directory as historical evidence.

## Source and target guards

Each OS used two detached clean clones under
`.flow/tmp/fn232-certified-{linux,macos}-3c314fb5b8/source-{a,b}`. The runner checked
the exact revision and empty `git status --porcelain --untracked-files=all` before
and after each build, and again after the actual host suites. It required the
proof root and each `target-{a,b}` to be initially absent. Only downloaded
dependencies were shared. No inherited compiler wrappers were configured on
either executor; each target compiled its own artifacts.

`comparisons.json` independently repeats all four final clean-source/revision
checks, records the identical Git tree, lists each artifact's byte count/hash,
and records four distinct target device/inode pairs. `linux-manifest.json` and
`macos-manifest.json` retain each native runner's artifact/target manifest.

## Native executors and pinned tools

| Executor | OS | Architecture | Rust host |
| --- | --- | --- | --- |
| Existing `sbx exec dev-build` | Linux 7.0.14 | aarch64 | aarch64-unknown-linux-gnu |
| Local Mac | Darwin 25.6.0 | arm64 | aarch64-apple-darwin |

Every clone used Rust 1.99.1, commit
`58cf8f9c35b355199727627d07ae34a4e2846d67`, LLVM 23.1.1, Cargo 1.99.1
(`5f94df478`), Go 1.27.0 and protoc 29.5. Installed `rust-src` and the
`wasm32-wasip1` target were recorded. Both runners used two Cargo jobs, the locked
dependency graph and the committed guest profile with rebuilt unwind std.
The transcripts retain the exact compiler versions, component/target lists,
commands, cleanliness guards and final exit statuses.

## Artifact and comparison results

| Guest | Bytes in every clone | SHA-256 in every clone |
| --- | ---: | --- |
| Stock | 192271 | `f3da11cbe34e75e24ec698979058a6f300aad016e5215a63fe2d392951341219` |
| Temporal | 3330006 | `5bd641822d7a124f10b247d022d8f6d296fe13632f418acfa23bd85152a6285e` |

`compare.py` executed all six pairwise `cmp` calls per guest. All twelve returned
0, covering both within-OS path comparisons and every Linux/macOS pairing.
Both runner statuses and `comparisons.rc` are 0. The gate-built artifacts and
each clone's committed embedded Temporal asset match the certified bytes.

## Actual host controls

Each native runner built stock, stateful, caught-panic, overflow and Temporal
guests through the committed builder, then generated native panic-control reports
and ran the full stock/control Go host suite and Temporal Go module. Controls
passed for caught panic/overflow parity against native reports and existing
goldens, repeated calls and Reset, uncaught trap/invalidation behavior and exact
import/descriptor denial. Temporal used its committed embedded asset, which the
runner compared with the certified Temporal output before testing.

Linux package times were 0.611 s for stock/control and 14.227 s for Temporal.
macOS package times were 0.898 s and 10.304 s. Linux ran from 18:09:44 to
18:19:10 UTC; macOS ran from 18:09:44 to 18:19:56 UTC. Both exited 0 and retained
clean clones after the suites. The expected native panic-hook messages are
caught-panic/overflow controls, and both runners continued successfully.

## Exact invocations

From `/Users/stephan/Workspace/skunkworks/umpire/temporal`:

```sh
sbx exec -w "$PWD" dev-build bash -c 'bash .flow/artifacts/fn-232-linux-macos-guest-byte-identity/certification-3c314fb5b8/run-proof.sh "$PWD" "$PWD/.flow/tmp/fn232-certified-linux-3c314fb5b8"'
bash .flow/artifacts/fn-232-linux-macos-guest-byte-identity/certification-3c314fb5b8/run-proof.sh "$PWD" "$PWD/.flow/tmp/fn232-certified-macos-3c314fb5b8"
python3 .flow/artifacts/fn-232-linux-macos-guest-byte-identity/certification-3c314fb5b8/compare.py "$PWD"
```

`run-proof.sh` pins the revision above and retains the complete clean-clone,
toolchain, build and host commands. Its only change from the original fn-232
runner is the source pin. `linux.log`, `macos.log`, the three `.rc` files,
manifests and `comparisons.json` retain the executed results. `SHA256SUMS` covers
this certification's files.
