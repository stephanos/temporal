# Restore explicit scripted Choice Exploration divergence coverage

Root admits fn-109.70 to support R5/R18/R19 and open fn-109.63/fn-112.10 source acceptance. It has no predecessor completion dependency. Root alone selects scope, mints the task, integrates, reviews, commits and controls lifecycle.

BASE is `2af15fcbc0052764f9804fd005dfe298189ef8c7`. Its documentation-only changes retain the product inputs researched at `7727b062b0c263046f0409e8f9d6cf5e58e7c0ef` and observed at frozen `f699252450b8e67f1edb50ed8e4cff4cb6e644c0`. Follow [choice-divergence-next-slice.md](../../fn-112-gomad-determinism-assurance-and-test/task-10/runner-preparation-design/choice-divergence-next-slice.md), SHA-256 `2168c6c9a9fc151a97b4705a677ce4cef6f7082e257eeda9ab0c2cdeb3370875`, for source tracing and retained evidence bindings.

## Exact boundary

**Touches:** [tools/gomad3/runner/choice_exploration_divergence_test.go, .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-70/**]

Add exactly three syntactic assignments using the existing `configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)` statement. Insert once in each existing subtest consumer after its configuration and executor mutation are final, immediately before its existing `exploreWith` call.

- `TestRunChoiceExplorationDivergencePoliciesAndInspection`
- `TestRunChoiceExplorationKeepsOtherErrorsAsHostErrors`
- `TestRunChoiceExplorationRetainsOnlyPrefixMismatchReasons`

Retain the outer `configDependencies.executor`; using its base would erase the scripted divergence/error behavior. Keep `divergenceCampaignConfig` and every helper unchanged. Deleting only these three new lines must reconstruct the entire BASE file byte-for-byte. Preserve all policy rows, 14 host-error rows, seven mismatch-reason rows, counters, causes, prefix mutations, committed-state assertions, real public Inspect and real campaign/round validation. This source-boundary check reinstates no encoded-output compatibility requirement.

## Verification

The combined66/67 packet observed 673 named outcomes with 389 PASS, 272 FAIL and 12 SKIP, plus aggregate lint RED50. Its selected 27 failures count three actual parents and 24 emitted table children. All failed before the intended observations at unsupported-host preparation; they are historical outcomes and predict no successor pass count.

Before any insertion, retain actual unchanged-source preparation RED for the three selected tables at the worker's bound candidate. After attachment, verify all original rows and assertions, including policy stopping, one retained divergence with no failure artifact, unchanged HostError reasons and root-only committed state for rejected candidates, and each accepted mismatch reason through real journal reopening. No additive fixture is required. Retain any newly reached original assertion or validator failure and return it to root for a separately bounded correction.

Keep existing preparation forwarding, operation-error, original-stage, default/bootstrap and isolated controls plus the public profile guard in the focused gate, with their source unchanged. Use pinned tools, `test_dep`, count1, externally bounded commands and one compact receipt binding source/tool/environment identities, argv, numeric exit, elapsed time and immutable raw-log hashes.

Retain formatting, whole-file reconstruction, affected vet/errortype, configured unfiltered Runner lint and required repository fast lint. Both supported source-set static checks, generated validation and other retained source gates need current evidence or exact unaffected-input reconciliation. Root serializes every shared Go/build/lint/generator lane; worker68 currently holds that lane and this draft grants none.

Root owns one future frozen ordinary comparison against combined66/67. Permit outcome changes only for actual emitted names in the root-selected fixture scopes, including these table children. Enumerate names from terminal events; synthesize no parent/intermediate slash names and forecast no PASS total. Require the complete current 50 lint blocks unchanged, with zero introduced or removed findings. Keep aggregate RED and errortype reachability explicit. Fresh independent integrated review and all remaining owned source gates govern completion.

## Exclusions

Exclude `TestRunChoiceExplorationResumePreservesDivergenceIdentity`, `TestProcessExplorationCompletionKeepsRunnerDomainFallback`, killed-process tests and every other consumer/helper, import, comment, assertion, table datum, target metadata, production/public/default path, bootstrap decoder, replay/minimize/guidance, adapter/installation identity, schema/generated input and runtime/toolchain change. Task68 and sibling scopes retain their own workers. Preserve dirty owner spec, MILESTONES.md including fn155, and unrelated user files. Native fn128/fn149 remain deferred and unverified; no native, CI, PR or push authority follows.
