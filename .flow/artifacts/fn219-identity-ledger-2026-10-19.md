# fn-219 identity ledger (2026-10-19, reconciled 2026-10-20)

Promotion base: `e50c6476fb`. Historical promotion measurements below belong to
the worker candidate later audited at `28c8c33729`. Raw evidence:
`/home/agent/wt/evidence/fn219/` (`identities-compare.txt`, `pins-before.txt`,
`helper-source-digest.txt`, `helper-binary-sha.txt`, `r2-jq-filtered-corpus-diff.txt`,
`r3-cumulative-header-filtered.diff`).

Final qualification candidate: `3f532910be73f622e923273bff86c401645fe7ba`,
frozen at `/home/agent/wt/qualification-final-frozen`. Its source manifest digest
is `39182df082494be677778bdbd642b4812defef546b81189bf7a2de36d52a2fba`.
Current qualification evidence is under
`/home/agent/wt/evidence/qualification-final/qual-3f532910be/`.
The ladder is running; no final qualification pass is recorded here. R5 remains
open until every required gate has its numeric exit and final identity binding.

Continuation audit (2026-10-20): implementation candidate `28c8c33729` also
includes the recovered lint-classification and native import-order fixes from
the earlier integrated batch. Focused model/generator/archcheck tests, default
generation/validation and native generation/validation were rerun successfully;
see [the acceptance audit](fn219-acceptance-audit-2026-10-20.md). The conductor
owns the integrated candidate's WASM qualification and final identity
measurements. The pending entries below remain pending until that evidence
exists; these source checks do not substitute for it.

The generator relocation itself produced zero output-body differences
(`step1-zero-diff.txt`). Across the complete migration, stripping generated
headers leaves exactly two changed `ImplementationSourceSHA256` literals:
the shared choice codec and volume's private copy. These are the explained
choice identity rotation in this ledger. Thus the cumulative R3 diff is not
empty, even though codec behavior and every other generated body are unchanged.

## Historical promotion identities

Values come from `go run` of each package's identity function at base and at the candidate.
The choice digest is decoded from the generated literal.

| Identity | Before | After | Status | Cause |
| --- | --- | --- | --- | --- |
| Helper source digest: `wasmhost/src/world/wire_generated.rs` | `b22d7a20…de9a` | `0a0517ac…ae17` | rotated | The generated-file header names `internal/model/cmd/protocolgen` instead of native `gomadtool`. Body is byte-identical. This is the only non-test helper source change; the `PINNED_*` pins are `#[cfg(test)]`. Helper source tree digest `946583cd…ac09` → `b4442d1e…e1b7`. |
| Helper binary SHA (release, default features) | `9027efde…90ce` | `d1dd1faa…4ffe` | **rotated** (measured, reproducible) | Built in one location (`/home/agent/wt/fn219`, target `/home/agent/.cache/fn219-target/default`) with only the header comment of `wire_generated.rs` swapped. Three rebuilds of the candidate gave the same hash. The plan expected "probably unchanged": that is wrong. The official helper must be rebuilt and its `HelperSHA256` re-measured; absolute values differ by build location. |
| `volume.ImplementationSHA256` | `3df0e2bd…e486` | `7c83fb25…d154` | rotated | Projection files renamed (`*_generated.go` → `fs.go`, `handles.go`, `local_handles.go`, `volume.go`, `types.go`, `hashwire/hash.go`); generated and `Source:` headers dropped; `provenance_generated.go` deleted; `hashwire/wire_generated.go` header and choice digest literal changed. Embedded paths and bytes both enter the hash. |
| `network.ImplementationSHA256` | `1e5469e6…38e3` | `d299f192…5f84` | rotated | `generated.go` → `simulation.go` without its header; `generate/main.go` no longer hashed; `stream_status_generated.go` header; adapter package doc. |
| `world.ImplementationSHA256` | `8131a110…399e` | `93724a22…7e29` | rotated | Header of `compressed_wire_generated.go`. |
| Cluster model identity (`internal/model/cluster`) | `64a90676…60bb` | `d62d0fe6…9430` | rotated | Headers of embedded `internal/wire/{process,network,volume}.go`. The changed `process_generated_test.go` is not an implementation hash input. |
| WASM cluster identity (`internal/backends/wasm/cluster`) | `0d186559…c261` | `730aabcc…92a2` | rotated | Composes the wasi, world and cluster model identities. |
| `wasi.ImplementationSHA256` = installation `ModelSHA256` | `9b087b96…4c99` | `f9607ca9…4f29` | rotated | Composes the volume, network and world identities. **Every existing WASM installation is refused and must be reinstalled.** |
| WASM overlay `toolchain.ImplementationSHA256` | `6940d421…b079` | `4ac4238f…4ac4` | rotated | Header of `net_stream_status_generated.go.txt`. |
| WASM runtime identity and preparation BuildKey | derived | derived | rotated by construction; historical values not measured | They bind the overlay identity and the published runtime sources, including the `gomad_choicewire_generated.go.txt` and `net_stream_status_generated.go.txt` headers. Current prepared values are recorded below. |
| Choice `ImplementationSourceSHA256` (shared and native codec literal) | `de8e839b…bc82` | `669bc871…27aa` | rotated | Step 2: template header edits (templates are digest inputs) → `0a4e3995…93a9`. Step 3: `gomad-choice-implementation-source-v3` over parent-module inputs only (schema, three templates, `gomad_semantics.go.txt`, `trace.go`, `tape.go`). v2 also hashed native `runtime/gomad.go` and `go1.27.1.patch`. |
| `choice.ImplementationIdentity(buildKey)` | e.g. `900cbea3…f375` | e.g. `8dd54e14…2706` | rotated | Example values are for a zero build key. Binds the source digest above and the (rotated) BuildKey. Retained artifacts replay only under their own candidate. |
| `choice.InterruptedReplayImplementationSHA256` | `fcf851ca…` | same | unchanged | — |
| `clock.BrokerImplementationSHA256` | `8885de54…88e6` | same | unchanged | `PINNED_GO_CLOCK_SHA256` and the clock corpus are untouched. |
| `strictjson.ImplementationSHA256` | `ae4ebc32…5795` | same | unchanged | — |
| `softassert.ImplementationSHA256` | `586f591f…` | same | unchanged | — |
| Native overlay `gomadchoicewire` digest literal (native toolchain identity) | v2 digest | v3 digest | rotated | Same value as the shared literal. Native execution was not performed or requalified. fn-128 and fn-149 closed as won't-do on 2026-10-19. |
| Guest module SHA | — | — | historical comparison unavailable | No historical promotion preparation was retained. Current prepared bytes are recorded below; their equality to an unmeasured historical module is not asserted. |

## Frozen integrated candidate identities

The full historical values are retained in `identities-before.tsv`,
`identities-after.tsv` and `identities-compare.txt` under the fn-219 evidence root
above. A read-only `git diff --quiet 28c8c33729..3f532910be` over
`internal/model/{volume,network,world,clock,choice,strictjson,cluster,runtimesource}`,
`internal/evidence/record/softassert` and `internal/backends/wasm/toolchain`
finds no source differences. Those measured promotion identities still apply.
The WASI and WASM cluster sources changed in later integrated work and require
the separate bindings below.

| Binding | Value on `3f532910be` | Evidence and disposition |
| --- | --- | --- |
| Volume implementation | `sha256:7c83fb255395eade8416dbd2819cc425cd60a07b1ae86af05dee4788abafd154` | Historical measured value; complete identity inputs unchanged. |
| Network implementation | `sha256:d299f192c384e1d3949df7f21039b94f2add8f9bfe493a96dafe2c22e5b45f84` | Historical measured value; complete identity inputs unchanged. |
| World implementation | `sha256:93724a229610f7933af001fe33a9c6323431d88f2b36085f038fd4939a207e29` | Historical measured value; complete identity inputs unchanged. |
| Shared cluster implementation | `sha256:d62d0fe6ed1fe7b9efa628e1ecf18c90cfefaac4f561fb62229fa56bb95c9430` | Historical measured value; complete identity inputs unchanged, including native runtime glue retained until fn-223. |
| WASI implementation / installation model | `sha256:73c1a4f3f64dedd4c18dad47849f6dcc8bc04d08e27067dedd096acde05e2e60` | Actual early preparation. Later WASI changes include diagnostic transport and measurement plumbing, followed by fn-227 shutdown and storage-selection ordering. See the integration checkpoint and `fn227-source-acceptance-2026-10-20.md`. |
| WASM cluster implementation | pending current probe | Rotates after promotion through the changed WASI identity and embedded `cluster/retained.go` host-configuration admission. Record the actual value from the current cluster-provider gate. |
| WASM overlay implementation | `sha256:4ac4238ff22066e38a7e67356f1388bb8d09995a3fc9f2bea2c57fce8c284ac4` | Actual early provenance, matching the unchanged promotion inputs. |
| Runtime overlay identity | `sha256:6453d2a998662888534e8641ff68e6411e6adc14d5315ca7ee873b8a7ce8e506` | Actual early provenance with stock Go 1.27.1 and the frozen runtime source root. |
| Choice implementation with zero BuildKey | `8dd54e14f8a56f5c41ec71eec5417dd233eaaf217c77e0f3c4af690a243f2706` | Historical measured probe; unchanged choice source inputs. This is a characterization input, not a prepared target's execution identity. |
| Helper source inventory digest | `1fe667637f277a0bd90a1c88a1095664a4646f0c7d7a8d2893982f32736f86f7` | Current `qual-3f532910be/helper-source-files.sha256`. The matching historical inventory reconstructed at `28c8c33729` is `helper-source-at-fn219.sha256`, digest `b4442d1e83677ff0ab4d8b798012fb0cf9473dd5a4c05041838d5f825331e1b7`. Later fn-188 helper ABI, event-order, diagnostics, stress-jitter and cluster-handoff changes rotate the source inventory beyond fn-219's generated-header change. |
| Early-preparation helper binary | `sha256:8956496039ed213d2c1d0e88266bb2f7a9f2c9916e299ae8b734387f2a6dbe5f` | `early-onebox-3f532910be/prepare-only-producer.json`, sibling of the qualification root. This exact helper bound the preparation below. |
| Admitted measurement helper binary | `sha256:b25439140f0c850a82f99471c9fd48ac864569a75e407a76850873f6ece055a1` | `qual-3f532910be/state/helper.env` names the executable and its build/admission receipts. Same frozen sources, distinct build location from the early helper. Final workload and runtime reports must name their actual helper. |
| Early prepared one-box module | `sha256:4e19470a4f6f68083f6bea28d3ff61d7ae7317d2799fdde7185e37e42506da78` | `early-onebox-3f532910be/prepare-only-producer.json`, selected `./tests` smoke workload with the four qualified Temporal guest tags. Preparation alone supplies no execution or replay pass. |
| Early prepared one-box BuildKey | `e6f427eda85c548890f7b084fadb53b33ef8fa92b7e49cf6ece42347de4bae81` | Same producer. Binds the early helper, module source, arguments, model, runtime and configuration. Final workload preparation with the admitted helper requires its own actual BuildKey. |

The early producer's encoded backend provenance records the runtime values
above. Its compiler digest is
`sha256:1675694ef690db0f18fbe7046a886170904bede1d9db6ec96ae27945c1705c64`.
Its engine is Wasmtime `47.0.3`, configuration
`cranelift-fuel-nan-canonical-nounwind-v2`, on `linux/aarch64`; the profile is
`gomad.wasi-cooperative/v1`. Clock, strict JSON, interrupted replay and
soft-assertion identities retain their full unchanged values in the historical
TSV measurements. Qualification remains pending for this integrated source.

## Pins and corpora regenerated (unchanged expectations)

Regenerated with `GOMAD_NETWORK_CORPUS_REGENERATE`, `GOMAD_NETWORK_ADAPTER_CORPUS_REGENERATE`,
`GOMAD_NETWORK_CLUSTER_CORPUS_REGENERATE`, `GOMAD_BROKER_LIFECYCLE_CORPUS_REGENERATE` and
`GOMAD_WORLD_CONFORMANCE_REGENERATE`. Two pins have no regenerator and were set by hand to the
new identity: `PINNED_GO_NETWORK_SHA256` and the `wasi/identity_test.go` literal.

