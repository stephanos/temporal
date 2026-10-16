---
satisfies: [R1, R2, R4]
---
# fn-147-migrate-elapsed-time-fields-to-protobuf.5 Close native Duration authoring coverage and guidance

## Description
Close the native-only validation/documentation residue from fn-147.2; contributes only the Scala authoring portion of R1/R2/R4. Existing source is integrated. No new Duration API, field admission policy or migration machinery is requested.

**Size:** M
**Files:** model/framework/realize/Elapsed.scala, model/framework/realize/Realize.scala, model/framework/realize/Scripts.scala, model/framework/realize/Construction.test.scala, model/framework/README.md
**Touches:** [model/framework/realize/Elapsed.scala, model/framework/realize/Realize.scala, model/framework/realize/Scripts.scala, model/framework/realize/*test.scala, model/framework/README.md]

**Admission:** committed fn-188.3 closure (satisfied). Priority 0, before fn-188.2, fn-174.24 and every later IR/export/schema/Go/backend task. No dependency on mixed fn-147.1-.4 or their whole-spec integration baselines. Those original tasks remain held with their complete obligations.

### Approach

1. Inventory the current generic framework Duration surface and existing native coverage before adding tests. Reuse Elapsed.scala:5, Realize.scala:452/549/623, Scripts.scala:53/114 and Construction.test.scala:120/443/473. The positive interval, timeout and 1234ms conversion controls already passed in fn-188.3; retain those results rather than repeating unchanged broad checks.
2. Pin exact Duration construction and value/presence retention, including None, Some(zero), positive values, copies and public helper forwarding. Cover signed millisecond decomposition and Long boundaries where supported by the existing helper. Do not round, normalize, reinterpret or silently discard supplied values.
3. Separate native value construction/retention from owning-field validation: Elapsed.scala explicitly delegates bounds/presence validation to lifting/admission. Record sign, precision, range and field-specific refusal obligations as deferred integration proof, not new native rejection behavior. Do not introduce a validation feature to make a test pass.
4. Clarify the existing author guidance and link native tests to the actual supported behavior. Preserve relevant existing fixtures; add only missing focused controls. Any demonstrated implementation mismatch must be scoped and evidenced before repair.

### Verification

Use the existing native Scala CLI test entrypoint with the reached framework.realize suite(s), --server=false and --require-tests; serialize compiler writes. Retain the completed native batch evidence for unchanged code. If source changes invalidate it, record the reached checks and reserve broad native validation/review for batch end. Never run make check-framework as a native-only verifier.

### Boundaries

No reads/edits of model/temporal, including disabled bodies. No model/project.scala, Temporal/API/Model jars, exporter/lifter execution, IR/schema changes, Go, runtime polling proof, equivalence, regeneration, new Scalafix/proof harness or parent-spec completion claims.

## Acceptance
- [ ] Inventory identifies current native Duration entrypoints and existing valid coverage; no already-integrated implementation is replayed.
- [ ] Focused native tests pin exact helper construction, optional/default/explicit-zero retention, copies and public forwarding, with actual executed counts and exits; no new field-admission policy is inferred.
- [ ] Author documentation distinguishes native representation from deferred owning-field sign/range/precision/presence checks and makes no runtime/IR validation claim.
- [ ] Existing fixtures/assertions are preserved; any missing focused controls use existing suites, with no new verifier or migration framework.
- [ ] Receipt leaves fn-147.1-.4, full R1/R2/R4, lifting/equivalence/schema/Go/runtime/artifact proofs open and consumes no Temporal input.


## Done summary
Authors now have focused Duration construction controls and guidance for the existing native API. Three table-driven tests cover signed millisecond decomposition through both Long endpoints and exact optional Duration retention through Command/Poll construction, helpers, copies and naming; the existing positive interval, timeout and 1234ms controls remain intact. No runtime source changed.

Inventory: Elapsed.scala's milliseconds(Long), Realize.scala's Command.timeout and TypedPoll.interval/default/copy paths, Instruction.readUntil and Scripts.scala's command/readUntil helpers. README guidance distinguishes native representation from owning-field sign/range/precision/presence admission. Direct negative, sub-millisecond and oversized values are retained as authored, without introducing a validation policy.

baseline: green. The exact focused Quick command executed 19 tests with exit 0 before edits (.flow/tmp/fn1475-baseline.log). Post-change execution passed 22 tests with exit 0 (.flow/tmp/fn1475-verify.log). These characterize existing behavior; no implementation mismatch or failing reproduction was found, and no artificial red result is claimed.

Native lint passed 29 tests with exit 0, including a 126-file block-form scan with zero findings (.flow/tmp/fn1475-lint.log). Touched-file scalafmt and repeated checks passed; git diff --check passed. gate classify required the task's focused gate; its post-commit receipt is .flow/tmp/green-receipts/2386401b-unittest.json. An initial receipt attempt correctly refused the pre-commit dirty worktree and was retried after committing.

The committed fn-188.3 final 463-test native batch and public-client proof remain prior evidence for unchanged runtime source; this task did not rerun or claim an expanded full suite. fn-147.1-.4, full R1/R2/R4, lifting/admission error cases, equivalence, schema, Go, runtime polling/monotonicity and artifact proofs remain open. No Temporal body or model/project.scala input was consumed. No new verifier, manifest or migration framework was added.

Tier: session (jev-unavailable(no_key))
GATE_SKIPPED:unittest:green-receipt 2386401b - final verification reused this task's post-gate pass; only Flow review metadata changed afterward.
stage: impl-review - ran [2026-10-16T14:31:51Z..2026-10-16T14:33:12Z]
Official Codex fan-out/finalize returned SHIP on all three axes with zero findings. Fresh same-family reviewer model gpt-6.1-sol at high is evidenced by .flow/tmp/fn1475-review-finalize.json; receipt /tmp/impl-review-receipt-8f37faba39e2-fn-147-migrate-elapsed-time-fields-to-protobuf.5.json. Reviewers' independent Scala attempts were inconclusive because the read-only sandbox refused build writes before executing tests. Worker execution supplies the green test evidence above.
stage: plan-sync - skipped(config: planSync.enabled=false)
Tracker sync: n/a (bridge inactive)
## Evidence
- Commits: 2386401b0bce2cc097cee14fe03d8233ca6d603a
- Tests: mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --test-only framework.realize.RealizationConstructionTest --require-tests, mise exec -- scala-cli test --server=false model/framework/check --require-tests --test-only framework.check.lint.*RuleSuite, mise exec -- scala-cli fmt --check --scalafmt-conf model/.scalafmt.conf model/framework/realize/Construction.test.scala, git diff --check, GATE_SKIPPED:unittest:green-receipt 2386401b - final verification reused this task's post-gate pass; only Flow review metadata changed afterward.
- PRs: