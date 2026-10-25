use crate::world_fairness::retry::{Model, World};
use umpire::dsl::{__lifted::Lifts, CATALOG_LIMIT, Claim, Machine};
use umpire_testing::replace_once;
use umpire_tla::observers::PENDING;

umpire_testing::fixture!(
    World = World::new(3),
    64,
    Claim::new("never_closed", |_, state: &Model| (!state.closed).into())
);

const SOURCE: &str = include_str!("model.rs");

fn folder() -> std::path::PathBuf {
    std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("fixtures/world_fairness")
}

#[test]
fn world_fairness_selected_worlds_agree_compiled_and_embedded() -> umpire_testing::TestResult {
    let readmitted = umpire_tla::Lift::parse("world_fairness/model.rs", SOURCE)?;
    let embedded = World::__umpire_lifted().model.map_err(|refusal| refusal.message)?;
    assert!(embedded.assumptions[0].conditional.is_some());
    assert!(readmitted.assumptions[0].conditional.is_some());
    for (maximum, states, edges) in [(1, 3, 4), (2, 6, 11), (3, 9, 20)] {
        let agreement = umpire_testing::interpreted(|| World::new(maximum), None, 64, None)?;
        assert_eq!((agreement.states, agreement.edges), (states, edges));
        let files = bound_for(World::new(maximum)).run()?;
        assert!(files["report.txt"].contains("claim closes: Verified"));
        assert!(files["negative_report.txt"].contains("claim never_closed: Refuted"));
    }
    Ok(())
}

fn readmitted_source() -> String {
    format!("{SOURCE}\n{}", include_str!("tests.rs"))
}

#[test]
fn world_fairness_exact_selected_values_ordinals_and_wf_obligations() -> umpire_testing::TestResult
{
    for maximum in 1..=3 {
        let world = World::new(maximum);
        let catalog = world.checked_catalog(CATALOG_LIMIT)?;
        let module = bound_for(world).lift(folder().join("model.rs"))?;
        let readmitted = bound_for(World::new(maximum))
            .lift_source("world_fairness/model.rs", &readmitted_source())?;
        assert_eq!(module.tla, readmitted.tla);
        assert_eq!(module.cfg, readmitted.cfg);
        let count = usize::from(maximum);
        assert_eq!(catalog.actions().len(), count + 3);
        let mut expected: Vec<_> = (1..=maximum)
            .map(|attempt| umpire_tla::Concrete {
                action: "Worker::Poll".into(),
                fields: vec![("0".into(), umpire_tla::Literal::Int(i128::from(attempt)))],
            })
            .collect();
        expected.extend(["Retry", "Close", "Heartbeat"].map(|action| umpire_tla::Concrete {
            action: format!("Worker::{action}"),
            fields: Vec::new(),
        }));
        assert_eq!(module.metadata.catalog, expected);
        let claim = &module.metadata.progress[0];
        assert_eq!(claim.name, "closes");
        assert_eq!(
            claim.allowed,
            ["Worker::Poll", "Worker::Retry", "Worker::Close", "Worker::Heartbeat"]
        );
        assert_eq!(claim.fair, ["Worker::Poll", "Worker::Close"]);
        assert_eq!(claim.allowed_inputs, (0..count + 3).collect::<Vec<_>>());
        assert_eq!(claim.fair_inputs, (0..count).chain([count + 1]).collect::<Vec<_>>());
        assert_eq!(claim.fair_inputs.len(), count + 1);
        assert_eq!(claim.bound, None);
        let spec = module
            .tla
            .lines()
            .find(|line| line.starts_with("ProgressSpec_closes =="))
            .ok_or("missing progress specification")?;
        assert_eq!(spec.matches("WF_vars(").count(), count + 1, "{spec}");
        for attempt in 1..=maximum {
            let operator = format!("Fair_closes_Worker_Poll_{attempt}");
            assert!(spec.contains(&format!("WF_vars({operator})")), "{spec}");
            assert!(module.tla.contains(&format!(
                "{operator} == ({PENDING} = 1) /\\ Worker_Poll_At({attempt}) /\\ Monitor_closes"
            )));
        }
        assert!(spec.contains("WF_vars(Fair_closes_Worker_Close)"), "{spec}");
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
            assert!(
                cfg.contains(&format!("CONSTANT __umpire_world_maximum = {maximum}\n")),
                "{cfg}"
            );
        }
    }
    Ok(())
}

#[test]
fn world_fairness_source_only_selectors_and_leaves_discriminate() -> umpire_testing::TestResult {
    let too_large = replace_once(SOURCE, "constants.maximum == 1", "constants.maximum == 2");
    let refusal = bound_for(World::new(1))
        .lift_source("world_fairness/model.rs", &too_large)
        .err()
        .ok_or("a selected larger-domain leaf was admitted")?;
    assert!(refusal.contains("catalog"), "{refusal}");
    for mutant in [
        replace_once(SOURCE, "constants.maximum == 2", "constants.maximum == 3"),
        replace_once(
            SOURCE,
            "fair([Worker::Poll(1), Worker::Poll(2), Worker::Poll(3), Worker::Close])",
            "fair([Worker::Poll(1), Worker::Poll(2), Worker::Close])",
        ),
    ] {
        let source = format!("{mutant}\n{}", include_str!("tests.rs"));
        let module = bound_for(World::new(3)).lift_source("world_fairness/model.rs", &source)?;
        let error = bound_for(World::new(3))
            .check_module(&folder(), &module)
            .err()
            .ok_or("source-only scope mutant certified agreement")?;
        assert!(error.to_string().contains("progress"), "{error}");
    }
    Ok(())
}

#[test]
fn world_fairness_metadata_only_mutations_refuse_before_tlc() -> umpire_testing::TestResult {
    let module = bound_for(World::new(3)).lift(folder().join("model.rs"))?;
    for field in ["fair_inputs", "name", "allowed", "fair", "allowed_inputs", "bound"] {
        let mut mutant = module.clone();
        let claim = &mut mutant.metadata.progress[0];
        match field {
            "name" => claim.name = "another".into(),
            "allowed" => {
                claim.allowed.pop();
            }
            "fair" => {
                claim.fair.pop();
            }
            "allowed_inputs" => {
                claim.allowed_inputs.pop();
            }
            "fair_inputs" => {
                claim.fair_inputs.pop();
            }
            _ => claim.bound = Some(1),
        }
        let error = bound_for(World::new(3))
            .check_module(&folder(), &mutant)
            .err()
            .ok_or_else(|| format!("{field} metadata mutation certified agreement"))?;
        assert!(error.to_string().contains("progress"), "{field}: {error}");
    }
    let mut typed_catalog = module.clone();
    typed_catalog.metadata.catalog[0].fields[0].1 = umpire_tla::Literal::Int(2);
    let error = bound_for(World::new(3))
        .check_module(&folder(), &typed_catalog)
        .err()
        .ok_or("typed catalog mutation certified agreement")?;
    assert!(error.to_string().contains("selected World binding"), "{error}");
    Ok(())
}

#[test]
fn world_fairness_legal_world_swaps_refuse_in_all_five_cfg_families() -> umpire_testing::TestResult
{
    let module = bound_for(World::new(3)).lift(folder().join("model.rs"))?;
    let error = bound_for(World::new(1))
        .check_module(&folder(), &module)
        .err()
        .ok_or("another legal selected World certified agreement")?;
    assert!(error.to_string().contains("selected World binding"), "{error}");
    if std::env::var("UMPIRE_BACKENDS").as_deref() != Ok("require") {
        return Ok(());
    }
    for family in ["main", "invariant", "postcondition", "negative", "progress", "all"] {
        let mut mutant = module.clone();
        let metadata = &mut mutant.metadata;
        let configs: Vec<_> = match family {
            "main" => vec![&mut mutant.cfg],
            "invariant" => vec![&mut metadata.invariants[0].cfg],
            "postcondition" => vec![&mut metadata.postconditions[0].cfg],
            "negative" => vec![&mut metadata.negative[0].cfg],
            "progress" => vec![&mut metadata.progress[0].cfg],
            _ => std::iter::once(&mut mutant.cfg)
                .chain(metadata.invariants.iter_mut().map(|claim| &mut claim.cfg))
                .chain(metadata.postconditions.iter_mut().map(|claim| &mut claim.cfg))
                .chain(metadata.negative.iter_mut().map(|claim| &mut claim.cfg))
                .chain(metadata.progress.iter_mut().map(|claim| &mut claim.cfg))
                .collect(),
        };
        for cfg in configs {
            *cfg = replace_once(
                cfg,
                "CONSTANT __umpire_world_maximum = 3\n",
                "CONSTANT __umpire_world_maximum = 1\n",
            );
        }
        let error = bound_for(World::new(3))
            .check_module(&folder(), &mutant)
            .err()
            .ok_or_else(|| format!("{family} CFG certified another legal World"))?;
        assert!(error.to_string().contains("selected World bindings differ"), "{family}: {error}");
    }
    Ok(())
}

