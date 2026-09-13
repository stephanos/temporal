# Testpilot

Testpilot runs behavior through Temporal and Workers using bounded Cases. Callers decode or construct a
`testpilot/v1` Case, prepare it against an immutable `Profile`, then execute the
returned `PreparedCase` through a caller-owned `Driver`.

Exact Case 1.0 is the only admitted format. A Program's symbolic binding graph is the closed set of
text binding IDs its roles and expressions reference, which `Prepare` derives and resolves against the
Profile; a resource-free Program references none. The Case owns the IDs and relationships; the Profile
owns their physical values. Symbolic endpoint IDs are not transport addresses, and bindings grant no
capabilities. Preparation also derives what a Case no longer writes: an instruction's outcome fields
follow from its instruction, the worker activations a reservation carrier reserves follow from the
Profile's carriers, and an instruction limit the Case omits takes the Profile's
`InstructionDefaults`.

`Prepare` performs static admission without Driver I/O, snapshots the Case and Profile, resolves
private prepared resources, and includes the complete binding fingerprint in Prepared Case identity.
`PreparedCase.Run` checks the Driver identity, calls `Driver.Validate` without target I/O, creates the
Monitor, and only then opens a per-Run `Session`. Validation failure produces no Session, Run, Verdict,
or effect. Scheduling, recording, expression admission, and Contract evaluation stay private to this
package. The reusable Temporal Driver lives in `common/testing/testpilot/temporal`; functional
fixtures and provisioning remain under `tests/`. Drivers cannot replace the prepared Contract evaluator.

Driver authors import this package. Its `Session`, handle, `Coordinate`, role-policy and `Opcode` types
are aliases of the `common/testing/testpilot/contract` leaf, which imports neither private execution nor
the IR, so a package that needs only that vocabulary may import the leaf instead. Execution hands every
effect and reservation handle back to the Session that issued it; refusing a handle it did not issue is
that Session's decision.

A Case may ask its Driver for a deliberate outage. `InjectFault` is a declared instruction like any
other: the Profile must authorize it, the role it names must be a task-queue role, and the Run
records one `FAULT_INJECTED` event per realized outage, carrying the fault in its `fault_injected`
payload, which a Contract reads through a path. Nothing about a requested fault is evidence until
that event exists.

A Case may also declare where its operation-correlated evidence comes from. A response read can
lift a projected value into a declared `CorrelatedEvidence` Observation through guarded rules, which is
the only way a Program supplies the evidence a `Contract.correlated` capability reads. A capability that
admits no evidence answers inconclusive: silence is not a satisfied property.

## Field paths and enum literals

A Case reads and writes protobuf fields through field paths: `PathExpression.path`, a request
assignment's `target`, a response read's `path` and an evidence-lift rule's `operation`. Each is a
string in one grammar, which preparation parses and types against the message it addresses:

```text
path     = "" | segment { "." segment }
segment  = name [ selector ]
selector = "<" name ">" | "[*]" | "[" key "]" | "?"
key      = JSON string | [ "-" ] digit { digit } | "true" | "false"
name     = ( letter | "_" ) { letter | digit | "_" }
```

- A `name` is a protobuf field name, not its JSON name. The empty path is the whole value.
- `<member>` after a oneof's name reads that oneof's member `member`, absent unless it is the selected
  one: `attributes<nexus_operation_completed_event_attributes>.scheduled_event_id`.
- `[*]` fans out over every element of a repeated field, so the path's value is a list:
  `history.events[*]`.
- `[key]` reads the entry of a map field with that key, absent when no entry has it. The key's kind
  must be the map's: a JSON string for text keys (`labels["key"]`), a canonical base-10 integer for
  integer keys (`counts[-42]`), `true` or `false` for boolean keys.
- A final `?` reads whether a presence-tracking field is set, as a boolean: `child.optional_text?`.

A segment takes at most one selector. `Testpilot.Authoring.Path.make` is the one Lean printer and
spells a text key escaping only the quote, the backslash and control characters. A path outside the
grammar, an unknown field or member, or a key of the wrong kind rejects at preparation located at the
path's field and quoting its text.

An enum literal is `EnumValue { name }`. Preparation resolves the name against the enum its context
expects (the other operand of a comparison, or the field a request assignment targets) and rejects an
undeclared name, a name where the expected type is no enum, and an enum literal with no expected type,
quoting the name. A value read from a protobuf message carries its name too; a number the enum does
not declare is spelled in decimal, which no literal names.

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
changing its rejection limits. A Case declares no resource ceilings: Profile ceiling violations and a
Case bound (an instruction timeout or attempt count) above its Profile ceiling retain their existing
categories.

These diagnostics cover static admission, including correlated Contracts. ProtoJSON decoding errors
and runtime Run/Driver failures keep their own error contracts. No diagnostic wire format is added.
See [the public error contract](preparation_error.go) and [Temporal Driver ownership](temporal/README.md).

## Running a Case from the command line

`tools/umpire/cmd/umpire-run` is the black-box consumer of these bytes. Given a fixture path, a gRPC
address, an HTTP address, and the namespace, task queue and optional Nexus endpoint the Case binds
to, it derives the Profile the Case implies through `temporal.DeriveProfile`, prepares the unchanged
bytes, opens a composite Driver with its own SDK worker, runs once, and prints the Run disposition, the
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
