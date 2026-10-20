# fn-188.6 Temporal installation test environment isolation — 2026-10-20

## Cause and scope

Base candidate: `81b97243a8`, checked out on worker branch `fn188-temporal-install-environment` in `/home/agent/wt/fn227-map-order`. The prior worker branch remains preserved; its cherry-picked history was not an ancestor of this base, so it was not reset or rebased.

The frozen `3f532910be` candidate's `/home/agent/wt/evidence/qualification-final/qual-3f532910be/ordinary.log` records failures at lines 2055–2087. `TestInstallTemporalAndDiscover` and four `TestInstallTemporalRefusesMissingInputs` cases inherited `GOMAD_STOCK_GO` from the validation driver and failed at the explicit-environment refusal before their intended assertions.

The production installation command deliberately scans `os.Environ()` and refuses every present `GOMAD_` key, including empty values. The driver exports `GOMAD_STOCK_GO`, `GOMAD_WASM_STOCK_GO` and `GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD`. The tests were missing isolation already used by `cmd/gomad/internal/cli/launcher_test.go`.

Only `internal/backends/wasm/command/temporal_test.go` changes outside this artifact. Its local helper uses the existing `t.Setenv(key, value)` plus `os.Unsetenv(key)` pattern, preserving cleanup restoration for each inherited value, including a present-empty value. Each top-level Temporal installation test clears ambient keys before its own inputs are configured. The negative guard test injects its variable only after this cleanup, inside its subtest; the helper does not erase the injected value.

The former two nonempty environment cases move into a dedicated six-name, two-value table. It covers `GOMAD_STOCK_GO`, `GOMAD_WASM_STOCK_GO`, `GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD`, `GOMAD_TEMPORAL_CONFIG`, `GOMAD_WASM_INSTALLATION` and generic `GOMAD_FUTURE_OPTION`, with both nonempty and empty values. Every case checks status 2, the exact variable-specific refusal, no stdout and no installation file. Production refusal logic, CLI grammar and compiled model/runtime/helper inputs are unchanged; no timeout, generator or pin changes are included. The full tracked-source manifest changes because it includes tests and must be frozen again by the conductor.

## RED and GREEN

Before edits, this command exited **1**, reproducing the frozen log's same five failures (`0.003s`):

```sh
cd /home/agent/wt/fn227-map-order/tools/gomad && env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off GOFLAGS=-p=1 GOMAXPROCS=2 GOMAD_STOCK_GO=/Users/stephan/Workspace/skunkworks/gomad/temporal/tools/gomad/.toolchain/stock/2198cf19890232fc/go/bin/go GOMAD_WASM_STOCK_GO=/Users/stephan/Workspace/skunkworks/gomad/temporal/tools/gomad/.toolchain/stock/2198cf19890232fc/go/bin/go GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD=1 /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go test -p=1 -tags test_dep -count=1 -timeout=3m -run '^TestInstallTemporal' ./internal/backends/wasm/command
```

After the test-only isolation and negative coverage changes, the exact same command exited **0**, reporting `0.006s`.

The empty inherited environment variant also exited **0**, reporting `0.007s`:

```sh
cd /home/agent/wt/fn227-map-order/tools/gomad && env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off GOFLAGS=-p=1 GOMAXPROCS=2 GOMAD_STOCK_GO= GOMAD_WASM_STOCK_GO= GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD= GOMAD_FUTURE_OPTION= /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go test -p=1 -tags test_dep -count=1 -timeout=3m -run '^TestInstallTemporal' ./internal/backends/wasm/command
```

These are focused command unit tests with inert fake compiler/helper files. They do not execute a compiler, build a guest or supply a qualification result. The command-tool outputs are the worker's evidence; no separate raw logs were retained for these focused runs. The preexisting qualification log path above is retained by the conductor.

`git -C /home/agent/wt/fn227-map-order diff --check` exited **0**. Focused lint remains to be recorded after the conductor releases the host. No Flow completion, push, PR or CI action is performed by this worker.
