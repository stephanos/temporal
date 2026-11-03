use super::{bound_for, configured};
use crate::activity::activity::{Action, Deadline, Model, Phase, Worker, World};
use umpire_dsl::{
    __lifted::ConcreteOf, Assumption, CheckOutcome, Machine, Outcome, Phase as _, Rejection,
};
use umpire_testing::replace_code;
use umpire_tla::observers;

fn source() -> Result<String, std::io::Error> {
    std::fs::read_to_string(
        std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("../feature/activity/model.rs"),
    )
}

fn source_tree(source: &str) -> Result<umpire_front::Lifted, Box<dyn std::error::Error>> {
    umpire_front::read(&syn::parse_file(source)?, &[])
        .map(|(tree, _, _)| tree)
        .map_err(|error| format!("{error:?}").into())
}

fn fair_actions(maximum: u8) -> Vec<Action> {
    (1..=maximum)
        .map(|attempt| Worker::Poll(attempt).into())
        .chain([Worker::RespondCompleted.into()])
        .collect()
}

#[test]
fn selected_worlds_have_exact_poll_and_completion_fairness() -> umpire_testing::TestResult {
    for maximum in [1, 2, 3] {
        let world = configured(maximum, true, [None; 4]);
        let expected = fair_actions(maximum);
        let native = world.assumptions();
        assert!(native.iter().any(|(_, scope)| scope.fair == expected));
        let parsed = source_tree(&source()?)?;
        let interpreted = umpire_checker::Interpreted::from_tree(world.clone(), &parsed)?;
        assert_eq!(interpreted.assumptions(), native);
        let module = bound_for(world.clone()).lift_source("model.rs", &source()?)?;
        let progress = module
            .metadata
            .progress
            .iter()
            .find(|claim| claim.name == "eventually_closed")
            .ok_or("missing closure claim")?;
        let ordinals: Vec<_> = expected
            .iter()
            .map(|action| {
                world
                    .actions()
                    .iter()
                    .position(|candidate| candidate == action)
                    .ok_or("fair action absent from catalog")
            })
            .collect::<Result<_, _>>()?;
        assert_eq!(progress.fair_inputs, ordinals);
        for (index, ordinal) in ordinals.iter().enumerate() {
            let expected = if index < usize::from(maximum) {
                umpire_tla::Concrete {
                    action: "Worker::Poll".into(),
                    fields: vec![(
                        "0".into(),
                        umpire_tla::Literal::Int(i128::try_from(index + 1)?),
                    )],
                }
            } else {
                umpire_tla::Concrete { action: "Worker::RespondCompleted".into(), fields: vec![] }
            };
            assert_eq!(module.metadata.catalog[*ordinal], expected);
        }
        assert_eq!(progress.allowed_inputs, (0..world.actions().len()).collect::<Vec<_>>());
        let header = "ProgressSpec_eventually_closed ==";
        let spec = module
            .tla
            .lines()
            .find(|line| line.starts_with(header))
            .ok_or("missing closure specification")?;
        assert_eq!(spec.matches("WF_vars(").count(), usize::from(maximum) + 1);
        for attempt in 1..=maximum {
            assert!(
                spec.contains(&format!("WF_vars(Fair_eventually_closed_Worker_Poll_{attempt})")),
                "{spec}"
            );
        }
        assert!(spec.contains("WF_vars(Fair_eventually_closed_Worker_RespondCompleted)"));
        let graph = umpire_checker::Checker::explore(&world, 4096)?;
        assert!(!graph.counts.cut);
        graph.verify_all()?.assert_verified();
        for (_, state) in graph.states() {
            for action in world.actions() {
                assert_eq!(
                    interpreted.checked_step(state, action.clone())?,
                    world.checked_step(state, action)?
                );
            }
        }
    }
    Ok(())
}

