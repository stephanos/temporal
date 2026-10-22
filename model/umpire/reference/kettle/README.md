# Kettle reference system

The reference includes the immutable `sample_kettle/model.rs` directly. Its 81
starts and exploration report remain canonical. `runtime()` selects the declared
20 °C, off, quiet start for a live instance; Reset restores that selected start.
The Go server is implemented independently from the protobuf service.

Heat requires its actual request and an exact resulting state. Fill, Pour and
Describe only match the current state. PollHeating callbacks must match the full
current state and carry a nonempty kettle identity and the enabled heater action.
Answer returns a RespondHeating request without committing the action. Only an
exact RespondHeating receipt advances the run; a captured request must agree
with the receipt's action and include the kettle identity.

From `model/`, run `mise exec -- cargo test -p umpire-reference-kettle`.
The guest uses the common pinned builder via `make model-check` at repository root.

The Go server and generated API live in the generic host module at
`umpire/hosts/go/reference/kettle` and `reference/kettle/v1`. Regenerate with
`bash model/scripts/proto-kettle-go.sh` at repository root; `--check` verifies
the pinned protoc 29.5, protoc-gen-go v1.36.6 and protoc-gen-go-grpc v1.3.0 output.
The host test requires the built guest and fails if it is missing.

The task's measured live Kettle tests took 0.525 seconds. The first Kettle guest
build took 25.81 seconds and a warm rebuild took 2.38 seconds. The full
`make model-check` run took 426.52 seconds, exceeding the 60-second R30 soft
target while rebuilding existing compile-failure fixtures and guest profiles.
These measurements include the new reference in the normal gate.
