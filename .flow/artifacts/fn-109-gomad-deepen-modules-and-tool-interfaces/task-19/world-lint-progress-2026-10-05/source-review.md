# World lint source-progress review

The two-file candidate is ready for a source-progress commit. It removes S1025 and SA4006 while preserving the reviewed World error and recording behavior. The configured scoped lint gate remains RED with 26 findings. This assessment grants no formal Flow SHIP verdict or task acceptance.

## Scope and identity

I reviewed the working-tree diff against `8c1904447e3483639cd9714577e3baeeba3485c3` on branch `gomad`, including the surrounding implementation and existing preservation tests. Production changes are confined to `tools/gomad3/world/snapshot.go` and `tools/gomad3/world/process/session.go`. Base and candidate SHA-256 values match `source.sha256`; the pinned linter binary hash also matches. Unrelated untracked Turbo files are outside the review.

This is a fresh-context independent senior source reviewer, in the same Codex family as the writer. Native dispatch requested the AGENTS reviewer route `gpt-6.1-sol` at `high`; that routing request is not an independently verified executed-model annotation. Tier disclosure supplied for the session is `session (jev-unavailable(no_key))`. I used no bridge or additional agent.

## Strengths

- `invalidSnapshot(field string)` passes the exact builtin string to the same `classifiedError(invalidSnapshotSentinel, ...)` constructor. Formatting `%s` supplied no transformation. The resulting detail, concrete `modelError`, cause identity, unwrap behavior and owned invalid-input terminal classification remain identical, including strings containing Unicode, newlines or percent characters.
- `RecordingHeader()` returns `[8]byte` and only returns `recordingMagic`. Both old uses measured its fixed length. `headerSize := len(world.RecordingHeader())` retains the same length of eight at the same bounds check and payload slice; the inspected function has no side effects. `Open` still writes the complete header bytes, and `EncodeRecording` still supplies the same envelope.
- Session finishing retains validation and error callback order, capacity-first classification, recorder finishing, encoding, `writeAll`, close-error joins and success-path state reset. Every existing failure branch and its error precedence is byte-unchanged. The patch changes no comments, public API, protocol, generated input, test or assertion, policy, exclusion or waiver.
- The retained lint logs have an exact two-diagnostic delta. Removing only the original three-line SA4006 and S1025 blocks and changing aggregate totals makes the baseline log byte-identical to the candidate log. All 24 ST1005 and two forbidigo findings retain their locations, excerpts and bytes.

## Issues

- Critical: none introduced by the reviewed diff.
- Important: none introduced by the reviewed diff. The remaining configured lint failures continue to prevent a green lint claim and broader acceptance.
- Minor: none requiring a source change.

## Checks performed

I read AGENTS.md, the Gomad README and milestone constraints, ran `flowctl usage` before state reads, then used `flowctl brief` and `flowctl show fn-109.19 --json` / `fn-109.18 --json`. Task 19 remains in progress and task 18 remains blocked. I inspected the production diff, error constructor, recording encoder/header, session finishing and existing terminal/snapshot preservation controls. `git diff --check` passed, and HEAD remained the stated base.

I independently ran these commands from `tools/gomad3` with stock Go 1.27.1 on developmental `linux/arm64`:

```sh
GOTOOLCHAIN=go1.27.1 GOWORK=off go test -count=1 -tags test_dep ./world ./world/process ./world/mailbox
GOTOOLCHAIN=go1.27.1 GOWORK=off /tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 run --config=../../.github/.golangci.yml --build-tags=test_dep --timeout=10m --fix=false ./world ./world/process ./world/mailbox
```

All three package tests passed. Actual unfiltered scoped lint exited 1 with exactly 26 findings, comprising 24 ST1005 and two forbidigo. I also verified source/linter hashes and the exact retained baseline-to-candidate lint comparison with `sha256sum`, `git show`, `perl` and `cmp`; the comparison passed.

I inspected the retained commands and logs for nine focused tests passing on both baseline and candidate, 45 candidate top-level package tests, baseline/candidate ordinary root suites, generator validation and three focused architecture/error-provenance controls. I did not rerun those root, generator or architecture commands in this review. The focused tests were already green on the baseline and provide preservation evidence; the configured analyzer supplies the RED 28 to RED 26 progress signal.

## Acceptance boundary

`READY_FOR_SOURCE_PROGRESS_COMMIT=yes` applies only to these two source substitutions and their retained evidence. Task 19, its predecessor obligations, original R8/R18/R19 requirements, complete/full/default/functional/affected-consumer and native Darwin acceptance remain open. Native Linux qualification retains fn-128 ownership. This developmental host test run supplies no supported-platform runtime qualification. The historical `sidecar_publish_failed` formal dispatch has no verdict and was not retried. I changed only this review artifact and performed no source, index, HEAD, Flow state or lifecycle mutation.
