---
satisfies: [R8]
---
# fn-71-standalone-lean-testpilot-protocol.7 Retire parallel ownership and verify the Testpilot cutover

## Description
Finish the cutover by removing parallel handwritten protocol and serializer ownership, reconciling active documentation, and running the complete migration gates. Compatibility names may remain only as thin aliases or forwarders to generated types and the sole library codec with a stated removal point.

**Size:** M
**Files:** `model/Umpire/Case/{Value,Program,Run,Contract,ProtoJSON}.lean`, `model/Umpire/Case.lean`, `model/Temporal/Testpilot/TestpilotProtoJSON.lean`, `model/Umpire/ARCHITECTURE.md`, `model/ARCHITECTURE.md`, `model/README.md`, `tests/testcore/testpilot/README.md`, `.plans/UMPIRE4_SPEC.md`, `.plans/UMPIRE4_COMPONENTS.md`, affected public module docstrings and import tests
**Touches:** [`model/Umpire/Case/{Value,Program,Run,Contract,ProtoJSON}.lean`, `model/Umpire/Case.lean`, `model/Temporal/Testpilot/TestpilotProtoJSON.lean`, `model/Umpire/ARCHITECTURE.md`, `model/ARCHITECTURE.md`, `model/README.md`, `tests/testcore/testpilot/README.md`, `.plans/UMPIRE4_SPEC.md`, `.plans/UMPIRE4_COMPONENTS.md`, affected public Lean docstrings/import tests]

### Approach
- Remove handwritten Testpilot wire-shaped structures and independent Umpire/Temporal serializer bodies after every producer has migrated.
- Retain old module names only where source compatibility materially requires aliases/forwarders; cover equivalent output and document their removal point.
- Update active architecture, component, README, and public module documentation to name the `.proto` schema, generated protocol, authoring facade, `Protobuf.Json`, Umpire provenance, and Go Prepare owners.
- Leave historical design records, unrelated generated catalogs, protobuf schemas, and broader CI expansion untouched.
- Run focused protocol/generation/import/fixture/Go checks, then the full model build, `make lint-model`, and `make lint-code GOLANGCI_LINT_FIX=false`; fix introduced failures and evidence any inherited baseline failure.

### Investigation targets
**Required** (read before coding):
- all migrated compatibility modules
- `model/Umpire/ARCHITECTURE.md`
- `model/ARCHITECTURE.md`
- `model/README.md`
- `tests/testcore/testpilot/README.md`
- `.plans/UMPIRE4_SPEC.md`
- `.plans/UMPIRE4_COMPONENTS.md`

**Optional** (reference as needed):
- `.flow/memory/declined/generated-api-drift-verification.md`
## Acceptance
- [ ] No parallel handwritten Testpilot wire type or independent Umpire/Temporal field serializer remains.
- [ ] Any retained compatibility surface delegates directly to generated types or `Testpilot.ProtoJSON` and has equivalent-output coverage plus a removal point.
- [ ] Active documentation consistently states final schema, protocol, authoring, codec, provenance, and Go admission ownership.
- [ ] Historical/generated/out-of-scope artifacts remain untouched unless an owned generator legitimately changes their output.
- [ ] Focused and full specified gates are recorded with no new proof placeholders or trust dependencies; introduced failures are fixed and inherited failures have concrete baseline evidence.
## Done summary
Retired the remaining parallel handwritten Testpilot protocol and serializer ownership. Umpire compatibility surfaces are now generated-type aliases or a direct `Testpilot.ProtoJSON` forwarder with colocated equivalence coverage and explicit removal points. `Umpire.Case.Compiler` retains the required source-bound validation, unsupported-lowering diagnostics, exact provenance, and final Case assembly over generated protocol values; it introduces no parallel wire representation. Active documentation assigns schema, authoring, codec, provenance, producer lowering, compiler assembly, and Go admission to their final owners.

Focused compatibility/compiler/producer, full model, generation/fixture drift, import-graph, scoped Go, and model lint gates passed. The final quality pass restored the generated-value compiler seam required by R6, added focused generated assembly and exact-error coverage, documented the public authoring/codec policies, declared both Make targets phony, and corrected stale ownership descriptions. `make lint-code GOLANGCI_LINT_FIX=false` was attempted twice after implementation but the host temp volume exhausted during Go package loading before source diagnostics; the accepted pre-existing baseline remains 1361 unrelated findings, recorded in `/tmp/fn71-baseline-lint-code.log` and `/tmp/fn71-task4-baseline-lint-code.log`.

stage: impl-review - ran (codex:gpt-5.6-sol:medium, SHIP)
stage: quality-audit - fixed compiler-removal correctness finding and all standards findings
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: (cd model && mise exec -- lake build Umpire.Case.CompilerTests Temporal.Feature.Nexus3.Tests Temporal.TestpilotTests TestpilotTests temporal-testpilot), TMPDIR=/private/tmp CGO_ENABLED=0 make umpire-check-case-runtime-conformance, TMPDIR=/private/tmp CGO_ENABLED=0 mise exec -- go test -p=1 -count=1 -tags test_dep ./common/testing/testpilot/... ./tools/umpire/cmd/umpire-gen-case-runtime-conformance ./tests/testcore/testpilot/..., (cd model && mise exec -- lake build Testpilot TestpilotTests UmpireTests TemporalModelTests temporal-testpilot), (cd model && mise exec -- lake exe modelLintTests), make lint-model, INHERITED_RED: make lint-code GOLANGCI_LINT_FIX=false - accepted baseline has 1361 unrelated findings; two task-7 attempts stopped during package loading on host temp-volume exhaustion before source diagnostics, impl-review: SHIP (codex:gpt-5.6-sol:medium; /tmp/fn71-task7-impl-review.json)
- PRs:
