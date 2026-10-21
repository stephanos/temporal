use super::ledger::Execution;
use std::collections::BTreeSet;

pub(super) fn stop(
    policy: &str,
    budget: u64,
    execution: &Execution,
    failures: &BTreeSet<String>,
) -> Result<Option<String>, String> {
    if !matches!(
        execution.record.outcome.class.as_str(),
        "target_failure" | "watchdog"
    ) {
        return Ok(None);
    }
    let signature = execution.record.failure_signature()?;
    let distinct = failures.len() as u64 + u64::from(!failures.contains(&signature));
    Ok(match policy {
        "first" => Some("first_failure".into()),
        "budget" if distinct >= budget => Some("failure_budget".into()),
        _ => None,
    })
}
