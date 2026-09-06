# Testpilot

Testpilot owns the reusable, bounded Case runtime. Callers decode or construct a
`testpilot/v1` Case, prepare it against an immutable `Profile`, then execute the
returned `PreparedCase` through a caller-owned `Driver`.

`Prepare` performs static admission without Driver I/O. `PreparedCase.Run`
checks the Driver identity before opening a per-Run `Session`; scheduling,
recording, expression admission, and Contract evaluation stay private to this
package. Functional test and future non-functional Drivers remain outside this
package and cannot replace the prepared Contract evaluator.