fn replay_native_lasso(
    world: &World,
    result: &umpire_checker::ClaimResult<Model, Action>,
) -> umpire_testing::TestResult {
    assert_eq!(result.outcome, CheckOutcome::Refuted);
    assert_eq!(result.verdict, Some(umpire_checker::ProgressVerdict::Cycle));
    let path = result.path.as_ref().ok_or("missing native lasso")?;
    let cycle = path.cycle_start.ok_or("missing native cycle index")?;
    let mut current = path.start.clone();
    let mut entry = None;
    for (index, edge) in path.steps.iter().enumerate() {
        if index == cycle {
            entry = Some(current.clone());
        }
        let steps = world.checked_step(&current, edge.action.clone())?;
        let (outcome, facts, after) = steps.get(edge.choice).ok_or("missing replay choice")?;
        assert_eq!(after, &edge.after);
        assert!(!after.phase.closed());
        assert_eq!(facts.as_slice(), []);
        assert!(matches!(
            outcome,
            Outcome::Accepted | Outcome::Rejected(Rejection::FailedPrecondition)
        ));
        current = after.clone();
    }
    assert_eq!(Some(current), entry);
    Ok(())
}

#[test]
fn native_fairness_omissions_expose_heartbeat_and_retry_starvation() -> umpire_testing::TestResult {
    for maximum in [1, 2, 3] {
        let world = configured(maximum, true, [None; 4]);
        let graph = umpire_checker::Checker::explore(&world, 4096)?;
        assert!(!graph.counts.cut);
        let progress = world
            .progress()
            .into_iter()
            .find(|claim| claim.name == "eventually_closed")
            .ok_or("missing closure claim")?;
        let missing_close = progress.assuming(Assumption::fair(
            &(1..=maximum).map(Worker::Poll).chain([Worker::Heartbeat]).collect::<Vec<_>>(),
        ));
        replay_native_lasso(&world, &graph.verify_progress(&world, &missing_close)?)?;
        if maximum > 1 {
            let progress = world
                .progress()
                .into_iter()
                .find(|claim| claim.name == "eventually_closed")
                .ok_or("missing closure claim")?;
            let stale = fair_actions(maximum - 1);
            let result =
                graph.verify_progress(&world, &progress.assuming(Assumption::fair(&stale)))?;
            replay_native_lasso(&world, &result)?;
            let path = result.path.as_ref().ok_or("missing stale Poll trace")?;
            assert!(path.steps.iter().any(|step| step.after.attempt == maximum));
        }
    }
    Ok(())
}

#[test]
fn total_tick_bound_laws_hold_on_complete_interleaving_graphs() -> umpire_testing::TestResult {
    for maximum in [1, 2, 3] {
        for ticks in [1, 2, 3] {
            let world = configured(maximum, true, [Some(3), Some(3), Some(ticks), Some(3)]);
            let graph = umpire_checker::Checker::explore(&world, 4096)?;
            assert!(!graph.counts.cut);
            graph.verify_all()?.assert_verified();
            assert_eq!(world.starts()[0].schedule_to_close_remaining.map(|r| r.ticks), Some(ticks));
            for (id, before) in graph.states() {
                let edges: Vec<_> = graph.rows(id)?.collect();
                assert_eq!(
                    edges.iter().any(|edge| edge.action == Deadline::Tick.into()),
                    !before.phase.closed()
                );
                for edge in edges {
                    let after = &edge.emitted.2;
                    if !before.phase.closed() && !after.phase.closed() {
                        let old = before
                            .schedule_to_close_remaining
                            .ok_or("missing global before")?
                            .ticks;
                        let new =
                            after.schedule_to_close_remaining.ok_or("missing global after")?.ticks;
                        assert!(new > 0 && new <= old);
                        if edge.action == Deadline::Tick.into() {
                            assert_eq!(old, new + 1);
                        }
                    }
                }
            }
        }
    }
    Ok(())
}

#[test]
fn shared_replay_checks_typed_activity_edges_and_retains_native_disagreement()
-> umpire_testing::TestResult {
    use umpire_dsl::{IntoValue, Values};
    use umpire_tla::{
        observers::{ACTION, CHOICE, EDGE_TICK, FACTS, INPUT, OUTCOME},
        replay,
    };
    use umpire_tlc::{State, Trace, TraceState, Value};
    let world = configured(1, true, [None, None, Some(1), None]);
    let module = bound_for(world.clone()).lift_source("model.rs", &source()?)?;
    let starts = world.starts();
    let action = Deadline::Tick.into();
    let ordinal =
        world.actions().iter().position(|candidate| *candidate == action).ok_or("missing tick")?;
    let steps = world.checked_step(&starts[0], action)?;
    let (outcome, facts, successor) = steps.first().ok_or("missing expiry edge")?;
    let state = |state: &Model| {
        replay::fields(
            state.values(),
            module.metadata.fields.iter().map(|field| (field.name.as_str(), &field.ty)),
        )
    };
    let after = TraceState {
        state: state(successor)?,
        observers: State::from([
            (EDGE_TICK.into(), Value::Bool(true)),
            (ACTION.into(), Value::Text("Deadline::Tick".into())),
            (INPUT.into(), Value::Int(i128::try_from(ordinal)?)),
            (CHOICE.into(), Value::Int(0)),
            (OUTCOME.into(), replay::admitted(outcome.value(), &module.metadata.outcome_type)?),
            (
                FACTS.into(),
                Value::Seq(vec![Value::Text("ExpiredDeadline::ScheduleToClose".into())]),
            ),
        ]),
    };
    assert_eq!(facts, &[crate::activity::activity::ExpiredDeadline::ScheduleToClose]);
    let mut before = after.clone();
    before.state = state(&starts[0])?;
    before.observers.insert(EDGE_TICK.into(), Value::Bool(false));
    let trace = Trace { states: vec![before, after.clone()], back_to: None };
    umpire_testing::replay_trace(&world, &starts, &module, &trace)?;
    for (field, value, reason) in [
        ("phase", Value::Text("Phase::Started".into()), "successor differs at field phase"),
        (
            OUTCOME,
            Value::Text("Outcome::Rejected(NotFound)".into()),
            "outcome/facts differ at outcome",
        ),
        (FACTS, Value::Seq(vec![]), "outcome/facts differ at facts"),
        (INPUT, Value::Int(i128::try_from(world.actions().len())?), "outside the catalog"),
        (CHOICE, Value::Int(1), "choice is not emitted"),
        (EDGE_TICK, Value::Bool(false), "observer-only TLC step changes Model state"),
    ] {
        let mut forged = trace.clone();
        let changed = &mut forged.states[1];
        if field == "phase" {
            changed.state.insert(field.into(), value);
        } else {
            changed.observers.insert(field.into(), value);
        }
        let error = umpire_testing::replay_trace(&world, &starts, &module, &forged)
            .err()
            .ok_or("forged activity edge replayed")?;
        assert_eq!(error.transition, 1);
        assert!(error.reason.contains(reason), "{field}: {error}");
        if matches!(field, "phase" | OUTCOME | FACTS) {
            let native = error.native.ok_or("missing native disagreement projection")?;
            assert_eq!(native.state, after.state);
            assert_eq!(native.outcome, after.observers[OUTCOME]);
            assert_eq!(Value::Seq(native.facts), after.observers[FACTS]);
        } else {
            assert!(error.native.is_none());
        }
    }
    let error = umpire_testing::replay_trace(&world, &[], &module, &trace)
        .err()
        .ok_or("trace accepted outside selected starts")?;
    assert_eq!(error.transition, 0);
    assert!(error.reason.contains("does not begin at a Model start"));
    Ok(())
}

fn global_disagreement(
    trace: &umpire_tlc::Trace,
    disagreement: &umpire_testing::Divergence,
) -> umpire_testing::TestResult {
    use umpire_tlc::Value;
    let before = &trace.states[disagreement.transition - 1];
    let after = &trace.states[disagreement.transition];
    let native = disagreement.native.as_ref().ok_or("missing native global-budget edge")?;
    let heartbeat = after.observers[observers::ACTION] == Value::Text("Worker::Heartbeat".into());
    assert!(
        heartbeat || after.observers[observers::ACTION] == Value::Text("Deadline::Tick".into())
    );
    assert_eq!(native.outcome, Value::Text("Outcome::Accepted".into()));
    assert_eq!(after.observers[observers::OUTCOME], native.outcome);
    assert_eq!(after.observers[observers::FACTS], Value::Seq(vec![]));
    if heartbeat {
        assert!(
            disagreement
                .to_string()
                .contains("successor differs at field schedule_to_close_remaining")
        );
        assert_eq!(native.facts, []);
        assert_eq!(native.state["phase"], Value::Text("Phase::Started".into()));
        assert_eq!(
            native.state["schedule_to_close_remaining"],
            before.state["schedule_to_close_remaining"]
        );
        assert!(matches!(before.state["schedule_to_close_remaining"], Value::Int(1..=3)));
        assert_eq!(after.state["schedule_to_close_remaining"], Value::Int(4));
    } else {
        assert!(disagreement.to_string().contains("successor differs at field phase"));
        assert_eq!(before.state["schedule_to_close_remaining"], Value::Int(2));
        assert_eq!(native.state["phase"], Value::Text("Phase::TimedOut".into()));
        assert_eq!(native.facts, [Value::Text("ExpiredDeadline::ScheduleToClose".into())]);
        assert_eq!(after.state["phase"], before.state["phase"]);
        assert_eq!(after.state["schedule_to_close_remaining"], Value::Int(1));
    }
    Ok(())
}

fn tlc() -> Result<umpire_tlc::Tlc, Box<dyn std::error::Error>> {
    Ok(umpire_tlc::Tlc::discover(&super::framework_root())?)
}

fn safety_only(mut module: umpire_tla::Module, properties: &[&str]) -> umpire_tla::Module {
    module.metadata.progress.clear();
    module.metadata.negative.clear();
    module
        .metadata
        .invariants
        .retain(|claim| properties.contains(&format!("Invariant_{}", claim.name).as_str()));
    module
        .metadata
        .postconditions
        .retain(|claim| properties.contains(&format!("Post_{}", claim.name).as_str()));
    module.cfg = module
        .cfg
        .lines()
        .filter(|line| {
            (!line.starts_with("INVARIANT") && !line.starts_with("PROPERTY"))
                || line.split_whitespace().last().is_some_and(|name| properties.contains(&name))
        })
        .flat_map(|line| [line, "\n"])
        .collect();
    module
}

fn closure_only(mut module: umpire_tla::Module) -> umpire_tla::Module {
    module.metadata.progress.retain(|claim| claim.name == "eventually_closed");
    module.metadata.invariants.clear();
    module.metadata.postconditions.clear();
    module.metadata.negative.clear();
    module.cfg = module
        .cfg
        .lines()
        .filter(|line| !line.starts_with("INVARIANT") && !line.starts_with("PROPERTY"))
        .flat_map(|line| [line, "\n"])
        .collect();
    module
}

fn fair_leaf(maximum: u8, control: &str) -> String {
    let mut actions: Vec<_> =
        (1..=maximum).map(|attempt| format!("Worker::Poll({attempt})")).collect();
    match control {
        "missing_close" => actions.push("Worker::Heartbeat".into()),
        "missing_current" => {
            actions.pop();
            actions.push("Worker::RespondCompleted".into());
        }
        "stale_payload" => {
            actions.pop();
            actions.push("Worker::Poll(1)".into());
            actions.push("Worker::RespondCompleted".into());
        }
        _ => actions.push("Worker::RespondCompleted".into()),
    }
    format!("fair([{}])", actions.join(", "))
}

#[test]
#[expect(clippy::print_stderr, reason = "Retain exact typed fairness lasso evidence")]
fn actual_tlc_selected_fairness_and_starvation_lassos() -> umpire_testing::TestResult {
    use std::fmt::Write as _;
    use umpire_tlc::Verdict::{Refuted, Verified};
    if std::env::var("UMPIRE_BACKENDS").as_deref() != Ok("require") {
        return Ok(());
    }
    for maximum in [1, 2, 3] {
        let world = configured(maximum, true, [None; 4]);
        let module = closure_only(bound_for(world.clone()).lift_source("model.rs", &source()?)?);
        let result = tlc()?.check_module(&module)?;
        assert_eq!(result.properties["Progress_eventually_closed"].verdict, Verified);
        let graph = umpire_checker::Checker::explore(&world, 4096)?;
        assert!(!graph.counts.cut);
        assert_eq!(result.states.len(), graph.counts.reached);
        for control in ["missing_close", "missing_current", "stale_payload", "union"] {
            if maximum == 1 && control != "missing_close" {
                continue;
            }
            let mutant = if control == "union" {
                let mut mutant = module.clone();
                let mut from = String::new();
                for attempt in 1..=maximum {
                    write!(from, " /\\ WF_vars(Fair_eventually_closed_Worker_Poll_{attempt})")?;
                }
                let to = format!(
                    " /\\ WF_vars(({} = 1) /\\ Worker_Poll /\\ Monitor_eventually_closed)",
                    observers::PENDING
                );
                assert_eq!(mutant.tla.matches(&from).count(), 1);
                mutant.tla = mutant.tla.replacen(&from, &to, 1);
                mutant
            } else {
                let healthy = source()?;
                let original = fair_leaf(maximum, "healthy");
                let replacement = fair_leaf(maximum, control);
                let changed = replace_code(&healthy, &original, &replacement);
                let parsed = source_tree(&changed)?;
                let interpreted = umpire_checker::Interpreted::from_tree(world.clone(), &parsed)?;
                let graph = umpire_checker::Checker::explore(&interpreted, 4096)?;
                assert!(!graph.counts.cut);
                let report = graph.verify_all()?;
                assert!(report.claims.iter().any(|claim| claim.name == "eventually_closed"
                    && claim.outcome == CheckOutcome::Refuted));
                closure_only(bound_for(world.clone()).lift_source("model.rs", &changed)?)
            };
            let result = tlc()?.check_module(&mutant)?;
            let property = &result.properties["Progress_eventually_closed"];
            assert_eq!(property.verdict, Refuted, "max={maximum}, {control}");
            let trace = property.trace.as_ref().ok_or("missing fairness lasso")?;
            assert!(trace.back_to.is_some());
            if control == "missing_close" {
                assert!(trace.states.iter().any(|state| state.observers.get(observers::ACTION)
                    == Some(&umpire_tlc::Value::Text("Worker::Heartbeat".into()))));
            } else {
                assert!(
                    trace.states.iter().any(|state| state.state["attempt"]
                        == umpire_tlc::Value::Int(i128::from(maximum)))
                );
                assert!(trace.states.iter().any(|state| state.observers.get(observers::ACTION)
                    == Some(&umpire_tlc::Value::Text("Worker::Poll".into()))
                    && state.observers.get(observers::OUTCOME)
                        == Some(&umpire_tlc::Value::Text(
                            "Outcome::Rejected(FailedPrecondition)".into()
                        ))));
            }
            eprintln!(
                "activity fairness maximum={maximum} control={control} fair={:?} trace={trace:#?}",
                mutant.metadata.progress[0].fair_inputs
            );
            umpire_testing::replay_trace(&world, &world.starts(), &mutant, trace)?;
        }
    }
    Ok(())
}

#[test]
fn forged_countdowns_refute_bounds_and_armed_zero_expires() -> umpire_testing::TestResult {
    use crate::activity::activity::{ExpiredDeadline, Remaining};
    use umpire_dsl::Verdict;
    let world = configured(3, true, [Some(3); 4]);
    let bounds = world
        .invariants()
        .into_iter()
        .find(|claim| claim.name == "countdowns_within_budget")
        .ok_or("missing bounds invariant")?;
    let start = world.starts()[0].clone();
    let started = world.step(&start, Worker::Poll(1).into())[0].2.clone();
    for field in 0..4 {
        for ticks in [0, 4] {
            let mut forged = if field == 0 { start.clone() } else { started.clone() };
            let slot = match field {
                0 => &mut forged.schedule_to_start_remaining,
                1 => &mut forged.start_to_close_remaining,
                2 => &mut forged.schedule_to_close_remaining,
                _ => &mut forged.heartbeat_remaining,
            };
            *slot = Some(Remaining { ticks });
            assert_eq!(bounds.evaluate(&world, &forged).verdict, Verdict::Fails);
            if ticks == 0 {
                let steps = world.checked_step(&forged, Deadline::Tick.into())?;
                assert_eq!(steps.len(), 1);
                assert_eq!(steps[0].0, Outcome::Accepted);
                let mut expected = forged.clone();
                if matches!(field, 1 | 3) {
                    expected.phase = Phase::Scheduled;
                    expected.attempt = 2;
                    expected.schedule_to_start_remaining = Some(Remaining { ticks: 3 });
                    expected.start_to_close_remaining = None;
                    expected.heartbeat_remaining = None;
                    expected.schedule_to_close_remaining = Some(Remaining { ticks: 2 });
                } else {
                    expected.phase = Phase::TimedOut;
                    expected.cancel_requested = false;
                    expected.schedule_to_start_remaining = None;
                    expected.start_to_close_remaining = None;
                    expected.schedule_to_close_remaining = None;
                    expected.heartbeat_remaining = None;
                }
                assert_eq!(steps[0].2, expected);
                assert_eq!(
                    steps[0].1,
                    [match field {
                        0 => ExpiredDeadline::ScheduleToStart,
                        1 => ExpiredDeadline::StartToClose,
                        2 => ExpiredDeadline::ScheduleToClose,
                        _ => ExpiredDeadline::Heartbeat,
                    }]
                );
            }
        }
    }
    let post = world
        .postconditions()
        .into_iter()
        .find(|claim| claim.name == "global_never_replenished")
        .ok_or("missing nonreplenishment law")?;
    let before = world.step(&started, Deadline::Tick.into())[0].2.clone();
    let mut wrong = before.clone();
    wrong.schedule_to_close_remaining = Some(Remaining { ticks: 3 });
    assert_eq!(
        post.holds(&world, &before, &Worker::Heartbeat.into(), &(Outcome::Accepted, vec![], wrong)),
        Verdict::Fails
    );
    Ok(())
}

fn global_mutants(healthy: &str) -> Vec<(&'static str, String, &'static str)> {
    let reset = "Worker::Heartbeat if self.phase == Phase::Started => {";
    let expiry = "if self.schedule_to_close_remaining == Some(Remaining { ticks: 0 }) {";
    let tick = "Deadline::Tick if !self.closed() && self.armed()";
    vec![
        (
            "reset",
            replace_code(
                healthy,
                reset,
                &format!(
                    "{reset}\n                    self.schedule_to_close_remaining = Some(Remaining {{ ticks: world.schedule_to_close.unwrap_or(Ticks(1)).0 }});"
                ),
            ),
            "Post_global_never_replenished",
        ),
        (
            "broken_expiry",
            replace_code(healthy, expiry, "if false {"),
            "Post_tick_descends_and_expires",
        ),
        (
            "missing_tick",
            replace_code(
                healthy,
                tick,
                "Deadline::Tick if false && !self.closed() && self.armed()",
            ),
            "",
        ),
    ]
}

#[test]
#[expect(clippy::print_stderr, reason = "Retain exact typed global-budget counterexamples")]
fn source_only_global_budget_controls_keep_laws_and_discriminate() -> umpire_testing::TestResult {
    let healthy = source()?;
    let world = configured(3, true, [None, None, Some(3), None]);
    for (name, mutant, property) in global_mutants(&healthy) {
        assert_eq!(
            healthy.split("        #[postcondition]").skip(1).collect::<Vec<_>>(),
            mutant.split("        #[postcondition]").skip(1).collect::<Vec<_>>()
        );
        let parsed = source_tree(&mutant)?;
        let interpreted = umpire_checker::Interpreted::from_tree(world.clone(), &parsed)?;
        let graph = umpire_checker::Checker::explore(&interpreted, 4096)?;
        assert!(!graph.counts.cut);
        if name == "missing_tick" {
            let start = world.starts()[0].clone();
            assert_eq!(world.step(&start, Deadline::Tick.into()).len(), 1);
            assert!(interpreted.step(&start, Deadline::Tick.into()).is_empty());
        } else {
            let report = graph.verify_all()?;
            assert_eq!(report.outcome(), CheckOutcome::Refuted, "{name}");
            assert!(
                report
                    .claims
                    .iter()
                    .any(|claim| claim.name == property.trim_start_matches("Post_")
                        && claim.outcome == CheckOutcome::Refuted),
                "{report}"
            );
        }
        if std::env::var("UMPIRE_BACKENDS").as_deref() == Ok("require") {
            let module = safety_only(
                bound_for(world.clone()).lift_source("model.rs", &mutant)?,
                &[property],
            );
            let result = tlc()?.check_module(&module)?;
            if name == "missing_tick" {
                let healthy_module =
                    safety_only(bound_for(world.clone()).lift_source("model.rs", &healthy)?, &[]);
                let healthy_result = tlc()?.check_module(&healthy_module)?;
                assert!(result.states.len() < healthy_result.states.len());
            } else {
                let claim = &result.properties[property];
                assert_eq!(claim.verdict, umpire_tlc::Verdict::Refuted, "{name}");
                eprintln!("activity global control={name} trace={:#?}", claim.trace);
                let trace = claim.trace.as_ref().ok_or("missing global defect trace")?;
                let disagreement =
                    umpire_testing::replay_trace(&world, &world.starts(), &module, trace)
                        .err()
                        .ok_or("global-budget mutant replayed against healthy Model")?;
                global_disagreement(trace, &disagreement)?;
            }
        }
    }
    Ok(())
}

#[test]
fn actual_tlc_global_tick_law_bundle_on_selected_worlds() -> umpire_testing::TestResult {
    if std::env::var("UMPIRE_BACKENDS").as_deref() != Ok("require") {
        return Ok(());
    }
    for (maximum, ticks) in [(1, 1), (2, 2), (3, 3)] {
        let world = configured(maximum, true, [Some(3), Some(3), Some(ticks), Some(3)]);
        let properties = [
            "Invariant_policy_in_range",
            "Invariant_attempt_within_policy",
            "Invariant_countdowns_within_budget",
            "Invariant_deadline_arming",
            "Post_global_never_replenished",
            "Post_tick_descends_and_expires",
            "Post_terminal_remains",
        ];
        let module =
            safety_only(bound_for(world).lift_source("model.rs", &source()?)?, &properties);
        let result = tlc()?.check_module(&module)?;
        for property in properties {
            assert_eq!(
                result
                    .properties
                    .get(property)
                    .ok_or_else(|| format!(
                        "missing {property}; available {:?}",
                        result.properties.keys()
                    ))?
                    .verdict,
                umpire_tlc::Verdict::Verified
            );
        }
    }
    Ok(())
}

#[test]
fn source_only_pending_cancellation_outcome_controls_keep_independent_laws()
-> umpire_testing::TestResult {
    use umpire_testing::replace_in;

    let healthy = source()?;
    for (start, end, original, replacement, deadlines, property) in [
        (
            "if retryable && self.cancel_requested {",
            "else if retryable && self.attempt < world.maximum_attempts",
            "self.phase = Phase::Failed;",
            "self.phase = Phase::Canceled;",
            [None; 4],
            "pending_retryable_failure_fails",
        ),
        (
            "fn time_out_attempt(",
            "fn clear_terminal(",
            "if self.cancel_requested {\n                self.time_out();",
            "if self.cancel_requested {\n                self.phase = Phase::Canceled; self.clear_terminal();",
            [None, Some(1), None, None],
            "tick_priority",
        ),
        (
            "fn time_out_attempt(",
            "fn clear_terminal(",
            "if self.cancel_requested {\n                self.time_out();",
            "if self.cancel_requested {\n                self.phase = Phase::Canceled; self.clear_terminal();",
            [None, None, None, Some(1)],
            "tick_priority",
        ),
    ] {
        let changed = replace_in(&healthy, start, end, original, replacement);
        assert_eq!(
            healthy.split("        #[invariant]").skip(1).collect::<Vec<_>>(),
            changed.split("        #[invariant]").skip(1).collect::<Vec<_>>()
        );
        let parsed = source_tree(&changed)?;
        for maximum in 1..=3 {
            let world = configured(maximum, true, deadlines);
            let interpreted = umpire_checker::Interpreted::from_tree(world, &parsed)?;
            let graph = umpire_checker::Checker::explore(&interpreted, 4096)?;
            assert!(!graph.counts.cut);
            let report = graph.verify_all()?;
            assert!(
                report
                    .claims
                    .iter()
                    .any(|claim| claim.name == property && claim.outcome == CheckOutcome::Refuted),
                "{property}: {report}"
            );
        }
    }
    Ok(())
}

fn timeout_retry_mutant(healthy: &str, name: &str) -> String {
    use umpire_testing::replace_in;

    let local = "fn time_out_attempt(";
    let retry = "fn retry(";
    let end = "fn clear_terminal(";
    match name {
        "old_terminalization" => replace_in(
            healthy,
            local,
            end,
            "else if self.attempt < world.maximum_attempts",
            "else if false",
        ),
        "retry_at_cap" => replace_in(
            healthy,
            local,
            end,
            "self.attempt < world.maximum_attempts",
            "self.attempt <= world.maximum_attempts",
        ),
        "retry_pending" => replace_in(healthy, local, end, "if self.cancel_requested", "if false"),
        "global_reset" => replace_in(
            healthy,
            retry,
            end,
            "self.attempt += 1;",
            "self.attempt += 1; self.schedule_to_close_remaining = Some(Remaining { ticks: world.schedule_to_close.unwrap_or(Ticks(1)).0 });",
        ),
        "wrong_waiting_rearm" => replace_in(
            healthy,
            retry,
            end,
            "ticks: world.schedule_to_start.unwrap_or(Ticks(1)).0",
            "ticks: 1",
        ),
        "wrong_attempt" => {
            replace_in(healthy, retry, end, "self.attempt += 1;", "self.attempt += 2;")
        }
        _ => replace_code(
            healthy,
            "if self.schedule_to_close_remaining == Some(Remaining { ticks: 0 }) {",
            "if false {",
        ),
    }
}

