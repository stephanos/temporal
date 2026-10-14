---
satisfies: [R4]
---
# fn-173-gomad-wasm-helper-per-run-preparation.3 Drop the Go adapter's per-run module hash

## Description
Remove the per-run module `verifyFile` from the Go transport and rely on preparation-time module identity. Map the helper's miss-path mismatch to the existing boundary error (R4).

**Size:** S
**Files:** `tools/gomad/internal/backends/wasm/wasi/process.go`, `tools/gomad/internal/backends/wasm/wasi/process_test.go`
**Touches:** [tools/gomad/internal/backends/wasm/wasi/process.go, tools/gomad/internal/backends/wasm/wasi/process_test.go]

### Approach
- Delete the module `verifyFile` call (`process.go:306`). Keep the helper `verifyFile` (`:303`) and the module path/digest-format validation.
- When the helper terminates `invalid` with the module-digest mismatch message, return the same `BoundaryError` class today's `verifyFile` returns (`process.go:219-236`), so callers (`backend/provider.go:496`, `simulation/provider.go:567`) see no new outcome kind.
- Split `TestTransportRejectsChangedFilesAndInvalidTerminalClaims` (`process_test.go:374`): a changed module with a cold cache gives the boundary error; a changed module after a warm hit succeeds (by design); a changed helper is still rejected.

### Investigation targets
**Required:**
- `tools/gomad/internal/backends/wasm/wasi/process.go:219-310`, `:760-800`
- `tools/gomad/internal/backends/wasm/wasi/process_test.go:318-420`

### Quick commands
```bash
make -C tools/gomad/internal/backends/wasm helper
(cd tools/gomad && env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off go test -tags test_dep -count=1 -run 'TestGuestTransport|TestTransport' ./internal/backends/wasm/wasi)
(cd tools/gomad && env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off go test -tags test_dep -count=1 ./internal/backends/wasm/backend ./internal/backends/wasm/cluster)
```

## Acceptance
- [ ] No full module read or hash remains on the Go per-run path; helper verification stays.
- [ ] A cold-cache module mismatch returns the existing boundary error; a warm hit with a changed module runs (tests).
- [ ] `./internal/backends/wasm/backend` and `./internal/backends/wasm/cluster` tests pass.

## Done summary
R4 accepted after task2: Go per-run transport validates the module path/digest format without reading/hashing full module bytes, retains helper executable verification and maps a pre-start helper module digest mismatch to the existing BoundaryError. Cold mismatch, changed/deleted warm-hit module and changed-helper refusal controls pass; retained ordinary backend/cluster/wasi suites pass. The historical simulation/provider.go consumer is now cluster/provider.go.

Execution is the joined post-rename qualification at75835196d7, original driver exit1 at its documentation-only HEAD guard plus separately reviewed continuation exit0. Current WASM production source remains byte-identical through11b54c7161; host-only fixture repair has passing architecture regression/lint. Formal integrated fn173.1–.4/R1–R5 fanout ea61c8cf8806fbfb5841f56de511f991 finalized three-axis SHIP, zero findings/unaddressed; receipt .flow/tmp/impl-review-fn1731to4-current-20261014.json, actual gpt-6.1-sol/high, primary01a13b87-6885-71c3-ac6a-0c88a5fcc417. Same-family/read-only review inspected retained actual execution evidence. Task5 warm gate and whole-spec completion remain open; native execution deferred.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 7963bb8473b6df17ebeff7c3bb78714293cc6666, 9e0d09b5e6779f78f3c2f658761956e82223f569, 75835196d7345cc0f2d9886d79851fdd3b05eeb3, 50c0dec14337d8e27aef363c51151be4fe2bc8f7, 11b54c7161
- Tests: Retained full post-rename qualification: /Users/stephan/Workspace/skunkworks/gomad-postrename-retry-20261014.TMcAas/run-qualification.sh; original terminal exit1 at documentation-only HEAD guard after passed gates, joined with reviewed /Users/stephan/Workspace/skunkworks/gomad-postrename-doc-continuation-20261014.bL7lrR/continue-qualification.sh terminal exit0; preserve raw stages/receipts/source hashes., Pinned release default and diagnostic Rust tests/all-target Clippy, ordinary Go WASM backend/cluster/wasi and actual Temporal module natural terminal/reap measurements retained in the joined qualification; native execution deferred per unchanged owners/ledger., bash /Users/stephan/Workspace/skunkworks/gomad-crc-qualified-20261014.FFOu3a/measure.sh: terminal0; cache::tests::crc_throughput 1GiB23.786GB/s on real Linux/aarch64; source/compiler/CPU/boot bindings verified before/after., Formal implementation review ea61c8cf8806fbfb5841f56de511f991, fn173.1–.4/R1–R5, finalized SHIP correctness/contracts/integration; receipt .flow/tmp/impl-review-fn1731to4-current-20261014.json; actual gpt-6.1-sol/high, same-family, zero findings/unaddressed; read-only source/retained-evidence review, no invented test execution.
- PRs: