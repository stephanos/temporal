---
satisfies: [R2, R3]
---
# fn-157-bound-native-verification-memory-and.3 Bound measured consumer and complete generator lifetimes

## Description
Adopt the proven ownership contract at measured consumer lifetimes and the complete generator (R2-R3). Dependency on .2 protects shared lower/Producer ownership seams.

**Size:** M
**Files:** lower/Producer/generated loop, lint and conformance preparation with their lifecycle and deterministic tests.
**Touches:** [tools/umpire/lower/**, tools/umpire/lint/**, tools/umpire/conformance/**]

### Approach

- Apply task .2's measured contract only where task .1 demonstrated retention or duplication. Read current Producer's Realizer plus independent Check, lint's original Realizer plus cloned Check plus Interpreter, and conformance's cloned Model/Case factory. Keep genuinely independent readers and each fresh assessor; avoid a larger unified owner abstraction.
- Inspect the full seven-Model generator loop, which retains encoded Cases/manifest and does not explicitly retain previous Producers. Change per-Model/per-Query lifetime or encoding retention only if profiles implicate it. Every discovered Model and Query still enters the complete managed inventory before selection; bounds, expectations, unsupported/error standings and exact deterministic bytes stay fixed.
- Keep deterministic lowering's two independently constructed Producers and replay's fresh interpretation meaningful. If test lifetimes dominate, sequence independent complete readers only with independently frozen output comparison and no reduction of assertion populations. This does not establish canonical fit.
- Run task .1's `native-preservation` harness for complete Programs, Contracts, Case bytes/IDs, manifest standings, native/check receipts, errors and replay witnesses. Re-run all owner inventory and lifecycle negatives affected by the change. Preserve completion guard mutation and exact GuardError; no semantic assertion edits.
- Measure full generator and implicated consumer phases on equivalent inputs. Leave Gate/Lift scratch edits and docs to .4/.5. Any new shared source seam requires a task re-anchor before parallel work continues.

### Investigation targets

**Required:**
- `tools/umpire/lower/lower.go:103`, `tools/umpire/lower/generated.go:61` - duplicate complete views and model loop.
- `tools/umpire/lower/activity_cases_test.go:222` - independent Producer oracle.
- `tools/umpire/lint/lint.go:147` - full simultaneous views and error recovery.
- `tools/umpire/conformance/conformance.go:116`, `tools/umpire/conformance/plan.go:94` - immutable snapshot factory.
- `tools/umpire/conformance/activity_test.go:190` - out-of-order and guard-error negatives.
- `tools/umpire/conformance/conformance_test.go:438` - concurrent independent assessors.
- `tools/umpire/lower/lower_test.go:90` - deterministic lowering isolation.

## Acceptance
- [ ] Each source change has a measured consumer/generator cause and respects .2's ownership contract; unsupported seams receive no speculative edits.
- [ ] Complete full-population independent oracle comparison passes with unchanged IR/artifact bytes and deterministic identities; independent Producers and witness/assessor readers remain independent.
- [ ] Fresh/repeated/concurrent supported instances and input/returned-value mutation isolation pass, alongside owner-qualified inventory, out-of-order evidence and completion GuardError controls. Strict inherited semantic failures remain attributed rather than weakened.
- [ ] Equivalent full-input resource receipts separate per-process/phase observations from concurrent fit, which remains .5's gate obligation.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
