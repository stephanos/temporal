---
satisfies: [R13]
---
# fn-87-tighten-the-testpilot-protocol-glossary.13 Structured Case provenance rows; glossary and ART-09 amendments drafted

## Description
Replace the opaque `producer_data` bytes with typed provenance rows (R13, spec "Readable provenance and identity"): Definition IDs with fingerprints and kinds, sources, Known Gaps and correlated rule bindings, readable in fixture diffs. The runtime still reads none of it. fn-85 R8 adds its abstraction-claim row to this structure later; this task adds no such row. Draft the glossary Case and Provenance amendments, and the ART-09 restatement its "generic opaque provenance" wording needs.

**Size:** M
**Files:** `proto/.../v1/case.proto`, `api/testpilot/v1/*`, `model/Umpire/Provenance.lean` (the `Metadata` → protocol rows lowering; `producerData` goes), `model/Umpire/Case/{Compiler,Producer}.lean`, `model/Temporal/Testpilot/*.lean`, `model/Temporal/Case/**`, typed Producers, `model/Testpilot/Authoring.lean:540-554` (`provenance`), `model/Testpilot/Tests/{Authoring,ProtoJSON}.lean`, `common/testing/testpilot/internal/verification/correlated_test.go:260-275`, `tests/testcore/testpilot/{artifact_test.go:55-80,300-320,protobuf_lean_authoring_test.go:30-40}`, `tools/umpire/cmd/umpire-gen-lean-api/case_schema_test.go:21`, fixtures, mapping, `.plans/UMPIRE4_SPEC.md` (glossary Case and Provenance entries, ART-09 restatement), docs (`model/README.md:33-34,46,53,256`, `model/ARCHITECTURE.md:123-161`, `model/Umpire/ARCHITECTURE.md:191-195`, `tests/testcore/testpilot/README.md:43`)
**Touches:** [proto/internal/temporal/server/api/testpilot/**, api/testpilot/**, model/Umpire/Provenance.lean, model/Umpire/Case/**, model/Temporal/**, model/Testpilot/**, common/testing/testpilot/**, tests/testcore/testpilot/**, tools/umpire/**, model/README.md, model/ARCHITECTURE.md, model/Umpire/ARCHITECTURE.md, .plans/UMPIRE4_SPEC.md]

### Approach
- Proto (`case.proto`): `CaseProvenance { string producer_id; string producer_version; repeated DefinitionBinding definitions; repeated SourceLocation sources; repeated KnownGap known_gaps; repeated CorrelatedRuleBinding correlated_rules; }` with messages mirroring `Umpire.Provenance` (`model/Umpire/Provenance.lean:18-80`): `DefinitionBinding { definition_id; behavior_fingerprint; DefinitionKind kind }`, `KnownGap { KnownGapKind kind; code; optional subject; optional detail }`, `SourceLocation { path; line; column; provenance }`, `CorrelatedRuleBinding { rule_id; property_id; property_fingerprint; projection_id; projection_fingerprint; SourceLocation source }`. Enums follow the api-linter value-prefix rule every existing enum obeys: `DefinitionKind` with `DEFINITION_KIND_*` values and `KnownGapKind` with `KNOWN_GAP_KIND_*` values; the current JSON spellings (`CASE_DEFINITION_KIND_TARGET`, ...) map to them by a declared literal-rename step. `CaseDefinitionKind`, `CaseKnownGap` and `CaseKnownGapKind` are pinned as retired in `protocol_test.go:49-50` (typed rows were once removed in favor of bytes; R13 deliberately reverses that), so do not reuse those names. These are generic (no Temporal names), so SCP-02 and MOD-01 hold. Leave room for fn-85's row by keeping rows as separate repeated fields, not a oneof.
- Lean: `Umpire.Provenance.make` builds the rows directly instead of `CanonicalJson` bytes; keep the Lean structure names `Umpire.Provenance.DefinitionBinding` and `Umpire.Provenance.KnownGap`, which the glossary cites and MOD-15's resolvable-glossary test checks. Row order stays the order the Producer lists them (deterministic, ART-11).
- Go readers: `correlated_test.go:266` and `artifact_test.go:313` unmarshal `producer_data` JSON; read the typed rows instead. `artifact_test.go:55-80` and `protobuf_lean_authoring_test.go:37` round-trip arbitrary bytes `{0,255,128}`; replace with a typed-row round trip. The runtime (`Prepare`, Drivers) must not start reading provenance; add no new reader outside tests.
- Glossary drafts in `.plans/UMPIRE4_SPEC.md`: after the Case and Provenance entries, add restatements marked `*(drafted by fn-87; awaiting GOV-02 approval.)*`: a Case carries structured provenance; Provenance is typed rows (Definition bindings with fingerprints and sources, Known Gaps, correlated rule bindings) that the runtime reads none of. ART-09 says "generic opaque provenance" and "producer-owned provenance bytes"; add an ART-09 restatement the same way (the spec's Boundaries name only the SEM-16 and glossary drafts; record in the done summary and the spec decision note that ART-09 is the one added draft because R13 cannot hold otherwise). Approved text is not edited. Cited dotted Lean names must exist in `model/` (MOD-15).
- Mapping: a step decodes the base64 `producerData` JSON of each baseline Case, validates it parses under the old Provenance JSON shape, and lifts it into the typed rows (`clauseId` becomes `ruleId` per .2; `CASE_DEFINITION_KIND_*` → `DEFINITION_KIND_*` and the Known Gap kind literals → `KNOWN_GAP_KIND_*` as declared literal renames). Retire `producer_data`, `producerData`, `ProducerData`.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Provenance.lean` (whole file)
- `proto/internal/temporal/server/api/testpilot/v1/case.proto`
- `tests/testcore/testpilot/artifact_test.go:50-80,300-320`
- `common/testing/testpilot/internal/verification/correlated_test.go:255-275`
- `.plans/UMPIRE4_SPEC.md` glossary Case, Provenance; ART-09; MOD-15

**Optional:**
- `tools/umpire/vocabulary/spec_names_test.go` — MOD-15 enforcement
- `model/Umpire/Case/Compiler.lean:99-127` — where provenance is attached

### Key context
- Provenance bytes currently count toward Case identity through the canonical encoding; the prepared Case identity (ART-10/ART-14) changes with any byte change, which is expected here.

## Acceptance
- [ ] `CaseProvenance` holds typed rows for definitions, sources, Known Gaps and correlated rule bindings; `producer_data` is gone; fixtures show readable rows
- [ ] Lean `Umpire.Provenance` builds the rows; `Umpire.Provenance.DefinitionBinding` and `KnownGap` keep their names; no runtime component reads provenance
- [ ] glossary Case and Provenance restatements and an ART-09 restatement are drafted under GOV-02; MOD-15 test passes
- [ ] equivalence test passes with the validated provenance-lift step; Verdict pins unchanged; retired tokens added
- [ ] `make umpire-check-regression` exit 0 with nine live identities; `make lint-model` 163; `make lint-code` no new issues


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

