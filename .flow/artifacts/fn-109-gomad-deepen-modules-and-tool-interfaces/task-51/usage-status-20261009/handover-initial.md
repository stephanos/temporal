# Usage-status fixture source progress

Task51 reconciles task46's compatibility-pack invalid-input expected status with the preserved failed-usage status 1. The sole existing-file change inserts literal invalidStatus 2, specializes it to 1 for compatibility-pack, and uses it only in the existing invalid row. The new public fixture covers missing and unknown subcommands, healthy status 2, actual read-only-file EBADF status 1, independent literal usage bytes, one stderr call, zero stdout calls and unchanged sentinel publication.

Task remains in_progress on gomad at base/HEAD `e09187751326abf393011052dd08fdfc9af61900`. Root owns review, lifecycle and commits. Worker staged/committed nothing and dispatched no reviewer. Tier: session (jev-unavailable(no_key)); project implementer requested gpt-6.1-sol/high; actual execution telemetry unobserved.

## Authority and identities

[Admission](admission.md) is the exact preservation exception. [Task8 admission](../../task-8/conductor-source-acceptance-20261008/admission.md), its immutable preimages and the [status diagnosis](../../../../../../tmp/maintainer-status-diagnosis-20261009.md) retain the original a3b9 command authority. Task46's historical status-2 pass and task50's exact96b/current RED receipts stay unchanged in their original packets. These historical passes supply no current-source proof. [Task50 root verification](../../task-50/generator-stderr-20261009/root-verification.md) retains predecessor source/tool/review and proxy-checksum bindings.

Final source inventory has 1,109 paths, SHA256 `c059aae0da0c99ec252b72d128c1a8f1233a517a0749767dcea30e2259bb49a8`, derived from task50's unchanged manifest `d11d5b9700d75b6cbd0d02d1d44bbaedd832fd1e34f2f02bc534af0abf31204c` plus exactly two overrides.

| Path | Final SHA256 |
| --- | --- |
| maintainer_output_test.go | `159277febf4bad18ca83d37bfea71c1413aaa0293bc85fd4d8bbd78036ca17a3` |
| usage_status_preservation_test.go | `1c9a2c73251aab05a60ed896eac1d52002ec96a6577d0ab67edbc9330fa320ed` |

Each raw receipt binds numeric exit, elapsed time, exact command, environment hash, source basis/overrides, tool hashes, stable control inputs and both stream hashes. All commands used pinned stock Go1.27.1 on developmental linux/arm64, test_dep and count1 for tests. Lint2.13.0 and all 26 predecessor tool/proxy inputs retain manifest `8f2aef1b9b27c90165396c6a2f15e92947df84bf05f853aec877a2ffce3a266a`. Existing checksummed file:// Sprig metadata/archive inputs supplied private fixture caches; no network fetch, dependency/pin edit or new tool followed.

## Retained observations

| Receipt | Actual exit and coverage |
| --- | --- |
| fixture-red | 1; exact existing compatibility invalid case reports primary status 1, want 2 |
| ordinary-package-baseline | 1; 186 named passes, two failed records solely from that stale case/family; zero skips |
| preservation-before / preservation-after | 0 / 0; 12 named passes each, permanent and public controls; zero failures/skips |
| fixture-green | 0; same exact command, 26 named passes; corrected existing case passes |
| task46-task50-final | 0; 76 named passes, eight top-level tests; zero failures/skips |
| ordinary-package-final | 1; 193 named passes, two failed records, 46 top-level passes and one failing family; zero skips |
| vet-final / errortype-final | 0 / 0; affected package vet and standalone errortype -test=true |
| architecture-final | 0; seven top-level architecture/purity/edge/public-boundary/HostPackageVet tests, ten named passes |
| source-darwin-final / source-linux-final | 0 / 0; each selects 12 production and 18 test files, including both task51 fixtures; no metadata errors |
| validate-final / format-final | 0 / 0; check-only version/wire/boundary/compiler/pack/manifest validation, formatting and diff checks |
| fast-final | 0; actual fixes-disabled Make selects 55 host packages, admission-base filter emits zero issues and reaches errortype |
| scoped-final | 1; 63 unchanged unfiltered errcheck diagnostics |
| integrated-final | 2; 208 unchanged configured original-base diagnostics; integrated errortype unreached |
| source-proof | 0; exact admitted edit, 1,036 unaffected original Gomad files, complete receipt/control/raw audit and generator preservation |

The baseline fast command returned 0 after selecting zero changed Go packages. It supplies no baseline coverage pass. Scoped63 and integrated208 remain RED. [Source proof](source-proof.stdout) compares every residual diagnostic's complete message and source-line bytes against the actual baseline and task50 successor, with zero introduced/resolved/changed records. Original lint base stays `951c5516e9e7b3066e7e069adda9565cfd68844c`; historical213 and narrower d635-filter22 retain their original scope.

The new controls already passed before the fixture edit and remain preservation evidence. [Mutation inputs](mutant-inputs.json) bind exactly the two failed-usage returns at production lines25/44, original/mutant/overlay hashes and scratch paths. mutant-permanent and mutant-public each return1 for the intended actual2/want1 failures. The permanent closed-writer and new EBADF terminal leaves both reject the mutant. Production never changed. The outer mutation-proof checker returned1 because its leaf filter incorrectly included intermediate public subtest ancestors. Its raw output and original mutation.mjs remain immutable. source-proof corrects only the audit of those existing terminal observations; no suite was repeated.

The final full-package failure is `TestRunCompatibilityPackRefreshStopsAtApprovalAndResumes/unselected_output`, refresh_test.go187. Git Trace2 records init0 followed by add128 at the same absolute -C directory, with repository recognition absent in add. The baseline counterpart passed. Both complete trace streams and hashes are retained. The cause remains unproved; no discovery flag, helper, assertion or environment-policy correction followed, and no blind retry ran. [Git research](../../../../../../tmp/git-fixture-diagnosis-20261009.md) records the recognition boundary and missing filesystem-signature observations. This independent failure keeps ordinary full-package acceptance open.

source-proof verifies all 160 generator inputs and 53 generated outputs against task50's final source hashes using its retained inventory. All compatibility production, permanent task8 fixture, other fixture assertions, comments, pins and generated bytes stay unchanged. Both user-owned Turbo files retain their dispatch hashes. The source packet uses predecessor references and two source overrides instead of duplicating historical inventories.

## Remaining authority and acceptance

Root's fresh independent integrated source/evidence review is pending. Full ordinary package and original-base lint acceptance remain red. Original first-baseline/fixed-identity, predecessor, preservation, full/default/affected-consumer/functional/formal source obligations remain with their owners wherever unproved. Static source selection and stock source tests supply no native execution qualification. Native fn128/fn149 remain deferred/unverified. No task completion, native revival, PR, push, CI or history rewrite occurred.

Defect route records existing task8 restoration and task46/task50 source history as prior work. The actual pre-edit fixture reproduces the status conflict; the admitted single expected datum corrects it while production remains preserved. Bisection and a reproduction commit did not run because the explicit root-owned admission already binds the immutable source chronology and prohibits worker commits. No live-app surface applies to these developer CLI usage branches.

stage: impl-review - skipped(policy: root owns review; required ordinary package and original-base lint remain red)
stage: completion - skipped(policy: root owns lifecycle; acceptance remains open)
