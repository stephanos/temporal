# Guest builds: reproduction and diagnostics

Historical and current Linux proofs' hashes, sizes, source commits, platform and toolchain are recorded
in [DESIGN.md](DESIGN.md). Actual WASI unwinding/caught-panic judgments remain
unresolved under fn-230.1. fn-232-linux-macos-guest-byte-identity owns the separate,
still-unverified macOS/cross-OS proof, which no longer gates fn-230. These commands
document the existing Linux evidence; they do not discharge those requirements.

`bash model/scripts/guest.sh <crate> <output> [--check]` builds a cdylib with the
mise-pinned toolchain, `--locked`, `wasm32-wasip1` and `profile.guest`. It discovers
the output through Cargo's artifact messages, including custom library names.
The script uses Bash and Python 3's standard library; it adds no Cargo dependency.
Exit 0 means built or fresh, 1 means build/tool/output failure, and 2 means missing
or stale output. Staleness reports existing/built sizes and the first differing
zero-based byte offset, including an EOF difference; check mode never updates it.
Relative output paths are relative to the caller. `CARGO_TARGET_DIR` selects the
build cache, defaulting to `model/target`.

The profile inherits release, strips symbols, disables debug information and uses
fat LTO with one codegen unit and checked overflow. The checker's release profile
is unchanged. Workspace, Cargo registry, toolchain sysroot and target-directory
paths are remapped by rustc, including generated protobuf sources under `OUT_DIR`.
The first independent-target proof failed at stock byte 130109 because generated
protobuf panic locations retained the target path; remapping that directory fixed
both guests. No wasm bytes are rewritten after the stock linker runs.

## Panic diagnostics

The manifest specifies `panic = "unwind"`, but the pinned Cargo/rustc target still
compiles this profile with `panic="abort"`. This does not establish caught-panic
judgments in WASI. The actual profile artifacts preserve the existing
trap-and-invalidate behavior. The diagnostic commands were:

```sh
cd model
CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2301-panic-probe \
  mise exec -- cargo rustc -p umpire-stock-guest --lib --profile guest \
  --target wasm32-wasip1 --locked -- --print cfg
# panic="abort"
printf '%s\n' 'pub fn probe() -> bool { std::panic::catch_unwind(|| panic!("probe")).is_err() }' | \
  mise exec -- rustc --crate-type cdylib --target wasm32-wasip1 -Cpanic=unwind \
  --crate-name panic_probe -o ../.flow/tmp/panic_probe.wasm -
# exit 1: the crate `panic_unwind` does not have the panic strategy `unwind`
```

## Historical two-path Linux proof

The exact build/comparison commands, from the repository root (the temporary
clones were made with `git clone --shared` and checked out at
`df58e04a076f9d44392a0151e812ad3ccf0824fc`):

```sh
root=$(pwd -P)
source_a="$root/.flow/tmp/fn2301-source-a"
source_b="$root/.flow/tmp/fn2301-different-source-b"
for pair in a b; do
  if [ "$pair" = a ]; then source_dir=$source_a; else source_dir=$source_b; fi
  target="/home/agent/.cache/codex-build/fn2301-proof-final-$pair"
  CARGO_TARGET_DIR="$target" bash "$source_dir/model/scripts/guest.sh" \
    umpire-stock-guest "$root/.flow/tmp/stock-final-$pair.wasm"
  CARGO_TARGET_DIR="$target" bash "$source_dir/model/scripts/guest.sh" \
    umpire-temporal-guest "$root/.flow/tmp/temporal-final-$pair.wasm"
done
cmp .flow/tmp/stock-final-a.wasm .flow/tmp/stock-final-b.wasm
cmp .flow/tmp/temporal-final-a.wasm .flow/tmp/temporal-final-b.wasm
sha256sum .flow/tmp/{stock,temporal}-final-{a,b}.wasm
wc -c .flow/tmp/stock-final-a.wasm .flow/tmp/temporal-final-a.wasm
```

Both `cmp` calls exited 0. `git status --porcelain` was empty in both clones before
and after; targets were initially absent and independent. The Cargo registry
supplied dependencies, but compiled targets were not shared.
This proves Linux path independence only. macOS and cross-OS byte identity remain
**unverified**: this executor is Linux/aarch64, and a read-only SSH `uname -s` probe
using the available agent's macOS host closed the connection before executing.
No reachable macOS executor was found. The owner subsequently moved this check
to fn-232-linux-macos-guest-byte-identity; the failed SSH probe remains historical evidence.

## Actual-profile parity and controls

