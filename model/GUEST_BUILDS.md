# Guest builds: reproduction and diagnostics

The Linux proof's hashes, sizes, source commit, platform and toolchain are recorded
in [DESIGN.md](DESIGN.md). Actual WASI unwinding/caught-panic judgments remain
unresolved, and macOS/cross-OS byte identity remains unverified. These commands
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

## Two-path Linux proof

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
No reachable macOS executor was found.

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
