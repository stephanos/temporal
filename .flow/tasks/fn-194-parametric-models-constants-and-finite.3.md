---
satisfies: [R5, R6]
---
# fn-194-parametric-models-constants-and-finite.3 DSL completion: Bank, Voting and TwoPhaseCommit take their world as parameters

## Description
**DSL completion phase (owner, 2026-10-17).** Rewrite the three drifted samples as parametric Models and move their concrete worlds into their test files: Bank takes its address type (`[Addr: Finite]`), `holder` and `supply`; Voting takes its values and quorum set; TwoPhaseCommit takes its resource-manager set. Each test file declares the enums and instantiates the Model with today's constants, so its intended native answers are reproduced unchanged on the default budget. A changed mathematical answer is a finding to fix, not a regeneration. The other three samples keep the object form. Realizations bind to an actual Model instance (R6).

**Size:** L

**Files:** `model/framework/samples/{Bank.scala,Bank.test.scala,Voting.scala,Voting.test.scala,VotingBehavior.test.scala,TwoPhaseCommit.scala,TwoPhaseCommit.test.scala,SampleClaims.test.scala}`, focused sample-owned instance-realization tests/clients, `samples/testdata/native-migration/inventory.md`; narrowly required `dsl/Syntax.scala` progress-bound admission and focused `dsl/ParametricBounds.test.scala`; existing native realization helpers if the instance proof requires them.

**Touches:** `model/framework/samples/**`, `model/framework/realize/**`, `model/framework/dsl/Syntax.scala`, `model/framework/dsl/ParametricBounds.test.scala`, `model/framework/dsl/InputSource.scala`, `model/framework/dsl/InputRecords.test.scala`, `model/framework/dsl/Action.scala`, `model/framework/checker/NativeCheck.scala`, `model/framework/checker/ParametricActionNames.test.scala`

**Integration-discovered class-authoring fix (2026-10-18).** Strict sample compilation in `sample-fourth.log` shows that `voter.voteFor(a = Acceptor.a2, b = 0, v = Value.v1)` loses the concrete enclosing Model substitutions when `InputSource.construct` reconstructs the nested record through an unqualified companion. Supplied-record calls establish the intended operation but do not resolve this defect. Preserve the sample's named call and fix the narrow constructor-reconstruction seam in `InputSource.scala`, with regression controls in `InputRecords.test.scala` and a fresh separate-library client. Retain exact receiver/path-dependent input types, defaults and named/empty-call refusals; no casts, erased input types, alternate world or new public API. Capture the real red before the fix. This is supporting class-authoring work for the sample migration, not a waiver to leave valid named calls broken.

**Integration-discovered action naming (2026-10-18).** The migrated Bank's pinned Leaky counterexample reports `(king,donkeykong,banana,-1)` instead of the original `send(king,donkeykong,banana,-1)` (`sample-sixth.log`). `NativeCheck.scan` currently reads static `MODULE$` objects only, missing class-instance-owned Actors. Narrow native instance discovery in `checker/NativeCheck.scala` and regress it in a focused `checker/ParametricActionNames.test.scala` under `framework.checker`, so the existing public-API `CheckerTest` package and registrations remain unchanged; preserve the original Bank trace assertion, exact action identity and both-instance independence. Do not use hand-written names, test aliases or weakened expectations. Discovery must not execute effects/guards/claim bodies, accidentally adopt a foreign owner's declarations, or let the static feature cache suppress discovery of later instances. Preserve existing object/derived/composition behavior and standalone channels/claims; any unsupported discovery surface remains refused rather than guessed.

Internal steps/timers share `Actor.system`, so actor identity alone cannot prove an Internal declaration's owner. A focused foreign-Internal alias red must precede narrowly required private owner metadata in `dsl/Action.scala`, retained through existing `boundTo` copies and checked by exact identity during instance discovery. Preserve the public step/timer syntax, declaration identity and declaration-only/core/plugin compatibility; no public naming API, global owner registry or claim that the old static alias ambiguity proves correctness.

### Verification
- Run an own focused pre-edit sample baseline; conductor milestone edits mean the preceding receipt cannot replace that run. Preserve all existing native registrations/assertion purposes and the committed 623-test baseline coverage.
- Bank, Voting and TwoPhaseCommit native suites pass for the reproduced worlds with the same default-budget verdicts, negative controls, concrete state/trace expectations and query targets. Bank's actual committed input domain is -1 through supply+1: 160 classes for four addresses, two denominations and supply2, not the stale historical 0..3/128 paragraph. Clarify current inventory prose without rewriting old evidence or fitting behavior to it.
- A second Voting instance with a different valid quorum family checks with a different reachable graph size. Independently derive any exact expectation and confirm it; scratch drafts are not execution evidence. Preserve ballot0/1 and sentinel-1 semantics, Consensus projections and the native unknown cross-refinement answer.
- Realization coverage compiles against actual instances in fresh realization-mode source and separate-library clients, with a lawful instance positive alongside refusal of a class/constructor in place of an instance. No current sample-specific realize/export fixture exists to repoint; add focused native evidence rather than claim a nonexistent migration. Prefer sample-owned fixtures over a realize-core dependency on samples.
- If progress bounds need constructor-constant support, establish real red tests first; preserve positive-literal admission, refuse state-dependent/arbitrary callback bounds, and validate accepted computed bounds as positive. Three managers must retain the authored bound3/longest3; do not freeze3 for larger worlds or silently substitute unbounded progress.
- Run affected native DSL/realize/samples, 31 retained RuleSuite and seven class-lint controls, boundary/loop/source-lint checks, fresh declaration-only/core/plugin builds and six-sample source/jar realization-mode clients under unchanged strict flags. Formatting and diff checks remain required. Quick uses the model README native recipes, not held Make/Temporal execution.

**Integrated predecessor.** fn-194.2 is done with source `e8977bcae6`, registration `b33e7e36b4`, closure `6e345185f8`; 623 native, seven class-lint, 31 retained lint, six cross-jar clients and 46 boundary/loop controls passed. Public author bounds stay Finite-only; no integer-address fallback, additional Data/StateData bounds, manual naming argument, hidden sample-world singleton, implicit world or default constructor values.

**Source-backed integration seams.** `SampleClaims.judged` must forward its explicit owner to the new exact-owner `ClaimEvaluation.evaluate` overload; fix the shared helper once, not local duplicates. Do not subclass a Machine class to host test queries: inherited claim getters fail exact section ownership. Test-owned `Questions extends Derived(model)` is an existing-vocabulary option sharing the actual vars/actions; document any question-namespace movement separately from unchanged mathematical verdicts. Voting's Consensus may be a sibling generic class instantiated directly in `Refinement(...)`, with tests retaining that exact `refinement.of` instance; matching source/product action identities are not a native refinement-engine requirement. Preserve equivalent product operations rather than introducing a registry or illegal nested Model declaration. Review the scratch ownership audit before selecting an arrangement.

**Prepared drafts, not acceptance evidence.** `.flow/tmp/fn1943/{bank-draft,voting-draft,twopc-draft}/` contain independent uncompiled replacements/patches and notes. Inspect and adapt them; do not blindly apply or trust guessed sizes. The generic Vocabulary/world extraction map is `.flow/tmp/fn1943/parallel-domain-map.md`. The TwoPhaseCommit draft correctly identifies the current literal-only `within` refusal for `resourceManagers.size`; settle that supporting native seam lawfully rather than weakening the promise.

**Holds.** No Temporal, IR/export/testdata, schema, Go, backend, generated files, lifting or regeneration work is authorized. Broad GuideSourcesTest and the two real-Go sample bodies remain held before execution. `model-check-changed` routes to held Temporal sources: record native substitutes and do not claim Make green. Prototype/design evidence stays historical. Documentation migration is .4; source-free user-facing REFERENCE must remain free of Scala-file pointers. The spec stays open; review remains deferred to the phase batch, tracker bridge inactive, no publishing authority.
## Acceptance
- [ ] Bank, Voting and TwoPhaseCommit are parametric; their tests declare the world and instantiate them
- [ ] Intended answers unchanged on the default budget for the reproduced instances
- [ ] Realization fixtures compile against instances

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
