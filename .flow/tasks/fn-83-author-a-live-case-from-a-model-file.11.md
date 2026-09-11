---
satisfies: [R11]
---
# fn-83-author-a-live-case-from-a-model-file.11 Author Known Gaps in the Model file instead of hard-coding them

## Description
Let a Model file declare its own Known Gaps in command syntax and delete the hard-coded ones (R11). Today `Authoring.check` attaches `completionKnownGaps` (`temporal.nexus.success.known-gap.cancellation` and `…operation-correlated-progress`) to every Query of every Model. After .5 and .6 the worker-outage and sync-Nexus Cases would carry Nexus cancellation gaps unrelated to them, and .6's acceptance requires a Known Gap in the sync-Nexus fixture's Provenance authored with "only the six commands", which no command can express.

**Size:** S
**Files:** the Umpire `query` command and authoring core from .10, `model/Temporal/Feature/Nexus/Success/Model.lean` (declares the two gaps it carries today), the command tests (`#guard_msgs` pins), `.flow/tasks/fn-83-author-a-live-case-from-a-model-file.6.md` only if the white-box decision below changes its wording
**Touches:** [model/Umpire/**, model/Temporal/Feature/Nexus/Success/**]

### Approach
- Grammar is a task decision. Known Gaps are Query-authored today (`Query.authoredKnownGaps`), so a repeated line on the `query` command is the default, for example `gap capability cancellation "Operation-correlated Nexus cancellation is unsupported by the success slice."`, with an optional subject. The code derives from the definition family (`<family>.known-gap.<name>`), which reproduces today's codes for the Nexus success Model.
- Kinds resolve against `Umpire.KnownGapKind` (`capability`, `input`, `interpretation`, `claim`), listing them on an unknown spelling. There is no white-box kind. .6 needs one for the admin-service mutable-state assertion: decide between representing it as an existing kind with a code and detail, or adding a kind (Lean enum only; Known Gaps reach the Case through Provenance, so no proto change). Record the decision; if it adds a kind, check whether `UMPIRE4_SPEC.md` enumerates kinds normatively and note the amendment rather than widening fn-83's spec edits silently.
- A subject names a declared Property by reference where one exists. Today's subject, `temporal.nexus.success.property.cancellationResolves`, names a Property no Model file declares; decide whether the success Model keeps a spelled subject or drops it, and record the fixture consequence.
- Canonicalize through `KnownGapSet.checkCanonical` as today; a duplicate code rejects located.
- Byte pin: the async-Nexus fixture's Known Gaps are unchanged unless the subject decision above changes them, in which case the receipt lists the diff.

### Investigation targets
**Required:**
- `Authoring.lean` (post-.10 location) — `cancellationKnownGap`, `operationCorrelatedProgressKnownGap`, `completionKnownGaps`, `check`
- `model/Umpire/KnownGap.lean` — `KnownGap`, `KnownGapKind`, `KnownGapSet.checkCanonical`
- `model/Umpire/Query.lean` — `authoredKnownGaps`
- `model/Umpire/Case/Compiler.lean` — how `knownGaps` reach Provenance
- `.flow/tasks/fn-83-author-a-live-case-from-a-model-file.6.md` — the white-box row this task unblocks

### Key context
- Depends on .10 so the grammar is added once, in the Umpire command module.

## Acceptance
- [ ] No Known Gap is hard-coded in the authoring layer; a Query's Known Gaps are exactly the ones its Model file declares
- [ ] The Nexus success Model declares its gaps in command syntax; the async-Nexus fixture's Known Gaps are unchanged or the receipt lists the diff and why
- [ ] The white-box representation for .6 is decided and recorded
- [ ] Unknown kind and duplicate code reject with located messages pinned by `#guard_msgs`
- [ ] `cd model && lake build`, `make umpire-check-case-runtime-conformance`, `make lint-model` pass


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
