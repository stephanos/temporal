---
satisfies: [R2]
---
# fn-151-wasm-gomad-execution-backend.2 Implement deterministic WASI environment

## Description
Add captured argv/environment, seeded entropy, modeled descriptors/namespace, coherent experimental clock/poll and capacity behavior with positive and negative seam tests. Verify fresh-guest repeated fixtures and unsupported operation classification.

## Acceptance
Add captured argv/environment, seeded entropy, modeled descriptors/namespace, coherent experimental clock/poll and capacity behavior with positive and negative seam tests. Verify fresh-guest repeated fixtures and unsupported operation classification.

## Done summary
Implemented the sealed WASI environment and helper transport in the separate tools/gomad_wasm module, reusing public Gomad process and bounded-file owners. Immutable captured inputs and in-memory files, entropy and exploratory clocks deny live host capabilities. Fixed four independently reviewed isolation/namespace/lifecycle defects and invalid UTF-8 identity collisions with retained failing regressions.

The final 33-file candidate 73245ac15bc931e609971cfea005c966a3a1d4f49770aa482c9aabf60e478c5c passed the complete pinned module suite, including 32 Rust guest tests, 2 unit tests and 100 fresh stock Go1.27.1 guests in 204.27 seconds with identical full unsorted output and callback/reply transcripts. All guests exited0 and helpers were reaped. Stock classic GC remained enabled (nogreenteagc); this proves observed fixture repeatability, not forced scheduling or strict virtual time.

Model, negative lifecycle/transport race, public bridge race, lint, generated validation and both-platform purity/public-boundary checks pass. Root's 229.663-second portable source batch passed architecture/API/vet, analysis, shared supervision/files, captured-input, record/artifact/choice/World checks on the prior candidate. Only the two external WASI UTF-8 validation files changed afterwards; unchanged core hashes retain that evidence and root refreshed final affected boundary checks. Root independently verified final source/compiler/helper/log hashes; fresh same-family gpt-6.1-sol/high review reports no remaining actionable findings.

Evidence lives in .tmp/wasm-backend-environment (handover.json, final-proof.json, final-module-suite.log, root review/source receipts). Durable crash models, strict runtime hooks, Temporal guest execution and later gates remain pending. Native patched-runtime test-host remains deferred under fn-149.2. No files staged or committed; user retains commits and no backend publication is authorized.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: make -C tools/gomad_wasm test GOMAD_WASM_STOCK_GO=/Users/stephan/.local/share/mise/installs/go/1.27.1/bin/go, python3 .tmp/wasm-backend-environment/root-portable-source-gate.py, cd tools/gomad3 && env -u GOROOT -u GOMADSEED -u GOMAD3_CHILD_SEED GOENV=off GOTOOLCHAIN=local GOWORK=off GOEXPERIMENT=nogreenteagc GOMAD3_STOCK_GO=/Users/stephan/.local/share/mise/installs/go/1.27.1/bin/go PATH=/Users/stephan/.local/share/mise/installs/go/1.27.1/bin:$PATH /Users/stephan/.local/share/mise/installs/go/1.27.1/bin/go test -tags test_dep -count=1 -run ^TestWASIPureEnvironmentAndPublicBoundary$ ., make lint-code-fast, make lint-code-gomad-wasm, flowctl validate --spec fn-151-wasm-gomad-execution-backend --coverage --json
- PRs: