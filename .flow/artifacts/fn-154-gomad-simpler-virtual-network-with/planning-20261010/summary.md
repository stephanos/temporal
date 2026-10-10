# Reviewed virtual network plan

Gomad now has a reviewed 12-task plan for byte-bounded streams, partition holding, FIFO healing and autonomous virtual stall expiry. All tasks remain todo. Fn-155 stays first for implementation.

## Review and corrections

Codex gpt-6.1-sol at high returned SHIP in round 2, using session 01a1262e-bc4c-78b0-acb0-22d15a4d2b4a. The writer and reviewer belong to the same model family. The first review identified two P1 findings. The final receipt marks both fixed and reports no blocking findings.

The plan reserves bounded terminal history capacity before admitting stalled connections and dials. Competing operations cannot consume those reservations. Expiry consumes its reserved slot, and cleanup or effective heal releases obsolete reservations. Tasks 5-7 own process Runner registration and execution before their both-backend acceptance. Task 10 audits the completed selection.

The source audit also moves closed record/codec vocabulary admission ahead of held/timeout producers, adds strict process error-code generation to task 3, and places composed-profile replay coverage with the isolated execution fixtures. Task 3 proves transport identity; task 6 proves actual autonomous expiry.

## Delivery order

Tasks 1 and 3 have disjoint source ownership and may run concurrently in isolated worktrees. Their shared compiler, generator and qualification lane remains serialized. Task 2 waits for both. Tasks 4 through 12 then follow dependency order. The provisional serial graph in gap-design.md is superseded by admission-audit.md and the canonical task dependencies.

Task 2 requires the source-accepted fn-109.14 semantic codec seam. The plan consumes the current fn-109.15/.16 lifecycle and fn-155 descriptor interfaces without adding a separate clock or scheduler. The module-design analysis selected one shared effectful local connection owner in gomadio, retaining process and descriptor adapters.

The direct network-only record deliberately loses same-length payload comparison when network hashes disappear. Composed Runner I/O checks retain their existing payload-divergence guarantees. The plan preserves caller deadline recovery, persistent stall timeout identity, explicit fault matching and bounded metadata/history.

## Verification and preservation

Flow validation reports 26 specs, 272 tasks, zero errors and two preexisting warnings. Fn-154 has 12 tasks, zero warnings and coverage for all 12 stable requirements. No product source changed, and no Go test, native execution, lint or generator gate ran for this planning checkpoint.

The 25 reviewed task/body files match their pre-dispatch SHA256 seals. Only parent spec JSON changed through the review handler. All 95 protected preexisting paths match their baseline hashes. Removing the new fn-154 table from MILESTONES.md reconstructs the pre-checkpoint file, including the three existing blocked-status edits. Those edits remain outside this commit.

Supported-platform execution remains required for fn-155.1 and for fn-154's future native gates. This linux/arm64 session supplies no such qualification. Native fn-128 and fn-149 remain deferred. This checkpoint claims no task completion, implementation acceptance, push, PR or CI authority.
