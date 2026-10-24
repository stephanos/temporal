use crate::optional_ticks::countdown;
use countdown::{Clock, Model, Remaining, Ticks, World};
use umpire::dsl::{__lifted::Lifts, Claim, Machine};
use umpire_checker::{Checker, Interpreted};

fn checked_world(timeout: Option<u8>, start: Option<u8>) -> Result<World, &'static str> {
    let timeout = timeout.map(|raw| Ticks::try_new(raw).ok_or("timeout")).transpose()?;
    let start = start.map(|raw| Remaining::try_new(raw).ok_or("remaining")).transpose()?;
    Ok(World::new(timeout, start, false, false, false))
}

fn selected(timeout: Option<u8>) -> World {
    checked_world(timeout, None).unwrap_or_else(|field| panic!("invalid fixture {field}"))
}

umpire_testing::fixture!(
    World = selected(Some(3)),
    32,
    Claim::new("never_started", |_, state: &Model| (!state.started).into())
);

fn folder() -> std::path::PathBuf {
    std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("fixtures/optional_ticks")
}

#[test]
fn optional_ticks_factory_and_native_validity_boundaries() -> umpire_testing::TestResult {
    for (timeout, start, field) in
        [(Some(0), None, "timeout"), (Some(4), None, "timeout"), (None, Some(4), "remaining")]
    {
        assert_eq!(checked_world(timeout, start).err(), Some(field));
    }
    assert!(checked_world(None, Some(0)).is_ok());
    for (world, field) in [
        (World::new(Some(Ticks(0)), None, false, false, false), "timeout"),
        (World::new(Some(Ticks(4)), None, false, false, false), "timeout"),
        (World::new(None, Some(Remaining { ticks: 4 }), false, false, false), "remaining"),
    ] {
        let refusal = bound_for(world)
            .lift(folder().join("model.rs"))
            .err()
            .ok_or("a forged binding was admitted")?;
        assert!(refusal.contains(field), "{refusal}");
    }
    for world in [
        World::new(None, Some(Remaining { ticks: 4 }), false, false, false),
        World::new(Some(Ticks(1)), None, false, false, true),
    ] {
        let report = Checker::explore(&world, 32)?.verify_all()?.to_string();
        assert!(report.contains("claim bounded: Refuted"), "{report}");
    }
    Ok(())
}

#[test]
fn optional_ticks_exact_countdown_and_interpreted_parity() -> umpire_testing::TestResult {
    for (timeout, ticks) in [(None, 0), (Some(1), 1), (Some(3), 3)] {
        let world = selected(timeout);
        let mut state = world.starts().remove(0);
        let started = world.checked_step(&state, Clock::Start.into())?;
        assert_eq!(started.len(), 1);
        assert!(started[0].1.is_empty());
        state = started[0].2.clone();
        for remaining in (0..ticks).rev() {
            let step = world.checked_step(&state, Clock::Tick.into())?;
            assert_eq!(step.len(), 1);
            assert!(step[0].1.is_empty());
            state = step[0].2.clone();
            assert_eq!(state.remaining, Some(Remaining { ticks: remaining }));
        }
        assert!(world.checked_step(&state, Clock::Tick.into())?.is_empty());
        assert_eq!(state.remaining, timeout.map(|_| Remaining { ticks: 0 }));
        let agreement = umpire_testing::interpreted(|| selected(timeout), None, 32, None)?;
        assert_eq!(
            (agreement.states, agreement.edges),
            (usize::from(ticks) + 2, usize::from(ticks) + 1)
        );
        let files = bound_for(selected(timeout)).run()?;
        assert_eq!(files["states.txt"].lines().count(), usize::from(ticks) + 2);
        assert_eq!(files["rows.txt"].lines().count(), usize::from(ticks) + 1);
        if timeout.is_some() {
            assert_eq!(
                files["paths/never_exhausted.txt"].matches("Clock::Tick").count(),
                usize::from(ticks)
            );
        } else {
            assert!(!files.contains_key("paths/never_exhausted.txt"));
        }
    }
    Ok(())
}

#[test]
fn optional_ticks_bindings_preserve_correlated_deduplicated_and_empty_starts()
-> umpire_testing::TestResult {
    for (empty, duplicates, expected) in [(false, false, 1), (false, true, 2), (true, false, 0)] {
        let world =
            World::new(Some(Ticks(3)), Some(Remaining { ticks: 2 }), empty, duplicates, false);
        let native = bound_for(world.clone()).run()?;
        assert_eq!(
            native["states.txt"].lines().count(),
            if empty {
                0
            } else if duplicates {
                6
            } else {
                5
            }
        );
        assert_eq!(native["rows.txt"].lines().count(), if empty { 0 } else { 4 });
        let agreement = umpire_testing::interpreted(|| world.clone(), None, 32, None)?;
        assert_eq!(agreement.states, native["states.txt"].lines().count());
        let module = bound_for(world).lift(folder().join("model.rs"))?;
        let binding = module.metadata.binding.as_ref().ok_or("missing binding")?;
        assert_eq!(binding.starts.len(), expected);
        if duplicates {
            assert_eq!(
                binding.starts[0][0].1,
                umpire_tla::Literal::Optional(Some(Box::new(umpire_tla::Literal::Bounded {
                    name: "Remaining".into(),
                    raw: 2
                })))
            );
            assert_eq!(binding.starts[1][0].1, umpire_tla::Literal::Optional(None));
        }
    }
    Ok(())
}

#[test]
fn optional_ticks_every_cfg_carries_the_same_selected_world() -> umpire_testing::TestResult {
    for (timeout, code) in [(None, 0), (Some(1), 2), (Some(3), 4)] {
        let module = bound_for(selected(timeout)).lift(folder().join("model.rs"))?;
        let metadata = &module.metadata;
        assert!(!metadata.invariants.is_empty());
        assert!(!metadata.postconditions.is_empty());
        assert!(!metadata.negative.is_empty());
        assert!(!metadata.progress.is_empty());
        for cfg in std::iter::once(&module.cfg)
            .chain(metadata.invariants.iter().map(|claim| &claim.cfg))
            .chain(metadata.postconditions.iter().map(|claim| &claim.cfg))
            .chain(metadata.negative.iter().map(|claim| &claim.cfg))
            .chain(metadata.progress.iter().map(|claim| &claim.cfg))
        {
            assert!(cfg.contains(&format!("CONSTANT __umpire_world_timeout = {code}\n")), "{cfg}");
        }
    }
    Ok(())
}

#[test]
fn optional_ticks_source_only_mutant_disagrees_with_compiled_model() -> umpire_testing::TestResult {
    let mut tree = World::__umpire_lifted().model.map_err(|refusal| refusal.message)?;
    let tick = tree
        .actions
        .iter_mut()
        .find(|action| action.name == "Clock::Tick")
        .ok_or("missing Clock::Tick")?;
    assert!(!tick.branches.is_empty());
    for branch in &mut tick.branches {
        branch.updates.remove("remaining");
    }
    let compiled = selected(Some(3));
    let mutant = Interpreted::from_tree(selected(Some(3)), &tree)?;
    let disagreement = umpire_testing::compare(&compiled, &mutant, None, 32, None)
        .err()
        .ok_or("tree-only countdown mutant agreed")?;
    assert!(
        disagreement.to_string().contains("interpreted exploration disagrees"),
        "{disagreement}"
    );
    Ok(())
}

#[test]
fn optional_ticks_selected_world_identity_rejects_another_legal_selection()
-> umpire_testing::TestResult {
    let module = bound_for(selected(Some(1))).lift(folder().join("model.rs"))?;
    let error = bound_for(selected(Some(3)))
        .check_module(&folder(), &module)
        .err()
        .ok_or("another selected World certified agreement")?;
    assert!(error.to_string().contains("selected World binding"), "{error}");
    Ok(())
}

#[test]
fn optional_ticks_required_tlc_replays_exact_countdown_witnesses() -> umpire_testing::TestResult {
    if std::env::var("UMPIRE_BACKENDS").as_deref() != Ok("require") {
        return Ok(());
    }
    for timeout in [None, Some(1), Some(3)] {
        let files = bound_for(selected(timeout)).check(&folder())?;
        let report = &files["tlc.txt"];
        assert!(report.contains("claim never_started: Refuted"), "{report}");
        assert!(
            report.contains(&format!(
                "claim never_exhausted: {}",
                if timeout.is_some() { "Refuted" } else { "Verified" }
            )),
            "{report}"
        );
        assert!(report.contains("claim decrements: Verified"), "{report}");
        assert!(report.contains("claim finishes: Verified"), "{report}");
        assert!(
            report.contains(&format!(
                "counterexamples replayed: {}",
                if timeout.is_some() { 2 } else { 1 }
            )),
            "{report}"
        );
    }
    Ok(())
}
