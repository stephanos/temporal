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
Program and Contract now name every Definition ID by a short Case-local name, and every model value by its declared spelling. Provenance rows map each name and spelling back. `typed-nexus-case.json` drops from 301,063 to 58,824 bytes.

**Protocol**
- `CaseProvenance` gains two row lists:
  - `local_names`: `LocalName { local_name, definition_id }`
  - `model_value_fingerprints`: `ModelValueFingerprint { local_name, spelling, fingerprint }`
- `local_name` carries an api-linter `core::0122::name-suffix` suppression.
- New comments:
  - `ModelValue`'s fields hold a Case-local name and a spelling.
  - The fingerprint is the lowercase SHA-256 hex of the encoding's UTF-8 bytes.
  - `projection_fingerprint` still covers the full canonical projection.

**Lean**
- New module `Umpire.Case.LocalNames`, called by `Umpire.Case.Compiler.compile` after coverage, so every Umpire Producer localizes. `Lowered`'s checked decode equality stays over the Definition-ID wire.
- Namespace: every Contract rule id, all correlated-contract ids, the Program's evidence-lift ids, and every model value that is itself a namespaced Definition ID.
- A local name is the shortest dotted suffix no other id of the Case shares.
- Spellings:
  - A structural key (new `Canonical.isKey`) takes the last segment of its definition.
  - When encodings of one definition collide, every member of the group takes `-` plus the shortest unique SHA-256 prefix of at least 8 characters.
  - The text a step condition compares is renamed as an encoding of its step's definition.
- `LocalName` rows are written only where the name changes. So get-system-info, worker-outage, synthetic and every conformance `case.json` and `expected.json` are byte-identical.
- Rejections at compile, each naming both sides: a shared name, a split Definition ID, and an ambiguous spelling.
- Tests are in `Umpire/Case/LocalNamesTests.lean`: naming, the nested-suffix edge, each error as a unit, compiled rows and the renamed step literal, and the ambiguous spelling through `Compiler.compile`.
- `LocalNames.nameIn` resolves names in tests.
- The correlated corpus evidence and test Drivers use local names. Projection work and event bytes count identifier bytes, so the `CorrelatedTests` boundary pins moved from 2960/36 to 1760/21.

**Oracle**
- A new R14 `Relate` step (`Step.Relate`, `Mapping.apply` gains `regenerated`) reads the regenerated rows before substituting names and spellings, including the events in `correlated.json`. It requires:
  - every row's Definition ID to occur in the baseline;
  - every fingerprint to be recomputed from a baseline encoding;
  - the renaming to be injective over Definition IDs and over each definition's encodings.
- `TestDeclaredLocalNameStepRelatesOnlyWhatTheBaselineNames` covers four rejections on the real fixture pair.
- With the step disabled, the oracle goes red for the async-nexus, typed-nexus, typed-unary and correlated fixtures.
- Verdict pins did not move: `correlated.json` `expected`, the conformance `expected.json` files (every rule id there is already its own local name), and live Verdicts.

**Go pins**
- The typed-unary and typed-nexus artifact rule-id and projection constants are now local names, and each test asserts the provenance mapping.
- The live tests expect local evidence kinds.
- `correlated_test.go` resolves ids through rows; the facade test uses the kind `request`; `protocol_test.go` pins the new field list.

**Sizes (typed-nexus)**

| Measured against | Before | After | Smaller by |
|---|---|---|---|
| Spec baseline (d9c77573) | 315,914 bytes | 58,824 bytes | 257,090 bytes |
| Pre-task | 301,063 bytes | 58,824 bytes | 242,239 bytes |

The 243,946 bytes of encodings are gone, less about 1.7 KB of new rows. The AC is met against the spec baseline the requirement describes, and 1.7 KB short against the pre-task file.

**Decision and docs**
- The fn-87 Planning decisions record ".14".
- The drafted Provenance glossary restatement in `.plans/UMPIRE4_SPEC.md` names the new rows.
- Also updated: `model/README.md`, `model/ARCHITECTURE.md`, `model/Umpire/ARCHITECTURE.md`, and the oracle README.
- Outside the declared Touches: `.plans/UMPIRE4_SPEC.md`, the model docs, and the `.flow` spec state.
- No tokens retired.

**Gates**
- Baseline: green through the d25f7256 regression receipt.
- `make umpire-check-regression`: exit 0 at 9340c83e and again at c7255198, both with 9 passing live identities. No flake occurred, including typed/async Nexus; the logs show the real workflow and Nexus RPCs.
- `make lint-model`: 163, the baseline.
- `go clean -cache && make lint-code`: 161, the baseline.
- The protocol, authoring, conformance and vocabulary checks passed.

stage: impl-review - ran (claude backend, SHIP on the first round; P3 applied in c7255198: the oracle lookups became methods, plus comments cross-referencing the Lean and Go traversals)
## Evidence
- Commits: 9340c83e3ba70b4bb5ae1c787003ecd2d8883549, c72551982c8a7a8a671cfadc509be46fd8ba7b63
- Tests: baseline: green via receipt d25f7256 (GATE_SKIPPED:regression:green-receipt d25f7256 - baseline reused from prior post-gate pass), go test -count=1 -tags test_dep ./common/testing/testpilot/internal/protocolmigration/, make umpire-check-testpilot-protocol umpire-check-testpilot-authoring umpire-check-case-runtime-conformance umpire-check-retired-vocabulary, CC=/usr/bin/cc TMPDIR=$(cd "${TMPDIR:-/tmp}" && pwd -P) make umpire-check-regression (exit 0 at 9340c83e and at c7255198, 9 passing live identities, no flake either run), make lint-model (163, baseline), go clean -cache && make lint-code GOLANGCI_LINT_FIX=false (161, baseline; none in touched files)
- PRs: