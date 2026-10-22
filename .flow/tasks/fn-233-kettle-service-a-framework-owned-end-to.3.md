---
satisfies: [R3, R5]
---
# fn-233-kettle-service-a-framework-owned-end-to.3 derive(Catalog) Kettle catalog, config.rs, Annotate in the live run, coverage and gap goldens

## Description
Write `api.rs` as the second `#[derive(Catalog)]` catalog (fn-217.6 surface): mirror structs for every in-scope Kettle message, kinds as field types, `#[rule]`/`#[mirror]` namespaces with presence rules at item level, `config.rs` over the generic scoping for the static and per-kettle limits, the hand-written exhaustive `match` over the generated `KettleServiceMethod`, and at least one `Ignored` field with a reason. Wire Annotate into the live run so every wire message is validated and filled; produce `coverage.snap` and `gaps.txt` with the generic probe.

**Size:** L
**Touches:** `model/umpire/reference/kettle/**`, `model/umpire/hosts/go/kettle*_test.go`, `model/umpire/hosts/go/reference/kettle/*.go` (annotation/probe integration controls), `model/umpire/hosts/go/reference/kettle/v1/**`, `model/scripts/proto-kettle-go.sh` (consumer probe schema generation/freshness), `model/umpire/catalog/**`, `model/umpire/catalog-build/**`, `model/umpire/annotations/**` (generic support only when required, with both consumer regressions), `model/temporal/foundation/dc/mod.rs` (shared resolved limits extraction; policy stays local), `model/temporal/hosts/go/umpire_temporal_guest.wasm` (required freshness after shared-code changes), `model/Cargo.lock`, `model/DESIGN.md`

### Approach
This is the delivered derive's second consumer (R5); record missing generic features and close them with controls for both consumers. Keep `api.rs` readable: `cargo fmt --check` at width 100, proto declaration order.

### Generic second consumer controls

Use the shared derive/Inspect/limits/method/wire-schema machinery delivered by .1; Kettle must compile without any temporal crate. Both Kettle and Temporal are regression consumers. Follow the parent RPC projection and mutant oracle contracts for every annotation/probe. Add focused compile-fail controls for omitted field, enum variant, service method and Model action, plus coverage/gap goldens with Ignored and Unverifiable. R5 records second-consumer portability evidence, not a retroactive first spike. Quick: catalog/syntax/descriptor fixtures and the focused Go healthy test; final: make model-check.
## Acceptance
- Every in-scope message mirrored; a field, enum value, method or Model action added to the Kettle proto or Model fails the build naming it (R3).
- Annotate runs on every wire message of the live run; `coverage.snap` and `gaps.txt` goldens with at least one `Ignored` reason and one `Unverifiable` (R3).
- DESIGN.md records every derive gap found, or states that none was (R5).
- No Temporal name under `umpire/`; `make model-check` green.


## Done summary
Kettle now uses the shared Catalog derive for its complete in-scope API, validates every healthy RPC request and response, fills omitted idempotency keys, and records coverage and actual-server gap goldens. Shared scalar-oneof, text-map and resolved-limit support closes the three second-consumer gaps documented in DESIGN.md.

R3 controls cover actual descriptor additions and compile refusals naming omitted idempotency_key, HeatingAction::Failover, KettleServiceMethod::Fill and User::DepressButton. The local field-shape refusal complements the actual Kettle descriptor-addition test. The healthy transcript, canonical Kettle Model/report and extension semantic goldens have no diff. Kettle's dependency tree contains no extension crate.

R5 controls include the generic scalar/map regression, existing extension derive/projected-oneof controls, both consumers' native suites, the actual extension slice and gap targets, and make model-check. Probes send original invalid bytes despite annotation predictions; unknown and transport failures refuse classification. Mutants remain owned by task .4.

baseline: green (focused catalog/syntax/build suites and Go healthy transcript before edits). An initial relative-cwd launch was inconclusive and replaced before editing.
Verification: make model-check passed in 438.622 seconds at source commit 5f7531e227; later changes contain README measurement prose and Flow evidence only. Kettle's complete Go selection took 26.480 seconds, native probes 0.16 seconds, and the independent extension gap target passed in 374.233 seconds. R30's 60-second whole-gate soft target remains exceeded. Scoped Rust Clippy and changed-line Go-host lint passed.

The first full gate rejected a stale embedded extension guest; the pinned builder regenerated it before the passing run. One earlier extension gap attempt is inconclusive because the worker consumed its stdout pipe, causing the Make summary gate to fail despite Go exit0. The retained untouched rerun passed the exact Make gate. Both failed/inconclusive observations remain separate in the durable artifacts.

stage: impl-review - ran [2026-10-22T18:53:18Z..2026-10-22T18:57:46.182909Z] (model: gpt-6.1-sol at high; correctness/contracts/integration all SHIP; same-family review)
Tier: session (jev-unavailable(no_key)); repository implementer override gpt-6.1-sol at medium.

Durable verification and review: .flow/artifacts/fn-233-kettle-service-a-framework-owned-end-to/task3-{verification,review}.json and task3-*.log.gz.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: a8542e8a393323de1a5516aafbcce8569b0bb16b, 5f7531e2272536dcf1fedd26ac9ef0f171b39095, 2a5429587642acac96ecfd4be58566d42761778f, 8948f8fb6e076e43817a2ed7d64232539a102ddf
- Tests: mise exec -- cargo test -p umpire-catalog -p umpire-catalog-syntax -p umpire-catalog-build -p umpire-reference-kettle -p umpire-temporal-annotate, mise exec -- cargo test -p umpire-reference-kettle --test probes --test annotations, make model-check, make umpire-check-model-slice, make umpire-check-model-gaps, go test -tags test_dep -run '^TestKettle' -count=1 ., mise exec -- cargo clippy -p umpire-reference-kettle -p umpire-catalog -p umpire-catalog-syntax -p umpire-temporal-annotate --all-targets -- -D warnings, Go-host changed-line golangci-lint v2.13.1 --fix=false: 0 issues, make GOLANGCI_LINT_BASE_REV=20a6f2a52881c0d2673b553e39da78b913cca203 GOLANGCI_LINT_FIX=false lint-code-fast, INCONCLUSIVE: earlier extension gap Make summary lost because worker read child stdout pipe; independent untouched exact Make rerun passed
- PRs: