---
satisfies: [R2, R3]
---
# fn-145-modularize-the-umpire-ir-schema.1 Teach generation and descriptor checks the full schema closure

## Description
Prepare the generation, staleness and descriptor-equivalence surfaces for R2 and R3 before declarations move. This is the proof point because the split is unsafe until every imported schema is generated, hashed and compared.

**Size:** M
**Files:** `model/check/Gate.scala`, `Makefile`, `proto/api-linter.yaml`, `tools/umpire/ir/schema_test.go`, `cmd/tools/getproto/main.go`, `cmd/tools/getproto/linked_test.go`, `model/check/test/Gate.test.scala`
**Touches:** [model/check/Gate.scala, Makefile, proto/api-linter.yaml, tools/umpire/ir/schema_test.go, cmd/tools/getproto/main.go, cmd/tools/getproto/linked_test.go, model/check/test/Gate.test.scala]

### Approach
- Replace the single-schema input assumption with deterministic closure discovery and hashing.
- Extend the descriptor ledger to compare the declaration union. Build a temporary multi-file fixture before production extraction; imported Scala declarations must generate and compile under the real ScalaPB path, not only the stubbed protoc test.
- Update `cmd/tools/getproto` to exclude the complete owned IR closure from the separate API Scala jar and to compare every owned linked Go descriptor against its current source. Add a stale-imported-descriptor/unchanged-root regression.
- Extend `model/check/test/Gate.test.scala` with imported-source stamp invalidation, missing imported-output failures and descriptor comparison outside the root file.
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
go test -tags test_dep ./tools/umpire/ir/... ./cmd/tools/getproto/...
```
## Acceptance
- [ ] R2 closure discovery, hashing and stale-output behavior are pinned.
- [ ] R3's pre-split descriptor and wire baseline is recorded without broad CI expansion.
- [ ] `go test -tags test_dep ./tools/umpire/ir/...` passes.
- [ ] The focused model gate generation checks pass.
- [ ] A real multi-file fixture generates compilable imported Scala declarations; imported edits and missing outputs fail the correct gates.
- [ ] Stale imported linked descriptors fail even when the root is unchanged; the two Scala jars contain no duplicate IR closure classes.
## Done summary
Generation, hashing, freshness checks, linked descriptors, API-jar exclusions, and schema declaration ledgers now cover the complete owned Umpire IR proto closure. Focused regression coverage includes a real two-file ScalaPB generation and Scala compilation proof for an imported declaration.

Tier: session (jev-unavailable(no_key))
stage: impl-review - ran [2026-10-09T04:01:18Z..2026-10-09T04:03:41Z] (codex: NEEDS_WORK -> SHIP)

Baseline: red (`go test -tags test_dep ./tools/umpire/ir/... ./cmd/tools/getproto/...` failed before edits because Batch 1's intentionally stale checked-in IR still names `umpire.Step`; production IR/Case regeneration is deferred to fn-145.4).

Verification:

- PASS: `mise exec -- scala-cli test --suppress-outdated-dependency-warning model/check --test-only umpire.check.GateSuite`
- PASS: `go test -tags test_dep ./cmd/tools/getproto -run 'TestLinkedModelDescriptors'`
- PASS: `go test -tags test_dep ./tools/umpire/ir -run 'TestSchema(Rename|Declarations)'`
- PASS: `mise exec -- scala-cli test --suppress-outdated-dependency-warning model/check`
- PASS: `mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check model/check/Gate.scala model/check/test/Gate.test.scala`
- PASS: `gofmt -d cmd/tools/getproto/main.go cmd/tools/getproto/linked_test.go tools/umpire/ir/schema_test.go` (empty diff)
- EXPECTED BATCH RED: `go test -tags test_dep ./tools/umpire/ir/... ./cmd/tools/getproto/...` reproduced the unchanged pre-edit `umpire.Step` admission failures; `cmd/tools/getproto` passed. The Batch 1 contract defers regeneration and full batch gates to fn-145.4.
- INCONCLUSIVE: `make lint-model-check` returned zero but Scalafix emitted a JDK 27 `NoSuchFieldException: path`, so it is not counted as green.
- INHERITED RED: `make lint-code-fast` reported 741 branch-wide findings outside this task's touched paths; its incidental formatter changes were restored to the clean base before commit.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 7a5ac5ed51b79e6410fd950e06ba617fbf36ada4, d2b174271d2832484181a637ad794b154b92878f
- Tests: mise exec -- scala-cli test --suppress-outdated-dependency-warning model/check --test-only umpire.check.GateSuite, go test -tags test_dep ./cmd/tools/getproto -run 'TestLinkedModelDescriptors', go test -tags test_dep ./tools/umpire/ir -run 'TestSchema(Rename|Declarations)', mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check model/check/Gate.scala model/check/test/Gate.test.scala, EXPECTED BATCH RED (fn-145.4 regeneration deferred): go test -tags test_dep ./tools/umpire/ir/... ./cmd/tools/getproto/...
- PRs: