use crate::sample_kettle_trait::KettleWorld;

pub(crate) fn run() -> umpire_testing::RunResult {
    let result = umpire_testing::Checker::explore(&KettleWorld, 1 << 20)?;
    let mut files = umpire_testing::render(&result)?;
    files.insert("rows.txt".into(), umpire_testing::render_rows(&result)?);
    Ok(files)
}
