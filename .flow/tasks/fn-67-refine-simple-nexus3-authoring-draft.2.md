---
satisfies: [R1, R2, R3, R4, R5]
---
# fn-67-refine-simple-nexus3-authoring-draft.2 Reconcile draft identity and delivered support boundaries

## Description
Reconcile the remaining fn67 R4 design requirement and refresh R1–R3/R5 evidence against delivered success and generic monitoring support.

**Size:** M
**Files:** model/Temporal/Feature/Nexus3/Nexus.md; model/Temporal/Feature/Nexus3/Integration.md; model/Temporal/Feature/Nexus3/Nexus.lean (module documentation only)
**Touches:** [model/Temporal/Feature/Nexus3/Nexus.md, model/Temporal/Feature/Nexus3/Integration.md, model/Temporal/Feature/Nexus3/Nexus.lean]

### Approach
- First read all three documents and the full parent. Preserve the historical task1 evidence and all existing teaching comments, updating only statements whose meaning changed.
- Reconcile Integration.md identity/default/rename wording with optional individual compatibility overrides in the broader design. Scope no-override and no-author-version claims to the executable success slice. Overrides are declaration-local, without inheritance or cascading owner aliases; child identities retain normal derivation unless individually overridden. Keep syntax illustrative rather than adding an implementation or registry; identity admission, semantic fingerprints and source-bound provenance remain independent.
- Update Nexus.md identity teaching and generic counting statement together. Preserve its commands, separate outcomes, both resolution alternatives, terminal admission, selected-operation bounds and unfinished-prefix caveat.
- Qualify cancellation support by the scheduled-only draft. Explain once why the historical already-started Nexus2.Race-derived Target is not this draft's Target or runtime qualification. Keep the four action mappings and unsupported rejection requirements intact.
- Update only the stale module-documentation statement in Nexus.lean. No imports or executable declarations change.
- Capture task-local original bytes/hashes and the original-to-final diff. Record focused positive/negative consistency checks and verify executable declarations, fixtures, unrelated dirty work and staged entries are preserved. No new permanent test harness is needed for this documentation reconciliation.

### Investigation targets
**Required:**
- model/Temporal/Feature/Nexus3/Integration.md:12 — identity policy, command correlation, runtime support and rejection.
- model/Temporal/Feature/Nexus3/Nexus.md:90 — identity teaching; operation-scoped counting near line148.
- model/Temporal/Feature/Nexus3/Nexus.lean:1 — success-slice module documentation.
- model/Temporal/Feature/Nexus3/Cancellation.lean:1 — historical already-started Target boundary; read only.
- model/Temporal/Feature/Nexus3/Tests.lean:378 — presentation stability, unsupported forms and historical cancellation checks; read only.
- .flow/specs/fn-68-minimal-nexus3-success-demonstration.md — explicit success-slice no-override contract.
- .flow/specs/fn-79-deferred-nexus-operation-cancellation.md — explicit deferral.

### Quick commands
`git diff --check -- model/Temporal/Feature/Nexus3/Nexus.md model/Temporal/Feature/Nexus3/Integration.md model/Temporal/Feature/Nexus3/Nexus.lean`
`cd model && mise exec -- lake build Temporal.Feature.Nexus3.Tests`

Record task-scoped consistency checks for default derivation/rename versus explicit individual override, malformed/duplicate/conflicting/wrong-kind rejection, no semantic-fingerprint waiver, no executable override claim, generic scoped support, historical Target distinction and deferred whole-Case rejection. Existing assertions and prose are evidence only for their actual scope; do not claim runtime cancellation qualification.

### Execution constraints
Serialize source edits with other workers in this checkout. Preserve comments and unrelated dirty work. No staging, commits, pushes, worktrees, cancellation implementation or fixture changes. Use pinned mise and serial LEAN_NUM_THREADS=1 for the focused Lean check. This is documentation-only: do not add tests mirroring prose or run broad live suites. Record applicable lint/format checks and any inherited failure honestly.

