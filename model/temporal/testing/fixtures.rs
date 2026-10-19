//! Temporal Models run through the same consumers as the generic fixtures.

include!(concat!(env!("OUT_DIR"), "/fixtures.rs"));

#[test]
fn fixtures() -> umpire_testing::TestResult {
    umpire_testing::run_at(
        &std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("../features"),
        registered(),
        &[umpire_testing::CHECKER],
    )
}
