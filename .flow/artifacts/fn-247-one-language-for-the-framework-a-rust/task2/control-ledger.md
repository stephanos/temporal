# Native identity control ownership

The owner says “feel free to make breaking changes from pytjon driver” and “we don't need backwards compatabiltiy” (2026-10-24; parent spec Conversation Evidence). These native controls retain behavior rather than Python bytes. Integration target: `cargo test -p xtask --test guest_identity --locked`.

| Retained Python control | Native Rust control in tests/identity |
| --- | --- |
| lock: unrelated workspace packages/dev exclusions | lock::unrelated_and_dev_only_dependencies_do_not_change_guest_closures |
| lock: workspace unified optional edges | lock::unified_edges_and_edge_topology_change_ancestor_identity |
| lock: checkout portability | lock::content_addressed_publication_is_repeatable_and_portable |
| lock: checksum propagation to every dependent | lock::checksum_changes_propagate_through_normal_and_all_platform_build_edges |
| lock: all-platform build dependencies | same table-driven checksum/edge control |
| lock: edge topology | lock::unified_edges_and_edge_topology_change_ancestor_identity |
| lock: ambiguous name/version sources | lock::ambiguous_name_version_coordinates_are_marked_unavailable |
| lock: missing sourced lock entry / git checksum optional | lock::sourced_entries_are_required_but_git_checksums_are_optional |
| lock: CLI publication / malformed document | lock::content_addressed_publication_is_repeatable_and_portable; lock::malformed_native_documents_and_graphs_are_rejected; actual CLI metadata evidence |
| rustc: portable host metadata/path identity | compiler::portable_identity_replaces_guest_metadata_and_remaps_generated_paths |
| rustc: dependency identity propagation | compiler::dependency_identity_and_transitive_variants_propagate |
| rustc: package/version/features/profile/compiler/lock distinctions | compiler::package_features_profile_compiler_and_closure_remain_distinct |
| rustc: missing/ambiguous closure | compiler::missing_ambiguous_and_malformed_lock_maps_refuse |
| rustc: build-std exemption and refusals | compiler::build_std_requires_wasm_marker_and_trusted_source_or_rust_src_coordinate |
| rustc: missing/corrupt sidecar | compiler::invalid_or_missing_dependency_sidecars_and_macro_collisions_refuse |
| rustc: portable native macro without argv rewriting | compiler::native_macro_identity_ignores_host_configuration_but_retains_argv_and_collisions |
| rustc: native macro collision | compiler::invalid_or_missing_dependency_sidecars_and_macro_collisions_refuse |
| rustc: probe passthrough | compiler::probes_pass_through_without_guest_environment |
| rustc: compiler failure cleanup | compiler::sidecar_precedes_child_and_cleanup_preserves_exit_launch_and_signal_outcomes |
| rustc: pipelined metadata availability | same lifecycle control; native compiler fixture reads the sidecar before exit |
| rustc: inherited jobserver FD | compiler::inherited_jobserver_descriptor_reaches_direct_compiler_child |

Additional native controls cover raw OS argv/environment bytes and lexical root boundaries; previous-wrapper ordering and loader environment; launch failure and signal status; actual dependency-free rustc proc-macro compilation in two checkout roots; malformed options/required inputs; sorted JSON and a known SHA-256 vector; concurrent atomic publication, rename-failure temporary cleanup, and serialized sidecar cleanup; current Cargo/mise/toolchain/rust-src TOML.

Explicitly retired under the owner amendment: Python import fixtures/interpreter launchers, shebang copying and Python-executable whitespace restrictions, Python exception names/text, json.dumps spaces/escaping/order, exact old hashes and schema-1 JSON. Native JSON is schema 2; serde_json/toml own grammar/errors/resource limits. No Python BigInt, date, NaN, recursion, digit, Unicode or diagnostic-order emulation is recreated. Paths with spaces and non-UTF-8 Unix bytes remain useful controls.

The retained Python helpers are unchanged and still serve current shell callers. Task .5 replaces their active invocation and invalidates schema-1 caches. Final task .3 wires the canonical workspace stage once and owns full integrated guest reproduction/freshness plus TLC/Apalache validation. No frozen artifact or guest bytes changed here. Linux evidence does not claim native macOS execution; loader bytes are exercised on the available host.
