# Gomad

Gomad runs Go tests across seeded executions and keeps failures as inspectable,
replayable artifacts. This guide follows the current WASM path from installation
to a retained failure, then shows how to select a test from a consumer checkout.

The cooperative WASM profile builds ordinary Go source with a pinned runtime
overlay. A Rust helper executes the module with controlled scheduling, virtual
time and modeled WASI operations. Each execution starts with fresh guest state.
The shared Gomad Runner owns campaigns, evidence storage, inspection and replay.

WASM remains experimental. Selecting a package and test is supported CLI syntax;
it does not guarantee that every Go or Temporal functional test fits the modeled
boundary or reproduces. Qualification applies only to the candidate, profile,
host, workload and executions named in retained evidence. The upstream sync
(fn-189) and helper work (fn-187) qualified on candidate `e50c6476fb`.
Read the [milestones](../../MILESTONES.md) and
[WASM qualification contract](internal/backends/wasm/QUALIFICATION.md) before
making a support claim.

## Install the WASM launcher and helper

You need this repository, Make, an existing stock **Go 1.27.1** distribution, and
**Rust 1.99.1** to build the helper. Cargo uses the checked lock file and Wasmtime
47.0.3. The module's [mise configuration](mise.toml) selects Rust with rustfmt
and Clippy; it does not install Go.

The cooperative compiler must live outside the Go module cache. Installation
requires regular executable files for the compiler and helper and rejects
symbolic links. Rust is needed to build or update the helper. Retained replay
needs the compatible helper, but does not require Rust or the compiler.
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
mise exec -- make wasm-helper
mkdir -p "$GOMAD_OPERATOR/bin"
env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off \
  "$GOMAD_STOCK_GO" build -trimpath \
  -o "$GOMAD_OPERATOR/bin/gomad" ./cmd/gomad
"$GOMAD_OPERATOR/bin/gomad" install \
  --compiler "$GOMAD_STOCK_GO" \
  --helper "$GOMAD_CHECKOUT/tools/gomad/internal/backends/wasm/wasmhost/target/release/gomad-wasmhost" \
  --runtime-root "$GOMAD_CHECKOUT/tools/gomad/internal/model/runtimesource" \
  --profile gomad.wasi-cooperative/v1 \
  --cache "$GOMAD_OPERATOR/preparation" \
  --compile-cache "$GOMAD_OPERATOR/engine" \
  --output "$GOMAD_OPERATOR/bin/gomad-wasm.json"
