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
Finalized the existing deterministic evaluation-contract packer/admission implementation after auditing commits `36fe448a877e887f5df4045f94ecc81f2b3862af` and `636f17f3c649853127d2bd6ab73826a32e6edf04`, their task receipt, and the authoritative Codex SHIP review. Current focused unit, race, fuzz, vet, and package lint verification is green; no product edit was warranted, and the unrelated user-owned config/schema modifications remain untouched.

The parent Quick commands that exercise later fn-28 tasks were not redundantly rerun; the known repository-wide `lint-model` OOM, `lint-code` findings, and later-task regression parity failure remain inherited and outside this task's acceptance surface.

baseline: green (focused evaluationcontract unit suite; no task code changed during finalization)

GATE_CLASSIFY_FULL: unrelated user-owned config/development.yaml working-tree modification

stage: impl-review - ran [2026-09-01T18:13:22Z..2026-09-01T18:16:51Z] (authoritative SHIP receipt reused after empty finalization diff)
## Evidence
- Commits:
- Tests: baseline: green - TMPDIR=$PWD/.flow/tmp/fn28_2_tmp GOTMPDIR=$PWD/.flow/tmp/fn28_2_tmp go test -count=1 -tags test_dep ./tools/umpire/evaluationcontract/..., TMPDIR=$PWD/.flow/tmp/fn28_2_tmp GOTMPDIR=$PWD/.flow/tmp/fn28_2_tmp go test -race -count=1 -tags test_dep ./tools/umpire/evaluationcontract/..., TMPDIR=$PWD/.flow/tmp/fn28_2_tmp GOTMPDIR=$PWD/.flow/tmp/fn28_2_tmp go test -count=1 -tags test_dep ./tools/umpire/evaluationcontract/... -run '^$' -fuzz '^FuzzAdmitRejectsSingleByteContractMutations$' -fuzztime=3s, TMPDIR=$PWD/.flow/tmp/fn28_2_tmp GOTMPDIR=$PWD/.flow/tmp/fn28_2_tmp go vet -tags test_dep ./tools/umpire/evaluationcontract/..., TMPDIR=$PWD/.flow/tmp/fn28_2_tmp GOTMPDIR=$PWD/.flow/tmp/fn28_2_tmp .bin/golangci-lint-v2.13.1 run --build-tags 'test_dep' --timeout 10m --config=.github/.golangci.yml ./tools/umpire/evaluationcontract/..., GATE_CLASSIFY_FULL: unrelated user-owned config/development.yaml working-tree modification, AUTHORITATIVE_REVIEW_SHIP: 62c8a5b085b174c4ea4c62951c19fa66c7c423c7..636f17f3c649853127d2bd6ab73826a32e6edf04
- PRs: