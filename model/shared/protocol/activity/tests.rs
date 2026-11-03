use super::activity::protocol::{
    Action, Client, Expired, Failure, Lifecycle, Model, Remaining, Terminal, Ticks, Timer, Worker,
    World,
};
use umpire_dsl::{Machine, Outcome, Rejection};
use umpire_temporal::activity_protocol::parameters;

type TestResult = umpire_testing::TestResult;
const BUDGET: usize = 4096;
const SOURCE: &str = include_str!("model.rs");

fn configured(maximum: u8, timeouts: [Option<u8>; 4], cooperative: bool) -> World {
    assert!((1..=3).contains(&maximum));
    let [waiting, running, global, heartbeat] = timeouts.map(|raw| {
        raw.map(|ticks| Ticks::try_new(ticks).unwrap_or_else(|| panic!("invalid timeout")))
    });
    World::new(maximum, waiting, running, global, heartbeat, cooperative)
}

fn world() -> World {
    configured(2, [Some(1), Some(1), None, None], true)
}

fn folder() -> std::path::PathBuf {
    std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("../shared/protocol/activity")
}

fn bound(world: World) -> umpire_testing::Bound<World> {
    umpire_testing::bound(world, BUDGET, None)
}

pub(crate) fn lifted(source: &std::path::Path) -> Result<umpire_tla::Module, String> {
    bound(world()).lift(source)
}

pub(crate) fn check(folder: &std::path::Path) -> umpire_testing::RunResult {
    bound(world()).check(folder)
}

pub(crate) fn run() -> umpire_testing::RunResult {
    umpire_testing::interpreted(world, None, BUDGET, None)?;
    bound(world()).run()
}

fn accepted(
    world: &World,
    before: &Model,
    action: Action,
) -> Result<Model, Box<dyn std::error::Error>> {
    let steps = world.checked_step(before, action)?;
    assert_eq!(steps.len(), 1);
    assert_eq!(steps[0].0, Outcome::Accepted);
    Ok(steps[0].2.clone())
}

#[test]
fn terminal_result_and_retained_failure_are_independent() -> TestResult {
    let world = world();
    let start = world.starts().remove(0);
    let running = accepted(&world, &start, Worker::Deliver(1).into())?;
    let retried = accepted(&world, &running, Worker::Fail { retryable: true }.into())?;
    assert_eq!(retried.lifecycle, Lifecycle::Queued);
    assert_eq!(retried.attempt, 2);
    assert_eq!(retried.retained_failure, Failure::RetryableWorker);
    assert_eq!(retried.terminal_outcome, Terminal::None);
    let running = accepted(&world, &retried, Worker::Deliver(2).into())?;
    let completed = accepted(&world, &running, Worker::Complete.into())?;
    assert_eq!(completed.terminal_outcome, Terminal::Completed);
    assert_eq!(completed.retained_failure, Failure::RetryableWorker);
    assert!(
        completed.queued_budget.is_none()
            && completed.attempt_budget.is_none()
            && completed.execution_budget.is_none()
            && completed.heartbeat_budget.is_none()
    );
    Ok(())
}

#[test]
fn cancellation_notification_failure_timeout_and_acknowledgement_are_distinct() -> TestResult {
    for cap in 1..=3 {
        let world = configured(cap, [Some(2), Some(1), None, Some(1)], true);
        let scheduled = world.starts().remove(0);
        let canceled = accepted(&world, &scheduled, Client::RequestCancel.into())?;
        assert_eq!(canceled.terminal_outcome, Terminal::Canceled);
        let running = accepted(&world, &scheduled, Worker::Deliver(1).into())?;
        let pending = accepted(&world, &running, Client::RequestCancel.into())?;
        assert!(pending.cancellation_pending);
        assert_eq!(pending.terminal_outcome, Terminal::None);
        let heartbeat = accepted(&world, &pending, Worker::Heartbeat.into())?;
        assert_eq!(heartbeat, pending);
        let ack = accepted(&world, &pending, Worker::AcknowledgeCancel(1).into())?;
        assert_eq!(ack.terminal_outcome, Terminal::Canceled);
        for retryable in [false, true] {
            let failed = accepted(&world, &pending, Worker::Fail { retryable }.into())?;
            assert_eq!(failed.terminal_outcome, Terminal::Failed);
            assert_eq!(failed.attempt, 1);
        }
        let tick = world.checked_step(&pending, Timer::Tick.into())?;
        assert_eq!(tick.len(), 1);
        assert_eq!(tick[0].1, [Expired::StartToClose]);
        assert_eq!(tick[0].2.terminal_outcome, Terminal::TimedOut);
        assert_eq!(tick[0].2.retained_failure, Failure::StartToClose);
        assert_eq!(
            world.checked_step(&pending, Client::RequestCancel.into())?.into_vec(),
            [(Outcome::Rejected(Rejection::FailedPrecondition), vec![], pending)]
        );
    }
    Ok(())
}

#[test]
fn latent_budgets_count_down_reset_span_retry_and_select_exact_priority() -> TestResult {
    let world = configured(3, [Some(3), Some(2), Some(3), Some(1)], true);
    let scheduled = world.starts().remove(0);
    let running = accepted(&world, &scheduled, Worker::Deliver(1).into())?;
    let expiry = world.checked_step(&running, Timer::Tick.into())?;
    assert_eq!(expiry[0].1, [Expired::Heartbeat]);
    let retried = &expiry[0].2;
    assert_eq!(retried.attempt, 2);
    assert_eq!(retried.queued_budget, Some(Remaining { ticks: 3 }));
    assert_eq!(retried.execution_budget, Some(Remaining { ticks: 2 }));
    assert!(retried.attempt_budget.is_none() && retried.heartbeat_budget.is_none());
    let repolled = accepted(&world, retried, Worker::Deliver(2).into())?;
    assert_eq!(repolled.attempt_budget, Some(Remaining { ticks: 2 }));
    assert_eq!(repolled.heartbeat_budget, Some(Remaining { ticks: 1 }));
    let next = world.checked_step(&repolled, Timer::Tick.into())?;
    assert_eq!(next[0].1, [Expired::Heartbeat]);
    let third = accepted(&world, &next[0].2, Worker::Deliver(3).into())?;
    let global = world.checked_step(&third, Timer::Tick.into())?;
    assert_eq!(global[0].1, [Expired::ScheduleToClose]);
    assert_eq!(global[0].2.retained_failure, Failure::ScheduleToClose);
    assert_eq!(global[0].2.terminal_outcome, Terminal::TimedOut);
    for (timeouts, expected) in [
        ([Some(1), None, None, None], Expired::ScheduleToStart),
        ([None, Some(1), None, Some(1)], Expired::StartToClose),
        ([None, Some(2), None, Some(1)], Expired::Heartbeat),
        ([Some(1), Some(1), Some(1), Some(1)], Expired::ScheduleToClose),
    ] {
        let world = configured(1, timeouts, true);
        let start = world.starts().remove(0);
        let before = if expected == Expired::ScheduleToStart {
            start
        } else {
            accepted(&world, &start, Worker::Deliver(1).into())?
        };
        let result = world.checked_step(&before, Timer::Tick.into())?;
        assert_eq!(result[0].1, [expected]);
        assert_eq!(result[0].2.terminal_outcome, Terminal::TimedOut);
    }
    Ok(())
}

#[test]
fn erased_budgets_and_present_zero_are_observable_to_full_state_laws() -> TestResult {
    let world = configured(2, [Some(2); 4], true);
    let start = world.starts().remove(0);
    for (index, replacement) in (0..4).flat_map(|index| {
        [None, Some(Remaining { ticks: 0 })].map(|replacement| (index, replacement))
    }) {
        let running = accepted(&world, &start, Worker::Deliver(1).into())?;
        let mut before = if index == 0 { start.clone() } else { running };
        let slot = match index {
            0 => &mut before.queued_budget,
            1 => &mut before.attempt_budget,
            2 => &mut before.execution_budget,
            _ => &mut before.heartbeat_budget,
        };
        *slot = replacement;
        let report = umpire_testing::Checker::explore_within(
            &world,
            vec![before],
            BUDGET,
            umpire_testing::Checker::CATALOG_LIMIT,
        )?
        .verify_all()?;
        assert_eq!(report.outcome(), umpire_dsl::CheckOutcome::Refuted);
    }
    let no_timer = configured(3, [None; 4], true);
    assert!(no_timer.checked_step(&no_timer.starts()[0], Timer::Tick.into())?.is_empty());
    Ok(())
}

#[path = "parameters_tests.rs"]
mod parameters_tests;

#[test]
fn authored_source_admits_without_graph_emission() -> TestResult {
    bound(world()).lift_source("model.rs", SOURCE)?;
    Ok(())
}

#[test]
fn protocol_local_adapter_and_feature_schema_share_exact_parameters_without_world_import()
-> TestResult {
    use umpire_temporal::activity_protocol::{adapter, parameters::Parameters};
    for maximum in 1..=3 {
        for cooperative in [false, true] {
            for timeouts in [[None; 4], [Some(1); 4], [Some(3), None, Some(2), Some(1)]] {
                let parameters = Parameters::new(maximum, timeouts, cooperative)?;
                let protocol = adapter::world(parameters)?;
                parameters.validate_world(&protocol)?;
                let [wait, run, global, heartbeat] =
                    timeouts.map(|raw| raw.map(crate::activity::activity::Ticks));
                let feature = crate::activity::activity::World::new(
                    maximum,
                    wait,
                    run,
                    global,
                    heartbeat,
                    cooperative,
                );
                parameters.validate_world(&feature)?;
            }
        }
    }
    Ok(())
}

fn selected_worlds() -> Vec<World> {
    vec![
        configured(1, [Some(1), Some(1), None, None], false),
        configured(3, [Some(1), Some(1), None, None], false),
        configured(3, [None; 4], true),
        world(),
        configured(1, [None, None, Some(3), None], true),
        configured(3, [None, None, None, Some(2)], true),
        configured(3, [Some(3); 4], true),
        configured(2, [Some(1); 4], true),
        configured(2, [None, Some(2), None, Some(2)], true),
        configured(3, [Some(3), Some(1), Some(3), None], true),
        configured(2, [None, Some(3), Some(3), Some(2)], true),
        configured(3, [Some(2), None, None, None], true),
    ]
}

fn scope(world: &World) -> Result<String, Box<dyn std::error::Error>> {
    use parameters::validate_effective_world;
    let values = validate_effective_world(world)?;
    let raw_timeouts = (values.start_to_close.is_some() || values.schedule_to_close.is_some())
        && (values.schedule_to_start.is_some() || values.schedule_to_close.is_none());
    Ok(format!(
        "{values:?}; timeout-scope={}; cancellation-scope={}; bound={BUDGET}",
        if raw_timeouts { "raw-realizable" } else { "abstract-checker" },
        if values.asynchronous_cancel { "cooperative" } else { "atomic-compatibility" }
    ))
}

fn world_reports() -> Result<String, Box<dyn std::error::Error>> {
    use std::fmt::Write;
    let mut report = String::new();
    for world in selected_worlds() {
        let graph = umpire_testing::Checker::explore(&world, BUDGET)?;
        assert!(!graph.counts.cut);
        assert_eq!(graph.verify_all()?.outcome(), umpire_dsl::CheckOutcome::Verified);
        let differential = umpire_testing::interpreted(|| world.clone(), None, BUDGET, None)?;
        writeln!(
            report,
            "{}; starts={}; states={}; edges={}; compiled/interpreted=agree",
            scope(&world)?,
            world.starts().len(),
            differential.states,
            differential.edges
        )?;
    }
    Ok(report)
}

pub(crate) fn scopes() -> umpire_testing::RunResult {
    use umpire_dsl::__lifted::Lifts;
    let model = World::__umpire_lifted().model.map_err(|error| format!("{error:?}"))?;
    umpire_temporal::activity_protocol::adapter::validate_state_schema(&model)?;
    parameters::validate_effective_world(&world())?;
    let mut files = umpire_testing::Files::from([
        ("worlds.txt".into(), world_reports()?),
        (
            "schema.txt".into(),
            format!(
                "source: shared/protocol/activity/model.rs\nWorld fields: {:#?}\nFull-state fields: {:#?}\n",
                model.world_fields, model.fields
            ),
        ),
    ]);
    for (index, world) in selected_worlds().into_iter().enumerate() {
        let graph = umpire_testing::Checker::explore(&world, BUDGET)?;
        let mut rendered = umpire_testing::render(&graph)?;
        rendered.insert("rows.txt".into(), umpire_testing::render_rows(&graph)?);
        rendered.insert("starts.txt".into(), format!("{:#?}\n", world.starts()));
        files.extend(
            rendered.into_iter().map(|(name, value)| (format!("worlds/{index:02}-{name}"), value)),
        );
    }
    Ok(files)
}

#[test]
fn selected_worlds_are_complete_compiled_interpreted_and_required_tlc_scopes() -> TestResult {
    use std::fmt::Write;
    let mut evidence = String::new();
    for world in selected_worlds() {
        let native = umpire_testing::Checker::explore(&world, BUDGET)?;
        assert!(!native.counts.cut);
        assert_eq!(native.verify_all()?.outcome(), umpire_dsl::CheckOutcome::Verified);
        if std::env::var("UMPIRE_BACKENDS").as_deref() == Ok("require") {
            let files = bound(world.clone()).check(&folder())?;
            writeln!(evidence, "{}\n{}", scope(&world)?, files["tlc.txt"])?;
        }
    }
    if let Some(path) = std::env::var_os("UMPIRE_PROTOCOL_SCOPE_REPORT") {
        std::fs::write(path, evidence)?;
    }
    Ok(())
}

#[test]
fn backend_refuses_wrong_world_cut_graph_and_unsupported_source() -> TestResult {
    let selected = bound(world());
    let wrong = bound(configured(1, [Some(1), Some(1), None, None], true))
        .lift_source("model.rs", SOURCE)?;
    let refusal = selected.check_module(&folder(), &wrong).err().ok_or("wrong World admitted")?;
    assert!(refusal.to_string().contains("World"), "{refusal}");
    let module = bound(world()).lift_source("model.rs", SOURCE)?;
    let mut wrong_start = module.clone();
    wrong_start.metadata.binding.as_mut().ok_or("missing World binding")?.starts.clear();
    let refusal = bound(world())
        .check_module(&folder(), &wrong_start)
        .err()
        .ok_or("missing complete starts admitted")?;
    assert!(refusal.to_string().contains("World"), "{refusal}");
    let graph = umpire_testing::Checker::explore(&world(), 1)?;
    assert!(graph.counts.cut);
    let native = umpire_testing::render(&graph)?;
    let refusal = umpire_testing::backend::check_exploration(
        &world(),
        &world().starts(),
        &graph,
        &module,
        &folder(),
        &native,
        None,
    )
    .err()
    .ok_or("cut graph admitted")?;
    assert!(refusal.to_string().contains("cut"), "{refusal}");
    let unsupported = umpire_testing::replace_in(
        SOURCE,
        "fn consume",
        "fn selected_attempts",
        "self.queued_budget =",
        "self.queued_budget = std::hint::black_box(None); self.queued_budget =",
    );
    assert!(bound(world()).lift_source("model.rs", &unsupported).is_err());
    Ok(())
}

#[path = "mutants.rs"]
mod mutants;
#[path = "view_tests.rs"]
mod view_tests;
