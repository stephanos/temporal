---
satisfies: [R7]
---
# fn-194-parametric-models-constants-and-finite.4 DSL completion: document parameters in SEMANTICS and the README

## Description
**DSL completion phase (owner, 2026-10-17).** Describe the two kinds of parameter (a domain the Model is generic over as a `Finite`-bounded type parameter, a constant as a value parameter of a liftable kind), instantiation in the suite, naming, and the unchanged object form, with Voting as the worked example. Add actual refusal texts to the API contracts list. State that lifting an instance is the IR phase.

**Size:** S

**Files:** `model/framework/SEMANTICS.md` (the Machines section and a new Parameters subsection), `model/framework/README.md`.

**Touches:** `model/framework/SEMANTICS.md`, `model/framework/README.md`

### Verification
- Docs match the final .2/.3 API, not the historical indexed-state prototype. SEMANTICS source links point at the actual Voting sample/test as appropriate. Its current native Machine description must not tell an author that only objects or explicit record-state declarations are supported; retain clearly separated historical IR material only where relevant.
- Worked Voting construction uses actual landed constructor types and test-owned domains/quorums. Compile the actual standalone excerpt/client against a fresh workspace and the integrated declarations; do not treat prose or an old pre-format client as a compiler proof. Keep public examples free of private MachineDeclaration/NativeModel carriers, extra Data/StateData bounds, hidden worlds, implicit/default construction, manual instance naming or future ModelSuite syntax.
- Describe first checked stable holder naming, aliases preserving that name, per-instance state/schema/starts/catalogs, constructor refusals and selected generic shape validation accurately. Verify error text against actual implementation/tests. Describe realization as binding an actual instance without claiming instance lifting/IR/Case identity support.
- Supporting constructor-set cardinality bounds are narrow, immutable and positive, not arbitrary state/callback expressions. Named calls on selected instance-owned records preserve defaults and exact input types. Do not expose private ownership-token implementation details as author API.
- Run focused documentation/reference checks and source-link validation, formatting/diff checks, and classify the actual diff. GuideSourcesTest's disabled Temporal inventory remains held; run applicable native reference/link substitutes without claiming the held full gate passed. REFERENCE remains user-facing and source-file-free; do not edit it or introduce Scala-file links there.

**Integrated predecessor.** .3 is done with source `cb1f2fead8`, external fixture `bb8749d7cf`, closure `31e294207d`, exact-byte supplemental receipt `9d98798903`; root milestone checkpoint `5885730f52`. Native635 (623 retained plus12), class lint7, retained RuleSuite31, boundary/loop46, source classification166 and fresh declaration/core/plugin/sample source/jar plus exact-current/committed-byte external clients passed. Handover `.flow/tmp/fn1943/{summary.md,evidence.json,final-commands.md}`. Backend completion snapshot has19 commands; tracked supplemental receipt/handover adds5 to24, because flowctl correctly refuses re-done. No source changes followed tested bytes.

**Preparation.** `.flow/tmp/fn1944/admission-notes.md` is source-backed guidance, not acceptance evidence. Read actual final source/docs and run an own focused pre-edit documentation baseline; no baseline handoff after intervening MILESTONES edits. Root owns MILESTONES; do not edit it.

**Holds.** No Scala production changes, Temporal, IR/export/testdata, Go/schema/backend/generated/lifting/regeneration. All222 disabled Temporal files, broad GuideSourcesTest and two real-Go sample bodies stay held before execution. No Make green claim if model-check-changed still routes to held Temporal. Subsequent ModelSuite/oracle documentation migrations remain fn-195/fn-196; avoid unrelated rewrites. Review stays deferred to the phase batch, plan-sync false, tracker inactive, parent spec OPEN, no publication/history rewrite authority.
## Acceptance
- [ ] SEMANTICS describes parameters, kinds, instantiation and naming with Voting as the example
- [ ] README test guidance updated

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
