# fn-232 Linux/macOS guest certification attempt

Certification is incomplete. On 2026-10-21, both guests reproduced across two clean
paths on each OS, but every Linux/macOS byte comparison failed. R1 remains unmet.
The build recipe and proof artifacts retain their original bytes. A separately
scoped reproducibility fix is required before retrying certification.

## Source and executors

All four clones checked out `8d429e97409b1537ecb347c359129bffafa63f78`, which includes
the completed fn-230 changes. The working checkout was shared with another agent;
its uncommitted changes and later commits never entered the detached proof clones.
The transcript records `git rev-parse HEAD` and empty
`git status --porcelain --untracked-files=all` checks. `comparisons.json` repeats
those checks independently for all four clones.

| Executor | OS | Architecture | Rust host |
| --- | --- | --- | --- |
| Local Mac | Darwin 25.6.0, xnu-12377.161.15.700.19~2 | arm64 | aarch64-apple-darwin |
| Existing `sbx exec dev-build` sandbox | Linux 7.0.14 | aarch64 | aarch64-unknown-linux-gnu |

Both executors reported Rust 1.99.1, commit
`58cf8f9c35b355199727627d07ae34a4e2846d67`, LLVM 23.1.1, Cargo 1.99.1
(`5f94df478`), Go 1.27.0 and protoc 29.5. Both had `rust-src` and the
`wasm32-wasip1` target installed. macOS needed the following prerequisite commands
before the proof. `mise exec -- rustc -Vv` installed the missing pinned compiler.

```sh
mise exec -- rustup component add rust-src rust-analyzer --toolchain 1.99.1
mise exec -- rustup target add wasm32-wasip1 --toolchain 1.99.1
```

Each clone used the unchanged `guest.sh`, pinned guest profile, `--locked`, rebuilt
std/unwinding, two Cargo jobs and no inherited Rust flags. Download caches could
be shared; the four compiled-artifact targets were independent and initially absent.
No worktrees, remote provisioning, dependency upgrades or commits were used.

## Exact execution

The repository path was
`/Users/stephan/Workspace/skunkworks/umpire/temporal` on both executors.
The retained [run-proof.sh](run-proof.sh) contains every clone, guard, toolchain,
build, hash, size, comparison and host-test command. Each clone ran
`mise trust` on its checked-out `mise.toml` before invoking the pinned tools.
The wrapper invocations were equivalent to these commands, with each shell's exit
code written immediately into the corresponding `.rc` file.

```sh
repository=/Users/stephan/Workspace/skunkworks/umpire/temporal
evidence="$repository/.flow/artifacts/fn-232-linux-macos-guest-byte-identity"
cd "$repository"
bash "$evidence/run-proof.sh" "$repository" "$repository/.flow/tmp/fn232-certification-macos" >"$evidence/macos.log" 2>&1
echo "$?" >"$evidence/macos.rc"
sbx exec -w "$repository" dev-build bash -c 'proof="$PWD/.flow/artifacts/fn-232-linux-macos-guest-byte-identity"; bash "$proof/run-proof.sh" "$PWD" "$PWD/.flow/tmp/fn232-certification-linux" >"$proof/linux.log" 2>&1; result=$?; echo "$result" >"$proof/linux.rc"; tail -35 "$proof/linux.log"; exit "$result"'
python3 "$evidence/compare.py" "$repository" >"$evidence/comparisons.json"
echo "$?" >"$evidence/comparisons.rc"
```

The exact source roots are `.flow/tmp/fn232-certification-{linux,macos}/source-{a,b}`
under that absolute repository path. Their targets are sibling `target-{a,b}`
directories. The retained per-OS manifests record the target device/inode pairs,
resolved paths, guest SHA-256 values and sizes. On Linux the target identities were
`58:453809937` and `58:453871988`; on macOS they were `16777231:453795894` and
`16777231:453887421`. Each runner asserts that the two targets are different objects.

## Actual byte comparisons

| OS | Guest | Bytes in both paths | SHA-256 in both paths |
| --- | --- | ---: | --- |
| Linux | Stock | 192255 | `5ec57e2e5371723adf318665c1048b5a4021e9f4203ed56884c5f849e7a2dfc1` |
| macOS | Stock | 192263 | `ea826b9088f9fae68943b149289d5646e349fc8f2751f0bab422312fe74e2e54` |
| Linux | Temporal | 3329990 | `10724102d03e1659f28b07c3a0c848d2cac5876668f3be9261d0a3b65023cf06` |
| macOS | Temporal | 3329996 | `791b33b3db2e8d57464d94fcdfb8b95594d15b434a6fb7e1c24905a5e4069547` |

