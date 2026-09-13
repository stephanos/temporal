---
satisfies: [R14]
---
# fn-87-tighten-the-testpilot-protocol-glossary.14 Case-local names and short model values mapped through provenance

## Description
Make Program and Contract refer to states, actions, outcomes, facts, fields, captures and rules by short Case-local names, with provenance mapping each local name to its Definition ID; make a model value its declared spelling, with a parameterized value's canonical encoding recorded as a fingerprint in provenance (R14). `typed-nexus-case.json` must shrink by at least its 244 KB of encoded values.

**Size:** M
**Files:** `proto/.../v1/{case,correlated}.proto` (provenance name and value rows; `ModelValue` field naming), `api/testpilot/v1/*`, `model/Umpire/Case/{Correlated,Compiler,Producer}.lean`, `model/Umpire/Case/Projection/Lowering.lean`, `model/Umpire/Provenance.lean`, `model/Testpilot/Correlated.lean`, `model/Shared/CorrelatedProjection.lean` (only if value identity is defined there), typed Nexus and typed unary Producers and tests, `model/Umpire/Case/Tests/CorrelatedFixtures.lean`, `common/testing/testpilot/internal/verification/{correlated.go,correlated_prepare.go,correlated_test.go}`, `tests/testcore/testpilot/{artifact_test.go,typed_unary_artifact_test.go}` and rule-id constants, conformance `expected.json` (through the generator), fixtures, mapping
**Touches:** [proto/internal/temporal/server/api/testpilot/**, api/testpilot/**, model/Umpire/**, model/Testpilot/**, model/Shared/**, model/Temporal/**, common/testing/testpilot/**, tests/testcore/testpilot/**, tests/testpilot_*_test.go]

### Approach
- Local names (decide and record): one namespace per Case covering every Definition ID the Program and Contract name (states, actions, outcomes, facts, evidence fields, captures, rules). The local name is the last dotted segment of the Definition ID, extended leftward one segment at a time only until it is unique within the Case (`temporal.nexus.success.typed-nexus.evidence.operation-identity` → `operation-identity`; a clash between `action.schedule` and `state.schedule` becomes `action.schedule` and `state.schedule`). Rule ids are localized too, so `RuleVerdict.rule_id` and `terminal_state_id` in Verdicts and `expected.json` change through a declared mapping step, not by regeneration alone.
- Provenance rows (extend .13's `CaseProvenance`): `LocalName { local_name; definition_id }` and `ModelValueFingerprint { local_name; spelling; fingerprint }`. The fingerprint is a SHA-256 hex of the value's canonical encoding (the bytes `Umpire/Case/Correlated.lean:28` `atom` produces today); state the hash in the proto comment so Go can recompute it in the equivalence test.
- Short values: a model value is its declared spelling (constructor name). typed-nexus has collisions: `outcome.completed`, `outcome.scheduled` and `state.scheduled` each have two distinct encodings, and `action.schedule` carries two dotted command IDs as values. Spellings must stay injective per definition or transitions merge and Verdicts move: when two encodings share a spelling within one definition, append a short disambiguator derived from the fingerprint (shortest unique hex prefix, at least 8 characters) and record the rule. Values that are themselves Definition IDs are localized like names.
- `ModelValue { definition_id; value }` keeps its field names, mirroring `Umpire.ModelValue` (`definitionId`, `value`); after this task `definition_id` holds the Case-local name, which the field's proto comment states.
- Errors at production (Lean, `Umpire.Case.Compiler` or the Producer): a local name used for two Definition IDs rejects naming both; one Definition ID given two local names rejects; a value spelling that stays ambiguous after disambiguation rejects naming the definition and both encodings. Lean unit tests for each.
- Runtime: Go correlated admission and evaluation compare names and value strings; nothing reads provenance. `projection_fingerprint` keeps covering the full canonical projection (state what it covers in its comment).
- Verdict pins: `correlated.json` `expected`, conformance `expected.json` modulo the declared rule-id map, typed unary/typed Nexus artifact tests' rule-id constants, and live Verdicts.
- Mapping: a relation, not a pure function: for each baseline Case, read the regenerated fixture's `LocalName` and `ModelValueFingerprint` rows, check every row's Definition ID occurs in the baseline, check each fingerprint by recomputing SHA-256 over the baseline encoding, then substitute local names and spellings into the baseline tree and compare. Record the typed-nexus byte size before and after in the done summary (at least 244 KB smaller).

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Case/Correlated.lean:20-60,250-280` — `atom` and model value construction
- `model/Testpilot/Correlated.lean:90-220` — decoding values
- `common/testing/testpilot/internal/verification/correlated_prepare.go` — transition table and value admission
- `tests/testcore/testpilot/testdata/typed-nexus-case.json` — the 21 long values and 34 Definition IDs
- `tests/testcore/testpilot/artifact_test.go:300-330`, `typed_unary_artifact_test.go:20-40`

**Optional:**
- `model/Shared/CorrelatedProjection.lean`
- `.plans/UMPIRE4_SPEC.md` `Umpire.ModelValue`, SEM-19

### Key context
- Case identity and prepared identities change; that is expected. Contract semantics must not: injective spellings are the invariant that keeps transitions distinct.

## Acceptance
- [ ] Program and Contract use Case-local names; provenance maps each to its Definition ID; rule ids are local and Verdict/expected mappings are declared
- [ ] model values are declared spellings (with fingerprint disambiguators only on collisions); provenance records each parameterized value's fingerprint; typed-nexus fixture at least 244 KB smaller (sizes in summary)
- [ ] a local name for two Definition IDs, two names for one Definition ID, and an ambiguous value spelling each reject at production naming both sides (Lean tests)
- [ ] equivalence relation step validates names and fingerprints; `correlated.json` expectations, conformance Verdicts (modulo the map) and live Verdicts unchanged
- [ ] `make umpire-check-regression` exit 0 with nine live identities; `make lint-model` 163; `make lint-code` no new issues


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

