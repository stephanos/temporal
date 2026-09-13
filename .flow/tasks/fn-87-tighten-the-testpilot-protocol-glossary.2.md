---
satisfies: [R1]
---
# fn-87-tighten-the-testpilot-protocol-glossary.2 Glossary renames, part one: Contract, Run and correlated names

## Description
Apply the Contract-, Run- and correlated-side rows of the spec's Renames table (R1) through every layer in one atomic change: proto, `make proto`, generated Lean (elaborated), `Testpilot.Authoring`/`Testpilot.Correlated`, Lean Producers, Go runtime and tests, regenerated fixtures, the equivalence mapping, and the retired-vocabulary gate. Split from the Program/instruction/value renames (.3) only to keep each diff reviewable; each half compiles and passes on its own.

**Size:** M (mechanical, wide)
**Files:** `proto/internal/temporal/server/api/testpilot/v1/{contract,run,instruction}.proto`, `api/testpilot/v1/*.pb.go` (regenerated), `model/Testpilot/{Authoring,Correlated}.lean`, `model/Testpilot/Tests/{Fields,Authoring,Protocol}.lean`, `model/Umpire/Case/{Correlated,Compiler,Producer,Provenance?}.lean` and their tests, `model/Temporal/**` call sites, `common/testing/testpilot/**` (Go), `tests/testcore/testpilot/**`, `tests/testpilot_*_test.go`, `tools/umpire/cmd/umpire-gen-case-runtime-conformance/*`, `tools/umpire/internal/retiredvocabulary/check.go`, `common/testing/testpilot/protocol_test.go`, regenerated fixtures, `common/testing/testpilot/internal/protocolmigration/mapping.go`, Testpilot READMEs that spell the renamed words, `.plans/UMPIRE4_ORDER.md:88` (cites `RunStatus`)
**Touches:** [proto/internal/temporal/server/api/testpilot/**, api/testpilot/**, model/Testpilot/**, model/Umpire/Case/**, model/Umpire/Provenance.lean, model/Temporal/**, common/testing/testpilot/**, tests/testcore/testpilot/**, tests/testpilot_*_test.go, tools/umpire/**, .plans/UMPIRE4_ORDER.md]

### Approach
- Renames in this task (spec Renames table): `RunStatus` → `RunDisposition` (enum values `RUN_DISPOSITION_*`, and `Run.status` → `Run.disposition`); `CorrelatedContract.clauses` → `rules`, `CorrelatedRule.clause_id` → `rule_id`; `ContractStateStatus.CONTRACT_STATE_STATUS_NONTERMINAL` → `..._PENDING`; `CorrelatedValue` → `ModelValue`; `INSTRUCTION_OUTCOME_STATUS_PROTOCOL_NON_SUCCESS` → `..._PROTOCOL_FAILURE`; drop the `Definition` suffix on Contract declarations (`ContractRuleDefinition` → `ContractRule`, `ContractStateDefinition` → `ContractState`, `ContractTransitionDefinition` → `ContractTransition`, `ContractCaptureDefinition` → `ContractCapture`); the evidence source name `CorrelatedEvidenceRule.source` and `CorrelatedIdentity.source` → `evidence_source`. `RunEvent.source_id` keeps its name. `RespondNexus`, `NexusResponseKind` and `StartNexusOperation` are NOT renamed (fn-85 R10 owns them).
- Lean name clash: `model/Umpire/Case/Correlated.lean:20-35` opens `temporal.server.api.testpilot.v1` and uses `ModelValue` unqualified for `Umpire.ModelValue` (`atom (value : ModelValue) : CorrelatedValue`). After the rename both types resolve as `ModelValue` there; qualify the wire type (`temporal.server.api.testpilot.v1.ModelValue`) or narrow the `open`, and check `Umpire/Case/CorrelatedProofs.lean`, `Testpilot/Correlated.lean` and `Testpilot/Tests/Fields.lean` for the same clash. List each resolution in the summary.
- Field numbers do not change in this task (renumbering is .4), so binary encodings stay stable and only JSON names move.
- Order: edit protos → `make proto` (regenerates `api/testpilot/v1`) → `cd model && PROTOC=$(mise exec -- which protoc) mise exec -- lake build Testpilot TestpilotTests` and fix Lean → `go build ./... && go vet -tags test_dep` over Testpilot packages and fix Go → `make umpire-gen-case-runtime-conformance` → mapping steps → gates.
- `model/Umpire/Case` uses "clause" as Umpire's own concept (`Compiler.lean:55` `clauses : List Provenance.CorrelatedRuleBinding`, provenance JSON `clauseId`). Rename only where the value is the protocol's Correlated Rule; the Umpire-side field and the provenance JSON key follow the same glossary word (`rules`, `ruleId`) because the glossary calls them Correlated Rules; no blind sed.
- Go hand-written identifiers that mirror generated names (`GetStatus()` on Run, `RUN_STATUS_*` constants in tests and Drivers) follow the new generated names. `protocol_test.go:52` lists `"RunDisposition"` as retired; remove it and add the names this task retires.
- Mapping steps (one per rename, `Requirement: "R1"`): message/enum literal renames and JSON key renames (`clauses`→`rules`, `clauseId`→`ruleId`, `status`→`disposition` under `Run` only, `source`→`evidenceSource` under the two correlated messages). Provenance `producerData` is base64 JSON; if its `clauseId` key is renamed, the step decodes, renames and re-encodes it, declared as its own step.
- Retired-vocabulary gate (`check.go` `exactTokens`): add `RunStatus`, `clause_id`, `GetClauseId`, `CorrelatedValue`, `ContractRuleDefinition`, `ContractStateDefinition`, `ContractTransitionDefinition`, `ContractCaptureDefinition`; SCREAMING_SNAKE constants need a prefix/whole-constant rule like the `SCOPED_*` rule (`check.go:630`): `RUN_STATUS_`, `CONTRACT_STATE_STATUS_NONTERMINAL`, `PROTOCOL_NON_SUCCESS`. Do NOT add `ClauseId` or `clauseId`: `clauseId` is Umpire's live Property-clause identifier in the scanned model trees (`Umpire/Property/Check.lean:679-707`, `Umpire/Evidence/Check.lean:61,135`, `Umpire/Search.lean:652-666,936`, `Temporal/Feature/Nexus/Race/Race.lean:288-295`), and a capitalized token also retires its lowerCamel form (`check.go:617-620`); Umpire's Property clauses keep their word. The protocol JSON key `clauseId` is covered by the equivalence test and the regenerated fixtures instead; record this in the summary. Reword `.plans/UMPIRE4_ORDER.md:88` so it no longer spells `RunStatus` (permitted document).
- Docs: `common/testing/testpilot/README.md:68` "Run status" → "Run disposition"; other READMEs only where they spell a renamed identifier.

### Investigation targets
**Required** (read before coding):
- `proto/internal/temporal/server/api/testpilot/v1/contract.proto`, `run.proto`
- `model/Testpilot/Correlated.lean:98,211,492` — decoder over `CorrelatedValue`
- `model/Umpire/Case/Compiler.lean:55,99-127` and `model/Umpire/Provenance.lean:114` — the Umpire "clause" word and provenance JSON
- `tools/umpire/internal/retiredvocabulary/check.go:330-640` — token rules, SCREAMING_SNAKE handling
- `common/testing/testpilot/protocol_test.go:34-75`

**Optional:**
- `common/testing/testpilot/internal/verification/correlated.go`, `correlated_prepare.go` — heaviest Go users of clauses
- `tests/testcore/testpilot/artifact_test.go` — fixture assertions on names

### Key context
- `model/lakefile.lean:14-52` invalidates `Testpilot/Protocol.olean` on proto change; if a build looks stale, confirm `Built Testpilot.Protocol` appears.
- The gate scans generated `api/testpilot` Go and every fixture JSON; it stays red until everything is regenerated, so run it last.
- Lint baselines: `make lint-model` 163, `make lint-code` 161 after `go clean -cache` (a lower number is a truncated run).

## Acceptance
- [ ] every rename listed in Approach is applied in proto, generated Go and Lean, Authoring, Producers, Go runtime, tests and fixtures; no renamed name survives outside the equivalence baseline
- [ ] fixtures regenerated only through `make umpire-gen-case-runtime-conformance`; the equivalence test passes with one declared step per rename; every `expected.json` is byte-identical
- [ ] retired tokens (compound, SCREAMING_SNAKE and lowerCamel JSON forms) are in the gate and `make umpire-check-retired-vocabulary` is green; `protocol_test.go` retired list updated
- [ ] `make umpire-check-testpilot-protocol`, `make umpire-check-testpilot-authoring`, `make umpire-check-case-runtime-conformance` green; `make lint-model` at 163
- [ ] `make umpire-check-regression` exit 0 with nine passing live identities (`CC=/usr/bin/cc`, physical `TMPDIR`); `make lint-code` no new issues


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

