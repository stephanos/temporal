# Ownership of the ten analysis refusals

All ten observations in `qualification-ordinary.log` are actual failures. Eight stop in existing report construction before their report assertions; two stop in the inspection operation owned by fn109.8. None exercises task 7's `preparation.Prepare` operation. The retained fn109.1-6 acceptance does not supply named passing coverage for these ten tests. This separates task ownership without relabelling any failure, waiving a portable assertion, or issuing an acceptance verdict.

Research used read-only source, Git, task and evidence inspection at `b4602685b3184387cf2d713178095247f0c11d8f`. No test, Go, lint, generator, Flow, staging, or source mutation ran. This document is the only file written by this follow-up.

## Per-case attribution

Paths below are relative to `tools/gomad3/qualification/analysis`. All first eight tests are whole-file unchanged from `a3b9f80efab9356c0be2080779133337e2471ac0`. The two prepared-review tests also match that anchor and the original task-8 preimage manifest.

| Failing test and assertion location | Intended assertions prevented by the observed error | Retained ownership |
| --- | --- | --- |
| `TestAnalyzeReviewsTheTargetBeforeBuildingTheClaim`, `analysis_test.go:31` | Supported classification and supplied BuildKey after review. The injected review callback's target-equality assertion executes before the failure. | Existing analysis contract; fn109.8 report-preservation surface. No task-7 owner call; no fn109.1-6 pass reuse. |
| `TestBuildReportsLexicographicallyFirstShortestDependencyPath`, `analysis_test.go:55` | Unsupported classification, canonical shortest path through `a/middle`, root list, closure digest and path redaction. | Existing analysis contract preserved by fn109.8, not task-7 preparation. |
| `TestBuildRedactsPathBearingArgumentsDeterministically`, `analysis_test.go:86` | Redacted positional/flag paths, unchanged test selector, canonical output and exact path digest. | Existing fn109.8 analysis-output preservation surface. |
| `TestBuildUsesAllSortedRootsAndRejectsUnreachableFindings`, `analysis_test.go:115` | The required `not reachable` rejection. The test sees the earlier host refusal instead. | Existing fn109.8 analysis-error preservation surface. |
| `TestBuildSupportedReportHasCanonicalNonNullEvidence`, `analysis_test.go:128` | Supported result and non-null empty blockers, packs and requirements in the value and canonical bytes. | Existing fn109.8 fixed-report preservation surface. |
| `TestBuildLinkedReportSeparatesEliminatedClosureBlockers`, `analysis_test.go:165` | Linked mode, manifest presence, supported classification, zero active blockers and one eliminated blocker. | Existing fn109.8 linked-mode/report contract. This fixture does not build or launch a target. |
| `TestBuildGuardedReportSeparatesNonblockingGuardEvidence`, `analysis_test.go:194` | Supported classification, one guarded blocker, no active/eliminated blockers. | Existing fn109.8 guarded-policy/report preservation surface. This fixture does not execute a guarded target. |
| `TestDecodeValidatesCanonicalCapabilityReport`, `analysis_test.go:209` | Round-trip equality, superseded-schema refusal and unsupported-without-blockers refusal. Build fails before Decode is called. | Existing fn109.8 report/error preservation surface. |
| `TestPrepareCapabilityReviewOwnsPrivateAdapterLifecycle`, `prepared_review_test.go:29` | Review schema, nonempty private root, successful Close and removed root. | Direct fn109.8 ACs 1-2 and Required investigation file. The preparation owner here is Inspect, not task-7 Prepare. |
| `TestPrepareCapabilityReviewSupportsTemporalBackoffWithGRPCAdapter`, `prepared_review_test.go:57` | Zero findings and pack allowances, exact gRPC selection, no x/sys/unix package, rewritten internal package provenance and no syscall/xsys imports. | Direct fn109.8 compatibility inspection and adapter-preservation scope. The real root-module closure is not reached. |

The first eight are inherited analysis tests, not tests introduced by fn109.8. Their follow-on ownership comes from task 8's explicit requirement to preserve analysis text/JSON bytes, mode semantics and classifications while migrating analysis and compatibility review. Their inheritance does not turn them into accepted fn109.1-6 coverage. The two prepared-review tests are expressly listed by task 8's Required investigation targets. Task 7 describes itself as the first half of R4 and explicitly leaves analysis/compatibility inspection to task 8.

## Exact paths to the failures

For the first eight, `analysis.go:116` calls `input.IOProfile.Requirements`. `deterministicio/requirements.go:17` calls `profile.validated`, which rejects actual linux/arm64. The error is wrapped as `project deterministic I/O requirements` at `analysis.go:118`. `shortestPaths`, blocker projection, classification, report construction and later Decode assertions have not run. `analyzeWith` calls the injected review and then Build; it does not call the preparation owner. `analysis.go` imports no preparation package.

For the last two, `PrepareCapabilityReview` calls `preparation.Inspect`; `inspection.go` creates its private root and calls `Default().PrepareTargetBuildAdapters`. That adapter operation begins with the same platform check. Inspect's failure cleanup runs, but its successful ReviewCapabilities result, returned handle and Close-after-success path do not. The generic cleanup on failure cannot count as the lifecycle test's unexecuted successful-close assertions.

Neither path calls `preparation.Prepare`. Inspect shares the private `stageError` type with Prepare, but task 8 introduced its own `StageReview` and inspection lifecycle. Task 8's exact patch changes `preparation.go` only to add that stage; it creates `inspection.go` separately. Its `stageError.Error` and `Unwrap` bodies remain byte-identical at the anchor and current source. A package-level name or shared error type is not evidence that these failures exercise task 7's sequence.

## Source bindings

| Source | Current SHA-256 and binding |
| --- | --- |
| `qualification/analysis/analysis.go` | `747f6be1d998a7831680c6d5f8f9f6ab5765f9693115f8ccc23886477a079852`; exact original task-8 preimage and anchor file. |
| `qualification/analysis/analysis_test.go` | `6a400763cf195ffbd6684ceb71992b5e4040aff6ab1ee5437399376e3eecb6ea`; exact anchor file. Last modifying commit `53c5cebc9076bdbb439a36b7fa78cc02bc051ab9`. |
| `qualification/analysis/prepared_review_test.go` | `b3ca39aabc7e409a3fb6be4e6afbbfb8619610ea37015c5822b13570ce25a3ce`; exact task-8 preimage and anchor file. Last modifying commit `c732d004e3228127e9df48d0b77d68a56b979b5c`. |
| `internal/preparation/inspection.go` | `8c37578631b251b8fae704acb17d91029660af5ee3d23e3bd87bad29b0939556`; exact task-8 `source-freeze.json` and anchor file. |
| `qualification/analysis/prepared_review.go` | Current whole-file `61cc3485434bdb022f9e9582730d3bc5c074662d9280eeb79d74cb9777d78992`. Later additions preclude whole-file equivalence; the relevant PrepareCapabilityReview and Close functions remain exact. |

Complete named function bytes were compared against the anchor. SHA-256 values are `Build=18c7da6fb078dabc5dfd7b1ecba345730319995e83fddbfb2db4ac17090846f3`, `analyzeWith=05a2ae0141b6ab0c7a755494ba13c7401e189b91f02915dc09c67990bd61688f`, `Decode=ff030ccf4bef61216291492be18699170ea45b0a30ed08b307205a0619b1a983`, `PrepareCapabilityReview=ec9c3eb1c12a462635e516561548eb0db590c1e74e660c77f4e40a5507085a1a`, `Close=b81b916042f6f3418fbf9347abd99d7ea0763aa5d8814a7ea9a357f12fdba79e`, and `profile.validated=6df480be19d55263d73ea270ed36c18b1ef38e73b074a30d314ca51ba0dbc5dd`. The entire shortest-path, blocker-projection and report-validation functions also compare equal. These are static preservation bindings, not passing execution observations.

