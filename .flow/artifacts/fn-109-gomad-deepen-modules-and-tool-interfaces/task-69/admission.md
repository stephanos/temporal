# Restore explicit scripted retained-success coverage

Root admits fn-109.69 to support R5/R18/R19 and open fn-109.63/fn-112.10 source acceptance. It has no predecessor completion dependency. Root alone selects scope, mints the task, integrates, reviews, commits and controls lifecycle.

BASE is `2af15fcbc0052764f9804fd005dfe298189ef8c7`. Its documentation-only changes retain the product inputs researched at `7727b062b0c263046f0409e8f9d6cf5e58e7c0ef` and observed at frozen `f699252450b8e67f1edb50ed8e4cff4cb6e644c0`. Follow [retained-success-next-slice.md](../../fn-112-gomad-determinism-assurance-and-test/task-10/runner-preparation-design/retained-success-next-slice.md), SHA-256 `f9b84fd1d78875610b9720f167932d2a4b9eef7d809f900e18400a2c1c248348`, for source tracing and retained evidence bindings.

## Exact boundary

**Touches:** [tools/gomad3/runner/retention_test.go, .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-69/**]

Add exactly two syntactic assignments using the existing `configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)` statement.

- In `TestRunCountsASharedTargetInFullAgainstTheSuccessByteLimit`, attach inside the local `run` closure after `SuccessBytesLimit` is final and immediately before its existing `return exploreWith(...)`. Both original invocations must construct and use their own final preparer/executor.
- In `TestRunRetainsSameOutputSuccessesWithMatchingDiskAndJournalCounts`, attach after `SuccessBytesLimit` is final and immediately before its existing `exploreWith` call.

Deleting only these two new lines must reconstruct the entire BASE file byte-for-byte. Preserve real source-target mutation/hash/size, copied target verification, shared-inode assertion, measured reduced-byte-limit failure, same-output distinct artifacts, seed/output identities, disk/journal/summary counts and full-file stored-byte totals. This source-boundary check reinstates no encoded-output compatibility requirement.

## Verification

The combined66/67 packet observed 673 named outcomes with 389 PASS, 272 FAIL and 12 SKIP, plus aggregate lint RED50. Its two selected failures precede retention assertions at unsupported-host preparation. Those observations establish historical preparation RED and predict no successor pass count.

Before either insertion, retain the actual unchanged selected tests' preparation-stage RED at the worker's bound candidate. After attachment, verify both original tests and their unchanged success/capacity/disk/journal assertions. No additive fixture is required. Retain any newly reached original assertion or validation failure and return it to root for a separately bounded correction.

Keep the existing preparation forwarding, operation-error, original-stage, default/bootstrap and isolated controls, public profile guard, success-count exhaustion and missing-transcript controls in the focused gate. Preserve their source. Use pinned tools, `test_dep`, count1, externally bounded commands and one compact receipt binding source/tool/environment identities, argv, numeric exit, elapsed time and immutable raw-log hashes.

Retain formatting, whole-file reconstruction, affected vet/errortype, configured unfiltered Runner lint and required repository fast lint. Both supported source-set static checks, generated validation and other retained source gates need current evidence or exact unaffected-input reconciliation. Root serializes every shared Go/build/lint/generator lane; worker68 currently holds that lane and this draft grants none.

Root owns one future frozen ordinary comparison against combined66/67. Permit outcome changes only for actual emitted names in the root-selected fixture scopes, including actual table children from sibling owners. Enumerate names from terminal events; synthesize no parent/intermediate slash names and forecast no PASS total. Require the complete current 50 lint blocks unchanged, with zero introduced or removed findings. Keep aggregate RED and errortype reachability explicit. Fresh independent integrated review and all remaining owned source gates govern completion.

## Exclusions

Exclude every other consumer, helper, import, comment, assertion, fixture datum, target metadata, production/storage owner, public/default path, bootstrap decoder, resume/replay/minimize, guidance, adapter/installation identity, schema/generated input and runtime/toolchain change. Task68 and sibling scopes retain their own workers. Preserve dirty owner spec, MILESTONES.md including fn155, and unrelated user files. Native fn128/fn149 remain deferred and unverified; no native, CI, PR or push authority follows.
