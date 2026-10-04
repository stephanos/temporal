# Host vet gate reconciliation

Before production edits, the exact inherited Linux/amd64 and actual-host
`GOWORK=off go vet -tags test_dep ./...` commands fail solely beneath
toolchain/runtime/overlay. See baseline-linux-vet.log and baseline-host-vet.log.
The logs show standard-library internal visibility errors and additive Gomad
stdlib imports missing from the stock GOROOT. These are overlay inputs, not
checkout host packages; the existing inventory scout independently observed
the same exact excluded boundary and no errors in included host packages.

The replacement is root `TestHostPackageVet`, exercising the same complete
architecture inventory before invoking vet on every discovered host package
for darwin/arm64, linux/amd64 and the actual host source set. Exact required
exclusions and independent hidden/fixture/nested-module coverage are validated;
all included metadata errors fail. Package arguments come from discovery,
never fixed roots or error-filtered output. Target settings apply only to child
list/vet processes, so the test itself executes on this host.

Both the full root test Quick and its explicit focused invocation execute this
gate. Original RED outputs stay retained. The newly scoped baseline and final
commands must run and pass; this record authorizes the precision correction,
not a passing result. No public CLI command or flag changes, new dependency,
qualified-platform execution or native acceptance is implied. Overlay vet/tests
remain required in their installed patched GOROOT under the native owners.

The task's inherited Darwin-host prose is also corrected to the actual
Linux/arm64 host. Static inspection and cross-vet remain distinct from native
qualification on the two supported hosts.
