---
satisfies: [R5, R7, R8]
---
# fn-164-expect-blocks-for-properties.9 Use and enforce colon blocks throughout the framework DSL

## Description
Framework-only colon-style checkpoint. Touches framework source/examples/fixtures/docs and reached check/lint BlockFormRule/SyntaxRule and formatting runner seams. Reuse pinned Scalafmt optional-braces rewriting and the existing scanners; no competing text rewriter. Convert admitted rules/on/from/effect/is/expect/oneOf/branch and realization blocks, keeping language-required braces and explicitly invalid refusal fixtures. Reject reintroduced optional brace DSL blocks, retain parenthesized-rule refusals, and ignore comments/strings/unrelated calls. Run formatter twice for idempotence, complete pre/post generic scratch-lift comparison with explicit source mappings, and make check-framework. No structural-hash or runtime-only substitute for exact IR preservation. No dependency on disabled Temporal migration; original full-scope .1/R8 remains held for later reconciliation.

## Acceptance
Framework authoring blocks and documentation use colon syntax wherever Scala admits it; existing formatter/scanner tests enforce that shape without false positives; second formatting pass changes nothing; full generic IR differs only at mapped source positions; framework-only check passes; Temporal callers and full-scope receipts remain unclaimed.

## Done summary
Blocked:
Owner override. Native Umpire framework DSL authoring, typing, runtime, caller/API retirement and native guidance come first. This original mixed/integration task remains open and held while native fn164.2/.10, fn165.7-.9, fn173.10/.11, fn123.9/.10 and fn153.7 complete. Consume native results later without repeating or claiming their deferred IR/Go/backend proofs. Retain all original exact IR, metadata, schema, compatibility, consumer and integration requirements. Full spec closure is not a native prerequisite. Ignore model/temporal entirely; no task resumes Temporal work without new owner authority. After native DSL completion, re-anchor this task to the final API before release. This is a deliberate priority/scope hold, not evidence of completion or a technical failure.
## Evidence
- Commits:
- Tests:
- PRs:
