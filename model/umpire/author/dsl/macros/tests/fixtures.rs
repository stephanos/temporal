//! Derive expansions and compilation diagnostics.

#[path = "../src/action.rs"]
mod action;
#[path = "../src/finite.rs"]
mod finite;
#[path = "../src/machine.rs"]
mod machine;
#[path = "../src/path.rs"]
mod path;
#[path = "../src/state.rs"]
mod state;
#[path = "../src/world.rs"]
mod world;

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
    trybuild::TestCases::new().compile_fail("tests/ui/*.rs");
}

#[test]
fn private_state() {
    trybuild::TestCases::new().pass("tests/pass/*.rs");
}
