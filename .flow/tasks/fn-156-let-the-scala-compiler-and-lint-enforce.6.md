---
satisfies: [R3]
---
# fn-156-let-the-scala-compiler-and-lint-enforce.6 Trial explicit nulls across compiler roots

## Description

Resolve R3's explicit-nulls adoption trial after the shared source migrations. R8 is the independent report-only task .8; this task does not edit its scratch checkout or report.

**Size:** M
**Files:** relevant compiler directives, ScalaPB/Java boundary modules only if the null trial succeeds, and `.plans/SCALA.md` null-trial section.
**Touches:** [model/project.scala, model/irgen/project.scala, model/irgen/testdata/**/project.scala, model/framework/realize/**, model/temporal/realize/**, model/irgen/**, .plans/SCALA.md, .flow/tmp/fn156/source/nulls/**]

## Approach

- Compile explicit nulls once over every intended independent build root under the compiler pinned at execution. Retain command/version/output and count all findings by boundary, reading diagnostics even on exit zero.
- Adopt only when findings are confined to ScalaPB/Java boundaries and fixable there while preserving lifted bytes; otherwise drop the flag and report the count and concrete reason. Avoid broad unsafeNulls or casts hiding findings. The gate's independent tooling caller gets a specific disposition, not an unmentioned omission.
- Review any retained boundary changes with task 1's exact artifact manifests. Rerun affected intended builds and negative fixtures. Keep compiler results separate from proof of runtime semantics and preserve existing warning/error causes.
- Apply the owner validation policy: defer a validation stuck beyond one hour across attempts unless it blocks all other available work; preserve actual diagnostics, unmet acceptance and revisit conditions, and continue available independent work.

## Investigation targets

**Required:**
- `model/framework/realize` - generic ScalaPB/Java boundary.
- `model/temporal/realize` - typed generated Temporal boundary.
- `model/project.scala`, `model/irgen/project.scala` and `model/check/project.scala` - independent callers/options.
- `.plans/SCALA.md` - compiler investigation record.

## Quick commands

Run actual-compiler null compiles with recorded inputs, version and diagnostics. Hold `/tmp/umpire-heavy-gates.lock` with actual flock/fcntl ownership for production-sized trials. If retained, rerun affected full Scala builds and exact scratch output comparison.

## Acceptance

- [ ] Explicit nulls is either retained with boundary-only fixes and successful intended builds, or removed with actual finding count, toolchain and reason in the report.
- [ ] All independent compiler callers and negative/refusal fixtures have a documented outcome; compiler diagnostics are not inferred from exit status alone.
- [ ] Any null adoption keeps exact artifact bytes, original semantics and no unsafe blanket exemption; stuck validation preserves unmet evidence rather than passing credit.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
