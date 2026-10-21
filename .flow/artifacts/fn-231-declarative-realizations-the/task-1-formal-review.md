# Task 1 formal implementation review

Backend pin on actual dispatch: codex:gpt-6.1-sol:high, read-only sandbox, MAX_REVIEW_ITERATIONS=3. This is a same-family review, not a cross-family independence claim.

## Round 1

The first three-lens fan-out reviewed b9f26782e39f815d740ea4b89925a723303c861d..653fd40331fb5740940d6f91df8ad3c234ab9c4e. RID: 3d75c3007d804aa38e4111388291ee77. Dispatch began at 2026-10-21T11:35:00Z and returned exit 0 through host handle 35362. All three parsed draws returned NEEDS_WORK; none failed or was retried. Exit 0 means successful dispatch, not shipping approval.

The merge plan kept correctness:1 and correctness:2, collapsing contracts:1 and integration:1 into correctness:1. No finding was dropped by the evidence gate. The deterministic finalizer returned NEEDS_WORK with two introduced survivors, one consumed round and cap three. CLI help showed no --spec option on this deterministic, non-dispatch finalizer; actual model dispatches remain explicitly pinned.

The method group retains all distinct examples: map_err introducing a generated error, custom unwrap-like methods returning generated enums, and tuple payloads losing their Ok wrapper through unwrap. The second group covers Some(..) and tuple rests, including shifted prefix/suffix positions. All raw examples remain below. Cached-binary executions are not fresh builds; read-only-blocked compilation and boundary mutations are not passes. The worker's final full gates remain pending until SHIP.

Original review files remain in .flow/review-fanout/3d75c3007d804aa38e4111388291ee77. The copies below normalize Markdown hard-break trailing spaces only; the original files retain their exact bytes.

## Raw correctness draw

I’ll read the task specification first, then trace the changed parsing, validation, and code generation paths for correctness.

The activity dispatch appears to preserve the existing decode and refusal order. I’m now checking whether the enum wildcard checker preserves type identity correctly through wrappers and method calls; that is the main risk identified in the recorded verification evidence.

The retained focused test binaries pass: 6 parser/admission tests, 4 declarative runtime tests, 14 realization tests, and 5 helper tests. I found two gaps those controls do not cover: `map_err` can introduce a generated enum without updating the checker’s tracked type, and `Some(..)` can hide an enum payload without triggering a refusal.

1. **Severity:** P2
   **Confidence:** 100
   **Classification:** introduced
   **File:Line:** `model/temporal/annotate/syntax/src/realize.rs:622`
   **R-IDs:** [R1]
   **Problem:** Method calls named `map_err`, `unwrap`, `unwrap_or_else`, or `expect` inherit the receiver’s enum positions without accounting for their result type. For example:

   ```rust
   fn helper(value: Result<(), i32>) {
       let mapped = value.map_err(|_| Status::Running);
       match mapped {
           Ok(()) => (),
           Err(_) => (),
       }
   }
   ```

   With `Status` imported from an admitted generated enum, this passes admission: `Result<(), i32>` contributes no tracked enum positions, and `map_err` preserves that empty set despite introducing `Status`. Future generated variants can therefore reach the wildcard silently. Custom methods with these names create the same gap.

   **Suggestion:** Treat unproved method results as unresolved. Preserve positions only when the receiver identity and the operation’s type transformation are established. Add a refusal control for this example.

2. **Severity:** P2
   **Confidence:** 100
   **Classification:** introduced
   **File:Line:** `model/temporal/annotate/syntax/src/realize/patterns.rs:105`
   **R-IDs:** [R1]
   **Problem:** Rest patterns bypass wildcard detection. For `value: Option<Status>`, this is admitted:

   ```rust
   match value {
       Some(..) => (),
       None => (),
   }
   ```

   The checker tracks the generated payload at `Some[0]`, but visiting its `Pat::Rest` reaches `_ => None`. Tuple patterns such as `match value { (..) => () }` similarly hide generated enum positions. These matches accept future variants without naming them.

   **Suggestion:** Recognize rest patterns covering tracked positions as wildcards, including correct positional handling around tuple rests. Add refusal controls for `Some(..)` and tuple rests.

