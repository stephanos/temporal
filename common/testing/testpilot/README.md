# Testpilot

Testpilot runs behavior through Temporal and Workers using bounded Cases. Callers decode or construct a
`testpilot/v1` Case, prepare it against an immutable `Profile`, then execute the
returned `PreparedCase` through a caller-owned `Driver`.

Literal-only Programs use Case 1.0. Case 1.1 Programs declare a nonempty closed graph of symbolic
text resources. The Case owns the IDs and relationships; the Profile owns their physical values.
Symbolic endpoint IDs are not transport addresses, and bindings grant no capabilities.

`Prepare` performs static admission without Driver I/O, snapshots the Case and Profile, resolves
private prepared resources, and includes the complete binding fingerprint in Prepared Case identity.
`PreparedCase.Run` checks the Driver identity, calls `Driver.Validate` without target I/O, creates the
Monitor, and only then opens a per-Run `Session`. Validation failure produces no Session, Run, Verdict,
or effect. Scheduling, recording, expression admission, and Contract evaluation stay private to this
package. The reusable Temporal Driver lives in `common/testing/testpilot/temporal`; functional
fixtures and provisioning remain under `tests/`. Drivers cannot replace the prepared Contract evaluator.
