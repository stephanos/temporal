//! Manual authored-source controls independent of runtime transition execution.
use umpire_lift::{Lift, Tla};

#[test]
fn manual_choices_keep_each_authored_outcome_and_fact() -> Result<(), Box<dyn std::error::Error>> {
    let source = include_str!("../../checker/tests/fixtures/choices/model.rs");
    let lifted = Lift::parse("choices/model.rs", source)?;
    let module = Tla::emit(&lifted)?;
    assert_eq!(module.metadata.actions.len(), 4);
    assert_eq!(module.metadata.postconditions[0].name, "choice_facts");
    assert!(module.tla.contains("accepted"));
    assert!(module.tla.contains("same_target"));
    assert!(module.tla.contains("rejected"));
    let aliased = source
        .replace("Machine,", "Machine as ModelMachine,")
        .replace("impl Machine for", "impl ModelMachine for");
    assert_eq!(Tla::emit(&Lift::parse("alias.rs", &aliased)?)?.tla, module.tla);
    Ok(())
}

#[test]
fn manual_authored_edge_tables_and_mode_wrappers_lift() -> Result<(), Box<dyn std::error::Error>> {
    let path = std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("../checker/tests/fixtures/progress_self_loop/model.rs");
    let module = Tla::emit(&Lift::source(&path)?)?;
    assert_eq!(module.metadata.actions.len(), 4);
    assert_eq!(module.metadata.progress[0].name, "reaches_target");
    Ok(())
}

#[test]
fn manual_trait_kettle_lifts_starts_effects_and_claims() -> Result<(), Box<dyn std::error::Error>> {
    let source = include_str!("../../checker/tests/fixtures/sample_kettle_trait/kettle.rs");
    let lifted = Lift::parse("sample_kettle_trait/model.rs", source)?;
    let module = Tla::emit(&lifted)?;
    assert_eq!(module.metadata.fields.len(), 3);
    assert_eq!(module.metadata.postconditions.len(), 2);
    assert_eq!(module.metadata.progress[0].allowed.len(), 3);
    assert!(module.metadata.actions[2].internal);
    Ok(())
}

#[test]
fn optional_states_preserve_shared_starts_and_step_claims() -> Result<(), Box<dyn std::error::Error>>
{
    let source = include_str!("../../checker/tests/fixtures/edge_checks/model.rs");
    let lifted = Lift::parse("edge_checks/model.rs", source)?;
    let module = Tla::emit(&lifted)?;
    assert_eq!(module.metadata.postconditions.len(), 2);
    let source = include_str!("../../checker/tests/fixtures/vacuity/model.rs");
    let vacuity = Tla::emit(&Lift::parse("vacuity/model.rs", source)?)?;
    assert!(vacuity.tla.lines().any(|line| line.starts_with("Post_unused_claim ==")
        && line.contains("Action::Unused")
        && line.contains("(FALSE)")));
    Ok(())
}

#[test]
fn manual_source_refuses_untranslated_statements_and_state_calls() {
    let original = include_str!("../../checker/tests/fixtures/choices/model.rs");
    for source in [
        original.replace("impl Machine for Tiny", "impl evil::Machine for Tiny"),
        original.replace("use umpire_dsl::", "use evil::"),
        original.replace("Value(2)", "Value(true)"),
        original.replace("vec![Value(0)]", "panic!(\"not a pure start\"); vec![Value(0)]"),
        original.replace("Value(2)", "mystery(2)"),
        original.replace("impl Machine for Tiny", "#[cfg(any())] impl Machine for Tiny"),
        original.replace("pub struct Tiny;", "pub struct Tiny; const TOO_LARGE: i64 = 2147483648;"),
    ] {
        let Err(error) = Lift::parse("unsupported_manual.rs", &source) else {
            panic!("untranslated source was admitted");
        };
        assert_eq!(error.file, "unsupported_manual.rs");
        assert!(error.line > 1);
        assert!(error.to_string().contains("refused"));
    }
}

