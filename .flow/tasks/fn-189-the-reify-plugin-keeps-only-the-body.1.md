---
satisfies: [R1, R5]
---
# fn-189-the-reify-plugin-keeps-only-the-body.1 Single-compilation contract: remove jar reads, trust checks, carrier versions and non-root positions

Touches: model/framework/reify/plugin, model/framework/dsl/Reify.scala, model/framework/export reify fixtures, model/README.md

## Description
**DSL completion phase (owner, 2026-10-17):** scheduled before the IR/plugin phase because it changes what authors write or can reach; the gate is the native checks plus the plugin compiling the six samples.

Record the contract in model/README.md: every Model and every definition it lifts from compile in one scala-cli compilation of sources under model/; the only jars on the classpath are the generated proto jars and the plugin. Then delete everything in the plugin that exists for another jar: JarSugar and the DottyUnpickler reads, jarSugar/sugarRoots/inJar in DefinitionStore, frameworkSourceLabels and the zip walk, FacetPositions.qualify and every qualify/qualifiedGroups/qualifiedSites call, retainSource and the source-label maps in PluginContext, trusted-artifact lists and artifact() checks in FamilyPhase and NamesPhase, the source-path anchors in canonicalImplementation, the version strings and fingerprints on every carrier (section-v1, facet, status, outcome, rule-call-groups-v2, family v1, rule-type-position-v1) and their malformed-or-stale replay. Positions: a stored ir.Model keeps a position on its root function only; remove ruleTypePosition, RuleTypeLocations, saveTypeLocation and locatedModel. Retire the test fixtures that package helpers or traits into jars (ReifyBoundary helper-jar cases, trait-before-client jars, spoofed-package regressions) and keep their same-compilation twins. Keep the one liftedIr annotation written before pickler and its warm-Bloop invalidation test.

**Size:** L

### Verification
- Plugin packages via the Makefile target and the six samples compile with -Xplugin under the native strict flags.
- The framework-only check entrypoint (fn-188.4) passes; reify and export suites pass after the retired fixtures are removed.
- Compile-time refusal fixtures still report at the same positions.
- grep shows no ZipFile, DottyUnpickler, binaryFile, underlyingSource, SourceFile annotation reads or fingerprint digests in model/framework/reify.

## Acceptance
- [ ] No code in model/framework/reify reads TASTy or trees from a jar or checks artifact identity, source paths or carrier fingerprints.
- [ ] Stored IR carries a position on its root function only; compile-time refusals keep their positions (existing refusal fixtures pass).
- [ ] A capture whose reached definition has neither a tree in the run nor a liftedIr annotation is refused with the R2 text.
- [ ] The contract is recorded in model/README.md and the cross-jar fixtures are retired, not disabled.

## Done summary
Single-compilation contract recorded in model/README.md (`ec8e76e236`). The plugin's jar reads, trust checks, carrier versions/fingerprints and non-root positions are deleted (`b2172d764a`). The cross-jar ReifyBoundary fixtures are retired (`50e83417b5`). Merged as `7dbd90b6d9`. reify/plugin shrinks from 5,072 to 4,194 lines. The task grep (ZipFile, DottyUnpickler, TastyLoader, binaryFile, underlyingSource, SourceFile annotation, digests) finds nothing in reify.

Gate on the merged tip:
- 637 native tests (`.flow/tmp/p2-1891-native.log`).
- The plugin packages (`p2-1891-pkg.log`).
- `-Xplugin` realization-mode compile of dsl/realize/samples passes (`p2-1891-xplugin.log`). Before, it failed with 11 trust-check refusals on DSL macro constructors.

Full-mode (export fixture) compilation still fails on the plain var Machine form, as before. ReifyBoundarySuite keeps its 4 pre-existing retired-vocabulary failures, and ReifySamplesSuite fails as before. Both are Phase 3 (fn-190.9) and are not counted as passing. The "mark its definition @lifted" refusal text changes with fn-189.5.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: ec8e76e236, b2172d764a, 50e83417b5, 7dbd90b6d9
- Tests: mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests, mise exec -- scala-cli compile --server=false --scalac-option -Xplugin:model/build/reify.jar --scalac-option -P:umpire-reify:mode=realization model/framework/dsl model/framework/realize model/framework/samples
- PRs: