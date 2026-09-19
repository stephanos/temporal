# Simplify the Umpire Go execution surface

## Superseded by fn-64

This specification is a historical record and must not be implemented. Its resident-executor facade, `PortableTestPlan` handoff, generated binding adapter, Run Evaluation path, and legacy transport assumptions were replaced by the standalone Case Runtime in fn-64.

The child tasks remain `todo` only because the tracker has no superseded/cancelled task state. They are intentionally unready and must never be started. No downstream spec may depend on fn-61. Useful testing or encapsulation ideas must be reconsidered against `Case`, `PrepareCase`, `PreparedCase.Run`, `Run`, `Verdict`, and the fn-64 Host/Monitor boundaries rather than copied forward.

## Historical acceptance criteria

- **R1 (retired):** establish a root resident-executor facade for the former portable-plan runtime.
- **R2 (retired):** migrate generated and end-to-end callers to the former portable-plan interface.
- **R3 (retired):** internalize the former generated binding handoff.
- **R4 (retired):** unify the former runtime contracts and execution state machine.
- **R5 (retired):** hide Temporal and Nexus mechanics behind that former facade.
- **R6 (retired):** retire the then-legacy HTTP and non-portable executor path.
- **R7 (retired):** lock down that discarded surface and its regression gates.

Fn-64 R1–R10 are the active replacement requirements. Historical implementation details remain available in version control; they are not active architecture guidance.
