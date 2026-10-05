# Task 40 checked command cleanup

Task 40's inherited lint correction is ready for a source-progress commit. Actual unfiltered pinned lint drops from five findings to zero. Baseline focused tests pass 56/56 and final worker and conductor tests pass 57/57, with no failures or skips. The added infrastructure/stderr seam control passed before production changes.

The four checked pipe defers keep their registration points and LIFO lifetimes. Nil cleanup preserves primary error identity and the complete Result. A sole cleanup error stays direct; simultaneous nonnil errors join primary first. Only deferred writers ignore their expected repeated-close os.ErrClosed. Explicit post-Start closes, startup/cancellation, capture, wait and group cleanup operations remain intact. Genuine previously discarded deferred-close failures become infrastructure failures, the sole admitted default-error correction.

The process-death helper keeps its immediate ESRCH-only probe, absolute two-second deadline and exact assertion while stopping its 10 ms ticker and deadline timer on return. Every original fixture byte outside that helper and the one added seam control remains unchanged.

[Evidence](evidence.json) retains literal commands, exits, counts and elapsed times. Architecture, errortype, formatting, race (16 results), check-only generator validation and six available consumer groups (166 results) pass. The [source freeze](source-freeze.sha256) and [tool/config freeze](tool-config.sha256) bind the candidate. Run their checksum checks from tools/gomad3. The conductor's initial wrong-directory checksum invocation failed; its corrected module-directory invocation matched every checksum before acceptance.

Both fresh [source reviews](reviews.md) find zero introduced findings. The plan review's SHIP covers the amendment only. Neither source review is a formal implementation verdict.

Original predecessor/matched-first-baseline, complete patched/full/default/functional/smoke/affected-native gates, native Darwin qualification, static full both-source-set and exact-input adapter regeneration and formal implementation review remain unproved. Actual deferred OS-close faults and simultaneous cleanup failures remain unexecuted. The seam control supplies no such proof. Task 40 stays blocked; task9/task21/fn113 retain their original dependency and aggregate ownership. Transferred Linux execution remains fn128-owned and nonblocking. [Earlier command progress](../command-compatibility-2026-10-05/progress.md) remains historical and unchanged.

stage: source-review - ran (correctness and standards each zero introduced findings)
stage: impl-review - skipped(policy: original required qualification gates remain unproved)
Tracker sync: n/a (bridge inactive)
