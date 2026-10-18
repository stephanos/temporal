---
satisfies: [R5, R7, R8]
---
# fn-164-expect-blocks-for-properties.9 Use and enforce colon blocks throughout the framework DSL

## Description
**Superseded by fn-190 (owner phasing 2026-10-17).** fn-190.4/.7 rewrite the framework surface and samples; colon-block enforcement applies to the final form there.

Framework-only colon-style checkpoint. Touches framework source/examples/fixtures/docs and reached check/lint BlockFormRule/SyntaxRule and formatting runner seams. Reuse pinned Scalafmt optional-braces rewriting and the existing scanners; no competing text rewriter. Convert admitted rules/on/from/effect/is/expect/oneOf/branch and realization blocks, keeping language-required braces and explicitly invalid refusal fixtures. Reject reintroduced optional brace DSL blocks, retain parenthesized-rule refusals, and ignore comments/strings/unrelated calls. Run formatter twice for idempotence, complete pre/post generic scratch-lift comparison with explicit source mappings, and make check-framework. No structural-hash or runtime-only substitute for exact IR preservation. No dependency on disabled Temporal migration; original full-scope .1/R8 remains held for later reconciliation.
## Acceptance
Framework authoring blocks and documentation use colon syntax wherever Scala admits it; existing formatter/scanner tests enforce that shape without false positives; second formatting pass changes nothing; full generic IR differs only at mapped source positions; framework-only check passes; Temporal callers and full-scope receipts remain unclaimed.

## Done summary
Already superseded by fn-190; status hygiene 2026-10-18

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: