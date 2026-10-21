# Task 1 quality audit evidence

The reports below are retained verbatim, separated by axis and checkpoint. Counts are not merged or reranked. Auditor dispatches were pinned to gpt-6.1-sol at high by the installed quality-auditor role. They were read-only and did not execute builds or tests. These are same-family audits, not a cross-family review claim.

The original correctness finding and shared-discovery Consider were addressed before the fresh delta audits. The fresh correctness delta found three remaining admission defects. Actual separately collected RED controls and subsequent GREEN runs are recorded in the resolution reports. Formal implementation review and the three final full gates are separate requirements.

## Original correctness at fe98

## Quality Audit — Correctness axis: fn-231.1

Files changed: 38 · Critical: 0 · Should Fix: 1 · Consider: 0

### Should Fix

- **model/temporal/annotate/syntax/src/realize.rs:462** (Conf 75): `generated()` recognizes enums only by their original short name. Importing `ActivityExecutionStatus as Status` and matching `Status::Running` with `_` leaves both scrutinee and pattern positions unrecognized, so wildcard admission succeeds and future variants silently reach the fallback. This violates the exhaustive-enum contract. Resolve or explicitly admit enum aliases and add an alias-based compile-fail control. Evidence: walked through `generated`, `type_positions`, `positions`, and `visit_expr_match`.

### Evidence and limits

- Walked activity dispatch, raw Cancel codes, refusal precedence, explicit phase projection, partial callbacks/replies, decoded presence guards, and single envelope packing; found no runtime regression.
- Executed scoped diff inspection: Model, ABI/protocol, existing tests and goldens are unchanged. No builds/tests executed under this audit’s read-only restriction. No optional findings.

## Original standards at fe98

## Quality Audit — Standards axis: fn-231.1

### Summary

- Files changed: 38; fn-230 artifact excluded. Should Fix 0 · Consider 1 · Blocking: none possible (standards axis).

### Consider

- **model/temporal/annotate/syntax/src/realization.rs:460** (Conf 75): `catalog_inputs` duplicates the declaration-discovery policy in `model/temporal/api/build.rs:76`, creating two owners for admitted filenames and traversal (walked; duplication baseline) — expose one shared input enumerator.

### Out-of-axis observations

> Out-of-axis observation: **model/temporal/annotate/syntax/src/realize.rs:462** — `generated()` matches canonical final identifiers, so an enum imported as `Status` escapes wildcard detection, contrary to the parent’s exhaustive-generated-enum contract (walked; reproduction not executed).

### Evidence and remaining checks

- Cited `.flow/tmp/fn2311-integrated-native.log` and `fn2311-integrated-clippy.log`: passing compile controls, expansion snapshot, fixtures, boundary checks and clippy. Auditor ran no builds/tests.
- Post-edit live slice/gaps, full default/required gates and timing remain pending per dispatch; the full baseline is explicitly pre-edit.

### What’s Good

- Shared Catalog parsing and descriptor admission preserve the acyclic dependency boundary; no generic-core, Model, protocol, existing golden or third-party-version changes (walked).
- Reply packing is documented and literal outgoing bytes are pinned; task .2’s coverage golden and DESIGN obligations remain separate (cited).

## Fresh correctness delta at eddb

## Quality Audit — Correctness axis: fn-231.1 delta

Files changed: 19 · Critical: 0 · Should Fix: 3 · Consider: 0

### Should Fix

- **model/temporal/annotate/syntax/src/realize.rs:595** (Conf 75): Concrete wrapper aliases lose payload identity. `type Wrapped = Option<Status>` resolves to `Option`, but `type_arguments(Wrapped)` is empty; `match value { Some(_) => (), None => () }` passes admission. Preserve alias type arguments or refuse unresolved wrapper aliases. Evidence: walked.
- **model/temporal/annotate/syntax/src/realize.rs:705** (Conf 75): `match *value { _ => () }`, with `value: &Status`, passes because every unary expression produces empty positions. Propagate dereferenced enum identity or classify it unresolved. Evidence: walked.
- **model/temporal/annotate/syntax/src/realize.rs:787** (Conf 75): `let value; value = Status::Running; match value { _ => () }` passes because deferred initialization records empty positions and assignment never invalidates them. Initialize unproved bindings as Unknown. Evidence: walked.

### Original finding

The original direct `as Status` alias finding is resolved. Source resolution recognizes it, and the retained native log records the alias compile-fail control passing. Evidence: walked; cited `.flow/tmp/fn2311-combined-native.log:133`. Wrapper aliases remain incomplete.

### Evidence and limits

- Executed diff inspection at frozen `eddbacd8`; original report preserved. The last two findings are residual task defects also present before this delta. Existing assertions were retained.
- Cited native/clippy/guest logs show successful completion. No builds/tests executed by this audit; the three examples were walked, not reproduced. Live slice/gaps and full gates remain unestablished here. No shipping verdict issued.

## Fresh standards delta at eddb

## Quality Audit — Standards axis: fn-231.1 delta

### Summary

- Files changed: 19 · Should Fix 0 · Consider 0 · Blocking: none possible (standards axis).
- No standards findings in `fe98aede..eddbacd8`; frozen HEAD remained unchanged.

### Original finding resolution

- **model/temporal/annotate/syntax/src/realization.rs:155**: shared-scanner Consider resolved (walked). API generation, schema loading and macro freshness use one discovery policy. Sorted traversal retains every visited directory, including empty directories.

### Evidence and limits

- Walked bounded alias/source admission using `syn` and the existing descriptor graph. Source tokens serve inspection only; expansion retains the original impl. Workspace syntax build-dependency adds no third-party dependency/version.
- Cited saved combined native/clippy/guest logs: scanner, alias/source controls and unchanged `build_inputs` control pass; clippy and guest compilation finish. Auditor executed no builds/tests. Slice/gaps and formal post-edit gates remain pending per dispatch.

### What’s Good

- README states source-admission limits. No new lint suppression or filename-policy duplication; no file crosses 1,000 lines (`realize.rs`: 729 → 970).
- Test additions use a shared case runner. Delta leaves Model, phase helper, existing goldens and ABI/protocol files untouched. Unrelated dirty fn-233/MILESTONES state was preserved.

## Correctness resolution at 05f448d

## Quality Audit — Correctness axis: resolution delta

Scope: `eddbacd8..05f448d` · Files changed: 3 · Critical: 0 · Should Fix: 0 · Consider: 0

All three prior findings are resolved. No introduced blocker found in this fix.

### Resolution evidence

- **model/temporal/annotate/syntax/src/realize/source.rs:92**: Argument-bearing aliases become unresolved instead of losing payload arguments. Wrapper wildcard coverage now refuses. Evidence: walked.
- **model/temporal/annotate/syntax/src/realize.rs:703**: Dereferences propagate operand positions, preserving generated-enum identity through `*value`. Evidence: walked.
- **model/temporal/annotate/syntax/src/realize.rs:790**: Deferred bindings start Unknown; assignments refresh positions, with uncertain writes retained across control-flow scope restoration. The reported deferred-initialization bypass now refuses. Evidence: walked.

### Tests and limits

- Cited `.flow/tmp/fn2311-inference-audit-red.log:34`: six collected tests, three targeted failures and three passes. Cited `.flow/tmp/fn2311-inference-frozen-native.log:227`: all six pass; the same log records ten UI controls passing.
- Cited strict-clippy completion and empty fmt log; executed committed-diff inspection and `git diff --check`. No builds/tests executed by this auditor. Combined owner verification, post-fix guest comparison and full gates remain outside this resolution evidence.

### Test budget

Added lines: ≤78 test/control versus 43 implementation; existing assertions retained. Original raw report untouched. No shipping verdict issued.

## Standards resolution at 05f448d

## Quality Audit — Standards axis: fn-231.1 resolution delta

### Summary

- Files changed: 3 · Should Fix 0 · Consider 0 · Blocking: none possible (standards axis).
- No standards findings in committed `eddbacd8..05f448d`.

### What’s Good

- Walked focused changes that reuse existing scope/position analysis. Argument-bearing aliases become unresolved without adding generic substitution machinery; tests share one runner with existing assertions retained.
- Executed diff inspection found no new suppression, dependency, snapshot, Model or ABI/protocol changes. No size crossing: `realize.rs` 970 → 994; `source.rs` 437 → 444.

### Evidence and limits

- Cited saved RED evidence showing three new controls failing; frozen native log shows all six focused tests and ten UI controls passing. Strict clippy log finishes successfully; handoff claims fmt exit 0.
- Auditor ran no builds/tests. Combined five-package verification, refreshed guest comparison and formal/full post-edit gates remain pending per dispatch. Previous live proof covers the earlier guest.

## Custom-Deref correctness addendum at 05f448d

## Correctness addendum — custom `Deref`

New findings: Critical 0 · Should Fix 1 · Consider 0

- **model/temporal/annotate/syntax/src/realize.rs:703** (Conf 75): Forwarding operand positions through arbitrary `Deref` silently admits `match *value { _ => () }` when local `Wrapper` dereferences to generated `Status`. Preserve native-reference provenance; propagate only through a proved built-in dereference and classify other dereference results Unknown. Evidence: walked.

The complete path is visible: `source.rs:110` registers `Wrapper` as local; `source.rs:365` classifies it Other; `realize.rs:606` and `position` at line 29 yield empty positions. Lines 703–704 forward those positions, and the wildcard-only match adds no identity, so lines 827–837 emit no error.

For native `&Status`, forwarding preserves the generated identity. For `&LocalEnum`, allowing the wildcard is legitimate. However, `type_positions` at line 607 currently discards reference provenance, preventing the checker from distinguishing these cases from overloaded dereferencing.

The prior three concrete findings remain resolved. This is an additional residual soundness gap. Static inspection only; no builds, tests, or writes. Prior reports remain untouched.


## Correctness custom-Deref resolution at 22482ab

## Quality Audit — Correctness axis: custom-Deref resolution

Scope: `05f448d..22482ab` · Files changed: 5 · Critical: 0 · Should Fix: 0 · Consider: 0

The custom-Deref addendum is resolved. No introduced blocker found in this fix.

- **model/temporal/annotate/syntax/src/realize.rs:49**: `dereferenced` removes exactly one proved Reference marker; an unproved/custom dereference becomes Unknown, so the reported Wrapper wildcard refuses. Native `&Status` retains Generated identity; `&LocalEnum` retains Other. Evidence: walked.
- **model/temporal/annotate/syntax/src/realize/patterns.rs:12**: Match normalization strips reference markers from a copy. Stored bindings retain provenance, while borrowed Option patterns retain their coverage checks. Existing pattern helpers otherwise preserve their logic. Evidence: walked.

Cited retained logs show the custom-Deref RED with one failure/five passes, subsequent six-test GREEN, final native six-test GREEN, ten UI controls passing, and strict-clippy completion. Existing assertions remain intact.

Executed committed-diff inspection and `git diff --check`; no builds/tests executed by this auditor. Owner combined verification, refreshed guest comparison and full gates remain outside this pass. Prior reports remain untouched. No shipping verdict issued.

## Standards custom-Deref resolution at 22482ab

## Quality Audit — Standards axis: custom-Deref resolution

### Summary

- Files changed: 5 · Should Fix 0 · Consider 0 · Blocking: none possible (standards axis).
- No standards findings in committed `05f448d..22482ab`.

### What’s Good

- Walked cohesive extraction: private `patterns.rs` owns existing pattern traversal, path merging and wildcard checks. Reference provenance reuses `Position`; README states the custom-Deref limit.
- Executed diff inspection found additive controls, retained assertions and no new suppression, dependencies, UI pins, snapshots, Model/phase or ABI changes. Main file shrinks 994 → 910 lines; helper is 129.
- Executed byte comparison confirmed all seven original raw reports occur verbatim in the committed audit artifact.

### Evidence and limits

- Cited saved logs show focused RED with one failure/five passes, then six passes; exact-source native log records ten UI controls passing. Strict clippy finishes successfully; handoff claims fmt exit 0.
- Auditor ran no builds/tests. Owner five-package Quick verification, guest regeneration/comparison and formal/full gates remain pending per dispatch.
