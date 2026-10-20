# fn-188.6 validation gate repairs (2026-10-20)

Source base: `8cd478404b2db7d2fb2c528d210bc7cfa29b8be8`; implementation checkout:
`/home/agent/wt/fn188`. Host: Linux/aarch64, stock Go 1.27.1.
Raw logs and numeric `.exit` files:
`/home/agent/wt/evidence/fn188/validation-repairs-2026-10-20/`.

The frozen candidate's full validation attempt failed after 276 seconds with
15 stages passing and `lint-root` plus `test-harness` failing. Its unchanged
commands, receipts and failure schedule remain at
`/home/agent/wt/evidence/qualification-final/qual-8cd478404b/`.

The root lint router now classifies only the two declared Gomad debug fixture
roots, `common/gomaddebug/testdata/determinism` and `fixture`, as qualification
fixtures. Eight regression cases cover both roots, descendants, similarly named
siblings, an unknown testdata directory and ordinary host source. Unrelated
build-tag and package-metadata errors retain their existing fail-closed behavior.

The launcher routing test clears inherited `GOMAD_*` entries with cleanup that
restores their original presence and values. Its original missing-package
assertion is preserved. Additional cases require both a nonempty and an empty
ambient guard to be refused. CLI production code and its explicit-input policy
are unchanged; helper, model and runtime sources are unchanged.

| Check / raw log prefix | Before | After |
| --- | --- | --- |
| Root fixture boundary regression / `root-router-{red,green}` | 1 | 0 |
| Actual root lint at the full ladder's base / `root-lint-{red,green}` | 2 | 0, both root and Gomad scopes report zero issues |
| Launcher routing with inherited guard=1 / `launcher-{red,green}` | 1 | 0, all three cases pass |
| Full root router suite / `root-router-full` | — | 0, 7.698 s |
| Original full `test-harness` with driver environment / `test-harness-green` | 2 in frozen attempt | 0 |
| `git diff --check` | — | 0 |

Rerun from the implementation checkout with the stock compiler directory first
on `PATH`, `GOROOT` unset, `GOENV=off GOTOOLCHAIN=local GOWORK=off GOFLAGS=-p=1`:

```sh
go test -v -tags test_dep -count=1 -timeout=20m ./cmd/tools/lintcode
make lint-code-fast GOLANGCI_LINT_BASE_REV=65a5f64ffbfaeba93b8f3f33ac3e025759ac5db1 GOLANGCI_LINT_FIX=false GOLANGCI_LINT=/home/agent/wt/evidence/fn219/qualification-inputs/host-tools/golangci-lint-v2.13.0 ERRORTYPE=/home/agent/wt/evidence/fn219/qualification-inputs/host-tools/errortype
cd tools/gomad
GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD=1 go test -v -tags test_dep -count=1 -run '^TestLauncherRoutesTemporalSetupBeforeInstallation$' ./cmd/gomad/internal/cli
GOMAXPROCS=2 GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD=1 GOMAD_STOCK_GO=/Users/stephan/Workspace/skunkworks/gomad/temporal/tools/gomad/.toolchain/stock/2198cf19890232fc/go/bin/go GOMAD_WASM_STOCK_GO=/Users/stephan/Workspace/skunkworks/gomad/temporal/tools/gomad/.toolchain/stock/2198cf19890232fc/go/bin/go make test-harness
```

These focused source checks repair the two observed gate failures. The conductor
must review and integrate this batch, then freeze the new candidate and rerun all
96 stages within 3600 seconds. No task completion or current qualification pass
is asserted here.
