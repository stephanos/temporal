# Early World terminal source review

The independent world_terminal_early_review identified one Important/P2
preservation defect in the initial World correction. No Critical or Minor
finding. The conductor forwarded it to the sole writer before final freeze.
This is early source review, not formal SHIP, frozen acceptance or native
qualification. Full preservation tests and docs were known writer work in
progress and were not counted as separate findings.

## Strengths

errors.go's closed projection rejects unknown and typed-nil inputs without
arbitrary callbacks. FinishTerminal restricts input to the three explicit error
kinds; existing validation requires nonempty detail without new bounds, while
Finish retains quiescence inference. Process Session retains validation before
callbacks, detail before classification, capacity/replay/invalid category order,
unknown %w identity and cleanup position. Public default sentinel variables
remain statically error; their private concrete implementation changes follow
the admitted migration.

## Important: preserve World-generated transition context errors

In the inspected replay.go:378, EncodeTransitions still wraps transition-shape
validation errors with fmt.Errorf. EncodeTransitions([]Transition{{}}) produces
"transition 0: invalid World snapshot: transition.shape", retaining the
ErrInvalidSnapshot cause. The old Recorder.FinishError classified it as invalid
input; the new closed ownedTerminal rejects that fmt.wrapError.

This wrapper originates inside World, not at an external caller, so it is
outside the admitted custom/external-wrapper migration. Use the owned
modelContextError constructor with the same transition-index prefix, preserving
the complete message, cause/errors.Is, terminal kind and recording bytes.
Require a real public-operation preservation fixture on old/new source.

## Inspected snapshot and limits

The reviewer found these seven selected production hashes unchanged before
and after inspection; only prefixes were returned:

```text
world/errors.go          867f9e3e3119
world/world.go           5de21e439b3f
world/snapshot.go        79c957ffd487
world/replay.go          9441eae3eb3f
world/recording.go       a5f32ef90e12
world/codec.go           74d895e31843
world/process/session.go be2f03fdd91f
```

process/session_terminal_test.go appeared during inspection; its reported SHA
was 87ff6fcff22795af6e2db25160313581f608a1d12a3031a96616175854f7d6e0.
The reviewer read the available tests and acknowledged that tests/docs were
still being expanded. Source can change after this review; these findings are
snapshot-specific and must be reconciled before final candidate acceptance.

Review compared relevant uncommitted changes at base/HEAD
0dd05b313acd0986312da7fd3159520e6a21f1bf, not the empty HEAD..HEAD range.
Tier: session (jev-unavailable(no_key)), resolved once for this bounded review.
Requested gpt-6.1-sol/high, same family as the writer; actual execution metadata
unobservable. The reviewer inspected source/hashes only; no writes, tests,
builds, generation, package loading, Flow/Git mutations, bridges or subagents.
Existing RED/GREEN logs were reported-run evidence, not reviewer executions.
