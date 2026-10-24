---
satisfies: [R8]
---
# fn-164-expect-blocks-for-properties.1 Use colon blocks consistently across the Scala DSL and enforce the style

## Description
**Note 2026-10-18 (one-language series):** IR parity proofs use the in-process internal IR (fn-201); no checked-in IR.

Touches: [model/.scalafmt.conf, model/.scalafix.conf, model/framework/, model/temporal/, model/irgen/, model/check/, model/README.md, model/framework/SEMANTICS.md]

Use colon-and-indentation blocks consistently across the Scala DSL after fn-164's `expect` implementation and caller migration, before the pre-fault meaning-preserving seal. Wire the dependency to the final expectation-migration task when the remaining fn-164 tasks are planned; this task alone does not implement or close R1-R7.

Convert optional DSL argument braces, including `rules`, `on`, `from`, `effect`, `is`, `expect`, `oneOf` and realization section blocks, wherever Scala admits the colon spelling. Include nested and single-requirement blocks and update framework core-form examples, positive/refusal fixtures and README/SEMANTICS guidance. Retain language-required braces and preserve deliberately invalid fixture behavior and located diagnostics.

Inspect the pinned Scalafmt 3.11.5 configuration and use its built-in optional-braces/fewer-braces rewrite first. The current config already removes some optional braces; prove the exact settings on representative nested and single-statement DSL examples, format twice to prove idempotence, and make the existing formatting check enforce the result. Avoid adding a second rewrite implementation when Scalafmt suffices. If it leaves gaps, extend the existing block-form gate or use an established Scalafix rule with focused tests. The current BlockFormRule detects parenthesized `on(...)(...)`, while its documentation/diagnostic recommends braces: update both to the colon convention and keep the parenthesized refusal.

Only Scala/model files are changed; checked-in IR and Cases remain stale, and Go, full gates, regeneration, review and live runs remain deferred under the current owner mode.

Quick commands (serialize heavy commands):

```sh
make fmt-model
make lint-model-syntax
mise exec -- scala-cli compile model/project.scala model/framework model/temporal
mise exec -- scala-cli test model/project.scala model/framework model/temporal
mise exec -- scala-cli test model/irgen
mise exec -- scala-cli test model/check
make umpire-check-equivalence BASE=.flow/tmp/fn160/baseline-ae3f36ee51 MAP=model/testdata/equivalence/batch-1d.map STEPS_ONLY=1
```

Also run the existing Scalafmt `--check` invocation from `lint-model` directly, so this Scala-only task does not invoke deferred consumer work. Compare scratch lifts before/after, allowing source positions only; use source maps for refusal-line changes rather than weakening assertions.
## Acceptance
- [ ] Optional DSL blocks consistently use colon indentation across Models, framework examples, lifter/gate fixtures and author documentation; required Scala braces and intentional refusal cases remain correct (R8).
- [ ] The pinned formatter rewrites representative nested and one-statement blocks idempotently, and the existing formatting/lint workflow rejects reintroduced optional braces; any formatter gap has a focused lint test (R8).
- [ ] Block-form diagnostics recommend `on(...):`, accept colon blocks, retain parenthesized-rule refusals, and ignore comments, strings and unrelated calls (R8).
- [ ] Scala compilation, framework/Model/lifter/gate tests and scratch-lift comparison pass; lifted changes are source positions only and step-table equivalence passes before the pre-fault seal (R8).
- [ ] The final fn-164 migration task is a recorded dependency before this task is admitted; no Go, managed-artifact regeneration, full gate, review or live-run closure is claimed (R8).


## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: