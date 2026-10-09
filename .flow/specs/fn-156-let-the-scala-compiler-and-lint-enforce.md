# Let the Scala compiler and lint enforce Model correctness

## Goal & Context
<!-- scope: business -->

Model authors and reviewers today catch several classes of mistake by reading:
- a builder result dropped on the floor;
- an `object` read before it is initialized;
- `==` between two unrelated `Phase` or `Fact` types;
- a section out of order;
- a new machine nobody added to the generic law tests.

This spec moves those checks to the compiler, scalafix and tests that find their subjects themselves. Authors then get the error while they type, and reviewers stop checking for it.

Source: a review of the Scala model build on 2026-10-09. The owner chose items 1–9, 22–29 and 31 of that review's list for this spec. Items 10–21, the compile-versus-lift gap, are recorded as notes on fn-141. Mutation testing and Stainless are out.

Evidence from a scratch compile of `framework/` and `temporal/` (332 files) on 2026-10-09, with no source change:
- `-Wunused:all -Wvalue-discard -Wnonunit-statement -Wsafe-init -Wimplausible-patterns` reported one warning: an unused `+=` result in the IR file registry.
- `-language:strictEquality` reported 351 errors. Nearly all are same-type comparisons of model enums and states that lack `CanEqual`, for example `Custody` 22, `Timeout` 21 and `Phase` 16.

## Architecture & Data Models
<!-- scope: technical -->

**Compiler options.** The model build's options (today `-Werror -deprecation -feature -unchecked -Wunused:imports`) gain:
- `-Wunused:all`
- `-Wvalue-discard`
- `-Wnonunit-statement`
- `-Wsafe-init`
- `-Wimplausible-patterns`
- `-language:strictEquality`

They apply to the framework, the Temporal Models, irgen and the lift fixtures' build. Where a fixture must not compile, it keeps its existing exemption.

**Equality.** `Finite` provides `CanEqual` for every type that derives it, so one framework given covers the model types. Types that do not derive `Finite` and are compared with `==` derive or declare `CanEqual` themselves. Scalafix `DisableSyntax.noUniversalEquality` turns on.

**Explicit nulls trial.** `-Yexplicit-nulls` is compiled once. If its findings are confined to the ScalaPB/Java boundary and fixable there, it is kept. Otherwise it is dropped, with the count and reason recorded.

**Convention rules.** Scalafix (or irgen, where it already reads the tree) enforces:
- the section order of a machine (states, refinement, effects, monitors, rules, properties, capabilities, queries) and of a composition;
- `in(...)` written dotted, never infix;
- the permitted imports of the module map for Scala packages;
- no wildcard `case _ =>` on a model enum or state.

**Cross-level type confusion.** A lint rule bans importing one level's `Phase`, `Fact` or `State` unqualified into another level's file. Renaming the types per level would change IR type identities, so it is out of scope unless the owner asks for it.

**Tests that find their subjects.** The generic law tests run over every machine reachable from the `irFile` exports, so a new machine is covered without being listed. The laws are refinement totality, closedness, determinism and binding order. Hand-listed tests such as the role-refinement test become instances of this. Where a test names examples but its subject is `Finite`, it iterates the whole domain instead. The step-table pin, which today covers only Product and System, extends to every machine through the interpreter's tables built from the lifted IR. It reuses fn-155's dump if that has landed.

**Capture checking spike.** A time-boxed spike checks whether `language.experimental.captureChecking` can prove that a `Draft` never escapes its `effect { }` block. It works in a scratch branch, and its output is a written finding, not a merged change.

## Edge Cases & Constraints
<!-- scope: technical -->

- **Scheduling.** The owner schedules this spec immediately after fn-155 closes on the integrated local baseline and before the authoring batch. Its only prerequisite is fn-155. Re-anchor names, files and proof inputs to that committed closure before implementation. The conductor records the authoring batch's reverse dependencies and owns milestone scheduling.
- **No meaning change.** Flags, lint rules and tests change no Model meaning. A regeneration after this spec is byte-identical in `model/ir` and `model/cases`. A `CanEqual` given must not alter what irgen lifts.
- **Exact bytes include positions.** Preserve declaration source locations during mechanical edits. Neither position stripping nor fn-155's mapped-identity projection satisfies this spec. Unexpected byte drift stops the comparison and remains an unresolved acceptance failure.
- **Discovered laws.** The subject universe comes from initialized exports and their complete root closure, independently cross-checked against lifted machine identities. Include machines reached through derivations, refinements, compositions, Queries, capabilities and realizations. Determinism means repeatable ordered results for the same state and class, preserving declared alternatives. Single-successor assumptions must not reject named choices. List each inapplicable or failing law with its machine and reason; never silently exclude one.
- **Verification inheritance.** Reuse predecessor evidence only when commands, source scope, fixtures and environment still apply. Strict compiler and lint changes invalidate affected Scala checks. Full native Model, Go and Case generator failures from resource exhaustion remain recorded under deferred fn-157; Quint resource work remains fn-154, and Activity semantic failures remain the Activity batch's obligations. These dispositions supply no passing credit for R9.
- **Lift fixtures.** irgen's refusal fixtures (`unsupported`, `werror`, `crossed`) must keep failing for the reason they test, not because of a new flag.
- **Scalafix toolchain.** Scalafix runs under JDK 25 because of a JDK 27 incompatibility. New rules must run in that setup or document a workaround.
- **Shared files.** This spec touches every model file mechanically, so it serializes with any spec editing model sources at the same time.
- **Internal parallelism.** The source-changing lane remains .1 → .2 → .3 → .4 → .5 → .6: compiler directives, typed lifter checks, framework law helpers and finite tests share inputs. Report-only capture task .8 starts from the same committed fn-155 baseline without waiting for that lane, in its own disposable checkout. Integration .7 joins .6 and .8; only .7 edits shared documentation.
- **Stuck validation.** A validation consuming over one hour across its attempts is deferred with exact logs, unmet acceptance and revisit conditions, unless it blocks all other available work. Continue independent work; never treat deferral as passing evidence or silently narrow a full gate.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** The model build passes `-Wunused:all -Wvalue-discard -Wnonunit-statement -Wsafe-init -Wimplausible-patterns` under `-Werror`, with the one current finding fixed. Errors: a lift fixture that must not compile keeps failing for its tested reason. A new warning in irgen or the fixtures is fixed or exempted with a reason.
- **R2:** `-language:strictEquality` is on and scalafix `noUniversalEquality` is enabled. `Finite` supplies `CanEqual`, and the remaining types declare it. Comparing two unrelated model types with `==` fails to compile, shown by a negative compile test. Errors: a regeneration differs from the baseline in any byte.
- **R3:** `-Yexplicit-nulls` is either on, with its findings fixed, or recorded as dropped with its finding count and reason. No error surface beyond the compile.
- **R4:** Lint rules enforce machine and composition section order, dotted `in(...)`, the module map's permitted Scala imports, and no wildcard case on model enums and states. Each rule fails on a seeded negative example and passes on the current tree after fixes. Errors: a rule that cannot run under the JDK 25 scalafix setup is implemented as an irgen check, or the gap is recorded.
- **R5:** A lint rule bans unqualified cross-level imports of `Phase`, `Fact` and `State`, and fails on a seeded negative example. No renaming of the types is done.
- **R6:** The generic law tests discover every machine through the `irFile` exports. Adding a machine to an export, without editing any test, puts it under refinement totality, closedness, determinism and binding-order checks. Errors: a machine that cannot satisfy a law is listed with its reason, never silently skipped.
- **R7:** Tests whose subject is a `Finite` domain iterate it fully. The step-table pin covers every activity and Nexus machine through interpreter-built tables. Errors: a domain too large to iterate within the test's time bound is recorded, with its size and the chosen sample.
- **R8:** A capture-checking spike, time-boxed to two days, reports whether `Draft` confinement can be proved under the compiler in use, and what it would cost. No merged source change. No error surface beyond the report.
- **R9:** `make umpire-gen-model` produces a byte-identical `model/ir` and `model/cases`. The model gate, `make lint-model` and the irgen fixture build pass.

## Early proof point

Task fn-156-let-the-scala-compiler-and-lint-enforce.1 proves both equality evidence and byte-preserving exhaustive-match enforcement before rollout. The current lifter emits different variants for wildcards and alternatives (`Expressions.scala:840`); a direct rewrite of TaskQueueSystem's `oneMoreDelivery` changes bytes. In a disposable copy, replace that wildcard by the explicit remaining enum cases and prove source-aware typed lowering can emit its historical wildcard encoding only when a final unguarded, nonbinding branch covers exactly all remaining cases. Preserve original branch order, results and source positions; independently compare semantics and raw production artifact bytes. Inventory every existing affected match and require the proof to cover each pattern shape before rollout. Existing explicit-match encodings must remain unchanged. If equality or lowering cannot meet these constraints, stop the source lane and surface the R4/R9 conflict; do not grandfather wildcards, alter goldens, normalize comparison inputs or relax acceptance.

## Quick commands

```bash
make lint-model
make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks
make umpire-check-cases
```

## Boundaries
<!-- scope: business -->

- **Compile-versus-lift gap:** items 10–21 are notes on fn-141, not part of this spec.
- **Out:** mutation testing and Stainless.
- **No renaming of `Phase`, `Fact` or `State` types:** that would change IR identities.
- **No Model meaning change and no Go change.**
- Broad generated API drift verification and new CI coverage remain outside this spec, consistent with the project's declined-concept record. Existing full gates and focused negative fixtures remain required.

## Decision Context
<!-- scope: both -->

- **Why `CanEqual` comes from `Finite`:** the cheapest route to strict equality. Nearly all model types already derive `Finite`, so one given replaces hundreds of `derives CanEqual` edits.
- **Rejected: renaming per-level types.** It would churn IR identities across every downstream spec. The import-ban lint gives most of the safety at no identity cost.
- **Capture checking stays a spike:** it is experimental in the compiler, and the owner wants the finding before any adoption.
- **Reuse section-order checks.** Extend the existing machine and composition declaration checks rather than introduce a second interpretation of their order. The later lifter refactor carries these checks and their located diagnostics with its lint extraction.
- **Equality lint policy.** Keep `noUniversalEquality` enabled and prove its interaction with native typed `==` in the compiler proof. Prefer existing precise line-scoped rule suppression for compiler-checked model comparisons if the built-in rule is syntactic. Any exemption names its reason and keeps unrelated-type compile failures; broad file or project disabling is not accepted.
- **Shared lint consumers.** Equality activation includes `model/check` and its tests because Makefile's shared Scalafix caller lints that root. Enable strict equality there with sound evidence before permitting exact typed-comparison suppressions; keep the gate's documented no-Werror exception and diagnostic-aware process behavior intact.
- **Wildcard compatibility is production lowering, not an oracle projection.** The narrow typed lowering above makes explicit exhaustive source enforceable while preserving the existing semantic IR representation. It may not consult baseline artifacts, match owner names, override positions, or canonicalize pre-existing explicit matches indiscriminately. A complete inventory and negative guarded, binding, incomplete and overlapping-match fixtures bound the compatibility rule. Task .3 implements only the behavior actually established by .1's proof.
- **Stakeholders.** Model authors get compiler and lint diagnostics, reviewers get discovered law coverage, and the gate's operators keep their current entry points and explicit failure statuses. Runtime behavior and deployment configuration do not change.
- **Short research scope.** Codebase, dependency, memory and gap analysis ground this plan. External research scouts are skipped at short depth. Implementers consult the compiler and Scalafix's official documentation for the pinned toolchain when testing equality evidence, nulls and capture checking.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | The model build passes `-Wunused:all -Wvalue-discard -Wnonunit-statement -Wsafe-init -Wimplausible-patterns` under `-Werror`, with the one current finding fixed. Errors: a lift fixture that must not compile keeps failing for its tested reason. A new warning in irgen or the fixtures is fixed or exempted with a reason. | fn-156-let-the-scala-compiler-and-lint-enforce.1, fn-156-let-the-scala-compiler-and-lint-enforce.7 | — |
| R2 | `-language:strictEquality` is on and scalafix `noUniversalEquality` is enabled. `Finite` supplies `CanEqual`, and the remaining types declare it. Comparing two unrelated model types with `==` fails to compile, shown by a negative compile test. Errors: a regeneration differs from the baseline in any byte. | fn-156-let-the-scala-compiler-and-lint-enforce.1, fn-156-let-the-scala-compiler-and-lint-enforce.2, fn-156-let-the-scala-compiler-and-lint-enforce.7 | — |
| R3 | `-Yexplicit-nulls` is either on, with its findings fixed, or recorded as dropped with its finding count and reason. No error surface beyond the compile. | fn-156-let-the-scala-compiler-and-lint-enforce.6, fn-156-let-the-scala-compiler-and-lint-enforce.7 | — |
| R4 | Lint rules enforce machine and composition section order, dotted `in(...)`, the module map's permitted Scala imports, and no wildcard case on model enums and states. Each rule fails on a seeded negative example and passes on the current tree after fixes. Errors: a rule that cannot run under the JDK 25 scalafix setup is implemented as an irgen check, or the gap is recorded. | fn-156-let-the-scala-compiler-and-lint-enforce.1, fn-156-let-the-scala-compiler-and-lint-enforce.3, fn-156-let-the-scala-compiler-and-lint-enforce.7 | — |
| R5 | A lint rule bans unqualified cross-level imports of `Phase`, `Fact` and `State`, and fails on a seeded negative example. No renaming of the types is done. | fn-156-let-the-scala-compiler-and-lint-enforce.3, fn-156-let-the-scala-compiler-and-lint-enforce.7 | — |
| R6 | The generic law tests discover every machine through the `irFile` exports. Adding a machine to an export, without editing any test, puts it under refinement totality, closedness, determinism and binding-order checks. Errors: a machine that cannot satisfy a law is listed with its reason, never silently skipped. | fn-156-let-the-scala-compiler-and-lint-enforce.4, fn-156-let-the-scala-compiler-and-lint-enforce.7 | — |
| R7 | Tests whose subject is a `Finite` domain iterate it fully. The step-table pin covers every activity and Nexus machine through interpreter-built tables. Errors: a domain too large to iterate within the test's time bound is recorded, with its size and the chosen sample. | fn-156-let-the-scala-compiler-and-lint-enforce.5, fn-156-let-the-scala-compiler-and-lint-enforce.7 | — |
| R8 | A capture-checking spike, time-boxed to two days, reports whether `Draft` confinement can be proved under the compiler in use, and what it would cost. No merged source change. No error surface beyond the report. | fn-156-let-the-scala-compiler-and-lint-enforce.7, fn-156-let-the-scala-compiler-and-lint-enforce.8 | — |

> HTML render lens: open local `.flow/artifacts/fn-156-let-the-scala-compiler-and-lint-enforce/spec.html` (ignored, regenerable; Markdown is the record). <!-- flow-next:artifact-link -->
| R9 | `make umpire-gen-model` produces a byte-identical `model/ir` and `model/cases`. The model gate, `make lint-model` and the irgen fixture build pass. | fn-156-let-the-scala-compiler-and-lint-enforce.7 | — |
