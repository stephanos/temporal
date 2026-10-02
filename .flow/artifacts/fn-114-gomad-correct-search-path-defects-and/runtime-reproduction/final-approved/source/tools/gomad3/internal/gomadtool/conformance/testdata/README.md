# Gomad v3 conformance fixtures

Module `gomad3.test` holds the black-box programs the conformance driver in
`internal/gomadtool/conformance` runs against the patched toolchain. The
driver's `runtime_*.go` files are the specification: each fixture's arguments,
environment, expected output, exit status, and timing come from there, and the
compiler fixtures under `intercept` and `interceptfail` come from
`deterministicio/boundary/compiler-tests.json`. The `io_*` fixtures are specified by the
runner's `runner/internal/execution/io_*_toolchain_test.go` and
`runner/replay_io_integration_test.go` instead: `io_filesystem` and `io_net` must reach every
modeled and every refused `os` and `net` entry in the boundary manifest, since
`TestBoundaryManifestSemanticCanaries` requires a positive probe for each one, and `io_signal`
and `io_user` build in guarded capability mode because their packages are forbidden imports. `net_bind` is specified by `runner/internal/execution/io_net_bind_toolchain_test.go`: it checks the
in-memory network's bind contract under the deterministic profile.

The Simulation fixture is not in this module. Capability review admits the
`tools/gomad3sim` runtime bridges only for the root module's own copy of that
package, so a fixture here would need a replacement that review refuses. It
lives at `tools/gomad3sim/testdata/simulation_exploration` in the root module
and is specified by `runner/coordinator_transport_test.go`, which runs a
simulation-exploration campaign over it locally and through the isolated
coordinator, and by `target/capability_review_test.go`, which requires its
closure review to stay supported.

The fixtures only use the standard library. Programs that print scheduling or
map-iteration order must not print addresses, because the driver compares their
output across address perturbations; the `GOMAD3_ADDRESS` marker printed by
`internal/perturb` is the one exception and only appears when
`-gomad-address-padding` is given.

`timer_callback_identity` starts two timer creators and arms both callbacks for
one virtual deadline before the process can quiesce. Each callback immediately
selects from its own two prefilled channels at a distinct site and sends its
label through a buffered channel. It performs no allocation, yield, or blocking
operation before that site. The driver correlates an immediately preceding
Runnable selection with the callback marker and the current parentless `/v1`
identity derivation. The first marker's observed two-alternative digest must match
two consecutive parentless IDs. When virtual time advances, main is blocked
and the timer-armers have exited or blocked; only the two due callbacks become
runnable. A callback without a qualifying selection has no inferred ID.
Seeds 6 and 16 and valid alternative prefixes under seed 6 expose opposite
firing orders and swapped callback identities. The same-seed prefixes complete;
a complete seed-6 prefix executed under seed 16 diverges at a select site. This
cross-seed experiment does not establish a supported replay failure.

`select_readiness` takes one stable shape name from the driver's registry:
`blocking-zero-ready`, `blocking-one-ready`, `blocking-two-ready`,
`nonblocking-default`, `timer-channel`, `closed-channel`, or `nil-channel`.
The readiness count describes the initial poll, independently of the case that
later completes. Buffered and closed channels are prepared before the select;
the zero-ready and timer cases only become ready after a one-second virtual
advance, which cannot happen while the polling goroutine is runnable. Nil
channels are disabled. Each run records one poll decision, including the six
shapes with fewer than two initially ready cases. The driver explores every
recorded decision without reduction, including runtime-owned Runnable choices,
until the prefix frontier is empty; the 2048-execution and 32-decision bounds
fail the check if reached instead of standing in for exhaustion.

`TestRuntimeSearchFixtures` runs these same checks directly when a patched
toolchain is installed. Setting `GOMAD3_RUNTIME_REPRODUCTION_DIR` to a new
absolute directory retains the fixture binaries, raw trace backings and terminal
frames, forced prefixes, command/exit records, and `search-reproduction.json`.
