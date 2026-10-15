# fn-176.6 broker split ownership acceptance

Task 6 qualifies the inherited split implementation from merge `cd2cf0c9f6` on
`gomad`, retaining the output accounting correction `2668af049e`. The lifecycle
base is `41efa72b189ec43c4e901240ef4f47e1756c3172`; implementation review uses
historical base `2237e0276f50f3b4b4b244f32a564ec17caca110` so it sees the
implementation as well as this task's stronger acceptance controls.

The conductor approved and recorded two additional Touches: `wasmhost/tests/engine.rs`
and `wasi/integration_test.go`. Changes extend tests; production sources and the
default helper remain unchanged. Helper SHA256:
`7ccd46787a123e3e7d7b108d31d778a4223b258de4f606fe9c66fcdb935bdb64`.
Go model identity: `sha256:36e0c79512ddcd3c284c1d2710b146b18fe8c4258f1b2597d06cdf7b0e1ee471`.

## Acceptance mapping

- Routing: `ownership` selects split ownership for cooperative Broker storage,
  including NetworkV2. Rust namespace `standard_output` checks original descriptor
  identity, so a closed/reused fd 1/2 becomes local. All other namespace/volume
  operations remain local. `requireSplitServedOnlyOutput` now also checks exact
  callback counts on interrupted-prefix, SQLite output-crash and both ordered
  storage-crash/revocation fixtures. Existing recovery tests check the same rule.
- Parity: `TestBrokerCrashRecoveryFreshGuest` passes for clock read steps 0 and
  1000, comparing choices, transcript bytes/digests, storage evidence/image and
  global event order across Go-owned and split runs. The interrupted-prefix
  fixture compares partial choices and diagnostics. `TestStorageOwnerParity`
  covers SQLite `CrashOnOutput`, recovered history
  `1:committed,2:committed`, and both storage-crash/revocation orders. Fresh run:
  `.flow/tmp/fn176.6-broker-parity.log`, exit 0, 20.158 seconds.
- Reply crossing: `split_revocation_preserves_storage_before_local_write_or_remote_output`
  runs the real helper with an already-open, synced volume file. For crash and
  stop it compares revocation delivered with the completed reply against
  revocation crossing the reply, with either a local write/sync before the next
  output or the next output directly. Storage log, structured evidence/history
  and image bytes match; the revoked write and sync never occur. Continue
  controls execute the write/sync and exit. The original local-open control remains.
- Stress: `.flow/tmp/fn176.6-stress.sh` ran the three Go transport/split revocation
  tests 50 times (150 top-level passes), then the real-helper matrix 50 times,
  under twelve busy loops throughout. Exit 0; `.flow/tmp/fn176.6-stress.log`.
  The trap killed and reaped only PIDs 4150496–4150507; subsequent checks found
  none alive.
- NetworkV2: `TestProviderActualCrashDurableRestart` and
  `TestProviderUnboundRevocationOfSplitStorageNodeRestarts` pass, including crash
  and stop. `.flow/tmp/fn176.6-network-storage.log`, exit 0, 27.599 seconds.
- Protocol: `protocol.md` documents split mode, local clock/yield/poll behavior,
  descriptor routing, cumulative offered-output accounting, `revocable`,
  `sync`/`continue(now)`, completed-call interrupts and storage evidence flushing.

The accepted local clock/yield/poll deviation is recorded in the parent spec.
The output accounting correction is covered again by
`TestStorageSplitOutputBudgetAfterClose`, `TestHelperBridgeSplitOutputStateAdmission`
and `TestTransportSplitOutputTerminalBounds`; see the task 4 acceptance artifact.
The new helper fixture initially supplied an empty interrupt record without
chunks; the protocol correctly refused it. Omitting the absent optional record
corrected the fixture; no production workaround was needed.

## Verification

Baseline Quick commands passed before edits: Rust volume filter (54 unit plus
one integration match), volume conformance, and the parent Go storage selection
(82.894 seconds). Both stock compiler variables point at
`tools/gomad/.toolchain/stock/878c9b2935e2c9e6/go/bin/go`; `TMPDIR=/tmp`.

Final logs under `.flow/tmp/`:

- `fn176.6-rust-final.log`: full release Rust suite, exit 0; 196 unit tests,
  151 integration tests, two existing ignored unit tests.
- `fn176.6-conformance-final.log`: volume conformance, exit 0.
- `fn176.6-fmt-clippy.log`: cargo fmt and conformance clippy with warnings denied,
  exit 0.
- `fn176.6-lint.log`: configured native Linux lint, exit 0, zero issues.
- `fn176.6-go-final.log`: parent Go storage Quick selection, exit 0, 84.440 seconds.
- `fn176.6-validate.log`: generated-source, pins and qualification-manifest
  validation, exit 0.

Final frozen qualification, Temporal committed-workflow gates, workload success
and canary cells, and all four WASM SQLite selected states/catalogue belong to
task 7 and the conductor. Native execution remains deferred under fn-128/fn-149.
These focused results make no full qualification claim.

Tier: session (jev-unavailable(no_key))