#[test]
fn conditional_pushes_keep_enabled_choice_order() -> Result<(), Box<dyn std::error::Error>> {
    let original = include_str!("../../checker/tests/fixtures/choices/model.rs");
    let alternative = original.replace(
        "        }\n        steps",
        "        } else { steps.push((true, vec![\"fallback\"], Value(0))); }\n        steps",
    );
    let module = Tla::emit(&Lift::parse("alternative.rs", &alternative)?)?;
    assert!(!module.tla.contains("__umpire_choice' = 3"));
    let overlapping = original.replace("        }\n        steps", "        }\n        if state.0 == 1 { steps.push((true, vec![], Value(0))); }\n        steps");
    assert!(Lift::parse("overlapping.rs", &overlapping).is_err());
    Ok(())
}

fn with_progress(builder: &str) -> String {
    include_str!("../../checker/tests/fixtures/choices/model.rs")
        .replace("{Claim, Machine,", "{Claim, Progress, Scope, Machine,")
        .replace("    fn invariants", &format!("    fn progress(&self) -> Vec<Progress<Self>> {{ vec![{builder}] }}\n    fn invariants"))
}

#[test]
fn manual_progress_new_keeps_the_native_fulfilled_default() -> Result<(), Box<dyn std::error::Error>>
{
    let module = Tla::emit(&Lift::parse(
        "default_progress.rs",
        &with_progress("Progress::<Self>::new(\"default\")"),
    )?)?;
    assert!(
        module
            .tla
            .lines()
            .any(|line| line.starts_with("Activate_default ==") && line.contains("~(TRUE)"))
    );
    Ok(())
}

#[test]
fn manual_progress_assuming_replaces_the_entire_prior_assumption()
-> Result<(), Box<dyn std::error::Error>> {
    let source = with_progress(
        "Progress::<Self>::new(\"replacement\").assuming(Scope::without(&[Action::Advance])).assuming(Scope::without(&[Action::Reset]))",
    );
    let module = Tla::emit(&Lift::parse("replacement.rs", &source)?)?;
    assert_eq!(
        module.metadata.progress[0].allowed,
        ["Action::Advance", "Action::Idle", "Action::Unused"]
    );
    Ok(())
}

#[test]
fn manual_overwritten_updates_keep_the_prior_checked_arithmetic()
-> Result<(), Box<dyn std::error::Error>> {
    let source = include_str!("../../checker/tests/fixtures/sample_kettle_trait/kettle.rs")
        .replace("next.temperature += 1;", "next.temperature += 1; next.temperature = 0;");
    let module = Tla::emit(&Lift::parse("overwritten.rs", &source)?)?;
    assert!(module.tla.contains("(temperature) + (1)"));
    Ok(())
}

#[test]
fn manual_start_map_refuses_arithmetic_with_an_unresolved_rust_integer_type() {
    let source = include_str!("../../checker/tests/fixtures/choices/model.rs")
        .replace("vec![Value(0)]", "(255..=255).map(|n| Value(n + 1 - 1)).collect()");
    let Err(error) = Lift::parse("start_inference.rs", &source) else {
        panic!("u8 map arithmetic was silently widened");
    };
    assert_eq!(error.file, "start_inference.rs");
    assert!(error.line > 1);
}

#[test]
fn manual_source_refuses_authored_clone_and_into_overrides() {
    let original = include_str!("../../checker/tests/fixtures/edge_checks/model.rs");
    for source in [
        original.replace(
            "pub struct Tiny;",
            "pub struct Tiny; impl Value { fn clone(&self) -> Self { Self(0) } }",
        ),
        original
            .replacen("#[derive(Clone, PartialEq, Eq, Hash)]", "#[derive(PartialEq, Eq, Hash)]", 1)
            .replace(
                "pub struct Tiny;",
                "pub struct Tiny; impl Clone for Value { fn clone(&self) -> Self { Self(0) } }",
            ),
        original
            .replace(
                "pub struct Tiny;",
                "pub struct Tiny; impl Action { fn into(self) -> Self { Self::Reset } }",
            )
            .replace("(state.0, action)", "(state.0, action.into())"),
    ] {
        let Err(error) = Lift::parse("overridden_intrinsic.rs", &source) else {
            panic!("authored clone/into was treated as identity");
        };
        assert_eq!(error.file, "overridden_intrinsic.rs");
        assert!(error.line > 1);
    }
}
