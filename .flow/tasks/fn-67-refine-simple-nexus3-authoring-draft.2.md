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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
