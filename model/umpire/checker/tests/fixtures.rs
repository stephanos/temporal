//! Synthetic checker behavior fixtures.

mod support;
include!(concat!(env!("OUT_DIR"), "/fixtures.rs"));

#[test]
fn fixtures() -> umpire_testing::TestResult {
    umpire_testing::run(env!("CARGO_MANIFEST_DIR"), registered())
}