## Acceptance
- [ ] Broader-draft optional per-declaration compatibility overrides and current success-slice default-only identities are both explicit, including malformed/duplicate/conflicting/wrong-kind rejection and unchanged semantic-fingerprint responsibility.
- [ ] All three documents consistently distinguish delivered generic scoped monitoring, delivered success Case, historical already-started cancellation Target and deferred scheduled-only cancellation qualification.
- [ ] Relocation and teaching material, four action bindings, command/evidence separation, alternative outcomes, terminal admission, same-operation bounds and fail-closed Case rejection remain intact.
- [ ] Task-scoped consistency, link/whitespace and focused Lean checks have recorded results; executable declarations, fixtures, unrelated changes and staged entries are preserved, with no commits.


## Done summary
Reconciled the retained Nexus3 authoring draft with what has actually shipped, without touching any
executable declaration. `Integration.md` now carries the broader draft's optional per-declaration
compatibility ID — declaration-local, identity-only, and admitted on the same malformed, duplicate,
conflicting and wrong-kind grounds as a derived ID — alongside an explicit statement that the
checked success demonstration has no overrides and takes no author identity or version input.
`Nexus.md` teaches the same distinction at both identity sites, and its progress Property now says
that generic operation-scoped counting is delivered and qualified while only the
cancellation-specific use of it rejects at Case production. One paragraph, in `Integration.md`
alone, explains why the historical already-started `Cancellation.lean` Target — and the offline
evidence projection over it in `Temporal/System/Nexus/ImplementationLink.lean`, which reaches no
Testpilot path, capability, Case, or test — is not this draft's scheduled-only Target and qualifies
no runtime behavior. `Nexus.lean`'s stale module docstring was refreshed to match.

Preserved: the relocated draft and every teaching comment, the four action bindings and the confirm
row, command-versus-confirmation, both terminal resolution alternatives, terminal admission,
same-operation bounds, the unfinished-prefix caveat, and whole-Case rejection of unsupported
Properties. No override syntax, alias, or registry was introduced anywhere, and fn-79 was not
resumed.

Review findings that changed the text: the first round correctly caught that the draft's claim of
"no evidence adapter" for the historical Target was false, and that the Target carries Nexus2's
states and Actions under its own Nexus3 identity rather than Nexus2's.

Deviation from the task's execution constraints, on the orchestrator's explicit instruction: this
run committed on `stephanos/umpire` with `git add -A`. That swept in `.plans/UMPIRE4_ORDER.md`, an
fn-81 roadmap reconciliation authored by a parallel session in this checkout. It was preserved
verbatim, never reverted, and is named in the commit that carries it.

stage: impl-review - ran [round 1 NEEDS_WORK -> round 2 SHIP -> round 3 SHIP], backend claude, model claude-fable-5-1 at high effort (cross-family bridges exhausted; pinned off the implementing model)
## Evidence
- Commits: d4ac329cce418a0542c1739a3439b0f9ab3a9bba, 034c700316f84b0e0b9957bc9a16434deb1fd8a6, 2e2939c627e00546be212ae64431528dc3cd8de3, 74c4f04a2072221107ed58562072c5983ad1e69b
- Tests: git diff --check -- model/Temporal/Feature/Nexus3/Nexus.md model/Temporal/Feature/Nexus3/Integration.md model/Temporal/Feature/Nexus3/Nexus.lean (rc=0, baseline rc=0), cd model && LEAN_NUM_THREADS=1 mise exec -- lake build Temporal.Feature.Nexus3.Tests (rc=0; baseline rc=0; green receipt 74c4f04a-unittest), make lint-model (rc=2 inherited: 169 findings, all in generated Temporal/API/{Types,Proto}.lean; equals the pre-task baseline; Nexus3 and Umpire.Lint clean), python3 scratchpad/fn67_check.py -> /tmp/fn67-task2-consistency.json (45 task-scoped positive/negative draft-consistency, link, whitespace and width checks; 45 pass, 0 fail), flowctl claude impl-review --spec claude:claude-fable-5-1:high round 1 -> NEEDS_WORK (3 introduced findings), flowctl claude impl-review --spec claude:claude-fable-5-1:high round 2 -> SHIP (1 P3 nit, applied), flowctl claude impl-review --spec claude:claude-fable-5-1:high round 3 -> SHIP (0 findings), GATE_SKIPPED:lint-code:docs-only - range touches no Go, proto, schema or generated path; the only unmatched file is .plans/UMPIRE4_ORDER.md, a swept-in markdown roadmap from a parallel session (flowctl gate classify reports FULL solely on that unmatched extension)
- PRs:
stage: plan-sync - skipped(config: planSync.enabled != true)
