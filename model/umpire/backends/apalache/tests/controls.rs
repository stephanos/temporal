//! Actual pinned-release safety and checked-arithmetic controls, never native-edge emission.
use std::{fs, path::Path};
use umpire_apalache::Apalache;
use umpire_lift::{Lift, Tla};
const KETTLE: &str = include_str!("../../../tests/fixtures/sample_kettle/model.rs");
fn required() -> bool {
    std::env::var("UMPIRE_BACKENDS").as_deref() == Ok("require")
}
fn retain(
    label: &str,
    module: &umpire_lift::Module,
    result: &umpire_apalache::Bounded,
) -> Result<(), Box<dyn std::error::Error>> {
    let root = Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("../../../..")
        .join(".flow/tmp/apalache-controls")
        .join(label);
    fs::create_dir_all(&root)?;
    fs::write(root.join(format!("{}.tla", module.name)), &module.tla)?;
    fs::write(root.join("output.txt"), &result.output)?;
    for (i, trace) in result.traces.iter().enumerate() {
        fs::write(root.join(format!("trace-{i}.itf.json")), trace)?;
    }
    Ok(())
}
#[test]
fn real_kettle_checks_source_safety_and_history() -> Result<(), Box<dyn std::error::Error>> {
    if !required() {
        return Ok(());
    }
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..");
    let tool = Apalache::discover(&root)?;
    let module = Tla::safety(&Lift::parse("kettle.rs", KETTLE)?)?;
    for p in module
        .metadata
        .invariants
        .iter()
        .chain(&module.metadata.postconditions)
        .chain(&module.metadata.arithmetic)
    {
        let result = tool.check(&module, &p.operator, 3)?;
        retain(&format!("kettle-{}", p.operator), &module, &result)?;
        assert!(!result.violated, "{}: {}", p.name, result.output);
    }
    Ok(())
}

#[path = "fixtures/arithmetic.rs"]
mod authored;
#[test]
fn real_arithmetic_faults_are_reachable_and_guards_remain_total()
-> Result<(), Box<dyn std::error::Error>> {
    use umpire_lift::{Binding, Concrete, Literal};
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..");
    let tool = if required() { Some(Apalache::discover(&root)?) } else { None };
    for (mode, start, divisor, fails) in [
        (0, 127, 1, true),
        (1, 127, 1, true),
        (2, 1, 0, true),
        (3, 1, 0, true),
        (4, -128, -1, true),
        (5, -128, -1, true),
        (6, 1, 0, false),
        (7, 1, 0, false),
        (8, 1, 0, true),
        (9, 1, 0, true),
        (10, 1, 0, false),
    ] {
        let world = authored::arithmetic::World::new(
            mode,
            if mode == 4 || mode == 5 { 0 } else { start },
            if mode == 4 || mode == 5 { 0 } else { divisor },
        );
        let native = umpire_checker::Checker::explore(&world, 10)?.verify_all()?.to_string();
        let arithmetic = native.contains("overflow")
            || native.contains("divide by zero")
            || native.contains("remainder with a divisor of zero");
        assert_eq!(arithmetic, fails, "mode {mode}: {native}");
        let Some(tool) = &tool else { continue };
        let module = Tla::safety(
            &Lift::parse("arithmetic.rs", include_str!("fixtures/arithmetic.rs"))?.bind(
                &Binding {
                    world: vec![
                        ("mode".into(), Literal::Int(mode.into())),
                        (
                            "start".into(),
                            Literal::Int(if mode == 4 || mode == 5 { 0 } else { start.into() }),
                        ),
                        (
                            "divisor".into(),
                            Literal::Int(if mode == 4 || mode == 5 { 0 } else { divisor.into() }),
                        ),
                    ],
                    starts: vec![vec![
                        ("value".into(), Literal::Int(start.into())),
                        ("divisor".into(), Literal::Int(divisor.into())),
                        ("done".into(), Literal::Bool(false)),
                    ]],
                    actions: vec![Concrete { action: "User::Calculate".into(), fields: vec![] }],
                },
            )?,
        )?;
        for length in if matches!(mode, 0 | 8) { vec![0, 1] } else { vec![1] } {
            let result = tool.check(&module, "ArithmeticSafe", length)?;
            let expected = fails && (length > 0 || mode == 8);
            retain(&format!("arithmetic-{mode}-{length}"), &module, &result)?;
            assert_eq!(result.violated, expected, "mode={mode} length={length}: {}", result.output);
        }
        fs::create_dir_all(root.join(".flow/tmp/apalache-controls"))?;
        fs::write(root.join(format!(".flow/tmp/apalache-controls/native-{mode}.txt")), native)?;
    }
    Ok(())
}

#[test]
fn real_postcondition_filter_excludes_unrelated_actions() -> Result<(), Box<dyn std::error::Error>>
{
    if !required() {
        return Ok(());
    }
    let source = r"
      use umpire_dsl::machine;
      #[machine] mod history {
        use umpire_dsl::{State,Action,check};
        #[derive(State)] struct Model {done:bool}
        #[derive(Action)] enum User {A,B}
        impl Model {
          #[on(User)] fn step(&mut self,a:User){match a {
            User::A if ENABLE_A && !self.done=>{self.done=true;},
            User::B if !self.done=>{self.done=true;},_=>{}
          }}
          #[postcondition(AFTER)] fn fails(&self,before:&Self){check!(self.done==before.done);}
        }
      }";
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..");
    let tool = Apalache::discover(&root)?;
    for (label, enable, after, expected) in [
        ("post-fails", "true", "after=User::A", true),
        ("post-filter", "false", "after=User::A", false),
        ("post-unfiltered", "false", "", true),
    ] {
        let source = source.replace("ENABLE_A", enable).replace("AFTER", after);
        let module = Tla::safety(&Lift::parse("history.rs", &source)?)?;
        let result = tool.check(&module, "Post_fails", 2)?;
        retain(label, &module, &result)?;
        assert_eq!(result.violated, expected, "{label}: {}", result.output);
    }
    Ok(())
}

