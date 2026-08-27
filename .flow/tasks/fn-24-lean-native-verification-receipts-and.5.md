---
satisfies: [R5]
---
# fn-24-lean-native-verification-receipts-and.5 Expose umpire-check-model list explain and named profiles

## Description
### Umpire4 reconciliation (normative)

The public command is `umpire-check-model`: default per-commit profile, `--profile nightly`, `--check <name>`, `list`, and `explain <name>`. The CLI selects or tightens model-declared checks only; it never authors targets, properties, Limits, or trust claims.

The legacy implementation detail below is retained for context but is subordinate to this reconciliation.

Add `Temporal.Tool.Verify` with the exact one-entry production registry and the non-default `temporal-model-verify` Lean executable, then expose it through root `make umpire-verify TARGET=...`. Keep semantic work in `CallerClosureFormal`; the tool only resolves the static target, renders the existing receipt, and applies the exact stdout/stderr/status contract. Reject all extra arguments, overrides, unknown targets, and output destinations. Preserve existing comments and put every Make change in the repository-root Makefile.

**Size:** M
**Files:** `model/Temporal/Tool/Verify.lean`, `model/Temporal/Tool/VerifyTests.lean`, `model/lakefile.toml`, `Makefile`
**Touches:** [model/Temporal/Tool/Verify.lean, model/Temporal/Tool/VerifyTests.lean, model/lakefile.toml, Makefile]
## Acceptance
The sole registered identity emits the exact fixture plus LF and status 0; missing/extra/unknown input and registry/serialization failures follow the exact error envelope with empty stdout. Output is preconstructed and attempted once; a simulated short write proves status 1, the final error line, and the explicitly indeterminate stdout prefix contract. Fixture-driven harness tests also pin status mapping for valid violated/unknown/unsupported/invalid receipts without registering the negative control. Target-graph and checkout snapshots prove the opt-in target is absent from default Lake/build/regression/CI/runtime/production paths and performs no repository write.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
