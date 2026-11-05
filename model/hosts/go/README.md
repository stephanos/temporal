# Temporal Model host

This module adds Temporal probes, the committed Temporal guest and the reference
worker executors to the generic Umpire Go host. Its module identity remains
`umpire.local/temporal`; the generic host is
`github.com/temporalio/umpire/hosts/go`.

Keep sibling checkouts under the same parent:

```text
<dev-root>/
  temporal/
  umpire/
  go.work
```

From that parent, run `go work init` if no workspace exists, then add the real
modules with the following command. `go work use` preserves unrelated workspace
entries.

```sh
go work use ./temporal ./umpire/hosts/go ./umpire/reference/kettle ./temporal/model/hosts/go
```

The server and this module also replace the canonical generic host with its exact
sibling source path. Keep that checkout present; local development does not require
publishing either module. Run the module's tests from this directory with
`CGO_ENABLED=0 go test -tags test_dep ./...`.

The checked-in guest is exposed by `temporal.Guest()`. Ordinary Go tests use it
without Rust; Model changes require `make umpire-guest` from the Temporal root.
`make model-check` checks that guest's freshness.

The reference worker Bridge delegates actual invocation correlation to the sibling
generic `worker.Coordinator`. Temporal owns its five method bindings, complete
task identity, namespace and token extraction, request copies and response
validation. Raw and native SDK executors keep their existing transport and shutdown
ownership. The generic coordinator does not replace the Model's policy coverage
or supply a Run-owned heartbeat journal. See the framework's
[worker capture contract](../../../../umpire/hosts/go/README.md#worker-capture-and-finite-tables).

`Program.Decode` admits exact Activity roots, heartbeat histories and declared
boolean successor domains through the generic finite Table. `Program.NewSession`
binds the complete actual task identity through the Bridge and prepares independent
namespace/token-bearing requests. Raw execution registers each ticket before
dispatch and waits for the validated captured completion before following its
successor. An unmatched initial task uses the existing nonretryable unprogrammed
failure. The SDK retains the Program's immutable Match/Rows/Responses interface
and its existing conversion, notification and shutdown contracts.

Temporal protobuf schemas stay in `model/proto`, with generic imports
resolved from the sibling Umpire `protocol/proto`. From the Temporal root,
`(cd model && mise exec -- cargo xtask proto-go temporal)` regenerates this module's three protobuf
files; adding `--check` compares them with fresh output. Keep generated sources
aligned with the canonical generic schemas rather than editing descriptors by hand.
