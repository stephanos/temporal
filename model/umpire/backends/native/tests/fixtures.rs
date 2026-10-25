//! Synthetic checker behavior fixtures.

#![forbid(unsafe_code)]

#[path = "support/formatting.rs"]
mod formatting;
#[path = "support/progress_evaluation.rs"]
mod progress_evaluation;
#[path = "support/recorder.rs"]
mod recorder;
mod support;
include!(concat!(env!("OUT_DIR"), "/fixtures.rs"));

#[test]
fn fixtures() -> umpire_testing::TestResult {
    umpire_testing::run(env!("CARGO_MANIFEST_DIR"), registered())
}
