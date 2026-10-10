# fn-152 planning handover

The append-log storage spec now has ten M-scoped task owners and coverage for
R1–R10. This is planning progress only. The spec remains open and all ten tasks
remain todo. No product code changed or implementation acceptance was granted.

Source input is `b341e60602d2fbebf2721bd9c64014c23b3e30b7`. The original capture
seals and all 22 reviewed spec/task file seals are in
[reviewed-file-bindings.json](reviewed-file-bindings.json).
The staged diff check caught one extra EOF blank line in the new spec. Removing
it preserved Flow's normalized review identity; the final file seals and the
metadata timestamp-only change are recorded alongside the reviewed seals.

## Review and design

Flow's Codex plan review returned `SHIP` on its second verdict in the same
reviewer session. The retained [receipt](plan-review-receipt.json) reports
`gpt-6.1-sol` at high effort, the same model family as the writer. Both original
P1 findings are fixed; the reviewer identified no remaining blocking,
duplication or structure findings.

The fix loop retained pinned private roots, guarded writable opens and lock/file
identity checks. It added the bounded admission/terminal-receipt owner before
Runner integration, preserving incremental refill, real buffered observations,
first-policy cancellation, budget-policy natural drain and committed cancellation
restoration. Stopping outcome and ordered drain commit atomically. Pending
receipt payloads stay live until adoption or retirement.

The corpus keeps its existing live budgets with separately growing history,
not a new churn-disabling history cap. Old storage layouts are rejected without
migration; global execution/runtime wire contracts stay unchanged. Shard merge
retains borrowed source ownership. No third-party dependency is planned.

## Verification and next work

The focused Flow coverage validation passed with ten tasks and no errors or
warnings. Whole-repository Flow validation passed with 26 specs, 242 tasks and
zero structural errors. Four existing coverage warnings remain in fn-104,
fn-107, fn-153 and fn-154; they are not excused by this plan.

Fresh document/diff checks passed. The tracked Gomad source diff and untracked
product-source inventory were empty. Protected unrelated dirty/untracked inputs
were checked against their preimages; MILESTONES adds only this spec's task table.
No Go, build, test, lint, generator or native execution ran for this planning pass.

Dependency waves are `.1/.2 → .3/.6 → .10 → .4 → .5 → .7 → .8 → .9`.
Use isolated worktrees for independent owners and serialize shared gates.
fn-155 remains first for implementation; its supported-platform proof is still
missing on the current linux/arm64 host. Existing source-red obligations and
fn-128/fn-149 native deferrals remain open. This plan grants no native revival,
CI, PR or push authority. New storage and built-CLI crash cases remain fn-152's
own acceptance rather than inheriting the older native transfer.