export GOMAD_WASM_INSTALLATION="$GOMAD_OPERATOR/bin/gomad-wasm.json"
"$GOMAD_OPERATOR/bin/gomad" discover
```

Without mise, install the crate's pinned Rust toolchain through rustup and run
`make -C tools/gomad wasm-helper` from the repository root. The target invokes
`cargo +1.99.1 build --release --locked` through the backend Makefile.

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
test. WASM uses `discover` here. Native `doctor` and `analyze` are unavailable
through this provider. Native execution is hibernating under fn-128/fn-149;
the patched native host toolchain is a separate workflow.

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
models belong to the installation's immutable `wasi.Config`. Native
`--io-ro-mount` flags cannot supply WASM inputs.

## Consumer example with Temporal

The current Temporal integration base is upstream commit
`48a1e5ff025e55b3d4d0f44247423d4c279ff636`, retained by merge
`0077477965a5b766b95a667247a9f562c100941b`. See the
[integration report](../../.flow/artifacts/fn189-upstream-sync-2026-10-17.md)
for conflict resolutions and historical evidence; current qualification remains open.

Temporal's integration owns its package choices, SQLite inputs, test arguments
and provider tags. Gomad currently contains Temporal fixtures, build-tag helpers
and a soft-assertion oracle. Complete application decoupling has not been
implemented. These are explicit integration inputs, not automatic package policy.

The small `./tests/gomadfunctional` package contains `TestFrontendSystemInfo`,
which starts testcore with in-memory SQLite. A Temporal installation needs
captured schema bytes and larger capacities than the first fixture. The command
below requires a complete `wasi.Config` JSON file prepared by the Temporal
integration. Set `GOMAD_TEMPORAL_CONFIG` to that file's absolute path. The current
CLI has no generic mount-capture/setup command, so arbitrary Temporal tests are
not yet a one-command workflow.

For this selection the configuration must declare the cooperative profile,
guest `WorkingDirectory=/workspace/tests/gomadfunctional`, matching `PWD`,
writable `/workspace`, `/tmp` and guest working directories, and immutable
captured `schema/sqlite` bytes at `/workspace/schema/sqlite`. It must also supply
appropriate file, descriptor, call and modeled network limits. The existing
[Temporal guest configuration fixture](internal/backends/wasm/wasi/measure_integration_test.go)
and [one-box inspection fixture](internal/backends/wasm/backend/temporal_inspection_integration_test.go)
show how current integration code captures inputs and declares those limits.
They are implementation references, not a consumer setup command or evidence
that another selection works. The [operator guide](internal/backends/wasm/CLI.md#build-and-pin-an-installation)
defines the complete configuration input contract.

Given that prepared configuration, create a separate installation and select
the test. These memory/fuel capacities follow Temporal's guest integration;
they are limits, not measured consumption or a current-candidate pass.
Standalone execution takes explicit environment values from `--env`;
configuration environment entries alone do not supply them.

```sh
export GOMAD_TEMPORAL_CONFIG=/absolute/consumer-prepared/temporal-config.json
"$GOMAD_OPERATOR/bin/gomad" install \
  --compiler "$GOMAD_STOCK_GO" \
  --helper "$GOMAD_CHECKOUT/tools/gomad/internal/backends/wasm/wasmhost/target/release/gomad-wasmhost" \
  --runtime-root "$GOMAD_CHECKOUT/tools/gomad/internal/model/runtimesource" \
  --profile gomad.wasi-cooperative/v1 \
  --config "$GOMAD_TEMPORAL_CONFIG" \
  --memory-bytes=4294967296 --fuel=1125899906842624 \
  --cache "$GOMAD_OPERATOR/temporal-preparation" \
  --compile-cache "$GOMAD_OPERATOR/temporal-engine" \
  --output "$GOMAD_OPERATOR/bin/gomad-temporal.json"
cd "$GOMAD_CHECKOUT"
"$GOMAD_OPERATOR/bin/gomad" \
  --installation "$GOMAD_OPERATOR/bin/gomad-temporal.json" explore \
  --working-dir "$GOMAD_CHECKOUT" \
  --build-tag=disable_grpc_modules --build-tag=gomad_wasm_slim \
  --build-tag=sqlite3_dotlk --build-tag=test_dep \
  --env=TEMPORAL_ROOT=/workspace --env=TMPDIR=/tmp \
  --env=TEMPORAL_TEST_LOG_LEVEL=warn \
  --choices --choice-bytes=64MiB \
  --output-limit=16MiB --seeds=0 --parallel=1 \
  --execution-timeout=15m --overall-timeout=30m \
  --artifacts "$GOMAD_OPERATOR/temporal-artifacts" \
  go-test ./tests/gomadfunctional -- '-test.run=^TestFrontendSystemInfo$' \
  -test.v -test.count=1 -test.timeout=72h \
  -persistenceType=sql -persistenceDriver=sqlite
