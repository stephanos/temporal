---
satisfies: [R4, R6]
---
# fn-60-deepen-authored-lean-canonical-json.7 Reconcile canonical JSON documentation and full gates

## Description
Reconcile the model documentation with the completed deep module and run the complete cross-domain verification once all migration tasks land. This is the single finalization task for R6.

**Size:** M
**Files:** `model/Umpire/ARCHITECTURE.md`, `model/ARCHITECTURE.md`, `model/README.md`
**Touches:** [model/Umpire/ARCHITECTURE.md, model/ARCHITECTURE.md, model/README.md]

### Approach
- Update only statements made incomplete by the widened set of internal codec owners: `Umpire.Json` owns generic typed construction/rendering; each domain still owns field names, order, semantics, validation, and diagnostics.
- Preserve all unrelated documentation and comments verbatim; do not add a user-facing migration guide because public interfaces and behavior are unchanged.
- Run the complete Lean consumer workspaces, aggregate regression gate, full import graph/linter, repository lint, exact compatibility fixtures, and trust/forbidden-path audits. Compare inherited failures as a complete identity set rather than narrowing to one green test.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/ARCHITECTURE.md:18-48` — public import responsibilities and internal implementation modules.
- `model/Umpire/ARCHITECTURE.md:625-645` — current canonical JSON ownership contract.
- `model/ARCHITECTURE.md:150-180` — deep internal seams and import graph rules.
- `model/README.md:230-245` — contributor-facing codec ownership explanation.
- `Makefile:1405-1418` — complete model lint and import-graph gate.

**Optional** (reference as needed):
- `.flow/memory/bug/integration/full-integration-gates-must-select-the-2026-09-04.md` — complete-suite baseline lesson.

### Key context
No root README, changelog, external user documentation, rollout, migration, or operations document should change. Generated API drift verification and CI coverage remain declined.

### Quick commands
```bash
(cd model && mise exec -- lake build UmpireTests Temporal TemporalModelTests TemporalExperimentalTests)
make umpire-build-model
make umpire-check-regression
make lint-model
GOLANGCI_LINT_FIX=false make lint-code
git diff --check
```

## Acceptance
- [ ] The three model documents accurately describe `Umpire.Json` as the generic typed construction/rendering owner and the domain modules as owners of meaning, field/order choices, validation, and diagnostics; unrelated prose remains unchanged.
- [ ] Public interface/import inventories, forbidden-path scans, exact byte/fingerprint/diagnostic fixtures, and trust checks show no contract or assurance drift.
- [ ] The complete model build, aggregate regression, import-graph/model lint, repository lint, and diff checks pass or report only a verified inherited baseline selected as a complete identity set.
- [ ] No user/operations docs, changelog, generated source, protocol/generator, CI workflow, `Umpire.Property`, or unrelated comment is modified.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
