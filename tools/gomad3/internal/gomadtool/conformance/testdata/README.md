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
