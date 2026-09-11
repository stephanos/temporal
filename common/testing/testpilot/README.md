# Testpilot

Testpilot runs behavior through Temporal and Workers using bounded Cases. Callers decode or construct a
`testpilot/v1` Case, prepare it against an immutable `Profile`, then execute the
returned `PreparedCase` through a caller-owned `Driver`.

Exact Case 1.0 is the only admitted format. Resource-bearing Programs declare a complete closed graph
of symbolic text resources; resource-free Programs may have an empty environment. The Case owns the IDs and relationships; the Profile owns their physical values.
Symbolic endpoint IDs are not transport addresses, and bindings grant no capabilities.

`Prepare` performs static admission without Driver I/O, snapshots the Case and Profile, resolves
private prepared resources, and includes the complete binding fingerprint in Prepared Case identity.
`PreparedCase.Run` checks the Driver identity, calls `Driver.Validate` without target I/O, creates the
Monitor, and only then opens a per-Run `Session`. Validation failure produces no Session, Run, Verdict,
or effect. Scheduling, recording, expression admission, and Contract evaluation stay private to this
package. The reusable Temporal Driver lives in `common/testing/testpilot/temporal`; functional
fixtures and provisioning remain under `tests/`. Drivers cannot replace the prepared Contract evaluator.

A Case may ask its Driver for a deliberate outage. `InjectFault` is a declared instruction like any
other: the Profile must authorize it, the role it names must be a task-queue role, and the Run
records one `FAULT_INJECTED` event per realized outage. Nothing about a requested fault is evidence
until that event exists.

A Case may also declare where its operation-correlated evidence comes from. A response projection can
lift a projected value into a declared `CorrelatedEvidence` Observation through guarded rules, which is
the only way a Program supplies the evidence a `Contract.correlated` capability reads. A capability that
admits no evidence answers inconclusive: silence is not a satisfied property.

## Preparation diagnostics

`NewCatalog`, `Prepare`, and `ProfileSpec.BindingFingerprint` return errors discoverable as
`*testpilot.PreparationError` through `errors.As`, including after ordinary error wrapping:

```go
prepared, err := testpilot.Prepare(source, profile)
if err != nil {
    var diagnostic *testpilot.PreparationError
    if errors.As(err, &diagnostic) {
        log.Printf("preparation %s at %s: %s", diagnostic.Category, diagnostic.Path, diagnostic.Detail)
    }
    return err
}
```

The six stable categories are `malformed`, `unknown`, `type_mismatch`, `unavailable`,
`unsupported`, and `limit_exceeded`. `Path` retains the admission input vocabulary and its
256-byte bound; `Detail` is human-readable, not a stable string API. Existing error messages,
including Contract rule context, are retained. Missing or invalid Profile/Catalog preconditions
are malformed; Profile binding validation, including binding ceilings, is also malformed without
changing its rejection limits. Program and Contract limit violations retain their existing categories.

These diagnostics cover static admission, including correlated Contracts. ProtoJSON decoding errors
and runtime Run/Driver failures keep their own error contracts. No diagnostic wire format is added.
See [the public error contract](preparation_error.go) and [Temporal Driver ownership](temporal/README.md).

## Running a Case from the command line

`tools/umpire/cmd/umpire-run` is the black-box consumer of these bytes. Given a fixture path, a gRPC
address, an HTTP address, and the namespace, task queue and optional Nexus endpoint the Case binds
to, it derives the Profile the Case implies through `temporal.DeriveProfile`, prepares the unchanged
bytes, opens a composite Driver with its own SDK worker, runs once, and prints the Run status, the
cleanup status, the Verdict status, and one line per rule Verdict.

Its exit codes separate the answer from the infrastructure: `0` satisfied, `1` violated, `2`
inconclusive, `3` preparation, infrastructure, or Run error. That is why a CI caller can tell an
unreachable server from a Run that really was inconclusive.

With `--create` it provisions the resources it names and deletes them on exit; without it they must
already exist and none is ever deleted. Only Cases whose Profile `DeriveProfile` derives are
runnable; a typed fixture rejects with its admission category on stderr. It links the Driver and the
SDK, never the functional test cluster.

It links no functional test cluster at all, and the only server packages in its transitive closure
are the three the Driver's own Nexus support already pulled in through `common/dynamicconfig`,
`common/persistence` and `chasm`. A unit test pins both halves of that, so a new coupling is a
failing test rather than a silent regression.