The actual guest-profile stock report and ordered rows match the native Kettle
goldens. `guest-parity.sh` overlays two panic controls into the existing Go host
suite without modifying the host: observation panic and outcome-comparison panic
both trap at `METHOD_JUDGE: umpire_call`, invalidate the instance and reject Reset.
The existing export/import, protocol and stateful controls run in that same suite.
These are evidence of the existing WASI behavior, not of unwinding. Reproduction
from the repository root, using `root` and `source_a` from the Linux proof block:

```sh
CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2301-proof-final-a \
  bash "$source_a/model/scripts/guest.sh" umpire-control-guest "$root/.flow/tmp/control-final-a.wasm"
UMPIRE_WASM="$root/.flow/tmp/stock-final-a.wasm" \
  UMPIRE_STATEFUL_WASM="$root/.flow/tmp/control-final-a.wasm" \
  bash model/scripts/tests/guest-parity.sh
CGO_ENABLED=0 UMPIRE_TEMPORAL_WASM="$root/.flow/tmp/temporal-final-a.wasm" \
  mise exec -- go -C model/temporal/hosts/go test -tags test_dep -count=1 ./...
python3 -m unittest discover -s model/scripts/tests -v
bash -n model/scripts/guest.sh model/scripts/tests/guest-parity.sh
```

Baseline and final `CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2301-guest
make model-check` passed. With the same isolated target, `make model-time` exited 0:
the warm gate took 12.332 s (30 s soft target), and the edit/test inner loop took
5.084 s (10 s soft target). The existing gate still builds release-profile guests;
the separate parity commands above establish the guest-profile results. Freshness
and embedding integration belong to fn-230.2 and are not included in this timing.

## Descriptor-order regression and current Linux proof

The newer closure at `feb4621de85e1889f4dd071ef3875b97a2ad8d22` failed the same
two-clean-source test. Stock remained identical at 143022 bytes and SHA-256
`f01424be60038282bcad227110ba76b6eb34a6cb3cc934cfb7422fa48bb23a9f`.
Temporal A/B each had 2128089 bytes but differed at zero-based offset 26293:
A `10d3e0af5c4f805bf44b3622d63c9da83a4356000fa90a5357d8476b0430e6c2`,
B `5a83e093fd50a410f61cf1e2089e717389c45bdee4d44e1f6c44dbad1045935c`.
The original failure, artifacts, generated sources and host parity results remain
under `.flow/tmp/fn230-current-proof-mgeeue/`; the earlier small proof above is
historical evidence, not proof of this newer closure.

`getproto` emits descriptor files by Go map iteration; prost preserves that order
when combining definitions from files sharing a package. The real API generation
regression reverses the upstream descriptor file list and compares every emitted
Rust filename and byte. It failed on `cel.expr.rs` in reproduction commit
`50463fdea7`, before the fix. The consuming API generator now sorts only the merged
file list by name immediately before `compile_fds`; messages, fields, enum values,
descriptor caches and the system-independent guest builder are unchanged.
The regression, existing freshness test and all four descriptor-admission controls
pass, without rewriting WASM or adding dependency versions.

On 2026-10-21 UTC, source `405889a77ac5364b53fa71acabc516778b531a80` produced
the current matching hashes/sizes in DESIGN. Both clean clones stayed at that pin
and clean before/after. Their initially absent targets had distinct identities
`29:9219512` and `29:9307580`; only downloaded dependencies were shared. Toolchain:
Linux 7.0.14/aarch64, Rust 1.99.1 (`58cf8f9c35b355199727627d07ae34a4e2846d67`,
LLVM 23.1.1), Cargo 1.99.1 (`5f94df478`), Go 1.27.0, protoc 29.5. The build overrode
`CARGO_BUILD_JOBS=4` and the per-clone target, with incoming Rust flags unset.

Exact reproduction commands from the repository root follow. The retained run used
`proof="$root/.flow/tmp/fn2301-determinism-proof-FclJoj"` and the target prefix shown
below; a repeat uses fresh source/output and target directories.

