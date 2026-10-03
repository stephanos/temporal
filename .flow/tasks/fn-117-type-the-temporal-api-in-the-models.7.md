---
satisfies: [R2, R3, R4, R5, R6, R7, R8, R9, R10]
---
# fn-117-type-the-temporal-api-in-the-models.7 Migrate Nexus and lifter examples to the typed author surface

## Description
Finish the existing Model and positive-fixture corpus with the typed forms, preserving the Nexus behavior and multiple schemas.

**Size:** M
**Files:** model/temporal/nexuscaller/Model.scala and Realization.scala; model/lifter/testdata/lifts/Realizations.scala and any fixture with action schema names; regenerated IR/expected positions.
**Touches:** [model/temporal/nexuscaller/Model.scala, model/temporal/nexuscaller/Realization.scala, model/lifter/testdata/lifts/**, model/lifter/Realizations.scala, model/lifter/test/Fixtures.test.scala, model/ir/**, model/cases/manifest.json, .flow/tmp/fn117-7/**]

### Approach
- Snapshot current IR, expected lift fixtures and Cases. Migrate Nexus message/method/field/enum declarations, ordered StartOperationResponse and HandlerError schemas, Payload.metadata map entries, typed response-variant helpers, request/response/evidence paths and explicit run-event/history roots. Model-owned IDs and existing script/wait behavior stay unchanged.
- Search every current Temporal Model and positive lifter fixture for remaining proto author strings; migrate actual remaining sites using tasks3-5. Do not re-organize Model files or implement fn-114's declaration deduplication. Deliberately invalid refusal fixtures remain meaningful, including task3's mismatched realization descriptors.
- Regenerate existing IR and expected positive fixtures only after proving decoded equality modulo positions; preserve default absence, field/schema/declaration order, names and operands. Case JSON bytes stay equal. Query-manifest `Position` strings may move to corresponding Scala declarations; its non-position data remains exact. Keep the fn-113 frozen originals/config unchanged; no recapture or broader projection.
- Complete the positive/negative typing matrix for any newly exercised context, count remaining/removed strings, list genuinely unknown operands, and pass relevant frozen model/lower goldens, lifter/model gates and lint once. Reuse unaffected package coverage per MILESTONES.md.

### Migration-discovered integration fix
- Preserve both existing terminal repeated-read spellings through typed selectors: direct selection retains a bare repeated field, while `.map(item => item)` explicitly retains terminal `[*]`. The current lifter forcibly appends `[*]`, changing Nexus IR. Retain the source terminal descriptor for an identity map, preserve the selected path unchanged, and still refuse a mapped selector ending at a singular message. Add focused bare/explicit read equivalence and refusal coverage; no Boolean/string spelling hint, new DSL language, Go change or broader normalization.

### Existing compatibility controls
- Retain only the existing `old*` declarations in Typed.scala (including oldLong) that serve typed-versus-legacy parity tests through this task. They are not authoring examples. Record their exact locations; task8 removes these controls with the legacy routes while preserving equivalent typed-output assertions and the compiler-refusal matrix. All Nexus and exported positive author declarations use typed forms.

### Investigation targets
**Required:**
- model/temporal/nexuscaller/Model.scala:57
- model/temporal/nexuscaller/Realization.scala:239-323
- model/lifter/testdata/lifts/Realizations.scala
- model/lifter/testdata/lifts/Admission.scala and Channels.scala
- model/lifter/test/Fixtures.test.scala:56-64,201-305
- tools/umpire/model/migration_golden_test.go:211-228
- tools/umpire/lower/migration_golden_test.go:223

### Quick commands
Existing model/fixture regeneration; decoded positions-only equality across all IR/positive expected outputs; focused frozen migration goldens with -tags test_dep -p 1 -parallel 1; check-mode model gate, full lifter and lint-model.

### Execution constraints
Preserve comments, no staging/commits/worktrees/push. No Go/schema/path-language changes, new Model behavior or later-spec helper work. Keep generation and heavy Go processes serial.
Migrate each Poll to the typed reference of its actual Recorded.Read/Single evidence declaration, preserving the original evidence ID and projection; do not manually annotate a root unrelated to that evidence.
## Acceptance
- [ ] All current Temporal Models and positive author examples use typed proto declarations, including ordered multi-schema and the existing map writes.
- [ ] Every IR/positive expected output equals its snapshot except positions; Cases and independent frozen goldens stay strict.
- [ ] Remaining sites, removed string counts and narrow unknown operand exceptions are recorded, and applicable typing/gate/lint checks pass.

## Done summary
Migrated Nexus caller and the positive `Realizations.scala` corpus to generated Temporal/Testpilot typed schemas, unary method descriptors, request/response/evidence field selectors, enum values, symbolic protobuf messages, and UTF-8 ByteString map entries. Each poll now derives its request and projection roots from its own typed evidence reference; IDs, roles, scripts, waits, declaration/schema order, and behavior are preserved. A narrow lifter change retains the author's repeated-read spelling: direct selection emits the bare path, while an explicit identity map emits `[*]`.

Baseline: green via task 6 handoff at the same HEAD `2093a63f2da7c6c66f3509414dd78476eb4d5c04`, with its 4 focused typed tests, generation, model/lower goldens, check-mode model and lint green; no pre-edit gate rerun. Exact pre-edit source bytes and all six IR, manifest, positive fixture, Case, and frozen Go migration file bytes/hashes were captured under `before/` and `preedit/` before any source edit. The initial positive lift gave a red nine-difference proof for repeated-read spelling; `fixtures-realizations-loop.log`, `red-nine-diffs.log`, and the first `artifact-compare-observed.log` record the failing comparisons.

`python3 .flow/tmp/fn117-7/compare_artifacts.py` exits 0 (`artifact-compare-observed-final.log`): all six IR JSONs have identical ordered decoded structure, values, and source filenames; only position lines moved, and every current line resolves within its source. Of 1,439 frozen Case/fixture files, 1,437 are byte-identical; only positive `closereset.json` and `realizations.json` differ in source-position lines. Every Case JSON, query manifest, frozen Go original/config/projection, and refusal fixture is byte-identical. The complete list of 11 changed task paths is `touched-files.json`. All 213 original comment lines across the six changed Scala files remain verbatim in order.

Removed free-text author sites in the three migrated files (pre-edit → final): 29 → 0 quoted protobuf full names, 13 → 0 method suffix literals, 40 → 0 `Assignment("...")` targets, 27 → 0 `ProtoField("...")` names, 10 → 0 legacy `Recorded` selectors, 16 → 0 legacy RPC/Poll constructors, 10 → 0 projected string paths, and 8 → 0 `EnumName` constructors (`migration-counts.json`). No unknown-root operand escape or generated unknown enum is needed in the migrated corpus. Existing standalone-activity's two explicit `Operand.Projected.as[InstructionOutcome]` dynamic-origin escapes remain outside this task.

The sole remaining positive-fixture free-text sites are task 3–5 parity controls in `model/lifter/testdata/lifts/Typed.scala`: `oldOne` (26), `oldTwo` (28), `oldCall` (225), `oldEvidence` (234), `oldSingleEvidence` (242), `oldPoll` (251), `oldHistoryCall` (261), `oldHistoryEvidence` (269), `oldEventEvidence` (283), `oldRunEventEvidence` (311), `oldPayload` (337), `oldFailure` (366), `oldCommand` (430), `oldResponse` (462), `oldRealization` (522), and `oldLongRealization` (646), plus `oldMachine` (51) which only holds old actions. They deliberately compare typed output with legacy spelling until task 8 retires those forms. No other Temporal or exported positive `Realizations.scala` protobuf author string remains.

Verification (all exit 0, foreground and serial; exact commands):
- `CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- scala-cli test --suppress-outdated-dependency-warning model/lifter --test-only 'umpire.lift.Fixtures*' -- '*typed*'` — 23s, `fixtures-typed-loop.log`; five typed checks including direct bare and explicit-map wildcard positives.
- `CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks` — 126s, `gen-model-observed-final.log`; complete and `== ok`.
- `CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- go test -json -tags test_dep -p 1 -parallel 1 ./tools/umpire/model -run '^TestMigrationGoldens$'` — 23s, `model-golden-observed-final.jsonl`; TestMigrationGoldens pass.
- `CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- go test -json -tags test_dep -p 1 -parallel 1 ./tools/umpire/lower -run '^TestMigrationGoldens$'` — 80s, `lower-golden-observed-final.jsonl`; TestMigrationGoldens pass.
- `CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks` — 96s, `check-model-observed-final.log`; full lifter/check-mode gate and `== ok`.
- `CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- make lint-model` — 13s, `lint-model-observed-final.log`; exit 0, zero current compiler errors, unused-parameter or DisableSyntax diagnostics. The inherited JDK 27 Scalafix `NoSuchFieldException: path` trace remains in the wrapper log.
- `CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- scala-cli test --suppress-outdated-dependency-warning model/lifter --test-only 'umpire.lift.Fixtures*' -- '*mapped read*'` — 19s, `mapped-refusal-observed-final.log`; the mapped-singular read still refuses.
- `python3 .flow/tmp/fn117-7/compare_artifacts.py` and `git diff --check` — exit 0.

No files were staged or committed; HEAD/base remain `2093a63f2da7c6c66f3509414dd78476eb4d5c04`. The task remains `in_progress` for the conductor's independent host review and Flow completion.

Independent task-local implementation review reached SHIP without findings; session 01a10243-0af2-7cd3-b84b-2181efcb8eb2. Conductor verified all 80 pinned inputs, final lint compiler diagnostics, artifact comparison and diff check.

stage: impl-review - ran (model: gpt-6-sol at high)
stage: plan-sync - skipped(config: planSync.enabled != true)
stage: wave-dispatch - ran (model: gpt-6-sol at high)
## Evidence
- Commits:
- Tests: CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- scala-cli test --suppress-outdated-dependency-warning model/lifter --test-only 'umpire.lift.Fixtures*' -- '*typed*', CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks, CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- go test -json -tags test_dep -p 1 -parallel 1 ./tools/umpire/model -run '^TestMigrationGoldens$', CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- go test -json -tags test_dep -p 1 -parallel 1 ./tools/umpire/lower -run '^TestMigrationGoldens$', CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks, CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- make lint-model, CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- scala-cli test --suppress-outdated-dependency-warning model/lifter --test-only 'umpire.lift.Fixtures*' -- '*mapped read*', python3 .flow/tmp/fn117-7/compare_artifacts.py, git diff --check, Independent implementation review SHIP, session 01a10243-0af2-7cd3-b84b-2181efcb8eb2; all 80 pinned hashes verified, Unchanged full Go source coverage reused from fn-113.16; focused model/lower migration goldens rerun against current inputs
- PRs: