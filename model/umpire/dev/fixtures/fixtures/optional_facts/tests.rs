use crate::optional_facts::facts;
use facts::{Fact, Model, Ordinary, Signal, World};
use umpire::dsl::{Machine, Outcome, Rejection};

umpire_testing::fixture!(World = World::new(false), 64);

fn folder() -> std::path::PathBuf {
    std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("fixtures/optional_facts")
}

#[test]
fn native_facts_preserve_enabledness_outcome_and_guard_priority() -> umpire_testing::TestResult {
    for selector in [false, true] {
        let world = World::new(selector);
        let start = Model { value: 0 };
        for (signal, value, fact) in [
            (Signal::A, 1, "A"),
            (Signal::B, 2, "B"),
            (Signal::C, 3, "C"),
            (Signal::D, 4, "D"),
            (Signal::Overlap, 1, if selector { "B" } else { "A" }),
        ] {
            let steps = world.checked_step(&start, signal.into())?;
            assert_eq!(steps.len(), 1);
            assert_eq!(steps[0].2.value, value);
            assert_eq!(format!("{:?}", steps[0].1), format!("[{fact}]"));
            let clear = world.checked_step(&steps[0].2, Ordinary::Clear.into())?;
            assert_eq!(clear.len(), 1);
            assert_eq!(clear[0].2.value, 5);
            assert!(clear[0].1.is_empty());
        }
        let changed = world.checked_step(&start, Signal::ChangedNone.into())?;
        assert_eq!(changed.len(), 1);
        assert_eq!(changed[0].2.value, 6);
        assert!(changed[0].1.is_empty());
        for signal in [Signal::UnchangedNone, Signal::UnchangedSome] {
            assert!(world.checked_step(&start, signal.into())?.is_empty());
        }
        let staying = world.checked_step(&start, Signal::Stay.into())?;
        assert_eq!(staying.len(), 1);
        assert_eq!(staying[0].2.value, 0);
        assert!(matches!(staying[0].1.as_slice(), [Fact::C]));
        let rejected = world.checked_step(&start, Signal::HelperReject.into())?;
        assert_eq!(rejected.len(), 1);
        assert_eq!(rejected[0].0, Outcome::Rejected(Rejection::Unavailable));
        assert!(matches!(rejected[0].1.as_slice(), [Fact::B]));
        let ordinary = world.checked_step(&start, Ordinary::Reject.into())?;
        assert_eq!(ordinary.len(), 1);
        assert_eq!(ordinary[0].0, Outcome::Rejected(Rejection::NotFound));
        assert!(ordinary[0].1.is_empty());
    }
    Ok(())
}

#[test]
fn selected_world_changes_only_the_authored_overlap_fact() -> umpire_testing::TestResult {
    let first = bound_for(World::new(false)).lift(folder().join("model.rs"))?;
    let second = bound_for(World::new(true)).lift(folder().join("model.rs"))?;
    assert_eq!(first.tla, second.tla);
    assert_ne!(first.cfg, second.cfg);
    assert!(first.cfg.contains("CONSTANT __umpire_world_selector = FALSE"));
    assert!(second.cfg.contains("CONSTANT __umpire_world_selector = TRUE"));
    assert_eq!(
        first.metadata.fields.iter().map(|field| (&field.name, &field.ty)).collect::<Vec<_>>(),
        second.metadata.fields.iter().map(|field| (&field.name, &field.ty)).collect::<Vec<_>>()
    );
    Ok(())
}

#[test]
fn required_tlc_replays_four_distinct_fact_witnesses_and_clearing() -> umpire_testing::TestResult {
    if std::env::var("UMPIRE_BACKENDS").as_deref() != Ok("require") {
        return Ok(());
    }
    for selector in [false, true] {
        let files = bound_for(World::new(selector)).check(&folder())?;
        let report = &files["tlc.txt"];
        for claim in ["never_a", "never_b", "never_c", "never_d", "never_clear"] {
            assert!(report.contains(&format!("claim {claim}: Refuted")), "{report}");
        }
        assert!(report.contains("counterexamples replayed: 5"), "{report}");
    }
    Ok(())
}

#[test]
fn source_only_fact_substitution_fails_native_replay() -> umpire_testing::TestResult {
    let source = std::fs::read_to_string(folder().join("model.rs"))?;
    let needle = "self.value = 3;\n                    Some(Fact::C)";
    assert_eq!(source.matches(needle).count(), 1);
    let mutant = source.replace(needle, "self.value = 3;\n                    Some(Fact::D)");
    let original = bound().lift_source("model.rs", &source)?;
    let module = bound().lift_source("model.rs", &mutant)?;
    assert_ne!(original.tla, module.tla);
    if std::env::var("UMPIRE_BACKENDS").as_deref() == Ok("require") {
        let error = bound()
            .check_module(&folder(), &module)
            .err()
            .ok_or("fact-only source mutant agreed")?;
        assert!(error.to_string().contains("outcome/facts differ"), "{error}");
    }
    Ok(())
}
