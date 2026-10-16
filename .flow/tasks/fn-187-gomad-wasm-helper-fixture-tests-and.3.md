---
satisfies: [R3]
---
# fn-187-gomad-wasm-helper-fixture-tests-and.3 Delete inline tests the corpora already cover

## Description
For network, volume, namespace, world, clock, broker and the helper: run cargo llvm-cov (or equivalent) with the inline mod tests excluded and the corpora on, list uncovered production lines, add hand-named corpus cases for each uncovered behavior (and regenerate on the Go side where the corpus is Go-generated), rerun coverage, then delete the inline tests. Record the before/after coverage per module and the list of unit tests kept as invariants (overlap and capacity counters, allocation-free probe paths, cache poison and lock handling) with the reason each cannot be a corpus case. Apply the same pass to the Go-side wasi unit tests that duplicate a corpus (namespace_test.go, volume_test.go, broker_test.go, network_v2_test.go).

**Size:** M
**Files:** wasmhost/src/{network,volume,namespace*,world,clock,broker,model,cache}.rs and their tests modules, wasmhost/tests/testdata/**, wasi/testdata/**, wasi/{namespace,volume,broker,network_v2}_test.go

## Acceptance
- [ ] Coverage report before and after per module retained; no production line loses coverage
- [ ] Every deleted test names the corpus cases that replace it
- [ ] Remaining unit tests listed as invariants with reasons
- [ ] Go-side duplicate tests handled the same way

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