- 375 files: `network_v2` 183, `network_cluster_cases` 51, `network_adapter_cases` 62,
  `broker_lifecycle_cases` 10, `world_conformance` 66, three `PINNED_GO_*` files and one test
  literal.
- **jq proof:** removing every key matching `implementation_sha256$`
  (`walk(if type=="object" then with_entries(select(.key|test("implementation_sha256$")|not)) else . end)`)
  leaves all 371 JSON files identical to base. 0 diffs.
- **Byte proof:** substituting each rotated identity (old → new) into the base bytes
  reproduces all 375 files exactly.
- After regeneration, these Go currency tests exit 0 (`t4-pins-after-regeneration.log`):
  `TestNetworkCorpusIsCurrent`, the adapter, cluster, broker-lifecycle, transport, broker-order,
  World and clock currency and drift tests, and the wasi model identity literal.
- Rust differential tests exit 0 with `-j 2`, release (`r2-rust-corpora.log`):
  `network_corpus` (16), `world_conformance` (8), `clock_conformance` (4), and the bin unit
  tests (138 passed, 2 ignored), including the cluster network corpus pins.
- Volume and namespace differential conformance with the conformance helper required: exit 0
  (`t1-test-conformance.log`, run after fn-219.1).

## Requalification gates (current run pending)

The conductor is running the current 96-stage ladder under
`qual-3f532910be`. Its `stages.log`, per-stage `.command` files and
`.receipt.json` files record actual commands and numeric exits. The table below
retains the requested gate coverage; final reconciliation must map every row to
the completed current stages and their actual helper/preparation identities.
An intermediate stage pass does not complete R5.

Run from the repository root of the frozen candidate with the official stock Go 1.27.1
(`GO=<stock go1.27.1>`), after `make -C tools/gomad/internal/backends/wasm helper`. Record each
numeric exit here.

| Gate | Command | Exit |
| --- | --- | --- |
| Helper corpora (4 builds) | `make -C tools/gomad/internal/backends/wasm test-helper` | pending |
| Conformance (fresh helper) | `make -C tools/gomad/internal/backends/wasm test-conformance GOMAD_WASM_STOCK_GO=$GO` | pending |
| WASM suite | `make -C tools/gomad test-wasm GOMAD_STOCK_GO=$GO` | pending |
| Retained inspection | `cd tools/gomad && GOMAD_STOCK_GO=$GO GOMAD_WASM_TEMPORAL_INSPECTION_ROOT=<retained dir> go test -tags test_dep -count=1 -timeout=120m -run '^TestTemporalInspectionRetainedFailure$' ./internal/backends/wasm/backend` | pending |
| Operator typed scenarios | `cd tools/gomad && GOMAD_STOCK_GO=$GO GOMAD_WASM_TEST_HELPER=<release helper> go test -tags test_dep -count=1 -timeout=60m -run '^TestInstalledOperatorTypedScenario(DetachedReplay\|KillResume)$' ./internal/backends/wasm/command` | pending |
| Runtime qualification | `GOMAD_WASM_QUALIFICATION_ROOT=<root> GOMAD_WASM_QUALIFICATION_MANIFEST=<manifest> make -C tools/gomad/internal/backends/wasm qualify-runtime GOMAD_WASM_STOCK_GO=$GO` | pending |
| Workload qualification (mandatory, after disk cleanup) | `GOMAD_WASM_QUALIFICATION_ROOT=… GOMAD_WASM_INTEGRATED_SOURCE_MANIFEST=… GOMAD_WASM_INTEGRATED_SOURCE_SHA256=… GOMAD_WASM_TEMPORAL_QUALIFICATION_INPUTS=… GOMAD_WASM_TEMPORAL_PREPARATION_SOURCE=… make -C tools/gomad/internal/backends/wasm qualify-workloads GOMAD_WASM_STOCK_GO=$GO` | pending |
| Measurements to add | Official helper `HelperSHA256`, guest module SHA, runtime identity, BuildKey | pending |
