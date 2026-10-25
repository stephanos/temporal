use umpire_tla::observers::PENDING;
use crate::action_inputs::inputs;
use inputs::World;
use umpire::dsl::{CATALOG_LIMIT, Claim, Machine};

// The golden World polls up to two attempts and has no skip slots.
umpire_testing::fixture!(
    World = World::new(2, 0),
    64,
    Claim::new("never_retried", |_, state: &inputs::Model| (state.attempt == 1).into())
);

/// The golden World, and a second World with three attempts and two skip slots.
fn worlds() -> [World; 2] {
    [World::new(2, 0), World::new(3, 2)]
}

#[test]
fn two_selected_worlds_differ_in_domains_counts_transitions_and_verdicts()
-> umpire_testing::TestResult {
    for (world, (actions, states, edges, vacuity, current)) in
        worlds().into_iter().zip([(4, 6, 16, true, "Verified"), (7, 18, 84, false, "Refuted")])
    {
        assert_eq!(world.checked_catalog(CATALOG_LIMIT)?.actions().len(), actions);
        let files = bound_for(world).run()?;
        assert_eq!(files["states.txt"].lines().count(), states);
        assert_eq!(files["rows.txt"].lines().count(), edges, "{}", files["rows.txt"]);
        assert_eq!(files["rows.txt"].contains("Worker::Skip"), !vacuity);
        let report = &files["negative_report.txt"];
        assert_eq!(report.contains("vacuity: variant Worker::Skip has no actions"), vacuity);
        assert!(report.contains(&format!("claim current_poll_settles: {current}")), "{report}");
        for claim in
            ["never_retried", "stale_poll_settles", "excluded_poll_settles", "settles_within_four"]
        {
            assert!(report.contains(&format!("claim {claim}: Refuted")), "{report}");
        }
    }
    Ok(())
}

#[test]
fn each_source_binding_carries_its_selected_catalog_into_every_configuration()
-> umpire_testing::TestResult {
    let model =
        std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("fixtures/action_inputs/model.rs");
    let [first, second] = worlds().map(|world| bound_for(world).lift(&model));
    let (first, second) = (first?, second?);
    assert_eq!(first.tla, lifted(std::path::Path::new("fixtures/action_inputs/model.rs"))?.tla);
    for (module, catalog, skip, constants) in [
        (
            &first,
            4,
            "Worker_Skip_Inputs == <<>>\n",
            "CONSTANT __umpire_world_attempts = 2\nCONSTANT __umpire_world_spares = 0\n",
        ),
        (
            &second,
            7,
            "Worker_Skip_Inputs == <<<<5, 0>>, <<6, 1>>>>\n",
            "CONSTANT __umpire_world_attempts = 3\nCONSTANT __umpire_world_spares = 2\n",
        ),
    ] {
        assert_eq!(module.metadata.catalog.len(), catalog);
        assert!(module.tla.contains(skip), "{}", module.tla);
        let metadata = &module.metadata;
        for cfg in std::iter::once(&module.cfg)
            .chain(metadata.invariants.iter().map(|claim| &claim.cfg))
            .chain(metadata.postconditions.iter().map(|claim| &claim.cfg))
            .chain(metadata.negative.iter().map(|claim| &claim.cfg))
            .chain(metadata.progress.iter().map(|claim| &claim.cfg))
        {
            assert!(cfg.contains(constants), "{cfg}");
        }
    }
    assert_eq!(
        first.metadata.catalog[0],
        umpire_tla::Concrete {
            action: "Worker::Poll".into(),
            fields: vec![("0".into(), umpire_tla::Literal::Int(1))],
        }
    );
    assert_eq!(
        first.metadata.catalog[2],
        umpire_tla::Concrete {
            action: "Worker::Fail".into(),
            fields: vec![("retryable".into(), umpire_tla::Literal::Bool(false))],
        }
    );
    let fair = |module: &umpire_tla::Module, claim: &str| {
        module.metadata.progress.iter().find(|progress| progress.name == claim).map(|progress| {
            (progress.allowed_inputs.clone(), progress.fair_inputs.clone())
        })
    };
    assert_eq!(fair(&first, "excluded_poll_settles"), Some((vec![1, 2, 3], vec![1])));
    assert_eq!(fair(&second, "excluded_poll_settles"), Some((vec![1, 2, 3, 4, 5, 6], vec![1])));
    Ok(())
}

/// Weak fairness is machine-closed here and bounded steps saturate at bound + 1, so fairness
/// cannot change a bounded verdict and no TLC run tells a fair bounded specification apart.
/// The rule that a bound receives no fairness benefit is pinned structurally instead: no
/// bounded claim's specification carries a fairness obligation, though its assumption has one.
#[test]
fn bounded_claims_carry_no_fairness_obligation() -> umpire_testing::TestResult {
    let model =
        std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("fixtures/action_inputs/model.rs");
    for world in worlds() {
        let module = bound_for(world).lift(&model)?;
        let bounded: Vec<_> =
            module.metadata.progress.iter().filter(|claim| claim.bound.is_some()).collect();
        assert!(!bounded.is_empty());
        for claim in bounded {
            assert!(!claim.fair_inputs.is_empty(), "{} assumes no fairness", claim.name);
            let header = format!("ProgressSpec_{} ==", claim.name);
            let spec = module
                .tla
                .lines()
                .find(|line| line.starts_with(&header))
                .ok_or_else(|| format!("missing {header}"))?;
            assert!(!spec.contains("WF_vars"), "{spec}");
        }
    }
    Ok(())
}

#[test]
#[allow(clippy::print_stderr)]
fn actual_tlc_agrees_and_replays_in_both_selected_worlds() -> umpire_testing::TestResult {
    if std::env::var("UMPIRE_BACKENDS").as_deref() != Ok("require") {
        eprintln!("input World control skipped: no TLC evidence (set UMPIRE_BACKENDS=require)");
        return Ok(());
    }
    let folder = std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("fixtures/action_inputs");
    for (world, (states, verdict, replayed)) in
        worlds().into_iter().zip([(6, "Verified", 4), (18, "Refuted", 5)])
    {
        let files = bound_for(world).check(&folder)?;
        let report = &files["tlc.txt"];
        assert!(report.contains(&format!("states: {states}; native: {states}; agreement: yes")));
        assert!(report.contains(&format!("claim current_poll_settles: {verdict}")), "{report}");
        assert!(report.contains(&format!("counterexamples replayed: {replayed}")), "{report}");
    }
    Ok(())
}

/// One progress claim of the emitted module, checked alone after replacing `from` by `to`.
fn mutated(
    module: &umpire_tla::Module,
    claim: &str,
    edits: &[(&str, &str)],
) -> Result<umpire_tlc::Verdict, Box<dyn std::error::Error>> {
    let mut reduced = module.clone();
    for (from, to) in edits {
        if !reduced.tla.contains(from) {
            return Err(format!("mutation target missing: {from}").into());
        }
        reduced.tla = reduced.tla.replace(from, to);
    }
    reduced.metadata.progress.retain(|progress| progress.name == claim);
    reduced.metadata.invariants.clear();
    reduced.metadata.postconditions.clear();
    reduced.metadata.negative.clear();
    reduced.cfg = module
        .cfg
        .lines()
        .filter(|line| !line.starts_with("INVARIANT") && !line.starts_with("PROPERTY"))
        .flat_map(|line| [line, "\n"])
        .collect();
    let root = std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..");
    let result = umpire_tlc::Tlc::discover(&root)?.check_module(&reduced)?;
    Ok(result.properties[&format!("Progress_{claim}")].verdict)
}

#[test]
fn exact_payload_fairness_mutations_disagree_with_the_native_verdicts()
-> umpire_testing::TestResult {
    use umpire_tlc::Verdict::{Refuted, Verified};
    if std::env::var("UMPIRE_BACKENDS").as_deref() != Ok("require") {
        return Ok(());
    }
    let module = lifted(std::path::Path::new("fixtures/action_inputs/model.rs"))?;
    let fair = format!("Fair_current_poll_settles_Worker_Poll_2 == ({PENDING} = 1) /\\ Worker_Poll_At(2)");
    let readmitted = " /\\ WF_vars(Fair_excluded_poll_settles_Worker_Poll_2)";
    for (claim, edits, native, mutant) in [
        // One obligation over the existential union is discharged by stale self-loops.
        (
            "current_poll_settles",
            vec![
                (
                    " /\\ WF_vars(Fair_current_poll_settles_Worker_Poll_1) /\\ WF_vars(Fair_current_poll_settles_Worker_Poll_2)",
                    format!(" /\\ WF_vars(({PENDING} = 1) /\\ Worker_Poll /\\ Monitor_current_poll_settles)").as_str(),
                ),
            ],
            Verified,
            Refuted,
        ),
        // Only the emitted fair payload changes: the second attempt's Poll is owed nothing.
        (
            "current_poll_settles",
            vec![(fair.as_str(), format!("Fair_current_poll_settles_Worker_Poll_2 == ({PENDING} = 1) /\\ Worker_Poll_At(1)").as_str())],
            Verified,
            Refuted,
        ),
        // Fairness alone readmitting the excluded first attempt, with its scope unchanged.
        (
            "excluded_poll_settles",
            vec![(
                readmitted,
                format!(" /\\ WF_vars(Fair_excluded_poll_settles_Worker_Poll_2) /\\ WF_vars(({PENDING} = 1) /\\ Worker_Poll_At(1) /\\ Monitor_excluded_poll_settles)").as_str(),
            )],
            Refuted,
            Verified,
        ),
        // Fair membership readmitting the excluded first attempt.
        (
            "excluded_poll_settles",
            vec![
                (
                    "Assumption_first_excluded == (\\E __umpire_i \\in {2} : Worker_Poll_At(__umpire_i))",
                    "Assumption_first_excluded == (Worker_Poll)",
                ),
                (
                    readmitted,
                    format!(" /\\ WF_vars(Fair_excluded_poll_settles_Worker_Poll_2) /\\ WF_vars(({PENDING} = 1) /\\ Worker_Poll_At(1) /\\ Monitor_excluded_poll_settles)").as_str(),
                ),
            ],
            Refuted,
            Verified,
        ),
    ] {
        assert_eq!(mutated(&module, claim, &[])?, native, "{claim}");
        assert_eq!(mutated(&module, claim, &edits)?, mutant, "{claim} mutant");
    }
    Ok(())
}
