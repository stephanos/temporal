---
satisfies: [R1, R2]
---
# fn-156-let-the-scala-compiler-and-lint-enforce.1 Enable warning checks and prove finite equality evidence

## Description
Implement R1 and the core equality proof for R2. Establish the strict, committed fn-155 comparison baseline before changing build inputs. This is the approach proof; equality rollout follows only after its positive and negative compiler examples succeed.

**Size:** M
**Files:** `model/project.scala`, `model/irgen/project.scala`, positive fixture `project.scala` files, `model/framework/Domain.scala`, `model/framework/IrFile.scala`, compiler tests and scratch proof evidence.
**Touches:** [model/project.scala, model/irgen/project.scala, model/irgen/testdata/**/project.scala, model/framework/Domain.scala, model/framework/IrFile.scala, model/framework/*test.scala, .flow/tmp/fn156/**]

### Approach

- Re-anchor fn-155's committed closure, input hashes and dumper. Freeze the complete filename/raw-byte manifests of both managed artifact trees. Keep the original baseline immutable throughout this spec. The fn-155 effect-name and position projection is not this task's equivalence harness.
- Reuse shared build directives and fixture materialization. Inventory authoring, framework-alone, irgen, positive and refusal fixture callers, including scalafix check/rewrite. Enable the R1 flags with warnings as errors where required; preserve documented gate/refusal build exceptions with specific reasons.
- Fix the discarded registry result with explicit Unit intent. Resolve new diagnostics mechanically; inspect compiler output as well as status because scala-cli can report compiler errors with exit zero.
- Prove how Finite-derived types acquire `CanEqual[T,T]` in their implicit scope before bulk edits. Exercise independently defined model Phase/Fact/State types, parameterized enum cases, records, options and UpTo. Reject broad `CanEqual[Any,Any]`, unrelated-type or union-widening evidence. Keep non-Finite types for task 2.
- Run same-type positive compilation and unrelated-type negative compilation with the actual pinned compiler. Seal a scratch lift/byte comparison of affected representative production roots before rolling out. Any new lifted given or source-position drift remains a failure.

### Investigation targets

**Required:**
- `model/framework/Domain.scala:15` - Finite witnesses and derivation.
- `model/framework/IrFile.scala:80` - discarded registration result.
- `model/project.scala:7` and `model/irgen/project.scala:5` - separate compiler roots.
- `model/irgen/test/Fixtures.test.scala:314` - materialization and located compile-refusal assertions.
- `model/check/Tools.scala` - diagnostics-aware process seam.
- `.flow/tasks/fn-155-name-the-standalone-activitys-repeated.1.md` - pending proof to re-anchor after closure.

### Quick commands

Run `mise exec -- scala-cli test --server=false model/project.scala model/framework` and the new compiler fixture checks. Serialize any production-sized scratch lift through `/tmp/umpire-heavy-gates.lock` using an actual flock/fcntl lock. Record command, complete output, exit status and exact inputs under `.flow/tmp/fn156/`; inherit valid unchanged evidence per MILESTONES.

## Acceptance
- [ ] Complete fn-155 baseline manifests are pinned before edits; every affected production artifact compares byte-identically in the proof, including positions.
- [ ] R1 flags reach every intended build caller; new warnings have mechanical fixes or reasoned fixture-specific exemptions; existing unsupported/werror/crossed fixtures retain their tested diagnostic causes.
- [ ] Finite-derived same-type comparisons compile through discoverable evidence; negative Phase/Fact/State comparisons and evidence-widening examples fail with actual compiler diagnostics.
- [ ] The warning registry fix, compiler fixtures and affected framework tests pass; output inspection catches diagnostics even when process status is zero.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
