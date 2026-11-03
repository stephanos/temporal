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

Temporal protobuf schemas stay in `model/proto`, with generic imports
resolved from the sibling Umpire `protocol/proto`. From the Temporal root,
`(cd model && mise exec -- cargo xtask proto-go temporal)` regenerates this module's three protobuf
files; adding `--check` compares them with fresh output. Keep generated sources
aligned with the canonical generic schemas rather than editing descriptors by hand.
