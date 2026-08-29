---
satisfies: [R1, R5, R7]
---
# fn-19-bounded-local-temporal-execution-and.5 Compose Nexus System execution programs and configuration

## Description
### Umpire4 reconciliation (normative)

Move all Nexus execution/program/configuration ownership and public facades from `Temporal.Feature` to `Temporal.System`. Feature retains product-visible semantics only. Compose the complete current `ExperimentSpec`; do not reconstruct participant programs or other omitted meaning from an incomplete alternate representation.

The legacy implementation detail below is retained for context but is subordinate to this reconciliation.

Complete R1/R5's model-owned Nexus-specific binding and canonical two-member input set for the one live program.

**Size:** M
**Files:** `model/Temporal/Feature/Nexus/Execution.lean`, `model/Temporal/Feature/Nexus/ExecutionTests.lean`, `model/Temporal/Feature/Nexus.lean`, `model/Temporal/Feature.lean`, `model/TemporalModelTests.lean`, `tools/umpire/temporal/nexus/testdata/caller-closure-input-set/**`
**Touches:** [model/Temporal/Feature/Nexus/Execution.lean, model/Temporal/Feature/Nexus/ExecutionTests.lean, model/Temporal/Feature/Nexus.lean, model/Temporal/Feature.lean, model/TemporalModelTests.lean, tools/umpire/temporal/nexus/testdata/caller-closure-input-set/**]

### Approach
- Compose the exact local profile, fn-4 evidence/profile/program/mapping references, one participant protocol descriptor, exact capabilities, and fixed budgets into a canonical RuntimeConfiguration for the existing caller-closure ExperimentSpec.
- Define checked inert participant-program metadata for the four phase commands and exact target/action/occurrence; add no callbacks or new persisted family.
- Emit/check in the canonical RuntimeConfiguration plus exact fn-18 manifest/bindings for the two-member input set.
- Prove target/action/occurrence/fault/participant/protocol/capability/program/ref changes fail composition or set admission.
- Wire the Nexus sub-facade through the existing `Temporal.Feature` facade and aggregate tests while keeping reusable Umpire modules domain-neutral.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Feature/Nexus/CallerClosure.lean:462-525`
- fn-4 Temporal evidence profile/program/mapping values
- Task `.1` profile and fn-18 Runtime/Set encoders
- canonical caller-closure ExperimentSpec fixture

### Acceptance
- [ ] The two-member input set is canonical, strictly admitted, and binds the exact current caller-closure artifact.
- [ ] Program/config identities change on every meaning-bearing mutation and remain insensitive only to declared provenance exclusions.
- [ ] Any unsupported fault, extra participant/action/target, or semantic-reference drift rejects before Go execution.
- [ ] Public Temporal facades and focused Lean tests pass.
## Acceptance
- [ ] R1/R5 exact Nexus configuration/program binding is model-owned and inspectable.
- [ ] Cross-language fixture bytes pass fn-18 strict set admission.
- [ ] No new semantic or artifact authority is introduced.

## Done summary
Implemented the deep `Temporal.System.Execution.Nexus` contract for inert, preflight-checked Nexus programs/configuration and a neutral integration proof that consumes the complete fn18 caller-closure experiment. Added the exact canonical pretty-printed Artifact Set fixture, strict admission/preflight drift coverage, and kept `Temporal.Feature.Nexus` independent of System with `Examples/` absent.

Verification is green for focused Go runtime/local/Nexus/temporaltest suites, exact artifact admission, System and neutral-integration Lean builds, the full model import-policy lint, and built-in Lean lint. The stale future-task CLI/Feature execution/Make surfaces were inherited absent and intentionally remain outside this task.

baseline: green for implemented dependency surfaces; inherited red for stale future-task `umpire-local-run`, `Temporal.Feature.Nexus.ExecutionTests`, and `umpire-run-local` Quick entries

review: SHIP after two fixed P1 findings; memory capture skipped because flow memory is not initialized

stage: impl-review - ran [2026-08-29T16:47:38Z..2026-08-29T16:58:46Z]
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: e3cfa915b6f9a4b112bcfff6434cfbcd3d313fdd, 5a9172fcdc966f96aa2275ce79003df0aa881fde, 22916a8c5cfe3d26a2474ea7f6e9dabcabc866b8
- Tests: go test -count=1 ./tools/umpire/runtime/..., go test -count=1 ./tools/umpire/temporal/local/..., go test -count=1 ./tools/umpire/temporal/nexus/..., go test -count=1 ./temporaltest/..., go run ./tools/umpire/cmd/umpire-artifact check-set --set tools/umpire/temporal/nexus/testdata/caller-closure-input-set, cd model && mise exec -- lake build Temporal.System.Execution.LocalProfileTests, cd model && mise exec -- lake build Temporal.System.Execution.NexusTests Temporal.NexusExecutionIntegrationTests TemporalModelTests TemporalExperimentalTests, cd model && mise exec -- lake exe modelLint, cd model && mise exec -- lake --wfail lint --builtin-only --lint-only=.all,.extra,-.missingDocs
- PRs:
