# Go model retirement inventory

Source: `dc76222c8a3a6a9e67170e019b9f7f7eefd8cc87` (full revision in [coverage.json](coverage.json)); task `fn-249-gomad-retire-the-go-reference-models.1`, R1. This is static discovery at that revision, not a runtime, mutation, regeneration, qualification or deletion result. Only `.1` analysis is admitted during fn-242; `.2` onward remains blocked until fn-242.8. No source, test, corpus expectation, Cargo feature or cycle contract changes here.

The owner permits replacing independent Go models with reviewed Rust corpora. `convert-to-Rust-corpus` below means a required future conversion, not completed coverage. Preserve functional observations, error class/identity, bounds, ordering and same-candidate replay; the legacy compatibility policy does not require old Go serialization spelling. A removal needs an explicit old-test-to-new-test mapping or a narrow obsolescence reason. No blanket deletion is authorized by this inventory.

## How to read the census

[coverage.json](coverage.json) enumerates all 25 directories containing Go packages under `internal/model` (including four testdata programs), each source file, each direct model-import consumer with file/line and test/tool classification, reverse transitive consumer packages, and same-package test files. Transitive sets deliberately include test edges; they do **not** assert production reach. Import classification is lexical and independent of build tags. Production reach below is rooted in actual current callers, not merely a non-`_test.go` filename.

It also scans every model/cluster Go test file and **every Rust source under wasmhost**, a conservative superset of the conformance test-module closure. This includes sibling broker tests and the helper corpus's `#[path]` common modules, rather than assuming a filename contains “corpus.” The census contains 286 source files and 966 discovered test entrypoints (518 Go). Existing Rust tests remain in place; their conversion disposition means preserve assertions while converting any Go/helper dependency, not replace unrelated Rust tests.

Each entry records its exact source symbol, declaration line, whole-file hash, disposition, role, named contract and reason. Its `source_span` runs from the declaration through EOF and is explicitly **source context, not a parsed function body**. Read the named test in that authenticated file for exact assertions; the inventory makes no per-function extraction, delegated-call or complete-assertion claim. This avoids truncating evidence at braces inside multiline strings. [tests.md](tests.md) is the per-entrypoint index. Named failures describe source contracts, not executed mutants. The fixture census retains 3,065 JSON paths, names, hashes and top-level fields, including malformed/supporting inputs.

524 path/helper references include Rust/Python/tooling consumers that Go imports cannot discover. `collect_inventory.py --check` recomputes the census without compiling or executing project code; regeneration records the actual current source revision. `verify_inventory.py` independently enumerates all wasmhost Rust files, so an omitted module fails even if it was never selected by the collector. New consumers after this revision require a fresh census before deletion.

## Retained production and generator roots

- `internal/backends/wasm/toolchain/overlay.go:18 → runtimesource`: overlay source identity and runtime files remain necessary.
- `internal/support/temporalinput/schema.go:6 → capture → iowire` (`capture/wire.go:9`) and `strictjson` (`capture/persistence.go:13`): schema capture, snapshot encoding and strict decoding remain. Capture also uses kept `internal/support/fs`.
- `Makefile` generate/validate → `cmd/protocolgen → generation/protocol`: preserve guest-boundary schemas and generated overlay codecs. `generation/protocol/generate.go` declares both retiring model targets and retained guest/Rust targets; it cannot be deleted wholesale.
- Rust `measurement/module_recipes.rs:91` and `measurement/module.rs:610` select `wasi/testdata/runtime_cost` as an actual guest program. Keep this fixture (or explicitly relocate all references); deleting its parent package is not permission to delete the fixture.
- `internal/backends/wasm/cluster/testdata/service/output_capacity_test.go::TestOutputCapacity` is **kept-Go**: an executable guest that prints output, not a reference-model assertion. `rust/gomad/src/replay/cluster/broad_guest_tests.rs:25` seals this file; `capacity_scenario` selects `-test.run=TestOutputCapacity` at line 235; `actual_output_capacity_guest_preserves_original_three_policies` at line 559 prepares and executes it. The broad harness used by cluster-provider/retained-cluster reaches these tests. Preserve the guest source and selection when retiring Go models.
- Keep `internal/archcheck`, `internal/backends/wasm/toolchain`, support and their tests. The generator's `choice/schema/choicewire*`, `schema/timewire*`, `schema/netstream_status.go.tmpl`, `schema/modelwire*`, `schema/iowire*` and runtime/overlay outputs carry current contracts. World Rust wire templates also remain generator inputs while current Rust outputs depend on them. Removing a Go package must not recursively remove live schema files in its directory.

The test/developer tools `cmd/sqlite-native` and `rust/wasmhost/src/world/deflate/testdata/gzip-oracle` import models, but are not surviving CLI model roots. Their exact oracle roles must convert before their imported models disappear. `rust/driver/sqlite_oracle.py` currently authenticates/builds the former; `world/deflate/tests.rs` executes the latter via `GOMAD_GO_GZIP_ORACLE`. Preserve SQLite evidence validation and compression/round-trip guarantees, not a live second implementation. Independent-oracle retirement is explicitly allowed by fn-249; source discovery alone does not settle downstream fn-242 reader edits.

## Package dispositions

Paths below are relative to `tools/gomad/internal/model`. Exact direct file consumers and all reverse transitive consumers are in `coverage.json.packages`; test consumers include same-package tests even when no import statement exists.

| Package | Disposition | Consumer/role and named failure to retain |
| --- | --- | --- |
| capture | kept-Go | temporalinput production capture; wasi oracle; missing/changed captured bytes, traversal, limits, persistence decoding and wire corruption |
| iowire | kept-Go | production capture/wire; volume oracle; malformed or changed captured-image wire |
| strictjson | kept-Go | production capture/persistence; cluster/wasi/world/softassert/sqlite oracle; duplicate/unknown/trailing fields or integer corruption accepted |
| runtimesource | kept-Go | toolchain/overlay; runtime semantic source or generated choice codec identity drift |
| cmd/protocolgen | kept-Go | Make generate/validate; stale or nondeterministic generated current codec |
| generation/protocol | kept-Go | protocolgen; schema/target drift or invalid generation; retire only model-only output targets |
| wasi/testdata/runtime_cost | kept-Go | Rust measurement module recipe; missing guest or changed runtime-cost workload; not a model library |
| choice | convert-to-Rust-corpus | wasi and tests; wrong tape/replay-plan acceptance, choice divergence or budget handling |
| choice/internal/wire | convert-to-Rust-corpus | choice codec and tests; changed record framing/hash; retain sibling guest schema independently |
| clock | convert-to-Rust-corpus | wasi broker/identity and cluster conformance tests; wrong deadline, advancement, overflow or admission ordering |
| cluster | convert-to-Rust-corpus | simulation testdata programs and tests; wrong lifecycle, oracle, process/network/storage projection or replay |
| cluster/internal/history | convert-to-Rust-corpus | cluster and cluster wire; reordered/misclassified or unauthenticated history |
| cluster/internal/wire | convert-to-Rust-corpus | cluster process/network/volume adapters; invalid envelope or altered model fields |
| cluster/testdata/simulation_exploration | convert-to-Rust-corpus | cluster test fixture; lost exploration selection/history behavior |
| cluster/testdata/simulation_replay | convert-to-Rust-corpus | cluster test fixture; lost exact replay/divergence behavior |
| cmd/sqlite-native | convert-to-Rust-corpus | driver/sqlite_oracle.py builds/executes reference projection; wrong durable storage projection/SQLite observation |
| network | convert-to-Rust-corpus | wasi broker, adapter and cluster; wrong streams, listener/epoch/deadline/capacity/replay outcomes |
| softassert | convert-to-Rust-corpus | wasi identity and tests; lost soft-assertion detection or attribution |
| volume | convert-to-Rust-corpus | wasi storage and tests; wrong crash cut, recovery, durable namespace/data or capacity refusal |
| volume/internal/hashwire | convert-to-Rust-corpus | volume/volume.go; altered canonical volume identity |
| wasi | convert-to-Rust-corpus | sqlite-native plus Go tests; wrong namespace, helper, broker, storage/network, execution or boundary outcomes |
| wasi/testdata/namespace_owner | convert-to-Rust-corpus | live-broker reference fixture; preserve namespace owner guest observations through linked execution |
| world | convert-to-Rust-corpus | wasi, world/process and gzip oracle; wrong events/time/identity/history/quiescence or snapshot decoding |
| world/mailbox | convert-to-Rust-corpus | same-package tests, no imported production caller; wrong mailbox admission/order/capacity |
| world/process | convert-to-Rust-corpus | same-package tests, no imported production caller; wrong process session/event transitions |

## Conformance and helper failure mapping

Each row supplements the exact per-test entries in `tests.md` and authenticated source context in `coverage.json`. All rows convert to Rust corpus tests unless a narrower disposition is stated. Existing Rust consumers remain the first reuse target; “convert” includes removing their runtime Go producer dependency while retaining assertions.

