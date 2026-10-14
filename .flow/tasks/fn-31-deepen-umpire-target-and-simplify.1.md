---
satisfies: [R1, R5, R6]
---
# fn-31-deepen-umpire-target-and-simplify.1 Freeze Target semantics and the public/private boundary

## Description
Establish the compatibility fixtures and module boundary for R1, R5, and R6 before moving target vocabulary or callers.

**Size:** M
**Files:** `model/Umpire/Core.lean`, `model/Umpire/Target/Language.lean`, `model/Umpire/Target/Tests/**`, `model/Umpire/Examples/SwitchTests.lean`
**Touches:** [model/Umpire/Core.lean, model/Umpire/Target/**, model/Umpire/Examples/SwitchTests.lean]

### Approach
- Inventory shared vocabulary versus target-owned composition machinery.
- Preserve `composeTarget` and its deterministic validation/canonicalization as the low-level semantic baseline rather than introducing a second checker.
- Add separate whole-value, stable-`SemanticSource` canonical-metadata, semantic-digest, and typed-error fixtures under the import-pure `Umpire.Target.Tests` boundary. Freeze the downstream stable role/action-domain tokens and persisted Query/artifact bytes in `Umpire.Examples.SwitchTests`, which may legitimately import Query, Planning, and Artifact.
- Follow the generalized authored-to-checked pattern at `model/Umpire/Target/Language.lean:8-32,369-401`.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Core.lean:130-200` — provider/connector vocabulary currently in Core
- `model/Umpire/Target/Language.lean:8-32` — existing generalized declaration and checked target
- `model/Umpire/Target/Language.lean:352-401` — deterministic composition and canonical projection boundary
- `model/Umpire/Target/Tests/Canonicalization.lean` — identity fixtures
- `model/Umpire/Examples/SwitchTests.lean` — downstream Query/Planning/artifact compatibility boundary

### Acceptance
- [ ] Import-pure Target fixtures independently cover checked values, typed errors, semantic identities/digests, and stable provenance-bearing canonical metadata; downstream Switch fixtures cover existing role/action-domain token strings and persisted Query/artifact bytes.
- [ ] The intended public/private ownership is explicit and import-safe.
- [ ] The existing pure checker is retained as the sole semantic implementation and focused expert seam.
- [ ] Existing comments are preserved.

## Acceptance
- [ ] R1/R5 equivalence fixtures fail on any semantic or canonical drift.
- [ ] R6 domain-purity/import checks pass; no `Umpire.Target.Tests.*` module imports Query, Planning, or Artifact.
- [ ] Focused Target and Switch tests pass.

## Done summary
Added an import-pure Target compatibility boundary that pins the complete checked-value projection, an exact typed failure, provenance-bearing canonical metadata, and semantic identity. Switch now pins its finite-domain tokens and byte-exact canonical Query/artifact products, and the invalid Query Quick target name is corrected.

R1/R5/R6 are covered by focused fixtures through the existing `composeTarget` seam; no Target test imports Query, Planning, Artifact, or Temporal. Similar-code investigation reused the Target fixture/error helpers and the existing Query/Artifact canonical serializers. The first focused build hit one transient Lake shared-directory race and its necessary retry passed; all canonical completion gates passed. Green gate receipts were not warrantable because an unrelated user-owned `.plans/UMPIRE4_SPEC.md` worktree edit remained intentionally untouched.

baseline: red (`Umpire.QueryTests` Quick target did not exist; corrected `Umpire.Query.Tests` passed, and the external broken Go-cache symlink was repaired before its exact retry passed)
stage: impl-review - ran [2026-08-27T02:48:19Z..2026-08-27T02:50:54Z]
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 9f92e14b301fa57568812d6869f54d45939b287f, ad468d40a7aa712af4b83b7fb1c477a7d92945b6
- Tests: baseline: red (cd model && mise exec -- lake build Umpire.TargetTests Umpire.QueryTests Umpire.Planning.Tests failed pre-edit: nonexistent Umpire.QueryTests module; corrected Umpire.Query.Tests command passed), baseline: red (make umpire-check-regression failed pre-edit: broken Go-cache symlink; exact retry passed after recreating the missing cache target directory), cd model && mise exec -- lake build Umpire.TargetTests Umpire.Examples.SwitchTests, cd model && mise exec -- lake build Umpire.TargetTests Umpire.Query.Tests Umpire.Planning.Tests, cd model && mise exec -- lake build Temporal.Feature.Nexus.LifecycleTests Temporal.Feature.Nexus.OperationsTests Temporal.Feature.Nexus.Experimental.CallerClosureTests, cd model && mise exec -- lake build UmpireTests TemporalModelTests, make umpire-check-regression, make lint-model, rg -n ^import Umpire.(Query|Planning|Artifact)|^import Temporal model/Umpire/Target/Tests model/Umpire/TargetTests.lean (no matches), git diff --check
- PRs:
