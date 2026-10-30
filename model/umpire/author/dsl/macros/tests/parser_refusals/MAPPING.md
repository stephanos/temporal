# Parser refusal conversion

Two parser controls validate all 61 table rows against the complete ordered diagnostic
messages and authored locations. The 18 retained trybuild cases keep their original source
and stderr bytes. The complete fixture binary passed all eight test identities, including
all 18 negative and four positive compiler inputs, in 41.383 seconds. No diagnostics were
re-accepted; the original 79 negative inputs remain partitioned into 61 parser and 18
compiler cases.

Each migrated source and its existing stderr are moved byte-for-byte into `inputs/`.
The stderr files are retained provenance, not newly accepted goldens. The table asserts the
complete ordered message/location vector in the existing `fixtures` integration binary.
All source locations below remain coordinates in the complete original authored source.

| Old `tests/ui` source | Table row in `cases.rs` | Entry | Preserved diagnostics |
| --- | --- | --- | --- |
| `action_domain_conflict.rs` | `action_domain_conflict` | `action::expand` | User::Set field `0` declares both #[inputs] and #[domain]; declare one domain (5:9) |
| `action_domain_duplicate.rs` | `action_domain_duplicate` | `action::expand` | User::Set field `0` has duplicate #[domain] declarations (5:26) |
| `action_domain_variant.rs` | `action_domain_variant` | `action::expand` | #[domain] belongs on one variant field, not the variant (5:5) |
| `action_domain_without_world.rs` | `action_domain_without_world` | `action::expand` | User::Set field `0` #[domain] requires the World of a #[machine] module that declares #[derive(World)] (9:9) |
| `action_inputs_duplicate.rs` | `action_inputs_duplicate` | `action::expand` | User::Set field `value` has duplicate #[inputs] domains (7:9) |
| `action_inputs_invalid.rs` | `action_inputs_invalid` | `action::expand` | User::Set field `value` #[inputs] requires a bounded range or finite list, such as 0..=3 or [false, true] (6:9) |
| `action_inputs_missing.rs` | `action_inputs_missing` | `action::expand` | User::Set field `value` requires #[inputs] with a bounded range or finite list (5:11) |
| `action_inputs_variant_fault.rs` | `action_inputs_variant_fault` | `action::expand` | #[fault] belongs on the action enum, not a variant (5:5) |
| `action_metadata_duplicate.rs` | `action_metadata_duplicate` | `action::expand` | duplicate action metadata namespace (6:16) |
| `action_metadata_namespace.rs` | `action_metadata_namespace` | `action::expand` | expected parentheses (5:24) |
| `action_metadata_placement.rs` | `action_metadata_placement` | `action::expand` | #[metadata] belongs on an action variant (4:1); #[metadata] belongs on an action variant (6:12) |
| `finite_shape.rs` | `finite_shape` | `finite::expand` | Finite derives for a fieldless enum or a struct with one #[inputs(lo..=hi)] integer field; give a variant with fields to derive(Action) (5:5); range bound 300 does not fit the field's integer type (9:26); a Finite newtype's field must be a primitive integer type (12:30); Finite derives for a fieldless enum or a struct with one #[inputs(lo..=hi)] integer field (15:14) |
| `machine_actor_arity.rs` | `machine_actor_arity` | `machine::expand` | an #[on] method takes its actor enum: fn user(&mut self, action: User) (13:9) |
| `machine_actor_mismatch.rs` | `machine_actor_mismatch` | `machine::expand` | #[on(User)] requires a User parameter (16:9) |
| `machine_actor_receiver.rs` | `machine_actor_receiver` | `machine::expand` | an #[on] method takes its actor enum: fn user(&mut self, action: User) (13:9) |
| `machine_after_inputs.rs` | `machine_after_inputs` | `machine::expand` | `User::Set` carries inputs; it is not one catalog action (15:33) |
| `machine_claim_async.rs` | `machine_claim_async` | `machine::expand` | invariants and postconditions must be synchronous (16:9) |
| `machine_claim_returns.rs` | `machine_claim_returns` | `machine::expand` | invariants and postconditions return nothing; use check! (16:25) |
| `machine_duplicate_actor.rs` | `machine_duplicate_actor` | `machine::expand` | actor User already has an #[on] method (15:12) |
| `machine_duplicate_kind.rs` | `machine_duplicate_kind` | `machine::expand` | a method requires exactly one machine kind attribute (16:9) |
| `machine_effect_arguments.rs` | `machine_effect_arguments` | `machine::expand` | unexpected token in attribute (15:17) |
| `machine_generic.rs` | `machine_generic` | `machine::expand` | #[machine] does not admit generic Model, World or actor declarations (7:17) |
| `machine_handler_async.rs` | `machine_handler_async` | `machine::expand` | #[on] methods must be synchronous (13:9) |
| `machine_handler_returns.rs` | `machine_handler_returns` | `machine::expand` | #[on] methods return nothing; enabledness is a changed state (13:42) |
| `machine_invariant_parameters.rs` | `machine_invariant_parameters` | `machine::expand` | an invariant takes only &self and an optional trailing World (15:9) |
| `machine_legacy_effects.rs` | `machine_legacy_effects` | `machine::expand` | use per-item #[on(Actor)], #[effect], #[invariant], #[postcondition], #[assumption], and #[progress] instead of #[effects] (11:5) |
| `machine_legacy_invariants.rs` | `machine_legacy_invariants` | `machine::expand` | use per-item #[on(Actor)], #[effect], #[invariant], #[postcondition], #[assumption], and #[progress] instead of #[invariants] (11:5) |
| `machine_legacy_scopes.rs` | `machine_legacy_scopes` | `machine::expand` | use per-item #[on(Actor)], #[effect], #[invariant], #[postcondition], #[assumption], and #[progress] instead of #[scopes] (16:5) |
| `machine_legacy_transitions.rs` | `machine_legacy_transitions` | `machine::expand` | use per-item #[on(Actor)], #[effect], #[invariant], #[postcondition], #[assumption], and #[progress] instead of #[transitions] (11:5) |
| `machine_marker_trait.rs` | `machine_marker_trait` | `machine::expand` | use per-item machine attributes on an inherent impl Model instead of marker traits (11:10) |
| `machine_misplaced_effect.rs` | `machine_misplaced_effect` | `machine::expand` | #[effect] methods must follow #[on] and precede #[invariant] (17:9) |
| `machine_missing_actor.rs` | `machine_missing_actor` | `machine::expand` | actor User has no #[on] method (9:10) |
| `machine_mutable_helper.rs` | `machine_mutable_helper` | `machine::expand` | helper takes &mut self; only #[on] and #[effect] methods change the state (15:12) |
| `machine_mutable_invariant.rs` | `machine_mutable_invariant` | `machine::expand` | claim takes &mut self; only #[on] and #[effect] methods change the state (16:12) |
| `machine_mutable_typed_helper.rs` | `machine_mutable_typed_helper` | `machine::expand` | helper takes &mut self; only #[on] and #[effect] methods change the state (15:12) |
| `machine_nonactor_parameter.rs` | `machine_nonactor_parameter` | `machine::expand` | an #[on] method takes its actor enum: fn user(&mut self, action: User) (13:9) |
| `machine_noncontiguous_kind.rs` | `machine_noncontiguous_kind` | `machine::expand` | #[invariant] methods must follow #[effect] and precede #[postcondition] (19:9) |
| `machine_out_of_order.rs` | `machine_out_of_order` | `machine::expand` | #[invariant] methods must follow #[effect] and precede #[postcondition] (18:9) |
| `machine_path_builders.rs` | `machine_path_builders` | `machine::expand` | a path! claim takes only .assuming(..) (16:38) |
| `machine_post_async.rs` | `machine_post_async` | `machine::expand` | invariants and postconditions must be synchronous (16:9) |
| `machine_post_returns_unit.rs` | `machine_post_returns_unit` | `machine::expand` | invariants and postconditions return nothing; use check! (16:25) |
| `machine_progress_chain.rs` | `machine_progress_chain` | `machine::expand` | a progress claim has one trigger and one target; ask a longer progression with find or Run::along (16:38) |
| `machine_progress_gap.rs` | `machine_progress_gap` | `machine::expand` | a path gap is .. or ..=n (16:61) |
| `machine_starts_without_world.rs` | `machine_starts_without_world` | `machine::expand` | computed starts require a declared World (9:9) |
| `machine_unknown_actor.rs` | `machine_unknown_actor` | `machine::expand` | #[on] names an actor enum declared in this module (12:9) |
| `machine_unknown_after.rs` | `machine_unknown_after` | `machine::expand` | unknown after action User::Missing (15:33) |
| `machine_unknown_scope.rs` | `machine_unknown_scope` | `machine::expand` | a progress claim names its assumption with .assuming(..) (16:9) |
| `machine_within_bare.rs` | `machine_within_bare` | `machine::expand` | a progress claim names its assumption with .assuming(..) (16:9) |
| `non_range_starts.rs` | `non_range_starts` | `state::expand` | #[starts] requires a bounded range, such as 0..=40 (5:14); #[starts] requires a bounded range, such as 0..=40 (7:14) |
| `non_unit_internal.rs` | `non_unit_internal` | `action::expand` | Heater::Heat field `0` requires #[inputs] with a bounded range or finite list (6:10) |
| `optional_facts_payload.rs` | `optional_facts_payload` | `machine::expand` | fact carrier requires a nongeneric nonempty fieldless enum with derived Debug and no discriminants (13:5) |
| `variant_internal.rs` | `variant_internal` | `action::expand` | #[internal] belongs on the action enum, not a variant (5:5) |
| `world_duplicate.rs` | `world_duplicate` | `machine::expand` | declare World once as #[derive(World)] struct World (2:5) |
| `world_parameter_duplicate.rs` | `world_parameter_duplicate` | `machine::expand` | declare the World parameter once (12:52) |
| `world_parameter_mutable.rs` | `world_parameter_mutable` | `machine::expand` | the World parameter must be an immutable &World (12:47) |
| `world_parameter_position.rs` | `world_parameter_position` | `machine::expand` | the World parameter must be last (12:28) |
| `world_starts_empty_mixed.rs` | `world_starts_empty_mixed` | `machine::expand` | declare starts once (7:20) |
| `world_starts_impl_block.rs` | `world_starts_impl_block` | `machine::expand` | #[starts] marks a method in impl Model: #[starts] fn starts(world: &World) -> impl IntoIterator<Item = Self> (10:5) |
| `world_starts_mixed.rs` | `world_starts_mixed` | `machine::expand` | declare starts once (7:20) |
| `world_starts_receiver.rs` | `world_starts_receiver` | `machine::expand` | #[starts] requires an associated fn taking only an optional trailing &World: fn starts(world: &World) -> impl IntoIterator<Item = Self> (12:9) |
| `world_starts_twice.rs` | `world_starts_twice` | `machine::expand` | declare starts once (13:9) |

## Retained trybuild cases

| Source | Reason |
| --- | --- |
| `tests/ui/action_domain_item_type.rs` | Rust compiler type/trait/exhaustiveness/unsafe diagnostic; parser expansion cannot establish it. |
| `tests/ui/action_domain_unbound.rs` | Rust compiler type/trait/exhaustiveness/unsafe diagnostic; parser expansion cannot establish it. |
| `tests/ui/action_domain_world_mismatch.rs` | Rust compiler type/trait/exhaustiveness/unsafe diagnostic; parser expansion cannot establish it. |
| `tests/ui/action_finite_missing.rs` | Rust compiler type/trait/exhaustiveness/unsafe diagnostic; parser expansion cannot establish it. |
| `tests/ui/machine_mutable_aliased_helper.rs` | Rust compiler type/trait/exhaustiveness/unsafe diagnostic; parser expansion cannot establish it. |
| `tests/ui/machine_mutable_const_helper.rs` | Rust compiler type/trait/exhaustiveness/unsafe diagnostic; parser expansion cannot establish it. |
| `tests/ui/machine_mutable_local_helper.rs` | Rust compiler type/trait/exhaustiveness/unsafe diagnostic; parser expansion cannot establish it. |
| `tests/ui/machine_mutable_local_projection.rs` | Rust compiler type/trait/exhaustiveness/unsafe diagnostic; parser expansion cannot establish it. |
| `tests/ui/machine_mutable_nested_helper.rs` | Rust compiler type/trait/exhaustiveness/unsafe diagnostic; parser expansion cannot establish it. |
| `tests/ui/machine_mutable_qualified_helper.rs` | Rust compiler type/trait/exhaustiveness/unsafe diagnostic; parser expansion cannot establish it. |
| `tests/ui/machine_nonexhaustive_actor.rs` | Rust compiler type/trait/exhaustiveness/unsafe diagnostic; parser expansion cannot establish it. |
| `tests/ui/machine_scope_actor.rs` | Rust compiler type/trait/exhaustiveness/unsafe diagnostic; parser expansion cannot establish it. |
| `tests/ui/machine_stay_effect.rs` | Declarative `stay!` expansion inside an effect runs after machine expansion; preserve its original rustc diagnostic. |
| `tests/ui/machine_stay_helper.rs` | Declarative `stay!` expansion inside a helper runs after machine expansion; preserve its original rustc diagnostic. |
| `tests/ui/machine_unsafe_allowed.rs` | Rust compiler type/trait/exhaustiveness/unsafe diagnostic; parser expansion cannot establish it. |
| `tests/ui/optional_facts_direct_reject.rs` | Rust compiler type/trait/exhaustiveness/unsafe diagnostic; parser expansion cannot establish it. |
| `tests/ui/stay_outside_machine.rs` | Declarative `stay!` wrapper behavior outside machine expansion. |
| `tests/ui/world_starts_return.rs` | Rust compiler type/trait/exhaustiveness/unsafe diagnostic; parser expansion cannot establish it. |

## Admission limits

The 61 migrated files have validated parser diagnostics, with 18 cases retained in trybuild.
The original 63-case proposal included two `stay!` wrapper cases; runtime validation
showed that their refusal occurs during later macro expansion, so both remain in trybuild.
`finite_shape` expands all four annotated items; `action_metadata_placement` and
`non_range_starts` retain both diagnostics; `non_unit_internal` preserves its existing
single diagnostic (the bool field is finite and has no second refusal in the old stderr).
Every selected annotated item must fail, and every emitted diagnostic must match exactly.

Machine inputs strip only the invoked machine attribute, as the production proc-macro
boundary does. They call the existing declaration/generate/finish expansion. The omitted
lifted embedding follows declaration and generation in production; these refusal inputs
fail before that embedding boundary. No generated Rust type-check case is migrated.

The moved stderr files keep original `tests/ui` paths as provenance. They are not read by
trybuild; the table uses original full sources and explicit expected authored coordinates.
The parser controls establish the listed message/location equivalence. The retained
trybuild cases also passed in the complete compiler run described above. Its timing
includes the existing compiler-cache state and is not a matched-baseline speedup claim.
