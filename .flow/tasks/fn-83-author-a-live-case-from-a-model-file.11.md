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
A `query` block declares its own Known Gaps; nothing is attached on its behalf.

Grammar (a repeated line on `query`, since Known Gaps are Query-authored):

    gap <kind> "<name>" [subject "<property>"] detail "<why>"

- `model/Umpire/Command/Syntax.lean`: the `modelGap` category, kind resolution against
  `Umpire.KnownGapKind` listing the four on an unknown spelling, duplicate-name rejection on the
  line that repeats it, and `KnownGapSet.checkCanonical` over what the Query declared.
- `model/Umpire/Command/Authoring.lean`: `Origin.knownGap` derives the code as
  `<family>.known-gap.<name>` and the subject as `<family>.property.<name>`.
- `model/Umpire/Command/Registry.lean`: `Conventions` loses its `knownGaps` field and
  `model_conventions` its `gaps` clause.
- `model/Temporal/Case/Conventions.lean`: the two hard-coded constants and `completionKnownGaps`
  are deleted; the file is now the root and namespace prefix alone.
- `model/Temporal/Feature/Nexus/Success/Model.lean`: the `completion` Query declares the two gaps it
  already carried.

Subject decision: the success Model keeps its spelled subject
(`temporal.nexus.success.property.cancellationResolves`), which names a Property no Model file
declares. That is the gap rather than a defect -- a cancellation requirement has no
operation-correlated shape in this slice, so there is no `require` line to write -- and keeping it
is what makes the fixture byte-identical. The Model file says so in a comment.

White-box decision for .6, recorded on that task's file: **no new kind.** `UMPIRE4_SPEC.md`
enumerates the four kinds normatively, so a `whiteBox` kind would need a GOV-02 amendment for
something the existing vocabulary carries. The admin-service mutable-state assertion is
`interpretation`, written as `gap interpretation "white-box-mutable-state" detail "..."`, and the
detail is where "white-box" is said.

Byte pin: `make umpire-check-case-runtime-conformance` is clean with no regeneration, so the
async-Nexus fixture's Known Gaps are unchanged.

Pins: `#guard_msgs` for an unknown kind and a duplicate code; `#guard`s that a Query declaring no
gap carries none and that the success Model's two carry the derived codes and subjects.

`make umpire-check-regression` is exit 0 end to end (571 Lean jobs, 9 passing live identities);
`make lint-model` reports 0 findings outside generated `Temporal/API/Proto.lean`.

Swept in, not mine: this commit records the deletion of
`model/Temporal/Feature/Workflow/Start/DESIGN.md`. The parallel session left that file untracked, my
.10 commit swept it in under `git add -A`, and the parallel session then removed it from disk while
this task was running -- one regression run even failed mid-walk on the vanishing file. The
deletion is theirs; `git add -A` is what recorded it here. The impl-review flagged it as the one P2,
correctly.

Review: SHIP, 1 finding, which is that swept deletion.
Pinned reviewer `claude:claude-fable-5-1:high` is account-limited for this session, so the review
ran on `claude:claude-sonnet-4-5:high` -- a same-family fallback, not an equivalent cross-family
review.

stage: impl-review - ran (model: claude-sonnet-4-5, high; fable pinned but account-limited)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 0462f51681
- Tests: cd model && mise exec -- lake build, make umpire-check-case-runtime-conformance (no regeneration needed), make lint-model (0 findings outside generated Temporal/API/Proto.lean), make umpire-check-regression (exit 0)
- PRs: