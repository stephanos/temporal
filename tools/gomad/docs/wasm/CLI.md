# WASM operator entrypoint

The host implementation now lives in the single [Rust workspace](../rust/README.md).
The Go CLI, runner and host qualification producers described in older sections
are retired source; retained Go code owns models, protocol generation, conformance
oracles and the guest runtime overlay. Current build and reference checks are
`make -C tools/gomad build`, `validate` and `test-wasm`. Final qualification uses
the [Rust driver](../driver/README.md) with frozen authenticated inputs and the unchanged
3,600-second limit. This prospective source consolidation supplies no R12/R13
acceptance; the conductor retains the frozen Go baseline until those gates pass.


The unified `gomad --installation FILE` launcher selects the provider declared
by its installation manifest. A WASM installation enables the commands below.
Campaign parsing, logical execution order, artifact layout, inspection, recovery,
resume, replay comparison, compatible minimization, reports and exit statuses
remain owned by Gomad.
The native backend and its installation commands were removed by fn-223.

The typed scenario workflow is owned by fn-181. Support advertisement describes
available operations; qualification requires retained evidence for the installed
candidate, profile, host and workload. A successful fixture run does not qualify
an arbitrary Temporal workload or a different engine/host.

## Inspect a retained cooperative failure

Instrumentation must already be dormant in the recorded binary. Replay reads the
retained module and authenticated metadata; it does not rebuild the application.
Debug output is diagnostic, written beside the artifact, and never changes its
identity. A normal `replay ARTIFACT` still requires the complete recorded tape and
reproduced outcome. Inspection instead ends with `inspected_prefix`: the pending
decision has been validated but not applied, the prefix and compulsory runtime
witness match, and application execution will not resume.

```sh
gomad --installation installation.json replay --instance=history --incarnation=1 --stop=failure \
  --dump=stacks --dump=heap --dump=temporal.shards \
  --dump=temporal.mutable_states --probe='temporal.*.snapshot' ARTIFACT
gomad --installation installation.json replay --instance=history --incarnation=2 \
  --stop=choice=120 --dump=stacks --debug-output=inspection-120 ARTIFACT
gomad --installation installation.json replay --stop=time=2s --dump=heap ARTIFACT
```

`choice=N` is a zero-based tape decision in the selected incarnation, before
decision N. Trace positions also include observations, so a trace index is not a
tape ordinal. A single-guest artifact infers its target; cluster inspection
requires both the recorded instance ID and positive incarnation number. Restarted
incarnations have separate tapes. `time=T` accepts shared virtual nanoseconds or a
duration and resolves to the first inspectable boundary at or after T. Equal-time
ties follow retained execution order. `failure` resolves to the selected
incarnation's last reachable boundary at or before its recorded failure, never a
post-trap heap. The report gives requested and resolved coordinates; neither
selector implies execution at an otherwise unobservable timestamp.

Standalone Go-owned network recordings keep ordinary recording/replay support,
but have no authenticated inspection metadata. Inspection refuses them as
unsupported; direct low-level inspection also requires helper-owned execution.
Use the Rust cluster scenario path for the Temporal one-box inspection cell.

`stacks` includes file/line frames; `heap.pprof` is a sampled heap profile, readable
with `go tool pprof -top ARTIFACT.inspection/heap.pprof`. Registered dump and probe names
are scoped to the selected incarnation. Unknown names, unavailable registration
prefixes, missing targets, out-of-range boundaries and old/incomplete metadata are
refused before execution. Changed evidence or runtime witnesses produce
instrumentation divergence, including an allocating probe that leaves stdout
unchanged. Optional `--diagnostics` cannot substitute for the witness.

The Temporal dumps read bounded completed mutation snapshots without acquiring
application locks: 32 shard write-unlock snapshots and 64 mutable-state snapshots.
Their headers identify stale-view semantics; each row has a snapshot sequence and
mutable-state rows identify the mutation seam. A sequence is not a choice index.
Live values may have changed while an application lock remains held. Identity
truncation is marked explicitly. These summaries are diagnostic copies, not a
complete cache inventory or transactionally current server state.

Use repeated `--dump`/`--probe` options to select output, and positive
`--debug-bytes`/`--dump-bytes` bounds to cap it. Per-dump panic/truncation is reported
separately; other callbacks can still complete. A callback timeout aborts and
reaps that inspection. Callbacks must never wait for a frozen application lock or
message. Fuel pressure, replay divergence and timeout are errors, not successful
inspection or predicate results.

```sh
gomad --installation installation.json replay --check=workflow.valid \
  --choice-range=10:200 --search-mode=scan --search-attempts=191 \
  --search-timeout=2m ARTIFACT
gomad --installation installation.json replay --check=workflow.once_invalid \
  --choice-range=10:200 --search-mode=bisect --search-attempts=16 \
  --search-timeout=2m ARTIFACT
```

Search evaluates a registered invariant after a fresh exact replay for every
candidate. The inclusive range is local to the selected incarnation. Ascending
scan establishes the first failure only in the range it actually examines.
Attempt and whole-search wall bounds include startup and cleanup; exhaustion
reports incomplete with checked positions and the remaining interval. Bisect
requires a retained monotonic declaration: once false, the check stays false in
the range. Its result is explicitly conditional on that declaration; observed
reversals invalidate it, and unobserved reversals cannot be excluded. Panic,
timeout and replay mismatch are inconclusive evaluations, never true/false.

## Build and pin an installation

Use an existing stock Go 1.27.1 distribution and the repository's locked Rust
helper build. For the cooperative profile, the Go distribution must be outside
the Go module cache. Keep the helper and compiler as regular executable files;
installation validation rejects symbolic links.

Rust is required to build or update the helper, not to discover an installation,
execute an already-built guest, or replay a retained artifact with that helper.
Preparing a new guest requires the pinned Go compiler; retained replay does not.
The installed helper remains required in both cases. From the repository root:

```sh
make -C tools/gomad/internal/backends/wasm helper
mkdir -p /absolute/operator/bin
cd tools/gomad
/absolute/stock-go/bin/go build -trimpath \
  -o /absolute/operator/bin/gomad ./cmd/gomad
/absolute/operator/bin/gomad install \
  --compiler /absolute/stock-go/bin/go \
  --helper /absolute/checkout/tools/gomad/rust/wasmhost/target/release/gomad-wasmhost \
  --runtime-root /absolute/checkout/tools/gomad/internal/model/runtimesource \
  --profile gomad.wasi-cooperative/v1 \
  --cache /absolute/operator/cache/preparation \
  --compile-cache /absolute/operator/cache/engine \
  --output /absolute/operator/bin/gomad-wasm.json
```

`install` pins existing tools and writes a new manifest with mode 0600. It
refuses to overwrite an existing manifest. It does not download or install a
compiler. Paths become absolute; compiler/helper digests, Go version, engine
and host, compiled environment/runtime identities, memory/fuel limits and the
complete immutable guest configuration are retained. Preparation verifies the
selected compiler and runtime sources; execution verifies the helper handshake.

The default guest namespace contains writable `/workspace` and `/tmp`, with
`PWD=/workspace`, bounded files/descriptors/output/transcript and explicit clock
policy. Install accepts `--memory-bytes` and `--fuel` to declare capacities.
`--config FILE` accepts a complete `wasi.Config` JSON value, including immutable
captured inputs produced by Gomad's `captured.CaptureReadOnlyMountInputs`.
The declared `--profile` must agree with that configuration. Runtime imports
never reopen original mount directories or access ambient host I/O.

## Prepare a Temporal test package

The `install` extension derives a cooperative installation directly from a
Temporal checkout. It shares schema capture and guest defaults with the Temporal
integration fixtures. From a clean checkout, use absolute paths for the built
launcher, pinned stock compiler, built helper, runtime sources and output:

```sh
env -i /absolute/operator/bin/gomad install \
  --temporal-checkout /absolute/checkout \
  --temporal-package ./tests/gomadfunctional \
  --compiler /absolute/stock-go/bin/go \
  --helper /absolute/checkout/tools/gomad/rust/wasmhost/target/release/gomad-wasmhost \
  --runtime-root /absolute/checkout/tools/gomad/internal/model/runtimesource \
  --cache /absolute/operator/temporal-preparation \
  --compile-cache /absolute/operator/temporal-engine \
  --output /absolute/operator/temporal.json
/absolute/operator/bin/gomad --installation /absolute/operator/temporal.json explore \
  --working-dir /absolute/checkout \
  --env=TEMPORAL_ROOT=/workspace --env=TMPDIR=/tmp --env=TEMPORAL_TEST_LOG_LEVEL=warn \
  --choices --choice-bytes=64MiB --output-limit=16MiB --seeds=0 --parallel=1 \
  --keep-successes=all --success-limit=1 --success-bytes=4GiB \
  --execution-timeout=15m --overall-timeout=30m \
  --artifacts /absolute/operator/temporal-artifacts \
  go-test ./tests/gomadfunctional -- '-test.run=^TestFrontendSystemInfo$' \
  -test.v -test.count=1 -test.timeout=72h -persistenceType=sql -persistenceDriver=sqlite
```

Replay the retained artifact path printed by `explore` with
`gomad --installation /absolute/operator/temporal.json replay ARTIFACT`.
Neither command needs hand-written JSON or `GOMAD_TEMPORAL_CONFIG`.
Standalone execution takes the explicit `--env` values above; Gomad supplies
`PWD` and UTC. The installation pins `backend.TemporalGuestBuildTags()`, so no
`--build-tag` flags are required for this selection.

Both Temporal flags are required together. The package is one checkout-relative
directory; absolute paths, parent escapes and package patterns are refused.
The command captures `schema/sqlite` immutably at `/workspace/schema/sqlite`.
The working directory is `/workspace/<package>` with matching `PWD`, and
`/workspace`, `/tmp` and that working directory are writable.
Changing schema bytes changes the installation identity. Setup refuses inherited
`GOMAD_*` variables, conflicting `--config`/`--profile`, and missing named inputs
with installation input status 2. Existing `install --config` usage remains available.

Defaults preserve the prototype's 256 MiB output, 1 GiB transcript, 2^28 calls,
4096 descriptors, 100000 files, 512 MiB filesystem and 4096 pending events.
The one-box network defaults are 128 listeners, 4096 connections, 128 MiB total
bytes, 4 MiB per connection, a 10-second stall bound and 2^20 transitions.
Memory defaults to 4 GiB and fuel to 2^50; explicit `--memory-bytes` and `--fuel`
can override those two bounds. These capacities are not consumption guarantees;
exhaustion uses the existing capacity termination. Standalone campaigns retain
their existing 64 MiB transcript limit even with a larger configuration bound.

`wasi.TemporalConfig` takes an explicit complete `TemporalPolicy` alongside the
checkout, guest directory and arguments. The command selects
`TemporalInstallation`; the prototype, inspection and one-box qualification
fixtures select `TemporalPrototype`, `TemporalInspection` and
`TemporalQualification`. The shared production derivation applies every policy
field, including network, environment, entropy, arguments, writable roots and
output/transcript bounds. Consumers do not adjust the resulting configuration.
For identical complete inputs, command and fixture derivation produce identical
final serialized configuration bytes; the equality test removes no fields.

Historical-policy tests separately preserve the prototype's nil network, the
inspection fixture's existing directory/writable roots, nil arguments, zero
entropy key, debug environment and 16 MiB output/256 MiB transcript, and the
qualification fixture's nil network and nil arguments supplied independently
by Runner. The installation policy retains the bounded network and capacities
listed above. Different explicit policies can produce different final bytes.

Support here is limited to `TestFrontendSystemInfo` in `tests/gomadfunctional`.
The setup path grants no additional guest capability or support for other tests.
One-box qualification and per-test support evidence are owned separately by
[fn-222](../../../../.flow/specs/fn-222-gomad-qualify-the-one-box-cell-and.md).

## Wrap existing Go commands

The Rust command grammar is:

```text
gomad [--installation FILE] [--seeds LIST] [--artifacts DIR] -- COMMAND [ARGS...]
gomad [--installation FILE] run-guest MODULE [GUEST-ARGS...]
```

Everything after the wrapper's first `--` belongs to the child, including names
such as `replay`. No verb forwards to a child; `gomad go` and `gomad make` are not
aliases. An empty child is a usage error, a missing executable is an input error,
and shim-directory creation failure is an infrastructure error. A normal child
exit is propagated unchanged. Go and Make can each translate a launcher's status;
read each layer's actual result.

The wrapper resolves stock Go 1.27.1 on the original `PATH`, then prepends a
private mode-0700 directory containing the Gomad `go` shim. The shim invokes that
pinned compiler. Go invoked by absolute path, relative to `GOROOT`, or through a
replaced/shadowed `PATH` is outside interception. Temporal Make prepends `.bin`;
an executable `.bin/go` would shadow the shim and must be absent. Completed private
shim state is removed; stale cleanup cannot remove another live wrapper's state.
No compiler is downloaded and guest builds disable toolchain switching.

| Invocation | Behavior |
| --- | --- |
| `go test` without caller `-exec` | Compile with the admitted cooperative overlay, full installation tags, `GOOS=wasip1`, `GOARCH=wasm`, `CGO_ENABLED=0`, `GOTOOLCHAIN=local`, and `-exec 'gomad run-guest'`. Add `-count=1` only if the caller did not set it. |
| `go test` with caller `-exec` | Apply the same guest build settings and preserve the caller's launcher. `-exec=true -count=0` compiles without running a guest. A nontrivial external launcher is invoked by Go and supplies no Gomad execution-evidence claim. |
| `go run` or `go build` matching `guest` | Execute the registered main package as a guest; `build -o PATH` writes `PATH.wasm` and a launcher script at `PATH`. |
| Other subcommands; unmatched `run`/`build` | Pass original arguments and the shim's inherited host environment to the pinned compiler. `go env GOOS` reports the host; repository generators and linters stay host tools. |

Injected driver flags precede `-args`/`--args`; everything after that boundary
remains a guest argument. Effective driver flags include `GOFLAGS`. Caller
`-run`, `-v`, `-json`, package patterns, `-p` and explicit `-count` remain owned by
Go. The shim never enables standalone Gomad `--json` on Go's stdout.

Guest invocations refuse these effective options before compilation with an
unsupported-option diagnostic on one stderr line: `-race`, `-fuzz`, `-cover`,
`-coverprofile`, `-coverpkg`, `-msan`, `-asan`, inherited `CGO_ENABLED=1`, tags
excluded by the installation, and `GOOS`/`GOARCH` other than unset or
`wasip1`/`wasm`. They are not silently dropped. Unsupported models, inputs and
capacity/fuel/watchdog failures retain their existing classifications.

### Repository configuration and installation selection

The module-root `gomad.toml` accepts exactly these keys:

```toml
runtime_root = "tools/gomad/internal/model/runtimesource"
derive = "temporal"
# guest = ["./cmd/example"]
```

The Temporal repository leaves `guest` absent. Listing a main package changes
`go run`/`go build` routing; it does not establish support. `runtime_root` must be
module-relative and canonically contained. Guest entries must be module-relative
without absolute paths or parent escapes. Unknown/duplicate keys, incorrect types,
unsupported `derive` values and escaping paths are input errors. `derive` accepts
`temporal` or `default`; the latter uses the stock cooperative namespace.

Explicit global `--installation FILE` wins over `GOMAD_WASM_INSTALLATION`; an
invalid selected installation fails without fallback or rewriting. Without either,
Gomad derives per-package installations using the same install derivation, current
compiler/runtime and linked host/helper identities, complete configuration and
captured schema bytes. It validates required inputs before cache lookup, including
schema files even when a previous entry exists. Unchanged entries can be reused;
changed inputs require matching new identities. Preparation and engine caches
share immutable code, never guest state. Auto-derivation adds no installed tool
or guest capability.

The wrapper supplies `GOMAD_SHIM_BINARY`, `GOMAD_SHIM_GO`,
`GOMAD_SHIM_INSTALLATION`, `GOMAD_SHIM_ARTIFACTS`, `GOMAD_SHIM_SEEDS` and
`GOMAD_SHIM_RUN`. Per-driver immutable invocation metadata is passed only to that
compiler process and its launchers with `GOMAD_SHIM_INVOCATION` and
`GOMAD_SHIM_INVOCATION_SHA256`. These are wrapper protocol values, not operator
configuration. Unrelated inherited `GOMAD_*` controls are not accepted.

### Launcher evidence and limits

`run-guest` authenticates the actual Go-produced module against its prepared
compiler, source, overlay, installation and invocation bindings before retaining
it. An arbitrary or unprepared WASM file is refused. Each package launcher gets
a distinct campaign under the wrapper's artifact root; Go's own package
parallelism controls overlapping launchers. Each seed runs sequentially in fresh
guest state. The first non-success stops the launcher; later selected seeds
remain unexecuted. Zero means every requested seed succeeded.

Guest stdout and stderr stream unchanged to their respective inherited channels;
Gomad diagnostics go to stderr. Standalone `run-guest --json` instead reserves
stdout for structured reporting and retains guest output in artifacts. Success
returns 0, target failure uses its supported guest status, and capacity/fuel/helper
failure uses infrastructure status 3. On a failure with a retained replayable
artifact the final stderr line is the exact explicit replay command. Otherwise
the diagnostic says no replayable artifact was retained and prints no fabricated
command. Go usually maps a nonzero launcher status to its own failure status.

Retained plans, preparation and executions bind `origin: intercepted-go`, original
argument bytes and wrapper/driver/launch identities. Replay and inspection
authenticate these bytes without the original temporary module or invocation
file. Compatible retained failures may be minimized; the origin remains
intercepted. Qualification admits only `origin: operator` and refuses intercepted
evidence. Successful interception, replay or a Go PASS is not qualification.
All capacity, output/choice limits and modeled boundaries still apply.

### Temporal Make smoke

Use the exact constrained command in the
[README](../README.md#run-existing-commands-through-gomad). It sets all three
functional package variables to `./tests/gomadfunctional` and selects only
`TestFrontendSystemInfo`: three sequential executions of one registered test,
not three independently supported packages or an NDC/XDC support claim.
The broad default-root command is routing evidence only until its individual
workloads have passed their support gates.

`functional-test` calls `go clean -testcache` and three `go test` commands; it
runs no generator or linter. The separate generator/linter passthrough probes
must therefore be retained independently. Keep the Makefile unchanged and retain
`make -n` expansion, exact variables/selector, raw outputs, authenticated campaigns
and actual statuses. Default race mode and explicit `TEST_RACE_FLAG=on` must both
refuse before any guest build. A direct shim refusal probe records the Go-level
status separately. Each Make recipe pipes through `tee -a test.log`; the final
verifier requires some `^ok` and no `^--- FAIL`. A stale passing log can mask a
refusal, so the acceptance harness requires a fresh log and does not add pipefail
or change the shell to improve the result. The wrapper returns Make's actual
status. See the [execution plan](../../testdata/parity/interception-make-plan.md)
for stage commands and evidence boundaries.

## Discovery

```sh
gomad discover
gomad --installation /absolute/operator/bin/gomad-wasm.json discover
```

Resolution uses the explicit `--installation` path, then
`GOMAD_WASM_INSTALLATION`, then `gomad-wasm.json` beside the running binary.
An invalid selected manifest fails without falling back to another installation.
Put the global `--installation` flag before the command. Discovery emits JSON
containing the selected path, resolution source, manifest digest and configuration.
It checks strict bounded JSON, current model/runtime/engine identities and the
installed helper's bytes. It does not open the compiler, runtime source directory,
application source or captured-input originals, which permits retained replay
after build inputs are unavailable.

## Explore a typed cluster scenario

The cooperative profile admits bounded seed campaigns over one prepared guest
module. Each scenario declares its nodes, directed links, durable volumes,
boot arguments, lifecycle actions and output-bound crash/stop markers. The
`gomad-wasm.scenario/v1` input is bounded to 256 KiB and retains the underlying
cluster spec's explicit resource limits. Its seed is zero: the campaign supplies
each execution's seed. Boot arguments exclude argv[0]. Unknown fields, invalid
topology, unused boundaries and inconsistent restart/storage bindings fail
preflight. Arbitrary callbacks and live lifecycle controls are not scenario inputs.

Use the checked acceptance inputs in `command/testdata/scenario`. Starting from
the repository root, build the launcher/helper as above, then create a separate
installation with their complete cooperative network configuration:

```sh
/absolute/operator/bin/gomad install \
  --compiler /absolute/stock-go/bin/go \
  --helper /absolute/checkout/tools/gomad/rust/wasmhost/target/release/gomad-wasmhost \
  --runtime-root /absolute/checkout/tools/gomad/internal/model/runtimesource \
  --profile gomad.wasi-cooperative/v1 \
  --config tools/gomad/internal/backends/wasm/command/testdata/scenario/config.json \
  --cache /absolute/operator/cache/scenario-preparation \
  --compile-cache /absolute/operator/cache/scenario-engine \
  --output /absolute/operator/bin/gomad-scenario.json
export GOMAD_WASM_INSTALLATION=/absolute/operator/bin/gomad-scenario.json
cd tools/gomad
/absolute/operator/bin/gomad discover
/absolute/operator/bin/gomad explore --json \
  --scenario internal/backends/wasm/command/testdata/scenario/durable.json \
  --seeds=7,19 --parallel=1 --on-failure=all \
  --keep-successes=all --success-limit=32 --success-bytes=512MiB \
  --execution-timeout=30s --overall-timeout=3m \
  --artifacts /absolute/operator/durable-artifacts \
  go-run ./internal/backends/wasm/command/testdata/scenario
/absolute/operator/bin/gomad explore --json \
  --scenario internal/backends/wasm/command/testdata/scenario/network.json \
  --seeds=7,19 --parallel=1 --on-failure=all \
  --keep-successes=all --success-limit=32 --success-bytes=512MiB \
  --execution-timeout=30s --overall-timeout=3m \
  --artifacts /absolute/operator/network-artifacts \
  go-run ./internal/backends/wasm/command/testdata/scenario
```

`durable.json` syncs a file and its directory, crashes node `a` at its declared
`committed durable-a` output, then reopens the durable value in incarnation 2
while node `b` runs independently. `network.json` partitions both directed links;
the first client/server incarnations observe timeouts. A timed heal at 20 virtual
seconds permits fresh incarnations to exchange `ping`. These fixtures exercise
specific storage and stream boundaries, not every storage fault or partition.

`ringpop.json` (fn-224) runs three nodes that each join a ringpop ring over the
modeled network with the real `ringpop-go`/`tchannel-go` client. The guest is the
nested module `internal/backends/wasm/testdata/ringpop`; its `go.mod` pins every
dependency through the Temporal module, so run it from that directory with a
cooperative installation built from `scenario/ringpop-config.json`:

```sh
/absolute/operator/bin/gomad install ... \
  --config tools/gomad/internal/backends/wasm/command/testdata/scenario/ringpop-config.json \
  --memory-bytes 1073741824 --fuel 1125899906842624 \
  --output /absolute/operator/bin/gomad-ringpop.json
cd tools/gomad/internal/backends/wasm/testdata/ringpop
/absolute/operator/bin/gomad --installation /absolute/operator/bin/gomad-ringpop.json \
  explore --json --scenario ../../command/testdata/scenario/ringpop.json \
  --seeds=7,19 --parallel=1 --on-failure=all \
  --keep-successes=all --success-limit=32 --success-bytes=4GiB \
  --execution-timeout=15m --overall-timeout=35m --terminate-grace=2s \
  --artifacts /absolute/operator/ringpop-artifacts --working-dir "$PWD" go-run .
```

Node `c` is partitioned from `a` and `b` at 10 virtual seconds and healed at 30.
At 25 seconds each node prints its reachable view, and `a` and `b` must already
have dropped `c`. After the heal each node must see exactly the three members on
three consecutive one-second polls, printing `converged <node> after heal <d>
checksum <n>`, and still see them at its 90-second deadline; otherwise it panics
and the failure is retained. Nodes report through their exit status rather than
an output stop marker, because a stop would end a node before its peers
converge. Equal checksums across the three nodes show one ring.
The scenario sets `observation_bytes` to its 64 MiB maximum: the cluster broker
charges 64 bytes of World payload per brokered operation for the whole run, and
three gossiping nodes use about 940 operations per virtual second. Three nodes
currently run at several wall seconds per virtual second, so keep the timeline
short (fn-235 tracks the cost and fast capacity failure).

For an intentional replayable failure, use `jq` to change only the recovery boot
in the same durable input, from the `tools/gomad` directory:

```sh
jq '.boots |= map(if .id == "recover" then .args = ["canary"] else . end)' \
  internal/backends/wasm/command/testdata/scenario/durable.json \
  > /absolute/operator/canary.json
/absolute/operator/bin/gomad explore --json --scenario /absolute/operator/canary.json \
  --seeds=7 --parallel=1 --on-failure=all \
  --execution-timeout=30s --overall-timeout=3m \
  --artifacts /absolute/operator/canary-artifacts \
  go-run ./internal/backends/wasm/command/testdata/scenario
/absolute/operator/bin/gomad inspect --json /absolute/retained/artifact
/absolute/operator/bin/gomad replay /absolute/retained/artifact
```

Take the artifact and campaign paths from the emitted shared campaign events;
the paths above are placeholders for those reported locations. The canary
campaign returns 1; matching replay also returns 1 with `reproduced=true`.
Successful matching replay returns 0. Cluster replay checks the retained module,
lifecycle, network/storage evidence and per-incarnation choices. It reports
`choice-replay=none`: a declared crash prefix is not a complete global tape.
`--verify-only` and `--observed` are unsupported for cluster artifacts.

These campaigns require seed strategy without `--choices`, `--diagnostics`,
coverage or guidance. They do not enable cluster-global forced-choice search,
cluster minimization. Resource,
replay-divergence and helper failures retain their existing classifications.

A cluster run ends as soon as an incarnation fails, instead of waiting for the
execution watchdog. Exhausting a broker World limit (for example the
`observation_bytes` payload budget) ends it as backend capacity, exit 3, with
`WASI capacity: <dimension> limit=<n> used=<n> delta=<n>`. The Runner-failure
artifact retains each incarnation's partial streams as an interrupted prefix.
`gomad inspect` adds a `cluster` section (JSON `artifact.cluster`): per
incarnation node, incarnation, terminal reason, stdout/stderr byte counts and a
256-byte tail, the abort virtual time and the exhausted limit. `gomad replay`
reproduces the capacity abort. Output and choice capacity still retain nothing.

## Recover and resume an interrupted scenario campaign

After an interruption, inspect the reported campaign path and recover its store
before resuming with the same compatible installed provider:

```sh
/absolute/operator/bin/gomad inspect --json /absolute/interrupted/campaign
/absolute/operator/bin/gomad recover --json /absolute/interrupted/campaign
/absolute/operator/bin/gomad resume --json /absolute/interrupted/campaign
```

Recovery validates committed evidence and repairs only a recognized torn tail;
resume preserves completed outcomes and deterministic ordinal order and runs
remaining work from retained prepared inputs. The scenario JSON, application
source, compiler, runtime sources and original captured-input mounts need not
remain available. The installed helper and compatible model/runtime identities
remain required. Damaged retained payloads, changed installations or missing
prepared inputs fail closed without replacing valid evidence. Inspection and
recovery execute no guest and require no installation; resume does.
An interrupted campaign is not a successful completed campaign. A reproduced
application failure during resume still returns status 1.

## Inspect and replay a standalone failure

From a source checkout, an unchanged Gomad canary demonstrates the workflow:

```sh
gomad explore --choices --diagnostics --seeds=7 --parallel=1 \
  --working-dir /absolute/checkout/tools/gomad/internal/backends/wasm/testdata/io_failure_module \
  --build-tag=gomad_fixture --build-tag=test_dep \
  --artifacts /absolute/operator/artifacts \
  go-test ./io_failure -- -test.run=^TestDeterministicIOFailure$ -test.v
gomad inspect --json /absolute/retained/artifact
gomad inspect --choices /absolute/retained/artifact
gomad replay --verify-only /absolute/retained/artifact
gomad replay --observed /absolute/operator/replay-output /absolute/retained/artifact
```

The first command intentionally returns status 1 and prints the actual retained
artifact path and a replay command that includes its installation selection.
Replay of that failure also returns status 1 when `reproduced=true`; this is
Gomad's existing target-failure status, not a replay mismatch. A matching
successful artifact and `--verify-only` return 0. Invalid input returns 2;
Runner/infrastructure failure returns 3. Inspection and recovery need no
installation because they read retained evidence without executing a guest.

The Rust host's artifact `inspect --json --choices` adds a `symbol` to every
aggregate site and listed decision, using only its authenticated retained Go
module. Available symbols carry `package`, `function`, optional proved
`file`/`line` and `inlined_from`. An empty inline list on physical fallback
means no inline chain was safely proved. Recorded missing runnable sites stay
`{"unavailable":"site_missing"}`; other unavailable reasons are `no_symbols`,
`offset_out_of_range`, `symbol_mismatch` and `symbol_table_too_large`. Missing
required names, including in real `-s` modules with retained pclntab, yield
`no_symbols`. Module digest/read errors still refuse inspection.
Analysis shares fixed cumulative limits and a single immutable result map;
exhaustion discards partial symbols. Artifacts and all recorded identities
remain unchanged. The [Rust inspection contract](../CLI.md#rust-artifact-inspection)
lists the limits and the pending authenticated campaign-divergence consumer.

Replay loads the retained module, provenance, modeled inputs, choices and
diagnostics. It validates the installed helper, compiled model/runtime code,
engine/host and immutable profile against that artifact. It does not rebuild.
The manifest must preserve those identities and guest configuration even when
the original compiler, source tree, runtime sources, captured mounts and
preparation cache have been removed. Moving the helper changes its path, which
can be updated while preserving its digest; changing helper bytes is rejected.
Evidence portability beyond the pinned engine and host remains unqualified.

Guest stdout and stderr are checked for Temporal soft assertions after execution,
including assertions logged after a test prints `PASS`. A detected assertion is
classified as `soft_assertion`; its signature uses the static message, with the
instance added for clusters. Dynamic log fields remain in the evidence.
Unreadable or truncated output makes the oracle inconclusive. If an output bound
terminates execution, the capacity error remains authoritative and reports that
the oracle is inconclusive; it does not claim complete replay evidence.

For a WASM `go-test` workload that deliberately triggers assertions, pass
`--soft-assertion-report-only` explicitly. The option is part of target identity,
is restored by replay and resume, and reports observed assertions after each
committed execution without failing otherwise successful execution. Reports are
also kept in campaign inspection, even when successful artifacts are not retained.
Other target kinds refuse it before preparation. Cluster callers bind per-incarnation
report-only policies in the sealed `OraclePlan` and declare a Go test boot kind.

## Minimize a compatible retained failure

The existing shared minimizer accepts a standalone cooperative target-failure
artifact with complete exact choice evidence and a compatible installed provider:

```sh
gomad --installation /absolute/operator/bin/gomad-wasm.json minimize \
  --attempt-budget=64 --max-bytes=512MiB \
  --artifacts /absolute/operator/artifacts /absolute/retained/artifact
gomad --installation /absolute/operator/bin/gomad-wasm.json minimize \
  --resume --attempt-budget=64 --max-bytes=512MiB \
  --artifacts /absolute/operator/artifacts /absolute/retained/artifact
gomad inspect --choices /absolute/minimized/artifact
gomad --installation /absolute/operator/bin/gomad-wasm.json replay \
  /absolute/minimized/artifact
```

Minimization reduces only the required forced-choice prefix. Module, seed, argv,
environment, captured inputs and model/fault configuration remain fixed. Every
accepted candidate must preserve the failure signature and outcome, retain its
complete internal/model/choice/evidence trace and match a fresh exact full-tape replay. It does
not shrink a guest heap, trace, storage image, network plan or typed cluster
lifecycle. Successes, watchdog/infrastructure observations, stock observed replay,
missing/incomplete choice evidence and simulation/connected-World
profiles are rejected. Typed multi-instance crash/stop prefixes are not complete
standalone tapes and are not admitted by this command.

The parent stays immutable. Published reductions retain parent/predicate/budget
evidence, with simulation replay explicitly absent for this standalone mode.
State uses the shared per-parent locked store below
`ARTIFACTS/minimized/.minimize`; resume requires the same parent, artifact root
and bounds. The attempt default remains 64, and omitted `--max-bytes` retains the
shared default. Syntax errors are rejected before installation resolution.
Retained execution uses the artifact's module and captured inputs,
without rebuilding or reopening original source/compiler/runtime inputs.

A completed minimization command returns 0 and reports `changed`, attempts,
accepted reductions, stop reason and the resulting artifact path; `--json` uses
the existing shared result format. Grammar and replay-preflight errors return 2;
other minimizer errors, including remaining semantic admission errors, and
infrastructure/output failures return 3. Replay of a reproduced minimized
application failure still returns 1. This standalone route does not extend
qualification to typed clusters or arbitrary workloads; source tests alone do
not establish a production qualification claim.

## Replay a retained multi-instance artifact

New simulation runs and retained replay use one Rust cluster helper, with fresh
stores for started and restarted incarnations. Boot configurations are sealed
before execution, while Start remains imperative. Retained evidence binds the
helper and model implementation; it cannot be relabeled as a per-process Go
reference run or replayed with an older helper identity.
The retained schema has no execution-path selector: replay chooses the current
in-process provider after verifying those identities. The Go-broker reference
is selected by the internal differential harness, not a CLI replay flag.

Typed retained multi-instance artifacts use the same `replay ARTIFACT` parser,
inspection and replay classification. The launcher recognizes only the verified
`gomad.wasm-simulation-retained/v2` evidence schema and delegates its sealed
typed lifecycle to the existing cluster replay owner. The evidence references the
single guest module by digest; its bytes are the artifact's verified target file,
copied without following links once its listed size is within the 512 MiB
module bound, and re-hashed before any helper starts. No helper
bytes are retained. Artifacts with the older `v1` schema, which embedded module
and helper bytes, are refused with a versioned error and must be re-recorded.
That refusal follows only the unsupported-option preflight (`--verify-only`,
`--observed` and `--toolchain-root` return 2), precedes installation
resolution and any retained payload read, and returns 3.
Payload hashes establish only self-consistency, so replay resolves
the WASM installation, requires every retained boot's helper digest and engine
identity to equal the installed `HelperSHA256` and pinned engine, re-hashes the
installed helper and executes it; a missing installation or any mismatch fails
closed before guest files are written, naming the recorded and installed
identities. Replay does not open the original source, compiler or mount paths.
A fresh private module file and modeled instance state are created for replay,
then the private temporary files are removed.
`--verify-only`, `--observed` and `--toolchain-root` are explicitly rejected for
this artifact shape before any guest executes. Successful matching clusters
return 0; matching retained application failures return 1. This route does not
expose a second campaign parser, arbitrary scenario callbacks or new exploration
semantics. Its final Temporal support cells remain pending their actual gates.
The sealed cluster lifecycle and per-incarnation choice prefixes are checked;
they are not one complete global choice tape, so the shared replay output reports
`choice-replay=none` rather than claiming global exact-choice coverage.

## On-demand qualification and retained cost report

```sh
gomad --installation /absolute/operator/bin/gomad-wasm.json \
  qualify --spec /absolute/qualification-spec.json
```

The spec is the qualification owner's JSON `Spec`: `Manifest`, `Campaign` and
absolute `ArtifactRoot`. `Options` must be absent; the selected installation is
the sole backend configuration owner. Use an explicit frozen integrated-source
digest in `Manifest.IntegratedSourceSHA256`, workload/ID, seed and 2..32 repeats.
Final candidates also supply the conductor's absolute
`Manifest.IntegratedSourceManifestPath`; exploratory fixture measurements must
be marked `Manifest.Provisional: true` and cannot become final support evidence.
`Campaign.Target` supplies the unchanged source, target kind/arguments,
absolute working/preparation directories and build tags. Campaign timeout
values are integer nanoseconds, as in the shared Go campaign record. Capacity,
profile and actual workload/compiler/runtime/model identities are frozen from
preparation; any declared identity that disagrees fails closed. The integrated
candidate digest remains a separate conductor-supplied identity, not a hash
invented from the workload. Do not replace it with a historical commit.

Qualification uses Gomad's existing repeated-execution, retained support report
and fresh replay owners. JSON output is the bounded
`{ "result_path": "...", "operator_receipt": { ... } }` response, at most
8 KiB including its newline; `result_path` is absolute and at most 1024 bytes.
Post-observation error details are truncated to 1024 bytes with an explicit
marker; the raw publication receipt remains retained for diagnosis.
The immutable full Result is durably published with the owner's 64 MiB bound.
Use `report --result` below to read its complete raw fields; qualify does not
serialize a second full Result after the terminal observation.
the exit status uses the existing qualification classification. In particular,
a repeatedly reproduced application failure returns 1, even when deterministic.
An invalid spec returns 2; preparation/Runner/report failures return 3.
After retaining the raw result and emitting its path/receipt, final candidates return 3
for `CostFailure`, `BudgetFailure`, a missing `Budget`, or `!Budget.WithinBudget`.
This cost/infrastructure gate does not rewrite the support classification
or label a resource failure as an application bug. Explicit provisional
diagnostic runs retain their support-only exit status and expose cost failures
in the raw result; they are not cost acceptance passes.
Missing or invalid terminal publication observations and publication/output
errors return 3, not an application-failure classification. When a Result was
published, its path and available raw receipt are emitted before that failure
when they fit the bounded response; invalid/oversize response paths fail closed.

```sh
gomad report --result /absolute/results/wasm-qualifications/v1/result-HASH.json
```

This source-less command opens the qualification owner's content-addressed
result and emits JSON. It does not discover an installation or open a compiler,
helper, source tree or original mounted input. It validates the existing support
report and recomputes costs from retained raw samples. Missing CPU, guest memory
or host RSS measurements remain an explicit `CostFailure`, never a measured
zero or a qualification pass. Malformed or changed retained results return 3;
command syntax errors return 2.

The result binds source, compiler, runtime, helper, engine, model, host,
configuration and capacity identities. Execution and replay counts remain
separate. Compiler/preparation, engine compilation/cache, fresh instantiation,
guest execution, process CPU/RSS, guest peak memory, fuel and trace size are
different measurements; virtual-time acceleration is not scheduling coverage.
Retained Result operator observations span spec decoding, installation/input
setup, preparation and execution. The separate `operator_receipt` binds the
exact published Result SHA256 and operator PID to actual post-publication
`RUSAGE_SELF` CPU/high-water RSS, including full Result serialization and durable
publication. Its owner rejects missing/backward counters and an operator peak
over 2 GiB, and retains the observed receipt beside `result_path` as
`result_path.publication.json`, using the owner's strict 8 KiB JSON receipt
limit. Only the bounded receipt/response emission and
process-exit tail follow that snapshot; it is not a measurement at process exit.
The mandatory corpus opens and validates the receipt in a separate reader
process and adds its measured self CPU tail once; that reader's full Result
validation is not attributed to the producer. Child compiler/helper accounting
remains separate. Final receipt-backed operator gates still await actual runs.
Support applies only to the named workload, profile and retained executions.

The declared 16 GiB host protocol permits one heavy preparation at a time,
at most two resident guests and one visible continuation. Individual cold
helpers must peak at or below 6 GiB, warm helpers at or below 4 GiB, compiler
processes at or below 6 GiB and the operator at or below 2 GiB. These individual
helper limits apply to the per-process reference. Its resident helpers' per-node
warm peak sum must stay at or below 8 GiB; this is a conservative bound, not a
measurement of simultaneous RSS. An in-process cluster instead records one
actual process image peak for all resident stores and retained boot templates,
bounded by 8 GiB regardless of compile-cache hit or miss. Each incarnation also
requires a nonzero guest linear-memory peak within its own `memory_bytes`.
Cluster helper samples match store samples by phase and iteration; replay must
supply its own process measurement. A helper without
image telemetry, such as one ended by a modeled crash or stop, has only an
inherited wait upper bound; it is reported separately and not checked against
these helper limits. Cold preparation
has a separate 10-minute budget and is reported separately; individual helper
execution/replay has a 15-minute budget. Missing process measurements remain an
explicit budget failure. These declared bounds are not evidence that a
particular Temporal cell passed them.

The owner accepted the measured 2.911× workflow execution speedup
(authorization to validated terminal) for now; no all-workload 5× threshold
applies. That acceptance covers execution cost only: the same retained workflow
pair's full-wall ratio was 0.694×, so no end-to-end or WASM-versus-native speedup
is claimed. Historical 5× gate failures remain unchanged evidence. An
execution-boundary measurement is conditional on the tested workload, profile and
clock and does not replace full helper wall time or cold preparation. Virtual time is not
compute time or schedule coverage.

## Capabilities and limits

| Profile | Operator behavior |
| --- | --- |
| `gomad.wasi-stock-exploratory/v1` | Stock WASI with modeled imports; seed campaigns and observed repetition. Runtime choices and diagnostics are rejected. Go 1.27.0 or 1.27.1 requires its own pinned compiler identity. |
| `gomad.wasi-cooperative/v1` | Go 1.27.1 cooperative runtime; standalone `--choices`, bounded full-tape replay, seed diagnostics and bounded choice exploration; typed `--scenario` seed campaigns with modeled networking, storage, declared crash/restart, retained replay and recovery/resume. The default read-step is zero; time advances through the guest idle/deadline handshake. |

Bounded choice exploration uses the same grammar and frontier owner as Gomad:

```sh
gomad explore --strategy=choice-exploration --choices --coverage=choice \
  --seeds=7 --parallel=1 --max-executions=16 --max-choice-depth=4 \
  --max-exploration-bytes=8MiB --on-failure=all \
  go-test ./supported/package -- -test.run=^TestCanary$
gomad recover --json /absolute/interrupted/campaign
gomad resume --json /absolute/interrupted/campaign
```

The execution/depth/byte bounds are required; they do not prove exhaustive
scheduling coverage. Diagnostics retain their current seed-only restriction.
Resume retains campaign identities and ordering and requires a compatible
installed provider. Forced decisions must match their retained sites and
alternatives; incompatible/missing/extra decisions fail as divergence.

Cooperative installation accepts validated network/storage configurations;
stock installation refuses them. Typed scenarios use the checked input route
above. The operator rejects cluster-global forced-choice/simulation exploration,
semantic coverage/guidance and unsupported clock/mount flags.
Captured WASM inputs belong to the installation
configuration. `exec`, adapters/overlays/modfile substitutions and
non-closure capability modes are rejected by source preparation. `plan`, shard
execution, analysis and doctor are not exposed by the WASM launcher.

WASM does not establish native race detection, weak-memory coverage, asynchronous
preemption, optional checkpoint safety or strict World clock qualification.
Fuel/capacity/non-progress and helper infrastructure failures remain distinct
from replayable application failures. Final changed-candidate runtime,
mandatory workload and retained-replay qualification, including the complete
behavioral reach of `make -C tools/gomad test-wasm`, remain pending under
fn-232 R12/task .13's frozen Rust qualification cycle. Deletion source checks
supply no final qualification pass.

Storage qualification additionally names its lock policy. Raw files, including
lock-named files, remain durable by default. The explicit
`StorageModel.SQLiteLockDatabases` allowlist names database paths whose ncruces
`sqlite3_dotlk` `.lock` and `-shm` artifacts are process-lifetime coordination
state. It does not make arbitrary lock files ephemeral or establish native
POSIX locking parity. Database/WAL/journal writes still use the shared volume
model: acknowledgement, sync/durable frontiers and crash recovery are separate.
Fault support must name the tested loss/reorder/corruption or crash boundary;
ordinary writes and successful restart are not proof of every storage fault.
Cooperative installation admits this explicit policy; a workload support claim
still requires its corrected guest and retained replay qualification gates.

Live WASM `Cluster.EnumerateCrashStates` is unsupported and returns the typed
`UnsupportedWASMCapabilityError` for `live-volume-crash-enumeration`, after the
existing stale-handle/state checks.
This does not reject every crash enumeration: `Request.StorageCrashLimits` and
the captured completed-result `StorageCrashEnumeration` remain supported
pure projections of retained post-crash volume state, within their tested scope.
In cooperative storage runs the Rust helper computes these enumerations, the
crash selection and the recovery image; Go decodes them strictly. A storage
image's identity is the SHA-256 of its self-contained recovery image v1 blob,
replacing the earlier snapshot/occurrence JSON identity, so storage image
identities in retained evidence produced before fn-176 do not compare with new
ones. Helper-owned runs return the blob when `Request.ReturnStorageImage` asks;
broker (split-owned) runs always return it because the broker may restart any
incarnation.

## Cache and evidence status

Preparation caches bind source/compiler/runtime/helper/model/profile/configuration
and limit identities through the existing complete build key. Cached module,
receipt, import inventory and initial memory are validated before use; a changed
identity fails closed. Compiler object caches are private to the build key and
external cache programs are disabled. The optional helper compiled-code cache
shares immutable engine code; every execution still creates a fresh guest heap,
store and model state. Cache location is host preparation state, not a guest
semantic input. The v3 cache binds actual helper/module/engine/platform identities
and validates a bounded CRC-32 envelope before deserialization from memory.
Hits use the artifact for the prepared module digest without reading the module;
misses verify module bytes before compilation. Checksums detect corruption, not
forgery by the trusted same-user cache owner. Earlier v2 entries are preserved.
The trust boundary requires a current-user-owned 0700 directory and regular,
single-link, no-follow 0600 entries. Old filename-only cache files are not
trusted or reused. The helper's engine tests reject corrupt, stale, symlinked,
hard-linked, permissive, directory-linked and oversized entries before a guest
starts. The opt-in installed operator test requires a poisoned or symlinked
compiled entry to fail replay of the stock and cooperative fixtures and a
restored entry to replay again. These are source tests, not a retained installed
gate result or qualification of arbitrary cached Temporal code. The current cache artifact bound is 1 GiB.

A provisional cooperative ready-select/timer fixture measured one cold and one
warm execution plus two fresh replays. The cold engine compilation took 1.256 s;
the warm sample compiled no code. Helper wall time was 1.346 s cold and 0.076 s
warm. Both executions retained the same outcome, 1,408,515 fuel units and 5,176
trace bytes. Compiler-inclusive CPU for the two executions and two replays was
8.356 s; peak guest memory was 3,014,656 bytes and whole-process/compiler peak
RSS was 567,066,624 bytes. These are observed fixture costs on pinned Linux
arm64, not a performance guarantee or Temporal support claim. They remain
`Provisional: true` until the integrated candidate gate is frozen and rerun.

Current unstripped corrected-source artifacts retain diagnostic names. A
read-only section inventory measured these candidates (decimal MB); their
network/storage/broker qualification is still independent and pending:

| Candidate | Artifact | WASM code | Data | Diagnostic names |
| --- | ---: | ---: | ---: | ---: |
| Temporal SDK retry/network fixture | 242.70 | 134.65 | 97.24 | 10.29 |
| Temporal storage recovery fixture | 240.26 | 133.28 | 96.28 | 10.18 |
| Temporal broker fixture | 240.27 | 133.29 | 96.29 | 10.18 |

These modules have no separate DWARF custom sections. Removing names would not
remove linked provider code or guest data and is not a supported optimization
here. Existing opt-in `test_dep`, `disable_grpc_modules` and `sqlite3_dotlk`
selections are retained.

The guest-only `gomad_wasm_slim` tag (fn-182) is the supported provider trim.
It leaves out the Google Cloud Storage and S3 archivers and the
MySQL/PostgreSQL functional-test drivers, which the qualified Temporal guests
never construct. Ringpop membership stays linked (fn-224). The one-box and multi-instance qualification cells
select it with `disable_grpc_modules,gomad_wasm_slim,sqlite3_dotlk,test_dep`;
other targets opt in through `Campaign.Target.BuildTags`. Selecting an excluded
provider in such a guest fails: `gs`/`s3` archival returns a "not built" error and
`-persistenceDriver=mysql8|postgres12|postgres12_pgx` keeps SQL visibility and
fails plugin lookup (`supported plugins: [sqlite]`). Elasticsearch stays linked
because the schedule query converter uses its query types. Tags and the selected
source closure are preparation build-key and provenance inputs, so trimmed and
full modules never share a preparation or compiled-cache entry, and retained
artifacts replay only with their recorded module. No other provider is pruned.

`TestInstalledOperatorDiscoversInspectsAndReplaysWithoutBuildInputs` drives a
built launcher, new manifest, unchanged failing Go test and a captured-input
canary in an actual guest, retained inspection,
then a fresh helper replay after original build/input directories are renamed
unavailable. It covers stock observed and cooperative exact replay separately.
Select it with an explicit existing stock compiler:

```sh
cd tools/gomad/internal/backends/wasm
GOMAD_STOCK_GO=/absolute/stock-go/bin/go \
  /absolute/stock-go/bin/go test -v -tags test_dep -count=1 -timeout=15m \
  ./installation ./command
```

The fn-181 installed scenario matrix adds durable crash/reopen, partition/heal,
detached replay and interrupted recovery/resume using the checked inputs above.
Its actual guest gate is `TestInstalledOperatorTypedScenarioDetachedReplay` and
its interrupted coordinator gate is `TestInstalledOperatorTypedScenarioKillResume`
in `command` (Linux-only). Their current-candidate command exits and host identities belong
to the retained Flow acceptance evidence; an opt-in skip supplies no pass.
This document supplies no numeric determinism bound or new measured performance
claim.
