# fn-184 isolation feasibility preparation

Task .1 is in progress; no mechanism is adopted and no acceptance is complete.
Main remains the qualified fn-180 snapshot source `90c1230140` plus task/docs
commits. Fn-180 R6 and fn-182 precede main-tree isolation adoption. The R5
baseline must be measured on unchanged main in a quiet Linux/arm64 window
before integration; no prototype timings count as that baseline.

One worker owns two disposable worktrees rooted at `90c1230140`:

- `/tmp/gomad-fn184-nested-prototype-20261016`, branch
  `prototype/fn184-nested-20261016`.
- `/tmp/gomad-fn184-tag-prototype-20261016`, branch
  `prototype/fn184-tag-20261016`.

They test module/constraint feasibility with explicitly temporary CLI stubs,
not production CLI behavior or portable-test preservation. Original native
sources, overlay bytes, fixtures and ledgers must remain preserved. Root owns
integration, commits, review and Flow completion. The worker runs
`gpt-6.1-sol` at medium; the tier judge was unavailable (`no_key`), so the
explicit project implementer model applies. Full prototype evidence is to be
retained at `/tmp/gomad-fn184-prototype-evidence-20261016`.

Read-only baseline preflight is retained at
`/tmp/fn184-r5-preflight-20261016-6UDVtR/commands.txt`. Current canonical
`test-harness` invokes patched-native runtime fixtures; those cannot run under
the existing hibernation waiver and must not be relabeled as portable coverage.
The source build command is `go build ./...`, not the Make runner/toolchain
build. Linux lint requires command-line tool overrides and
`GOLANGCI_LINT_FIX=false`. Race/cgo compiler support is available but has not
yet been exercised. Capture selected package/file inventories with `go list`,
deduplicating test variants; raw directory line counts are not selected-set
measurements. No baseline gate or timed cell was launched in this preflight.

Native execution stays deferred under fn-128/fn-149. No push, PR or CI action
is authorized. The prototypes neither change WASM semantics/identities nor
waive eventual default/selected source checks, preservation, baseline timings,
production CLI refusal controls or independent review.
