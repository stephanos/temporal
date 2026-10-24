//! Evidence-only acknowledgement replay against the committed activity library and source.
use umpire_dsl::{Machine, Outcome, Rejection};
use umpire_temporal::activity::{Action, ExpiredDeadline, Model, Phase, Remaining, Worker, World};
fn check_trace_state(native: &Model, state: &umpire_tlc::State) {
    use umpire_tlc::Value;
    let remaining =
        |value: Option<Remaining>| Value::Int(value.map_or(0, |value| i128::from(value.ticks) + 1));
    assert_eq!(state.len(), 7);
    assert_eq!(
        state["phase"],
        Value::Text(format!("Phase::{:?}", native.phase))
    );
    assert_eq!(state["attempt"], Value::Int(i128::from(native.attempt)));
    assert_eq!(
        state["cancel_requested"],
        Value::Bool(native.cancel_requested)
    );
    for (field, expected) in [
        (
            "schedule_to_start_remaining",
            native.schedule_to_start_remaining,
        ),
        ("start_to_close_remaining", native.start_to_close_remaining),
        (
            "schedule_to_close_remaining",
            native.schedule_to_close_remaining,
        ),
        ("heartbeat_remaining", native.heartbeat_remaining),
    ] {
        assert_eq!(state[field], remaining(expected), "{field}");
    }
}

type Replayed = (Model, Action, Outcome, Vec<ExpiredDeadline>, Model);

fn replay(
    world: &World,
    module: &umpire_tla::Module,
    trace: &umpire_tlc::Trace,
) -> Result<Vec<Replayed>, Box<dyn std::error::Error>> {
    use umpire_tlc::Value;
    let mut current = world.starts().remove(0);
    check_trace_state(&current, &trace.states.first().ok_or("empty trace")?.state);
    let mut replayed = Vec::new();
    let mut pairs: Vec<_> = trace
        .states
        .windows(2)
        .map(|pair| (&pair[0], &pair[1]))
        .collect();
    if let Some(index) = trace.back_to {
        pairs.push((
            trace.states.last().ok_or("empty trace")?,
            trace.states.get(index).ok_or("bad back edge")?,
        ));
    }
    for (before, after) in pairs {
        check_trace_state(&current, &before.state);
        if before.observers["__umpire_edge_tick"] == after.observers["__umpire_edge_tick"] {
            assert_eq!(before.state, after.state);
            continue;
        }
        let Value::Int(ordinal) = after.observers["__umpire_input"] else {
            return Err("missing input ordinal".into());
        };
        let ordinal = usize::try_from(ordinal)?;
        let action = world
            .actions()
            .get(ordinal)
            .ok_or("invalid action ordinal")?
            .clone();
        let identity = concrete(&action);
        assert_eq!(
            after.observers["__umpire_action"],
            Value::Text(identity.action.clone())
        );
        assert_eq!(module.metadata.catalog[ordinal], identity);
        let Value::Int(choice) = after.observers["__umpire_choice"] else {
            return Err("missing choice".into());
        };
        let steps = world.checked_step(&current, action.clone())?;
        let (outcome, facts, destination) = steps
            .get(usize::try_from(choice)?)
            .ok_or("unavailable native edge")?;
        check_trace_state(destination, &after.state);
        let expected_outcome = match outcome {
            Outcome::Accepted => "Outcome::Accepted",
            Outcome::Rejected(Rejection::NotFound) => "Outcome::Rejected(NotFound)",
            Outcome::Rejected(Rejection::FailedPrecondition) => {
                "Outcome::Rejected(FailedPrecondition)"
            }
            other @ Outcome::Rejected(_) => panic!("unexpected outcome: {other:?}"),
        };
        assert_eq!(
            after.observers["__umpire_outcome"],
            Value::Text(expected_outcome.into())
        );
        let expected_facts = facts
            .iter()
            .map(|fact| Value::Text(format!("ExpiredDeadline::{fact:?}")))
            .collect();
        if after.observers["__umpire_facts"] != Value::Seq(expected_facts) {
            return Err("outcome/facts differ from exact native edge".into());
        }
        replayed.push((
            current,
            action,
            *outcome,
            facts.clone(),
            destination.clone(),
        ));
        current = destination.clone();
    }
    Ok(replayed)
}
fn concrete(action: &Action) -> umpire_tla::Concrete {
    let (name, fields) = umpire_dsl::ActionValues::<World>::action_value(action);
    umpire_tla::Concrete {
        action: name.into(),
        fields: fields
            .into_iter()
            .map(|(name, value)| {
                let literal = match value {
                    umpire_dsl::Value::Int(value) => umpire_tla::Literal::Int(value),
                    umpire_dsl::Value::Bool(value) => umpire_tla::Literal::Bool(value),
                    other => panic!("unexpected action payload: {other:?}"),
                };
                (name.into(), literal)
            })
            .collect(),
    }
}

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let root = std::path::PathBuf::from(std::env::args().nth(1).ok_or("repository root required")?);
    let source = std::fs::read_to_string(root.join("model/temporal/feature/activity/model.rs"))?;
    let world = World::new(3, None, None, None, None, true);
    let graph = umpire_testing::Checker::explore(&world, 4096)?;
    assert!(!graph.counts.cut);
    let tlc = umpire_tlc::Tlc::discover(&root)?;
    for (label, attempt, phase, pending, token, expected_outcome, expected_phase) in [
        (
            "current-first",
            1,
            Phase::Started,
            true,
            1,
            Outcome::Accepted,
            Phase::Canceled,
        ),
        (
            "current-cap",
            3,
            Phase::Started,
            true,
            3,
            Outcome::Accepted,
            Phase::Canceled,
        ),
        (
            "stale-scheduled",
            3,
            Phase::Scheduled,
            false,
            1,
            Outcome::Rejected(Rejection::NotFound),
            Phase::Scheduled,
        ),
        (
            "stale-pending",
            3,
            Phase::Started,
            true,
            1,
            Outcome::Rejected(Rejection::NotFound),
            Phase::Started,
        ),
    ] {
        let before = graph
            .states()
            .find(|(_, state)| {
                state.attempt == attempt
                    && state.phase == phase
                    && state.cancel_requested == pending
            })
            .map(|(_, state)| state.clone())
            .ok_or("unreachable acknowledgement premise")?;
        let action = Action::from(Worker::RespondCanceled(token));
        let native = world.checked_step(&before, action.clone())?;
        assert_eq!(native.len(), 1);
        assert_eq!(native[0].0, expected_outcome);
        assert_eq!(native[0].1, Vec::<ExpiredDeadline>::new());
        assert_eq!(native[0].2.phase, expected_phase);
        assert_eq!(native[0].2.attempt, attempt);
        if expected_outcome != Outcome::Accepted {
            assert_eq!(native[0].2, before);
        } else {
            assert!(!native[0].2.cancel_requested);
            assert_eq!(native[0].2.schedule_to_start_remaining, None);
            assert_eq!(native[0].2.start_to_close_remaining, None);
            assert_eq!(native[0].2.schedule_to_close_remaining, None);
            assert_eq!(native[0].2.heartbeat_remaining, None);
        }
        let method = format!(
            r"
        #[progress]
        fn false_ack_case(&self) -> LeadsTo {{
            path![self.attempt == {attempt} && self.phase == Phase::{phase:?} && self.cancel_requested == {pending}, ..=1, self.phase == Phase::Completed]
                .assuming(only([Worker::RespondCanceled({token})]))
        }}
"
        );
        let suffix = "\n    }\n}";
        assert_eq!(source.matches(suffix).count(), 1);
        let augmented = source.replacen(suffix, &format!("\n{method}\n    }}\n}}"), 1);
        let mut module =
            umpire_testing::bound(world.clone(), 4096, None).lift_source("model.rs", &augmented)?;
        let ordinal = world
            .actions()
            .iter()
            .position(|candidate| *candidate == action)
            .ok_or("acknowledgement absent from catalog")?;
        let claim = module
            .metadata
            .progress
            .iter()
            .find(|claim| claim.name == "false_ack_case")
            .ok_or("missing scoped witness")?;
        assert_eq!(claim.allowed_inputs, [ordinal]);
        assert_eq!(claim.fair_inputs, []);
        assert_eq!(module.metadata.catalog[ordinal], concrete(&action));
        module
            .metadata
            .progress
            .retain(|claim| claim.name == "false_ack_case");
        module.metadata.invariants.clear();
        module.metadata.postconditions.clear();
        module.metadata.negative.clear();
        module.cfg = module
            .cfg
            .lines()
            .filter(|line| !line.starts_with("INVARIANT") && !line.starts_with("PROPERTY"))
            .flat_map(|line| {
                [
                    line, "
",
                ]
            })
            .collect();
        let result = tlc.check_module(&module)?;
        assert_eq!(result.states.len(), graph.counts.reached);
        let property = &result.properties["Progress_false_ack_case"];
        assert_eq!(property.verdict, umpire_tlc::Verdict::Refuted);
        let trace = property
            .trace
            .as_ref()
            .ok_or("missing acknowledgement trace")?;
        println!(
            "ACK label={label} world={world:?} reachable_before={before:?} ordinal={ordinal} native={native:?} trace={trace:#?}"
        );
        let edges = replay(&world, &module, trace)?;
        let edge = edges
            .iter()
            .find(|(state, delivered, _, _, _)| state == &before && delivered == &action)
            .ok_or("missing exact acknowledgement edge")?;
        assert_eq!(edge.2, expected_outcome);
        assert_eq!(edge.3, Vec::<ExpiredDeadline>::new());
        assert_eq!(edge.4, native[0].2);
        println!("ACK_REPLAY_PASS:{label}:edges={}", edges.len());
    }
    Ok(())
}
