---
satisfies: [R1, R2, R3, R4, R5]
---
# fn-67-refine-simple-nexus3-authoring-draft.1 Move and refine the standalone authoring draft

## Description
Resolve the five review findings as a design iteration: compact model, derived identities with optional overrides, explicit action roles and results, operation-scoped bounds, and separate fail-closed Case integration draft.

## Acceptance
Nexus3 owns the draft; IDs derive from namespace, kind, and name with optional compatibility overrides and no registry; runtime support is accurately qualified; applicable draft consistency checks are recorded; no commits or unrelated edits.

## Done summary
Moved the teaching draft from Nexus2/Nexus.lean to Nexus3/Nexus.lean. Preserved and updated the teaching comments; separated command and wait actions, outcomes and facts, scoped model progress to one operation, moved terminal consistency to admission, and reused named Properties in witness Queries. Added Integration.lean as an explicitly non-executable design for action bindings, correlation, Case lowering, and unsupported-case rejection. Per user steering, removed the temporary Identity.lean registry and use namespace/kind/name-derived IDs with optional individual compatibility overrides. Both retained Lean files are proposed syntax and are not imported into production model roots.

Verification: inline Python consistency check passed for relocation, derived IDs/no registry, complete four-action binding coverage, four Query Property references, scoped bounds, terminal admission, and runtime rejection requirements. git diff --check passed; flowctl validate passed. make lint-code GOLANGCI_LINT_FIX=false exited 2 with Go diagnostics outside edited drafts; output is /tmp/nexus3-lint-code.log. No compiler or runtime test result is claimed for the proposed syntax. No commits created.
## Evidence
- Commits:
- Tests: python3 inline Nexus3 draft consistency check (passed), git diff --check (passed), flowctl validate --spec fn-67-refine-simple-nexus3-authoring-draft --json (passed), make lint-code GOLANGCI_LINT_FIX=false (failed: Go diagnostics outside edited drafts; /tmp/nexus3-lint-code.log)
- PRs: