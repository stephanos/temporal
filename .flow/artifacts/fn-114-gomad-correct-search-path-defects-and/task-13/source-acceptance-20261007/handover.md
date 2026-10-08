# fn-114.13 retained R9 source acceptance

The current local-queue rule, primary fixtures and preceding-controller refusal have retained source proof and fresh applicable source checks. This run changed no product source. The conductor owns review, Git and Flow completion; task .13 remains `in_progress`.

Tier: explicit implementer gpt-6.1-sol at high, retained by conductor after judge unavailable(no_key). Execution metadata did not expose the actual model.

## Current source and provenance

The frozen base is `59f104491dcb14aa5c6337d995ff60c9f6956977`. [source-proof.json](source-proof.json) verifies all 87 fn-110.2 preservation inputs against their recorded hashes. The complete Gomad module also matches that receipt's source base `1deced3efa4e7000163cb269e70e35f0c6b7dbd7`; the current worktree comparison returns no changed module paths. It verifies all 72 retained raw files, 521,317 original bytes, and selected receipt/output digests before reusing narrow static results.

The actual first committed R9 implementation is `a3b9f80efab9356c0be2080779133337e2471ac0`. Its complete `gomadChoiceRunqIndex` body equals today's body after exactly one `gomadIdentity` to `gomadID` substitution. The body also matches compact base `1b0bc277589d141aca8b534b03135ab3e57fc050` under that substitution. The primary matcher, test, runtime-symbol fixture and previous-controller tape match both the historical snapshot hashes and first R9 commit byte-for-byte. Current primary/secondary fixture context, `runtime_scheduling.go` and SPEC equal the compact base in full.

The historical `integrated-source-hashes.json` names `d635e23f00d926a43b942f25a9d05bd0ccb72025`, whose committed runtime still has the pre-R9 unfiltered selector. Its dirty-snapshot file hashes, rather than that committed HEAD alone, bind the historical run. This handover preserves that distinction.

The completed fn-110.2 source chain verifies whole-source alpha/gofmt equivalence across 20 materialized patch files plus the runtime overlay, 30 field sites, and unchanged 20/79 allowlists. Exact current inputs permit reuse of its real pinned-archive source inventories for darwin/arm64 and linux/amd64, zero-fuzz equivalence, collision checks, and identity calculation. Inventories contain 273 draw, 86 seeded, 48 clock and 10 goroutine rows. The descriptor archive remains SHA256 `4e408abae126d916b6164627193f2c54f0e3ca1312d693b86db45f862ab238b1`. This run downloaded and built no toolchain.

## Gates and acceptance boundaries

[evidence.json](evidence.json) records exact argv, environment, cwd, timestamps, exits, tools, source identities and lossless base64 output containers. Current checks passed with zero source edits:

- Check-only `make -C tools/gomad3 validate`, including generated protocols, patch/script ownership, pack-profile test and qualification-manifest check.
- Portable choice/replay/trace/prefix and conformance controls, 50 top-level PASS, zero FAIL/SKIP. `TestRuntimeOwnedRejectsPreviousController` validates the old tape, then rejects today's implementation with the recorded target, Darwin platform and build key held fixed.
- Architecture, public signatures, purity and complete host vet, four top-level PASS, zero FAIL/SKIP. Vet explicitly covers 55 packages per supported source set, plus the stock host source set.
- Configured `make lint-code` with `--new-from-rev=d635e23...`, `--fix=false`, packages `./internal/gomadtool/conformance ./choice/internal/wire`, followed by configured errortype vet. The diff filter excludes untouched inherited findings. This is scoped standards success, with no unfiltered or whole-project lint claim. fn-109.21 owns aggregate lint; the original 265-finding receipt is historical before the two admitted panic exceptions, and this run predicts no new count.
- Pinned gofmt checks of the primary matcher/tests, scheduling fixture code and both fixture programs emit no filenames. The runtime overlay's retained normalized equality is covered by the exact preservation chain.

Baseline is green for these applicable source checks. Evidence-only additions required no duplicate post-edit suite. The first capture helper failed on a tracked gitlink directory before launching validation; adding the file-kind check corrected only the evidence helper, and validation ran once.

Historical same-source control data remains unchanged and hash-verified. Across 32 seeds it records 2,261 to 2,080 decisions and 31 to zero runtime selections, with its original source hash and build keys in [source-proof.json](source-proof.json). No current native measurement is inferred. Zero/one-user deterministic picks, all-user alternative sets, runtime/user mutual progress and exact replay/prefix assertions remain in the primary fixture. Their native execution, cause/count measurements, runtime/overlay, full host and qualification commands stay deferred under fn-149.1/.2 and fn-128.1/.4/.7.

## Required conductor review context

The compact committed diff alone does not establish R9. Independently inspect the complete current `gomadChoiceRunqIndex` at `gomad.go:1194`, `gomadChoiceDecision:833`, `gomadChoiceRunqSeeded:767`, and the full `runqget` patch path at `go1.27.1.patch:385`. Inspect the head-class dispatch, capacity/identity checks, static buffers, user-only alternatives, no-decision path, queue advancement and recording/replay/prefix behavior.

Inspect primary `runtime_owned.go:16`, its zero/one/two/busy modes, every alternative-set digest and mutually gated progress, its fixture's no-argument runtime symbol, old-controller test/tape, and canonical `[RUNTIME.SCHEDULING]` at `SPEC.md:230`. Inspect source-derived controller identity and the exact compact preservation chain, rather than granting approval from an empty evidence-only range.

The secondary `runq_user_choice` fixture participates through `requireSchedulingBehavior` to `requireSearchReproduction` to `requireRunqueueUserChoice` (`runtime_scheduling.go:21/277/380`), with `requireUserAlternatives:473`. Its finalizer callback counts as a user, consistent with the contract; the primary runtime-symbol busy fixture supplies the distinct runtime-owned progress assertion. The secondary header at `testdata/runq_user_choice/main.go:4` says "run first by a fixed rule", a pre-existing annotation inconsistent with canonical head-class wording. It predates this acceptance and equals the compact-base bytes. It remains untouched under AGENTS' separate-comment rule and is explicitly passed to review.

Defect route:
- prior fixes: retained integrated R9 implementation and completed fn-110.2 source chain reused; external PR/tracker search not done because no new fix was proposed and the conductor owns research.
- diagnosis: historical explicit-two-user attribution and red fixture retained; native re-execution transferred.
- introduced by: bisect not done because this run reconciles an existing implementation and the native reproduction is transferred.
- base: historical pre-R9 fixture fails with 31 runtime selections; head: historical same-source fixture passes and fresh ordinary controller-refusal passes; no current native claim.
- live: no live surface; no new fix or fixture execution on a supported patched runtime.

stage: impl-review - skipped(policy: conductor-deferred - conductor owns review and completion)

All owned command handles are terminal. The Go/cache writer lane is released. No product hashes changed; the two user-owned untracked `.turbo` documents were untouched. This subtree and the ignored persisted base are the worker's only writes.
