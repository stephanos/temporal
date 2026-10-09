# fn-109.55 source-progress handover

Verify-only replay now returns status 3 when its single success-report write fails, and status 0 when it succeeds. The exact message, artifact-path operand, completed verification, request fields, callback timing and empty stderr remain unchanged.

Task remains `in_progress`. Root owns independent progress review, lifecycle and commit. Required source acceptance remains open.

## Final packet

Use [reconciled-evidence.json](reconciled-evidence.json) and [reconciled-source-proof.json](reconciled-source-proof.json) for the current candidate. Base is `a2b020a178bd8cae92016d2c8cd4cdca81023d23`; final source fingerprint is `6d6c4da4def556ee2d191eea7d9a832c1a459f2e332e3d349c61178f6cccd660`. Each `reconciled-*.json` receipt binds its actual command, exit, elapsed time, raw output hashes, source before/after and pinned tools.

The immutable [initial evidence](evidence.json) and [initial proof](source-proof.json) retain the first candidate and all mismatch/probe receipts. Initial ordinary results were 430 passes and 5 failed test observations. Root subsequently admitted only the stale `replay_verification` expected datum 0-to-3 correction and removal of its immediately owning obsolete comment. Final ordinary results are 432 passes and 3 failures, with zero skips. The direct mismatch and its enclosing group failure disappear.

## Causal proof and preservation

[focused-red-final.json](focused-red-final.json) records the old status 0 against expected 3 after a real read-only-file write returns `syscall.EBADF`; healthy and earlier-error controls pass independently. [reconciled-focused.json](reconciled-focused.json) records 33 passing observations covering the new tests, ordinary replay characterization and output-writer characterization. Tests retain one exact output attempt, the distinct returned artifact path, completed callback, context/request fields, installation requests and earlier statuses 3/2/3 with no success output.

The final proof reconstructs complete base `cli.go` by reversing only the write check, and complete base `characterization_test.go` by restoring only the admitted datum/comment. The other 1,049 tracked Gomad sources and old tests are unchanged. Both unrelated Turbo files remain byte-identical and unstaged; the git index is empty.

Recent CLI history and relevant memory supplied no competing fix. No known-good historical revision justified a bisect. Root owns external admission; this worker performed no tracker or PR writes.

## Final gates and remaining gaps

Focused tests, affected vet, standalone errortype, architecture/public/purity checks, private-injection checks, both supported source-set static checks, fresh check-only validation, formatting and actual task-base `FIX=false` fast lint pass. Full ordinary CLI exits 1 with only the three inherited failures named in final evidence. The `cmd/gomad` TestMain build also fails before collection because repository `.toolchain/bin/go` is absent; this is no native pass.

Actual unfiltered affected configured lint exits 1 with the unchanged `application.go` ST1005 finding. Actual original-base `make --trace lint-code-gomad3` exits 2 with 95 findings. Retained task-54 baselines were 2 and 96 respectively. Each comparison removes exactly the verified-report errcheck finding and introduces zero findings. Integrated errortype remains unreached after configured lint fails; standalone and fast passes do not replace it. See [task-54 handover](../task-54/handover.md) for retained predecessor defects and admission limits.

The bounded exported-`Run` attempt used the actual stock-Go test executable/buildinfo, current I/O profile, real host and SHA256-bound stock-tool identity without World, choice or simulation state. Artifact publication and preflight reached adapter verification, which refused actual `linux/arm64`. [public-fixture-diagnosis.json](public-fixture-diagnosis.json) and [attempted fixture source](public-fixture-attempt.go.txt) retain that result. Exported-`Run` success-write/EBADF coverage and patched-runtime/native replay remain unproved. No host spoof, arbitrary executable bytes, production seam or native guard shipped. Native owners fn-149/fn-128 remain deferred and unverified.

Tier: session (jev-unavailable(no_key))

Implementer preference was `gpt-6.1-sol at high`; actual execution metadata is unavailable.

stage: impl-review - skipped(config: REVIEW_MODE=none; required source gates remain red and root owns independent progress review)

All command handles are terminal. The implementation/shared-gate lane is released. Changes remain unstaged and uncommitted for root review and commit; no Done, formal verdict, push, PR, CI or native qualification occurred.
