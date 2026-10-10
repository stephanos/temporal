---
satisfies: [R4, R5]
---
# fn-156-let-the-scala-compiler-and-lint-enforce.3 Enforce Model conventions and Scala import boundaries

## Description

Implement R4 and R5 by extending existing declaration checks and their seeded fixtures. Shared Model edits follow equality activation to prevent overlapping migrations.

**Size:** M
**Files:** `model/irgen/Order.scala`, `model/irgen/Structure.scala`, typed syntax/import checker support, `model/irgen/test/Fixtures.test.scala`, refusal fixtures, affected Model sources and current gate/lint integration.
**Touches:** [model/irgen/**, model/check/**, Makefile, model/temporal/**, model/framework/**, .flow/tmp/fn156/source/**]

## Approach

- Reuse Order's section sequence and its composition syncs-as-rules handling. Extend seeded positive/negative section fixtures; do not maintain a second sequence in Scalafix. Preserve located diagnostic owner/file/line behavior for the later fn-141 extraction.
- Use typed tree identity for enum/state whole-case wildcard checks and cross-level imports. Check wildcard imports as well as explicit Phase/Fact/State imports; permitted renamed aliases and qualified references pass. Reject whole-scrutinee `case _ =>` and a name bound to that whole-case wildcard when the scrutinee is a model enum/state. Permit explicit constructor patterns with unconstrained payloads such as `case Handler.done(_)`: these are explicit enum cases, not whole-case catch-alls. Seed a positive fixture for the non-final nested payload match in Nexus ClosePolicy.nothingOwed and prove its existing pattern bytes stay unchanged. Keep unrelated Scala collection/error matches usable.
- Enforce dotted membership and the current Scala portion of the module map, covering production and tests separately. Resolve package ancestry, kind-owned Products and fixture exceptions from the map rather than textual prefix guesses; no feature dependency in framework/shared kit.
- Implement checks in Scalafix only when compatible with JDK 25, otherwise use the existing irgen/gate typed-check seam allowed by R4. A tool limitation has an exact rule/coverage disposition; do not record a blanket pass.
- Implement task 1's proven source-aware typed compatibility lowering in Expressions' match/pattern seam before rewriting inventoried enum/state wildcards into exhaustive alternatives. The rule preserves historical wildcard encoding only for proven final unguarded nonbinding exhaustive remainders; it cannot consult baseline artifacts, match named owners, mask coordinates or change pre-existing explicit patterns. Seed incomplete, guarded, binding and overlapping refusals plus unchanged explicit-pattern fixtures. Preserve branch results/order and real positions. Any byte drift fails the immutable pin; an unproven shape blocks rollout rather than grandfathering it.

## Investigation targets

**Required:**
- `model/irgen/Order.scala:373` - machine and composition order.
- `model/irgen/Structure.scala` - package and level ownership.
- `model/irgen/testdata/sectionOrder/SectionOrder.scala` - seeded convention refusals.
- `model/irgen/test/Fixtures.test.scala:1325` - located fixture assertions.
- `model/temporal/features/nexus/standalone/Realization.scala:11` - permitted cross-level alias.
- `model/temporal/features/nexus/workflow/system/ClosePolicy.scala:645` - explicit constructor payload wildcard positive; re-anchor after fn-155 closure.
- `.plans/UMPIRE_MODULES.md:30` - authoritative permitted Scala imports.

## Quick commands

Run the changed seeded fixture tests and gate checker tests, then `make lint-model` under JDK 25. Run the irgen fixture suite with retained refusal causes. Scratch lifts use the shared lock and exact task 1 byte manifests.

- [ ] Machine/composition order, dotted in, permitted imports and enum/state wildcard rules each fail on a seeded violation and pass on corrected current sources.
- [ ] Cross-level explicit and wildcard imports fail; qualified references and permitted renamed aliases pass, including Nexus kind-owned Product imports.
- [ ] Type-directed positives cover unrelated wildcard matches and explicit constructor payload wildcards (including non-final Handler.done(_)); their existing encodings stay unchanged. Whole-case enum catch-alls fail with file/line/rule diagnostics; exemptions stay narrow and reasoned.
- [ ] Every rule runs in the existing JDK 25 workflow or the permitted existing irgen/gate seam with its coverage stated; exact production artifact bytes stay pinned.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

## Acceptance
- [ ] TBD
