# Task 39 independent source review

Assessment: `SOURCE_PROGRESS_COMMIT_ONLY`.

No actionable introduced defect was found. Critical, important and minor source
findings are all empty. Root may commit this bounded source progress before
admitting another writer. Task completion and formal qualification remain open.

This fresh review covers fn-109.39 on branch `gomad`, with BASE and unchanged HEAD
`5c5c025adeabacd9f3217de2b9ab8f7171b0c33c`. Requested reviewer model is
`gpt-6.1-sol` at `high`; reviewer and writer are from the same Codex family.
Executed-model metadata was not exposed, so the requested identifier is not a
runtime attestation. Review conduct was bounded by the dispatch; the host did
not enforce read-only access. The reviewer changed only this report, ran the
allowed portable checks sequentially, and made no source, lifecycle, index,
history, configuration, bridge or subagent changes.

## Source findings

The BASE diff changes three existing files and adds `target/cleanup_test.go`.
Existing comments, mutation-fixture assertions and operation statements remain
intact outside the admitted named results and cleanup checks. Naming return
values does not change a Go function type, including the public
`AdapterPreparedSourceSetSHA256` type. No public seam, dependency, compatibility
grant, pin or production-policy change appears in the diff.

| Owner and final location | Inspection result |
| --- | --- |
| `adapter_source_set.go:31-45` | One deferred RemoveAll after successful MkdirTemp, at the original function-return boundary. Non-nil cleanup clears digest; nil cleanup leaves the primary object untouched. Sole cleanup is returned directly; both errors join primary first. Listing, projection and source selection retain their order. |
| `target.go:882-912` | One deferred hash-reader Close after successful OpenPath. Non-nil Close clears digest and size; conditional direct/join composition preserves primary identity and unwrap shape when Close succeeds. Validation and hashing retain their order. |
| `target.go:914-953` | One deferred input Close. Each output branch attempts Close once before that defer runs, including Chmod/copy failures and the existing final explicit Close. Early output cleanup joins after the original wrapped primary only when cleanup fails. The existing `close prepared exec target: %w` branch remains intact. |
| `target.go:956-979` | The three early Chmod/Write/Sync branches check their original immediate Close once, return the exact primary object when Close succeeds, and join primary first otherwise. The successful final Close remains unchanged. |
| `target_test.go:517-524` | The failed mutation Write still reaches its original `t.Fatal(err)`. A Close failure is reported first with nonfatal `t.Error(closeErr)`; each path closes once. |

These are all nine mapped formerly unchecked returns, comprising three deferred
production releases, five early output releases and one fixture release.
`hostfs.OpenPath` retains ownership of its own failed-open cleanup; the new
defers register only after a successful open transfers the file. Copy releases
destination before source, never retries a Close, and introduces no deferred
second output Close. Original executable validation, exclusive creation,
copy/Write/Sync order, permissions and retained partial-destination behavior
remain intact. No branch removes a destination after failure.

The checked cleanup failures deliberately expose an additional failure surface.
This inspection establishes the code's composition and lifetime logic; it does
not establish universal byte equivalence or actual cleanup-fault execution.

## Verification and evidence

The reviewer read applicable AGENTS guidance, the complete Gomad README and
milestones, flowctl usage, parent fn-109 and task 39 through flowctl, the actual
source diff, all raw logs/receipts, and the completed [handover](handover.md) and
[evidence](evidence.json). The twelve receipt log hashes match their raw files.
BASE lint reports exactly nine errcheck findings; final actual unfiltered target
lint reports zero, with nine resolved and zero introduced. Configuration and
tool hashes match the receipts.

The initial directory-copy characterization incorrectly expected a direct
error and failed on BASE's existing `*fmt.wrapError`. The corrected assertion
requires that wrapper and its plain cause. This is a test-authoring correction,
not a production regression reproduction. Corrected BASE and final logs each
pass 14 selected top-level tests, including four cleanup tests with twelve
subcases. The final test file combined with BASE's 1,039-entry manifest hashes
to `306ce57f417e2ef3e63dc81ee1cf2dc34a2215738437bbeacee321a7ef1f7018`,
matching the corrected BASE receipt.

