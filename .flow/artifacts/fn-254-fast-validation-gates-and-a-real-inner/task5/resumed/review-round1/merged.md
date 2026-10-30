Coordinator merge of three SHIP draws; duplicates collapsed, malformed multi-location anchors normalized. All four follow-ups will be addressed before closure.

- **Severity**: P2
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `model/umpire/dev/xtask/src/test_run.rs:25`
- **R-IDs**: [R4]
- **Problem**: Required configure unconditionally overrides an inherited lower nextest thread allocation with four; preserve a valid lower allocation in both workspaces and truthful standalone receipts.
- **Suggestion**: Compute the minimum of inherited positive thread count and policy limit and test lower/higher/default values.

- **Severity**: P2
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `model/umpire/author/dsl/macros/tests/fixtures.rs:40`
- **R-IDs**: [R5]
- **Problem**: The explicit eighteen-file UI list silently omits future files added under tests/ui, unlike the former glob.
- **Suggestion**: Restore compile_fail("tests/ui/*.rs"); all converted parser inputs already live elsewhere.

- **Severity**: P2
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `model/umpire/dev/testing/src/runner.rs:171`
- **R-IDs**: []
- **Problem**: Unconditional prevalidation of existing golden text prevents update mode from repairing corrupt consumer-owned goldens, which snapshot previously allowed.
- **Suggestion**: Preserve full sibling admission in normal mode and permit INSTA_UPDATE=always to repair selected consumer-owned snapshots; cover the actual repair route.

- **Severity**: P3
- **Confidence**: 100
- **Classification**: introduced
- **File:Line**: `model/dev/xtask/src/gate.rs:13`
- **R-IDs**: []
- **Problem**: Temporal test commands apply identical policy in both test_run::run and stage; only one application is needed.
- **Suggestion**: Keep the tested stage hook and remove duplicate test_run configuration.

Unaddressed R-IDs: []

```json
{"classification_counts":{"introduced":4,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>
