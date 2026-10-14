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
Property admission now rejects unknown majors before body validation while retaining the supported v1/v2 diagnostic order. Compatibility tests freeze legacy and guarded fingerprints, exercise the real Query-to-Artifact identity projection and strict Go reader, and document every semantic, rejection, fingerprint-only, and non-consumer boundary.

The exact Lean Quick and lint-model gates passed. The tagged Artifact package passed with physical `TMPDIR=/private/tmp/umpire-fn65-task13`; its first run under macOS's symlinked `/var` temp path failed only the pre-existing containment tests, and that log is retained separately. `make lint-code GOLANGCI_LINT_FIX=false` reproduced all 1,316 inherited diagnostics byte-for-byte after normalization (`sha256:aee7770bec1fe01dab8826427cc89e9ffa7e764fbac25ce6b68bf5f2e3c0b077`, zero individual diagnostic diff).

No checked-Property-to-ContractLowering producer exists. Case Compiler remains an already-lowered consumer; no generic decoder, Property wire body, universal compiler, or second runtime evaluator was added.

stage: impl-review - ran [2026-09-05T21:31:51Z..2026-09-05T21:33:53Z] SHIP (`codex:gpt-5.6-sol:medium`, session `01a0737c-0040-7433-9bdd-6999cc4c7ad0`)
stage: plan-sync - skipped(config: planSync.enabled=false)
stage: tracker-sync - skipped(config: tracker integration inactive)
## Evidence
- Commits:
- Tests: (cd model && mise exec -- lake build Umpire.Property.Tests Umpire.Artifact.Tests.Codecs Umpire.Artifact.Tests.Goldens Umpire.Observation.Tests Umpire.Query.Tests Umpire.CaseTests Umpire.Case.CompilerTests) (98 jobs, green; /tmp/fn65-task13-final-quick.log), make lint-model (272 import/lint jobs and 242 full jobs, green; /tmp/fn65-task13-final-lint-model.log), TMPDIR=/private/tmp/umpire-fn65-task13 go test -tags test_dep ./tools/umpire/artifact (green; /tmp/fn65-task13-final-go-artifact-canonical-tmp.log), go test -tags test_dep ./tools/umpire/artifact -run TestExperimentV2RejectsOneAtATimeMutations (green; /tmp/fn65-task13-go-reader.log), make lint-code GOLANGCI_LINT_FIX=false (inherited red reproduced exactly: 1316 diagnostics, sha256:aee7770bec1fe01dab8826427cc89e9ffa7e764fbac25ce6b68bf5f2e3c0b077, zero individual diagnostic diff; /tmp/fn65-task13-final-lint-code.log), lake env lean /tmp/fn65-task13-trust-audit.lean (green; only propext, Classical.choice, Quot.sound; /tmp/fn65-task13-trust-audit.log), git diff --check, git diff --cached --check
- PRs: