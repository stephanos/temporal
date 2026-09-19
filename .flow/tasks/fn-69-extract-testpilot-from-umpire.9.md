---
satisfies: [R1, R6, R7, R8]
---
# fn-69-extract-testpilot-from-umpire.9 Refine the Testpilot protobuf data model

## Description
Refine the copied Umpire-compatible protobuf schema into the public Testpilot v1 data model before the facade or producers adopt it. Preserve Case Runtime behavior, admission outcomes, execution bounds, event ordering, diagnostics, cleanup precedence, and Verdict semantics; compatibility with the temporary descriptor introduced by task .2 is not required because it has no supported consumers.

**Size:** L
**Files:** `proto/internal/temporal/server/api/testpilot/v1/*.proto`, `api/testpilot/v1/**`, `common/testing/testpilot/**`, focused generators and compatibility tests

### Approach
- Inventory every message, enum, reference, and cross-file dependency; retain only concepts interpreted by Testpilot or required by a concrete Temporal case.
- Split broad files into cohesive modules around Case/provenance, values, expressions, instructions, Programs, instruction outcomes, Run events, Contracts, and Runs. Prefer small files, but combine modules when the dependency boundary would otherwise be artificial.
- Apply a uniform human-facing vocabulary: authored definitions use `*Spec` or `*Definition`, runtime facts use `*Outcome` or `*Result`, references use `*Ref`, categorical enums use `*Kind`, lifecycle/result enums use `*Status`, and identifiers use `*_id`. Remove stale Umpire and Host wording in favor of Testpilot-neutral language and Driver where environment authority is meant.
- Reassess abstractions rather than mechanically rename them: remove or make opaque producer-only definition taxonomy, remove speculative roles without a concrete use case, separate Program and Contract expression scopes when their legal references differ, and use `oneof` or derived fields for mutually exclusive slot and activation states.
- Replace wrapper messages that add no domain meaning with protobuf presence/cardinality features where supported by the generators. Preserve wrappers when absence and an empty value have distinct semantics that cannot otherwise be represented safely.
- Regenerate owned Go descriptors and adapt the extracted Testpilot core and focused tests. Replace exact Umpire/Testpilot descriptor parity with semantic migration checks covering the intentional schema changes; do not add a runtime translation or compatibility layer.

### Verification
- `make proto`
- `go test -count=1 -tags test_dep ./common/testing/testpilot/...`
- Existing focused Case Runtime conformance and regression checks applicable before producer cutover

## Acceptance
- [ ] Proto files are organized around cohesive domain concepts; no file remains a grab bag of unrelated values, schemas, expressions, instructions, events, and verdicts.
- [ ] Public names follow one documented scheme, read naturally to humans, and contain no stale Umpire or Host vocabulary where Testpilot or Driver is intended.
- [ ] Producer-specific or speculative concepts not interpreted by Testpilot are removed, made opaque provenance, or justified by a concrete admitted Case.
- [ ] Program and Contract expression/reference scopes cannot silently accept each other's illegal sources, and contradictory slot or entrypoint states are structurally impossible where protobuf can express the invariant.
- [ ] Case Runtime behavior, admission decisions, resource bounds, event ordering, diagnostics, cleanup precedence, and Verdict semantics remain covered and unchanged; no new instruction, evidence source, effect, retry, or runtime capability is introduced.
- [ ] Generated Testpilot API, descriptors, extracted core, and focused tests use the refined schema without aliases or a runtime translation layer; every intentional incompatibility with the temporary task-.2 descriptor is recorded and tested.

## Done summary
Refined the temporary Testpilot descriptor into eight cohesive domain files, regenerated its Go API, and adapted the extracted private core to consume the new schema directly while preserving runtime semantics. Added descriptor invariants and a migration-ledger record for every intentional incompatibility; the user-owned checkout remains unstaged and uncommitted.

baseline: green (`go test -count=1 -tags test_dep ./common/testing/testpilot/...`)
stage: impl-review - ran | SHIP | receipt `/tmp/impl-review-receipt-fn-69-extract-testpilot-from-umpire.9.json`

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: make proto, go test -count=1 -tags test_dep ./common/testing/testpilot/..., git diff --check -- .flow/artifacts/fn-69-extract-testpilot-from-umpire/migration-ledger.md common/testing/testpilot proto/internal/temporal/server/api/testpilot/v1 api/testpilot/v1
- PRs:
