---
satisfies: [R5]
---
# fn-187-gomad-wasm-helper-fixture-tests-and.5 Measure and document the simplified test suite

## Description
On the same host, retain before/after line counts (Rust production, inline test,
tests/, Go wasi test) and `make -C internal/backends/wasm test` wall time with
medians and ranges. Document the corpus as the contract, regeneration workflow,
invariant-test policy and wasmmeasure in README and ARCHITECTURE. Produce the
batch handover mapping tasks .1 to .5 and R1 to R5 to evidence and independent
review. Use the original pre-fn187 baseline, not merely the current R3 comparison
baseline. Final counts/timings must use the frozen restructured candidate.

Baseline preparation (not R5 acceptance): exact original Git object
bad3c401b5e7dbdf8bc8afaca3d08cad4da154fe yields physical counts24429 Rust
production/11640 inline-test/6734 tests-dir/21898 Go-WASI-test lines via existing
review3/count_lines.py (SHA25692d858a03836e2e7ba7b4163253c0d035b85589bb9748f60bcb18d5fc7aad95b).
Raw TSV is /home/agent/wt/evidence/fn187/r5-current/original-baseline-physical-counts-2026-10-19.tsv.
This counter classifies test modules, not every test-only helper item; state the
same convention explicitly on both sides rather than claiming compiled-code LOC.
The baseline worktree itself has a dirty cluster guest spin-count edit2000->4000,
retained as baseline-worktree-contamination-2026-10-19.patch in the same root.
Do not time that dirty source or revert it without establishing ownership.
Use pristine original bytes for actual baseline timings. No timing result is
claimed from these static counts; final after counts/medians/ranges remain due.

Source-only preparation is not acceptance: primary ARCHITECTURE records the
shared Rust/Go helper corpus contract, Go-oracle preservation, repeated
regeneration and byte stability, coverage-before-deletion, retained-invariant
policy and separate wasmmeasure invocation/execution owners. Gomad README links
that section and existing WASM regeneration instructions. These were checked
against the helper loader, wasmmeasure and Makefile; referenced files exist and
the documentation diff passes whitespace checks. Final source-bound review and
all original R5 deliverables remain required after R4.

The owner authorized WASM resource-sensitive execution, superseding the old
hold on timing, qualification and broad coverage/build matrices. Serialize all
compiler/runtime/LLVM work and preserve all original guards and budgets. Native
qualification remains deferred under fn128/fn149; fn185 profiling and ignored
benchmark roots remain out of scope. Keep the original test-wall-time medians
and ranges; do not substitute source-only checks or narrower timing commands.

**Size:** S
**Files:** tools/gomad/internal/backends/wasm/README.md, tools/gomad/README.md,
tools/gomad/ARCHITECTURE.md, .flow/artifacts/fn187-*.md
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
