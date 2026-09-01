---
satisfies: [R1, R8]
---

# fn-28-portable-evaluation-contract-and.2 Implement deterministic contract packing and admission
## Description

Build the semantic-free Go codec that converts Lean-produced canonical ProtoJSON into deterministic protobuf bytes and strictly admits those bytes before runtime I/O.

**Size:** M
**Files:** `tools/umpire/evaluationcontract/**`
**Touches:** [`tools/umpire/evaluationcontract/**`]

### Approach
- Use generated protobuf types and deterministic protobuf marshaling; keep checksums outside their own digest input and require canonical re-encoding equality.
- Reject unknown fields, invalid enum values, unsupported major versions/operators, duplicates, noncanonical ordering, invalid Limits, crossed bindings, and checksum/fingerprint drift.
- Keep the packer structural: it may validate schema invariants but cannot select clauses, synthesize expected behavior, or consult model definitions.

### Investigation targets

**Required** (read before coding):
- Parent spec and task `.1` schema.
- `tools/umpire/artifact` and `tools/umpire/internal/artifactv2` canonical admission patterns.
- Existing protobuf deterministic marshal/descriptor helpers in `tools/umpire`.

## Acceptance
- [ ] Canonical ProtoJSON packs to stable deterministic protobuf bytes and re-admits byte-identically.
- [ ] Representative one-field, unknown-field/operator, ordering, checksum, crossed-binding, N/N+1, and fuzz mutations fail closed before execution.
- [ ] Focused unit, race, fuzz, and lint checks pass using `require` assertions.

## Done summary
Implemented canonical ProtoJSON packing and deterministic protobuf admission for portable Umpire evaluation contracts, including bounded structural validation, domain-separated checksums, canonical-order enforcement, closed binding checks, and typed admission errors. Added focused unit, race, fuzz, and mutation coverage for one-field drift, unknown fields/operators, invalid enums and Limits, crossed fingerprints, coordinate shapes, cyclic ordering, duplicate/contradictory mappings, checksums, canonical bytes, and exact N/N+1 collection bounds.

Verification: focused unit, race, fuzz, vet, and golangci-lint checks passed. `make lint-code` reproduced the approved inherited 1373 repository-wide findings with zero findings under `tools/umpire/evaluationcontract`; the Lean lowering, portable interpreter/executor aggregate, and tagged integration Quick commands remain deferred to tasks `.3`, `.4`/`.6`, and `.9` respectively.

stage: impl-review - ran [2026-09-01T18:13:22Z..2026-09-01T18:16:51Z]
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 36fe448a877e887f5df4045f94ecc81f2b3862af, 636f17f3c649853127d2bd6ab73826a32e6edf04, bd499e9d6d84f7cc8402e77a878330504186869c
- Tests: baseline: green (make proto), go test -count=1 -tags test_dep ./tools/umpire/evaluationcontract/..., go test -race -count=1 -tags test_dep ./tools/umpire/evaluationcontract/..., go test -count=1 -tags test_dep ./tools/umpire/evaluationcontract/... -run '^$' -fuzz '^FuzzAdmitRejectsSingleByteContractMutations$' -fuzztime=3s, go vet -tags test_dep ./tools/umpire/evaluationcontract/..., .bin/golangci-lint-v2.13.1 run --build-tags 'test_dep' --timeout 10m --config=.github/.golangci.yml ./tools/umpire/evaluationcontract/..., INHERITED_RED: make lint-code (1373 pre-existing repository-wide findings; zero tools/umpire/evaluationcontract findings), DEFERRED(task .3): cd model && mise exec -- lake build Temporal.Tool.PortableEvaluationContractTests, DEFERRED(tasks .4/.6): go test -count=1 -tags test_dep ./tools/umpire/evaluationcontract/... ./tools/umpire/portableevaluation/... ./tools/umpire/executor/..., DEFERRED(task .9): go test -count=1 -tags 'test_dep integration' ./tests -run '^TestUmpirePortableCanaryExecutor$'
- PRs:
