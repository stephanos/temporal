---
satisfies: [R2, R6]
---
# fn-44-seal-observation-traces-and-centralize.2 Migrate Observation and Property coordinate consumers

## Description
Replace Observation and Property positional helpers with Task 1's Core API (R2), keeping their domain-specific diagnostics and prior-state semantics local.

**Size:** M
**Files:** `model/Umpire/Observation/Evaluation.lean`, `model/Umpire/Observation/Check.lean`, `model/Umpire/Observation/Verdict.lean`, `model/Umpire/Property/Language.lean`, `model/Umpire/Observation/Tests/EvidenceLink.lean`, `model/Umpire/Observation/Tests/Verdict.lean`
**Touches:** [model/Umpire/Observation/Evaluation.lean, model/Umpire/Observation/Check.lean, model/Umpire/Observation/Verdict.lean, model/Umpire/Property/Language.lean, model/Umpire/Observation/Tests/EvidenceLink.lean, model/Umpire/Observation/Tests/Verdict.lean]

### Approach
- Remove Observation's private enumeration, lookup, and kind classifiers in favor of Core operations.
- Own the `PropertyTraceField` compatibility adapter in Property so Core does not import an authoring language.
- Route Check and Verdict evidence selection through the same adapter and preserve caller-specific failure construction.
- Add explicit zero-coordinate and prior-state boundary regressions before simplifying duplicate helpers.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Observation/Check.lean:24-53` — duplicated lookup and field compatibility.
- `model/Umpire/Observation/Verdict.lean:197-223` — second duplicated Property adapter.
- `model/Umpire/Observation/Evaluation.lean:1379-1392` — saturating lookup bug.
- `model/Umpire/Observation/Evaluation.lean:1493-1497` — duplicated kind mapping.
- `model/Umpire/Property/Language.lean:9-33` — Property field vocabulary owner.
- `model/Umpire/Property/Language.lean:853-876` — current field-value semantics.
- `model/Umpire/Observation/Tests/EvidenceLink.lean:161-177` — coordinate compatibility regression.

### Key context
Initial state counts as `priorState` only when a step exists; a resulting state counts as `priorState` only before a later step. Preserve those exact semantics and existing diagnostics.

### Quick commands
```bash
cd model && mise exec -- lake build Umpire.Property.Tests Umpire.Observation.Tests
```

## Acceptance
- [ ] Observation Evaluation, Check, and Verdict contain no private coordinate enumeration, positional lookup, or kind classifier.
- [ ] Property owns one documented field/coordinate compatibility adapter used by both Observation consumers.
- [ ] Zero and oversized coordinates fail closed without aliasing the first value and retain applicable existing diagnostics.
- [ ] Initial/prior/resulting-state and observation/relation field semantics remain exact for empty, one-step, and multi-step traces.
- [ ] Focused Property and Observation suites pass with preserved comments, import direction, and warnings baseline.

## Done summary
Migrated Observation coordinate enumeration, lookup, and kind handling to the Core API, and centralized strict Property field compatibility for both verdict consumers. Added zero-coordinate admission and empty/one-step/multi-step prior-state regressions; the focused Lean build and model lint pass, while `make lint-code` reproduces exactly 1,373 inherited Go findings outside this task.

baseline: green (`cd model && mise exec -- lake build Umpire.Property.Tests Umpire.Observation.Tests`)

stage: impl-review - ran (Codex SHIP; receipt `/tmp/impl-review-receipt-fn-44-seal-observation-traces-and-centralize.2.json`)
## Evidence
- Commits: 36fc1a5ef2017109209e1183b4dc8b96b5d3a8e1
- Tests: cd model && mise exec -- lake build Umpire.Property.Tests Umpire.Observation.Tests, make lint-model, make lint-code (inherited red: 1373 pre-existing Go findings; no task-path findings)
- PRs: