use crate::fixture_progress_5funknown::{Graph, Mode};

pub(crate) fn run() -> umpire_testing::RunResult {
    let Err(error @ umpire_checker::Refusal::ScopeAction { .. }) =
        umpire_checker::Checker::explore(&Graph(Mode::UnknownScope), 8)
    else {
        return Err("unknown scope action did not return Refusal::ScopeAction".into());
    };
    Ok(std::collections::BTreeMap::from([("report.txt".into(), format!("refused: {error}\n"))]))
}