#[test]
#[expect(clippy::print_stderr, reason = "Retain exact local-timeout mutation counterexamples")]
fn source_only_timeout_retry_controls_discriminate_exact_edges() -> umpire_testing::TestResult {
    use crate::activity::activity::ExpiredDeadline;
    use umpire_testing::replace_once;

    let healthy = source()?;
    for (index, fact) in [(1, ExpiredDeadline::StartToClose), (3, ExpiredDeadline::Heartbeat)] {
        for name in [
            "old_terminalization",
            "retry_at_cap",
            "retry_pending",
            "global_reset",
            "wrong_waiting_rearm",
            "wrong_attempt",
            "global_loses_priority",
        ] {
            let maximum = if name == "retry_at_cap" { 1 } else { 3 };
            let pending = name == "retry_pending";
            let global = if name == "global_loses_priority" { 1 } else { 3 };
            let mut deadlines = [Some(3), None, Some(global), None];
            deadlines[index] = Some(1);
            let world = configured(maximum, true, deadlines);
            let start = world.starts()[0].clone();
            let mut before = world.checked_step(&start, Worker::Poll(1).into())?[0].2.clone();
            if pending {
                before = world
                    .checked_step(&before, crate::activity::activity::Client::Cancel.into())?[0]
                    .2
                    .clone();
            }
            let changed = timeout_retry_mutant(&healthy, name);
            let parsed = source_tree(&changed)?;
            let interpreted = umpire_checker::Interpreted::from_tree(world.clone(), &parsed)?;
            let native = world.checked_step(&before, Deadline::Tick.into())?;
            let mutant = interpreted.checked_step(&before, Deadline::Tick.into())?;
            assert_eq!(native.len(), 1);
            assert_eq!(mutant.len(), 1);
            assert_eq!(native[0].0, Outcome::Accepted);
            assert_eq!(
                native[0].1,
                [if global == 1 { ExpiredDeadline::ScheduleToClose } else { fact }]
            );
            assert_ne!(mutant, native, "{name}: {fact:?}");
            let graph = umpire_checker::Checker::explore(&interpreted, 4096)?;
            assert!(!graph.counts.cut);
            let report = graph.verify_all()?;
            assert_eq!(report.outcome(), CheckOutcome::Refuted, "{name}: {fact:?}: {report}");
            eprintln!(
                "activity retry control={name} fact={fact:?} before={before:?} healthy={native:?} mutant={mutant:?} laws={report}"
            );

            if std::env::var("UMPIRE_BACKENDS").as_deref() == Ok("require") {
                let witness = format!(
                    "#[postcondition(after = Deadline::Tick)] fn false_local_expiry(&self, before: &Self) {{ check!(before.phase != Phase::Started || before.attempt != 1 || before.cancel_requested != {pending} || self.phase == before.phase); }}\n        #[assumption]"
                );
                let source = replace_once(&changed, "#[assumption]", &witness);
                let module = safety_only(
                    bound_for(world.clone()).lift_source("model.rs", &source)?,
                    &["Post_false_local_expiry"],
                );
                let result = tlc()?.check_module(&module)?;
                let claim = &result.properties["Post_false_local_expiry"];
                assert_eq!(claim.verdict, umpire_tlc::Verdict::Refuted, "{name}: {fact:?}");
                let trace = claim.trace.as_ref().ok_or("missing local-timeout mutant trace")?;
                let disagreement =
                    umpire_testing::replay_trace(&world, &world.starts(), &module, trace)
                        .err()
                        .ok_or("local-timeout mutant replayed against healthy Model")?;
                assert!(disagreement.native.is_some(), "{name}: {disagreement}");
                eprintln!(
                    "activity retry TLC control={name} fact={fact:?} trace={trace:#?} disagreement={disagreement}"
                );
            }
        }
    }
    Ok(())
}
