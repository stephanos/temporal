# Rust verdict measurements and editor evidence

## Final measurements and editor evidence

Measured 2026-10-19 at source checkpoint `7d1b9a5e93aa87b822d2227fbcd6b053a61e2f39`,
after the flat surface and sample renames. Subsequent changes in this task only record these
observations. The canonical sample SHA-256 is
`76bf39abd4a108f7d7b72054b731b6e9506cfa65c803487bcb2d8dc41788806d`.
No concurrent latency measurements ran. Host: Linux `dev-build 7.0.14`, aarch64, 12 CPUs;
Rust/rust-analyzer 1.99.1 (`58cf8f9`, 2026-10-13), Cargo 1.99.1, Go 1.27.0, wazero 1.12.0.
Raw CPU/OS data, exact argv/cwd/environment, artifact paths, hashes and every subprocess exit
are retained under `.flow/tmp/fn2107-measure-final/`; native profile metadata and raw samples
are in `fn2107-measure-final-release/` and `fn2107-measure-final-dev/` beside it.

### Native fresh exploration and all claims

Each of five fresh processes makes three calls, without warmup. Every call constructs a fresh
exploration at budget 1,048,576, then calls `verify_all`; it never reuses an explored Result.
World construction is outside the in-process timer; starts, claims, exploration and report
cloning are inside. Formatting, assertion, printing and destruction are outside. Every call
reports 81 starts, 283 states, 424 emitted edges, 1,132 evaluated actions, 708 refusals, depth
62, no cut, all four claims Verified and no vacuity. The table reports every sample in order.

| Profile / process | First call (µs) | Second call (µs) | Third call (µs) | Whole process wall (ms) |
| --- | ---: | ---: | ---: | ---: |
| release / 0 | 102.917 | 69.000 | 66.333 | 15.206625 |
| release / 1 | 200.084 | 74.250 | 73.333 | 23.011500 |
| release / 2 | 103.375 | 72.750 | 67.375 | 23.628125 |
| release / 3 | 133.750 | 70.750 | 67.083 | 7.820500 |
| release / 4 | 101.583 | 1,249.042 | 658.417 | 29.888334 |
| dev / 0 | 1,048.083 | 988.250 | 991.667 | 4.443792 |
| dev / 1 | 1,209.875 | 1,099.583 | 1,125.959 | 4.324500 |
| dev / 2 | 1,438.792 | 1,015.792 | 984.959 | 4.423708 |
| dev / 3 | 1,024.959 | 979.042 | 1,017.125 | 7.115750 |
| dev / 4 | 1,091.750 | 989.292 | 988.083 | 3.880875 |

Process wall includes spawn/startup, all three calls, output capture and shutdown, excluding
builds. It is not a startup-only estimate. The release outliers remain in the table; there was
no retry or minimum selection. The standalone driver includes the actual fixture and links
the exact rlibs reported by that invocation's Cargo JSON, rather than selecting a cache glob.
Both profiles use overflow checks and unwind. Release uses opt-level 3, debug 1, thin LTO,
one codegen unit and no debug assertions; dev uses opt-level 0, debug 2, no LTO, 256 codegen
units and debug assertions. These explicit driver flags match the workspace profiles;
environment rustflags were absent. Driver compilation is outside both timing boundaries.

Reproduction from `temporal/`, with the shared target idle (output directories must be new):

```sh
CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target python3 .flow/tmp/fn2107-measure-native.py --execute --out .flow/tmp/fn2107-measure-final-release
CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target python3 .flow/tmp/fn2107-measure-native.py --execute --profile dev --out .flow/tmp/fn2107-measure-final-dev
```

The frozen Scala note measured the first public `verifyAll` call as **0.365004916 s**, then
0.109971000 and 0.111102584 s in the same JVM, before instance-retention fixes. The first
call already includes lazy checker initialization and JIT effects. It excludes JVM startup,
build, Model construction, initial-state materialization and rendering; later calls reuse
the Model and caches. Rust times fresh exploration each call and has a smaller feature scope.
These are historical cross-implementation observations, not a controlled speed ratio.
Standalone JVM startup, Scala clean/incremental builds and controlled post-fix Scala timings
are **unavailable**. The archived source was neither rebuilt nor repaired.

### Test-binary and build wall time

Cargo JSON from `cargo test -p umpire-tests --release --no-run --locked --offline
--message-format=json` identifies the actual `fixtures` executable. Its `--list` contains
`fixtures: test`; each `--exact fixtures --nocapture` run reports one passed test, exercising
all 23 shared Models and deterministic rendering. Five whole-binary walls were
**117.228501, 66.000416, 67.033625, 65.912000 and 65.446000 ms**. This includes fixture I/O,
rendering and comparisons and is not a Kettle-only checker timing. Actual executable paths,
list output and all five logs are in `fn2107-measure-final/test-*`.

The build command was `mise exec -- cargo build -p umpire-testing --lib --release --locked
--offline`, covering the DSL, macros, checker and testing library. A new empty target from
`mktemp` was used; downloaded registry sources were already cached. No shared target was
cleaned. Whole-command walls, including mise/Cargo startup, were:

| Build boundary | Wall (s) |
| --- | ---: |
| Clean artifacts, empty temporary target | 5.612547878 |
| Same command, no source change | 0.119822000 |
| Checker source mtime touched, same bytes, rebuild dependents | 1.356116667 |

The third row models edit-triggered dependency recompilation without changing behavior;
it does not measure a particular code edit or Rust's incremental compiler cache (release
incremental compilation is off). `clean-target.txt` records the retained temporary target;
`clean-build`, `nochange-build` and `edit-rebuild` logs show the actual scope and commands.

### Parameterized-world execution proof

The final manual-trait probe compiled against the exact final dev-profile Cargo rlibs and
matched all three literal expectations in `fn210-parameter-readiness.expected`:

```text
u8 start=0 limit=1 actions=2 states=2 edges=1 verified=true
u16 start=2 limit=4 actions=3 states=3 edges=3 verified=true
u16 start=5 limit=4 refuted=true
```

Its generic `World<T>` carries starts, a limit and an input domain; `Input<T>::Set` carries
values. Invariant, postcondition and progress predicates read the world. This proves the
manual engine contracts above on two type instantiations and the invalid-start control;
it implements none of the five deferred attribute gaps. Source, expected output, exact
compiler argv and output are retained as `fn210-parameter-readiness.rs`, `.expected` and
`fn2107-measure-final/parameter-*` under `.flow/tmp`.

### Macro-enabled rust-analyzer

The actual LSP client used the `model/Cargo.toml` workspace, idle shared Cargo target,
locked/offline build scripts, enabled proc macros and attribute expansion, and native
diagnostics with experimental diagnostics enabled. Cargo check/check-on-save were disabled,
so the unsaved diagnostic below is native analyzer evidence. Workspace readiness was
5.725970 s after opening (6.169540 s from launch), using warm build artifacts. Expansion
proved generated `World`, `world()`, `Machine for World` and `type State = Model`.

Definitions returned the authored file, field at zero-based (19,12) and helper at (95,11),
from calls at (102,26) and (102,78). Hovers showed `heating_on: bool` and
`fn never_switched_off() -> Assumption`. Individual request latencies in milliseconds:

| Request | First | Warm 1 | Warm 2 | Warm 3 |
| --- | ---: | ---: | ---: | ---: |
| Field definition | 8.216875 | 0.333500 | 0.251250 | 0.251416 |
| Field hover | 11.787375 | 0.362250 | 0.223792 | 0.246667 |
| Helper definition | 0.463125 | 0.317041 | 0.268959 | 0.212791 |
| Helper hover | 0.656000 | 0.477459 | 0.301791 | 0.232167 |

An unsaved version-2 overlay changed only that field reference to `heating_oon`. Explicit
`textDocument/diagnostic` returned native rust-analyzer E0559, “no field `heating_oon` on
type `&Model`”, at the edited range. Observation took **0.581092 s** including the deliberate
overlay hover/expansion checks before the pull. Restoring the exact version-3 text returned
empty diagnostic items in **0.051902 s**. Source hash remained identical; shutdown returned
0 with both reader threads stopped. These results establish the tested fields/helper,
not universal diagnostics for arbitrary macros or generic blanket traits.

An initial push-only client never received the typo diagnostic (`fn2107-ra-final-result.json`,
`fn2107-ra-experimental-result.json`, `fn2107-ra-overlay-result.json`); the server advertises `diagnosticProvider`,
and the corrected pull run is `fn2107-ra-pull-result.json`. This was a client correction, with
no macro change; the analyzer was stopped before benchmark and gate timing runs.

### Final stock guest and Go host

`cargo build -p umpire-wasm --target wasm32-wasip1 --release --locked --offline
--message-format=json` selected the actual guest artifact
`/home/agent/.cache/codex-build/cargo-target/wasm32-wasip1/release/umpire_wasm.wasm`.
It is **3,423,000 bytes**, including release debug information, SHA-256
`e6f119c9929c735a58012b2ee24370a970cf5d77b005295511bf1712893cbd12`.
The CGO-disabled Go host's uncached `go test -tags test_dep -count=1 ./...` passed,
retaining byte-for-byte native trait Kettle report/ordered-Rows parity and ABI/import controls.
That stock-toolchain parity is R24's recommendation criterion.

The actual benchmark commands, from `model/umpire/hosts/go`, set `UMPIRE_WASM` to that
artifact and `CGO_ENABLED=0`:

```sh
mise exec -- go test -tags test_dep -run '^$' -bench '^BenchmarkColdLoad$' -benchmem -benchtime=5x -count=5
mise exec -- go test -tags test_dep -run '^$' -bench '^BenchmarkRows$' -benchmem -benchtime=100000x -count=5
mise exec -- go test -tags test_dep -run '^TestRepeatedAndConcurrentCalls$' -count=5 -v
```

| Sample | Cold compile/instantiate/close (ns/op) | Go B/op | Go allocs/op | Rows (ns/op) | Go B/op | Go allocs/op |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| 1 | 67,639,233 | 21,625,740 | 32,991 | 15,792 | 1,442 | 28 |
| 2 | 83,999,283 | 21,624,760 | 32,990 | 17,130 | 1,443 | 28 |
| 3 | 130,485,142 | 21,624,689 | 32,989 | 17,557 | 1,438 | 28 |
| 4 | 120,552,000 | 21,624,840 | 32,991 | 15,672 | 1,437 | 28 |
| 5 | 91,333,683 | 21,624,744 | 32,990 | 14,685 | 1,437 | 28 |

ColdLoad excludes reading the bytes, with five operations per row. Rows explores once
before timing, then cycles the 283 state IDs over 100,000 calls per row, including protobuf
encoding/decoding, allocation, transfer, guest call, copy and release. It is not a full checker
benchmark. Every Rows sample began and ended at **18 pages**, or **1,179,648 bytes** of guest
linear memory (65,536 bytes/page), with sink-bytes 17,705,239. All five repeated/concurrent
controls passed their stable-pages-after-warmup assertion. Guest pages are not RSS or host
memory; B/op and allocs/op are Go allocations, not guest allocations. Every row is retained,
without selecting the fastest result. Raw logs are `fn2107-measure-final/go-*`.

### Final feedback loop and verification

Exactly one warm `make model-time` ran after the source checkpoint, exiting 0 without retry:
gate total 4.279 s (then a 30 s soft target) and checker edit/test inner loop 6.569 s (10 s),
with every step under 2 s (`.flow/tmp/fn2107-model-time.log` keeps the step table).
fn-210.22's 3.038 s inner / 3.629 s gate observation was retained without retry. The current
receipts, after the gate grew, are under [Front end and back ends](../DESIGN.md#front-end-and-back-ends-fn-238).

The pre-edit release workspace baseline passed. Final `make model-check` and
`cargo test --workspace --release --message-format=json` both exited 0, covering the real
fixture/refusal suites, capability controls, protobuf freshness, formatting, all-target
Clippy, docs, stock WASM builds and uncached Go-host parity. The four fixture-owning crates'
active build-script outputs all point to this root checkout, including Cargo's separate
host/target release variants. The shared registry selects exactly 23 Models and the final
sample/control names. Logs are `fn2107-baseline-release.log`, `fn2107-model-check.log`,
`fn2107-final-release.jsonl`, `.stderr` and `fn2107-final-registry-proof.log` under `.flow/tmp`.

## Historical measurements

Superseded above; raw output is under `.flow/tmp/fn210-9-*`. fn-210.9's 2,872,544-byte guest
took a median 27.874 ms per cold load and 14,003 ns/op per Rows call; `make model2-time` took
5.069 s gate and 1.210 s inner loop. The fn-210.16 direct-export guest was 3,370,765 bytes.

