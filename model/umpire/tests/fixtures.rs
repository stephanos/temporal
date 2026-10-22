//! Every shared Model runs through the consumers selected by its expected files.

include!(concat!(env!("OUT_DIR"), "/fixtures.rs"));
include!(concat!(env!("OUT_DIR"), "/lifted.rs"));

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
fn fixtures() -> umpire_testing::TestResult {
    let root = std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("fixtures");
    let required = std::env::var("UMPIRE_BACKENDS").as_deref() == Ok("require");
    if required {
        if !admitted(&root).contains(&"sample_kettle") {
            return Err(
                "unselected backend coverage: attribute Kettle must be source-admitted".into()
            );
        }
        for name in admitted(&root) {
            if !root.join(name).join("expected/tlc.snap.txt").is_file() {
                return Err(format!("unselected backend coverage: {name} has no TLC golden").into());
            }
        }
    }
    umpire_testing::run_at_skipping(
        &root,
        registered(),
        &[umpire_testing::CHECKER, umpire_testing::SUITE, LIFT, TLC],
        if required { &[] } else { &["tlc"] },
    )
}
