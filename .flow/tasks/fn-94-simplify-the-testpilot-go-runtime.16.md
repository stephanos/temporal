---
satisfies: [R9, R11]
---
# fn-94-simplify-the-testpilot-go-runtime.16 Remove the dormant Reference arms (wire change) with per-arm decisions

## Description
Lane G: record the per-arm D1 decision, then remove the arms it removes in one wire commit with regeneration, Lean edits, gate entries, a checklist removal subsection and a re-record.

**Owner decision, per arm (recommended defaults, taken by the conductor unless the owner said otherwise):** remove `Reference.evidence_field_id`, `Reference.correlated_capture`, `Reference.model_value`; keep `supporting_event_sequence` (set and read), `SingularType.enumeration` (fn-89), `ValueType.repeated`/`map`, `SingularType.any`, `Expression.any` (used by `ir`), `Deadline.elapsed_milliseconds` (EVD-21), `InstructionLimits.max_attempts` (SEM-16), `Entrypoint.activity` (MOD-13). Record each with its evidence in the receipt.

**Size:** M
**Files:** `proto/internal/temporal/server/api/testpilot/v1/expression.proto`, regenerated `api/testpilot/v1/**`, `model/Testpilot/Authoring.lean`, `model/Testpilot/Correlated.lean`, `model/Umpire/Case/LocalNames.lean`, Lean tests naming the arms, `common/testing/testpilot/internal/verification/{correlated,correlated_prepare}.go` and tests, `common/testing/testpilot/protocol_test.go`, `tools/umpire/internal/retiredvocabulary/{check,check_test}.go`, root `common/testing/testpilot/README.md` (checklist removal subsection, "A new expression reference"), the catalog golden, pinned records and receipts
**Touches:** [proto/internal/temporal/server/api/testpilot/v1/expression.proto, api/testpilot/v1/**, model/Testpilot/Authoring.lean, model/Testpilot/Correlated.lean, model/Umpire/Case/LocalNames.lean, model/Testpilot/Tests/**, common/testing/testpilot/internal/verification/**, common/testing/testpilot/protocol_test.go, common/testing/testpilot/README.md, common/testing/testpilot/temporal/catalog_test.go, common/testing/testpilot/testdata/**, tests/testcore/testpilot/testdata/**, tools/canary/**/testdata/**, tools/umpire/replay/testdata/**, tools/umpire/internal/retiredvocabulary/**]
**Depends on (cross-spec):** fn-89-one-contract-rule-per-entity.5 and fn-89-one-contract-rule-per-entity.6 (README checklist). **Start gate:** run `flowctl show fn-92-compose-entity-machines-into-one-system` and `flowctl show fn-93-simplify-the-lean-model`; start only if each has no task started or is closed, otherwise stop and report the block. This is the spec's only wire change.

### Approach
- Re-verify each removed arm: grep `model/` for producers (camel and snake case), all fixtures, all open specs (`.flow/specs/*.md`), and `.plans/UMPIRE4_SPEC.md`. Any hit moves the arm to "kept" with the reason (R9 errors).
- Remove from `expression.proto:92-96` per the checklist numbering rule (fn-89.1 added the instance-value arm here; rebase on it); `make proto` (if it fails at the pre-existing `lint-api` finding on `case.proto`, use `make lint-protos protoc proto-codegen` as fn-89.1 did). A cold `make umpire-build-model` takes ~12 minutes; `lake lint` needs `LEAN_NUM_THREADS=1` on a 16 GB machine. With no live cluster for the re-record, the task blocks rather than landing stale records (R11).
- Remove Go handlers (`correlated.go:336-338`, `correlated_prepare.go:349,358`), Lean builders (`Authoring.lean:281-286`), Lean decode cases (`Testpilot/Correlated.lean:116,204,207`), renamer cases (`LocalNames.lean:198-201`) and Lean tests (`Tests/Fields.lean:60`).
- Gate: add `EvidenceFieldId`, `CorrelatedCapture`, `CorrelatedCaptureReference` (if that descriptor name exists) with a spec comment; add retired descriptor names to `protocol_test.go:99`. Do not gate `ModelValue` (the `ModelValue` message stays) — say so in the receipt.
- Add a removal subsection to the extension checklist (root README), and update "A new expression reference" steps 3-4.
- Regenerate fixtures (expect no diff: no fixture uses the arms), run `make umpire-rerecord-pinned-runs`, update the catalog golden in this commit.

### Investigation targets
**Required:**
- `proto/internal/temporal/server/api/testpilot/v1/expression.proto:80-110`
- `common/testing/testpilot/internal/verification/correlated_prepare.go:340-365`
- `model/Testpilot/Authoring.lean:275-290`
- `common/testing/testpilot/README.md:109-280`
- `tools/umpire/internal/retiredvocabulary/check.go:600-680`

### Quick commands
```sh
make proto
make umpire-check-testpilot-protocol
make umpire-check-testpilot-authoring
make umpire-check-case-runtime-conformance
make umpire-check-retired-vocabulary
make umpire-rerecord-pinned-runs
go test -race -tags test_dep ./common/testing/testpilot/... ./tools/umpire/... ./tools/canary/...
```

## Acceptance
- [ ] Per-arm decisions recorded with evidence before any removal.
- [ ] Each removed arm is gone from `.proto`, generated Go, Lean builders/decoders/renamer and Go handlers; no fixture changed except as regenerated for the removal.
- [ ] Gate and `protocol_test.go` hold the gateable names; the checklist has a removal subsection.
- [ ] Pinned Runs re-recorded and catalog golden updated in this commit; all listed gates pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
