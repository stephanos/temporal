//! Compiled manual behavior must be refused when its semantics are unsupported.
use umpire_dsl::{Machine, StepError};
use umpire_lift::Lift;

#[path = "fixtures/manual_admission_checked.rs"]
mod checked;
#[path = "fixtures/manual_admission_cfg.rs"]
mod conditional;
#[path = "fixtures/manual_admission_guard.rs"]
mod guarded;
#[path = "fixtures/manual_admission_members.rs"]
mod members;

fn located_refusal(
    file: &str,
    source: &str,
    marker: &str,
    message: &str,
) -> Result<(), Box<dyn std::error::Error>> {
    let expected_line =
        source.lines().position(|line| line.contains(marker)).ok_or("missing refusal marker")? + 1;
    let error =
        Lift::parse(file, source).err().ok_or("unsupported native behavior was admitted")?;
    assert_eq!(error.file, file);
    assert_eq!(error.line, expected_line, "{error}");
    assert!(error.column > 0);
    assert!(error.message.contains(message), "{error}");
    Ok(())
}

#[test]
fn conditional_associated_types_and_parameters_refuse_before_lifting()
-> Result<(), Box<dyn std::error::Error>> {
    let world = members::World;
    assert_eq!(world.step(&world.starts()[0], members::Action::Go).len(), 1);
    let original = include_str!("fixtures/manual_admission_members.rs");
    let attributes = [
        "cfg(any())",
        "cfg_attr(all(), cfg(any()))",
        "r#cfg(any())",
        "r#cfg_attr(all(), cfg(any()))",
    ];
    for selected in attributes {
        let mut source = original.to_string();
        for removed in attributes {
            if selected != removed {
                source = source.replace(&format!("    #[{removed}]\n    type Outcome = ();\n"), "");
            }
        }
        located_refusal(
            "conditional_type.rs",
            &source,
            &format!("#[{selected}]"),
            "conditional compilation",
        )?;
    }
    let mut signature = original.to_string();
    for attribute in attributes {
        signature = signature.replace(&format!("    #[{attribute}]\n    type Outcome = ();\n"), "");
    }
    located_refusal(
        "conditional_signature.rs",
        &signature,
        "#[cfg(any())]",
        "conditional compilation",
    )
}

#[test]
fn checked_step_override_refuses_instead_of_lifting_the_unchecked_graph()
-> Result<(), Box<dyn std::error::Error>> {
    let world = checked::World;
    let starts = world.starts();
    assert_eq!(world.step(&starts[0], checked::Action::Go).len(), 1);
    assert!(matches!(
        world.checked_step(&starts[0], checked::Action::Go),
        Err(StepError::RejectedAfterMutation)
    ));
    let source = include_str!("fixtures/manual_admission_checked.rs")
        .replace("fn r#checked_step", "fn checked_step");
    located_refusal("checked_override.rs", &source, "fn checked_step", "checked_step")
}

#[test]
fn nested_conditional_compilation_refuses_at_the_consumed_attribute()
-> Result<(), Box<dyn std::error::Error>> {
    let world = conditional::World(conditional::Mode::Chosen);
    let starts = world.starts();
    assert_eq!(starts[0].n, 0);
    assert_eq!(conditional::World(conditional::Mode::Other).starts()[0].n, 2);
    let steps = world.step(&starts[0], conditional::Action::Go);
    assert_eq!(steps.len(), 1);
    assert_eq!(steps[0].2.n, 3);
    let source = include_str!("fixtures/manual_admission_cfg.rs").replace("#[r#cfg", "#[cfg");
    let error = Lift::parse("conditional_arm.rs", &source)
        .err()
        .ok_or("conditional source was admitted")?;
    assert_eq!(format!("{error}\n"), include_str!("expected/manual_admission.txt"));
    located_refusal("conditional_arm.rs", &source, "#[cfg(any())]", "conditional compilation")?;
    let statement = source.replacen("#[cfg(any())]", "", 1);
    located_refusal("conditional_attr.rs", &statement, "#[cfg_attr", "conditional compilation")?;
    let expression = statement.replace("#[cfg_attr(all(), cfg(any()))]", "");
    located_refusal(
        "conditional_statement.rs",
        &expression,
        "#[cfg(any())]",
        "conditional compilation",
    )
}

#[test]
fn raw_conditional_attribute_spellings_are_not_silently_ignored()
-> Result<(), Box<dyn std::error::Error>> {
    let source = include_str!("fixtures/manual_admission_cfg.rs");
    located_refusal("raw_conditional.rs", source, "#[r#cfg(any())]", "conditional compilation")?;
    let source = source.replacen("#[r#cfg(any())]", "", 1);
    located_refusal("raw_conditional_attr.rs", &source, "#[r#cfg_attr", "conditional compilation")
}

#[test]
fn raw_machine_method_spellings_cannot_hide_overrides_or_claims()
-> Result<(), Box<dyn std::error::Error>> {
    located_refusal(
        "raw_checked.rs",
        include_str!("fixtures/manual_admission_checked.rs"),
        "fn r#checked_step",
        "raw-identifier",
    )?;
    let source = include_str!("../../tests/fixtures/choices/model.rs")
        .replace("fn invariants", "fn r#invariants");
    located_refusal("raw_invariants.rs", &source, "fn r#invariants", "raw-identifier")
}

#[test]
fn guarded_world_match_refuses_instead_of_selecting_a_false_arm()
-> Result<(), Box<dyn std::error::Error>> {
    assert_eq!(guarded::World(guarded::Mode::Chosen).starts()[0].n, 0);
    assert_eq!(guarded::World(guarded::Mode::Other).starts()[0].n, 2);
    located_refusal(
        "guarded_world.rs",
        include_str!("fixtures/manual_admission_guard.rs"),
        "Mode::Chosen if false",
        "guarded world",
    )
}
