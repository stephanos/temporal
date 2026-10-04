# Developmental lint tooling diagnostic

The original task-21 preflight failed before linting: the installed
`.bin/golangci-lint-v2.13.0` is Mach-O ARM64, while this host is Linux aarch64.
Its first sixteen bytes begin `cf fa ed fe 0c 00 00 01`; SHA-256 is
`61f380f1d4c0c57b6cc0a4df3b72183f067c860818943a258aa76171456761e8`.
Make considers an existing tool path satisfied and does not rebuild it for the
current host. The original binary and failed preflight log remain unchanged.

The conductor used the existing Make installer to build the same pinned
golangci-lint v2.13.0 and errortype v0.0.7 under the isolated
`LOCALBIN=/tmp/fn109-lint-tools.ZdNe1t50`. Pinned stock Go 1.27.1 was first on
PATH, with GOENV=off, GOFLAGS empty, GOWORK=off, GOTOOLCHAIN=local and
GOMAXPROCS=2; GOMADSEED and GOMAD3_CHILD_SEED were unset. Installation session
86907 exited 0. No source, tool version, configuration or old binary changed.

The resulting Linux ELF linter reports version 2.13.0 built with Go 1.27.1.
Its SHA-256 is `acacd04faf1d17489a890a4589ae36a62484c1076cf5fbcb7a33274cbc6a6bdc`;
errortype is `db481b4086fb85962e98ae625984f2560a883dce91f8b7d8c705414c207be8cc`.
[The installation log](lint-tool-install-linux.log) has SHA-256
`5dada105abfa3481fedc11d96d17444084dc05bf5eec296e9f381a224147ddd0`.

The conductor then ran the unchanged `make lint-code-fast` target with that
LOCALBIN and `GOLANGCI_LINT_FIX=false`, under the same stock-Go environment.
Session 66346 is terminal: Make exited 2; golangci-lint exited 7 after
147.354353404 seconds. Its root-module loader rejected nested Gomad overlay
and conformance fixture package paths. In particular, the main module does not
contain `tools/gomad3/toolchain/runtime/overlay/src/cmd/internal/gomadcap` or
`tools/gomad3/internal/gomadtool/conformance/testdata/io_filesystem`.
The existing nested-module discovery limitation is now directly evidenced,
rather than concealed by the incompatible executable.

[The complete lint output](lint-code-fast-linux-development.log) has SHA-256
`e2943234170feda43d260270139ef8567ac9d3092d31c585b4558abcbbab9cd7`.
No tracked source changed during either command. This is a failed developmental
lint gate, not a passing check or either platform's native qualification.
Task 21 records the gap; an implementation owner must resolve package discovery
or supply the existing correctly scoped gate evidence. Do not alter selection,
baselines or lint policy under this verification task to manufacture a pass.
