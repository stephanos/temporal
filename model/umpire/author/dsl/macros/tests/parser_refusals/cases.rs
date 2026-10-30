use super::{Case, Diagnostic, Entry};

pub(super) const CASES: &[Case] = &[
    Case {
        name: "action_domain_conflict",
        source: include_str!("inputs/action_domain_conflict.rs"),
        entry: Entry::Action,
        expected: &[Diagnostic {
            message: "User::Set field `0` declares both #[inputs] and #[domain]; declare one domain",
            line: 5,
            column: 9,
        }],
    },
    Case {
        name: "action_domain_duplicate",
        source: include_str!("inputs/action_domain_duplicate.rs"),
        entry: Entry::Action,
        expected: &[Diagnostic {
            message: "User::Set field `0` has duplicate #[domain] declarations",
            line: 5,
            column: 26,
        }],
    },
    Case {
        name: "action_domain_variant",
        source: include_str!("inputs/action_domain_variant.rs"),
        entry: Entry::Action,
        expected: &[Diagnostic {
            message: "#[domain] belongs on one variant field, not the variant",
            line: 5,
            column: 5,
        }],
    },
    Case {
        name: "action_domain_without_world",
        source: include_str!("inputs/action_domain_without_world.rs"),
        entry: Entry::Action,
        expected: &[Diagnostic {
            message: "User::Set field `0` #[domain] requires the World of a #[machine] module that declares #[derive(World)]",
            line: 9,
            column: 9,
        }],
    },
    Case {
        name: "action_inputs_duplicate",
        source: include_str!("inputs/action_inputs_duplicate.rs"),
        entry: Entry::Action,
        expected: &[Diagnostic {
            message: "User::Set field `value` has duplicate #[inputs] domains",
            line: 7,
            column: 9,
        }],
    },
    Case {
        name: "action_inputs_invalid",
        source: include_str!("inputs/action_inputs_invalid.rs"),
        entry: Entry::Action,
        expected: &[Diagnostic {
            message: "User::Set field `value` #[inputs] requires a bounded range or finite list, such as 0..=3 or [false, true]",
            line: 6,
            column: 9,
        }],
    },
    Case {
        name: "action_inputs_missing",
        source: include_str!("inputs/action_inputs_missing.rs"),
        entry: Entry::Action,
        expected: &[Diagnostic {
            message: "User::Set field `value` requires #[inputs] with a bounded range or finite list",
            line: 5,
            column: 11,
        }],
    },
    Case {
        name: "action_inputs_variant_fault",
        source: include_str!("inputs/action_inputs_variant_fault.rs"),
        entry: Entry::Action,
        expected: &[Diagnostic {
            message: "#[fault] belongs on the action enum, not a variant",
            line: 5,
            column: 5,
        }],
    },
    Case {
        name: "action_metadata_duplicate",
        source: include_str!("inputs/action_metadata_duplicate.rs"),
        entry: Entry::Action,
        expected: &[Diagnostic {
            message: "duplicate action metadata namespace",
            line: 6,
            column: 16,
        }],
    },
    Case {
        name: "action_metadata_namespace",
        source: include_str!("inputs/action_metadata_namespace.rs"),
        entry: Entry::Action,
        expected: &[Diagnostic { message: "expected parentheses", line: 5, column: 24 }],
    },
    Case {
        name: "action_metadata_placement",
        source: include_str!("inputs/action_metadata_placement.rs"),
        entry: Entry::Action,
        expected: &[
            Diagnostic { message: "#[metadata] belongs on an action variant", line: 4, column: 1 },
            Diagnostic { message: "#[metadata] belongs on an action variant", line: 6, column: 12 },
        ],
    },
    Case {
        name: "finite_shape",
        source: include_str!("inputs/finite_shape.rs"),
        entry: Entry::Finite,
        expected: &[
            Diagnostic {
                message: "Finite derives for a fieldless enum or a struct with one #[inputs(lo..=hi)] integer field; give a variant with fields to derive(Action)",
                line: 5,
                column: 5,
            },
            Diagnostic {
                message: "range bound 300 does not fit the field's integer type",
                line: 9,
                column: 26,
            },
            Diagnostic {
                message: "a Finite newtype's field must be a primitive integer type",
                line: 12,
                column: 30,
            },
            Diagnostic {
                message: "Finite derives for a fieldless enum or a struct with one #[inputs(lo..=hi)] integer field",
                line: 15,
                column: 14,
            },
        ],
    },
    Case {
        name: "machine_actor_arity",
        source: include_str!("inputs/machine_actor_arity.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic {
            message: "an #[on] method takes its actor enum: fn user(&mut self, action: User)",
            line: 13,
            column: 9,
        }],
    },
    Case {
        name: "machine_actor_mismatch",
        source: include_str!("inputs/machine_actor_mismatch.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic {
            message: "#[on(User)] requires a User parameter",
            line: 16,
            column: 9,
        }],
    },
    Case {
        name: "machine_actor_receiver",
        source: include_str!("inputs/machine_actor_receiver.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic {
            message: "an #[on] method takes its actor enum: fn user(&mut self, action: User)",
            line: 13,
            column: 9,
        }],
    },
    Case {
        name: "machine_after_inputs",
        source: include_str!("inputs/machine_after_inputs.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic {
            message: "`User::Set` carries inputs; it is not one catalog action",
            line: 15,
            column: 33,
        }],
    },
    Case {
        name: "machine_claim_async",
        source: include_str!("inputs/machine_claim_async.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic {
            message: "invariants and postconditions must be synchronous",
            line: 16,
            column: 9,
        }],
    },
    Case {
        name: "machine_claim_returns",
        source: include_str!("inputs/machine_claim_returns.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic {
            message: "invariants and postconditions return nothing; use check!",
            line: 16,
            column: 25,
        }],
    },
    Case {
        name: "machine_duplicate_actor",
        source: include_str!("inputs/machine_duplicate_actor.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic {
            message: "actor User already has an #[on] method",
            line: 15,
            column: 12,
        }],
    },
    Case {
        name: "machine_duplicate_kind",
        source: include_str!("inputs/machine_duplicate_kind.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic {
            message: "a method requires exactly one machine kind attribute",
            line: 16,
            column: 9,
        }],
    },
    Case {
        name: "machine_effect_arguments",
        source: include_str!("inputs/machine_effect_arguments.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic { message: "unexpected token in attribute", line: 15, column: 17 }],
    },
    Case {
        name: "machine_generic",
        source: include_str!("inputs/machine_generic.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic {
            message: "#[machine] does not admit generic Model, World or actor declarations",
            line: 7,
            column: 17,
        }],
    },
    Case {
        name: "machine_handler_async",
        source: include_str!("inputs/machine_handler_async.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic {
            message: "#[on] methods must be synchronous",
            line: 13,
            column: 9,
        }],
    },
    Case {
        name: "machine_handler_returns",
        source: include_str!("inputs/machine_handler_returns.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic {
            message: "#[on] methods return nothing; enabledness is a changed state",
            line: 13,
            column: 42,
        }],
    },
    Case {
        name: "machine_invariant_parameters",
        source: include_str!("inputs/machine_invariant_parameters.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic {
            message: "an invariant takes only &self and an optional trailing World",
            line: 15,
            column: 9,
        }],
    },
    Case {
        name: "machine_legacy_effects",
        source: include_str!("inputs/machine_legacy_effects.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic {
            message: "use per-item #[on(Actor)], #[effect], #[invariant], #[postcondition], #[assumption], and #[progress] instead of #[effects]",
            line: 11,
            column: 5,
        }],
    },
    Case {
        name: "machine_legacy_invariants",
        source: include_str!("inputs/machine_legacy_invariants.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic {
            message: "use per-item #[on(Actor)], #[effect], #[invariant], #[postcondition], #[assumption], and #[progress] instead of #[invariants]",
            line: 11,
            column: 5,
        }],
    },
    Case {
        name: "machine_legacy_scopes",
        source: include_str!("inputs/machine_legacy_scopes.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic {
            message: "use per-item #[on(Actor)], #[effect], #[invariant], #[postcondition], #[assumption], and #[progress] instead of #[scopes]",
            line: 16,
            column: 5,
        }],
    },
    Case {
        name: "machine_legacy_transitions",
        source: include_str!("inputs/machine_legacy_transitions.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic {
            message: "use per-item #[on(Actor)], #[effect], #[invariant], #[postcondition], #[assumption], and #[progress] instead of #[transitions]",
            line: 11,
            column: 5,
        }],
    },
    Case {
        name: "machine_marker_trait",
        source: include_str!("inputs/machine_marker_trait.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic {
            message: "use per-item machine attributes on an inherent impl Model instead of marker traits",
            line: 11,
            column: 10,
        }],
    },
    Case {
        name: "machine_misplaced_effect",
        source: include_str!("inputs/machine_misplaced_effect.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic {
            message: "#[effect] methods must follow #[on] and precede #[invariant]",
            line: 17,
            column: 9,
        }],
    },
    Case {
        name: "machine_missing_actor",
        source: include_str!("inputs/machine_missing_actor.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic { message: "actor User has no #[on] method", line: 9, column: 10 }],
    },
    Case {
        name: "machine_mutable_helper",
        source: include_str!("inputs/machine_mutable_helper.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic {
            message: "helper takes &mut self; only #[on] and #[effect] methods change the state",
            line: 15,
            column: 12,
        }],
    },
    Case {
        name: "machine_mutable_invariant",
        source: include_str!("inputs/machine_mutable_invariant.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic {
            message: "claim takes &mut self; only #[on] and #[effect] methods change the state",
            line: 16,
            column: 12,
        }],
    },
    Case {
        name: "machine_mutable_typed_helper",
        source: include_str!("inputs/machine_mutable_typed_helper.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic {
            message: "helper takes &mut self; only #[on] and #[effect] methods change the state",
            line: 15,
            column: 12,
        }],
    },
    Case {
        name: "machine_nonactor_parameter",
        source: include_str!("inputs/machine_nonactor_parameter.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic {
            message: "an #[on] method takes its actor enum: fn user(&mut self, action: User)",
            line: 13,
            column: 9,
        }],
    },
    Case {
        name: "machine_noncontiguous_kind",
        source: include_str!("inputs/machine_noncontiguous_kind.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic {
            message: "#[invariant] methods must follow #[effect] and precede #[postcondition]",
            line: 19,
            column: 9,
        }],
    },
    Case {
        name: "machine_out_of_order",
        source: include_str!("inputs/machine_out_of_order.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic {
            message: "#[invariant] methods must follow #[effect] and precede #[postcondition]",
            line: 18,
            column: 9,
        }],
    },
    Case {
        name: "machine_path_builders",
        source: include_str!("inputs/machine_path_builders.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic {
            message: "a path! claim takes only .assuming(..)",
            line: 16,
            column: 38,
        }],
    },
    Case {
        name: "machine_post_async",
        source: include_str!("inputs/machine_post_async.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic {
            message: "invariants and postconditions must be synchronous",
            line: 16,
            column: 9,
        }],
    },
    Case {
        name: "machine_post_returns_unit",
        source: include_str!("inputs/machine_post_returns_unit.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic {
            message: "invariants and postconditions return nothing; use check!",
            line: 16,
            column: 25,
        }],
    },
    Case {
        name: "machine_progress_chain",
        source: include_str!("inputs/machine_progress_chain.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic {
            message: "a progress claim has one trigger and one target; ask a longer progression with find or Run::along",
            line: 16,
            column: 38,
        }],
    },
    Case {
        name: "machine_progress_gap",
        source: include_str!("inputs/machine_progress_gap.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic { message: "a path gap is .. or ..=n", line: 16, column: 61 }],
    },
    Case {
        name: "machine_starts_without_world",
        source: include_str!("inputs/machine_starts_without_world.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic {
            message: "computed starts require a declared World",
            line: 9,
            column: 9,
        }],
    },
    Case {
        name: "machine_unknown_actor",
        source: include_str!("inputs/machine_unknown_actor.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic {
            message: "#[on] names an actor enum declared in this module",
            line: 12,
            column: 9,
        }],
    },
    Case {
        name: "machine_unknown_after",
        source: include_str!("inputs/machine_unknown_after.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic {
            message: "unknown after action User::Missing",
            line: 15,
            column: 33,
        }],
    },
    Case {
        name: "machine_unknown_scope",
        source: include_str!("inputs/machine_unknown_scope.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic {
            message: "a progress claim names its assumption with .assuming(..)",
            line: 16,
            column: 9,
        }],
    },
    Case {
        name: "machine_within_bare",
        source: include_str!("inputs/machine_within_bare.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic {
            message: "a progress claim names its assumption with .assuming(..)",
            line: 16,
            column: 9,
        }],
    },
    Case {
        name: "non_range_starts",
        source: include_str!("inputs/non_range_starts.rs"),
        entry: Entry::State,
        expected: &[
            Diagnostic {
                message: "#[starts] requires a bounded range, such as 0..=40",
                line: 5,
                column: 14,
            },
            Diagnostic {
                message: "#[starts] requires a bounded range, such as 0..=40",
                line: 7,
                column: 14,
            },
        ],
    },
    Case {
        name: "non_unit_internal",
        source: include_str!("inputs/non_unit_internal.rs"),
        entry: Entry::Action,
        expected: &[Diagnostic {
            message: "Heater::Heat field `0` requires #[inputs] with a bounded range or finite list",
            line: 6,
            column: 10,
        }],
    },
    Case {
        name: "optional_facts_payload",
        source: include_str!("inputs/optional_facts_payload.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic {
            message: "fact carrier requires a nongeneric nonempty fieldless enum with derived Debug and no discriminants",
            line: 13,
            column: 5,
        }],
    },
    Case {
        name: "variant_internal",
        source: include_str!("inputs/variant_internal.rs"),
        entry: Entry::Action,
        expected: &[Diagnostic {
            message: "#[internal] belongs on the action enum, not a variant",
            line: 5,
            column: 5,
        }],
    },
    Case {
        name: "world_duplicate",
        source: include_str!("inputs/world_duplicate.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic {
            message: "declare World once as #[derive(World)] struct World",
            line: 2,
            column: 5,
        }],
    },
    Case {
        name: "world_parameter_duplicate",
        source: include_str!("inputs/world_parameter_duplicate.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic {
            message: "declare the World parameter once",
            line: 12,
            column: 52,
        }],
    },
    Case {
        name: "world_parameter_mutable",
        source: include_str!("inputs/world_parameter_mutable.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic {
            message: "the World parameter must be an immutable &World",
            line: 12,
            column: 47,
        }],
    },
    Case {
        name: "world_parameter_position",
        source: include_str!("inputs/world_parameter_position.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic {
            message: "the World parameter must be last",
            line: 12,
            column: 28,
        }],
    },
    Case {
        name: "world_starts_empty_mixed",
        source: include_str!("inputs/world_starts_empty_mixed.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic { message: "declare starts once", line: 7, column: 20 }],
    },
    Case {
        name: "world_starts_impl_block",
        source: include_str!("inputs/world_starts_impl_block.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic {
            message: "#[starts] marks a method in impl Model: #[starts] fn starts(world: &World) -> impl IntoIterator<Item = Self>",
            line: 10,
            column: 5,
        }],
    },
    Case {
        name: "world_starts_mixed",
        source: include_str!("inputs/world_starts_mixed.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic { message: "declare starts once", line: 7, column: 20 }],
    },
    Case {
        name: "world_starts_receiver",
        source: include_str!("inputs/world_starts_receiver.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic {
            message: "#[starts] requires an associated fn taking only an optional trailing &World: fn starts(world: &World) -> impl IntoIterator<Item = Self>",
            line: 12,
            column: 9,
        }],
    },
    Case {
        name: "world_starts_twice",
        source: include_str!("inputs/world_starts_twice.rs"),
        entry: Entry::Machine,
        expected: &[Diagnostic { message: "declare starts once", line: 13, column: 9 }],
    },
];
