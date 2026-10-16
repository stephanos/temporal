---
satisfies: [R8]
---
# fn-164-expect-blocks-for-properties.11 Enforce native DSL colon blocks with framework-scoped lint

## Description
Implement the native colon-style scanner/enforcement portion split from .10. This task is independent of .2's expectation implementation. Keep caller conversion and formatter idempotence in .10, and exact positioned IR proof in held .9.

**Size:** M
**Files:** model/framework/check/lint/BlockFormRule.scala and BlockFormRule.test.scala.
**Touches:** [model/framework/check/lint/BlockFormRule.scala, model/framework/check/lint/BlockFormRule.test.scala]

### Approach
- Reuse SyntaxRule.code and the existing balanced-call scanner. Add explicit framework-native source discovery and colon-style diagnostics for optional DSL blocks, retaining rejection of parenthesized on cases. Do not restore or inspect Temporal sources or keep a Temporal-only empty scan as a pass.
- Cover the admitted rules/on/from/effect/is/expect/oneOf/branch and realization-block vocabulary where Scala permits colon syntax. Distinguish declaration/member/unrelated calls and language-required braces; ignore comments/string contents and generated/build cache trees. Preserve line diagnostics.
- Red/green fixture tests prove source discovery visits nonzero native files, optional brace DSL blocks reject and colon counterparts pass; retain parenthesized-rule negatives and unrelated-call/string/comment positives. Include balanced/nested/multiline arguments and inline source fixtures as appropriate.
- Expose findings through the existing BlockFormRule API rather than create a competing linter/rewriter. .10 migrates the real native corpus and runs the final clean scan; report any current native findings here without claiming that migration is done.

### Investigation targets
**Required:** model/framework/check/lint/BlockFormRule.scala; model/framework/check/lint/BlockFormRule.test.scala; model/framework/check/lint/SyntaxRule.scala (read-only literal scanner); model/framework/check/ModelSources.scala; model/.scalafmt.conf.

### Verification
Run focused framework.check.lint.BlockFormRuleSuite using the framework check project's existing Scala settings with nonzero tests. Serialize compiler access with the conductor/native_expect worker. No full make check-framework, source formatter sweep, exporter/IR, schemas, Go or Temporal inputs.

### Integration
Work in the conductor-provided isolated worktree. Do not modify shared Flow state or MILESTONES; report task-unique summary/evidence and a source commit for conductor integration. The conductor closes .11 only after integration and verification. .10 consumes the guard and completes caller/style migration.

## Acceptance
- [ ] Fixture corpus proves nonzero framework-native discovery, ignored build/generated trees and no Temporal source dependency.
- [ ] Optional brace DSL blocks and parenthesized rule cases reject with positions; colon forms, language-required braces, unrelated calls, comments and strings pass.
- [ ] Focused scanner tests pass, including nested/multiline cases, with positive executed counts.
- [ ] Real native corpus findings are recorded for .10; no complete native migration, formatter idempotence or IR proof is claimed.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
