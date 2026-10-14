---
satisfies: [R2, R3, R4, R5, R6, R7, R8, R9, R10, R11, R12]
---
# fn-117-type-the-temporal-api-in-the-models.8 Retire free-text proto forms and close with regression checks and documentation

## Description
Remove the migration bridge, enforce the final author contract and complete the spec's validation and documentation once.

**Size:** M
**Files:** model/umpire/Action.scala and realize modules; model/lifter/Declarations.scala and Realizations.scala; model/gate/Gate.scala/tests; model README/SEMANTICS and module map; focused fixture tests as needed.
**Touches:** [model/umpire/Action.scala, model/umpire/realize/**, model/lifter/Declarations.scala, model/lifter/Realizations.scala, model/lifter/test/**, model/lifter/testdata/**, model/gate/**, model/README.md, model/SEMANTICS.md, .plans/UMPIRE_MODULES.md, .flow/tmp/fn117-8/**]

### Approach
- Remove the old string schema/method/path/enum/Proto/ProtoField/ProtoEntry author routes and corresponding lifter branches once task7 has no author caller; retire its explicitly retained old* Typed.scala parity controls in the same change and preserve equivalent typed-output assertions without legacy constructors. Keep Model-owned IDs and map data keys as values. Do not add another deprecated escape hatch.
- Add the required focused gate regression check for proto package/message/method/field/enum string literals in Models. Inspect typed source/declaration context or descriptor-backed categories so Definition/role/evidence IDs and user text/data keys are not false positives. Cover each banned category and at least Definition ID, role ID, evidence ID and Payload.metadata encoding data as allowed controls. Include generated strings from helper concatenation insofar as they can reach an author API; retired constructors must not compile.
- Verify the complete R10 matrix: misspelled field, wrong message/root, wrong literal/symbolic value, unknown enum; nested optional/repeated/oneof paths and typed map constant construction. Keep Go's independent validation/tests unchanged for edited or older IR.
- Run required closure coverage once, reusing applicable prior passing results under MILESTONES.md: full model gate, lint-model, complete tooling Go tests with -json/-tags test_dep/serial package and subtest settings, whole-tools vet, lint-code-fast, Cases/fixtures/canary. Separate complete Go coverage permits the documented skip-go-checks Scala gate. Record exact command/environment/source hashes/new and reused results, exit and wall time; backend comparison remains owner-deferred. Broaden only for a concrete invalidation.
- Verify all six IRs and positive fixture data equal the spec baseline except positions, with Case JSON bytes/frozen goldens strict, manifest Position metadata allowing only corresponding source-line updates and no schema/lowering/validation code change. Re-measure Model-edit warm compilation and artifact reuse on the same setup as R1; report before/after warm/cold measurements and removed string counts by category.
- Update README's authoring examples, build/cache/error guidance and module map once with the actual final syntax and permitted runtime/generated class imports. SEMANTICS explains unchanged path/operand meaning and Go execution; Scala never sends messages. List genuinely unknown dynamic operands with reasons/retained Go checks. Supply MILESTONES facts for the conductor.

### Investigation targets
**Required:**
- model/README.md:130-155,274-329 and model/SEMANTICS.md:295-334,384-400
- .plans/UMPIRE_MODULES.md:29-32,269-275,308-324
- model/gate/Gate.scala and test/Gate.test.scala
- model/umpire/Action.scala and realize/Realize.scala
- model/lifter/Declarations.scala and Realizations.scala
- tools/umpire/lower/descriptor.go and realization.go
- Makefile:684-732

### Quick commands
Full closure model gate coverage, mise exec -- make lint-model, Go tooling tests/vet with -tags test_dep and serial settings, GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main mise exec -- make lint-code-fast, make umpire-check-cases umpire-check-fixtures canary-check-case, git diff --check. Record which current-source passing evidence is reused.

### Execution constraints
Owner approved scoped local fn-117 checkpoint commits on 2026-10-03; the conductor may stage and commit this task and its fixes. No push or worktrees. Preserve comments, no broad API drift/CI expansion, Go validation or IR schema change. No fn-112/114 showcase/deduplication or fn-118 behavior metadata. No simultaneous heavy suites/generation; no extra audits or caching framework. Conductor owns spec completion review and close.
The final compiler-refusal matrix includes both wrong-request poll assignment and wrong-projected-value poll condition controls; ensure string evidence IDs cannot bypass the known typed link.

### Review context
The owner supplied checkpoint f3ed2083b8 after approving the local checkpoint. Review the committed fn-117 task-8 implementation relative to d72de724b8, including the pinned-Run/companion/test repair documented in `.flow/tmp/fn117-record-repair/evidence.md` and `.flow/tmp/fn117-8/summary.md`. The same commit also contains unrelated fn-112/114/118/119/120 planning, owner VISION/TLA/KNOWN_BUG edits and the requested test relocation; do not treat those as task-8 implementation. Passing current-source gates are recorded in `.flow/tmp/fn117-8/evidence.json`; no production Go validation or IR schema changed.

### Review repair checkpoint
The committed-range fan-out in `.flow/review-fanout/7adbf24cdc6f4870802b4cfd110521e9/` identified R8 scanner false positives/misses and an R12 README import omission. Its finalization was refused because the unrelated owner VISION commit `4a4da7aab1` moved HEAD during review; those draws have no finalized receipt and do not establish SHIP. Scoped repair checks and regression evidence are in `.flow/tmp/fn117-review-fix/summary.md` and `evidence.json`. Retain the applicable original closure evidence for unchanged Go/runtime/schema/artifact scopes. Final implementation review and spec completion remain required before this task/spec can be marked complete.

## Acceptance
- [ ] All free-text proto author routes are removed, and the focused Model check rejects proto names while allowing Model IDs and data values; the complete positive/negative typing matrix passes.
- [ ] Required full coverage, frozen goldens, lint and artifact checks pass with applicable reused evidence identified; no Go validation/schema/meaning change exists.
- [ ] Final author/build/cache docs, exact removed-string counts, before/after warm/cold times and unknown-operand list are recorded; a Model edit reuses API generation/compilation.

## Done summary
Retired every free-text proto author route (schema/method/path/enum/Proto/ProtoField/ProtoEntry) and its lifter branches; Models use only the typed API. Added the `ProtoLiterals` gate check that refuses proto package/message/method/field/enum string literals while allowing Model IDs and data values, repaired after the committed-range review (comment/string masking, multiline declarations, dotted paths, bound IDs). The full R10 typing matrix refuses in `typedInvalid` and `retiredInvalid`. Six IRs, fixtures, Cases and manifest are unchanged apart from positions; no Go validation, proto or IR schema change. README, SEMANTICS and the module map show the final syntax.

Final review: claude:claude-fable-5-1:high, range d72de724b8..13c8ad07ef, verdict SHIP (codex unavailable in this environment). Two P3 test-rigor notes were left as is: the retired-constructor control asserts `>= 12` positions rather than the exact list, and the forged-carrier controls refuse twice because they wrap retired constructors.

Evidence: .flow/tmp/fn117-8/evidence.json, .flow/tmp/fn117-review-fix/evidence.json, .flow/tmp/fn117-record-repair/evidence.md.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: f3ed2083b8, 13c8ad07ef
- Tests: mise exec -- scala-cli test --suppress-outdated-dependency-warning model/gate (exit 0), mise exec -- scala-cli test --suppress-outdated-dependency-warning model/lifter (exit 0), CC=/usr/bin/gcc GOMEMLIMIT=4500MiB mise exec -- make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks (exit 0), CC=/usr/bin/gcc GOMEMLIMIT=4500MiB mise exec -- make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks (exit 0), mise exec -- make lint-model (exit 0), CC=/usr/bin/gcc GOMEMLIMIT=4500MiB mise exec -- go test -tags test_dep -p 1 -parallel 1 ./tools/umpire/lower ./tools/umpire/model -run '^(TestWhatTestpilotCannotRunIsNamedWithItsOwner|TestARedactedFieldIsNamedAsALimit|TestEveryQueryOfTheActivityModelLowersOrNamesItsLimit|TestValidateRejectsAStepWithTheWrongArity)$' -count=1 (exit 0), CC=/usr/bin/gcc GOMEMLIMIT=4500MiB mise exec -- make umpire-check-cases umpire-check-fixtures canary-check-case (exit 0), git diff --check (exit 0), make umpire-rerecord-pinned-runs (exit 0; live rerecord repair and affected packages detailed in .flow/tmp/fn117-record-repair/evidence.md), CC=/usr/bin/gcc GOMEMLIMIT=4500MiB mise exec -- go test -tags test_dep -p 1 -parallel 1 -count=1 ./common/testing/testpilot/replay ./common/testing/testpilot/evaluation ./common/testing/testpilot/temporal ./tools/canary/... ./tools/umpire/cmd/umpire-assess (exit 0), CC=/usr/bin/gcc GOMEMLIMIT=4500MiB mise exec -- go vet -tags test_dep -p 1 ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... ./tests/testcore/testpilot/... (exit 0), mise exec -- scala-cli test model/project.scala model/umpire model/temporal --test-only temporal.standaloneactivity.StandaloneActivityPins (exit 0; unchanged test co-located at owner's request), make GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main lint-code-fast: golangci-lint zero issues; initial errortype exec-format failure repaired by rebuilding pinned v0.0.7; exact errortype go-vet phase rerun with Makefile-derived target list and tags disable_grpc_modules,test_dep exited 0; source unchanged
- PRs: