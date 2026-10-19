//! Standard formatting of DSL states and actions.

#[path = "support/recorder.rs"]
mod recorder;

include!(concat!(env!("OUT_DIR"), "/fixtures.rs"));

#[test]
fn fixtures() -> umpire_testing::TestResult {
    umpire_testing::run_at(
        &std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("tests/fixtures"),
        registered(),
        &[umpire_testing::Consumer {
            name: "formatting",
            expected: &["values.txt"],
            render: |produce, _| produce(),
        }],
    )
}
