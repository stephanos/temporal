use crate::countdowns::clock::{Clock, World};
use umpire::dsl::{ActionValues, CheckOutcome, Machine, Value};
use umpire_checker::{Checker, Interpreted};

impl ActionValues<Interpreted<World>> for crate::countdowns::clock::Action {
    fn action_value(&self) -> (&'static str, Vec<(&'static str, Value)>) {
        <Self as ActionValues<World>>::action_value(self)
    }

    fn domains(_: &Interpreted<World>) -> Vec<(&'static str, &'static str, Vec<Value>)> {
        Vec::new()
    }
}

fn selected(first: Option<u8>, second: Option<u8>) -> World {
    World::new(first.map(umpire::dsl::countdown::Ticks), second.map(umpire::dsl::countdown::Ticks))
}

umpire_testing::fixture!(World = selected(Some(3), Some(3)), 64);

fn folder() -> std::path::PathBuf {
    std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("fixtures/countdowns")
}

#[test]
fn countdowns_native_interpreter_and_priority() -> umpire_testing::TestResult {
    for (first, second, winner) in [
        (None, None, (false, false)),
        (Some(1), None, (true, false)),
        (None, Some(1), (false, true)),
        (Some(3), Some(1), (false, true)),
        (Some(1), Some(3), (true, false)),
        (Some(3), Some(3), (true, false)),
    ] {
        let world = selected(first, second);
        let graph = Checker::explore(&world, 64)?;
        assert!(!graph.counts.cut);
        assert_eq!(graph.verify_all()?.outcome(), CheckOutcome::Verified);
        umpire_testing::interpreted(|| selected(first, second), None, 64, None)?;
        let mut state = world.starts().remove(0);
        for _ in 0..3 {
            if let Some(step) = world.checked_step(&state, Clock::Tick.into())?.first() {
                state = step.2.clone();
            }
        }
        assert_eq!((state.first_won, state.second_won), winner);
        assert!(state.first.is_none() && state.second.is_none());
        if let Some(step) = world.checked_step(&state, Clock::Reset.into())?.first() {
            assert_eq!(step.2, world.starts()[0]);
        }
        if std::env::var("UMPIRE_BACKENDS").as_deref() == Ok("require") {
            let files = bound_for(selected(first, second)).check(&folder())?;
            for law in ["within", "priority", "descends"] {
                assert!(files["tlc.txt"].contains(&format!("claim {law}: Verified")));
            }
        }
    }
    Ok(())
}

#[test]
fn countdowns_source_mutants_refute_and_replay() -> umpire_testing::TestResult {
    let source = std::fs::read_to_string(folder().join("model.rs"))?;
    let tick = "self.probe = countdown::tick(self.probe);";
    for (name, from, to, first, property) in [
        ("no decrement", tick, "self.probe = self.probe;", Some(3), "Post_descends"),
        (
            "absent arming",
            tick,
            "self.probe = if self.probe.is_none() { Some(Remaining { ticks: 1 }) } else { countdown::tick(self.probe) };",
            None,
            "Post_descends",
        ),
        (
            "zero corruption",
            tick,
            "self.probe = if self.probe.is_some() && self.probe.unwrap_or(Remaining { ticks: 0 }).ticks == 0 { Some(Remaining { ticks: 1 }) } else { countdown::tick(self.probe) };",
            Some(1),
            "Post_descends",
        ),
        (
            "within boundary",
            "countdown::within(self.first, world.first);",
            "check!(self.first.is_none() || (world.first.is_some() && self.first.unwrap_or(Remaining { ticks: 0 }).ticks > 0 && self.first.unwrap_or(Remaining { ticks: 0 }).ticks < world.first.unwrap_or(Ticks(1)).0));",
            Some(3),
            "Invariant_within",
        ),
    ] {
        let mutant = umpire_testing::replace_once(&source, from, to);
        let (tree, _, _) = umpire_front::read(&syn::parse_file(&mutant)?, &[])
            .map_err(|error| format!("{error:?}"))?;
        let world = selected(first, None);
        let interpreted = Interpreted::from_tree(world.clone(), &tree)?;
        let disagreement = umpire_testing::compare(&world, &interpreted, None, 64, None)
            .err()
            .ok_or_else(|| format!("{name} agreed with compiled good Model"))?;
        assert!(disagreement.to_string().contains("interpreted exploration disagrees"));
        let graph = Checker::explore(&interpreted, 64)?;
        assert!(!graph.counts.cut);
        let report = graph.verify_all()?;
        assert!(
            report.claims.iter().any(|claim| {
                claim.name == property.trim_start_matches("Post_").trim_start_matches("Invariant_")
                    && claim.outcome == CheckOutcome::Refuted
            }),
            "{name}: {report}"
        );
        if std::env::var("UMPIRE_BACKENDS").as_deref() != Ok("require") {
            continue;
        }
        let module = bound_for(world.clone()).lift_source("model.rs", &mutant)?;
        let root = std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("../..");
        let result = umpire_tlc::Tlc::discover(&root)?.check_module(&module)?;
        let claim = &result.properties[property];
        assert_eq!(claim.verdict, umpire_tlc::Verdict::Refuted, "{name}");
        let trace = claim.trace.as_ref().ok_or("missing mutant witness")?;
        umpire_testing::replay_trace(&interpreted, &interpreted.starts(), &module, trace)?;
        if property == "Post_descends" {
            assert!(
                umpire_testing::replay_trace(&world, &world.starts(), &module, trace).is_err(),
                "{name}"
            );
        } else {
            let state = world.starts().remove(0);
            assert_eq!(
                world.invariants()[0].evaluate(&world, &state).verdict,
                umpire::dsl::Verdict::Holds
            );
            assert_eq!(
                interpreted.invariants()[0].evaluate(&interpreted, &state).verdict,
                umpire::dsl::Verdict::Fails
            );
        }
    }
    Ok(())
}
