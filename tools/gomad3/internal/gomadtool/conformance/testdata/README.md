# Gomad v3 conformance fixtures

Module `gomad3.test` holds the black-box programs the conformance driver in
`internal/gomadtool/conformance` runs against the patched toolchain. The
driver's `runtime_*.go` files are the specification: each fixture's arguments,
environment, expected output, exit status, and timing come from there, and the
compiler fixtures under `intercept` and `interceptfail` come from
`deterministicio/boundary/compiler-tests.json`.

The fixtures only use the standard library. Programs that print scheduling or
map-iteration order must not print addresses, because the driver compares their
output across address perturbations; the `GOMAD3_ADDRESS` marker printed by
`internal/perturb` is the one exception and only appears when
`-gomad-address-padding` is given.
