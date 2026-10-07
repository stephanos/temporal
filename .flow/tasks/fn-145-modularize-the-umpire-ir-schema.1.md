---
satisfies: [R2, R3]
---
# fn-145-modularize-the-umpire-ir-schema.1 Teach generation and descriptor checks the full schema closure

## Description
Prepare the generation, staleness and descriptor-equivalence surfaces for R2 and R3 before declarations move. This is the proof point because the split is unsafe until every imported schema is generated, hashed and compared.

**Size:** M
**Files:** `model/check/Gate.scala`, `Makefile`, `proto/api-linter.yaml`, `tools/umpire/ir/schema_test.go`
**Touches:** [model/check/Gate.scala, Makefile, proto/api-linter.yaml, tools/umpire/ir/schema_test.go]

### Approach
- Replace the single-schema input assumption with deterministic closure discovery and hashing.
- Extend the existing descriptor ledger to compare the union of declarations while the schema is still one file.
- Pin cold invalidation, missing-output failure and unchanged warm-build reuse without adding a CI workflow.

### Investigation targets
**Required** (read before coding):
- `model/check/Gate.scala:119-181` - schema stamp and ScalaPB generation inputs
- `model/check/Gate.scala:399-420` - stale generated-output checks
- `tools/umpire/ir/schema_test.go:125-210` - descriptor and wire compatibility ledger
- `Makefile:700-715` - IR schema prerequisites
- `proto/api-linter.yaml:52-60` - Umpire schema exemption

### Key context
The generated-API drift ledger permits focused closure checks only. Preserve default-empty fingerprints and reject missing declarations before comparing behavior.


### Quick commands

```bash
go test -tags test_dep ./tools/umpire/ir/...
```

## Acceptance
- [ ] R2 closure discovery, hashing and stale-output behavior are pinned.
- [ ] R3's pre-split descriptor and wire baseline is recorded without broad CI expansion.
- [ ] `go test -tags test_dep ./tools/umpire/ir/...` passes.
- [ ] The focused model gate generation checks pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
