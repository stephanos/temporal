# Original measurement module recipe reconciliation

Source-only checkpoint. Reference: immutable Git commit `1d695061b6382009d0659238c652a5a765b1d5f5`. No Cargo, Go, guest, installation or qualification run is authorized by this audit. The new constructor tests are written and formatted but unexecuted under the current disk restriction. Central `measurement/module.rs` wiring remains with the measurement owner; these constructors are not yet registered in `measurement/mod.rs`.

The six CLI roots in `internal/backends/wasm/cmd/wasmmeasure/main_test.go` test output failure, checkout refusal, keep propagation, selected-checkout producer build, required-input refusal and flag propagation. They are not six guest compilation recipes. Their production/API parity remains with the measurement input-contract owner. The external retained map `/home/agent/evidence/fn232-test-wasm-reach/measurement-input-contract-map.json` lists 25 producer names; the following audit distinguishes their compilation behavior instead of assigning all names to a workflow compilation.

| Original producer | Original module source/custody |
| --- | --- |
| stress | Dedicated stress workload registry; stress owner retains its exact runtime_probes/Temporal recipes. |
| cluster-owner-profile | `temporalRecoveryRequestWithAdmission("temporal_broker")`, derived full three-tag test fixture. |
| compiled-identity-owners | Compiled owner identities only; no guest module compilation. |
| diagnostic-helper-build | Helper Cargo build only; no Go guest module compilation. |
| diagnostic-cluster-baseline | Untagged choices feature-probe build plus derived temporal_broker test fixture. |
| failing-suite-profile | Required external matched module, cache and runtime identity. |
| guest-profile | `diagnosticAdmission.buildModule`, full `./tests`, three tags. |
| guest-variants | Same full `./tests`, separate empty and `satconv,signext` GOWASM variants. |
| host-call-profile | Optional external module; otherwise full three-tag `./tests` through prototypeBuild. |
| host-owner-profile | Optional external module; otherwise explicit **four-tag slim** `./tests` from host_owner.go. |
| legacy-workflow-profile | Required external matched module, cache and runtime identity. |
| matched-release-candidate | Derived temporal_broker fixture through temporalRecoveryRequestWithAdmission. |
| matched-suite | Required external matched module; suites choose execution configuration, not a replacement compiler target. |
| migration | Required retained choices/runtime_probes/time_probes modules and their separate identities; no compilation in measureMigration. |
| namespace-activity | Required external module and terminal module preparation receipt; later requests select activity. |
| namespace-sample | Parent common request/module custody; no new guest compilation. |
| operation-retention | Required external matched module, cache and runtime identity. |
| runtime-cost-profile | Cooperative untagged runtime_cost build; original request seed7, 180s. |
| scratch-smoke-profile | Full three-tag `./tests` compiled only when its retained module is absent. |
| terminal-exit | Required external module and actual TestPrepareTemporalTerminalExitModule receipt. |
| terminal-exit-helper-build | Helper Cargo build only. |
| terminal-exit-helper-stage | Existing helper staging only. |
| warm-preparation | Required external module and terminal preparation receipt. |
| warm-preparation-helper-build | Helper Cargo build only. |
| warm-preparation-prime | Required external module and terminal preparation receipt. |

The audited source-function locations and compile-call paths are retained at `/home/agent/wt/evidence/fn232-original-module-recipe-audit.json`. The direct call-path extraction is a navigation aid, not an execution proof or a complete closure argument for delegated stress and cluster adapters.

## Constructor seam

`module_recipes::ModuleRecipe` carries only kind, source operand, working directory, build tags and target arguments. `temporal_test` uses original three tags `[disable_grpc_modules,test_dep,sqlite3_dotlk]`; `derived_temporal_test` preserves original recovery order `[test_dep,disable_grpc_modules,sqlite3_dotlk]`; `slim_host_owner` alone retains `[disable_grpc_modules,gomad_wasm_slim,sqlite3_dotlk,test_dep]`. `runtime_cost` and `diagnostic_probe` retain their untagged build operands. Native preparation may normalize the build-tag set for its canonical provenance; the original command helpers remain independently tested and retain their exact grammar. Compiler controls and source custody are still enforced by the production preparation boundary.

`terminal_exit(repo)` returns `PreparationRecipe`: original `./tests` go-test, no target arguments, memory4GiB, fuel2^50, cooperative profile and clock step0. Its inherited integrationProvider configuration has nil Args; PWD=/workspace; writable /workspace and /tmp; zero entropy; epoch946684800000000000; output1MiB, transcript8MiB, calls10000, descriptors64, files1000, filesystem8MiB, pending4096. This is the preparation tuple, not the later Temporal execution request. The central CLI constructor must derive its output bound from this configuration; hardcoded256MiB would alter the tuple.

`temporal_request` independently builds the original prototype configuration through the existing schema-capture policy: Temporal argv prefix `temporal.test,-test.v,-test.count=1,-test.timeout=72h`, then the actual selection/persistence args; no installation-only network; original environment, entropy and complete capture. Its request bounds are memory4GiB/fuel2^50, output256MiB/transcript1GiB/calls2^28/descriptors4096/files100000/filesystem512MiB/pending4096, **choice1GiB**, and the caller's original timeout (terminal/warm30min, legacy180s, failing300s). Seeds, roles and replay policy remain in the owning producer's explicit runtime request and must still be bound. The request recipe explicitly carries diagnostics=false and digest_transcript=true; individual producer overrides remain explicit.

`diagnostic_probe_request` retains unmodified cooperativeModuleRequest: memory256MiB, fuel4000000000, timeout60s, choice1MiB, diagnostics=true, digest_transcript=false, standalone args/env/entropy, clockstep1000 and integration-sized limits. `runtime_request` derives its explicit overrides from that request and retains runtime-cost's standalone config/args/env/entropy, memory256MiB/fuel2^40, output1MiB/transcript64MiB/calls2^20/descriptors64/files1000/filesystem8MiB/pending4096, choice64MiB and180s. The existing workload plan retains original seed7. Runtime-probes stress60s is a different recipe and remains with its owner.

Byte-exact source relocation was checked without compilation: original `internal/backends/wasm/wasi/testdata/runtime_cost/main.go` equals current `internal/model/wasi/testdata/runtime_cost/main.go` (368bytes, SHA256 `81a802dcdbbdad7df57c457537396cfd36e8cac4982817a48ef8421c611b362d`); the choices feature probe is unchanged (455bytes, SHA256 `8871b9b6a283b71bc2564e2430e59edde3885e0ac0dec319b96c25eeed4a9737`). Raw comparison table: `/home/agent/wt/evidence/fn232-module-recipe-source-bindings.json`. Neither source equality nor constructor tests establish a final compiled-module or guest proof.

Eight source tests cover the exact terminal tuple, full/derived tag separation, mapped runtime/probe operands, runtime request bounds, complete Temporal prototype configuration through native admission, configuration refusal/admission and the host-owner slim exception. Wiring, actual source-frozen RED/GREEN execution, independent raw receiving checks and all original behavioral obligations remain required; no original test root or final qualification is marked PASS here.
