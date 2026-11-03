use super::super::activity::protocol::{Failure, Model, World};
use super::{BUDGET, SOURCE, configured, folder};
use umpire_dsl::__lifted::Lifts;
use umpire_testing::{replace_in, replace_once};

fn tree(source: &str) -> Result<umpire_front::Lifted, Box<dyn std::error::Error>> {
    umpire_front::read(&syn::parse_file(source)?, &[])
        .map(|(tree, _, _)| tree)
        .map_err(|error| format!("{error:?}").into())
}

#[expect(clippy::too_many_lines, reason = "One independently selected defect table")]
fn edits() -> Vec<(String, String)> {
    let mut edits = Vec::new();
    for field in ["queued_budget", "attempt_budget", "execution_budget", "heartbeat_budget"] {
        let decrement =
            format!("ticks: self.{field}.unwrap_or(Remaining {{ ticks: 0 }}).ticks - 1");
        edits.push((
            format!("frozen-{field}"),
            replace_in(
                SOURCE,
                "fn consume",
                "fn selected_attempts",
                &decrement,
                &decrement.replace("- 1", "- 0"),
            ),
        ));
        let assignment = format!("self.{field} = None;");
        edits.push((
            format!("uncleared-{field}"),
            replace_in(
                SOURCE,
                "fn finish",
                "fn consume",
                &assignment,
                &format!("self.{field} = Some(Remaining {{ ticks: 1 }});"),
            ),
        ));
    }
    for (name, begin, end, from, to) in [
        (
            "erased-queued-budget",
            "fn schedule_retry",
            "fn finish",
            "self.queued_budget = if world.schedule_to_start.is_some()",
            "self.queued_budget = if false && world.schedule_to_start.is_some()",
        ),
        (
            "erased-execution-budget",
            "fn worker",
            "fn timer",
            "stay!();",
            "stay!(); self.execution_budget = None;",
        ),
        (
            "erased-attempt-budget",
            "fn worker",
            "fn timer",
            "self.attempt_budget = if world.start_to_close.is_some()",
            "self.attempt_budget = if false && world.start_to_close.is_some()",
        ),
        (
            "erased-heartbeat-budget",
            "fn worker",
            "fn timer",
            "self.heartbeat_budget = if world.heartbeat_timeout.is_some()",
            "self.heartbeat_budget = if false && world.heartbeat_timeout.is_some()",
        ),
        (
            "wrong-retry-waiting-reset",
            "fn schedule_retry",
            "fn finish",
            "world.schedule_to_start.unwrap_or(Ticks(1)).0",
            "1",
        ),
        (
            "replenished-spanning-budget",
            "fn schedule_retry",
            "fn finish",
            "self.attempt += 1;",
            "self.attempt += 1; self.execution_budget = Some(Remaining { ticks: world.schedule_to_close.unwrap_or(Ticks(1)).0 });",
        ),
        (
            "wrong-heartbeat-reset",
            "Worker::Heartbeat if",
            "Worker::AcknowledgeCancel(token)",
            "world.heartbeat_timeout.unwrap_or(Ticks(1)).0",
            "1",
        ),
        (
            "wrong-global-priority",
            "fn timer",
            "fn attempt_expired",
            "if self.execution_budget.is_some()",
            "if false && self.execution_budget.is_some()",
        ),
        (
            "wrong-local-priority",
            "fn timer",
            "fn attempt_expired",
            "else if self.attempt_budget.is_some()",
            "else if false && self.attempt_budget.is_some()",
        ),
        (
            "wrong-expiry-fact",
            "fn timer",
            "fn attempt_expired",
            "Some(Expired::StartToClose)",
            "Some(Expired::Heartbeat)",
        ),
        (
            "wrong-pending-timeout-outcome",
            "fn attempt_expired",
            "fn schedule_retry",
            "Terminal::TimedOut",
            "Terminal::Canceled",
        ),
        (
            "wrong-pending-failure-outcome",
            "Worker::Fail { retryable } if",
            "Worker::Heartbeat if",
            "Terminal::Failed",
            "Terminal::Canceled",
        ),
    ] {
        // Some shared arming expressions occur at both delivery and heartbeat. Target one call site.
        let changed = if name == "erased-heartbeat-budget" {
            let begin =
                SOURCE.find(begin).ok_or("missing worker").unwrap_or_else(|e| panic!("{e}"));
            let stop = SOURCE[begin..]
                .find("Worker::Complete")
                .unwrap_or_else(|| panic!("missing complete"))
                + begin;
            format!(
                "{}{}{}",
                &SOURCE[..begin],
                replace_once(&SOURCE[begin..stop], from, to),
                &SOURCE[stop..]
            )
        } else {
            replace_in(SOURCE, begin, end, from, to)
        };
        edits.push((name.into(), changed));
    }
    edits
}

#[test]
fn source_only_budget_reset_priority_outcome_and_fact_mutants_disagree()
-> umpire_testing::TestResult {
    use std::fmt::Write;
    let world = configured(3, [Some(3), Some(2), Some(3), Some(2)], true);
    let healthy = World::__umpire_lifted().model.map_err(|error| format!("{error:?}"))?;
    assert_eq!(healthy.fields.len(), 9);
    let mut controls = String::new();
    for (name, source) in edits() {
        assert_eq!(
            SOURCE.split("#[invariant]").skip(1).collect::<Vec<_>>(),
            source.split("#[invariant]").skip(1).collect::<Vec<_>>(),
            "{name}: laws changed"
        );
        let interpreted = umpire_checker::Interpreted::from_tree(world.clone(), &tree(&source)?)?;
        let disagreement = umpire_testing::compare(&world, &interpreted, None, BUDGET, None)
            .err()
            .ok_or_else(|| format!("mutant {name} agreed with the full protocol graph"))?;
        writeln!(controls, "{name}: rejected; {disagreement}")?;
    }
    if let Some(path) = std::env::var_os("UMPIRE_PROTOCOL_MUTANT_REPORT") {
        std::fs::write(path, controls)?;
    }
    Ok(())
}

#[test]
fn required_tlc_fact_counterexample_replays_and_refuses_changed_expiry_fact()
-> umpire_testing::TestResult {
    if std::env::var("UMPIRE_BACKENDS").as_deref() != Ok("require") {
        return Ok(());
    }
    let world = configured(2, [Some(1), Some(1), None, None], true);
    let claim = || {
        umpire_dsl::Claim::new("no_attempt_timeout", |_: &World, state: &Model| {
            (state.retained_failure != Failure::StartToClose).into()
        })
    };
    let suffix = "\numpire_testing::fixture!(World = World::new(2, Some(Ticks(1)), Some(Ticks(1)), None, None, true), 4096, Claim::new(\"no_attempt_timeout\", |_, state: &Model| (state.retained_failure != Failure::StartToClose).into()));";
    let selected = umpire_testing::bound(world.clone(), BUDGET, Some(claim()));
    let healthy = selected.lift_source("model.rs", &format!("{SOURCE}{suffix}"))?;
    let result = selected.check_module(&folder(), &healthy)?;
    assert!(result["tlc.txt"].contains("counterexamples replayed: 1"));
    let mutated = replace_in(
        SOURCE,
        "fn timer",
        "fn attempt_expired",
        "Some(Expired::StartToClose)",
        "Some(Expired::Heartbeat)",
    );
    let selected = umpire_testing::bound(world, BUDGET, Some(claim()));
    let mutant = selected.lift_source("model.rs", &format!("{mutated}{suffix}"))?;
    let refusal = selected
        .check_module(&folder(), &mutant)
        .err()
        .ok_or("changed expiry fact replayed against healthy protocol")?;
    assert!(refusal.to_string().contains("fact"), "{refusal}");
    Ok(())
}
