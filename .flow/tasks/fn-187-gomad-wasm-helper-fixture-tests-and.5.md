---
satisfies: [R5]
---
# fn-187-gomad-wasm-helper-fixture-tests-and.5 Measure and document the simplified test suite

## Description
On the same host, retain before/after line counts (Rust production, inline test, tests/, Go wasi test) and `make -C internal/backends/wasm test` wall time with medians and ranges. Document the corpus as the contract, regeneration workflow, invariant-test policy and wasmmeasure in README and ARCHITECTURE. Produce the batch handover mapping tasks .1 to .5 and R1 to R5 to evidence and the independent review.

**Size:** S
**Files:** tools/gomad/internal/backends/wasm/README.md, tools/gomad/README.md, tools/gomad/ARCHITECTURE.md, .flow/artifacts/fn187-*.md

Source-only preparation (2026-10-18): the primary checkout's ARCHITECTURE now records the shared Rust/Go helper corpus contract, Go-oracle preservation, repeated regeneration and byte-stability requirements, coverage-before-deletion and retained-invariant policy, and separate wasmmeasure invocation/execution owners. The Gomad README links to that section and the existing WASM README regeneration instructions. Checked against the current helper loader, wasmmeasure source and Makefile; referenced local files exist and the documentation diff passes `git diff --check`. This is documentation preparation, not an R3/R4 completion or qualification claim; independent review remains pending.

The owner's non-performance/resource-sensitive scope keeps timing, profiling, resource qualification and broad build/coverage matrices held. Final before/after counts must use the eventual frozen restructured tree, not the current partial candidate. Task remains todo behind .4; all original R5 deliverables and acceptance criteria remain required.
## Acceptance
- [ ] Before/after table with host identity retained
- [ ] Docs describe corpus contract, regeneration, invariant policy and wasmmeasure
- [ ] Batch handover maps every task and R-ID

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
