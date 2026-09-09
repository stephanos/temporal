---
satisfies: [R2]
---
# fn-80-close-the-model-to-case-seam-and-harden.12 Add located Nexus3 syntax diagnostics

## Description
Step (c) of task .5's recorded sequence. Depends on step (b).

**Size:** S
**Files:** model/Temporal/Feature/Nexus3/Syntax.lean; model/Temporal/Feature/Nexus3/Tests.lean
**Touches:** [model/Temporal/Feature/Nexus3/Syntax.lean, model/Temporal/Feature/Nexus3/Tests.lean]

### Scope
Add the located diagnostics for the five error classes task .5's block enumerates, each pinned by `#guard_msgs`: constructor with arguments, unknown constructor, duplicate `before + action`, unreachable terminal, and transition count over 256. Their message text needs designing — it is not inherited from anywhere.

Each diagnostic must point at the offending source coordinates, not at the macro.

## Acceptance
- [ ] All five error classes produce a located diagnostic at the offending source coordinates.
- [ ] Each is pinned by a `#guard_msgs` test so the message text cannot drift silently.
- [ ] A newcomer renaming an Action or adding a transition gets an actionable message, which is the Goal-section defect this requirement exists to fix.
- [ ] No fixture bytes move; `make lint-model` adds nothing to the 169 baseline.

## Done summary
Added the five located model-declaration diagnostics, each thrown at the author's own coordinates
and pinned by `#guard_msgs`:

- a constructor that takes arguments, reported at the type the model names;
- an identifier that resolves to no constructor, reported at that identifier and naming every
  declared spelling of the domain, which is the actionable message a newcomer renaming an Action
  needs;
- a duplicate `before + action` pair, reported at the second row's key and naming the row that
  already declared it;
- a terminal state unreachable from every initial state over the declared rows, reported at that
  terminal;
- a table over the 256-row elaboration bound, reported at the first excess row.

Each transition row is resolved once into a record; the duplicate scan, the reachability walk and
the table all read that record, so a row is never resolved twice and diagnostics are reported in
declaration order. Spellings are compared with macro scopes erased, so a model declared through
another macro resolves and reports its authored spelling — which is what lets the over-bound test
generate its 257 rows from a local test macro rather than writing them out.

No fixture bytes moved; `make umpire-check-case-runtime-conformance` is green and `make lint-model`
holds at the 169 generated-API baseline with `Umpire.Lint` and `Shared` clean.

stage: impl-review - ran | round 1 NEEDS_WORK, round 2 SHIP (model: claude-fable-5-1 at high). Round
1's P1 was correct: the bound had been pinned on its message builder rather than through the
elaborator, justified by a comment that a test-side macro disproved. All findings from both rounds
are addressed.
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: db18ebb59, b1158935a, HEAD
- Tests: cd model && lake build Temporal TemporalModelTests UmpireTests TestpilotTests, make umpire-check-case-runtime-conformance (green, no fixture bytes moved), make lint-model (169 errors, all generated Temporal/API; Umpire.Lint and Shared clean; unchanged from baseline), CGO_ENABLED=0 go test -tags test_dep ./common/testing/testpilot/... ./tests/testcore/testpilot/..., GATE_SKIPPED:live-integration:go test -tags 'test_dep integration' ./tests -run TestTestpilot needs a live cluster; the conformance gate proves the generated Case bytes are unmoved
- PRs: