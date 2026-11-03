//! System Models use the same source-lifted backend agreement consumer.

#![forbid(unsafe_code)]

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
    render: |_, folder| match lifted(fixture_name(folder)?, folder) {
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
    expected: "gaps.txt.snap",
};

#[allow(clippy::print_stderr)]
fn render_tlc(
    run: fn() -> umpire_testing::RunResult,
    folder: &std::path::Path,
) -> umpire_testing::RunResult {
    let name = fixture_name(folder)?;
    eprintln!("TLC fixture {name}: checking the lifted Model");
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
        if admitted(&root).is_empty() {
            eprintln!(
                "No source-admitted system fixture: no system TLC execution or agreement evidence; located refusal goldens remain required"
            );
        }
        for name in admitted(&root) {
            if !root.join(name).join("expected/tlc.txt.snap").is_file() {
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

/// Import only an explicitly selected report produced by the live Go gaps test.
#[test]
#[ignore = "requires an explicitly selected live Go gap report"]
fn import_live_gap_candidate() -> umpire_testing::TestResult {
    let candidate = std::env::var_os("UMPIRE_GAP_CANDIDATE").ok_or("missing live gap candidate")?;
    let candidate = std::fs::read_to_string(candidate)?;
    let mut settings = insta::Settings::clone_current();
    settings.set_snapshot_path("../feature/activity/expected");
    settings.set_prepend_module_to_snapshot(false);
    settings.set_omit_expression(true);
    settings.bind(|| insta::assert_snapshot!("gaps.txt", candidate));
    Ok(())
}

mod protocol_fixtures {
    include!(concat!(env!("OUT_DIR"), "/protocol_fixtures.rs"));
    pub(super) use activity_tests::{check, lifted, scopes};

    pub(super) fn fixtures() -> Vec<umpire_testing::Fixture> {
        registered()
    }
}

#[test]
fn standalone_activity_protocol_fixtures() -> umpire_testing::TestResult {
    const PROTOCOL_LIFT: umpire_testing::Consumer = umpire_testing::Consumer {
        name: "protocol source lift",
        expected: &["*.tla", "*.cfg", "lift.txt"],
        render: |_, folder| {
            let module = protocol_fixtures::lifted(&folder.join("model.rs"))?;
            Ok(umpire_testing::Files::from([
                (format!("{}.tla", module.name), module.tla),
                (format!("{}.cfg", module.name), module.cfg),
                ("lift.txt".into(), "admitted: independent protocol source lift\n".into()),
            ]))
        },
    };
    const PROTOCOL_TLC: umpire_testing::Consumer = umpire_testing::Consumer {
        name: "protocol tlc",
        expected: &["tlc.txt"],
        render: |_, folder| protocol_fixtures::check(folder),
    };
    const PROTOCOL_SCOPES: umpire_testing::Consumer = umpire_testing::Consumer {
        name: "protocol scopes",
        expected: &["worlds.txt", "schema.txt", "worlds/*.txt"],
        render: |_, _| protocol_fixtures::scopes(),
    };
    let root = std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("../shared/protocol");
    umpire_testing::run_at_skipping(
        &root,
        protocol_fixtures::fixtures(),
        &[umpire_testing::CHECKER, PROTOCOL_LIFT, PROTOCOL_TLC, PROTOCOL_SCOPES],
        if std::env::var("UMPIRE_BACKENDS").as_deref() == Ok("require") {
            &[]
        } else {
            &["protocol tlc"]
        },
    )
}
