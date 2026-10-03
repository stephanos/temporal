---
satisfies: [R11, R12, R14]
---
# fn-120-adopt-what-quint-does-well-named.5 Complete ITF interchange and close the Quint-inspired tools

Touches: [tools/umpire/export/**, tools/umpire/explore/**, tools/umpire/lower/**, model/gate/**, model/README.md]

## Description
Complete Part D and the full spec gate after the choice rollout, lint and explorer are settled. Extend the existing Quint ITF reader with a versioned Umpire witness state variable; standard state-only ITF is not a lossless witness. A trace is eligible for lowering only after the importer validates its entire path against the IR and the selected Query.

**Size:** M
**Files:** tools/umpire/export/itf.go, Go trace/lowering fixtures, model gate and README.

### Approach
- Encode the initial and each step's action/class, outcome, next-state and fact Atoms (IDs and values) in a versioned state-variable extension. Export a result ordinal chosen as the lowest ordered IR result index matching all stored Trace fields at the source/class, including fact order, and label it derived canonical metadata: core `TraceStep` has no historical result index. Import verifies any supplied ordinal against the full row and ITF state projections. Document what plain external ITF omits; refuse ambiguity only when candidates differ in replay-critical Trace data, and canonicalize fully identical candidates to the first ordered result.
- Round-trip an Umpire `Trace` exactly. A same-source/same-next-state/same-outcome different-facts fixture must refuse a plain trace lacking disambiguating metadata; exact-duplicate results must canonicalize without claiming the original branch. Reject malformed ordinals, altered IDs/facts/fact order and inconsistent state projections at their numbered step.
- For a Quint-produced trace, match its complete validated path to a named Query's machine, Scenario schedule/limits, Property and outcome. Feed that imported witness through the existing realization preflight, gap checks and Case production path; do not lower `Query.Answer()`'s different search witness. A test with two valid witnesses proves the emitted Program and Contract follow the imported one.
- Run the closing model, Scala lint, Go tooling and fast Go lint gates once.

## Acceptance
- [ ] R11 versioned-extension witness round-trip preserves every stored `Trace` Atom ID/value and fact/step order. Exported ordinal is the lowest fully matching IR result and is labeled derived canonical; import checks a supplied ordinal against the full row. Plain external traces with differing replay-critical candidates get a numbered refusal, while exact duplicate results canonicalize. Different-facts ambiguity, malformed ordinal and altered facts have refusal fixtures; no historical branch identity or extension-byte equality is claimed.
- [ ] R12 one Quint trace matches a named Query, lowers through unchanged preflight and realization checks, and is admitted by Testpilot. A valid imported witness different from Go search's witness determines the Case Program and Contract; mismatch or unplaceable steps never fall back to Go search.
- [ ] R14 full gates pass and README tells authors how to use lint and explorer.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
