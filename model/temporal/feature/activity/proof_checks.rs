use umpire_testing::replace_code;
use super::{bound_for, configured};
use crate::activity::activity::{Action, Deadline, Model, Phase, Worker, World};
use umpire_dsl::{Assumption, CheckOutcome, Machine, Outcome, Phase as _, Rejection, __lifted::ConcreteOf};

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

fn state_value(value: umpire_dsl::Value) -> umpire_tlc::Value {
    use umpire_dsl::Value as Native;
    use umpire_tlc::Value;
    match value {
        Native::Bool(value) => Value::Bool(value),
        Native::Int(value) | Native::Bounded { raw: value, .. } => Value::Int(value),
        Native::Enum(value) => Value::Text(value.into()),
        Native::Optional(None) => Value::Int(0),
        Native::Optional(Some(value)) => {
            let Value::Int(raw) = state_value(*value) else { panic!("noninteger activity option") };
            Value::Int(raw + 1)
        }
        Native::Sequence(values) => Value::Seq(values.into_iter().map(state_value).collect()),
    }
}

fn matches_state(state: &Model, decoded: &umpire_tlc::State) -> bool {
    use umpire_dsl::Values;
    state.values().into_iter().all(|(name, value)| decoded.get(name) == Some(&state_value(value)))
}

fn global_disagreement(
    current: &Model,
    action: &Action,
    outcome: Outcome,
    facts: &[crate::activity::activity::ExpiredDeadline],
    destination: &Model,
    after: &umpire_tlc::TraceState,
) -> umpire_testing::TestResult {
    use umpire_tlc::Value;
    assert!(*action == Deadline::Tick.into() || *action == Worker::Heartbeat.into());
    assert!(!current.phase.closed());
    assert_eq!(outcome, Outcome::Accepted);
    assert_eq!(after.observers["__umpire_outcome"], Value::Text("Outcome::Accepted".into()));
    assert_eq!(after.observers["__umpire_facts"], Value::Seq(vec![]));
    if *action == Worker::Heartbeat.into() {
        assert_eq!(facts, []);
        assert_eq!(destination.phase, Phase::Started);
        assert_eq!(destination.schedule_to_close_remaining, current.schedule_to_close_remaining);
        assert!(
            current.schedule_to_close_remaining.ok_or("missing before reset budget")?.ticks < 3
        );
        assert_eq!(after.state["schedule_to_close_remaining"], Value::Int(4));
    } else {
        assert_eq!(current.schedule_to_close_remaining.ok_or("missing expiry budget")?.ticks, 1);
        assert_eq!(destination.phase, Phase::TimedOut);
        assert_eq!(facts, [crate::activity::activity::ExpiredDeadline::ScheduleToClose]);
        assert_eq!(after.state["phase"], Value::Text(format!("Phase::{:?}", current.phase)));
        assert_eq!(after.state["schedule_to_close_remaining"], Value::Int(1));
    }
    Ok(())
}

fn replay_tlc(
    world: &World,
    module: &umpire_tla::Module,
    trace: &umpire_tlc::Trace,
    expect_disagreement: bool,
) -> umpire_testing::TestResult {
    use umpire_dsl::{ActionValues, IntoValue};
    use umpire_tlc::Value;
    let graph = umpire_checker::Checker::explore(world, 4096)?;
    assert!(!graph.counts.cut);
    let mut pairs: Vec<_> = trace.states.windows(2).map(|pair| (&pair[0], &pair[1])).collect();
    if let Some(index) = trace.back_to {
        pairs.push((
            trace.states.last().ok_or("empty trace")?,
            trace.states.get(index).ok_or("bad back edge")?,
        ));
    }
    let mut actual_edges = 0;
    let mut disagreement = false;
    for (before, after) in pairs {
        if before.observers["__umpire_edge_tick"] == after.observers["__umpire_edge_tick"] {
            assert_eq!(before.state, after.state);
            continue;
        }
        actual_edges += 1;
        let current = graph
            .states()
            .find(|(_, state)| matches_state(state, &before.state))
            .map(|(_, state)| state)
            .ok_or("trace before state absent from healthy graph")?;
        let Value::Int(ordinal) = after.observers["__umpire_input"] else {
            return Err("missing input".into());
        };
        let ordinal = usize::try_from(ordinal)?;
        let action = world.actions().get(ordinal).ok_or("bad catalog ordinal")?.clone();
        let identity = umpire_tla::Concrete::of::<World>(&action);
        assert_eq!(module.metadata.catalog[ordinal], identity);
        assert_eq!(after.observers["__umpire_action"], Value::Text(identity.action.clone()));
        let Value::Int(choice) = after.observers["__umpire_choice"] else {
            return Err("missing choice".into());
        };
        let steps = world.checked_step(current, action.clone())?;
        let Some((outcome, facts, destination)) = steps.get(usize::try_from(choice)?) else {
            assert!(expect_disagreement && action == Deadline::Tick.into());
            disagreement = true;
            break;
        };
        if !matches_state(destination, &after.state) {
            assert!(expect_disagreement);
            global_disagreement(current, &action, *outcome, facts, destination, after)?;
            disagreement = true;
            break;
        }
        assert_eq!(after.observers["__umpire_outcome"], state_value(outcome.value()));
        assert_eq!(
            after.observers["__umpire_facts"],
            Value::Seq(
                facts
                    .iter()
                    .map(|fact| Value::Text(format!("ExpiredDeadline::{fact:?}")))
                    .collect()
            )
        );
    }
    assert!(actual_edges > 0);
    assert_eq!(disagreement, expect_disagreement);
    Ok(())
}

fn tlc() -> Result<umpire_tlc::Tlc, Box<dyn std::error::Error>> {
    let manifest = std::path::Path::new(env!("CARGO_MANIFEST_DIR"));
    Ok(umpire_tlc::Tlc::discover(&manifest.join("../../.."))?)
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
                let to = " /\\ WF_vars((__umpire_pending = 1) /\\ Worker_Poll /\\ Monitor_eventually_closed)";
                assert_eq!(mutant.tla.matches(&from).count(), 1);
                mutant.tla = mutant.tla.replacen(&from, to, 1);
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
                assert!(trace.states.iter().any(|state| state.observers.get("__umpire_action")
                    == Some(&umpire_tlc::Value::Text("Worker::Heartbeat".into()))));
            } else {
                assert!(
                    trace.states.iter().any(|state| state.state["attempt"]
                        == umpire_tlc::Value::Int(i128::from(maximum)))
                );
                assert!(trace.states.iter().any(|state| state.observers.get("__umpire_action")
                    == Some(&umpire_tlc::Value::Text("Worker::Poll".into()))
                    && state.observers.get("__umpire_outcome")
                        == Some(&umpire_tlc::Value::Text(
                            "Outcome::Rejected(FailedPrecondition)".into()
                        ))));
            }
            eprintln!(
                "activity fairness maximum={maximum} control={control} fair={:?} trace={trace:#?}",
                mutant.metadata.progress[0].fair_inputs
            );
            replay_tlc(&world, &mutant, trace, false)?;
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
        .find(|claim| claim.name == "state_bounds")
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
                assert_eq!(steps[0].2.phase, Phase::TimedOut);
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
    let expiry = "if self.schedule_to_close_remaining.is_some() && self.schedule_to_close_remaining.unwrap_or(Remaining { ticks: 0 }).ticks == 0 {";
    let tick = "Deadline::Tick if !self.closed() && (";
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
            replace_code(healthy, tick, "Deadline::Tick if false && !self.closed() && ("),
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
                replay_tlc(
                    &world,
                    &module,
                    claim.trace.as_ref().ok_or("missing global defect trace")?,
                    true,
                )?;
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
            "Invariant_state_bounds",
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
