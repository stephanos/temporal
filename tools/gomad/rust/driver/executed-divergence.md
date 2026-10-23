# Original executed divergence receiving

Build the `counterfactual-fixture` and `counterfactual-harness` roles with
`main_cli_build.py` against the same immutable candidate and tool plan. Then run:

```sh
python3 executed_divergence.py produce \
  --cli-build CLI_BUILD/main-build.json \
  --harness-build HARNESS_BUILD/main-build.json \
  --compiler-root STOCK_GO_ROOT --module-cache SEALED_MODULE_CACHE \
  --root FRESH_EVIDENCE_ROOT
python3 executed_divergence.py verify --receipt FRESH_EVIDENCE_ROOT/result.json
```

The producer admits both complete build receipts before executing either test.
It invokes the exact original-fixture preparation test and the four-pair campaign
test, retaining actual command/process journals. The preparation recipe contains
only a fresh root, stock compiler and separately admitted public CLI. Rust derives
all four campaign options through the production parser, pins the original guest,
applies the original runtime patch and retains the original two-owner scenario.
The original provider's 8 MiB transcript policy and the campaign's effective
64 MiB override remain separate evidence. No serialized campaign receipt is
supplied by the launcher.

Setup performs four genuine installations with separate compilation caches and
constructs their requests; it does not compile a Go guest. The controller refuses
a precompiled WASM module before starting the campaign producer. Each baseline
creates its actual preparation receipt inside its five-minute clock. That
receipt's separate preparation journal remains untouched while the real campaign
journal executes, and the fault run binds to the actual baseline preparation.

`--module-cache` supplies the genuine `golang.org/x/sys v0.48.0` source and Go
download metadata needed by the compiler overlay helper. The archive is checked
against the candidate's `go.sum`, copied into the private run root and checked
again before the campaign producer and by the final reader. Both child commands
receive the private `GOMODCACHE` explicitly. Go build caches are not prewarmed.

Each baseline/fault run prepares and executes through the production scheduler,
journal and retention code under its original five-minute overall deadline.
The reader reopens the one-failure/one-artifact ledger, diagnostics and both
provider and literal public prefix replays. It checks both build receipts and
the compiler inventory again after execution. Cache files are retained as
reproducible build state; runtime sources, process journals, preparation receipts,
campaigns and artifacts are hashed in the receiving result.

The historical handoff predicate requires both owners to exit, while the original
Go execution and retained Rust execution stop the peer after the first owner
exits one. The receiving gate preserves that failure, including libtest exit 101.
A completed receiving receipt may therefore report `FAIL_ORIGINAL_PREDICATE`.
It grants no final qualification pass or inventory migration.
