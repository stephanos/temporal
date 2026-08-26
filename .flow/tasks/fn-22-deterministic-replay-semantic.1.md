---
satisfies: [R1, R2]
---
# fn-22-deterministic-replay-semantic.1 Admit replay bundles and normalize violation signatures

## Description
### Umpire4 reconciliation (normative)

The replay bundle must name and bind three non-interchangeable classes: canonical semantic replay, concrete complete-ExperimentSpec rerun, and Temporal SDK history replay. Only canonical semantic replay can establish a semantic violation or support promotion; SDK history replay is diagnostic compatibility evidence.

The legacy implementation detail below is retained for context but is subordinate to this reconciliation.

Create the deep `tools/umpire/replay` entry boundary over fn-18's existing decoder/validator. Define an in-memory checked ReplayBundle for the exact fn-21 six-member set and the spec's exact canonical ViolationSignature projection over admitted qualified Results. Recompute all member/set identities and require the closed succeeded/qualified/violated, single-Property caller-closure binding before returning either value. Implement the closed evidence-role table, opaque-value equality classes, field-by-field derivation projection, ordering, and canonical identity formula while excluding only the enumerated plan/runtime transport facts. Keep this a projection over existing semantic authority, with no new wire type or evaluator. Add independent fixture/mutation tests for every included and excluded field and preserve existing comments.

**Size:** M
**Files:** `tools/umpire/replay/bundle.go`, `tools/umpire/replay/signature.go`, `tools/umpire/replay/bundle_test.go`, `tools/umpire/replay/signature_test.go`
**Touches:** [tools/umpire/replay/bundle.go, tools/umpire/replay/signature.go, tools/umpire/replay/bundle_test.go, tools/umpire/replay/signature_test.go]
## Acceptance
ReplayBundle admission accepts only the exact complete fn-21 qualified-violation set and rejects crossed, stale, partial, unsupported, identity-invalid, status-invalid, or multi-Property inputs before execution. ViolationSignature implements the spec's exact ordered projection, role table, equality-class normalization, and field-by-field oracle; every required semantic/causal mutation changes or invalidates it, every explicitly excluded transport mutation does not, and no caller can supply or override it. No fn-18 schema or persisted artifact family is added.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
