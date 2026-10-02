# C2 evidence correction for parent application

Update C2's spec status to **changed — callback identity instability reproduced; valid same-seed swapped prefixes succeed**. Keep the schedule-independent identity correction owned by task 5. The task 1 source finding remains valid: parentless callbacks consume the process-wide creation ordinal.

The final `timer_callback_identity` fixture's identifying select is the callback's first operation, using two channels prepared before either timer is armed. It performs no allocation, yield, child creation, or blocking operation before that site. The label is sent on a buffered channel immediately afterward. Only an adjacent Runnable selection matching a current parentless callback identity is associated; an unrecorded handoff produces no inferred association.

The final seed-6 parent associates A with ordinal 3 and B with ordinal 2. On the same seed, valid `BuildRankPrefix` alternatives at decision 3/rank 2 and decision 4/rank 1 associate A with ordinal 2 and B with ordinal 3. They succeed. All 16 alternatives of this one recorded parent succeed; this is not a global non-divergence claim. Prefix construction changes its final forced choice and drops the later suffix, so an alternative set created afterward has no retained suffix record against which to diverge.

A complete seed-6 prefix executed under seed 16 instead ends at a select-site divergence. This is a cross-seed forced-prefix experiment; normal replay binds the recorded seed. The earlier child probe also produced an alternative-set mismatch under cross-seed execution, but its callback-to-ID association was not sound because it yielded before its marker. Neither experiment proves a supported same-seed replay correctness failure.

For task 5, replace the approach bullet about the task-2 assertions with:

> Update task 2's characterization to require schedule-independent callback identities under the opposite firing schedules, including the valid same-seed prefixes that currently swap identities. Preserve those prefixes' successful execution; task 2 already established that all 16 alternatives of its seed-6 parent succeed. Do not describe this preservation check as fixing a reproduced same-seed alternative-set divergence. Cross-seed experiments are separate evidence and do not establish failure of supported seed-bound replay.

Add the same evidence correction to task 5's Key context. Its existing stable-ID and no-alternative-set-divergence acceptance requirements remain; the latter is a preservation check. No task closes solely because its stronger failure premise was narrowed.

Task 2's original alternative-set-divergence reproduction checkbox should close through its explicit changed-finding rule, referencing this correction and the retained final fixture evidence. E3 remains confirmed: all seven shapes emit one poll decision, six with fewer than two initially ready cases. The soundness of suppressing those decisions remains task 12's obligation.
