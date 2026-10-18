//! Every shared Model runs through the consumers selected by its expected files.

include!(concat!(env!("OUT_DIR"), "/fixtures.rs"));

#[test]
fn fixtures() -> umpire_testing::TestResult {
    umpire_testing::run_at(
        &std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("fixtures"),
        registered(),
        &[umpire_testing::CHECKER],
    )
}