[comparisons.json](comparisons.json) records all twelve actual `cmp` invocations,
their stdout/stderr, exit codes and first differing offsets. All four within-OS
comparisons exited 0. All eight cross-OS comparisons exited 1. Every cross-OS
stock comparison first differs at offset 443; every Temporal comparison first
differs at offset 635, both zero-based. The aggregate comparison command exited 1.

The Linux runner exited 0. The macOS runner exited 1 at the comparison with the
clone's committed Temporal asset, which equals the Linux build. Both macOS path
comparisons had already passed. The failed comparison was retained, and the
committed asset was left unchanged.

## Host parity

The Linux transcript includes the actual guest-profile control builds, native
panic-control reports, stock/control Go-host suite and Temporal Go-host suite.
`TestKettleParity`, caught-panic/overflow repeated-call and Reset controls, and
uncaught-trap controls passed. The generic module took 0.846 s and the Temporal
module took 13.240 s. Its embedded Temporal asset was compared to the newly built
artifact before those tests ran.

The macOS runner stopped before host tests because its Temporal bytes differed
from the committed Linux asset. [run-parity.sh](run-parity.sh) separately tests
the actual macOS stock/control artifacts and a temporary copy of the existing
Temporal host test module with the macOS guest copied into its embedded-asset
location. Only that test copy's local module replacement and asset differ from
the pinned test sources. The four clean proof clones and all guest outputs remain
unchanged. This behavioral check cannot satisfy byte identity.

```sh
bash "$evidence/run-parity.sh" "$repository/.flow/tmp/fn232-certification-macos" >"$evidence/macos-parity.log" 2>&1
echo "$?" >"$evidence/macos-parity.rc"
```

The macOS parity runner exited 0. Its generic host module took 1.151 s and its
Temporal host module took 11.154 s. The stock goldens, caught-panic/overflow,
repeated calls/Reset and uncaught-trap controls passed. `macos-parity.log` records
the actual test output and final clean-source guards; `macos-parity.rc` retains 0.
No full `make model-check` success is claimed for this proof-only task.

## Mismatch investigation and retained evidence

[diagnostics.log](diagnostics.log) records wasm-tools 1.244.0, wasm-objdump 1.0.39
and the stock section inventories. Both artifacts have 352 defined functions,
the same section inventory, and no custom sections. The code section is 172917
bytes on Linux and 172925 on macOS. The first binary difference lies in the
function section. The [compressed WAT diff](stock-wat.diff.gz) additionally shows
different function-table entries, function bodies and data layout. The exact
diagnostic commands were:

```sh
wasm-tools print .flow/tmp/fn232-certification-linux/stock-a.wasm >.flow/tmp/fn232-stock-linux.wat
wasm-tools print .flow/tmp/fn232-certification-macos/stock-a.wasm >.flow/tmp/fn232-stock-macos.wat
diff -u .flow/tmp/fn232-stock-linux.wat .flow/tmp/fn232-stock-macos.wat
wasm-objdump -h .flow/tmp/fn232-certification-linux/stock-a.wasm
wasm-objdump -h .flow/tmp/fn232-certification-macos/stock-a.wasm
```

The initial WAT files and diff were moved from the evidence directory into
`.flow/tmp/fn232-*` after inspection; the retained gzip contains that original
diff, including its original path headers. No diagnostic rewrote a Wasm artifact.
The compiler/linker cause remains unresolved. Pinning source and versions plus
path remapping proves the observed within-OS result, but does not eliminate the
observed cross-OS code-generation difference.

The per-OS logs, `.rc` files, manifests and comparison report live beside this
report. Original Wasm artifacts, native control reports, source clones and build
targets remain in the two task-owned `.flow/tmp/fn232-certification-*` directories
while certification is open. Historical fn-230 source pins, Linux hashes and gate
results remain in `model/GUEST_BUILDS.md`; those receipts do not certify this
source pin or macOS output. The design summary remains within 1000 lines.
