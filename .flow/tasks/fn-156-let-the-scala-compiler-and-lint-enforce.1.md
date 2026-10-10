---
satisfies: [R1, R2, R4]
---
# fn-156-let-the-scala-compiler-and-lint-enforce.1 Enable warning checks and prove finite equality evidence

## Description
Implement R1 and the core equality proof for R2, plus the early R4/R9 byte-preserving match proof. The source lane proceeds only after both positive/negative compiler evidence and exhaustive-match compatibility proofs succeed.

Execution-order amendment, explicitly authorized by the owner on 2026-10-10: implement this lane in parallel with fn-146 while fn-157 is deferred. For initial implementation and focused proofs, pin the current integrated local umpire source containing committed fn-155.1–.5 and the complete filename/raw-byte manifests of both managed artifact trees before changing build inputs. This supersedes the requirement to wait for fn-155 closure before starting, not the strict equivalence requirements. Fn-155.6 remains blocked; its missing complete fresh-generation provenance and gates cannot be inherited as passing evidence. Reconcile against the eventual closed fn-155 baseline before final baseline acceptance, full source rollout or spec closure. Never use fn-155's identity/position projection as fn-156's raw-byte oracle. Return precise partial proof and outstanding holds if full equivalence cannot yet be established.

**Size:** M
**Files:** `model/project.scala`, `model/irgen/project.scala`, positive fixture `project.scala` files, `model/framework/Domain.scala`, `model/framework/IrFile.scala`, compiler tests and scratch proof evidence.
**Touches:** [model/project.scala, model/irgen/project.scala, model/irgen/testdata/**/project.scala, model/irgen/testdata/lifts/Scripts.scala, model/framework/Domain.scala, model/framework/IrFile.scala, model/framework/Compose.scala, model/framework/*test.scala, model/irgen/Claims.scala, model/irgen/Declarations.scala, model/irgen/Expressions.scala, model/irgen/Lift.scala, model/irgen/Lifting.scala, model/irgen/Realizations.scala, model/irgen/Structure.scala, model/irgen/test/DeadlinePresets.test.scala, model/irgen/test/DefaultEnds.test.scala, model/irgen/test/Fixtures.test.scala, model/irgen/test/PhaseCapabilities.test.scala, model/irgen/test/QualifiedNames.test.scala, model/temporal/capabilities/Deadline.test.scala, model/check/Tools.scala, .flow/tmp/fn156/source/**]

Warning inventory re-anchor: actual pinned compiler probes also identify a safe-initialization warning in the derived Composition initializer. Task .1 may mechanically correct `model/framework/Compose.scala` while preserving behavior, derived-phase refusal diagnostics and complete raw-byte proofs. Do not suppress the warning broadly or treat the historical one-finding count as exhaustive.

The strict irgen compiler probe records sixteen E175/E176 discarded-value findings across the seven named irgen files and shared `model/check/Tools.scala`. Correct only their Unit intent so the required warning flags can be enforced. Preserve algorithms, admission/refusal classes and diagnostic causes; do not replace enforcement with broad exclusions. The expanded source set remains disjoint from the concurrently admitted Testpilot task.

The next strict test-scope compile records fourteen additional findings in the five named irgen test files. Their Unit-intent corrections and lazy initialization of the immutable `kindRefusals` fixture vector are in scope. Preserve all assertions and negative fixture diagnostic checks; no test-population reduction or warning waiver follows from the inventory.

The full `make lint-model` inventory additionally reports two E176 unused exception values at `model/temporal/capabilities/Deadline.test.scala:191` and `:199`. Task .1 may make only those outer `intercept` result discards explicit Unit intent, retaining their exception assertions and complete bodies. No production Temporal source changes or wider test suppression are admitted.

The positive fixture inventory also identifies deliberately unused Model declaration parameters in `initOrder` and `lifts`, four negative lifter-control discarded expressions and two positive typed request-body collector expressions in `lifts/Scripts.scala`. Fixture-project-only parameter-warning exclusions and precise source/diagnostic/message filters for negative controls are permitted with their complete inventory and reasons; production/global warning exclusions are not. Prefer two explicit Unit-intent edits for the positive collector, only with unchanged lifted fixture bytes and collector behavior; if the collector's supported syntax prevents that proof, retain its source and document two precise fixture-only diagnostic filters instead. Existing refusal classes, causes and column expectations remain unchanged; discrepant diagnostic runners do not supply canonical-suite passing evidence.

Pinned compiler evidence corrects the equality-discovery mechanism: deriving `Finite` alone does not place `CanEqual` in the domain's implicit scope. `Finite` may supply sound same-type evidence via an explicit `framework.Finite.given` import; task .2 owns caller rollout with unchanged source coordinates/artifact bytes. Preserve the automatic-discovery negative proof, unrelated-type, container and Any/union-widening negatives. No broad universal equality evidence or claim of automatic companion discovery is accepted.
## Approach

- Re-anchor fn-155's committed closure, input hashes and dumper. Freeze the complete filename/raw-byte manifests of both managed artifact trees. Keep the original baseline immutable throughout this spec. The fn-155 effect-name and position projection is not this task's equivalence harness.
- Reuse shared build directives and fixture materialization. Inventory authoring, framework-alone, irgen, positive and refusal fixture callers, including scalafix check/rewrite. Enable the R1 flags with warnings as errors where required; preserve documented gate/refusal build exceptions with specific reasons.
- Fix the discarded registry result with explicit Unit intent. Resolve new diagnostics mechanically; inspect compiler output as well as status because scala-cli can report compiler errors with exit zero.
- Prove how Finite-derived types acquire `CanEqual[T,T]` in their implicit scope before bulk edits. Exercise independently defined model Phase/Fact/State types, parameterized enum cases, records, options and UpTo. Reject broad `CanEqual[Any,Any]`, unrelated-type or union-widening evidence. Keep non-Finite types for task 2.
- Run same-type positive compilation and unrelated-type negative compilation with the actual pinned compiler. Seal a scratch lift/byte comparison of affected representative production roots before rolling out. Any new lifted given or source-position drift remains a failure.
- Inventory every current model-enum/state wildcard that R4 must remove. In a disposable project copy, replace TaskQueueSystem.oneMoreDelivery's wildcard with explicit remaining alternatives, then prototype typed lowering of a final unguarded nonbinding exhaustive remainder back to the historical wildcard IR variant. Current Expressions.pattern distinguishes Wildcard/Alternatives, so mere source-line preservation is insufficient. Cover every inventoried pattern shape, compare complete affected root bytes and behavior independently, and keep pre-existing explicit-pattern bytes unchanged. Reject guarded, binding, incomplete and overlapping matches from compatibility lowering; do not inspect baseline artifacts or owner names to choose output. Preserve actual source coordinates rather than mask them. If any shape cannot meet both invariants, stop with the concrete R4/R9 conflict, not a waiver or normalized oracle.

## Investigation targets

**Required:**
- `model/framework/Domain.scala:15` - Finite witnesses and derivation.
- `model/framework/IrFile.scala:80` - discarded registration result.
- `model/project.scala:7` and `model/irgen/project.scala:5` - separate compiler roots.
- `model/irgen/test/Fixtures.test.scala:314` - materialization and located compile-refusal assertions.
- `model/irgen/Expressions.scala:840` and `model/irgen/Context.scala:269` - distinct pattern encodings and real source positions.
- `model/temporal/foundations/taskqueue/system/System.scala:29` - representative lifted enum wildcard.
- `.flow/tasks/fn-155-name-the-standalone-activitys-repeated.1.md` - pending proof to re-anchor after closure.

## Quick commands

Run `mise exec -- scala-cli test --server=false model/project.scala model/framework` and the new compiler fixture checks. Serialize any production-sized scratch lift through `/tmp/umpire-heavy-gates.lock` using an actual flock/fcntl lock. Record command, complete output, exit status and exact inputs under `.flow/tmp/fn156/source/`; inherit valid unchanged evidence per MILESTONES.

- [ ] Complete fn-155 baseline manifests are pinned before edits; every affected production artifact compares byte-identically in the proof, including positions.
- [ ] R1 flags reach every intended build caller; new warnings have mechanical fixes or reasoned fixture-specific exemptions; existing unsupported/werror/crossed fixtures retain their tested diagnostic causes.
- [ ] Finite-derived same-type comparisons compile through discoverable evidence; negative Phase/Fact/State comparisons and evidence-widening examples fail with actual compiler diagnostics.
- [ ] The warning registry fix, compiler fixtures and affected framework tests pass; output inspection catches diagnostics even when process status is zero.
- [ ] Every inventoried enum/state wildcard shape has a proven exhaustive source rewrite with narrow typed lowering, unchanged behavior and raw artifact bytes; pre-existing explicit matches and refusal causes stay unchanged. An infeasible shape stops the source lane with an unresolved R4/R9 conflict.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

## Acceptance
- [ ] TBD
