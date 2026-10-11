---
satisfies: [R7]
---
# fn-140-one-sentence-witness-queries-with.5 Document witness authoring and update the layout template

Touches: [model/README.md, model/SEMANTICS.md, model/irgen/testdata/layout/lamp/Lamp.scala, model/irgen/testdata/layout/lamp/product/Product.scala, model/irgen/testdata/layout/lamp/system/System.scala, model/irgen/test/Fixtures.test.scala]

## Description
Update author documentation and the existing layout template for R7. This task follows .4 in the owner's single-worker Scala lane. Only model/ authoring documents and template files are edited; module-map documentation, managed publication and full integration gates remain in deferred .6.

**Size:** M
**Files:** `model/README.md`, `model/SEMANTICS.md`, the lamp template's level files, existing layout fixture assertions if needed.

### Approach
- Rewrite README's path/Property/Scenario/Query example at `:62` around one witness, then give its exact ordinary core form and explain when shared triples or query verify remain appropriate. Update live Case generation at `:1590` and naming/read-order guidance without adding new semantics.
- Document the source-only reason rule for convenience, full Run and monitor expectations; distinguish the explanatory text from existing judge reason IDs and expected_run metadata.
- Add one ordinary witness to the existing lamp template while keeping its verify/refinement example. Exercise the template through the established layout fixture; keep its package-only root in normal compilation, per the current build-errors memory.
- Describe witness ownership and no-new-IR lifting in Model author documentation. Remove obsolete Query expect examples there. The matching .plans/UMPIRE_MODULES.md update stays in deferred .6, because the current owner mode allows source edits under model/ only.
- Preserve .4's committed proof baseline; .10 includes this documentation/template in its final source seal. Run the template's focused Scala compile/lift assertions positively with a nonzero population, without a production regeneration or full gate. Deferred .6 consumes this source and closes the actual full checks once at fn-123.8.
- Preserve the user's existing uncommitted model/SEMANTICS.md draft in the main checkout. Read it for overlap and reconcile required localized documentation edits with the conductor before changing or merging that file; do not copy or commit unrelated draft changes.

### Narrow tooling checkpoint
The isolated layout compile fails under the normal `-Werror` on the six symbols in Lamp.scala's two obsolete `product`/`system` imports. Remove only those unused import lines in a separate tooling checkpoint, retaining the package-only collection root and all declarations, compiler flags, assertions and diagnostics. Record the inherited red and a fresh positive layout compile before interpreting the new witness regression.

### Investigation targets
**Required:**
- `model/README.md:62-84` - initial author example.
- `model/README.md:1127` and `:1590` - layout and live generation.
- `model/SEMANTICS.md:984` - generated Case expectations.
- `model/irgen/testdata/layout/lamp/product/Product.scala:37` - template claims.
- `model/irgen/testdata/layout/lamp/system/System.scala` - template refinement.
**Optional:**
- `.plans/UMPIRE_MODULES.md:28` - DSL/lifter ownership.

### Quick commands
```bash
rg -n '\.expect\(' model/README.md model/SEMANTICS.md
git diff --check
```

## Acceptance
- [ ] Author docs explain witness versus verify/shared triple, show the core triple, and identify live as the Case-generation switch (R7).
- [ ] The template contains a witness alongside its invariant/refinement example, with a focused layout fixture assertion.
- [ ] Source-only explanation text and existing emitted judge reason IDs remain distinct in the documentation.
- [ ] Documentation/task .4 ownership stays disjoint and the single-worker lane is retained; this task edits only model/ and publishes no model/ir or model/cases outputs. Module-map documentation remains required in deferred .6.

## Done summary
# fn-140-one-sentence-witness-queries-with.5

Implemented R7 witness-first author teaching, the exact ordinary Property/Scenario/Query expansion, localized semantics clarification, and one distinct model-only switch-off witness alongside the existing lamp verify/refinement. A dedicated focused layout suite compiles and lifts the actual template without launching the broad fixture campaign.

Implementation commit: bd166d858697b71fdb3a32273d85efafe03abc5e. Separate authorized import-only tooling checkpoint: 3ec7d8a47d4abf8ffce7153fb5afbd2d75dd2543. Receipt base: 02d65d0c5790b4b0ad9cf96f659bfc6321b0f1db; original source/context base: 1e3e8e3cd34a08f3d84c21b02deef5b7ffc03b23. Main-only .10 planning commits were not integrated.

Tier: session (jev-unavailable(no_key)); preserve project implementer, no fast-tier switch.
stage: impl-review - skipped(config: REVIEW_MODE=none)
stage: plan-sync - skipped(config: planSync=false)
stage: tracker-sync - skipped(config: external tracker bridge inactive)

### Verification

Baseline: red inherited normal strict layout compile, exactly six unused imports in Lamp.scala (fn140-5-layout-red.log). The separately scoped import-only checkpoint resolved this without weakening the compiler. Independently validated API/IR archive integrity, rebuilt current Scala library and printed its classpath with normal --server=false, then compiled and lifted the original template positively: 5 machines, 1 Query, 1 Property, 1 Scenario. Logs: fn140-5-baseline-package.log, fn140-5-baseline-classpath.log, fn140-5-layout-baseline-package.log, fn140-5-layout-baseline-lift.log.

Intended regression red: the dedicated test compiled/lifted successfully but observed one Query instead of two before adding the witness (fn140-5-witness-red.log). A subsequent assertion-only correction used the existing camelCase switchOn/switchOff IDs rather than incorrect kebab spelling; that failed attempt is retained as fn140-5-layout-green1.log, not counted green.

