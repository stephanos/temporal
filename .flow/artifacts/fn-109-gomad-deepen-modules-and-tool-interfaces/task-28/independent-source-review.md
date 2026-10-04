# Task 28 independent source review

Ready for a source progress commit only. I found no actionable introduced Critical,
Important or Minor issue in the seven-file campaign working diff.
Actual affected-package lint remains red with two unchanged invariant findings.
This manual review supplies no formal SHIP verdict or task completion.

BASE and HEAD are `d7c6695cff81a1160ee482cb82b62cf592f4f199` on `gomad`.
The committed range is empty. The review covers the frozen working source rather
than an empty committed diff. Root owns Git, index, Flow and milestone updates.

Requested reviewer preference is `gpt-6.1-sol` at high, the same configured
family as the writer. Root supplied the once-resolved session tier
`jev-unavailable(no_key)`. Actual executing model metadata is unavailable.
I did not repeat judge resolution, bridge, delegate or change source.

## Source assessment

The constructor keeps source/positive-parallelism validation, budget-zero
validation, then unknown-policy rejection before copying every initial counter.
Its direct predicates stop first-policy admission for existing failures and stop
budget admission at or above the distinct threshold. All preserves admission.
The new literal constructor table checks complete statistics, source-call counts,
active/Done/stopped state and below/at/above budget cases.

Complete retains both exact panic literals and both checks before the first
mutation. The entire prefix through active decrement, attempted/classification
accounting, watchdog/divergence counters, distinct-failure assignment and the
already-stopped guard matches BASE byte-for-byte. First returns cancellation once;
budget stops admission and returns false so active work drains. All returns false.
Existing completion vectors and zero-mutation rejection tests remain unchanged.
`controller_completion_test.go` is byte-identical to BASE, and removing only the
new constructor test restores the original `controller_test.go` exactly.

All thirteen cleanup findings are checked at their original lifetime.
Eight journal/resumed closes in `campaign_journal_test.go` retain their deferred
registration positions and report nonnil errors with `t.Error`.
The ninth deferred journal close in `publishMergeShard` runs before helper return,
including fatal paths. It is not delayed until test cleanup.
The torn-tail Write failure closes once, reports a nonnil Close error separately,
then calls the original fatal with the Write error. The success Close and literal
torn fragment are unchanged. Normalizing only those checked closures restores
the complete merge-helper and torn-tail test files to BASE.

`OpenCampaign`, `RecordPlan` and `ReadResumePlan` name their error result and
conditionally join `errors.Join(retErr, closeErr)` only for nonnil Close errors.
Nil Close errors leave the original error object, text, Unwrap shape and data
untouched. Named result labels do not change function types. Removing only those
labels and checked root closures restores both production files exactly to BASE,
proving validation, publication and config-update statement order is unchanged.

The strengthened primary-error characterizations cover existing-plan refusal
without wrapping, the original direct `*IntegrityError` on a changed published
segment, and exact two-cause prepared-target errors in order with zero rejected
data. These pass on the original production source before implementation.
The retained first characterization failure expected too few causes.
`files.go:44` checks target size and returns the existing metadata cause through
`hashValidatedFile`; its source supports the corrected literal. That failed receipt
is assertion calibration, not a runtime RED proof. The actual source-policy RED
is the unfiltered pinned analyzer baseline.

## Fresh verification

All commands use the cached stock Go 1.27.1 and pinned local analyzers on
developmental linux/arm64, with `GOWORK=off GOTOOLCHAIN=local GOPROXY=off`,
cleared seed variables, `-count=1` and `test_dep`.
The unchanged gitroot lint configuration and `--fix=false` are bound by SHA-256.
[Checks](independent-source-review-checks.json) and the referenced `review-*.json`
receipts retain exact commands, cwd, environment, times, exits and log/source hashes.
The capture wrapper's zero exit is not a child-command success claim.

| Fresh gate | Child exit | Seconds |
| --- | --- | --- |
| Entire ordinary campaign package | 0 | 6.280 |
| Focused controller/resume/cancel/plan/publication/torn-tail/rejection tests, 51 top-level tests | 0 | 3.619 |
| Unfiltered pinned campaign golangci-lint | 1 | 0.759 |
| Pinned campaign errortype, no diagnostics | 0 | 1.394 |
| Root architecture, public-alias and external Runner consumer checks | 0 | 4.989 |
| Scoped diff check and pinned gofmt | 0 | 0.027 |

I recomputed diagnostics from the raw baseline and final logs and verified their
source lines. Baseline has 13 errcheck, 2 exhaustive and 2 forbidigo diagnostics.
Final and fresh output have exactly the original 2 forbidigo diagnostics at
`controller.go:129` and `:132`. The mapped delta is 15 to 0, introduced 0.
The intermediate log contains two QF1003 staticcheck findings; compound predicates
remove them without changing a rule, pin, configuration or golden expectation.

I independently verified all 13 selected worker receipts against their log hashes,
actual tools, lint configuration, stable source snapshots and command/environment
bindings. All 30 baseline campaign hashes match BASE; all final campaign and
frozen-boundary snapshots match current source. Both protected receipts and current
selection of 1,007 tracked noncampaign inputs have aggregate
`310442e1a38ecfd7505423c495f499a1d2e32c9b73144a677d1e64f118881c9a`.
The audit uses the wrapper's sorted-key JSON encoding with ordinary separator spaces.
[The audit command record](review-audit.json) contains the reproduction commands,
without another bulk path inventory.

Original task 3 retains SHA-256
`a057f2e4163c878e86a8357db9f5e3d0bd2a54c69c5f18bf24a912a940201406`;
Runner retains
`bdf6d21c8e09713446a803bc112b48db4eb50718827129bd53d4617ce4ff606b`.
Both match BASE. The parent's original API/criteria/boundary prefix is intact.

The worker's `make validate` receipt is verified and exits 0. Current source
differs from that receipt only in the final controller predicate refinement.
I inspected Makefile VERSION_INPUTS, BOUNDARY_INPUTS and COMPATIBILITY_INPUTS;
campaign predicates are outside them. Protected runtime, protocol, overlay,
generated, pin, dependency and build-key inputs are unchanged, so the retained
validation result remains relevant. Unchanged broad rootfast/full/native gates
were not rerun.

## Residual obligations

Pinned `os/root.go:96` delegates to `root_openat.go:32`, which always returns nil
after its Unix close handling. I verified both pinned source hashes and inspected
that implementation. These functions provide no real nonnil-root-close injection.
The join order is proven by source inspection only; there is no executed dual-error
root-close proof. Adding a fabricated seam is outside this repair.

The patched `.toolchain/bin/go` is absent. Developmental stock-host gates do not
qualify actual patched runtime or native darwin/arm64 and linux/amd64 behavior.
Original R16/R18/R19, task 3, predecessors, task 21, first-baseline fixed-identity
evidence, full/formal gates and both native qualifications remain open wherever
unproved. The two invariant findings remain open without suppression or a
completion-error redesign. The broader 419-finding receipt remains historical;
package progress supplies no new whole-scope count.

Root may commit the verified source progress before another writer. Source, index,
HEAD and Flow lifecycle writes by this reviewer are zero. All owned commands are
terminal; delegates, live handles and pending commands are zero.
