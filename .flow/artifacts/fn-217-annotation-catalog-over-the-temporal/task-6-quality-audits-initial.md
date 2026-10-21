# fn-217.6 checkpoint quality audits

Pinned task range: `851c0e92a92637c5a2fef0fc073d83a206092670..f55725be9b1ea31908870e265ed3a2ae42c89209`.
These independent axis reports are preserved verbatim below. They are not the
formal implementation review and do not establish live/full-gate success.

## Standards report (verbatim)

## Quality Audit — Standards axis: fn-217.6

### Summary

- Pinned range: `851c0e92..f55725be` · Files changed: 119 · Should Fix 1 · Consider 0 · Blocking: none possible.

### Should Fix

- **model/temporal/realization/src/probes.rs:953** (Conf 100): File grows from 879 to 1,002 lines, crossing standards charter §9’s 1,000-line threshold; **executed** pinned line-count inspection — extract the reporting functions into a coherent sibling module.

### Evidence and limits

- **Walked:** derive and descriptor scanning share `DeriveCatalog::parse`; item-level presence, reviewed enum inventories, generated dispatch, and supplemental response linkage follow fn-217.6.
- **Executed:** pinned diff inspection and `git diff --check`; no builds or runtime tests run. Focused green checks are **claimed** by dispatch; live/full gates remain pending.

### Test Budget

- Path-based additions: 1,786 test-source lines versus 5,519 other source lines, excluding generated Go and compile fixtures; inline tests remain in the latter count. No >2:1 advisory signal.

### What’s Good

- Generic annotations retain system-free vocabulary; dependency changes reuse existing workspace packages.
- Removed protobuf fields are reserved, generated Go changes accompany the schema, and documentation reflects black-box probing and the single-crate inspection constraint.

## Correctness report (verbatim)

## Quality Audit — Correctness axis: pinned checkpoint 851c0e92..f55725be

Files changed: 119 · Critical 0 · Should Fix 1 · Consider 0. No shipping verdict.

### Should Fix

- **model/temporal/annotate/syntax/src/derive_expand.rs:294** (Conf 75, walked): Generated `.expects(...)` precedes `.repeated(...)`; expectations snapshot existing constraints, so valid request declarations such as `#[rule(max_items = 1, rejects)] values: Vec<Text>` produce `MissingExpectation("repeated.max_items")` when `catalog()` runs. The previous expansion ordered these correctly. Apply repeated rules before expectations and add a request regression covering repeated constraints.

### Evidence and limits

- Walked prediction completeness/index binding, rejection precedence, fallible setters, and Go dispatch/resource isolation. No builds, tests, or repository mutations performed.
- Existing repeated-field regression covers responses without expectations, leaving this request path uncovered. Actual live and full Model gate outcomes remain pending per dispatch.

Suppressed findings: 1 at anchor 50.

## Fix verification: correctness (verbatim)

No correctness findings in seven-file range `f55725be9b1ea31908870e265ed3a2ae42c89209..edd4829162610c9d840dd50e4cf5ef56b746208c`.

- **Walked:** `derive_expand.rs:294` applies `.repeated(...)` before `.expects(...)`, fixing both request admission and response expectation retention. The executable regression at `derive_catalog.rs:51` checks successful catalogs, exactly one `repeated.max_items` violation, and `RejectsCode(InvalidArgument)` for both directions. No existing assertions weakened.
- **Executed:** normalized SHA-256 comparison confirms all extracted classification functions are byte-identical, apart from `classify` visibility. Strict/generic classification, case binding, fail aggregation, dispositions, and report/privacy behavior are preserved.
- **Walked:** `Makefile:584` changes only the authorized aggregate timeout to 20 minutes. The 30-second operation bounds, complete-inventory checks, frame limits, classification assertions, golden comparison, and privacy canary remain.

No builds, tests, edits, or Flow operations performed. Worker-reported green suites remain **claimed** in this pass; the live gate’s final result remains outstanding. No shipping verdict.

## Fix verification: standards (verbatim)

## Quality Audit — Standards axis: audit fixes

- Files changed: 7 · Should Fix: 0 · Consider: 0.
- No findings in `f55725be9b1ea31908870e265ed3a2ae42c89209..edd4829162610c9d840dd50e4cf5ef56b746208c`.

Executed: pinned line counts confirm `probes.rs` shrank from 1,002 to 852 lines; `probes/report.rs` has 155. The prior standards charter §9 finding is resolved; that threshold is not an AGENTS rule.

Executed: extracted functions match the originals exactly after normalizing `classify` visibility. Walked: the private report module owns classification and rendering behind one `pub(super)` entry point, consistent with UMPIRE4_SPEC MOD-07 and adjacent submodule conventions.

Cited: the remaining changes are the accepted constraint-order fix/regression and the authorized aggregate timeout adjustment with its exact task scope. No unrelated changes appear in the pinned diff.

Limits: no builds, tests, writes, or shipping verdict. Focused suite/Clippy/fmt results are claimed by dispatch; live/full results remain pending. Foreign worktree edits were excluded.
