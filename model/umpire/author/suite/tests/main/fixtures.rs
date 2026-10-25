//! The suite's failure messages, pinned by fixtures.

#![forbid(unsafe_code)]

#[path = "../support/coin.rs"]
pub(super) mod coin;
#[path = "../support/kettle.rs"]
#[allow(unreachable_pub, clippy::module_inception)]
pub(super) mod kettle;
#[path = "../../../../dev/fixtures/fixtures/sample_kettle/model.rs"]
#[allow(unreachable_pub)]
pub(super) mod sample_kettle;

include!(concat!(env!("OUT_DIR"), "/fixtures.rs"));

/// The panic message of a failing `expect!`.
pub(super) fn failure(test: impl FnOnce()) -> String {
    std::panic::catch_unwind(std::panic::AssertUnwindSafe(test))
        .err()
        .and_then(|error| error.downcast::<String>().ok())
        .map_or_else(|| "passed".into(), |message| *message)
}

#[test]
fn fixtures() -> umpire_testing::TestResult {
    umpire_testing::run_at(
        &std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("tests/fixtures"),
        registered(),
        &[
            umpire_testing::Consumer {
                name: "suite-messages",
                expected: &["messages.txt"],
                render: |produce, _| produce(),
            },
            umpire_testing::SUITE,
        ],
    )
}