#[test]
fn real_evaluated_start_failure_cannot_disappear_into_an_empty_range()
-> Result<(), Box<dyn std::error::Error>> {
    if !required() {
        return Ok(());
    }
    let source = r"
      use umpire_dsl::machine;
      #[machine] mod starts {
        use umpire_dsl::{State,Action};
        const LIMIT:i8=127;
        #[derive(State)] struct Model {#[starts(1..=(LIMIT+1))] value:i8}
        #[derive(Action)] enum User {Idle}
        impl Model {#[on(User)] fn step(&mut self,a:User){match a {User::Idle=>{}}}}
      }";
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..");
    let tool = Apalache::discover(&root)?;
    let module = Tla::safety(&Lift::parse("starts.rs", source)?)?;
    let result = tool.check(&module, "ArithmeticSafe", 0)?;
    retain("start-overflow", &module, &result)?;
    assert!(result.violated, "evaluated invalid range disappeared: {}", result.output);
    Ok(())
}

#[test]
fn real_typed_vector_projection_keeps_eager_fallback_errors()
-> Result<(), Box<dyn std::error::Error>> {
    if !required() {
        return Ok(());
    }
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..");
    let tool = Apalache::discover(&root)?;
    let source = include_str!("../../../lift/tests/fixtures/vector.rs");
    let module = Tla::safety(&Lift::parse("vector.rs", source)?)?;
    for (property, violated) in
        [("Invariant_contents", false), ("Invariant_never_two", true), ("ArithmeticSafe", false)]
    {
        let result = tool.check(&module, property, 2)?;
        retain(&format!("vector-{property}"), &module, &result)?;
        assert_eq!(result.violated, violated, "{}", result.output);
    }
    let source = include_str!("../../../lift/tests/fixtures/vector_fallback.rs");
    let module = Tla::safety(&Lift::parse("fallback.rs", source)?)?;
    let result = tool.check(&module, "ArithmeticSafe", 1)?;
    retain("eager-vector-fallback", &module, &result)?;
    assert!(result.violated, "{}", result.output);
    Ok(())
}

#[test]
fn pinned_installer_and_runner_agree() {
    let script = include_str!("../../../../scripts/fetch-apalache.py");
    for pin in
        [umpire_apalache::VERSION, umpire_apalache::ARCHIVE_SHA256, umpire_apalache::JAR_SHA256]
    {
        assert!(script.contains(pin), "pin drift: {pin}");
    }
}

#[test]
fn real_payload_identity_and_outcome_history_are_typed() -> Result<(), Box<dyn std::error::Error>> {
    use umpire_lift::{Binding, Concrete, Literal};
    if !required() {
        return Ok(());
    }
    let source = r"
      use umpire_dsl::machine;
      #[machine] mod inputs {
        use umpire_dsl::{Action,State,World,Outcome,Rejection,reject,check};
        #[derive(World)] struct World {limit:u8}
        #[derive(State)] struct Model {done:bool}
        #[derive(Action)] enum User {Set(bool)}
        impl Model {
          #[on(User)] fn step(&mut self,a:User){match a {
            User::Set(true) if !self.done=>{self.done=true;},
            User::Set(false)=>{reject!(Rejection::NotFound);},_=>{}
          }}
          #[postcondition] fn answer(&self,before:&Self,result:&Outcome){
            if *result==Outcome::Rejected(Rejection::NotFound) {check!(self.done==before.done);}
            else {check!(self.done);}
          }
        }
      }";
    let binding = Binding {
        world: vec![("limit".into(), Literal::Int(1))],
        starts: vec![vec![("done".into(), Literal::Bool(false))]],
        actions: [false, true]
            .into_iter()
            .map(|b| Concrete {
                action: "User::Set".into(),
                fields: vec![("0".into(), Literal::Bool(b))],
            })
            .collect(),
    };
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..");
    let tool = Apalache::discover(&root)?;
    let module = Tla::safety(&Lift::parse("inputs.rs", source)?.bind(&binding)?)?;
    assert_eq!(module.metadata.catalog, binding.actions);
    for property in ["Post_answer", "ArithmeticSafe", "ValidStep"] {
        let result = tool.check(&module, property, 2)?;
        retain(&format!("payload-{property}"), &module, &result)?;
        assert!(!result.violated, "{}", result.output);
    }
    let mutant = source
        .replace("reject!(Rejection::NotFound);", "self.done=true; reject!(Rejection::NotFound);");
    let module = Tla::safety(&Lift::parse("inputs-mutant.rs", &mutant)?.bind(&binding)?)?;
    let result = tool.check(&module, "ValidStep", 1)?;
    retain("rejecting-mutant", &module, &result)?;
    assert!(result.violated, "{}", result.output);
    Ok(())
}
