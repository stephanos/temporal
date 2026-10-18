---
satisfies: [R3, R7]
---
# fn-65-design-and-prototype-approachable.13 Freeze guarded Property encoding and fail-closed consumer compatibility

## Description
Implements R3, R7; use the parent spec and Nexus2 DESIGN.md for the approved semantics and prototype exceptions.

**Size:** M
**Files:** `model/Umpire/Property/Check.lean`, `model/Umpire/Property/COMPATIBILITY.md`, `model/Umpire/Property/Tests/Canonicalization.lean`, `model/Umpire/Property/Tests/GuardedCases.lean`, `model/Umpire/Property/Tests/GuardedTemporal.lean`, `model/Umpire/Artifact/Tests/Codecs.lean`, `tools/umpire/artifact/experiment_test.go`
**Touches:** [model/Umpire/Property/Check.lean, model/Umpire/Property/COMPATIBILITY.md, model/Umpire/Property/Tests/Canonicalization.lean, model/Umpire/Property/Tests/GuardedCases.lean, model/Umpire/Property/Tests/GuardedTemporal.lean, model/Umpire/Artifact/Tests/Codecs.lean, tools/umpire/artifact/experiment_test.go]

### Approach
Inventory all consumers of PropertyDeclaration/ResolvedPropertyClause and their canonical fingerprints/artifact bindings. Freeze an explicit compatible version scheme: legacy data/meaning stays unchanged; new guarded behavior cannot be read as an old unguarded form. Reuse Codecs.lean:126 identity metadata, Query canonicalization at Language.lean:511, and Case unsupported lowering. Update only affected consumers; no runtime semantic implementation is authorized.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Property/Check.lean` — semantic JSON and IDs
- `model/Umpire/Artifact/Codecs.lean` — persisted bindings
- `model/Umpire/Artifact/Tests/Goldens.lean` — canonical fixtures
- `model/Umpire/Observation/Check.lean` — downstream capabilities/requirements
- `model/Umpire/Case/Compiler.lean` — explicit unsupported lowering
- `model/Umpire/Query/Language.lean` — checked/canonical Query linkage

### Quick commands
```bash
(cd model && mise exec -- lake build Umpire.Property.Tests Umpire.Artifact.Tests.Codecs Umpire.Artifact.Tests.Goldens Umpire.Observation.Tests Umpire.Query.Tests Umpire.CaseTests Umpire.Case.CompilerTests)
make lint-model
make lint-code GOLANGCI_LINT_FIX=false
```

Baseline only existing roots before creation; after implementation include the new roots named below. Run focused commands during iteration, and the parent final gates at prototype completion. Use the Makefile LEAN_LAKE platform wrapper if direct Lake invocation cannot find the macOS SDK. Preserve comments and existing unrelated changes. No commits unless the user requests them.

Export reusable APIs through their existing owning facades and add the corresponding focused import checks when the public surface changes; keep new generic modules in the named owner. New tests must be imported into the named gate root immediately.

## Acceptance
- [x] Document the consumer inventory and each new-form decision: evaluate with authoritative semantics or reject with typed unsupported/version error. Include planning/artifact codecs, Observation, Case production and any actual Go readers; do not invent a new generic Property wire protocol where only fingerprints currently cross the boundary.
- [x] Canonical new data includes all behavior-affecting guards, cases, exceptions, group flags, clause bounds and references. Reordering declarations/cases and changing docs/source does not change semantic fingerprints; changing a guard or bound does. Explicit enumeration/trace order remains semantic where already defined.
- [x] Test legacy semantic JSON/fingerprints and persisted goldens unchanged, supported new-form deterministic round trips where applicable, unknown major versions and unknown behavior-affecting fields/operators rejected. If meaning of old data must change, stop for an explicit named deterministic migration decision; never silently reinterpret.
- [x] Unsupported downstream clauses cannot be flattened, omitted or accepted as unconditional, and cannot claim success. Observation/Case rejection tests preserve typed provenance and do not implement a second evaluator or alter established Nexus/runtime behavior.
- [x] Run affected consumer builds/goldens plus tagged Go tests when Go formats are touched. Record any reviewed source-only checksum delta separately from semantic identity and preserve warnings/trust/diagnostic precedence.
- [x] Locate actual checked-Property-to-ContractLowering producers and test new-form rejection from those checked inputs. Compiler.lean consumes lowered monitor/unsupported data and is not itself a Property recognizer. Where no generic lowering exists, document that non-consumer boundary; never count hand-authored unsupported fixtures or add an unrelated universal compiler.

## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: