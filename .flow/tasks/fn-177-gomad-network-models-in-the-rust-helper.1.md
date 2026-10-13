---
satisfies: [R2]
---
# fn-177-gomad-network-models-in-the-rust-helper.1 Go-owner network operation corpus with currency and drift pins

## Description
Generate a versioned corpus of network operation scripts and their expected results from fn-154's Go owner. The Rust owner replays it in tasks 2-3. Add a currency test and a drift pin so the hand-written Rust copy cannot silently fall behind the generated Go projection (R2 model level).

**Size:** M
**Files:** `tools/gomad3/simulation/network/corpus_test.go` (new: scenario builders, generator, currency test), `tools/gomad3/backends/wasm/wasmhost/tests/testdata/network_v2/*.json` (new, generated), `tools/gomad3/backends/wasm/wasmhost/src/network/PINNED_GO_NETWORK_SHA256` or a const in the Rust module (pin), `tools/gomad3/backends/wasm/wasi/network_v2_test.go` (pin check)
**Touches:** [tools/gomad3/simulation/network/corpus_test.go, tools/gomad3/backends/wasm/wasmhost/tests/testdata/network_v2/**, tools/gomad3/backends/wasm/wasmhost/src/network/**, tools/gomad3/backends/wasm/wasi/network_v2_test.go]

### Approach
- Script steps mirror the Go `Model` API (`simulation/network/adapter.go`): register/revoke domains, dispatch of ops 0-13 with full inputs, cancel, partition/heal/delay/reset, advance(now) → ready ids, next deadline, and the final Record (transitions plus snapshot structure). Expected outputs per step: results, errno, state, ready ids and deadlines.
- Scenarios: port fn-154's model tests as scripts. That includes equal-time heal versus expiry in both orders (`stall_test.go:114`), simultaneous pair expiry (`:292`), reservation exhaustion (`:384`, `:413`), autonomous expiry using the reserved slot (`:65`), overflow rejection (`:183`), held/timeout replay admission (`adapter_test.go:88`), plus FIFO release after heal, delay, reset, refused dial, pending-dial stall and capacity. Add seeded random sequences from the generator behind `TestGeneratedOperationSequencesExactNativeReplay` (`adapter_test.go:279`). Add multi-node, broker-shaped scenarios as `brokerNetwork` builds them (`wasi/broker_network.go`): several node pairs, strict topology, partition/heal/delay across all four client/server pairs, revocation of a node's domains, and per-slot request bases. fn-178 is the owner's only consumer.
- Writing: an env var regenerates the files; the default run regenerates in memory and fails on any difference (pattern: `TestGeneratedNetworkIsCurrent` in `simulation/network/generate/main_test.go`).
- Drift pin: store `network.ImplementationSHA256()` (`simulation/network/identity.go`) in the Rust module. A Go test fails with an "update the Rust network owner" message when it differs.
- Gate: start only after fn-154 tasks 7 (pending-dial stall) and 8 (delivery history removal) are in source.

### Investigation targets
**Required:**
- `tools/gomad3/simulation/network/adapter.go`, `tools/gomad3/simulation/network/adapter_test.go:88`, `:279`
- `tools/gomad3/simulation/network/stall_test.go`
- `tools/gomad3/simulation/network/generate/main_test.go`
- `tools/gomad3/simulation/network/identity.go`

### Quick commands
```bash
(cd tools/gomad3 && env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off go test -tags test_dep -count=1 -run 'TestNetworkCorpus|TestModel' ./simulation/network/...)
```


### Source-gate admission (2026-10-13)

The owning task's explicit start gate is satisfied: fn154.7 pending-dial controls and fn154.8 payload-free history are integrated in the shared source candidate (connection/history checkpoint5ee4a93d58 and conformance6dbb0895ee/fe84b745d7). MILESTONES batch-first permits this disjoint dependent source lane while fn154 native acceptance remains open. Generate against current post-fn154 source; no pre-fn154 semantics, network production changes or helper integration. This lane touches only its declared corpus, pin and pin-test surfaces and runs in an isolated worktree. Final root review/integration and all fn154 acceptance stay unchanged. Later fn166 relocations require path/pin resynchronization, not waiver of corpus currency. Root owns lifecycle and review.
## Acceptance
- [ ] Corpus covers every fn-154 scenario named above, the broker-shaped multi-node scenarios and seeded random sequences; each script records per-step outputs and the final Record.
- [ ] Currency test fails on any Go-owner behaviour change until the corpus is regenerated.
- [ ] Drift pin test fails when the Go projection digest changes without a Rust pin update.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
