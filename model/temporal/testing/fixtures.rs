//! System Models use the same source-lifted backend agreement consumer.

include!(concat!(env!("OUT_DIR"), "/fixtures.rs"));
include!(concat!(env!("OUT_DIR"), "/lifted.rs"));

mod realization;
mod realization_control;

fn fixture_name(folder: &std::path::Path) -> Result<&str, Box<dyn std::error::Error>> {
    folder.file_name().and_then(|name| name.to_str()).ok_or_else(|| "invalid fixture folder".into())
}

const LIFT: umpire_testing::Consumer = umpire_testing::Consumer {
    name: "source lift",
    expected: &["*.tla", "*.cfg", "lift.txt"],
    render: |_, folder| match lifted(fixture_name(folder)?) {
        Ok(module) => Ok(umpire_testing::Files::from([
            (format!("{}.tla", module.name), module.tla),
            (format!("{}.cfg", module.name), module.cfg),
            ("lift.txt".into(), "admitted: independent authored-source lift\n".into()),
        ])),
        Err(diagnostic) => {
            Ok(umpire_testing::Files::from([("lift.txt".into(), format!("{diagnostic}\n"))]))
        }
    },
};

const TLC: umpire_testing::Consumer =
    umpire_testing::Consumer { name: "tlc", expected: &["tlc.txt"], render: render_tlc };

const GAPS: umpire_testing::ExternalConsumer = umpire_testing::ExternalConsumer {
    name: "make umpire-check-model-gaps (TestTestpilotModelGaps)",
    fixture: "activity",
    expected: "gaps.txt",
};

#[allow(clippy::print_stderr)]
fn render_tlc(
    run: fn() -> umpire_testing::RunResult,
    folder: &std::path::Path,
) -> umpire_testing::RunResult {
    let name = fixture_name(folder)?;
    eprintln!("TLC fixture {name}: checking independently lifted source");
    let files = check_backend(name, folder, run)?;
    eprintln!(
        "TLC fixture {name}: {}",
        files.get("tlc.txt").ok_or("missing TLC agreement report")?
    );
    Ok(files)
}

#[test]
#[allow(clippy::print_stderr)]
fn fixtures() -> umpire_testing::TestResult {
    let root = std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("../feature");
    let required = std::env::var("UMPIRE_BACKENDS").as_deref() == Ok("require");
    if required {
        if admitted().is_empty() {
            eprintln!(
                "No source-admitted system fixture: no system TLC execution or agreement evidence; located refusal goldens remain required"
            );
        }
        for name in admitted() {
            if !root.join(name).join("expected/tlc.snap.txt").is_file() {
                return Err(format!("unselected backend coverage: {name} has no TLC golden").into());
            }
        }
    }
    umpire_testing::run_at_skipping_with_external(
        &root,
        registered(),
        &[umpire_testing::CHECKER, umpire_testing::SUITE, LIFT, TLC, realization::COVERAGE],
        if required { &[] } else { &["tlc"] },
        &[GAPS],
    )
}
