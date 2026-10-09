---
satisfies: [R3, R8]
---
# fn-156-let-the-scala-compiler-and-lint-enforce.6 Trial explicit nulls and report Draft capture checking

## Description
Resolve R3's adoption trial and deliver R8's bounded report. Null-boundary changes follow the shared mechanical migrations; the capture spike stays disposable throughout.

**Size:** M
**Files:** relevant compiler directives and ScalaPB/Java boundary modules if the null trial succeeds; `.plans/SCALA.md` and `.plans/SCALA_CAPTURE_CHECKING.md` report; scratch spike under `.flow/tmp/fn156/`.
**Touches:** [model/project.scala, model/irgen/project.scala, model/irgen/testdata/**/project.scala, model/framework/realize/**, model/temporal/realize/**, model/irgen/**, .plans/SCALA.md, .plans/SCALA_CAPTURE_CHECKING.md, .flow/tmp/fn156/**]

### Approach

- Compile explicit nulls once over every intended independent build root under the compiler actually pinned at execution. Retain command/version/output and count all findings by boundary. Adopt only when findings are confined to ScalaPB/Java boundaries and fixable there while preserving lifted bytes; otherwise drop the flag and report the count and concrete reason. Avoid broad unsafeNulls or casts hiding findings.
- Run capture checking in a disposable checkout/scratch compiler project for at most two elapsed days. Report-only investigation uses the project research tier, gpt-6-astra at high; implementation and review retain their configured tiers. Re-fetch official compiler documentation for the pinned release; compiler incompatibility or experimental-feature restrictions are report outcomes.
- Exercise valid effect blocks, escaping Draft through return values, retained closures, object fields and nested effects; test whether the compiler rejects each escape and accepts valid uses. Record minimal diagnostic examples, limits, required API/type changes and cost of adoption. Do not merge scratch capture source changes or turn the spike into adoption.
- Review null boundary changes with task 1's exact artifact manifests. Keep compiler check results separate from proof of runtime semantics.

### Investigation targets

**Required:**
- `model/framework/Syntax.scala:125` - Draft context.
- `model/framework/Syntax.scala:171` - effect scope.
- `model/framework/realize` - generic ScalaPB/Java boundary.
- `model/temporal/realize` - typed generated Temporal boundary.
- `model/project.scala` and `model/irgen/project.scala` - compiler versions/options.
- `.plans/SCALA.md` - existing compiler investigation record.

### Quick commands

Run disposable actual-compiler null/capture compiles with recorded start/end timestamps and diagnostics. Hold `/tmp/umpire-heavy-gates.lock` for any production-sized trial. If nulls is retained, rerun its affected full Scala builds and the exact scratch output comparison.

## Acceptance
- [ ] Explicit nulls is either retained with boundary-only fixes and successful intended builds, or removed with actual finding count, toolchain and reason in the report.
- [ ] The capture report states whether the current compiler can prove Draft confinement, valid-use/escape evidence, limitations, adoption cost and the two-day elapsed bound.
- [ ] Capture experiments leave no merged source change; null adoption keeps exact artifact bytes and no new semantic or unsafe blanket exemption.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