Retained focused binaries passed for parsing, expansion snapshots, runtime behavior, helpers, and native fixtures. The generic boundary scan passed. Fresh compilation and the boundary mutation control were blocked by the read-only filesystem; the findings above follow directly from code tracing.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | partial | Generated activity binding and regression controls pass; the two enum admission gaps remain. |
| R2 | deferred | Coverage golden explicitly belongs to task .2. |
| R3 | met | Model unchanged; generic boundary scan passes. |
| R4 | partial | Expansion snapshot passes; DESIGN section and line counts explicitly belong to task .2. |

Unaddressed R-IDs: [R1]

Classification counts: 2 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":2,"pre_existing":0},"unaddressed":["R1"]}
```

<verdict>NEEDS_WORK</verdict> - Fix the two introduced enum admission gaps.

## Raw contracts draw

I’ll read the task spec first, then review the diff and focused test evidence for agreement between the implementation and its stated contracts.
The activity rewrite keeps its projection mapping explicit, and the new metadata separates responses, callbacks, and outgoing requests. I’m checking whether the enum checker delivers its documented exhaustiveness guarantee, especially after wrapper operations such as `unwrap`.
The existing focused binaries passed: 6 parser/admission controls, 4 declarative controls, 14 realization tests, and 5 helper tests. I found a contract gap in the enum checker: it treats `map_err` and `unwrap` as preserving the receiver’s enum positions, though those operations can change the result type. I’m tracing concrete cases to pin down the impact.
- **Severity:** P2
- **Confidence:** 100
- **Classification:** introduced
- **File:Line:** `model/temporal/annotate/syntax/src/realize.rs:622`
- **R-IDs:** [R1]
- **Problem:** Method calls preserve receiver positions solely by name:

  ```rust
  "map_err" | "unwrap" | "unwrap_or_else" | "expect"
  // ...
  self.expression_positions(&call.receiver)
  ```

  This silently admits generated-enum wildcards. For example, with `value: Result<(), ()>`:

  ```rust
  match value.map_err(|_| Status::Running) {
      Ok(()) => (),
      Err(_) => (),
  }
  ```

  The receiver has no enum positions, so the transformed generated error remains unchecked. Adding upstream variants still compiles, contradicting the exhaustiveness contract. Custom methods named `unwrap` can likewise return a generated enum while inheriting an empty receiver classification.

- **Suggestion:** Model transformations only when receiver identity and result positions are proved; otherwise classify the result as Unknown. Add refusal controls for `map_err` introducing a generated error and custom methods returning generated enums.

Existing focused binaries passed 32 tests, including the expansion snapshot; the generic boundary scan passed. The boundary mutation control encountered the read-only filesystem. No fresh build or live suite was run.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | partial | Activity rewrite and runtime controls pass; enum admission bypass above remains. |
| R2 | deferred | Coverage golden belongs to task .2. |
| R3 | met | Model and generic runtime unchanged; generic boundary scan passed. |
| R4 | partial | Expansion snapshot passes; DESIGN.md portion belongs to task .2. |

Unaddressed R-IDs: []

Classification counts: 1 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":1,"pre_existing":0},"unaddressed":[]}
```

<verdict>NEEDS_WORK</verdict> - Fix the generated-enum admission bypass.

## Raw integration draw

I’ll read the task specification first, then trace the changes against their callers and shared types. I’ll keep verification focused on the named checks and any concrete issue I find.
The activity rewrite preserves the existing helper flow, including Cancel’s raw status codes and the partial SDK reply envelope. I’m now checking unsupported-field guards and enum exhaustiveness admission, where the macro relies on the existing descriptor graph and Rust source analysis.
The cached test binaries pass all six parser tests and all four generated-handler integration tests. I found a gap outside those controls: the enum checker preserves the receiver’s classification through methods named `unwrap`, `expect`, and `map_err`, even when their result has a different type. I’m tracing whether that can admit a forbidden wildcard.
- **Severity:** P2
- **Confidence:** 100
- **Classification:** introduced
- **File:Line:** `model/temporal/annotate/syntax/src/realize.rs:622`
- **R-IDs:** [R1]
- **Problem:** Method results inherit the receiver’s enum positions unchanged:

  ```rust
  "map_err" | "unwrap" | "unwrap_or_else" | "expect"
  // ...
  self.expression_positions(&call.receiver)
  ```

  For `value: Result<(ActivityExecutionStatus, u8), ()>`, this admits:

  ```rust
  match value.unwrap() {
      (_, _) => (),
  }
  ```

  The checker retains the path `[Ok, Tuple(0)]`, although `unwrap()` removes `Ok`. Tuple wildcard checking therefore misses the generated enum, allowing future variants to pass silently. Custom methods with these names can likewise return a generated enum while inheriting an empty classification from a local receiver.
- **Suggestion:** Propagate positions only through proven standard operations, applying their actual result shape. Treat unproved method results as Unknown. Add refusal controls for tuple unwrapping and a custom method returning a generated enum.

Verification: cached focused binaries passed 24 tests; `git diff --check` passed. These were not fresh builds. The finding follows directly from the position propagation and wildcard traversal.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | partial | Activity migration and regression controls are present; enum wildcard admission has the gap above. |
| R2 | deferred | Coverage golden belongs to task .2. |
| R3 | met | Model and generic runtime remain unchanged; boundary verification is recorded. |
| R4 | partial | Expansion snapshot is pinned; DESIGN.md work belongs to task .2. |

Unaddressed R-IDs: []

Classification counts: 1 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":1,"pre_existing":0},"unaddressed":[]}
```

<verdict>NEEDS_WORK</verdict>

## Round 2: SHIP

The fix was committed as f8df139c1cb23586972f4841ffb9358942cf74a1 after actual RED/GREEN and owner integrated Quick/guest verification. The single re-review used the same receipt and primary session 01a15e3e-49b1-78e3-a759-2146b749ca52, explicit codex:gpt-6.1-sol:high, read-only and MAX_REVIEW_ITERATIONS=3. Host handle 60250 returned exit 0. Reservation: 22a0416dbf6f460e86e41ff1d6cd4df5. Actual receipt timestamp: 2026-10-21T11:59:33.310803Z.

The actual receipt retains findings.round=2, supersedes the first receipt and marks both findings fixed. Its terminal verdict is SHIP with zero introduced, pre-existing or unaddressed findings. The CLI summary review_rounds=0 is not evidence of resetting the review cycle. Exact terminal receipt is archived in task-1-formal-ship-receipt.json. No new fan-out, forced review, model fallback or verdict refund occurred.

### Round-2 reviewer text

Prior finding #1: fixed
Prior finding #2: fixed

Method results now default to unresolved instead of inheriting receiver positions. Regression controls cover `map_err`, custom `unwrap`, and wrapper removal.

Rest patterns now cover omitted payload positions using tracked arity, with controls for `Some(..)`, tuple rests, and prefix/suffix alignment.

Re-ran retained focused binaries: all 26 tests passed, including the new controls, expansion snapshot, realization tests, and generic boundary scan. No fresh build ran in this read-only review. No remaining findings or new blockers.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | met | Both admission gaps fixed; focused regression controls pass. |
| R2 | deferred | Coverage golden belongs to task .2. |
| R3 | met | Model unchanged; generic boundary scan passes. |
| R4 | partial | Expansion snapshot passes; remaining DESIGN work deferred to task .2. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict> - Both prior findings are fixed; no blocking findings remain.
