//! Derive expansions and compilation diagnostics.

#![forbid(unsafe_code)]

#[path = "../src/action.rs"]
mod action;
#[path = "../src/finite.rs"]
mod finite;
#[path = "../src/machine.rs"]
mod machine;
#[path = "../src/path.rs"]
mod path;
#[path = "../src/roots.rs"]
#[allow(dead_code, reason = "expansion fixtures use admission without token root rewriting")]
mod roots;
#[path = "../src/state.rs"]
mod state;
#[path = "../src/world.rs"]
mod world;

#[path = "parser_refusals/mod.rs"]
mod parser_refusals;

include!(concat!(env!("OUT_DIR"), "/fixtures.rs"));

#[test]
fn fixtures() -> umpire_testing::TestResult {
    umpire_testing::run_at(
        &std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("tests/fixtures"),
        registered(),
        &[umpire_testing::Consumer {
            name: "expansion",
            expected: &["expansion.rs"],
            render: |produce, _| produce(),
        }],
    )
}

#[test]
fn refusals() {
    let tests = trybuild::TestCases::new();
    tests.compile_fail("tests/ui/action_domain_item_type.rs");
    tests.compile_fail("tests/ui/action_domain_unbound.rs");
    tests.compile_fail("tests/ui/action_domain_world_mismatch.rs");
    tests.compile_fail("tests/ui/action_finite_missing.rs");
    tests.compile_fail("tests/ui/machine_mutable_aliased_helper.rs");
    tests.compile_fail("tests/ui/machine_mutable_const_helper.rs");
    tests.compile_fail("tests/ui/machine_mutable_local_helper.rs");
    tests.compile_fail("tests/ui/machine_mutable_local_projection.rs");
    tests.compile_fail("tests/ui/machine_mutable_nested_helper.rs");
    tests.compile_fail("tests/ui/machine_mutable_qualified_helper.rs");
    tests.compile_fail("tests/ui/machine_nonexhaustive_actor.rs");
    tests.compile_fail("tests/ui/machine_scope_actor.rs");
    tests.compile_fail("tests/ui/machine_stay_effect.rs");
    tests.compile_fail("tests/ui/machine_stay_helper.rs");
    tests.compile_fail("tests/ui/machine_unsafe_allowed.rs");
    tests.compile_fail("tests/ui/optional_facts_direct_reject.rs");
    tests.compile_fail("tests/ui/stay_outside_machine.rs");
    tests.compile_fail("tests/ui/world_starts_return.rs");
}

#[test]
fn private_state() {
    trybuild::TestCases::new().pass("tests/pass/*.rs");
}
