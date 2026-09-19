---
satisfies: [R6, R7]
---
# fn-69-extract-testpilot-from-umpire.5 Cut over Lean/API identities and functional Case fixtures

## Description
Cut over the generated Lean API, Umpire Producer's embedded protocol identities, and namespace-bearing functional Case fixtures before moving the functional Driver (R6, R7).

**Size:** M
**Files:** `model/Temporal/CaseRuntime.lean`, `model/Temporal/API/**`, `model/Temporal/API.lean`, `tools/umpire/cmd/umpire-gen-lean-api/**`, `tools/umpire/cmd/umpire-gen-case-runtime-conformance/**`, new `tests/testcore/testpilot/testdata/**`, migration ledger
**Touches:** [model/Temporal/CaseRuntime.lean, model/Temporal/API/**, model/Temporal/API.lean, tools/umpire/cmd/umpire-gen-lean-api/**, tools/umpire/cmd/umpire-gen-case-runtime-conformance/**, tests/testcore/testpilot/testdata/**, .flow/artifacts/fn-69-extract-testpilot-from-umpire/migration-ledger.md]

### Approach
- Retarget the generated Lean API and CaseRuntime's embedded `InstructionOutcomeStatus` identity to `temporal.server.api.testpilot.v1` without changing Umpire authoring or Producer semantics.
- Extend the Case Runtime fixture generator with a separate transactional functional-fixture mode. It renders the existing `temporal-case-runtime get-system-info` and `async-nexus` entries, validates them through task 4's top-level Testpilot ingestion, and publishes exactly the two canonical outputs under `tests/testcore/testpilot/testdata`; retain the generator's exactly-six conformance manifest and old fixture root unchanged in this slice.
- Keep the old functional adapter, its Umpire fixtures, and remaining Umpire consumers intact in this slice so their focused tests remain green. Do not remove `caseartifact`, add translation, or claim the old runtime import graph is clean yet.
- Verify generated output determinism and model checks. The adapter's focused/live verification follows in task 6 after its imports and ownership move together.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/CaseRuntime.lean:30-38` — embedded protocol type identity
- `model/Temporal/API/Types.lean:1506-1792` — generated namespace baseline
- `tools/umpire/cmd/umpire-gen-lean-api/case_schema_test.go` — generated Lean schema checks
- `tools/umpire/cmd/umpire-gen-case-runtime-conformance/generate.go:20-118` — renderer, transactional publisher, output-root contract, and separate six-class manifest
- `model/Temporal/Tool/CaseRuntime.lean:11-21` — exact `get-system-info` and `async-nexus` renderer entries
- `tools/umpire/temporal/testdata/*.json` — namespace-bearing source fixtures
- `.plans/LEAN_GUIDELINES.md` — mandatory Lean authoring rules

## Acceptance
- [ ] Generated Lean API and CaseRuntime embedded protocol identities name Testpilot while Umpire authoring, Query, Producer, Program, and Contract semantics remain unchanged.
- [ ] The extended Case Runtime fixture generator has an explicit functional mode that transactionally renders, validates, and exclusively publishes `get-system-info-case.json` and `async-nexus-case.json` into `tests/testcore/testpilot/testdata`; every namespace-derived byte or identity difference matches the task-1 allowlist and migration ledger.
- [ ] The old Umpire fixtures remain byte-for-byte unchanged for the still-active adapter; no adapter/runtime imports, `caseartifact` callers, or generic conformance ownership move in this slice, and no alias, converter, registry, or fallback is introduced.
- [ ] Lean/API generation, generated-schema tests, model build/lint, focused fixture-generator tests for atomic publication/stale-file rejection, and deterministic functional fixture regeneration pass before task 6 moves and verifies the adapter.

## Done summary
Implemented the producer-side Testpilot cutover for the two functional Case Runtime fixtures. The generated Lean API now carries refined Testpilot descriptor identities; a dedicated typed Lean ProtoJSON serializer lowers the authored Umpire Case model directly into Testpilot data because recursive generated fields are descriptor-only `MessageRef`s, while the six legacy conformance fixtures and old Umpire adapter fixtures remain byte-identical. The functional generator now requires explicit `--mode functional`, validates every artifact through top-level Testpilot decode/pack, publishes the exact two-file transactional set under `tests/testcore/testpilot/testdata`, and records the migration evidence in the ledger.

Verification was green for Lean API generation, the focused generator packages, the Testpilot model package, the Case Runtime Lean build/tests, deterministic repeated functional fixture generation, `make lint-model` (261-target inventory), and `git diff --check`. The shared checkout remains uncommitted and unstaged at the user's request; evidence therefore records an empty commit list.

stage: impl-review - ran; synthetic Codex review verdict SHIP with zero findings; receipt `/tmp/impl-review-receipt-fn-69-extract-testpilot-from-umpire.5.json`
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: TMPDIR=/private/tmp mise exec -- go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-gen-case-runtime-conformance ./tools/umpire/cmd/umpire-gen-lean-api, make umpire-gen-lean-api, TMPDIR=/private/tmp mise exec -- go test -count=1 -tags test_dep ./common/testing/testpilot/..., cd model && mise exec -- lake build Temporal.CaseRuntimeTests temporal-case-runtime, deterministic functional fixture generation repeated twice with top-level Testpilot DecodeCaseProtoJSON and PackCaseProtoJSON validation, make lint-model, git diff --check
- PRs:
