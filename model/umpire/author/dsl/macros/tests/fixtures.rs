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
    tests.compile_fail("tests/ui/*.rs");
}

#[test]
fn private_state() {
    trybuild::TestCases::new().pass("tests/pass/*.rs");
}
