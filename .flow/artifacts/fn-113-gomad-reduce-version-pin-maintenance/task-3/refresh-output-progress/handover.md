# Refresh stdout failures

Refresh now returns infrastructure status 3 when any of its six stdout writes fails. The existing format strings, arguments and write order stay intact. The checks run after `authoring.Refresh`, so completed request and report publication survives the output failure. Earlier input and infrastructure errors retain their status and stderr behavior.

The source candidate uses base `1a7cd2bf7d7627fee75fb6be54058ccea0acf310`. [checks.json](checks.json) binds the three owned Go files, stock Go 1.27.1, lint tools, configuration, logs, commands, exits and elapsed seconds. The worker made no commit, push or Flow lifecycle change.

## Regression evidence

[The first RED](red-before-production.log) ran before the production edit and failed on seven real read-only-file write failures, with the original statuses 0 or 1. [The final-test RED](red-final-tests.log) repeats those failures using a stock-Go overlay of the exact Git-base production file. It also runs the invalid-input and missing-Go controls before the fault. [The final GREEN](green.log) uses the same frozen final tests and passes.

The focused run has 15 parent-inclusive terminal records and 12 terminal leaves. Seven real EBADF invocations cover six output call sites. The empty-refresh fault lives in the parent of its two precedence subtests. The approval fixture fails the header, awaiting-approval line and approval instruction after fresh publication; it checks the cleared approval, updated version, exact request/report/generation bytes and removal of the unapproved pack before approving and rerunning. Other cases fail the current, not-evaluable and unselected lines and preserve the existing artifact bytes.

## Verification

The selected CLI, compatibility-pack and authoring controls plus portable pin-impact controls have 499 parent-inclusive run records and 462 terminal leaves across four packages. They record zero failures and one skip. [The count receipt](portable-test-counts.json) lists the selected top-level tests. `TestHostPacksBindCurrentProfile` retains its existing skip because linux/arm64 has no deterministic profile. Check-only generator validation also encounters that skip.

Scoped vet and errortype, check-only `make validate`, all four architecture checks, formatting and `git diff --check` pass. [Unfiltered scoped lint](scoped-final.log) remains red with 132 inherited findings. [The comparison](scoped-lint-comparison.json) shows exactly six removals from the 138-finding baseline, with every remaining complete diagnostic block equal after diagnostic line and column normalization.

[Mandatory fast lint](fast-lint.log) passes across 55 host packages with fixes disabled, zero findings after diff filtering and integrated errortype reached. Its configured residual before diff falls from the prior baseline receipt's 308 to 302. That count comparison does not establish byte equality for the full integrated residual. The worker corrected two test-only tagged-switch findings discovered during verification before freezing this candidate.

The historical v041 report has SHA-256 `a83095bc7ffb68d1c79b6eb17260c08171003827d80e47789fa0e9be4cbf666e`, identical to the Git base. The complete compatibility-pack, toolchain, deterministic-I/O, Makefile and module files also remain unchanged against that base.

## Scope limits

The portable pin-impact run excludes `TestFixtureBumpMatchesBuildRejections`, `TestSameVersionWithChangedSum` and `TestReplacedModules`. The selected CLI run omits unavailable driver fixtures `TestRunCheckedRunWritesCompatibilityResult` and `TestRunCheckedRunDistinguishesExit124FromTimeout`, and the six unrelated pin-impact CLI tests named in `checks.json`. Existing native guards remain intact.

The patched `.toolchain/bin/go` is absent. This developmental linux/arm64 stock-Go evidence makes no native Darwin/Linux, full/default/functional, affected-consumer, workload qualification, determinism, formal SHIP or task-completion claim. Original acceptance, task .1/.2 dependencies and required native/formal gates remain open. Native Linux qualification remains deferred under fn-128. The conductor owns fresh source-progress review, lifecycle records and the separate checkpoint commit.
