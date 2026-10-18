---
satisfies: [R5]
---
# fn-50-migrate-system-callerclosure-to.3 Verify and document the CallerClosure migration

## Description
Review public guidance/comments and run final compatibility gates (R5).

**Size:** S
**Files:** `model/Temporal/System/Nexus/CallerClosure.lean`, `model/Umpire/ARCHITECTURE.md`, `model/README.md`, `model/ARCHITECTURE.md`
**Touches:** [model/Temporal/System/Nexus/CallerClosure.lean, model/Umpire/ARCHITECTURE.md, model/README.md, model/ARCHITECTURE.md]

### Approach
- Keep architecture docs unchanged where they already describe FiniteMachine as the ordinary route; revise only factually stale direct-kernel wording.
- Audit that every pre-existing CallerClosure comment remains or is minimally corrected for ownership.
- Run focused suites, aggregate builds, exact regressions, trust/import checks, and lint.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/System/Nexus/CallerClosure.lean:3-10` — module meaning that should remain stable
- `model/Umpire/ARCHITECTURE.md:82-105` — ordinary versus expert Target routes
- `model/README.md:140-155` — model authoring overview
- `model/ARCHITECTURE.md:160-175,290-305` — existing Target boundary

## Acceptance
- [ ] No public document or comment makes a stale claim about CallerClosure construction.
- [ ] Existing comments unrelated to the changed representation remain byte-for-byte present.
- [ ] Focused and aggregate Lean builds, exact regression, trust/import, and `make lint-model` pass.
- [ ] Literal `make lint-code` runs and reproduces only the pre-edit inherited 1,373-finding golangci baseline with no task Go path; the existing `tools/umpire/runtime/errors.go:60:9` `errortype` finding is unchanged, and a no-fix golangci run scoped from the task base passes.
- [ ] No generated, artifact, checksum, fingerprint, or fn50-authored unrelated-file drift remains; separately authorized roadmap commits `428fc87d9` and `69ce0485f` are excluded from fn50 implementation and evidence.

## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: