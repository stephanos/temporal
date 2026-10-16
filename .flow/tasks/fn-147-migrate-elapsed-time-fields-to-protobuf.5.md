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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
