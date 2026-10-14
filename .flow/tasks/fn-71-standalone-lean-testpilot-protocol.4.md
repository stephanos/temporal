---
satisfies: [R6]
---
# fn-71-standalone-lean-testpilot-protocol.4 Move Umpire provenance and compiler onto Testpilot

## Description
Make Umpire a producer of generated Testpilot Cases while retaining sole ownership of semantic lowering and its provenance payload. Preserve checked-input and unsupported-lowering behavior as compiler output moves through the neutral authoring facade.

**Size:** M
**Files:** `model/Umpire/Case/Compiler.lean`, `model/Umpire/Case/CompilerTests.lean`, `model/Umpire/Case/Provenance.lean`, `model/Umpire/Case.lean`, current Umpire Case producers and tests
**Touches:** [`model/Umpire/Case/Compiler*.lean`, `model/Umpire/Case/Provenance.lean`, `model/Umpire/Case.lean`, affected `model/Umpire/**/*Test*.lean`, affected `model/Umpire/Examples/**`]

### Approach
- Freeze the logical and byte representation of current Umpire `producerData`, including definitions, fingerprints, sources, Known Gaps, ordering, and optional subject/detail fields.
- Give Umpire sole responsibility for encoding that payload and construct generated provenance and Cases through `Testpilot.Authoring`.
- Migrate Umpire compiler and producers without exposing unchecked semantic evaluators or changing typed `LoweringError` behavior.
- Keep Testpilot and Go admission opaque to Umpire payload contents; malformed producer-owned JSON remains an Umpire concern rather than a generic protocol rejection.
- Retain temporary old-module compatibility only where required to keep the staged consumer migration buildable.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Case/Compiler.lean`
- `model/Umpire/Case/CompilerTests.lean`
- `model/Umpire/KnownGap.lean`
- current provenance encoding in `model/Temporal/Testpilot/TestpilotProtoJSON.lean`
- `.flow/memory/bug/integration/keep-raw-semantics-behind-checked-input-2026-09-05.md`
## Acceptance
- [ ] Compiler and Umpire Producers return generated Testpilot Cases through the public authoring API.
- [ ] Exact Umpire producer bytes preserve definitions, fingerprints, sources, Known Gaps, list order, and optional fields.
- [ ] Checked-input rules and all existing unsupported-lowering diagnostics remain typed and unchanged.
- [ ] Generic Testpilot and Go paths neither require nor interpret the Umpire payload schema.
- [ ] Intermediate compatibility access creates no second protocol type or serializer implementation.
## Done summary
Umpire now compiles checked inputs into generated Testpilot Cases exclusively through `Testpilot.Authoring`, with the exact legacy producerData bytes encoded in the Umpire-owned provenance module. Existing unsupported-lowering diagnostics remain unchanged; fixed-width protobuf numbers are range checked with typed `LoweringError`s, and current Temporal/Nexus3 consumers retain compile-only compatibility pending task 5's fixture migration.

Validation passed for focused and aggregate Lean builds, `modelLintTests`, `make lint-model`, direct Lifecycle target, and Testpilot/Umpire Go packages. `make umpire-check-case-runtime-conformance` reached the expected task-5-only fixture presentation mismatch while retaining identical producerData base64; `make lint-code GOLANGCI_LINT_FIX=false` remains inherited red with the baseline 1361 unrelated findings. The initial post-change model lint attempt completed all 351 build targets but stalled in built-in lint while disk was full; after cache cleanup the complete command passed.

stage: impl-review - ran [2026-09-06T23:03:00-07:00..2026-09-06T23:23:00-07:00]
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: (cd model && mise exec -- lake build Umpire.Case.CompilerTests Temporal.Feature.Nexus3.Tests Temporal.TestpilotTests Temporal.Testpilot.TestpilotProtoJSON temporal-testpilot), (cd model && mise exec -- lake build Testpilot TestpilotTests UmpireTests TemporalModelTests temporal-testpilot), (cd model && mise exec -- lake exe modelLintTests), (cd model && mise exec -- lake build Temporal.Feature.Nexus.Lifecycle.TargetTests), make lint-model, TMPDIR=/private/var/folders/k8/tl9x33wj7mz_cw1z_420xzs80000gn/T CC=/usr/bin/clang CXX=/usr/bin/clang++ mise exec -- go test -count=1 -tags test_dep ./common/testing/testpilot/... ./tools/umpire/cmd/umpire-gen-case-runtime-conformance ./tests/testcore/testpilot/..., EXPECTED_TASK5_RED:make umpire-check-case-runtime-conformance - generated ProtoJSON presentation differs from old fixtures while Umpire producerData base64 is identical, INHERITED_RED:make lint-code GOLANGCI_LINT_FIX=false - baseline has 1361 unrelated findings
- PRs:
