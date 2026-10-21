# Final root gate verification

Final code checkpoint: `8f59fcf4e85cb7fc7cf6f9f519338f5bee4c1885`.
Default, required-backend and uncompromised full live gates pass. Timing and
formal implementation review are separate worker-owned gates and are not
claimed by this artifact. Earlier failed checkpoints below remain explicit.

Golden checkpoint: `424f6eb7a7b8c160b7270c8ce21b639aa3aedaea`.
Activity gaps golden SHA-256:
`a3974ea8da0f3c46234770068fe0f163d5e17529b275cea086c2292a124284a0`.
Root verified it byte-identical by SHA-256 to the immutable corrected live report.

## First default gate: failed

Command: `CARGO_BUILD_JOBS=4 CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2177-rename GOCACHE=/home/agent/.cache/codex-build/fn2177-go make model-check`.
Root session 71089 exited 2. Log: `fn2176-root-final-model-check-default.log`.

Two native catalog controls failed: `failure_activity_type_does_not_inherit_start_request_constraints`
and `worker_failure_requires_application_info_only_when_failure_is_supplied`.
Root and worker independently traced both to FailedRequest fixtures omitting
task_token, which is now correctly required. Adjacent fixtures already supply an
opaque nonempty test token. Root required preserving the catalog and existing
exact assertions, supplying the required token in those two baselines and
pinning the negative failure-kind finding specifically.

Root authorized that native-test-only correction while the independent final
live session 51904 continued, provided no production/guest/golden source changed.
The subsequent sections record guest identity, source equivalence and renewed
full gates at the later test-fixture checkpoint.
The required-backend gate was not started on the red checkpoint.

Baseline receipt was not honored: `gate check --gate unittest --command 'make model-check'`
reported dirty MILESTONES.md at 424f6eb7. Foreign plans and milestone changes were
not hidden to manufacture a clean receipt. That failed checkpoint did not
establish final gate success.

## Native fixture correction

Checkpoint: `616d378223cd29b38af4771f11fa126542377406`.
Root inspected its complete diff from 424f: only
`model/temporal/realization/tests/catalog.rs` changed. Both affected requests now
supply an opaque nonempty token. Positive and byte-preservation assertions remain;
the application-info negative now demands exactly one finding naming Failure
and application_failure_info. Focused log `fn2176-native-fixture-catalog-green.log`
ends with all ten catalog tests passing, zero failures.

No production source, guest, protocol or golden changed across that commit.
Before renewed full gates, root hashed the existing Temporal Wasm used by the
ongoing live run:
`b5e21ee94a519d995f4447b2f6d2159d65c92b650c762044e57e579ac1ba09dc`.
Path: `/home/agent/.cache/codex-build/fn2177-rename/wasm32-wasip1/release/umpire_temporal_guest.wasm`.
Default gate rerun session 53087 uses the same full command and cache leases;
log `fn2176-root-final-model-check-default-fixtures-corrected.log`.
Receipt reuse was again refused for dirty MILESTONES.md at 616d3782.

Session 53087 exited 2 at Clippy, before full tests: two newly added assertion
messages at catalog.rs:405/406 violate uninlined_format_args. This is a test-code
formatting failure, not a passing gate. Its log is preserved. The worker then
inlined the formatting arguments, passed focused Clippy and catalog tests, and
committed the next checkpoint before the next full gate run. Production/guest
and the golden remained unchanged; the independent live run continued.

## Corrected final verification checkpoint

Checkpoint: `8f59fcf4e85cb7fc7cf6f9f519338f5bee4c1885`.
The added commit changes only the two assertion-format arguments in the same
native test file. Root inspected the complete 424f..8f59 path set: that native
test is the sole changed file. Focused fmt-check, Clippy and all ten catalog
tests pass (`fn2176-native-fixture-focused-lint.log`).

Default full gate session 31571, log
`fn2176-root-final-model-check-default-lint-corrected.log`, is the renewed run.
Receipt reuse was refused for dirty MILESTONES.md at 8f59fcf4. During the renewed
gate, root rehashed the Temporal guest and verified it still matches the
live run's recorded b5e21ee9 SHA above. Thus the fixture-only commits did not
change the actual live artifact. Full terminal results are recorded below.

## Default full gate: passed

Root session 31571 exited 0 at 8f59fcf4. Same full command as above; log
`fn2176-root-final-model-check-default-lint-corrected.log`. Generic Go host
0.920 seconds, Temporal Go host 59.483 seconds. This default run explicitly
skips TLC; it is not evidence of backend execution. After terminal success,
the Temporal Wasm still has the exact recorded b5e21ee9 SHA.
Root attempted `gate receipt --gate unittest --command 'make model-check'`;
the writer returned `written=false`: dirty MILESTONES.md outside the ignore set,
receipt not warrantable. The full passing command remains execution evidence,
not a reusable clean-worktree receipt.

## Final uncompromised full live gate: passed

Worker session 51904 exited 0; root independently inspected the complete log
`fn2176-final-exact-golden-live.log`, terminal PASS and actual final report.
Package time 485.060 seconds. Both required top-level tests pass, including
the privacy logger control. No timeout or failure marker appears.

Root matched every one of the 2,317 strict/generic report identities to its
final PASS subtest: zero missing. The log records 2,336 PASS subtests total,
the corpus plus nineteen namespace and scope-boundary controls. The report
contains 79/79 strict confirmations, 754 tested generic observations and
1,484 explicit local untested outcomes (NotApplicable, Unchanged or Refused).
All 217 scoped Unannotated findings remain
visible; no Mismatch or unexpected acceptance appears.

Final actual report SHA-256 matches the committed golden exactly:
`a3974ea8da0f3c46234770068fe0f163d5e17529b275cea086c2292a124284a0`.
Inventory/CLASSIFY-request/response sizes: 692,785 / 503,633 / 879,201 bytes,
below the unchanged 1 MiB cap. Guest identity and production equivalence
across the two native-fixture commits were verified above; no live restart.

The worker's live receipt attempt was refused for dirty MILESTONES.md
(`written=false`); there is no clean-worktree smoke receipt.

## Required-backend full gate: passed

Root session 24704 ran
`UMPIRE_BACKENDS=require CARGO_BUILD_JOBS=4 CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2177-rename GOCACHE=/home/agent/.cache/codex-build/fn2177-go make model-check`
at the same 8f59fcf4 checkpoint. Log:
`fn2176-root-final-model-check-required.log`. The real-TLC-required banner
is present; the full command exited 0. Generic Go host 0.978 seconds; Temporal
Go host 57.870 seconds. Root confirmed HEAD remained 8f59fcf4 and the guest still
matched the recorded b5e21ee9 SHA after terminal success.

Receipt reuse was refused for dirty MILESTONES.md, so this was an actual full
rerun. Root then attempted the required gate's receipt writer; it returned
`written=false`, dirty MILESTONES.md outside the ignore set, receipt not
warrantable. Neither full-gate success is a reusable clean-worktree receipt.
No source changes or skipped stages were hidden to manufacture one.

All root build commands are terminal. Worker timing may run without contention
from root builds, followed by its configured formal implementation review.