The reviewer independently hashed literal `#!/bin/sh\nexit 0\n` and obtained
17 bytes and SHA-256
`306c6ca7407560340797866e077e053627ad409277d1b9da58106fce4cf717cb`.
Real-file controls retain contents, source 0751/copied 0700 modes, private-write
0400/0700 modes, empty/nonempty bytes, missing/symlink/nonexecutable/directory
rejection before destination mutation, and collision/missing-parent errors.
The three assertion helpers check concrete plain/wrapped errors and exact
PathError message, operation, path, errno and unwrap shape. Separate operation
calls do not prove cross-call pointer identity of primary errors.

Fresh reviewer commands ran from `tools/gomad3`, using cached stock Go1.27.1
linux/arm64 first on PATH, `GOENV=off GOWORK=off GOTOOLCHAIN=local GOPROXY=off
GOSUMDB=off GOFLAGS=`, with all three dispatch-named Gomad seed variables unset.

| Command | UTC window on 2026-10-05 | Terminal result |
| --- | --- | --- |
| `go test -v -count=1 -tags test_dep ./target -run 'TestTargetFileCleanup\|TestPreparedCacheDigest\|TestCapabilityReviewGoldenCanonicalBytes\|TestCompatibilityPackProjectionPreserves'` | 09:08:08.401-09:08:08.749 | Exit 0; 14 top-level tests pass, target package 0.048 seconds |
| `go test -v -count=1 -tags test_dep . -run '^TestPackageArchitecture$'` | 09:08:08.907-09:08:09.940 | Exit 0; one test passes, package 0.751 seconds |
| `/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 run --config=../../.github/.golangci.yml --build-tags=test_dep --timeout=10m --fix=false ./target` | 09:08:10.098-09:08:10.432 | Exit 0; `0 issues.` |
| `go vet -tags test_dep -vettool=/tmp/fn109-lint-tools.ZdNe1t50/errortype ./target` | 09:08:10.587-09:08:10.825 | Exit 0; no diagnostics |

Reviewer `gofmt -l` on all four Touches and `git diff --check` also returned
empty output and exit 0. Each command above had successful checks of all 1,040
manifest entries immediately before and after execution. All reviewer commands
returned terminally. Their output is retained in the review session transcript;
the worker's exact command/log/environment bindings remain in the linked
receipts instead of duplicate review logs.

The reviewer verified BASE's three changed-file hashes through `git show` and
verified its retained task 38 manifest against the committed copy. Both manifest
copies hash to
`a5eb12d3228b6fe5b829532d784229a2c8bdcfe7df61d53f81cba5646dbec7af`.
The [final source manifest](final-source-manifest.log) hashes to
`f49d379db52a343ea61d5a0824a463dd66f0ff5f014ea774a8bc08099be52520`.
Independent comparison confirms exactly three existing changed inputs, one new
test, 1,036 unchanged inputs and zero missing inputs. This closure includes
module, runtime, profile, pin, generator and lint inputs, rather than a claim
about whole-repository identity. The Makefile generator input groups and livecap
protocol input list exclude these four Touches; unchanged generator inputs
support the handover's generator-validation applicability decision.

## Required proof still open

Genuine first-Close failures, simultaneous primary/cleanup failures, multiple
cleanup failures, post-open Chmod/Write/Sync failures and RemoveAll failures
remain unexecuted. Digest/size clearing, exact primary-object retention and
once-only release on those fault paths have source-inspection support only.
Ordinary success, second Close, descriptor theft, arbitrary callback injection
and source-text assertions supply no substitute fault proof. No new fault seam
was admitted or added.

The [adapter-pin command](final-adapter.log) exits 0 with SKIP at
`adapter_regenerate_test.go:41` because `.toolchain/bin/go` is missing. It supplies
zero native or adapter-pin qualification proof. The reviewer retained this
unchanged unavailable-input result without rerunning it. Fn-113 keeps adapter
regeneration approval, source-pin, publication and qualification ownership.

Original R18/R19, task 21 and predecessor acceptance, matched first-baseline
identities, full/default/functional/affected-consumer/formal/native Darwin
requirements remain open wherever unproved. These portable development checks
establish only their listed source scope, not formal SHIP, task completion,
merge readiness, whole-Gomad lint or native qualification. Transferred native
Linux execution remains nonblocking under fn-128. The adapter source-set direct
exec transport gap belongs to task 9/R10 and remains open without expanding
task 39's cleanup scope. Root retains commit and lifecycle ownership.
