---
satisfies: [R1, R5]
---
# fn-189-the-reify-plugin-keeps-only-the-body.1 Single-compilation contract: remove jar reads, trust checks, carrier versions and non-root positions

Touches: model/framework/reify/plugin, model/framework/dsl/Reify.scala, model/framework/export reify fixtures, model/README.md

## Description
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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
