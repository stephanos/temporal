//! Canonical spelling of the DSL's leaf values.

include!(concat!(env!("OUT_DIR"), "/fixtures.rs"));

#[test]
fn fixtures() -> umpire_testing::TestResult {
    umpire_testing::run_at(
        &std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("tests/fixtures"),
        registered(),
        &[umpire_testing::Consumer {
            name: "spell",
            expected: &["values.txt"],
            render: |produce, _| produce(),
        }],
    )
}