```

Host preparation uses the checkout root; guest execution uses
`/workspace/tests/gomadfunctional`. Captured `schema/sqlite` lives at
`/workspace/schema/sqlite`, and `TEMPORAL_ROOT=/workspace` lets testcore find it.
Gomad supplies guest `PWD` and UTC timezone. No live mount is exposed.

For `./tests`, change the package and `-test.run`, set the configuration's guest
working directory to `/workspace/tests`, regenerate it and install a new
manifest. For example,
`^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$` selects an existing
subtest. Other tests can need additional captured files, different capacities
or model support. `gomad_wasm_slim` omits cloud archivers, ringpop and
MySQL/PostgreSQL test drivers. Selecting excluded providers fails explicitly.

Start with one seed and one heavy preparation at a time. Whole Temporal modules
can exhaust memory, fuel, transcript or choice limits. The public standalone
choice limit is 64 MiB. Standalone CLI execution uses its 64 MiB transcript
default even when the installation configuration declares more;
`--io-transcript-bytes` is a native-only flag. Overflow cannot establish exact
replay. Temporal soft assertions fail Gomad's oracle even after a test prints
`PASS`. The explicit
WASM `go-test` option `--soft-assertion-report-only` reports them without failing
otherwise successful execution for tests that intentionally trigger them.
It is part of target identity and replay/resume restore it.

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
and crash/restart boundaries. The [WASM operator guide](internal/backends/wasm/CLI.md#explore-a-typed-cluster-scenario)
provides complete configuration and checked durable-reopen and partition/heal
examples. Cluster replay checks lifecycle, network/storage and per-incarnation
evidence. Crash prefixes are not complete global tapes. Cluster-global forced
choice search, native simulation exploration and cluster minimization are
unsupported.

The shared [minimizer](internal/backends/wasm/CLI.md#minimize-a-compatible-retained-failure)
accepts standalone cooperative target failures with complete exact choice
evidence. It reduces the required forced prefix while preserving module, seed,
inputs and failure identity. Every accepted candidate must pass exact replay.

Dormant instrumentation in a recorded cooperative module can support
[stop-and-inspect replay](internal/backends/wasm/CLI.md#inspect-a-retained-cooperative-failure)
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
replayable application failures. Native qualification remains deferred. The
[qualification ledger](internal/backends/wasm/QUALIFICATION.md) defines required
gates and unresolved cells. This guide supplies no current qualification result,
performance gain or numeric determinism bound.

### Deterministic I/O

The WASM namespace uses captured files, writable directories and explicit
limits. Consult [WASM capabilities](internal/backends/wasm/CLI.md#capabilities-and-limits)
and [I/O ownership](ARCHITECTURE.md#where-determinism-comes-from) for model scope.

### Network streams, partitions and replay

The [network contract](SPEC.md#simulationnetwork-network-model) and
[architecture](ARCHITECTURE.md#clusters-and-scenarios) define stream,
partition, lifecycle, fidelity and replay semantics.

### Compatibility-pack development

Native pack maintenance lives in the [maintainer CLI guide](CLI.md#extend-support-through-compatibility-packs).
Native approvals do not admit arbitrary WASM imports or host access.

### Go caller migrations

Runner operations and `backends.Provider` own campaign preparation and
execution. See [campaign intent and construction](ARCHITECTURE.md#one-execution-end-to-end)
for the current Go interface boundary; Git history retains prior migrations.

## Maintainer checks

Rust and Go consume the same helper transcript corpus as their wire/evidence
contract. Regeneration compares repeated executions; a second pass must leave
the fixtures byte-identical. Keep unit tests for invariants a corpus cannot
express. Measurement producers run through the explicit `wasmmeasure` command,
not test targets. The [WASM implementation guide](internal/backends/wasm/README.md)
documents regeneration commands, retained invariants and measurement admission.

Shared wire codecs are generated from the schemas under `internal/model` by
`make generate` (`internal/model/cmd/protocolgen`, stock Go, no build tag);
`make validate` reports each stale output and fails. The volume and network
models are owned source with no generator. The native backend tree is frozen:
it is neither executed nor qualified, and its generators and validators run
only under its own selection, `GOFLAGS=-tags=gomad_native make generate-native
validate-native`. See [ARCHITECTURE.md](ARCHITECTURE.md#clusters-and-scenarios).

## Further reading

- [WASM operator guide](internal/backends/wasm/CLI.md) for installed commands and flags.
- [WASM implementation](internal/backends/wasm/README.md) for helper builds, corpora and measurement ownership.
- [WASM qualification](internal/backends/wasm/QUALIFICATION.md) for retained gate requirements and limits.
- [Product specification](SPEC.md) for vocabulary and requirements.
- [Architecture](ARCHITECTURE.md) for ownership and design rationale.
- [Milestones](../../MILESTONES.md) for delivery order and open acceptance.
- [Shared/native CLI guide](CLI.md) for retained native and maintainer commands.