| Go source / Rust consumer | Named failure visible in checks | Required disposition |
| --- | --- | --- |
| cluster/clock_conformance_test.go / clock_conformance.rs | response/step mismatch, missing scripted/generated/recorded cases, nondeterministic generated responses | Preserve corpus replay and case coverage. `TestClockConformanceDriftPinIsCurrent` alone becomes obsolete as a live Go-hash comparison. Preserve `clock_corpus_headers_match_the_go_pin`: it checks captured header/pin integrity through its loader, quoted-participant cases and scripted/generated/recorded coverage. |
| cluster/world_conformance_test.go / world_conformance.rs | wrong World responses, snapshots/quiescence/history, final binary wire round-trip | Preserve replay and round-trip. `TestWorldConformanceDriftPinIsCurrent` alone becomes obsolete as a live Go-hash comparison. Preserve `world_corpus_headers_match_the_go_pin`: captured header/pin integrity, Unicode seed cases and scripted/generated/recorded coverage remain required. |
| choice/tape_conformance_test.go / Rust model replay_plan tests | wrong replay-plan decode/admission, forced prefix/seed/selection behavior or output bytes | Capture Go inputs/observations; retain current Rust replay-plan and malformed input assertions. |
| network/corpus{,_admission,_healed_epoch}_test.go / network_corpus.rs | serialized replay differs; mutable inputs alias corpus; listener histories lose exact replay; admission, cancellation, healed epoch or resource outcomes change | Preserve all fixture cases, repeated replay and literal guards. |
| wasi/namespace_{wire,image,transcript,host_adapter,transcript_public}_conformance_test.go / namespace image tests and conformance.rs | malformed wire/image accepted, digest/replies differ, admission order changes, output/transcript counters or expected bytes changed | Preserve typed namespace input/output/state and refusal checks; adapt stdio framing separately. |
| wasi/namespace_conformance_test.go | supporting namespace scripts (including error precedence, detached FDs, raw path bytes, offered-byte limits) | Preserve generated script cases; file has support functions rather than standalone tests. |
| wasi/volume_conformance_{core,cases,image,recovery,faults,boundary}_test.go / conformance.rs volume_conformance_corpus | core API/errno mismatch, image identity/admission/recovery mutation, wrong fault decisions, storage envelope/collection/run limits or recovered constructor | Preserve every ordered case and negative check, including tests without “Conformance” in their names. |
| volume/conformance_state_test.go and remaining volume tests | snapshots omit/change model state; wrong crash/recovery, rename/link/FD behavior, history and capacity | Convert semantic unit cases too; corpus-only filename filtering would miss these. |
| wasi/volume_transport_corpus_test.go / storage_transport/{vector,crash_failure}_corpus.rs | changed comparison inputs/outputs, callback reservation or crash failure state | Drive typed linked storage; keep state/result checks. Stdio-only framing can retire only with a per-case reason. |
| wasi/storage_selection_order_corpus_test.go / storage_transport/validation_order_corpus.rs | wrong first validation failure for crash selection | Preserve ordered refusal case expectations in Rust. |
| wasi/broker_lifecycle_corpus_test.go / cluster/handoff/broker/corpus.rs | incorrect idle/quiescent return, registration/revocation, caller capacity or epoch state | Preserve lifecycle observations and exact error class, not only error text. |
| wasi/network_adapter_corpus_test.go / cluster/network/standalone/corpus.rs | callback version/deadline or operation result/state mismatch | Preserve runtime idle callback and model bridge observations. |
| wasi/network_cluster_corpus_test.go / cluster/network/corpus.rs | wrong cluster network history, reservation/revocation and resource exhaustion | Preserve typed cluster observations. |
| wasi/corpus_assertion_preservation_test.go | missing idle return; creation capacity loses identity; identical error text collapses distinct classes; resource exhaustion changes | Keep all four explicit negative controls as Rust assertions. |
| wasi/helper_corpus_test.go plus helper_corpus_{wire,refusal,closed_refusal,residual} tests / tests/helper_corpus.rs and tests/common/*corpus* | outcome/evidence mismatch, wrong refusal stage or affected instance, discarded raw write bytes, extra frames, wrong callback IDs, unsupported inspection schema, bad max-buffer local state | Move behavioral cases to linked/linked_cluster. Preserve literal guards and producer/cross-case relations. Duplicate JSON keys/LF/key-order/extra stdio frames have no typed API equivalent: record those narrow wire-only assertions obsolete in .4 after separating mixed behavior. Currently conservatively marked conversion, not licensed for deletion. |
| wasmhost cluster/execution/{transport,transport_input,transport_trace}_corpus.rs and linked_simulation_corpus.rs | wrong execution/admission/trace projection or linked outcome | Preserve typed linked behavior; separate manual transport envelope-only checks at conversion. |
| wasmhost network/stream_abi/corpus.rs | altered guest stream ABI request/status encoding | Retain guest ABI assertions; retirement of host stdio is not retirement of guest wires. |
| wasmhost model/tests/default_cut_corpus.rs | wrong chosen storage cut/default cut observations | Preserve checked inputs and storage outcomes in Rust. |
| wasmhost world/deflate/tests.rs + Go gzip oracle | compression result/decoding or streaming behavior differs | Convert independent Go oracle cases to reviewed bytes where required by current format; retain round-trip/refusal properties. Legacy gzip-byte parity alone is obsolete under owner policy, not current snapshot integrity. |
| wasi/measure_process_test.go and other model unit tests | helper-path admission and fixture WASM inventory decoding in support functions; other named model assertions appear in the index | This file has no standalone Test functions. Preserve needed fixture-validation helpers when moving callers; helper executable selection becomes obsolete with stdio removal. |

`helper_corpus_closed_refusal_literals_test.go`, `closed_cluster_corpus_support_test.go`, network admission constructors and Rust `tests/common` files provide assertions/cases through callers. The census also includes `tests/common/{closed_cluster_guards,residual_helper_literals,profile_stderr}.rs` and `src/cluster/handoff/broker/tests.rs::broker_replays_recorded_go_orders`; these preserve raw cluster results, node/bound checks, refusal/callback projections, profile diagnostics and recorded broker orders. A support file without its own entrypoint still carries caller coverage obligations. The source inventory includes both support and test modules explicitly.

## Cycle stages and leaf commands

At this revision paths are `rust/gomad/src/cycle/plan/{declarations.rs,leaf/static_commands.rs,leaf/ordinary.rs,leaf/primary.rs,leaf/broad.rs}`. The former `internal/backends/wasm/wasmhost` spelling in the spec resolves to current `rust/wasmhost`; the declaration file is **not** under `leaf/`.

Here `$go`, `$cargo`, `$repo` and harness paths denote authenticated `LeafBindings`, not commands executed for this inventory. Common Cargo test/build arguments are `--manifest-path $repo/tools/gomad/rust/Cargo.toml --release --locked --offline --no-default-features`. Common Go tests use `-tags test_dep -count=1`. Retained package selection is `./internal/model/... ./internal/backends/wasm/cluster ./internal/backends/wasm/toolchain/... ./internal/support/...`.

| Stage(s) | Current leaf reach | Future disposition |
| --- | --- | --- |
| validate | Go protocolgen `-check`; generation/protocol tests; `-run ^TestReplayPlanConformance$ ./internal/model/choice`; Rust `--lib` replay_plan; builds conformance/reference-helper roles; Go retained set `-count=2`; Rust library twice; inventory test | Keep generator/production checks, replace Go oracle regeneration and helper builds with reviewed Rust corpus checks. |
| portable | retained Go set `-timeout=20m`, Reference flag | Reduce to kept Go; replace model behavior with Rust corpus coverage. |
| default-build, default-vet, compile-only | `go build ./...`; `go vet -tags test_dep ./...`; Go `-exec $true -run ^$ ./...`; Cargo workspace `--no-run` | Keep remaining-source gates; update expected packages/features. |
| lint-module | root Make `lint-code-gomad` with declared lint-base/fix=false/tool bindings | Keep lint on reduced Go source. |
| architecture | Go `./internal/archcheck -timeout=15m`, AllPlatforms | Keep; update retired-model/generator/fixture inventories explicitly. |
| test-harness | Go archcheck/support plus Cargo `-p gomad` | Keep retained production/capture and driver tests; update oracle path assumptions. |
| world-race | `go test -race -tags=test_dep -p=1 -parallel=1 -count=1 ./internal/model/world/...` | Remove Go implementation race gate after semantic tests map to Rust; no claim Rust corpora reproduce Go race detector coverage. |
| rust-lint | Cargo fmt + Clippy all targets over host feature matrix and helper `reference-helper`, `conformance`, diagnostic-profile, diagnostic-fuel-off, guest-profile, combinations, stress-jitter | Keep lint; remove retired features only under owning tasks. |
| rust-default-test | Cargo wasmhost `--features reference-helper`, then host tests | Replace feature/test entrypoint; preserve library and integration behavior. |
| rust-conformance-unit | Cargo wasmhost `--features conformance --lib` | Replace with native corpus library tests; preserve namespace/volume assertions. |
| rust-diagnostic-test, rust-fuel-off-test, rust-guest-profile-test, rust-combined-test | wasmhost tests with reference-helper plus diagnostic-profile / diagnostic-fuel-off / guest-profile / guest-profile,diagnostic-fuel-off | Change feature reach; preserve current profile/capacity/noninterference behavior. |
| helper-corpus-regen-1, helper-corpus-regen-2 | wasmhost `--features reference-helper --test helper_corpus`, Regenerate flag; Solo isolation | Replace with reviewed linked corpus checks/generation; stage delta must state new repeat and regeneration coverage. |
| conformance-helper-build | primary.rs Cargo build wasmhost manifest `--features conformance --target-dir ... --message-format=json` | Remove binary build once consumers converted; rewire declaration dependencies. |
| ordinary | helper list/ignored/run; explicit `conformance::tests::volume_conformance_corpus` and inode-alias check; `ordinary-namespace-{list,run}` Go `^(TestNamespaceConformance|TestVolumeConformance)`; `ordinary-recursive-{list,run}` retained Go set, skip `^TestNamespaceConformance` | Preserve actual namespace/volume/literal Rust reach; remove Go oracle rows only with mappings; retained builder/slim/workload checks remain. |
| cluster-provider, cluster-functional, retained-cluster | broad.rs `helper-lib` admitted harness (all tests), World/compressed-World probes, allocation and identity harnesses | These select Rust corpora transitively; retain behavior and refresh harness source/test inventories. |
| wasmmeasure-build, helper-build, ordinary-helper-publication | bootstrap `terminal-exit-helper-build` / `terminal-exit-helper-stage`, then ordinary discover | “helper” name alone does not mean stdio: keep linked host publication. Publication currently depends on conformance-helper-build; revise that edge explicitly. |
| workload-corpus, workload-corpus-after-relocation, sqlite-catalogue and sqlite-selection-0..3 | declared workload/SQLite adapters; current Python SQLite reader uses cmd/sqlite-native | Reconcile oracle conversion with fn-242 reader cutover; retain SQLite selection and evidence/integrity guarantees. |

`declarations.rs` attaches resources, Heavy/Reference-style stage treatment, locks, isolation, semantic subgates and repetition counts to these labels. Do not delete those contracts implicitly when replacing commands. Dependency descendants of `ordinary-helper-publication` include runtime-0..31 and runtime-aggregate, artifact/operator, storage/network/clusters, retained-durable-restart, busy-grouped/busy-boundaries/network-streams, terminal-module/host-admission/exit and workload families; `temporal-inspection` depends on helper-build. These are **dependency/linked-engine impact**, not evidence they directly execute retired Go models. Keep their current guarantees; no blanket descendant retirement.

[stages.json](stages.json) lists all 96 expanded declaration labels and dependency edges, with a disposition for direct versus dependency-only reach. It binds the declaration and leaf source hashes. This is discovery for `.5`, not the owner-accepted `stage-delta.md` and not authority to shrink fn-246 acceptance.

Other entrypoints: `Makefile` generate, validate-protocol, validate-replay-plan, validate-model-corpora, validate, test-wasm, test-rust-engine and race/lint commands carry the same roots. Keep generation and current runtime guarantees; replace the two-pass Go corpus producers and helper feature/binary invocations. `coverage.json.path_and_helper_references` records exact paths/lines, including `GOMAD_WASM_TEST_HELPER`, `GOMAD_WASM_CONFORMANCE_HELPER` and the gzip oracle.

## Verification and limits

Baseline: none; the task/spec defines no Quick suite and the dispatch forbids heavy builds/tests. Inventory verification is `python3 .flow/artifacts/fn-249/collect_inventory.py --check`, independent symbol/package coverage comparison, stage dependency closure comparison, and `git diff --check`. Numeric exits and exact source revision are retained in the task handoff. No Flow completion or review verdict is issued by this worker.

The census covers static imports, literal path references and named test functions, not dynamic runtime reach. Table-driven generated cases retain their source and committed fixture paths; function enumeration is not a count of executed cases. Downstream work must capture Go oracle outputs at its actual capture revision, preserve reviewed negative controls, and name exact Rust replacements before deleting tests. Current-source census agreement alone cannot prove behavioral equivalence or qualification.
