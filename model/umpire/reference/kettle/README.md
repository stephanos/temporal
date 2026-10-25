# Kettle reference system

The reference includes the immutable `sample_kettle/model.rs` directly. Its 81
starts and exploration report remain canonical. `runtime()` selects the declared
20 °C, off, quiet start for a live instance; Reset restores that selected start.
The Go server is implemented independently from the protobuf service.

Heat uses bounded Reach with its actual request and an `Exact<Model>` target.
Only the request-authorized action can explain that exact resulting state. Fill,
Pour and Describe only match the current state. PollHeating callbacks must match the full
current state and carry a nonempty kettle identity and the enabled heater action.
Answer returns a RespondHeating request without committing the action. Only an
exact RespondHeating receipt advances the run; a captured request must agree
with the receipt's action and include the kettle identity. Each successful report
retains the complete witness from Reset, including earlier callback delivery;
a later Current observation repeats that witness. Reports already returned remain
claims about their own observation prefixes.

From `model/`, run `mise exec -- cargo test -p umpire-reference-kettle`.
The guest uses the common pinned builder via `make model-check` at repository root.

The Go server, native/wasm parity and live host tests live here in module
`umpire.local/kettle`, with generated API under `gen/kettle/v1`. Its `go.mod`
replaces `umpire.local/model` with `../../hosts/go`; it imports the transport
from `umpire.local/model/host`. Regenerate with
`bash model/scripts/proto-go.sh kettle` at repository root; `--check` verifies
the pinned protoc 29.5, protoc-gen-go v1.36.6 and protoc-gen-go-grpc v1.3.0 output.
The host test requires the built guest and fails if it is missing.

Task fn-233.2's measured live Kettle tests took 0.525 seconds. The first Kettle guest
build took 25.81 seconds and a warm rebuild took 2.38 seconds. The full
`make model-check` run took 426.52 seconds, exceeding the 60-second R30 soft
target while rebuilding existing compile-failure fixtures and guest profiles.
These measurements include the new reference in the normal gate.

`api.rs` is the exhaustive annotation mirror catalog; generated messages live under
`wire::umpire::reference::kettle::v1`. `config.rs` resolves the static ID limit and
per-kettle tag limit through the shared resolved-limit context. The live guest uses
`annotation_runtime()` so every healthy RPC request and response is inspected.

The native `tests/main/probes.rs` checks `expected/coverage.txt.snap`. From `model/`,
produce candidates with `INSTA_UPDATE=new mise exec -- cargo test -p umpire-reference-kettle --test main probes::`.
Go remains the live producer/comparator of `expected/{coverage,gaps,mutants,transcript}.txt.snap`.
To export live candidates, from `model/umpire/reference/kettle` run:

```sh
go test -tags test_dep -run '^TestKettle(AnnotationCoverageAndActualWireGaps|Mutants|JudgeAnswerTranscript)$' -count=1 . -args -umpire-snapshot-candidates /tmp/kettle-candidates
```

Comparisons still run and mismatches fail. Then, from `model/`, import those explicit candidates:

```sh
UMPIRE_SNAPSHOT_CANDIDATES=/tmp/kettle-candidates INSTA_UPDATE=new mise exec -- cargo test -p umpire-reference-kettle --test live_snapshots -- --ignored
```

Review the resulting text snapshot diffs before acceptance. The import control only bridges
live output into insta; it establishes no Go execution or agreement itself.
Invalid probes bypass normal request filling so the original bytes reach the server;
unsupported mutations and transport failures cannot establish a server finding.

Task fn-233.3's complete annotation/probe Go selection took 26.480 seconds
(27.754 seconds wall time), including fresh healthy controls for each applicable
wire mutation. Its final full `make model-check` took 438.622 seconds. This exceeds
the R30 60-second whole-gate soft target; the record retains that cost rather than
changing the test selection. Native probe execution took 0.16 seconds.
