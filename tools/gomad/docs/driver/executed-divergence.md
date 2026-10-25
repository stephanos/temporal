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

The final-cycle endpoint is `live_corpus.test_wasm_reach.executed_divergence`, with
exact absolute-path keys `cli_build`, `harness_build`, `compiler_root`, and
`module_cache`. Register the two separately built closed counterfactual roles in
`live_corpus.roles` and freeze `counterfactual_receiving.input_paths(entry, repo)`.
The existing `cluster-functional` stage runs the producer into its fresh
`executed-divergence` directory. Its original campaign budgets remain unchanged;
this adds no qualification stage. The final reader reopens the enclosing stage,
its actual child PID/clock interval, both role receipts and the complete four-pair
receipt before publishing any `original-divergence/<case>` receiving IDs.

The original helper's line170 assertion that every handoff incarnation terminates
with `exit` conflicts with the subsequently accepted fn235 R2 failure-stop
contract (implementation `a959c074baca3624107f159a8a6aeddd3070c4c6`). The original
FAIL and producer exit101 remain in the evidence. Only the exact `b/1` peer stop
in both handoff artifacts can receive an assertion-scoped
`SUPERSEDED_BY_ACCEPTED_OWNER_CONTRACT` applicability record. The originating
`a/1` must actually exit1; the peer must be interrupted by `cluster failed`.
All other original campaign, divergence and prefix-replay assertions still run.
The same owning stage must also provide the independently reopened native
`trap`, `exit`, and `held-peer-release` drain observations. These facts establish
current behavior; they do not make the unchanged original test pass.

The complete original source map must bind the original root to all four
`original-divergence/<case>` IDs and retain the exact applicability record from
the receiver. Its dynamic subcases bind to their corresponding single actual
case. The lineage reader refuses attempts to apply that record to another root,
rename a case, omit a pair, or replace the original failure with PASS.
