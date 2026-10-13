---
satisfies: [R4]
---
# fn-173-gomad-wasm-helper-per-run-preparation.3 Drop the Go adapter's per-run module hash

## Description
Remove the per-run module `verifyFile` from the Go transport and rely on preparation-time module identity. Map the helper's miss-path mismatch to the existing boundary error (R4).

**Size:** S
**Files:** `tools/gomad3/backends/wasm/wasi/process.go`, `tools/gomad3/backends/wasm/wasi/process_test.go`
**Touches:** [tools/gomad3/backends/wasm/wasi/process.go, tools/gomad3/backends/wasm/wasi/process_test.go]

### Approach
- Delete the module `verifyFile` call (`process.go:306`). Keep the helper `verifyFile` (`:303`) and the module path/digest-format validation.
- When the helper terminates `invalid` with the module-digest mismatch message, return the same `BoundaryError` class today's `verifyFile` returns (`process.go:219-236`), so callers (`backend/provider.go:496`, `simulation/provider.go:567`) see no new outcome kind.
- Split `TestTransportRejectsChangedFilesAndInvalidTerminalClaims` (`process_test.go:374`): a changed module with a cold cache gives the boundary error; a changed module after a warm hit succeeds (by design); a changed helper is still rejected.

### Investigation targets
**Required:**
- `tools/gomad3/backends/wasm/wasi/process.go:219-310`, `:760-800`
- `tools/gomad3/backends/wasm/wasi/process_test.go:318-420`

### Quick commands
```bash
make -C tools/gomad3/backends/wasm helper
(cd tools/gomad3 && env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off go test -tags test_dep -count=1 -run 'TestGuestTransport|TestTransport' ./backends/wasm/wasi)
(cd tools/gomad3 && env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off go test -tags test_dep -count=1 ./backends/wasm/backend ./backends/wasm/simulation)
```

## Acceptance
- [ ] No full module read or hash remains on the Go per-run path; helper verification stays.
- [ ] A cold-cache module mismatch returns the existing boundary error; a warm hit with a changed module runs (tests).
- [ ] `./backends/wasm/backend` and `./backends/wasm/simulation` tests pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