There is a later dependency change. Accepted fn113.2 commit `8f292989713ff55b97016e5156eb92667b29102d` extracted `projectAdapterRequirements` and the prepared-target shape helper. The public Requirements wrapper still checks the unchanged platform guard first. Its accepted `TestPortableRequirementsProjection` covers requirement ordering, package membership and non-null empties below that wrapper; `TestPortablePreparedTargetShape` covers a different target-validation operation. Those assertions do not cover analysis shortest paths, redaction, linked/guarded blocker separation, Decode, Inspect or the actual Backoff closure. See fn113.2 `conductor-source-acceptance-20261008/{final-evidence.json,final-handover.md}` for that exact bounded dependency coverage. Its handover expressly excludes qualified public profile wrappers.

## Accepted predecessor evidence and limits

The canonical fn109.4 and fn109.5 conductor completion evidence invokes `go test ... ./qualification`, with no `/...`. Their 23 qualification passes cover the parent package, not `qualification/analysis`. The accepted CLI `TestRunAnalyzeForwardsTargetAndClassifiesReport` injects report-building functions that return fabricated Reports; it covers request forwarding, statuses, writers and cleanup routing, not production Build. `TestAnalyzeClassifiesLinkedCapabilityCapacityAsUnsupported` calls the error reporter directly. Reuse only those named CLI assertions if their dependency bindings still hold.

Fn109.2 accepted portable selections concern options, transport, CLI, plan/resume and mounts. Fn109.3 concerns atomic completion and its consumers. Fn109.6's accepted 81 names concern operation-independent controls, external/API checks, process controls and minimizer/planning. A search of all JSON, Markdown, stdout and log files under current worker/conductor source-acceptance directories found none of these ten test names across tasks 2-6 (72, 145, 111, 146 and 177 files respectively). That scoped negative result agrees with the exact conductor command selections; it is not a claim that no historical run ever executed them.

Fn109.1's receipt names Runner/coordinator tests and a historical Darwin full-host gate, with no current assertion/dependency receipt for these ten. Original task 8 has historical Darwin package/full-host passes and fixed-output receipts, but task 8 remains the follow-on source-acceptance owner; those historical logs are not a current task-7 or accepted-predecessor pass. Task 7's historical broad baseline and original host-wrapper limitation likewise supply no present pass.

## Remaining task-7 scope

These ten failures supply no demonstrated task-7 implementation regression or direct task-7 assertion gap. Task 7 still owns the successful/custom/independent preparation values and record projections, actual Explore/portable-plan behavior, cleanup/error precedence, bad sums/replacement conflicts, changed binaries, cancellation/timeouts, and its architecture boundary. Any unexecuted portable assertion in those own controls remains a task-7 gap until the conductor establishes matching source-bound coverage. A passing report helper, accepted predecessor task, or native transfer cannot fill that gap.

Keep the complete `qualification-ordinary.log` red and list this ten-case attribution alongside it. Carry the eight report assertions and two inspection/compatibility controls into task 8's retained source acceptance rather than marking them passed, removing them, or describing all ten as deferred native tests. The first eight are synthetic report tests; their host coupling is especially insufficient grounds for a native-only classification. If task 8 needs a portable test seam, its owner must establish the unchanged public guard, the exact exercised production report functions, and named assertion coverage without rewriting these original assertions into expected failures.

Primary evidence pointers are `task-7/source-acceptance-20261008/qualification-ordinary.log`, `.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.{7,8}.md`, `task-8/{preimages.json,source-freeze.json,task-only.patch}`, and task 4/5/6 `conductor-source-acceptance-20261008/completion-evidence.json` plus their completion summaries. All task-relative artifact pointers use `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/` as their base.
