---
satisfies: [R18, R19]
---
# fn-190-models-are-plain-scala-state-as-vars.13 Validate every native effect result and lint inferred multiline definitions

## Description
Owner follow-up 2026-10-17. Add a compiler check for every method in native Machine, Facet and native Derived effects sections, including private, parameterized and unused methods: the inferred or declared result must be exactly Unit. Add separate author lint forbidding explicit effect return annotations, allowing typed parameters. Require multiline val/def bodies only for object members within native Models and their nested sections. Include literal constant vals. Exempt vars, locals inside methods/functions and declarations outside Models. Keep objects merely named effects and legacy RecordMachine compatibility fixtures unrestricted by the effect checks. Apply inferred multiline style to all current sample effects. Use existing compiler and typed lint seams with positioned errors; focused positive/refusal fixtures prove both checks, including unused defs, aliases, Nothing, typed params, facets and derivations. No schema, Go, Temporal or broader exporter release.

## Acceptance
Every authored native effect def is checked even unused; inferred Unit compiles, non-Unit and Nothing fail at the definition. Style lint rejects explicit effect return annotations and same-line val/def member bodies within Models, including literal constant vals. Multiline bodies, vars, locals inside methods/functions and declarations outside Models pass; typed effect parameters remain allowed. Native sample effects follow the rules and all relevant checks pass.

## Done summary
Every authored native effect definition is checked for an exact Unit result, including unused/private/parameterized methods. Compiler-generated default getters are excluded using their compiler name kind; authored methods with similar names remain checked.

Author lint forbids explicit effect return annotations and requires all val/def object-member bodies inside native Models and nested sections to begin on the next line, including literal constants and tuple patterns. Vars, method/function locals, ordinary class fields and declarations outside Models remain exempt. The typed rules share owner/source classification and reuse the existing source lexer. Samples and native effect fixtures follow the style; SEMANTICS records it.

Verification: 538 native tests, 4 focused compiler/style tests, 29 source-lint tests, 8 suite-inventory tests and 3 boundary tests pass. Scoped formatting passes. Fresh native compilation and a typed scan of 258 live sample Model members report zero new-rule findings; an inline Model constant negative control reports exactly one finding. Fresh read-only review found no remaining material issue.

The whole signature compile remains held at NativeCheck.scala:238 (Any has no finite catalog); full conventions retain 19 existing liftable-subset IR refusals and zero new R4 findings. These broad checks are not counted as passes. No schema, Go or Temporal work was released.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests, mise exec -- scala-cli run --test --server=false model/framework/export --main-class org.junit.runner.JUnitCore -- framework.lint.EffectDefinitionsSuite, mise exec -- scala-cli test --server=false model/framework/check --require-tests --test-only 'framework.check.lint.*RuleSuite', mise exec -- scala-cli test --server=false model/framework/check --require-tests --test-only framework.check.FrameworkLoopSuite, mise exec -- scala-cli test --server=false model/framework/check --require-tests --test-only framework.check.lint.BoundarySuite, mise exec -- scala-cli run --server=false .flow/tmp/native-definition-proof/StyleProof.scala, scoped scalafmt checks: .flow/tmp/definition-style-fmt-check.log, .flow/tmp/effect-fixture-fmt.log, .flow/tmp/sample-split-fmt.log
- PRs: