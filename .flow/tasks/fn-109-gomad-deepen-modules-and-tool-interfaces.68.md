---
satisfies: [R5, R18, R19]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.68 Restore explicit scripted Choice Exploration coverage

## Description
Restore exactly three existing fresh Choice Exploration test calls through the existing explicit scripted preparation adapter. Follow [the bounded admission](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-68/admission.md).

**Touches:** [tools/gomad3/runner/runner_test.go, .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-68/**]

Exactly three assignments after final configuration and before the existing call are admitted in TestRunChoiceExplorationExecutesRootAndEveryNonSelectedRank, TestRunChoiceExplorationDivergingPrefixRetainsCompletedRound and TestRunChoiceExplorationExpandsCompleteTargetFailures. Use final config.Preparer and the outer configDependencies.executor. Preserve the entire BASE file after removing only those three assignments, including all sixteen earlier attachments. No production, helper, import, fixture, assertion, public/default, replay/resume/guidance, runtime, generation or lint-policy change.

Quick checks retain meaningful unchanged three-test RED then the original behavioral assertions and prior scripted/dependency/error/default/isolated/real-local/public controls, format/body preservation, affected vet/errortype, configured unfiltered Runner lint and repository fast lint. Root owns frozen full named-outcome/full-block comparisons, boundary reconciliation, integrated source review and lifecycle. A newly reached failure requires separate correction admission.

## Acceptance
- [ ] The three unchanged tests first show actual current preparation-stage RED on bound inputs; exactly three assignments then allow every original behavioral assertion to pass, including immutable round corruption rejection, divergence retention and complete-failure expansion.
- [ ] Removing only the three new assignments recovers BASE runner_test.go byte-for-byte, retaining all sixteen prior attachments and all setup, helpers, comments, fixture data, assertions, channels and cleanup.
- [ ] Prior task-65/66 originals and forwarding/error/stage/default/isolated/real-local/public controls preserve their actual behavior. Outer divergence executor and final target/preparer matching remain intact.
- [ ] Formatting/body preservation, affected vet/errortype and required fast lint pass on bound inputs. Unfiltered configured lint, generated validation and both source-set checks remain accurately reconciled; no new finding or policy waiver.
- [ ] Root frozen ordinary comparison preserves all 673 original names, permitting exactly the three admitted FAIL-to-PASS transitions and no other change. Complete original-base lint blocks retain the current RED50 unchanged; aggregate nonzero exits and errortype reachability stay explicit.
- [ ] Fresh independent integrated review accepts the bounded source change. Formal completion requires all task-owned source acceptance; native fn-128/fn-149 remains deferred and unverified.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
