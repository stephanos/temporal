# fn-153 planning handover

Gomad maintainers now have 18 task owners covering R1 and R2 for JSON and
whole-file publication cleanup. Sizes are two S and sixteen M. The spec remains
open and all tasks remain todo. This checkpoint grants no implementation or
native acceptance and changes no product source.

The source baseline is `0bff0019baf1753cb04fedf9672ace0b5f00dcf7`.
[Reviewed and final file bindings](reviewed-file-bindings.json) retain the
original capture seals, 38 reviewed spec/task seals, final seals and 52 protected
preimages. The only post-review changes remove one spec EOF blank line and
update its metadata timestamp. Flow's normalized review text is unchanged.

## Review and probe

The independent Codex plan review returned `SHIP` in its second verdict in the
same reviewer session. Its [receipt](plan-review-receipt.json) reports
`gpt-6.1-sol` at high effort, the same model family as the writer. Both introduced
findings are fixed. Task 17 now stages all twelve matching approval requests
before complete generation/check, retaining stale-approval refusal and live
preimages on preparation failure. The identified consumer test migrations now
have declared owners. No blocking or maintainability findings remain in the
retained review.

The disposable [World probe](world-probe.md) matched 5,029 byte comparisons,
covered 76 serialized fields and checked seven complete recording frames.
Seven invalid-UTF-8 controls exposed rejection supplied by the old encoder;
the planned typed validators preserve that rejection before lossy Marshal.
[Evidence](world-probe-evidence.json), [source](world-probe-test.go.txt) and
[raw output](world-probe-output.txt) retain the command, compiler/input hashes,
corpus counts and result. The probe used ordinary Go 1.27.1 on linux/arm64.
Its finite corpus supplies no patched-runtime, soak, native qualification,
performance or full-host acceptance. No probe rerun was needed after the
planning-only review fixes.

## Verification and execution

Focused Flow validation passed with 18 tasks, complete R1/R2 coverage and zero
errors or warnings. Whole-Flow validation passed with 26 specs, 260 tasks and
zero structural errors. Three existing coverage warnings remain in fn-104,
fn-107 and the unplanned fn-154. Current source inspection found 49 test files
mentioning canonicaljson, all covered by the tasks' declared Touches.

Fresh diff checks passed. Tracked and untracked Gomad product-source inventories
were empty. The 52 protected inputs retain their preimages; removing the new
18-row MILESTONES table restores the original working document. Only that
table is selected for staging. No product Go, build, lint, generator or native
gate ran for this planning checkpoint; the disposable probe is separate evidence.

Dependency waves are `.1/.2 → .3/.4/.5/.6/.13 → .7/.10/.11/.14 → .8/.12 → .9 →
.15 → .16 → .17 → .18`. These are parallel candidates; isolate independent
owners and serialize shared source, Go, lint and generator gates. Fn-152 remains
the sole spec prerequisite. Reinventory its landed storage before implementation.

Fn-155 remains the next implementation priority. Its first supported-platform
execution proof remains open on this linux/arm64 host. Existing source-red
obligations and fn-128/fn-149 native deferrals remain open. This checkpoint
grants no native revival, CI dispatch, PR or push authority. Fn-153's new checks
retain their own acceptance rather than inheriting the older native transfer.
