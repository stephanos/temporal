---
satisfies: [R1, R3, R6]
---
# fn-25-optional-callerclosure-veil-binding-and.6 Expose the optional TemporalVerify command or record clean deferral

**Size:** M
**Files:** `model/Temporal/Tool/VerifyVeil.lean`, `model/lakefile.toml`, repository-root `Makefile`, `.plans/UMPIRE4_COMPONENTS.md`, focused model documentation
**Touches:** opt-in target registry, v2 stdout/error/progress contract, root Make target, C11 roadmap status

## Description
### Umpire4 reconciliation (normative)

The opt-in aggregate is `TemporalVerify.lean` (or an explicitly isolated equivalent) and must not enter `Umpire.lean`, `Temporal.lean`, ordinary tests, artifacts, runtime binaries, or generated tests. If the compatibility gate fails, record clean deferral without a partial command.

The legacy implementation detail below is retained for context but is subordinate to this reconciliation.

In adopt mode, add the statically registered temporal-model-verify-veil executable and repository-root make umpire-verify-veil TARGET=... target with the exact v2 receipt, semantic statuses, umpire-verification-error/v2 code/phase envelope, bounded ordered umpire-verification-progress/v1 NDJSON, resource ceilings, no repository writes, and no default/CI/runtime coupling. Document only the supported focused model workflow. In defer mode, expose no command, target, dependency, source, or model documentation; record the exact compatibility outcome in Flow and the C11 roadmap. In both modes update UMPIRE4_COMPONENTS.md with the reviewed/implemented status and retained omissions.
## Acceptance
Adopt mode accepts only workflow-nexus.target.caller-closure, emits one canonical v2 receipt, honors exact status/resource/error/progress caps and terminal-line rules, and changes only the repository-root Makefile among Makefiles. Defer mode has no unsupported placeholder surface. Native v1 verification and default regression commands pass in both modes, the roadmap states the exact branch and trust/omissions, and no qualification/promotion/runtime/CI behavior is added.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
