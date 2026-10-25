//! Authored-source admission and emitter controls.
use umpire_tla::observers::EDGE_TICK;
use umpire_tla::{Lift, Tla, Type};

const KETTLE: &str = include_str!("../../../dev/fixtures/fixtures/sample_kettle/model.rs");

#[test]
fn kettle_source_lifts_registered_properties_and_explicit_assumptions()
-> Result<(), Box<dyn std::error::Error>> {
    let lifted = Lift::parse("sample_kettle/model.rs", KETTLE)?;
    let module = Tla::emit(&lifted)?;
    assert_eq!(module.name, "Kettle");
    assert_eq!(module.metadata.fields.len(), 3);
    assert_eq!(module.metadata.actions.len(), 4);
    assert_eq!(module.metadata.invariants[0].name, "beeps_only_when_off");
    assert_eq!(module.metadata.postconditions.len(), 2);
    assert_eq!(module.metadata.progress[0].allowed.len(), 3);
    assert_eq!(module.metadata.progress[0].fair, Vec::<String>::new());
    assert!(!module.tla.contains("WF_vars("));
    assert!(module.tla.contains(&format!("{EDGE_TICK}' = ~{EDGE_TICK}")));
    assert!(module.tla.contains("Scoped_boils == Assumption_never_switched_off"));
    Ok(())
}

#[test]
fn the_state_struct_keeps_its_authored_name() -> Result<(), Box<dyn std::error::Error>> {
    // Native generation accepts any `#[derive(State)]` name; emission never names the struct.
    let renamed = KETTLE.replace("Model", "Brew");
    let original = Tla::emit(&Lift::parse("sample_kettle/model.rs", KETTLE)?)?;
    let module = Tla::emit(&Lift::parse("sample_kettle/model.rs", &renamed)?)?;
    assert_eq!(module.metadata.state_name, "Brew");
    assert_eq!((module.tla, module.cfg), (original.tla, original.cfg));
    let before = renamed.replace("before: &Self", "before: &Brew");
    assert_ne!(before, renamed, "the before-state parameter is spelled &Self");
    Tla::emit(&Lift::parse("sample_kettle/model.rs", &before)?)?;
    Ok(())
}

#[test]
fn unsupported_expression_refuses_with_source_location() -> Result<(), Box<dyn std::error::Error>> {
    let source =
        KETTLE.replace("self.temperature += 1;", "self.temperature = mystery(self.temperature);");
    let error = Lift::parse("unsupported.rs", &source)
        .err()
        .ok_or("unsupported expression was admitted")?;
    assert_eq!(error.file, "unsupported.rs");
    assert!(error.line > 1);
    assert!(error.to_string().contains("mystery"));
    assert!(error.to_string().contains("warning"));
    assert!(error.to_string().contains("refused"));
    assert_eq!(format!("{error}\n"), include_str!("expected/unsupported.txt"));
    Ok(())
}

#[test]
fn enum_helper_and_terminal_postcondition_are_admitted() -> Result<(), Box<dyn std::error::Error>> {
    let source = r"
        use umpire_dsl::machine;
        #[machine] mod phases {
            use umpire_dsl::{State,Action,check};
            #[derive(Clone,Copy,Debug,Default,PartialEq,Eq,Hash)] enum Mode { #[default] Off, On, Done }
            impl Mode { fn is_done(self)->bool { matches!(self,Self::Done) } }
            #[derive(State)] struct Model { mode:Mode }
            #[derive(Action)] enum User { Advance }
            impl Model {
                #[on(User)] fn user(&mut self,action:User) {
                    match action { User::Advance if self.mode==Mode::Off=>{self.mode=Mode::On;}, User::Advance if self.mode==Mode::On=>{self.mode=Mode::Done;}, _=>{} }
                }
                #[postcondition] fn stays_done(&self,before:&Self) {
                    if before.mode.is_done() { check!(self.mode==before.mode); }
                }
            }
        }
    ";
    let module = Tla::emit(&Lift::parse("enum.rs", source)?)?;
    assert_eq!(module.metadata.actions.len(), 1);
    assert!(
        matches!(&module.metadata.fields[0].ty, Type::Enum { name, variants } if name == "Mode" && variants.len() == 3)
    );
    assert_eq!(module.metadata.postconditions[0].name, "stays_done");
    Ok(())
}

#[test]
fn admission_refuses_type_cfg_resolution_and_observer_collisions()
-> Result<(), Box<dyn std::error::Error>> {
    for (replacement, reason) in [
        ("self.temperature = true;", "type mismatch"),
        ("#[cfg(feature = \"changed\")] self.temperature += 1;", "unsupported"),
        ("self.temperature = self.temperature.abs();", "unsupported method"),
        ("if self.beeping { stay!(); }", "unsupported effect statement"),
    ] {
        let source = KETTLE.replace("self.temperature += 1;", replacement);
        let error =
            Lift::parse("refusal.rs", &source).err().ok_or("unsupported source was admitted")?;
        assert!(error.message.contains(reason), "{error}");
    }
    let source = KETTLE.replace("temperature", EDGE_TICK);
    assert!(
        Lift::parse("collision.rs", &source)
            .err()
            .ok_or("observer collision was admitted")?
            .message
            .contains("reserved observer")
    );
    Ok(())
}

#[test]
fn pure_same_impl_helper_is_typed_and_lifted() {
    let source =
        KETTLE.replace("self.temperature < BOILING_TEMPERATURE", "self.is_cold()").replace(
            "impl Model {",
            "impl Model { fn is_cold(&self) -> bool { self.temperature < BOILING_TEMPERATURE }",
        );
    assert!(Lift::parse("helper.rs", &source).is_ok());
}

#[test]
fn integer_literal_types_follow_field_context_on_either_side()
-> Result<(), Box<dyn std::error::Error>> {
    let source = r"
        use umpire_dsl::machine;
        #[machine] mod tiny {
            use umpire_dsl::{State, Action};
            #[derive(State)] struct Model { #[starts(-128i8..=-128i8)] value: i8 }
            #[derive(Action)] enum User { Set }
            impl Model {
                #[on(User)] fn user(&mut self, action: User) {
                    match action { User::Set if -128 <= self.value => { self.value = 0; }, _ => {} }
                }
            }
        }
    ";
    let module = Tla::emit(&Lift::parse("integer.rs", source)?)?;
    assert!(matches!(module.metadata.fields[0].ty, Type::Int { min: -128, max: 127 }));
    Ok(())
}

#[test]
fn generated_tla_names_refuse_authored_namespace_collisions()
-> Result<(), Box<dyn std::error::Error>> {
    for source in [
        KETTLE.replace("temperature", "vars"),
        KETTLE.replace("BOILING_TEMPERATURE", "Init"),
        KETTLE.replace("temperature", "__umpire_a"),
    ] {
        let error = Lift::parse("namespace.rs", &source)
            .err()
            .ok_or("generated namespace collision was admitted")?;
        assert!(error.message.contains("generated TLA+ name"), "{error}");
        assert!(error.line > 1);
    }
    let source = r"
        use umpire_dsl::machine;
        #[machine] mod collision {
            use umpire_dsl::{State,Action};
            #[derive(State)] struct Model { value:bool }
            #[derive(Action)] enum A { B_C }
            #[derive(Action)] enum A_B { C }
            impl Model {
                #[on(A)] fn first(&mut self,action:A) { match action { A::B_C=>{self.value=true;} } }
                #[on(A_B)] fn second(&mut self,action:A_B) { match action { A_B::C=>{self.value=true;} } }
            }
        }
    ";
    assert!(
        Lift::parse("actions.rs", source)
            .err()
            .ok_or("action operator collision was admitted")?
            .message
            .contains("generated TLA+ name")
    );
    Ok(())
}

#[test]
fn progress_bounds_are_fail_closed() -> Result<(), Box<dyn std::error::Error>> {
    for bound in ["0", "2147483647", "18446744073709551615u64"] {
        let source = KETTLE.replace(", .., self.beeping]", &format!(", ..={bound}, self.beeping]"));
        let error = Lift::parse("bound.rs", &source)
            .err()
            .ok_or("unsupported progress bound was admitted")?;
        assert!(error.line > 1, "{error}");
    }
    let source = KETTLE.replace(", .., self.beeping]", ", ..=2147483646, self.beeping]");
    let module = Tla::emit(&Lift::parse("bound.rs", &source)?)?;
    assert!(matches!(
        module.metadata.observer_types[umpire_tla::observers::STEPS],
        Type::Int { max: 2_147_483_647, .. }
    ));
    Ok(())
}

#[test]
fn repeated_progress_builders_refuse_instead_of_reversing_rust_order()
-> Result<(), Box<dyn std::error::Error>> {
    for suffix in
        [".eventually(false)", ".within(3).within(4)", ".assuming(Self::never_switched_off())"]
    {
        let source = KETTLE
            .replace(
                "path![self.heating_on, .., self.beeping]",
                "whenever(self.heating_on).eventually(self.beeping)",
            )
            .replace(
                ".assuming(Self::never_switched_off())",
                &format!(".assuming(Self::never_switched_off()){suffix}"),
            );
        let error = Lift::parse("repeated.rs", &source)
            .err()
            .ok_or("repeated builder was silently reordered")?;
        assert!(error.message.contains("repeated progress builder"), "{error}");
    }
    Ok(())
}

#[test]
fn unsupported_control_characters_refuse_at_the_authored_literal()
-> Result<(), Box<dyn std::error::Error>> {
    for literal in [r#""a\0b""#, r#""a\u{7f}b""#] {
        let source = KETTLE.replace(
            "/// Current temperature.",
            &format!("/// Current temperature.\n#[doc = {literal}]"),
        );
        let error = Lift::parse("literal.rs", &source)
            .err()
            .ok_or("unsupported string control character was admitted")?;
        assert!(error.message.contains("control character"), "{error}");
        assert!(error.line > 1);
    }
    Ok(())
}

#[test]
fn inherent_enum_into_is_resolved_before_primitive_conversion()
-> Result<(), Box<dyn std::error::Error>> {
    let source = r"
        use umpire_dsl::machine;
        #[machine] mod intrinsic {
            use umpire_dsl::{State,Action};
            #[derive(Clone,Copy,Debug,Default,PartialEq,Eq,Hash)] enum Mode { #[default] Off, On }
            impl Mode { fn into(self)->Self { Self::On } }
            #[derive(State)] struct Model { mode:Mode }
            #[derive(Action)] enum User { Set }
            impl Model {
                #[on(User)] fn user(&mut self,action:User) {match action {User::Set=>{self.mode=self.mode.into();}}}
            }
        }
    ";
    let module = Tla::emit(&Lift::parse("intrinsic.rs", source)?)?;
    assert!(module.tla.contains("mode' = \"Mode::On\""));
    Ok(())
}

#[test]
fn path_claims_refuse_builders_chains_and_other_gaps() -> Result<(), Box<dyn std::error::Error>> {
    let mut refusals = Vec::new();
    for replacement in [
        "path![self.heating_on, .., self.beeping].within(3)",
        "path![self.heating_on, .., self.heating_on, .., self.beeping]",
        "path![self.heating_on, 1..3, self.beeping]",
    ] {
        let source = KETTLE.replace("path![self.heating_on, .., self.beeping]", replacement);
        let error =
            Lift::parse("path.rs", &source).err().ok_or("malformed path claim was admitted")?;
        refusals.push(format!("{error}\n"));
    }
    assert_eq!(refusals.concat(), include_str!("expected/path_refusals.txt"));
    Ok(())
}

#[test]
fn hand_written_machine_refusal_prints_the_type_as_written()
-> Result<(), Box<dyn std::error::Error>> {
    let source = "impl<W> Machine for &'static mut W {}";
    let error = Lift::parse("by_hand.rs", source).err().ok_or("manual Machine was admitted")?;
    assert!(error.message.contains("`impl Machine for &'static mut W`"), "{error}");
    Ok(())
}

#[test]
fn hook_refusal_in_tests_rs_names_tests_rs() -> Result<(), Box<dyn std::error::Error>> {
    let root = std::env::temp_dir().join(format!("umpire-hook-refusal-{}", std::process::id()));
    std::fs::create_dir_all(&root)?;
    std::fs::write(root.join("model.rs"), KETTLE)?;
    std::fs::write(
        root.join("tests.rs"),
        "use super::*;\n\numpire_testing::fixture!(\n    kettle::world(),\n    16,\n    Claim::new(\"bad\", |_, state: &kettle::Model| mystery(state.beeping).into())\n);\n",
    )?;
    let (model, source, _) = umpire_front::read(&syn::parse_file(KETTLE)?, &[])
        .map_err(|refusal| format!("{refusal:?}"))?;
    let file = root.join("model.rs").to_string_lossy().into_owned();
    let embedded = umpire_tla::Embedded { file, model: Ok(model), source };
    let error = Lift::embedded(Some(&embedded), root.join("model.rs"))
        .err()
        .ok_or("refused hook claim admitted");
    std::fs::remove_dir_all(&root)?;
    let error = error?;
    assert!(error.file.ends_with("tests.rs"), "{error}");
    assert_eq!(error.line, 6, "{error}");
    assert!(error.message.contains("mystery"), "{error}");
    Ok(())
}

#[test]
fn imports_that_can_rebind_a_macro_read_by_name_refuse() -> Result<(), Box<dyn std::error::Error>> {
    // The subset reads check!, stay!, reject!, unknown!, path! and matches! by name.
    for import in [
        "use umpire::dsl::{Action, State, path}; use crate::support::check;",
        "use umpire::dsl::{Action, State, check, path}; use crate::support::*;",
        "use umpire::dsl::{Action, State, check, path, stay as matches};",
    ] {
        let source = KETTLE.replace("use umpire::dsl::{Action, State, check, path};", import);
        let error =
            Lift::parse("imports.rs", &source).err().ok_or("rebinding import was admitted")?;
        assert!(error.message.contains("can rebind a macro"), "{error}");
        assert!(error.line > 1, "{error}");
    }
    Ok(())
}

#[test]
fn a_world_of_another_machine_refuses_at_this_source() -> Result<(), Box<dyn std::error::Error>> {
    let root = std::env::temp_dir().join(format!("umpire-other-world-{}", std::process::id()));
    std::fs::create_dir_all(&root)?;
    std::fs::write(root.join("model.rs"), KETTLE)?;
    std::fs::write(root.join("other.rs"), KETTLE)?;
    let (model, source, _) = umpire_front::read(&syn::parse_file(KETTLE)?, &[])
        .map_err(|refusal| format!("{refusal:?}"))?;
    let file = root.join("other.rs").to_string_lossy().into_owned();
    let embedded = umpire_tla::Embedded { file, model: Ok(model), source };
    let error = Lift::embedded(Some(&embedded), root.join("model.rs")).err();
    std::fs::remove_dir_all(&root)?;
    let error = error.ok_or("another source's Model was admitted")?;
    assert!(error.file.ends_with("model.rs"), "{error}");
    assert!(error.message.contains("is not the #[machine] World this source declares"), "{error}");
    Ok(())
}

#[test]
fn a_hook_claim_name_collision_refuses_in_tests_rs() -> Result<(), Box<dyn std::error::Error>> {
    let root = std::env::temp_dir().join(format!("umpire-hook-collision-{}", std::process::id()));
    std::fs::create_dir_all(&root)?;
    std::fs::write(root.join("model.rs"), KETTLE)?;
    std::fs::write(
        root.join("tests.rs"),
        "use super::*;\n\numpire_testing::fixture!(\n    kettle::world(),\n    16,\n    Claim::new(\"never beeps\", |_, state: &kettle::Model| (!state.beeping).into())\n);\n",
    )?;
    let (model, source, _) = umpire_front::read(&syn::parse_file(KETTLE)?, &[])
        .map_err(|refusal| format!("{refusal:?}"))?;
    let file = root.join("model.rs").to_string_lossy().into_owned();
    let embedded = umpire_tla::Embedded { file, model: Ok(model), source };
    let error = Lift::embedded(Some(&embedded), root.join("model.rs")).err();
    std::fs::remove_dir_all(&root)?;
    let error = error.ok_or("a claim name outside TLA+ identifiers was admitted")?;
    assert!(error.file.ends_with("tests.rs"), "{error}");
    assert_eq!((error.line, error.column), (6, 16), "{error}");
    assert!(error.message.contains("`Negative_never beeps`"), "{error}");
    Ok(())
}
