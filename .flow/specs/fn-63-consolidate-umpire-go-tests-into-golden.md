# Consolidate Umpire Go tests into golden scenarios

## Superseded by fn-64

This specification is a historical record and must not be implemented. Its golden ownership model and baselines were designed around fn-61, `PortableTestPlan`, Run Evaluation, caller closure, and the legacy runtime removed by fn-64.

The child tasks remain `todo` only because the tracker has no superseded/cancelled task state. They are intentionally unready and must never be started. Any future Case Runtime test consolidation requires a new evidence-backed spec after fn-64 establishes stable Case, execution, verification, Host, Run, and Verdict boundaries.

## Historical acceptance criteria

- **R1 (retired):** prove a shared golden-scenario harness for the discarded runtime.
- **R2 (retired):** consolidate portable execution and property-specific evaluation scenarios.
- **R3 (retired):** consolidate Run Evaluation and caller-closure Nexus scenarios.
- **R4 (retired):** consolidate legacy artifact-family acceptance goldens.
- **R5 (retired):** prune tests and lock regression gates around those legacy baselines.

Fn-64 owns current runtime coverage. Historical implementation details remain available in version control; they are not active architecture guidance.