Final focused test: `timeout 600 mise exec -- scala-cli test --server=false model/irgen --test-only '*LayoutWitnessFixtures*'`, exit 0, one executed test (fn140-5-layout-final.log). Generated layout counts: 5 machines, 2 Queries, 2 Properties, 2 Scenarios. The test pins generated switchedOff Query/Property/Scenario names and lampProduct ownership, two ordered path classes, start state, derived limits 2 steps / 2 actions / 512 states and total 4, no expected_run, same-step tail selector, exact switchedOff fact record and dark end predicate identities. Existing lights verify, switchingOnLights Property, switchedOn Scenario and LampSystem product refinement remain present. Full old machine/verify/refinement projection compares byte-identically after only an explicitly bounded OnlyOn declaration-position adjustment (Product.scala line 48 to original 45); raw diff retains that sole three-line shift. Proofs: fn140-5-layout-core-proof.json, fn140-5-layout-baseline-preserved.json, fn140-5-layout-final-preserved.json, fn140-5-layout-final-preserved-position-adjusted.json.

Pinned Scala format writes and two idempotent --check runs passed on Fixtures.test.scala, Lamp.scala and Product.scala (fn140-5-format-check2.log and fn140-5-format-check3.log). Earlier check1 numeric exit was not captured and is inconclusive, not credited. Postcommit `git diff --check` exit 0; task `rg -n '\.expect\(' model/README.md model/SEMANTICS.md` exit 1 with no matches (expected success). Flow coverage validation exit 0, valid true. Gate classification returned FULL/unmatched:model/README.md; held full gates were not run and no full-gate receipt or green skip is claimed.

Library hash remains the exact prior .4 source jar: 3f1f1960d257db6b2a52f9c4d55e785dea8384cdff0392d6b52bfc570a0cb64f. Classpath hash: 4f00d4b5d02bd93070c7bc8f927d750572e79416424a7b073363263069fcde98. Validated dependency hashes: API ca66e13980809d30068c5d8e64302ee1bed71cca8e10379e349b9b744c9fe6f2; IR 45654467e918f9b4d1cb41812adfd3d208ab264a45de5f484f9020dd188f4301. No .4 native source/pins changed; existing source ownership is preserved, not a claim of new runtime/native execution. Final scratch IR hash: 1a4b8e173439f0e86ff9b5afd5c5c39fe760eecd44e989328cd5305b07313a49; core proof hash: b03e484a5b12960429bba4424c8e50a941b1f378fc3d7844a99e6d1a16fe5fc7.

### Draft reconciliation and held work

Main user SEMANTICS.md remains untouched with SHA256 c000b24118163766cf2f6bba7eeecd44796bf6266e127d826f1496a1092f7d1e. Root approved the exact localized worktree additions and matching user-draft patch at `.flow/tmp/scala-dsl/fn140-5-user-draft.patch` (SHA256 2b10a21f39ffac2dce5a3544d42a05ff1e1a053fa873d90bff885d75dea4e7b7). Read-only `git apply --check --unidiff-zero` against main passed; root owns applying it while preserving the uncommitted draft. No full draft was copied or committed.

Docs distinguish authored .live from capability-generated live expectations, typed member selector composition, source-only non-satisfied because text versus existing emitted judge reasons/expected_run, and ordinary same-step Property semantics without a new IR declaration kind. Exact composed-selector downstream admission remains held .7. MODULES migration, Go, linked API/schema generation, production managed IR/Cases/goldens/mirrors, full gates, independent review and live work remain deferred; inherited Reset/Subject cancellation debt and historical source differences are not waived. All validation outputs are ignored Scala/scratch artifacts. No attributable command remains live.
## Evidence
- Commits: 3ec7d8a47d4abf8ffce7153fb5afbd2d75dd2543, bd166d858697b71fdb3a32273d85efafe03abc5e
- Tests: timeout 600 mise exec -- scala-cli --power package --server=false model/project.scala model/framework model/temporal --library -f -o model/build/model-scala.jar (exit 0), timeout 600 mise exec -- scala-cli compile --server=false model/project.scala model/framework model/temporal --print-classpath (exit 0), unzip -t model/build/model-api.jar; unzip -t model/build/model-ir.jar (both exit 0; independently hash-validated), timeout 600 mise exec -- scala-cli --power package --server=false --library model/irgen/testdata/layout --exclude 'kinds/**' --exclude 'r4/**' -f -o .flow/tmp/scala-dsl/fn140-5-layout-baseline.jar (exit 0), timeout 600 mise exec -- scala-cli run --server=false model/irgen -M umpire.irgen.lift -- --ir .flow/tmp/scala-dsl/fn140-5-layout-baseline.jar=model/irgen/testdata/layout/,model/build/model-scala.jar=model/ model/build/model-scala.classpath .flow/tmp/scala-dsl/fn140-5-layout-baseline-ir lamp (exit 0; 5 machines, 1 Query/Property/Scenario), timeout 600 mise exec -- scala-cli test --server=false model/irgen --test-only '*LayoutWitnessFixtures*' (final exit 0, 1 executed test; 5 machines, 2 Queries/Properties/Scenarios; earlier intended missing-witness red exit 1 retained), mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check model/irgen/test/Fixtures.test.scala model/irgen/testdata/layout/lamp/Lamp.scala model/irgen/testdata/layout/lamp/product/Product.scala (two idempotent checks exit 0), rg -n '\.expect\(' model/README.md model/SEMANTICS.md (exit 1 expected; zero obsolete examples), git diff --check (exit 0), flowctl validate --spec fn-140-one-sentence-witness-queries-with --coverage --json (exit 0, valid true), cmp preserved layout projections after exact OnlyOn source-position 48-to-45 adjustment (exit 0), git apply --check --unidiff-zero fn140-5-user-draft.patch against main (exit 0; read-only)
- PRs: