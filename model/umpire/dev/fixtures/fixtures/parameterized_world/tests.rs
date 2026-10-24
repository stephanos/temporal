use crate::parameterized_world::counter;
use counter::World;
use umpire::dsl::Claim;

umpire_testing::fixture!(
    World = World::new(1, 3),
    64,
    Claim::new("never_three", |_, state: &counter::Model| (state.value < 3).into())
);

fn model() -> std::path::PathBuf {
    std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("fixtures/parameterized_world/model.rs")
}

#[test]
fn each_selected_world_checks_natively() -> umpire_testing::TestResult {
    for (world, states, edges, verdict) in [
        (World::new(1, 3), 5, 3, "Refuted"),
        (World::new(0, 1), 3, 1, "Verified"),
        (World::new(2, 1), 0, 0, "Verified"),
    ] {
        let files = bound_for(world).run()?;
        assert_eq!(files["states.txt"].lines().count(), states);
        assert_eq!(files["rows.txt"].lines().count(), edges);
        assert!(
            files["negative_report.txt"].contains(&format!("claim never_three: {verdict}")),
            "{}",
            files["negative_report.txt"]
        );
        if verdict == "Refuted" {
            assert!(files["paths/never_three.txt"].contains("User::Advance"));
        }
    }
    Ok(())
}

#[test]
fn each_source_binding_preserves_selection_and_empty_starts() -> umpire_testing::TestResult {
    let modules = [World::new(1, 3), World::new(0, 1), World::new(2, 1)]
        .map(|world| bound_for(world).lift(model()));
    let [first, second, empty] = modules;
    let (first, second, empty) = (first?, second?, empty?);
    assert_eq!(first.tla, lifted(std::path::Path::new("fixtures/parameterized_world/model.rs"))?.tla);
    assert_ne!(first.cfg, second.cfg);
    assert_ne!(first.tla, second.tla);
    assert!(empty.tla.contains("FALSE"));
    Ok(())
}

#[test]
#[allow(clippy::print_stderr)]
fn actual_tlc_replays_the_false_claim_in_the_same_selected_world() -> umpire_testing::TestResult {
    if std::env::var("UMPIRE_BACKENDS").as_deref() != Ok("require") {
        eprintln!("World replay control skipped: no TLC evidence (set UMPIRE_BACKENDS=require)");
        return Ok(());
    }
    let folder =
        std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("fixtures/parameterized_world");
    for (world, verdict, replayed) in [(World::new(1, 3), "Refuted", 1), (World::new(0, 1), "Verified", 0)]
    {
        let files = bound_for(world).check(&folder)?;
        let report = &files["tlc.txt"];
        assert!(report.contains(&format!("never_three: {verdict}")), "{report}");
        assert!(report.contains(&format!("counterexamples replayed: {replayed}")), "{report}");
    }
    Ok(())
}
