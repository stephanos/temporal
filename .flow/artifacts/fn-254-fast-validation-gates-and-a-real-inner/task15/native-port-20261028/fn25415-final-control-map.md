# Task15 final source control map

Source HEAD: `4d980c9acdb6654b7d460e0aaf87b92aa14709a4`. Runtime acceptance: **false**. Current joined working bytes, including root WIP, are fingerprinted in the JSON.

The frozen inventory contains29 shell scripts:27 removable scripts and2 retained bootstraps. All31 explicit shell cases and47 shared vectors are accounted for. No original shell control is left unmapped; the shell-substitution and external-grep pipeline cases have explicit moot reasons. No scripts were deleted and no runtime checks were run by this lane.

## Coverage amendments

- Temporal spec/generator composition is now covered by `proto::bridge_controls::temporal_spec_composes_with_native_generation_freshness_and_refusals` (integrated4fd176c256). Latest runtime remains pending.
- Guest argument refusal now has root’s `guest::tests::invalid_guest_arguments_are_refused_before_tool_discovery` five-vector inline control. Latest runtime remains pending.
- Temporal empty target now appears in its path control. Root reports a RED failure, filter-empty production correction and GREEN pass; this lane only inspected source and records that report without treating it as final acceptance.

## Script replacements

| Script | Disposition | Replacement |
| --- | --- | --- |
| `model/scripts/fetch-tlc.sh` | remove after runtime acceptance | cargo xtask fetch-tlc [framework-root] |
| `model/scripts/guest-freshness.sh` | remove after runtime acceptance | cargo xtask guest-freshness PACKAGE OUTPUT / umpire-guest-driver guest-freshness |
| `model/scripts/guest.sh` | remove after runtime acceptance | cargo xtask guest PACKAGE OUTPUT [--check] / umpire-guest-driver guest |
| `model/scripts/proto-go.sh` | remove after runtime acceptance | cargo xtask proto-go temporal [--check] |
| `model/scripts/retired-names.sh` | remove after runtime acceptance | cargo xtask retired-names |
| `model/scripts/test-capabilities.sh` | remove after runtime acceptance | cargo xtask capability-controls / native_leaves::capability_controls_enforce_core_and_unsafe_policies |
| `model/scripts/test-model-validate.sh` | remove after runtime acceptance | model_controls::shipping_model_recipes_with_disposable_children |
| `model/scripts/test-model-wiring.sh` | remove after runtime acceptance | model_controls::shipping_model_recipes_with_disposable_children |
| `model/scripts/test-time-gate.sh` | remove after runtime acceptance | cargo xtask timing-controls / native registered test family |
| `model/scripts/tests/build-guests.sh` | remove after runtime acceptance | cargo xtask build-guests |
| `model/scripts/tests/guest-roots.sh` | remove after runtime acceptance | guest_routes::tests seven-package/two-entry process controls |
| `model/scripts/tests/proto-go.sh` | remove after runtime acceptance | cargo xtask proto-go temporal [--check] -> native proto controls |
| `model/scripts/time-gate.sh` | remove after runtime acceptance | cargo xtask time / make model-time / make model-validate |
| `model/scripts/xtask.sh` | retain | retain minimal bootstrap |
| `model/umpire/scripts/fetch-tlc.sh` | remove after runtime acceptance | cargo xtask fetch-tlc [framework-root] |
| `model/umpire/scripts/guest-freshness.sh` | remove after runtime acceptance | cargo xtask guest-freshness PACKAGE OUTPUT / umpire-guest-driver guest-freshness |
| `model/umpire/scripts/guest.sh` | remove after runtime acceptance | cargo xtask guest PACKAGE OUTPUT [--check] / umpire-guest-driver guest |
| `model/umpire/scripts/proto-go.sh` | remove after runtime acceptance | cargo xtask proto-go core\|kettle [--check] |
| `model/umpire/scripts/proto-tools.sh` | remove after runtime acceptance | cargo xtask proto-tools |
| `model/umpire/scripts/retired-names.sh` | remove after runtime acceptance | cargo xtask retired-names PATTERN (NUL-record stdin); framework gate native retired scanner |
| `model/umpire/scripts/test-capabilities.sh` | remove after runtime acceptance | cargo xtask capability-controls / native_leaves::capability_controls_enforce_core_and_unsafe_policies |
| `model/umpire/scripts/test-time-gate.sh` | remove after runtime acceptance | cargo xtask timing-controls / native registered test family |
| `model/umpire/scripts/tests/bootstrap.sh` | remove after runtime acceptance | guest_pipeline retained bootstrap/process controls |
| `model/umpire/scripts/tests/build-guests-controls.sh` | remove after runtime acceptance | build_guests::path_controls and native seven-job failure controls |
| `model/umpire/scripts/tests/build-guests.sh` | remove after runtime acceptance | cargo xtask build-guests |
| `model/umpire/scripts/tests/proto-go.sh` | remove after runtime acceptance | cargo xtask proto-go core\|kettle [--check] -> native proto controls |
| `model/umpire/scripts/tests/proto-tools.sh` | remove after runtime acceptance | cargo xtask proto-tools |
| `model/umpire/scripts/time-gate.sh` | remove after runtime acceptance | cargo xtask time --step NAME PROGRAM [ARG...] |
| `model/umpire/scripts/xtask.sh` | retain | retain minimal bootstrap |

