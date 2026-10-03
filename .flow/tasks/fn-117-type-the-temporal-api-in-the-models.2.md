---
satisfies: [R1, R12]
---
# fn-117-type-the-temporal-api-in-the-models.2 Integrate stamped API generation and the chosen compile classpath

## Description
Turn task1's chosen mechanism into the gate's normal build path and settle the public author spelling before task3.

**Size:** M
**Files:** model/gate/Gate.scala and test/Gate.test.scala; Makefile; model/project.scala; model/lifter/project.scala; .plans/UMPIRE_MODULES.md; cmd/tools/getproto/** if needed for the proved linked descriptor assembly.
**Touches:** [model/gate/**, Makefile, model/project.scala, model/lifter/project.scala, .plans/UMPIRE_MODULES.md, cmd/tools/getproto/**, .flow/tmp/fn117-2/**]

### Approach
- Reuse Gate.scala's stamped IR ScalaPB jar generation rather than add a second build system. Package the selected API/Testpilot/well-known types once in model/gen and put them on the authoring and lifter classpaths. If task1 chose the catalog, integrate that same mechanism here. Keep model/gen generated and ignored.
- Carry the successful scratch descriptor assembly into a repository-owned generation path. Reuse/extend cmd/tools/getproto only as needed to supply the service and linked Testpilot/Umpire import closure without depending on ignored spike programs. Preserve the existing default descriptor target behavior; this is focused generation plumbing, not a broad drift gate. If Go source changes, add focused tests with -tags test_dep and run make lint-code-fast against origin/main.
- Key the stamp to all descriptor contents and generator/options/tool versions that affect the artifact. A descriptor edit invalidates the jar; a Model edit does not. Validate missing/stale descriptor diagnostics identify the target supplying the actual artifact, including the service descriptor. Follow existing atomic replacement/stale generation handling and test seams.
- Add focused gate tests for initial generation, descriptor/tool/options invalidation, missing descriptor, failed generation without a fresh stamp, and unchanged jar/stamp across a Model edit. Use scratch fixture inputs; do not corrupt or clear shared caches.
- Record actual compiling author forms from the successful spike in the parent API contract: ordered multi-schema types, unary method/request/response typing, optional nested selections, repeated and oneof traversal, symbolic typed operands, constant messages/enums/map entries and explicit unknown-origin payload roots. Do not leave the illustrative sketch as an implementation contract.
- Update the module map's approved exception: generic ScalaPB typing/runtime in the DSL, with no Temporal-specific imports there; generated API/Testpilot/well-known classes and their runtime in Models; metadata in lifter; generation in gate. Preserve Scala declarations/lifter/Go execution boundaries and add no service transport or behavior-hint layer.

### Investigation targets
**Required:**
- .plans/UMPIRE_TYPED_API_SPIKE.md
- model/gate/Gate.scala:63-235
- model/gate/test/Gate.test.scala:69-140
- model/project.scala and model/lifter/project.scala
- Makefile:684-725
- .plans/UMPIRE_MODULES.md:29-32,269-275,308-324

### Quick commands
mise exec -- scala-cli test model/gate; CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks; mise exec -- make lint-model; affected cmd/tools/getproto tests with -tags test_dep and make lint-code-fast if its source changes. Reuse unaffected Go evidence when hashes/commands/environment apply.

### Execution constraints
Follow MILESTONES.md verification reuse, preserve comments, no commits/staging/worktrees/push. Only one gate or generation process at a time. No Go lowering or IR schema change, no broad API drift/CI gate.

## Acceptance
- [ ] The gate builds the chosen stamped artifact/classpaths, handles incomplete or stale inputs with actionable errors, and reuses the artifact after a Model edit.
- [ ] Focused invalidation/failure tests and relevant model/gate lint checks pass; existing lifted outputs remain equal.
- [ ] The exact author contract and approved module imports are recorded before downstream implementation.

## Done summary
Integrated the complete linked Temporal API/Testpilot ScalaPB jar into the model gate, Make prerequisites, and authoring/lifter classpaths. The gate checks `make proto/api.binpb`, compiles current Testpilot/Umpire `.proto` descriptors, and refuses linked generated-Go descriptors that differ with a `make protoc` remedy. It hashes the complete linked closure, generation source/configuration, options and tool versions; it atomically replaces the jar and writes a stamp only after success. An unchanged Model edit reuses the jar. The module map records the approved import exception, and `author-contract.md` supplies the canonical parent contract for tasks 3–5.

Review findings were addressed: a Testpilot `.proto` edit with unchanged external `api.binpb` is detected against current source descriptors, including newly added internal files; a Gate source edit changes the stamp, builds once, then reuses the jar. The focused Go regression holds external descriptor bytes fixed while changing internal descriptors. Real generation passed, and `make -q model/gen/api-scalapb.jar` returned 0 after rebuild. Final verification passed: Scala gate suite, getproto Go tests, real API jar build, check-mode model gate, model lint, and fast Go lint. All 30 pre-edit IR/Case/fixture hashes still match. The same independent review session reached SHIP.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: mise exec -- scala-cli test model/gate (exit 0), CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- go test -json -tags test_dep ./cmd/tools/getproto -count=1 (exit 0), CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- make model/gen/api-scalapb.jar (exit 0), mise exec -- make -q model/gen/api-scalapb.jar (exit 0), CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks (exit 0), CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks (exit 0), mise exec -- make lint-model (exit 0), CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- make GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main lint-code-fast (exit 0), sha256sum -c .flow/tmp/fn117-2/artifacts-before.sha256 (30 files, exit 0), git diff --check (exit 0), Independent implementation review SHIP; conductor verified 49 pinned source/evidence hashes and 30 preserved artifact hashes
- PRs: