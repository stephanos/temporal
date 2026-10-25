//! Optional handler results are independently emitted from authored source.
#![forbid(unsafe_code)]

use umpire_front::expr::{ExprKind, Op, Unary};
use umpire_tla::observers::{ERROR, FACTS};
use umpire_tla::{Lift, Tla, Type};

const SOURCE: &str = r"
    use umpire_dsl::machine;
    #[machine] mod optional_facts {
        use umpire_dsl::{State, Action};
        #[derive(Debug)] enum Fact { A, B }
        #[derive(State)] struct Model { value: bool }
        #[derive(Action)] enum User { Set, Clear }
        impl Model {
            #[on(User)] fn user(&mut self, action: User) -> Option<Fact> {
                match action {
                    User::Set => { self.value = true; Some(Fact::A) },
                    User::Clear => { self.value = false; None },
                }
            }
        }
    }
";

type Result = std::result::Result<(), Box<dyn std::error::Error>>;

#[test]
fn source_only_constructor_swap_changes_both_emitters() -> Result {
    let original = Lift::parse("optional_facts.rs", SOURCE)?;
    let changed = SOURCE.replace("Some(Fact::A)", "Some(Fact::B)");
    assert_ne!(changed, SOURCE);
    let mutated = Lift::parse("optional_facts_mutated.rs", &changed)?;
    let fact =
        Type::Enum { name: "Fact".into(), variants: vec!["Fact::A".into(), "Fact::B".into()] };
    assert_eq!(original.fact_type, fact);
    for action in &original.actions {
        for branch in &action.branches {
            assert_eq!(branch.facts.ty, Type::Seq(Box::new(fact.clone())));
        }
    }
    for emit in [Tla::emit, Tla::safety] {
        let before = emit(&original)?;
        let after = emit(&mutated)?;
        assert_eq!(before.metadata.observer_types[FACTS], Type::Seq(Box::new(fact.clone())));
        assert!(before.tla.contains("<<\"Fact::A\">>"), "{}", before.tla);
        assert!(!before.tla.contains("<<\"Fact::B\">>"));
        assert!(after.tla.contains("<<\"Fact::B\">>"));
        assert!(!after.tla.contains("<<\"Fact::A\">>"));
        assert_ne!(before.tla, after.tla);
        assert!(before.tla.contains(&format!("{FACTS} = <<>>")));
    }
    Ok(())
}

#[test]
fn prefix_effects_and_locals_feed_the_tail_in_ssa_order() -> Result {
    let source = SOURCE.replace(
        "self.value = true; Some(Fact::A)",
        "let before = self.value; self.set(); if self.value != before { Some(Fact::A) } else { Some(Fact::B) }",
    ).replace("\n        }\n    }", "\n            #[effect] fn set(&mut self) { self.value = true; }\n        }\n    }");
    let lifted = Lift::parse("optional_facts_ssa.rs", &source)?;
    for emit in [Tla::emit, Tla::safety] {
        let module = emit(&lifted)?;
        assert!(module.tla.contains("<<\"Fact::A\">>"));
        assert!(module.tla.contains("<<\"Fact::B\">>"));
        assert!(module.tla.contains("(TRUE) # (value)"), "{}", module.tla);
        assert!(!module.tla.contains("(value) # (value)"));
        assert!(!module.tla.contains("__umpire_l0"));
    }
    Ok(())
}

#[test]
fn overlapping_action_guards_keep_the_first_matching_fact() -> Result {
    let source = SOURCE.replace(
        "User::Set => { self.value = true; Some(Fact::A) },",
        "User::Set if self.value => { self.value = false; Some(Fact::A) }, User::Set if self.value => { self.value = false; Some(Fact::B) }, User::Set => None,",
    );
    let lifted = Lift::parse("optional_facts_priority.rs", &source)?;
    let branches = &lifted.actions[0].branches;
    assert_eq!(branches.len(), 3);
    let ExprKind::Binary(Op::And, previous, current) = &branches[1].guard.kind else {
        return Err("second arm lost first-arm exclusion".into());
    };
    assert!(matches!(&previous.kind, ExprKind::Unary(Unary::Not, first) if first == current));
    for emit in [Tla::emit, Tla::safety] {
        let module = emit(&lifted)?;
        assert!(module.tla.contains("<<\"Fact::A\">>"));
        assert!(module.tla.contains("<<\"Fact::B\">>"));
    }
    Ok(())
}

#[test]
fn unsupported_tail_results_refuse_at_the_source() -> Result {
    for tail in [
        "return Some(Fact::A)",
        "Some(Fact::A);",
        "if self.value { Some(Fact::A) }",
        "let saved = Some(Fact::A); saved",
        "let Some = true; Some(Fact::A)",
        "let None = true; None",
        "Some(Fact::Missing)",
        "::Option::Some(Fact::A)",
        "::Option::None",
        "Some(Other::A)",
        "self.result()",
    ] {
        let source = SOURCE.replace("Some(Fact::A)", tail);
        let error = Lift::parse("optional_facts_refused.rs", &source)
            .err()
            .ok_or_else(|| format!("admitted unsupported tail: {tail}"))?;
        assert_eq!(error.file, "optional_facts_refused.rs");
        assert!(error.line > 1 && error.column > 0, "{error}");
    }
    Ok(())
}

#[test]
fn fact_tail_world_and_input_references_receive_emitted_names() -> Result {
    use umpire_tla::{Binding, Concrete, Literal};
    let source = r"
        use umpire_dsl::machine;
        #[machine] mod optional_facts_names {
            use umpire_dsl::{State, Action, World};
            #[derive(Debug)] enum Fact { A, B }
            #[derive(World)] struct World { flag: bool }
            #[derive(State)] struct Model { value: bool }
            #[derive(Action)] enum User { Set(bool) }
            impl Model {
                #[on(User)] fn user(&mut self, action: User, world: &World) -> Option<Fact> {
                    match action { User::Set(flag) => {
                        self.value = true;
                        if flag == world.flag { Some(Fact::A) } else { Some(Fact::B) }
                    } }
                }
            }
        }
    ";
    let lifted = Lift::parse("optional_facts_names.rs", source)?.bind(&Binding {
        world: vec![("flag".into(), Literal::Bool(true))],
        starts: vec![vec![("value".into(), Literal::Bool(false))]],
        actions: [false, true]
            .into_iter()
            .map(|flag| Concrete {
                action: "User::Set".into(),
                fields: vec![("0".into(), Literal::Bool(flag))],
            })
            .collect(),
    })?;
    for emit in [Tla::emit, Tla::safety] {
        let module = emit(&lifted)?;
        assert!(
            module.tla.contains("(__umpire_input_0) = (__umpire_world_flag)"),
            "{}",
            module.tla
        );
        assert!(!module.tla.contains("world.flag"));
        assert!(!module.tla.contains("__umpire_l0"));
    }
    Ok(())
}

#[test]
fn disabled_handler_tail_arithmetic_is_checked_by_both_emitters() -> Result {
    let source = SOURCE
        .replace("value: bool", "value: u8")
        .replace(
            "self.value = true; Some(Fact::A)",
            "if 1 / self.value == 0 { Some(Fact::A) } else { None }",
        )
        .replace("self.value = false", "self.value = 0");
    let lifted = Lift::parse("optional_facts_arithmetic.rs", &source)?;
    let main = Tla::emit(&lifted)?;
    let safety = Tla::safety(&lifted)?;
    assert!(main.tla.contains("__umpire_Div((1), (value))"), "{}", main.tla);
    assert!(safety.tla.contains("value # 0"), "{}", safety.tla);
    assert!(safety.tla.contains(&format!("{ERROR}' = (~(")));
    assert!(safety.tla.contains(&format!("{FACTS}' = IF ")));
    Ok(())
}

#[test]
fn mixed_unit_handler_clears_the_previous_fact_on_every_real_edge() -> Result {
    let source = SOURCE.replace(
        "#[derive(Action)] enum User",
        "#[derive(Action)] enum Ordinary { Reset } #[derive(Action)] enum User",
    ).replace(
        "\n        }\n    }",
        "\n            #[on(Ordinary)] fn ordinary(&mut self, action: Ordinary) { match action { Ordinary::Reset => { self.value = false; } } }\n        }\n    }",
    );
    let lifted = Lift::parse("optional_facts_clear.rs", &source)?;
    let ordinary = lifted
        .actions
        .iter()
        .find(|action| action.name == "Ordinary::Reset")
        .ok_or("missing ordinary action")?;
    assert!(
        matches!(&ordinary.branches[0].facts.kind, ExprKind::Sequence(values) if values.is_empty())
    );
    let main = Tla::emit(&lifted)?;
    let relation = main
        .tla
        .lines()
        .find(|line| line.starts_with("Ordinary_Reset =="))
        .ok_or("missing ordinary relation")?;
    assert!(relation.contains(&format!("{FACTS}' = <<>>")), "{relation}");
    let safety = Tla::safety(&lifted)?;
    let relation = safety
        .tla
        .split("Ordinary_Reset ==")
        .nth(1)
        .ok_or("missing safety ordinary relation")?
        .split("User_Set ==")
        .next()
        .ok_or("missing safety relation end")?;
    assert!(relation.contains(&format!("{FACTS}' = IF ")), "{relation}");
    assert!(relation.contains("ELSE <<>>"));
    Ok(())
}

#[test]
fn legacy_unit_fact_metadata_remains_exact() -> Result {
    let source = include_str!("../../../../dev/fixtures/fixtures/sample_kettle/model.rs");
    let lifted = Lift::parse("kettle.rs", source)?;
    assert_eq!(lifted.fact_type, Type::Unit);
    for action in &lifted.actions {
        for branch in &action.branches {
            assert_eq!(branch.facts.ty, Type::Seq(Box::new(Type::Unit)));
            assert!(matches!(&branch.facts.kind, ExprKind::Sequence(values) if values.is_empty()));
        }
    }
    for emit in [Tla::emit, Tla::safety] {
        let module = emit(&lifted)?;
        assert_eq!(module.metadata.observer_types[FACTS], Type::Seq(Box::new(Type::Unit)));
    }
    Ok(())
}

#[test]
fn canonical_option_constructor_paths_are_admitted() -> Result {
    for prefix in [
        "Option::",
        "core::option::Option::",
        "::core::option::Option::",
        "std::option::Option::",
        "::std::option::Option::",
    ] {
        let source = SOURCE
            .replace("Some(Fact::A)", &format!("{prefix}Some(Fact::A)"))
            .replace("None", &format!("{prefix}None"));
        let lifted = Lift::parse("optional_facts_canonical.rs", &source)?;
        assert!(matches!(lifted.fact_type, Type::Enum { .. }));
    }
    Ok(())
}

#[test]
fn canonical_optional_imports_preserve_the_existing_fact_carrier() -> Result {
    let original = Lift::parse("optional_facts.rs", SOURCE)?;
    for import in [
        "use core::option::Option;",
        "use std::option::{Option, Some, None};",
        "use ::core::option::Option::{Some, None};",
    ] {
        let source = SOURCE.replace(
            "use umpire_dsl::{State, Action};",
            &format!("use umpire_dsl::{{State, Action}}; {import}"),
        );
        let imported = Lift::parse("optional_facts.rs", &source)?;
        assert_eq!(imported.fact_type, original.fact_type);
        for (before, after) in original.actions.iter().zip(&imported.actions) {
            assert_eq!(before.branches[0].facts, after.branches[0].facts);
        }
        for emit in [Tla::emit, Tla::safety] {
            assert_eq!(emit(&original)?.tla, emit(&imported)?.tla);
        }
    }
    Ok(())
}
