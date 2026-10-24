//! Source-derived symbolic safety projection.
use umpire_tla::{Lift, Tla};
const KETTLE: &str = include_str!("../../../dev/fixtures/fixtures/sample_kettle/model.rs");

#[test]
fn tlc_declarations_carry_snowcat_types() -> Result<(), Box<dyn std::error::Error>> {
    let module = Tla::emit(&Lift::parse("kettle.rs", KETTLE)?)?;
    assert!(module.tla.contains("VARIABLE \\* @type: Bool;\nbeeping"));
    assert!(module.tla.contains("\\* @type: Bool;\nInit =="));
    Ok(())
}

#[test]
fn safety_has_typed_history_and_preserving_stutter() -> Result<(), Box<dyn std::error::Error>> {
    let lifted = Lift::parse("kettle.rs", KETTLE)?;
    let tlc = Tla::emit(&lifted)?;
    let module = Tla::safety(&lifted)?;
    for forbidden in ["ENABLED", "Assert(", "Complete ==", "PROPERTY "] {
        assert!(!module.tla.contains(forbidden), "{forbidden}");
    }
    assert!(module.tla.contains("__umpire_before_temperature' = temperature"));
    assert!(module.tla.contains("__umpire_has_step = FALSE"));
    assert!(module.tla.contains("Stutter == __umpire_stutter' = TRUE /\\ UNCHANGED"));
    assert_eq!(module.metadata.postconditions.len(), tlc.metadata.postconditions.len());
    assert_eq!(module.metadata.arithmetic[0].operator, "ArithmeticSafe");
    let root =
        std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../build/apalache/projection");
    std::fs::create_dir_all(&root)?;
    std::fs::write(root.join("Kettle.tla"), &module.tla)?;
    Ok(())
}

#[test]
fn unsupported_source_type_refuses_at_its_declaration() -> Result<(), Box<dyn std::error::Error>> {
    let source = KETTLE.replace("temperature: i32", "temperature: i64");
    assert_ne!(source, KETTLE);
    let error = Lift::parse("wide.rs", &source).err().ok_or("unsupported integer admitted")?;
    assert_eq!(error.file, "wide.rs");
    assert!(error.line > 1);
    assert!(error.message.contains("i64"), "{error}");
    Ok(())
}

#[test]
fn safety_definition_growth_stays_linear() -> Result<(), Box<dyn std::error::Error>> {
    fn source(count: usize) -> String {
        let updates = "if self.value<100 {self.value+=1;}".repeat(count);
        format!(
            "use umpire_dsl::machine; #[machine] mod linear {{ use umpire_dsl::{{State,Action}}; #[derive(State)] struct Model {{value:u8}} #[derive(Action)] enum User{{Run}} impl Model {{ #[on(User)] fn step(&mut self,a:User){{match a{{User::Run=>{{{updates}}}}}}} }} }}"
        )
    }
    let a = Tla::safety(&Lift::parse("linear.rs", &source(16))?)?;
    let b = Tla::safety(&Lift::parse("linear.rs", &source(32))?)?;
    assert!(b.tla.len() < a.tla.len() * 3, "{} -> {}", a.tla.len(), b.tla.len());
    Ok(())
}

#[test]
fn safety_history_names_refuse_collisions_and_temporaries_are_hygienic()
-> Result<(), Box<dyn std::error::Error>> {
    let source = KETTLE.replace("heating_on", "__umpire_before_temperature");
    let lifted = Lift::parse("collision.rs", &source)?;
    Tla::emit(&lifted)?;
    let refusal = Tla::safety(&lifted).err().ok_or("history collision admitted")?;
    assert!(refusal.message.contains("reserved safety name"));
    let source = KETTLE.replace("heating_on", "__umpire_c2").replace("beeping", "__umpire_s2");
    let module = Tla::safety(&Lift::parse("hygiene.rs", &source)?)?;
    assert!(!module.tla.contains("__umpire_c2 =="));
    assert!(!module.tla.contains("__umpire_s2 =="));
    Ok(())
}

#[test]
fn initialization_growth_stays_linear_in_declared_fields() -> Result<(), Box<dyn std::error::Error>>
{
    fn source(count: usize) -> String {
        let fields = (0..count)
            .map(|i| format!("#[starts(0..=(LIMIT+0))] field{i}:i8,"))
            .collect::<Vec<_>>()
            .join(" ");
        format!(
            "use umpire_dsl::machine; #[machine] mod initial {{ use umpire_dsl::{{State,Action}}; const LIMIT:i8=1; #[derive(State)] struct Model {{{fields}}} #[derive(Action)] enum User{{Idle}} impl Model {{#[on(User)] fn step(&mut self,a:User){{match a{{User::Idle=>{{}}}}}}}} }}"
        )
    }
    let a = Tla::safety(&Lift::parse("initial.rs", &source(64))?)?;
    let b = Tla::safety(&Lift::parse("initial.rs", &source(128))?)?;
    assert!(b.tla.len() < a.tla.len() * 3, "{} -> {}", a.tla.len(), b.tla.len());
    Ok(())
}