```sh
root=$(pwd -P)
proof="$root/.flow/tmp/fn2301-determinism-proof-FclJoj"
pin=405889a77ac5364b53fa71acabc516778b531a80
for lane in a b; do
  source_dir="$proof/source-a"
  if [ "$lane" = b ]; then source_dir="$proof/different-absolute-source-b"; fi
  git clone --shared --no-checkout . "$source_dir"
  git -C "$source_dir" checkout --detach "$pin"
  test -z "$(git -C "$source_dir" status --porcelain=v1)"
  target="/home/agent/.cache/codex-build/fn2301-determinism-proof-FclJoj-$lane"
  test ! -e "$target"
  CARGO_BUILD_JOBS=4 CARGO_TARGET_DIR="$target" bash "$source_dir/model/scripts/guest.sh" \
    umpire-stock-guest "$proof/stock-$lane.wasm"
  CARGO_BUILD_JOBS=4 CARGO_TARGET_DIR="$target" bash "$source_dir/model/scripts/guest.sh" \
    umpire-temporal-guest "$proof/temporal-$lane.wasm"
  test -z "$(git -C "$source_dir" status --porcelain=v1)"
done
cmp "$proof/stock-a.wasm" "$proof/stock-b.wasm"
cmp "$proof/temporal-a.wasm" "$proof/temporal-b.wasm"
sha256sum "$proof"/{stock,temporal}-{a,b}.wasm
wc -c "$proof"/{stock,temporal}-{a,b}.wasm
source_a="$proof/source-a"
target_a=/home/agent/.cache/codex-build/fn2301-determinism-proof-FclJoj-a
CARGO_BUILD_JOBS=4 CARGO_TARGET_DIR="$target_a" bash "$source_a/model/scripts/guest.sh" \
  umpire-control-guest "$proof/control-a.wasm"
GOCACHE=/home/agent/.cache/codex-build/fn2301-determinism-proof-FclJoj-go \
  UMPIRE_WASM="$proof/stock-a.wasm" UMPIRE_STATEFUL_WASM="$proof/control-a.wasm" \
  bash "$source_a/model/scripts/tests/guest-parity.sh"
for lane in a b; do
  GOCACHE=/home/agent/.cache/codex-build/fn2301-determinism-proof-FclJoj-go \
    CGO_ENABLED=0 UMPIRE_TEMPORAL_WASM="$proof/temporal-$lane.wasm" \
    mise exec -- go -C "$source_a/model/temporal/hosts/go" test -tags test_dep -count=1 -v ./...
done
for crate in stock temporal; do
  CARGO_BUILD_JOBS=4 CARGO_TARGET_DIR="$target_a" bash "$source_a/model/scripts/guest.sh" \
    "umpire-$crate-guest" "$proof/$crate-a.wasm" --check
done
```

The complete retained runner is `$proof/run-proof.sh`; `$proof/terminal-results.log`
captures exact commands, working directories, timestamps and terminal exit codes.
Session 84844 exited 0. Both comparisons and actual freshness checks passed.
Stock/control parity passed 19 top-level tests (0.875 s), including native report
and ordered-row goldens and both panic trap/invalidation controls. Each Temporal
artifact independently passed all 22 tests (47.260 s and 47.458 s). Control A was
103754 bytes, SHA-256 `62e593deea35a7e39b562e2e416e68a20bbbef40354109b533ef566995a11647`.
These are profile.guest artifacts, not the release artifacts built by model-check.
Both clones, independent targets and all old/new proof artifacts are retained.

Post-fix full verification used the isolated lane:

```sh
UMPIRE_BACKENDS=require CARGO_BUILD_JOBS=4 \
  CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2301-determinism-fix \
  GOCACHE=/home/agent/.cache/codex-build/fn2301-determinism-fix-go make model-check
CARGO_BUILD_JOBS=4 CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2301-determinism-fix \
  GOCACHE=/home/agent/.cache/codex-build/fn2301-determinism-fix-go make model-time
```

Both commands exited 0 (sessions 4430 and 20204). The required gate ran shared
source/TLC agreement and semantic controls, not Temporal TLC agreement; the current
Temporal source shape remains refused. The final warm default gate took 56.738 s,
exceeding the 30 s soft target; the Temporal Go-host step took 46.277 s. The
edit/test inner loop took 1.886 s, under its 10 s soft target. The pre-edit lane
measured 76.864 s / 5.845 s, with 67.886 s in the Temporal host step. These are
individual executor observations, not idle-host performance guarantees.
Full-suite outputs and observed exit codes were captured once; receipt attempts
returned `NO_RECEIPT` because the externally owned `MILESTONES.md` draft is dirty.
The draft was preserved, and actual terminal command results supply the evidence.

The codegen correction establishes current Linux path independence. Actual WASI
caught-panic/unwind judgments remain unresolved under fn-230.1, so its dependency
for fn-230.2 remains unsatisfied. The owner moved macOS/cross-OS proof to
fn-232-linux-macos-guest-byte-identity, with no dependency back from fn-230.