#[test]
fn world_fairness_required_tlc_agrees_and_replays_at_each_maximum() -> umpire_testing::TestResult {
    if std::env::var("UMPIRE_BACKENDS").as_deref() != Ok("require") {
        return Ok(());
    }
    for maximum in 1..=3 {
        let module = bound_for(World::new(maximum))
            .lift_source("world_fairness/model.rs", &readmitted_source())?;
        let files = bound_for(World::new(maximum)).check_module(&folder(), &module)?;
        let report = &files["tlc.txt"];
        let states = 3 * maximum;
        assert!(
            report.contains(&format!("states: {states}; native: {states}; agreement: yes")),
            "{report}"
        );
        for claim in ["bounded", "monotonic", "closes"] {
            assert!(report.contains(&format!("claim {claim}: Verified")), "{report}");
        }
        assert!(report.contains("claim never_closed: Refuted"), "{report}");
        assert!(report.contains("counterexamples replayed: 1"), "{report}");
    }
    Ok(())
}

#[test]
#[allow(clippy::print_stderr)]
fn world_fairness_required_tlc_mutants_replay_heartbeat_and_starvation()
-> umpire_testing::TestResult {
    if std::env::var("UMPIRE_BACKENDS").as_deref() != Ok("require") {
        return Ok(());
    }
    let module = bound_for(World::new(3)).lift(folder().join("model.rs"))?;
    let poll = |attempt| format!(" /\\ WF_vars(Fair_closes_Worker_Poll_{attempt})");
    let all_polls = format!("{}{}{}", poll(1), poll(2), poll(3));
    let mut missing_close = module.clone();
    missing_close.tla = replace_once(&module.tla, " /\\ WF_vars(Fair_closes_Worker_Close)", "");
    let mut missing_current = module.clone();
    missing_current.tla = replace_once(&module.tla, &poll(3), "");
    let mut stale_payload = module.clone();
    stale_payload.tla = replace_once(
        &module.tla,
        format!("Fair_closes_Worker_Poll_3 == ({PENDING} = 1) /\\ Worker_Poll_At(3)").as_str(),
        format!("Fair_closes_Worker_Poll_3 == ({PENDING} = 1) /\\ Worker_Poll_At(1)").as_str(),
    );
    let mut variant_union = module.clone();
    variant_union.tla = replace_once(
        &module.tla,
        &all_polls,
        format!(" /\\ WF_vars(({PENDING} = 1) /\\ Worker_Poll /\\ Monitor_closes)").as_str(),
    );
    let root = std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("../..");
    let tlc = umpire_tlc::Tlc::discover(&root)?;
    for (name, mutant, witness) in [
        ("missing_close", missing_close, Some("Worker::Heartbeat")),
        ("missing_current", missing_current, Some("Worker::Poll")),
        ("stale_payload", stale_payload, None),
        ("variant_union", variant_union, None),
    ] {
        if let Some(action) = witness {
            let result = tlc.check_module(&mutant)?;
            let property = &result.properties["Progress_closes"];
            assert_eq!(property.verdict, umpire_tlc::Verdict::Refuted, "{name}");
            let trace = property.trace.as_ref().ok_or("missing lasso")?;
            assert_cycle(trace, action)?;
            eprintln!("world_fairness {name} selected maximum=3 trace: {trace:#?}");
        }
        let error = bound_for(World::new(3))
            .check_module(&folder(), &mutant)
            .err()
            .ok_or_else(|| format!("{name} fairness mutant agreed"))?;
        assert!(
            error.to_string().contains("verdict difference closes: TLC Refuted; native Verified"),
            "{name}: {error}"
        );
    }
    Ok(())
}

fn assert_cycle(trace: &umpire_tlc::Trace, action: &str) -> umpire_testing::TestResult {
    use umpire_tlc::Value;

    let cycle = &trace.states[trace.back_to.ok_or("missing cycle back edge")?..];
    let heartbeat = action == "Worker::Heartbeat";
    for state in cycle {
        assert_eq!(state.state["closed"], Value::Bool(false));
        assert_eq!(state.state["ready"], Value::Bool(heartbeat));
        if !heartbeat {
            assert_eq!(state.state["attempt"], Value::Int(3));
        }
    }
    let edge = cycle
        .iter()
        .find(|state| {
            state.observers.get(umpire_tla::observers::ACTION) == Some(&Value::Text(action.into()))
        })
        .ok_or("cycle lacks the expected concrete action")?;
    assert_eq!(edge.observers[umpire_tla::observers::CHOICE], Value::Int(0));
    assert_eq!(edge.observers[umpire_tla::observers::OUTCOME], Value::Seq(Vec::new()));
    assert_eq!(edge.observers[umpire_tla::observers::FACTS], Value::Seq(Vec::new()));
    if heartbeat {
        assert_eq!(edge.observers[umpire_tla::observers::INPUT], Value::Int(5));
    } else {
        assert!(matches!(edge.observers[umpire_tla::observers::INPUT], Value::Int(0 | 1)));
    }
    Ok(())
}