## Exact31-case mapping

| Frozen case | Native source controls | Disposition |
| --- | --- | --- |
| `validate.backend_precedence` | `model_controls::shipping_model_recipes_with_disposable_children`; `gate::timing::tests::source_edit_restores_original_content_after_failed_child` | source_control_present_runtime_pending |
| `time.backend_preservation` | `model_controls::shipping_model_recipes_with_disposable_children`; `gate::timing::tests::source_edit_restores_original_content_after_failed_child` | source_control_present_runtime_pending |
| `validate.gate_fail` | `model_controls::shipping_model_recipes_with_disposable_children`; `gate::timing::tests::source_edit_restores_original_content_after_failed_child` | source_control_present_runtime_pending |
| `validate.benchmark_fail` | `model_controls::shipping_model_recipes_with_disposable_children`; `gate::timing::tests::source_edit_restores_original_content_after_failed_child` | source_control_present_runtime_pending |
| `time.raw_benchmark_fail` | `model_controls::shipping_model_recipes_with_disposable_children`; `gate::timing::tests::source_edit_restores_original_content_after_failed_child` | source_control_present_runtime_pending |
| `time.bootstrap_fail` | `model_controls::shipping_model_recipes_with_disposable_children`; `gate::timing::tests::source_edit_restores_original_content_after_failed_child` | source_control_present_runtime_pending |
| `timer.environment` | `model_controls::shipping_model_recipes_with_disposable_children`; `gate::timing::tests::source_edit_restores_original_content_after_failed_child` | source_control_present_runtime_pending |
| `timer.bootstrap_resources` | `model_controls::shipping_model_recipes_with_disposable_children`; `gate::timing::tests::source_edit_restores_original_content_after_failed_child` | source_control_present_runtime_pending |
| `timer.cleanup_and_soft_warnings` | `model_controls::shipping_model_recipes_with_disposable_children`; `gate::timing::tests::source_edit_restores_original_content_after_failed_child` | source_control_present_runtime_pending |
| `wiring.ci_minimum` | `model_controls::shipping_model_recipes_with_disposable_children`; `gate::timing::tests::source_edit_restores_original_content_after_failed_child` | source_control_present_runtime_pending |
| `wiring.combined_gate_once` | `model_controls::shipping_model_recipes_with_disposable_children`; `gate::timing::tests::source_edit_restores_original_content_after_failed_child` | source_control_present_runtime_pending |
| `wiring.backend_mode_once` | `model_controls::shipping_model_recipes_with_disposable_children`; `gate::timing::tests::source_edit_restores_original_content_after_failed_child` | source_control_present_runtime_pending |
| `wiring.quick_once` | `model_controls::shipping_model_recipes_with_disposable_children`; `gate::timing::tests::source_edit_restores_original_content_after_failed_child` | source_control_present_runtime_pending |
| `wiring.bootstrap_jobs` | `model_controls::shipping_model_recipes_with_disposable_children`; `gate::timing::tests::source_edit_restores_original_content_after_failed_child` | source_control_present_runtime_pending |
| `wiring.producer_prerequisites` | `model_controls::shipping_model_recipes_with_disposable_children`; `gate::timing::tests::source_edit_restores_original_content_after_failed_child` | source_control_present_runtime_pending |
| `wiring.gate_fail` | `model_controls::shipping_model_recipes_with_disposable_children`; `gate::timing::tests::source_edit_restores_original_content_after_failed_child` | source_control_present_runtime_pending |
| `wiring.fmt_update_order` | `model_controls::shipping_model_recipes_with_disposable_children`; `gate::timing::tests::source_edit_restores_original_content_after_failed_child` | source_control_present_runtime_pending |
| `wiring.framework_fail_stops` | `model_controls::shipping_model_recipes_with_disposable_children`; `gate::timing::tests::source_edit_restores_original_content_after_failed_child` | source_control_present_runtime_pending |
| `wiring.paths_and_auto_install` | `model_controls::shipping_model_recipes_with_disposable_children`; `gate::timing::tests::source_edit_restores_original_content_after_failed_child` | source_control_present_runtime_pending |
| `warnings.default_equal` | `gate::timing_controls::tests::soft_budget_defaults_equality_overrides_and_warning_order` | source_control_present_runtime_pending |
| `warnings.inner_over` | `gate::timing_controls::tests::soft_budget_defaults_equality_overrides_and_warning_order` | source_control_present_runtime_pending |
| `warnings.gate_over` | `gate::timing_controls::tests::soft_budget_defaults_equality_overrides_and_warning_order` | source_control_present_runtime_pending |
| `warnings.override_equal` | `gate::timing_controls::tests::soft_budget_defaults_equality_overrides_and_warning_order` | source_control_present_runtime_pending |
| `warnings.override_both_over` | `gate::timing_controls::tests::soft_budget_defaults_equality_overrides_and_warning_order` | source_control_present_runtime_pending |
| `warnings.transitive_controls` | `gate::timing_controls::tests::soft_budget_defaults_equality_overrides_and_warning_order`; `model_controls::shipping_model_recipes_with_disposable_children`; `gate::timing::tests::source_edit_restores_original_content_after_failed_child` | source_control_present_runtime_pending |
| `step.status` | `native_leaves::public_timing_front_door_preserves_status_diagnostics_and_cleanup` | source_control_present_runtime_pending |
| `step.diagnostics.success` | `native_leaves::public_timing_front_door_preserves_status_diagnostics_and_cleanup` | source_control_present_runtime_pending |
| `step.diagnostics.failure` | `native_leaves::public_timing_front_door_preserves_status_diagnostics_and_cleanup` | source_control_present_runtime_pending |
| `step.cwd` | `native_leaves::public_timing_front_door_preserves_status_diagnostics_and_cleanup` | source_control_present_runtime_pending |
| `step.frontdoor37` | `native_leaves::public_timing_front_door_preserves_status_diagnostics_and_cleanup` | source_control_present_runtime_pending |
| `step.shell_substitution` | `native_leaves::public_timing_front_door_preserves_status_diagnostics_and_cleanup` | obsolete_shell_composition |

## Maintained callers

No unexpected maintained caller was found. The only3 references outside the replaced script bodies and historical/design/archive roots (including hidden maintained files such as `.github`) remain in `MIGRATE.md:227`, `:326` and `:327`. Its header explicitly defers extraction and preserves earlier proof checkpoints; docs handover explicitly preserves this deferred migration record. The JSON records raw results and exclusions, including these references.

## Explicitly moot or changed expectations

- step.shell_substitution: Removed Bash wrapper has no command substitution; public native37/143 contracts remain.
- external grep absent/failure9/ERE child errors and Git-grep SIGPIPE precedence: Rust byte regex and completed Git ls-files inventory remove the grep child and producer/filter pipeline; Git failures, malformed inventories, native I/O and invalid patterns remain controlled.
- old timer repository cwd/relative Cargo strings at final benchmark boundary: Owner-approved framework test-run cwd and admitted caller Cargo paths; Make caller strings remain separately preserved.
- old update cache flags always unset: Owner-approved inherited flags correction; native Make fixture requires inherited1:0.
- mise presence/version probes removed in task14: Task15 must not restore retired preflight probes; native generator pins and source CI mise floor checks remain.

## Runtime boundary

These source mappings preserve task17’s scheduler, nextest inventory/count authority, leases/cancellation and owned cleanup, and task16’s explicit mise provider. Native fake-child controls do not establish live Model execution. Root owns latest bridge/argument/lint checks, full changed gate, script deletion and final acceptance. Re-run both inventories and refresh this map’s HEAD/hashes after deletion and any final edits.

The JSON includes per-script frozen identities/current hashes, actual Rust test identities and line anchors, all47 per-vector source mappings, all31 exact case mappings, handover fingerprints and remaining root steps.
