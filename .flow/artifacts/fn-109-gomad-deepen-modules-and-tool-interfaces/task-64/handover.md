# Task 64 source-progress handover

The fixture now exits 3 immediately when its existing readiness write fails. The two real-child regressions pass, and all ten original parent cases pass unchanged. Required package lint remains red, so task 64 remains in_progress with an uncommitted candidate for conductor review.

Base is e1f40c2fdc75b3838e25aaa03112e1f88089725d. Workspace is /Users/stephan/Workspace/skunkworks/.gomad-runner-preparation-research.ETZ8a4ZF/watchdog. No commit, Flow lifecycle mutation or review verdict was issued.

## Verification

| Receipt prefix | Exit | Seconds | Observation |
| --- | --- | --- | --- |
| baseline-focused | 0 | 7 | Ten original parent cases pass |
| baseline-lint | 2 | 5 | 11 unfiltered package findings |
| baseline-vet / baseline-errortype | 0 / 0 | 1 / 0 | Affected stock-host analyzers pass |
| baseline-format | 0 | 0 | No gofmt differences |
| baseline-fast | 2 | 1 | Missing sparse input tests/mixedbrain/go.mod; proto/internal and chasm/lib also absent |
| regression-red | 1 | 12 | Both children reach the terminal frame then hit their five-second deadlines |
| final-focused | 0 | 7 | Both new child modes and ten unchanged parent cases pass |
| final-lint | 2 | 2 | Ten byte-identical inherited findings remain |
| final-vet / final-errortype | 0 / 0 | 1 / 1 | Affected stock-host analyzers pass |
| final-format | 0 | 0 | Both admitted files have no gofmt differences |
| final-fast | 0 | 20 | Task-base diff lint and its errortype stage pass after authorized sparse materialization |

Each prefix names its raw .log, exact -command.txt, numeric -result.txt and pre/post source .sha256 manifests in this directory. Every captured source comparison returned 0. The preserved wrappers are run-gate.sh and baseline-focused-wrapper.sh, matching their recorded hashes. The first wrapper emitted missing-path errors outside the suite log and hashed only materialized files; its original receipt is limited accordingly. Later manifests explicitly record missing tracked inputs. Root authorized adding only tests/mixedbrain, proto/internal and chasm/lib before final-fast; those source bytes were not edited.

affected-lint-comparison.json compares complete header/source/caret blocks. Exactly watchdog_io_test.go:114 errcheck disappears, zero findings appear, and all ten other blocks match byte-for-byte. Final-fast filters 52 pre-existing findings to zero reports; its pass establishes no aggregate lint pass. The full original-base comparison is pending the conductor's frozen joined task 64/65 batch, using [task 63's retained RED53](../task-63/final-integrated-lint.log). Integrated errortype remains unobserved here. Required unfiltered affected lint remains red.

## Preservation and limits

The additive test observes actual parent-side EBADF, an unchanged stdout input, empty stderr, the exact valid terminal frame and prompt child exit 3 in both modes. On the old helper, both modes pass the frame/input checks before failing at the deadline. The additive file has identical SHA256 before and after the fix. The helper's only diff wraps fmt.Fprintln in its error check. Its first 113 lines and tail starting at old line 115 match the candidate's first 113 lines and tail starting at line 117, preserving all original assertions, terminal write/close, SIGTERM ignore placement and healthy wait behavior. preservation.log records both cmp exits 0 and tracked diff-check exit 0. The new-file no-index check returned 1 for the added-file diff and emitted no whitespace diagnostics.

Candidate SHA256 values are 0f625ab98f20a996779337dccfcae87ead30b4425943f61ca89d702c1c520a0f for watchdog_io_test.go and 6737e39304a5020c1d56a05d43451d9bf5395699ab1ad0b9bf2c84966935909c for watchdog_fixture_output_test.go. Receipt command files bind actual Go/gofmt/lint/errortype hashes, pinned Linux ARM64 Go1.27.1, file proxy, reused caches and cleared ambient Go/runtime seeds. This stock-host fixture proof supplies no supported-native qualification; fn128/fn149 remain deferred.

Defect route:
- prior fixes: local history retains df2642da26, with no competing watchdog branch observed; GitHub PR/tracker lookup unchecked after HTTP401; memory unavailable because it is uninitialized.
- diagnosis: valid terminal delivery rules out descriptor setup failure; actual EBADF and unchanged input precede both old-child deadline failures, confirming the unchecked readiness write reaches the wait loop.
- introduced by: bisect skipped because no known-good revision was provided.
- base: regression-red fails both modes on unchanged base helper; head: final-focused passes both modes on the uncommitted candidate.
- live: no live application surface; reproduction commit deferred because root explicitly reserves all commits.

All owned handles are terminal. Shared Go/build/lint lane released to root. The conductor owns independent review, the bounded progress checkpoint, joined aggregate verification and lifecycle.

stage: impl-review - skipped(policy: host-deferred - conductor owns the gate)
Tier: session (jev-unavailable(no_key)); explicitAGENTSimplementermodelretained
