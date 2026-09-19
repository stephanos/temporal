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
`CaseProvenance` now carries typed rows instead of opaque `producer_data` bytes: `definitions`, `sources`, `known_gaps` and `correlated_rules`, one repeated field per row kind so fn-85 R8's row is one more field. Fixtures show the rows readably, and no runtime component reads them.

**Protocol** (`case.proto`)
- New messages: `DefinitionBinding`, `SourceLocation` (`int32` line and column), `KnownGap` and `CorrelatedRuleBinding`.
- New enums `DefinitionKind` (`DEFINITION_KIND_*`) and `KnownGapKind` (`KNOWN_GAP_KIND_*`). The enums sit after the messages, as api-linter's file layout requires.
- `KnownGap` keeps subject and detail presence through single-arm oneofs (`subject_presence`, `detail_presence`), so a present empty detail differs from an absent one.
- The retired `CaseDefinitionKind` and `CaseKnownGap*` names are not reused.

**Lean**
- `Umpire.Provenance.make` lowers `Metadata` into the rows in the Producer's order. It now returns `Except Umpire.SourceLocation CaseProvenance` and rejects a line or column above the `int32` range rather than wrapping it. `Umpire.Case.Compiler.compile` reports that as construct `provenance.source-position`.
- `Umpire.Provenance.DefinitionBinding`, `KnownGap` and `CorrelatedRuleBinding` keep their names.
- `Testpilot.Authoring.provenance` takes the row arrays, and `Testpilot.Authoring.knownGap` encodes the presence oneofs.
- The Umpire-free synthetic Producer writes no rows.
- `TypedNexus` and `TypedUnary` open both `Umpire` and the protocol namespace, so they now hide the protocol's `SourceLocation` and `DefinitionKind`.
- Lean tests (`CompilerTests`, the Nexus success tests, the typed Nexus tests, ProtoJSON, Synthetic) read typed rows. `CompilerTests` pins the `int32` boundaries: 2147483647 is accepted, and 2147483648 is rejected for both line and column.

**Go**
- `artifact_test.go` round-trips a Go-built Case with typed rows, including a present empty detail.
- `protobuf_lean_authoring_test.go` pins the Lean ProtoJSON fixture's rows, one of each kind.
- `correlated_test.go` and `case_schema_test.go` read typed rows.
- `protocol_test.go` pins the `CaseProvenance` field list.

**Oracle**
- New R13 step in `protocolmigration/provenance.go` lifts each baseline payload into rows. It accepts a payload only when it decodes under the baseline shape with no unknown key and re-encodes to exactly its own bytes. It renames the kind literals to the new enum prefixes, admitting only the frozen baseline vocabulary, and requires canonical `int32` positions.
- It drops exactly the synthetic fixture's three opaque bytes.
- `TestDeclaredProvenanceStepLiftsOnlyTheBaselinePayload` covers the lift and each rejection. The oracle goes red with the step disabled. `expected.json` Verdict pins are unchanged.

**Retired tokens:** `ProducerData`/`producerData`, `GetProducerData`, `producer_data`, and the `CASE_DEFINITION_KIND_*` and `CASE_KNOWN_GAP_KIND_*` families (`tools/umpire/internal/retiredvocabulary/check.go`). The oracle spells them only through split literals.

**Spec drafts** (`.plans/UMPIRE4_SPEC.md`, under GOV-02; approved text unchanged)
- Restatements for the glossary Case and Provenance entries.
- ART-09 restatement, the one added rule draft: its "generic opaque provenance" and "provenance bytes" contradict R13. It also folds in .10's "independent limits" follow-up.
- Profile glossary restatement, for "independent Program and Contract ceilings".
- The MOD-15 test passes. The fn-87 Planning decisions record "decided in .13", and Boundaries now name the Profile glossary draft.

**Docs:** `model/README.md`, `model/ARCHITECTURE.md`, `model/Umpire/ARCHITECTURE.md`.

**Outside the declared Touches:** `tools/umpire/internal/retiredvocabulary/check.go`, `model/Testpilot/Tests/Synthetic.lean`, and the `.flow` spec decision and review state.

**Gates**
- Baseline was green: a full regression at `5b08b380`, exit 0, 9 live identities.
- After the change, at `d25f7256`:
  - `make umpire-check-regression`: run 1 exited 2 on known flake (c), `TestTestpilotAsyncNexusCase` ending INCONCLUSIVE. Run 2 exited 0 with 9 passing live identities.
  - `make lint-model`: 163, the baseline.
  - `go clean -cache && make lint-code`: 161, the baseline, none in touched files.

stage: impl-review - ran (claude backend, SHIP on the first round; the one FYI was applied in d25f7256, a comment explaining why the oracle freezes the baseline kind vocabulary)
## Evidence
- Commits: 60e967356d3e6f1ae51b3b662e3c56b3341ec96f, d25f7256da776c31aa1ee04a345bac8dc79a0c76
- Tests: baseline: green (CC=/usr/bin/cc TMPDIR=$(cd "${TMPDIR:-/tmp}" && pwd -P) make umpire-check-regression, exit 0, 9 passing live identities at 5b08b380), go test -count=1 -tags test_dep ./common/testing/testpilot/internal/protocolmigration/ (confirmed red with the R13 step disabled), go test -count=1 -tags test_dep ./common/testing/testpilot/... ./tests/testcore/testpilot/... ./tools/umpire/cmd/umpire-gen-lean-api/ ./tools/umpire/vocabulary/ ./tools/umpire/internal/retiredvocabulary/, make umpire-check-testpilot-authoring, make umpire-check-retired-vocabulary, lake build Testpilot TestpilotTests Temporal Umpire UmpireTests TemporalModelTests TemporalExperimentalTests umpire-correlated-fixtures, make umpire-check-regression run 1 at d25f7256: exit 2, known flake (c) TestTestpilotAsyncNexusCase verdict INCONCLUSIVE instead of SATISFIED, CC=/usr/bin/cc TMPDIR=$(cd "${TMPDIR:-/tmp}" && pwd -P) make umpire-check-regression run 2 at d25f7256: exit 0, 9 passing live identities (green receipt d25f7256), make lint-model: 163 errors (baseline 163), go clean -cache && make lint-code GOLANGCI_LINT_FIX=false: 161 issues (baseline 161), none in files this task touched
- PRs: