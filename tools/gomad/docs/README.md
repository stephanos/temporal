# Gomad

Gomad runs Go tests across seeded executions and keeps failures as inspectable,
replayable artifacts. This guide follows the current WASM path from installation
to a retained failure, then shows how to select a test from a consumer checkout.

The cooperative WASM profile builds ordinary Go source with a pinned runtime
overlay. The linked Rust engine executes the module with controlled scheduling, virtual
time and modeled WASI operations. Each execution starts with fresh guest state.
The shared Gomad Runner owns campaigns, evidence storage, inspection and replay.

WASM remains experimental. Selecting a package and test is supported CLI syntax;
it does not guarantee that every Go or Temporal functional test fits the modeled
boundary or reproduces. Qualification applies only to the candidate, profile,
host, workload and executions named in retained evidence. The upstream sync
(fn-189) and helper work (fn-187) qualified on candidate `e50c6476fb`.
Read the [milestones](../../../MILESTONES.md) and
[WASM qualification contract](wasm/QUALIFICATION.md) before
making a support claim.

## Build and install the Rust host

You need this repository, Make, an existing stock **Go 1.27.1** distribution, and
**Rust 1.99.1** to build the host. Cargo uses the checked lock file and Wasmtime
47.0.3. The module's [mise configuration](../mise.toml) selects Rust with rustfmt
and Clippy; it does not install Go.

The cooperative compiler must live outside the Go module cache. Installation
requires regular executable files and rejects symbolic links. The engine is
linked into `gomad`; optional `--helper` admits only the same host executable
or a byte-identical copy. Rust is needed to build or update the host. Retained
replay needs the compatible host, but does not require Rust or the compiler.
Preparing new Go targets requires the pinned compiler.

Choose absolute paths for your checkout, compiler and private operator directory.
Replace the three example paths first. `GOMAD_CHECKOUT` names the Temporal
repository root containing `tools/gomad`.

```sh
export GOMAD_CHECKOUT=/absolute/path/to/temporal
export GOMAD_STOCK_GO=/absolute/stock-go1.27.1/bin/go
export GOMAD_OPERATOR=/absolute/private/gomad-operator
cd "$GOMAD_CHECKOUT/tools/gomad"
mise install rust
mise exec -- rustc --version
mise exec -- make build
mkdir -p "$GOMAD_OPERATOR/bin"
cp "$GOMAD_CHECKOUT/tools/gomad/rust/target/release/gomad" "$GOMAD_OPERATOR/bin/gomad"
"$GOMAD_OPERATOR/bin/gomad" install \
  --compiler "$GOMAD_STOCK_GO" \
  --runtime-root "$GOMAD_CHECKOUT/tools/gomad/internal/model/runtimesource" \
  --profile gomad.wasi-cooperative/v1 \
  --cache "$GOMAD_OPERATOR/preparation" \
  --compile-cache "$GOMAD_OPERATOR/engine" \
  --output "$GOMAD_OPERATOR/bin/gomad-wasm.json"
export GOMAD_WASM_INSTALLATION="$GOMAD_OPERATOR/bin/gomad-wasm.json"
"$GOMAD_OPERATOR/bin/gomad" discover
```

`make build` invokes `cargo +1.99.1 build --release --locked` from the single
workspace in `tools/gomad/rust`. It publishes an independent mode-0755 executable at `tools/gomad/rust/target/release/gomad`,
including when `CARGO_TARGET_DIR` selects another build cache. The standalone
reference transport is built and published only by the explicit
`make -C tools/gomad wasm-test-helper` target with `reference-helper` enabled.

`install` pins existing tools and writes a new mode-0600 manifest. It refuses to
overwrite one and does not download a compiler. Changed helper/runtime/model
inputs need matching installations and qualification. The preparation cache
stores verified modules. The optional engine cache shares compiled code; it
does not reuse guest heaps or model state.

Resolution uses global `--installation FILE`, then `GOMAD_WASM_INSTALLATION`,
then `gomad-wasm.json` beside the launcher. Put an explicit global flag before
the command:

```sh
"$GOMAD_OPERATOR/bin/gomad" \
  --installation "$GOMAD_OPERATOR/bin/gomad-wasm.json" discover
```

An invalid selected manifest fails without fallback. Discovery reports its
identities and configuration and checks helper bytes; it does not execute a
test. WASM uses `discover` here. The native backend and its commands were
removed by fn-223; fn-128 and fn-149 closed as won't-do on 2026-10-19.

## Keep and replay a first failure

Start with Gomad's intentional failure fixture, a WASM-owned test module under
`internal/backends/wasm/testdata/io_failure_module`:

```sh
cd "$GOMAD_CHECKOUT"
"$GOMAD_OPERATOR/bin/gomad" explore \
  --choices --diagnostics --seeds=7 --parallel=1 \
  --working-dir "$GOMAD_CHECKOUT/tools/gomad/internal/backends/wasm/testdata/io_failure_module" \
  --build-tag=gomad_fixture --build-tag=test_dep \
  --execution-timeout=30s --overall-timeout=5m \
  --artifacts "$GOMAD_OPERATOR/canary-artifacts" \
  go-test ./io_failure -- '-test.run=^TestDeterministicIOFailure$' -test.v
```

The expected application failure returns status 1. Take the actual artifact path
from Gomad's output, then inspect and replay that exact directory:

```sh
export GOMAD_ARTIFACT=/absolute/path/reported/by/gomad
"$GOMAD_OPERATOR/bin/gomad" inspect --json "$GOMAD_ARTIFACT"
"$GOMAD_OPERATOR/bin/gomad" inspect --choices "$GOMAD_ARTIFACT"
"$GOMAD_OPERATOR/bin/gomad" replay "$GOMAD_ARTIFACT"
```

A matching failure replay also returns 1. Look for `reproduced=true`; exit 1
alone cannot distinguish reproduction from divergence. Matching success replay
returns 0. Invalid input returns 2 and Runner/infrastructure failure returns 3.
Use `--json` on exploration for the shared campaign event stream.

## Select an ordinary Go test

`go-test PACKAGE -- TEST-ARGS...` selects a package and passes the arguments
after `--` to its compiled test binary. Use `-test.run`, rather than the Go
driver's `-run`. Put Gomad flags before `go-test`. For a package whose runtime
needs fit the default installation:

```sh
"$GOMAD_OPERATOR/bin/gomad" explore \
  --choices --seeds=0-9 --parallel=1 \
  --working-dir /absolute/path/to/module --build-tag=test_dep \
  --execution-timeout=1m --overall-timeout=10m \
  --artifacts "$GOMAD_OPERATOR/test-artifacts" \
  go-test ./path/to/package -- '-test.run=^TestSomethingConcurrent$' -test.v
```

The package path is relative to the host module directory selected by
`--working-dir`. An explicit directory must be absolute and contain `go.mod`;
omitting the flag selects the current directory. `--build-tag` is repeatable.
Gomad does not infer consumer tags such as Temporal's `test_dep`. Subtest
selectors use the usual slash-separated `testing` regular expressions.

Preparation uses `wasip1/wasm`, cgo disabled, explicit tags and the cooperative
runtime overlay. It rejects unsupported imports and preparation options. A
successful build still needs execution evidence for its model and capacity
requirements. Guest files, working directory, entropy and network/storage
models belong to the installation's immutable `wasi.Config`.

## Consumer example with Temporal

The current Temporal integration base is upstream commit
`48a1e5ff025e55b3d4d0f44247423d4c279ff636`, retained by merge
`0077477965a5b766b95a667247a9f562c100941b`. Historical conflict resolutions remain
in Git history and the fn-189 Flow records; current qualification remains open.

Temporal's integration owns its package choices, SQLite inputs, test arguments
and provider tags. Gomad currently contains Temporal fixtures, build-tag helpers
and a soft-assertion oracle. Complete application decoupling has not been
implemented. These are explicit integration inputs, not automatic package policy.

The small `./tests/gomadfunctional` package contains `TestFrontendSystemInfo`,
which starts testcore with in-memory SQLite. A Temporal installation needs
captured schema bytes and larger capacities than the first fixture. One `install`
command derives the cooperative configuration from the checkout and package,
captures immutable SQLite schema bytes, and pins the slim guest build tags.
No configuration JSON or `GOMAD_TEMPORAL_CONFIG` is needed. The shared derivation
also supplies the guest working directory, matching `PWD`, writable directories,
and bounded file, descriptor, call and modeled network capacities.
The [operator guide](wasm/CLI.md#prepare-a-temporal-test-package)
records the defaults and input contract.

Create a separate installation and select the named test. These memory/fuel
capacities follow Temporal's guest integration; they are limits, not measured consumption.
Setup refuses inherited `GOMAD_*` variables; `env -i` supplies only its named inputs.
Standalone execution takes explicit environment values from `--env`;
configuration environment entries alone do not supply them.

```sh
env -i "$GOMAD_OPERATOR/bin/gomad" install \
  --temporal-checkout "$GOMAD_CHECKOUT" \
  --temporal-package ./tests/gomadfunctional \
  --compiler "$GOMAD_STOCK_GO" \
  --runtime-root "$GOMAD_CHECKOUT/tools/gomad/internal/model/runtimesource" \
  --cache "$GOMAD_OPERATOR/temporal-preparation" \
  --compile-cache "$GOMAD_OPERATOR/temporal-engine" \
  --output "$GOMAD_OPERATOR/bin/gomad-temporal.json"
cd "$GOMAD_CHECKOUT"
"$GOMAD_OPERATOR/bin/gomad" \
  --installation "$GOMAD_OPERATOR/bin/gomad-temporal.json" explore \
  --working-dir "$GOMAD_CHECKOUT" \
  --env=TEMPORAL_ROOT=/workspace --env=TMPDIR=/tmp \
  --env=TEMPORAL_TEST_LOG_LEVEL=warn \
  --choices --choice-bytes=64MiB \
  --output-limit=16MiB --seeds=0 --parallel=1 \
  --keep-successes=all --success-limit=1 --success-bytes=4GiB \
  --execution-timeout=15m --overall-timeout=30m \
  --artifacts "$GOMAD_OPERATOR/temporal-artifacts" \
  go-test ./tests/gomadfunctional -- '-test.run=^TestFrontendSystemInfo$' \
  -test.v -test.count=1 -test.timeout=72h \
  -persistenceType=sql -persistenceDriver=sqlite
```

Replay the successful artifact path reported by `explore` with
`gomad --installation "$GOMAD_OPERATOR/bin/gomad-temporal.json" replay ARTIFACT`.

Host preparation uses the checkout root; guest execution uses
`/workspace/tests/gomadfunctional`. Captured `schema/sqlite` lives at
`/workspace/schema/sqlite`, and `TEMPORAL_ROOT=/workspace` lets testcore find it.
Gomad supplies guest `PWD` and UTC timezone. No live mount is exposed.

This workflow supports only the named `TestFrontendSystemInfo` selection;
current-candidate one-box qualification is owned separately by fn-222.
Preparing another package does not establish its support. Other tests can need
additional captured files, different capacities or model support.
`gomad_wasm_slim` omits cloud archivers and MySQL/PostgreSQL test drivers;
ringpop membership stays linked. Selecting excluded providers fails explicitly.

Start with one seed and one heavy preparation at a time. Whole Temporal modules
can exhaust memory, fuel, transcript or choice limits. The public standalone
choice limit is 64 MiB. Standalone CLI execution uses its 64 MiB transcript
default even when the installation configuration declares more.
Overflow cannot establish exact
replay. Temporal soft assertions fail Gomad's oracle even after a test prints
`PASS`. The explicit
WASM `go-test` option `--soft-assertion-report-only` reports them without failing
otherwise successful execution for tests that intentionally trigger them.
It is part of target identity and replay/resume restore it.

## Run existing commands through Gomad

The Rust launcher can wrap an existing command without changing its Makefile or
script. From the Temporal module root, with stock Go 1.27.1 on `PATH`:

```sh
gomad -- go test ./tests/gomadfunctional -run '^TestFrontendSystemInfo$' -v
```

The checked-in `gomad.toml` selects the cooperative runtime sources and Temporal
installation derivation. No `guest` entries are configured, so `go run` and
`go build` remain host tools. An explicit `--installation FILE` before `--`
overrides auto-derivation. Nothing downloads or switches compilers.

The mandatory `--` separates Gomad options from child arguments. Use
`gomad -- make`, never `gomad make` or `gomad go`. The wrapper returns the child's
actual status. It inserts a private `go` shim at the front of `PATH`; absolute
compiler paths and programs that replace or shadow that `PATH` bypass it.
Go retains package selection, test flags, output parsing and package parallelism.
Guest stdout reaches Go unchanged; Gomad diagnostics use stderr.

`go test` compiles for cooperative wasip1/wasm and uses `run-guest` through Go's
`-exec`. A caller's `-exec` is preserved with the guest build environment; an
external launcher supplies no Gomad execution-evidence claim. Other Go
subcommands and unmatched `go run`/`go build` pass through to the pinned compiler
with host settings. Guest builds refuse race detection, fuzzing, coverage,
MSan/ASan, cgo, excluded tags and incompatible GOOS/GOARCH before compilation;
see the [complete contract](wasm/CLI.md#wrap-existing-go-commands).

For the unchanged Temporal Make target, restrict all three recipe package
variables to the registered selection:

```sh
gomad -- make functional-test \
  TEST_RACE_FLAG=off TEST_SHUFFLE_FLAG=off \
  PERSISTENCE_TYPE=sql PERSISTENCE_DRIVER=sqlite \
  FUNCTIONAL_TEST_ROOT=./tests/gomadfunctional \
  FUNCTIONAL_TEST_NDC_ROOT=./tests/gomadfunctional \
  FUNCTIONAL_TEST_XDC_ROOT=./tests/gomadfunctional \
  TEST_ARGS='-run=^TestFrontendSystemInfo$$ -v -count=1' TEST_TIMEOUT=5m
```

The doubled `$$` survives Make variable expansion as the single regex anchor `$`.
This runs the same supported test three times sequentially. It establishes no
NDC/XDC package support or suite breadth. Default `TEST_RACE_FLAG=on` is refused.
The Makefile pipes Go through `tee` and checks `test.log` afterward; Make's status
can differ from Go's. Use a fresh log and inspect the actual refusal/output.
The [Make acceptance harness](../testdata/parity/interception-make-plan.md) retains
these separate observations and checks host tools separately.

Intercepted campaigns retain `origin: intercepted-go`, original argv and distinct
wrapper, driver and launch identities. Explicit replay and inspection accept
authenticated retained evidence; compatible failures can be minimized. Qualification
refuses intercepted evidence, including derived minimized artifacts. This convenience
path adds no guest capabilities or support beyond separately proven selections;
[fn-221](../../../.flow/specs/fn-221-gomad-functional-suite-triage-sweep-on.md) and
[fn-222](../../../.flow/specs/fn-222-gomad-qualify-the-one-box-cell-and.md) own per-test
support and qualification.

## Seeds, campaigns and artifacts

`--seeds=0-9` selects ten seeds, `--seeds=0,7,42` selects three, and `--count=100`
selects 0 through 99. Zero is valid. Seeds can reach different runnable and
select choices, or the same execution. Changed source, tags, inputs or runtime
change execution identity, so a seed is meaningful within that identity.

The cooperative runtime uses one P and disables asynchronous preemption. With
the default frozen clock, an idle guest advances to its next deadline.
`-test.timeout` bounds virtual time; `--execution-timeout` bounds host wall time.
A busy loop that never yields can stall virtual time and reach fuel/watchdog
limits. Neither seed sampling nor bounded choice exploration exhausts schedules.

Gomad prepares one immutable target per campaign and commits outcomes in logical
selection order. Failure artifacts retain the module, provenance, seed,
arguments, captured inputs, limits, output and modeled evidence. `--choices`
records bounded runtime evidence. Exact standalone replay forces and validates
the complete recorded tape, then compares outcome and evidence. Seed-only or
stock observed replay makes a narrower claim; inspect the reported replay mode.

Success artifacts are discarded by default. Add
`--keep-successes=all --success-limit=10 --success-bytes=4GiB` to exploration
to retain them for replay. The bounds include target bytes and fail visibly
when exhausted. Copy complete reported artifacts when sharing reproductions.

Replay reads retained bytes without rebuilding or reopening captured input
originals. It requires compatible helper, model/runtime, engine, host and
configuration identities. Current readers reject unsupported historical formats.
Inspection and recovery read retained evidence without an installation or guest.
For an interrupted campaign, use its reported campaign path:

```sh
"$GOMAD_OPERATOR/bin/gomad" inspect --json /absolute/interrupted/campaign
"$GOMAD_OPERATOR/bin/gomad" recover --json /absolute/interrupted/campaign
"$GOMAD_OPERATOR/bin/gomad" resume --json /absolute/interrupted/campaign
```

Recovery validates committed evidence and repairs only recognized torn-tail
state. For Temporal replay or resume, select the Temporal manifest with global
`--installation` as in its exploration command. Resume needs a compatible
installation and executes remaining work from retained preparation. An
interrupted campaign is not a completed pass.

## Scenarios, inspection and minimization

Typed `--scenario FILE` seed campaigns declare nodes, links, durable volumes
and crash/restart boundaries. The [WASM operator guide](wasm/CLI.md#explore-a-typed-cluster-scenario)
provides complete configuration and checked durable-reopen and partition/heal
examples. Cluster replay checks lifecycle, network/storage and per-incarnation
evidence. Crash prefixes are not complete global tapes. Cluster-global forced
choice search and cluster minimization are
unsupported.

The shared [minimizer](wasm/CLI.md#minimize-a-compatible-retained-failure)
accepts standalone cooperative target failures with complete exact choice
evidence. It reduces the required forced prefix while preserving module, seed,
inputs and failure identity. Every accepted candidate must pass exact replay.

Dormant instrumentation in a recorded cooperative module can support
[stop-and-inspect replay](wasm/CLI.md#inspect-a-retained-cooperative-failure)
with stacks, heap profiles and registered dumps/probes. It ends at a validated
prefix and does not claim full reproduction. Standalone Go-owned network
recordings lack this metadata; the Temporal one-box inspection cell uses the
Rust cluster scenario path.

## Contract

WASM admits modeled imports and captured inputs. Unsupported host I/O,
subprocesses, cgo and unmodeled dependencies fail closed. Stock exploratory WASI
and cooperative WASI advertise different capabilities. Stock observations do
not establish forced scheduling replay. Neither profile establishes native race
detection, weak-memory coverage, asynchronous preemption, cross-host replay,
checkpoint safety or strict World-time qualification.

Finite repetitions and exact replay apply to retained executions. They establish
neither arbitrary Temporal support nor exhaustive schedule exploration.
Capacity/fuel, watchdog and infrastructure failures remain separate from
replayable application failures. Native retirement supplies no qualification. The
[qualification ledger](wasm/QUALIFICATION.md) defines required
gates and unresolved cells. This guide supplies no current qualification result,
performance gain or numeric determinism bound.

The [WASM determinism inventory](wasm/README.md#determinism-contract)
classifies each reviewed influence and names its enforcement tests. Historical
native escape findings have the following wasip1 disposition; the removed
backend's qualification does not establish these rules.

| Historical native finding | Applies under wasip1? | Cooperative wasip1 rule |
| --- | --- | --- |
| GC `LastGC` and `PauseEnd` stamps | No | Read the transcribed virtual wall clock. |
| FIPS `monoTime` | No | Reaches the virtual clock through `time_now`. |
| Tracer clock snapshot | No | Reads the virtual clock. |
| `Gettimeofday` | No | WASI libc reaches the modeled `clock_time_get`; host syscall access is unavailable. |
| `cputicks` | No | Returns virtual `nanotime` on wasm. |
| GC pacer `enlistWorker` | No | The single-P guard avoids its random P pick at startup; any later draw uses seeded `cheaprand`. |

"No" means the escape is controlled under wasip1 by the inventory rows and
tests above. It does not mean every host influence is eliminated. The WASM
inventory currently accepts no escapes; see
[known determinism escapes](wasm/README.md#known-determinism-escapes).

### Deterministic I/O

The WASM namespace uses captured files, writable directories and explicit
limits. Consult [WASM capabilities](wasm/CLI.md#capabilities-and-limits)
and [I/O ownership](ARCHITECTURE.md#where-determinism-comes-from) for model scope.

### Network streams, partitions and replay

The [network contract](SPEC.md#simulationnetwork-network-model) and
[architecture](ARCHITECTURE.md#clusters-and-scenarios) define stream,
partition, lifecycle, fidelity and replay semantics.

## Maintainer checks

The Rust host, linked engine, fixture-only execution and evidence/store gates
are documented in [rust/README.md](rust/README.md). `make build` produces the
ordinary Rust host. `make test-wasm` executes the Rust suites and retained Go
reference models. `make qualify` requires a frozen driver configuration and
its SHA-256; source checks never replace the final runtime/workload/replay cycle.

Rust and Go consume the same helper transcript corpus as their wire/evidence
contract. Regeneration compares repeated executions; a second pass must leave
the fixtures byte-identical. Keep unit tests for invariants a corpus cannot
express. Qualification and measurement use the Rust driver with explicit authenticated
inputs; the retired Go host producers are absent. The [WASM implementation guide](wasm/README.md)
documents regeneration commands, retained invariants and measurement admission.

Shared wire codecs are generated from the schemas under `internal/model` by
`make generate` (`internal/model/cmd/protocolgen`, stock Go, no build tag);
`make validate` reports each stale output and fails. The volume and network
models are owned source with no generator. The native backend and its
generators were removed. See [ARCHITECTURE.md](ARCHITECTURE.md#clusters-and-scenarios).

Final changed-candidate runtime, mandatory workload and retained-replay
qualification, including the complete behavioral reach of
`make -C tools/gomad test-wasm`, remain pending under fn-232's original acceptance.
fn-246 executes and reconciles them after fn-242's Rust driver cutover and the full
fn-240 extraction. Native deletion source checks supply no final qualification pass.

## Further reading

- [WASM operator guide](wasm/CLI.md) for installed commands and flags.
- [WASM implementation](wasm/README.md) for helper builds, corpora and measurement ownership.
- [WASM qualification](wasm/QUALIFICATION.md) for retained gate requirements and limits.
- [Product specification](SPEC.md) for vocabulary and requirements.
- [Architecture](ARCHITECTURE.md) for ownership and design rationale.
- [Milestones](../../../MILESTONES.md) for delivery order and open acceptance.
- [Maintainer guide](CLI.md) for shared generated contracts and qualification ownership.

## Retained Go oracles and generated codecs

`make -C tools/gomad generate` uses the Go protocol generator to write the Go
oracle codecs, guest overlay codecs, and the Rust choice and World codecs
consumed by the linked helper library. `make -C tools/gomad validate` refuses
stale outputs and checks isolated, byte-identical second generation. It compares
the Go-produced helper transcript, replay-plan, namespace, volume, network, clock
and World corpora with their Rust consumers. Clock, network and World corpus
drivers execute inside the helper library against the modules linked by the CLI;
namespace, volume and replay-plan corpus tests use that library's default build.
The Go producers and linked library suite run twice. Namespace transport tests
require the test conformance helper and fail if it is unavailable.

Archcheck inventories only retained Go models, generators, conformance oracles,
runtime overlays, filesystem support and architecture checks. Default packages
outside that set fail as `retired-host-source`, and a default import of a retired
Go host package fails as `retired-host-import`. The Go CLI, runner and host
qualification producers are retired; the Rust `gomad` binary is the only host.
The frozen fn-232 Go baseline and binaries remain independent reference custody
until actual all-verb parity and final R12 qualification permit retirement.
