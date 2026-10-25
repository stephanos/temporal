//! Actual pinned-release safety and checked-arithmetic controls, never native-edge emission.
use std::{
    fs,
    path::{Path, PathBuf},
};
use umpire_apalache::{Apalache, Confirmation, Replay, native_bound};
use umpire_dsl::Machine;
#[allow(unreachable_pub, reason = "the canonical fixture is also included by public host crates")]
#[path = "../../../dev/fixtures/fixtures/sample_kettle/model.rs"]
mod selected_kettle;
use umpire_tla::{Lift, Tla};
const KETTLE: &str = include_str!("../../../dev/fixtures/fixtures/sample_kettle/model.rs");
// Apalache stays out of `UMPIRE_BACKENDS=require` until fn-211 resumes; opt in separately.
fn required() -> bool {
    std::env::var("UMPIRE_APALACHE").as_deref() == Ok("require")
}
fn evidence_root() -> PathBuf {
    Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..").join(".flow/tmp/apalache-controls")
}
fn retain(
    label: &str,
    module: &umpire_tla::Module,
    result: &umpire_apalache::Bounded,
) -> Result<(), Box<dyn std::error::Error>> {
    let root = evidence_root().join(label);
    fs::create_dir_all(&root)?;
    fs::write(root.join(format!("{}.tla", module.name)), &module.tla)?;
    fs::write(root.join("output.txt"), result.output())?;
    for (i, trace) in result.traces().iter().enumerate() {
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
    let native = umpire_checker::Checker::explore(
        &selected_kettle::kettle::world(),
        umpire_checker::Checker::BUDGET,
    )?;
    let length = native_bound(native.counts)?;
    assert_eq!(
        (native.counts.reached, native.counts.depth, native.counts.cut, length),
        (283, 62, false, 63)
    );
    fs::create_dir_all(evidence_root())?;
    fs::write(
        evidence_root().join("kettle-native-bound.json"),
        format!(
            "{{\"reached\":{},\"depth\":{},\"cut\":{},\"transition_bound\":{length}}}\n",
            native.counts.reached, native.counts.depth, native.counts.cut
        ),
    )?;
    let module = Tla::safety(&Lift::parse("kettle.rs", KETTLE)?)?;
    for p in module
        .metadata
        .invariants
        .iter()
        .chain(&module.metadata.postconditions)
        .chain(&module.metadata.arithmetic)
    {
        let result = tool.check(&module, &p.operator, length)?;
        retain(&format!("kettle-{}", p.operator), &module, &result)?;
        assert!(!result.violated()?, "{}: {}", p.name, result.output());
    }
    Ok(())
}

#[path = "fixtures/arithmetic.rs"]
mod authored;
#[test]
fn real_arithmetic_faults_are_reachable_and_guards_remain_total()
-> Result<(), Box<dyn std::error::Error>> {
    use umpire_tla::{Binding, Concrete, Literal};
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
            assert_eq!(
                result.violated()?,
                expected,
                "mode={mode} length={length}: {}",
                result.output()
            );
            if expected {
                let starts = world.starts();
                let catalog = world.checked_catalog(umpire_dsl::CATALOG_LIMIT)?;
                let replay = Replay {
                    world: &world,
                    starts: &starts,
                    actions: catalog.actions(),
                    negative: &[],
                };
                for raw in result.traces() {
                    let confirmed = replay.violation(&module, "ArithmeticSafe", length, raw)?;
                    assert!(matches!(confirmed.confirmation, Confirmation::Arithmetic { .. }));
                    fs::write(
                        evidence_root().join(format!("arithmetic-{mode}-{length}/replay.txt")),
                        confirmed.to_string(),
                    )?;
                }
            }
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
        assert_eq!(result.violated()?, expected, "{label}: {}", result.output());
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
    assert!(result.violated()?, "evaluated invalid range disappeared: {}", result.output());
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
    let source = include_str!("../../tla/tests/fixtures/vector.rs");
    let module = Tla::safety(&Lift::parse("vector.rs", source)?)?;
    for (property, violated) in
        [("Invariant_contents", false), ("Invariant_never_two", true), ("ArithmeticSafe", false)]
    {
        let result = tool.check(&module, property, 2)?;
        retain(&format!("vector-{property}"), &module, &result)?;
        assert_eq!(result.violated()?, violated, "{}", result.output());
    }
    let source = include_str!("../../tla/tests/fixtures/vector_fallback.rs");
    let module = Tla::safety(&Lift::parse("fallback.rs", source)?)?;
    let result = tool.check(&module, "ArithmeticSafe", 1)?;
    retain("eager-vector-fallback", &module, &result)?;
    assert!(result.violated()?, "{}", result.output());
    Ok(())
}

#[test]
fn pinned_installer_and_runner_agree() {
    let script = include_str!("../../../../dev/xtask/src/apalache_pins.rs");
    for pin in
        [umpire_apalache::VERSION, umpire_apalache::ARCHIVE_SHA256, umpire_apalache::JAR_SHA256]
    {
        assert!(script.contains(pin), "pin drift: {pin}");
    }
}

#[test]
fn real_payload_identity_and_outcome_history_are_typed() -> Result<(), Box<dyn std::error::Error>> {
    use umpire_tla::{Binding, Concrete, Literal};
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
        assert!(!result.violated()?, "{}", result.output());
    }
    let mutant = source
        .replace("reject!(Rejection::NotFound);", "self.done=true; reject!(Rejection::NotFound);");
    let module = Tla::safety(&Lift::parse("inputs-mutant.rs", &mutant)?.bind(&binding)?)?;
    let result = tool.check(&module, "ValidStep", 1)?;
    retain("rejecting-mutant", &module, &result)?;
    assert!(result.violated()?, "{}", result.output());
    Ok(())
}

#[path = "fixtures/empty_condition.rs"]
mod empty_body_fixture;
#[path = "fixtures/start_empty_first.rs"]
mod start_empty_first;
#[path = "fixtures/start_fault_first.rs"]
mod start_fault_first;

#[test]
fn real_start_evaluation_respects_field_order_before_empty_domains()
-> Result<(), Box<dyn std::error::Error>> {
    use umpire_dsl::Machine;
    assert!(
        std::panic::catch_unwind(|| start_fault_first::faulty_first::world().starts()).is_err()
    );
    assert_eq!(start_empty_first::empty_first::world().starts(), vec![]);
    if !required() {
        return Ok(());
    }
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..");
    let tool = Apalache::discover(&root)?;
    for (label, source, fails) in [
        ("start-fault-first", include_str!("fixtures/start_fault_first.rs"), true),
        ("start-empty-first", include_str!("fixtures/start_empty_first.rs"), false),
    ] {
        let module = Tla::safety(&Lift::parse(label, source)?)?;
        let result = tool.check(&module, "ArithmeticSafe", 0)?;
        retain(label, &module, &result)?;
        assert_eq!(result.violated()?, fails, "{label}: {}", result.output());
        if fails {
            let world = start_fault_first::faulty_first::World;
            let actions = world.actions();
            let replay = Replay { world: &world, starts: &[], actions: &actions, negative: &[] };
            for raw in result.traces() {
                let confirmed = replay.violation(&module, "ArithmeticSafe", 0, raw)?;
                assert!(matches!(
                    confirmed.confirmation,
                    Confirmation::Arithmetic { index: 0, .. }
                ));
                fs::write(
                    evidence_root().join("start-fault-first/replay.txt"),
                    confirmed.to_string(),
                )?;
            }
        }
    }
    Ok(())
}

#[test]
fn real_discarded_condition_keeps_its_evaluation_guard() -> Result<(), Box<dyn std::error::Error>> {
    use umpire_tla::{Binding, Concrete, Literal};
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..");
    let tool = if required() { Some(Apalache::discover(&root)?) } else { None };
    for enabled in [true, false] {
        let world = empty_body_fixture::empty_condition::World::new(0, enabled);
        let native = umpire_checker::Checker::explore(&world, 10)?.verify_all()?.to_string();
        assert_eq!(native.contains("divide by zero"), enabled, "{native}");
        let Some(tool) = &tool else { continue };
        let module = Tla::safety(
            &Lift::parse("empty_condition.rs", include_str!("fixtures/empty_condition.rs"))?.bind(
                &Binding {
                    world: vec![
                        ("divisor".into(), Literal::Int(0)),
                        ("enabled".into(), Literal::Bool(enabled)),
                    ],
                    starts: vec![vec![]],
                    actions: vec![Concrete { action: "User::Check".into(), fields: vec![] }],
                },
            )?,
        )?;
        let result = tool.check(&module, "ArithmeticSafe", 1)?;
        retain(&format!("discarded-condition-{enabled}"), &module, &result)?;
        assert_eq!(result.violated()?, enabled, "{}", result.output());
        if enabled {
            let starts = world.starts();
            let actions = world.actions();
            let replay =
                Replay { world: &world, starts: &starts, actions: &actions, negative: &[] };
            for raw in result.traces() {
                let confirmed = replay.violation(&module, "ArithmeticSafe", 1, raw)?;
                assert!(matches!(
                    confirmed.confirmation,
                    Confirmation::Arithmetic { index: 1, .. }
                ));
                fs::write(
                    evidence_root().join("discarded-condition-true/replay.txt"),
                    confirmed.to_string(),
                )?;
            }
        }
    }
    Ok(())
}

#[test]
fn real_kettle_negative_is_replayed_and_tampering_fails() -> Result<(), Box<dyn std::error::Error>>
{
    if !required() {
        return Ok(());
    }
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..");
    let tool = Apalache::discover(&root)?;
    let world = selected_kettle::kettle::world();
    let starts = world.starts();
    let catalog = world.checked_catalog(umpire_dsl::CATALOG_LIMIT)?;
    let native = umpire_checker::Checker::explore_within(
        &world,
        starts.clone(),
        umpire_checker::Checker::BUDGET,
        umpire_dsl::CATALOG_LIMIT,
    )?;
    let length = native_bound(native.counts)?;
    assert_eq!(
        (native.counts.reached, native.counts.depth, native.counts.cut, length),
        (283, 62, false, 63)
    );
    let source = format!(
        r#"{KETTLE}
        umpire_testing::fixture!(kettle::world(),1000,Claim::new("never_beeps",|_,s:&kettle::Model|(!s.beeping).into()));
    "#
    );
    let module = Tla::safety(&Lift::parse("kettle.rs", &source)?)?;
    let negative =
        [umpire_dsl::Claim::new("never_beeps", |_, s: &selected_kettle::kettle::Model| {
            (!s.beeping).into()
        })];
    let replay =
        Replay { world: &world, starts: &starts, actions: catalog.actions(), negative: &negative };
    let result = tool.check(&module, "Negative_never_beeps", length)?;
    retain("kettle-negative", &module, &result)?;
    assert!(result.violated()?, "{}", result.output());
    for raw in result.traces() {
        let confirmed = replay.violation(&module, "Negative_never_beeps", length, raw)?;
        assert_eq!(confirmed.confirmation, Confirmation::Violation);
        fs::write(evidence_root().join("kettle-negative/replay.txt"), confirmed.to_string())?;
        assert_eq!(confirmed.to_string(), include_str!("expected/kettle-negative.txt"));
        let mut tampered: serde_json::Value = serde_json::from_str(raw)?;
        tampered["states"][1]["__umpire_action"] = serde_json::json!("User::DepressButton");
        assert!(
            replay
                .violation(&module, "Negative_never_beeps", length, &tampered.to_string())
                .is_err()
        );
        assert!(replay.violation(&module, "Invariant_beeps_only_when_off", length, raw).is_err());
    }
    Ok(())
}

#[path = "fixtures/arithmetic_prefix.rs"]
mod prefix_fixture;
#[test]
fn real_arithmetic_refusal_follows_a_replayed_valid_prefix()
-> Result<(), Box<dyn std::error::Error>> {
    if !required() {
        return Ok(());
    }
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..");
    let tool = Apalache::discover(&root)?;
    let world = prefix_fixture::prefix::World;
    let starts = world.starts();
    let actions = world.actions();
    let module =
        Tla::safety(&Lift::parse("prefix.rs", include_str!("fixtures/arithmetic_prefix.rs"))?)?;
    let result = tool.check(&module, "ArithmeticSafe", 2)?;
    retain("arithmetic-prefix", &module, &result)?;
    assert!(result.violated()?, "{}", result.output());
    let replay = Replay { world: &world, starts: &starts, actions: &actions, negative: &[] };
    for raw in result.traces() {
        let confirmed = replay.violation(&module, "ArithmeticSafe", 2, raw)?;
        assert!(matches!(confirmed.confirmation, Confirmation::Arithmetic { index: 2, .. }));
        fs::write(evidence_root().join("arithmetic-prefix/replay.txt"), confirmed.to_string())?;
    }
    Ok(())
}

#[path = "fixtures/replay.rs"]
mod replay_fixture;
#[test]
fn real_typed_payload_rejection_and_after_claims_replay() -> Result<(), Box<dyn std::error::Error>>
{
    use umpire_tla::{Binding, Concrete, Literal};
    if !required() {
        return Ok(());
    }
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..");
    let tool = Apalache::discover(&root)?;
    let world = replay_fixture::inputs::World::new(2);
    let starts = world.starts();
    let catalog = world.checked_catalog(umpire_dsl::CATALOG_LIMIT)?;
    let source = format!(
        r#"{}
        umpire_testing::fixture!(inputs::World::new(2),100,Claim::new("never_two",|_,s:&inputs::Model|(s.value!=2).into()));
    "#,
        include_str!("fixtures/replay.rs")
    );
    let binding = Binding {
        world: vec![("limit".into(), Literal::Int(2))],
        starts: vec![
            vec![("value".into(), Literal::Int(0))],
            vec![("value".into(), Literal::Int(1))],
        ],
        actions: vec![
            Concrete { action: "User::Set".into(), fields: vec![("0".into(), Literal::Int(1))] },
            Concrete { action: "User::Set".into(), fields: vec![("0".into(), Literal::Int(1))] },
            Concrete { action: "User::Set".into(), fields: vec![("0".into(), Literal::Int(2))] },
            Concrete { action: "User::Reset".into(), fields: vec![] },
        ],
    };
    let module = Tla::safety(&Lift::parse("inputs.rs", &source)?.bind(&binding)?)?;
    let negatives =
        [umpire_dsl::Claim::new("never_two", |_, s: &replay_fixture::inputs::Model| {
            (s.value != 2).into()
        })];
    let replay =
        Replay { world: &world, starts: &starts, actions: catalog.actions(), negative: &negatives };
    for property in ["Negative_never_two", "Post_no_reset", "Post_no_rejection"] {
        let result = tool.check(&module, property, 1)?;
        retain(&format!("replay-{property}"), &module, &result)?;
        assert!(result.violated()?, "{}", result.output());
        for raw in result.traces() {
            let confirmed = replay.violation(&module, property, 1, raw)?;
            assert_eq!(confirmed.confirmation, Confirmation::Violation);
            fs::write(
                evidence_root().join(format!("replay-{property}/replay.txt")),
                confirmed.to_string(),
            )?;
            let mut tampered: serde_json::Value = serde_json::from_str(raw)?;
            tampered["states"][1]["__umpire_outcome"] =
                serde_json::json!("Outcome::Rejected(Unavailable)");
            assert!(replay.violation(&module, property, 1, &tampered.to_string()).is_err());
        }
    }
    Ok(())
}

#[path = "fixtures/start_later_fault.rs"]
mod start_later_fault;
#[test]
fn real_later_start_fault_preserves_partial_binding() -> Result<(), Box<dyn std::error::Error>> {
    let world = start_later_fault::later_fault::World;
    assert!(std::panic::catch_unwind(|| world.starts()).is_err());
    if !required() {
        return Ok(());
    }
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..");
    let tool = Apalache::discover(&root)?;
    let module =
        Tla::safety(&Lift::parse("later.rs", include_str!("fixtures/start_later_fault.rs"))?)?;
    let result = tool.check(&module, "ArithmeticSafe", 0)?;
    retain("start-later-fault", &module, &result)?;
    assert!(result.violated()?, "{}", result.output());
    assert_ne!(result.traces().len(), 0);
    let actions = world.actions();
    let replay = Replay { world: &world, starts: &[], actions: &actions, negative: &[] };
    for raw in result.traces() {
        let confirmed = replay.violation(&module, "ArithmeticSafe", 0, raw)?;
        assert!(matches!(confirmed.confirmation, Confirmation::Arithmetic { index: 0, .. }));
        fs::write(evidence_root().join("start-later-fault/replay.txt"), confirmed.to_string())?;
        let mut alternate: serde_json::Value = serde_json::from_str(raw)?;
        alternate["states"][0]["earlier"]["#bigint"] = "8".into();
        replay.violation(&module, "ArithmeticSafe", 0, &alternate.to_string())?;
        for (field, value) in [("earlier", "0"), ("later", "1")] {
            let mut tampered: serde_json::Value = serde_json::from_str(raw)?;
            tampered["states"][0][field]["#bigint"] = value.into();
            assert!(replay.violation(&module, "ArithmeticSafe", 0, &tampered.to_string()).is_err());
        }
    }
    Ok(())
}

#[path = "../../tla/tests/fixtures/vector.rs"]
mod vector_fixture;
#[test]
fn real_vector_witness_replays_typed_sequence() -> Result<(), Box<dyn std::error::Error>> {
    use umpire_dsl::{Value, Values};
    let world = vector_fixture::collection::World;
    let starts = world.starts();
    assert_eq!(starts[0].values(), [("values", Value::Sequence(vec![]))]);
    let actions = world.actions();
    if !required() {
        return Ok(());
    }
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..");
    let tool = Apalache::discover(&root)?;
    let module = Tla::safety(&Lift::parse(
        "vector.rs",
        include_str!("../../tla/tests/fixtures/vector.rs"),
    )?)?;
    let result = tool.check(&module, "Invariant_never_two", 2)?;
    retain("vector-replay", &module, &result)?;
    assert!(result.violated()?, "{}", result.output());
    assert_ne!(result.traces().len(), 0);
    let replay = Replay { world: &world, starts: &starts, actions: &actions, negative: &[] };
    for raw in result.traces() {
        let confirmed = replay.violation(&module, "Invariant_never_two", 2, raw)?;
        assert!(matches!(confirmed.confirmation, Confirmation::Violation));
        assert_eq!(confirmed.steps.len(), 2);
        fs::write(evidence_root().join("vector-replay/replay.txt"), confirmed.to_string())?;
        let mut tampered: serde_json::Value = serde_json::from_str(raw)?;
        tampered["states"][2]["values"][1]["#bigint"] = "6".into();
        assert!(
            replay.violation(&module, "Invariant_never_two", 2, &tampered.to_string()).is_err()
        );
    }
    Ok(())
}

#[test]
fn strict_envelope_presence_and_object_controls() -> Result<(), Box<dyn std::error::Error>> {
    use std::collections::BTreeMap;
    use umpire_apalache::decode_itf;
    use umpire_tla::{
        Type,
        values::{Schema, Value},
    };

    let schema = Schema {
        fields: BTreeMap::from([("flag".into(), Type::Bool)]),
        observers: BTreeMap::new(),
    };
    let raw = r##"{"#meta":{"format":"ITF"},"vars":["flag"],"states":[{"#meta":{"index":0},"flag":true}]}"##;
    for declared in [raw.to_owned(), raw.replacen("\"states\":", "\"params\":[],\"states\":", 1)] {
        let trace = decode_itf(&declared, &schema, &BTreeMap::new(), 0)?;
        assert!(trace.parameters.is_empty());
        assert_eq!(trace.states[0].state["flag"], Value::Bool(true));
    }
    let base: serde_json::Value = serde_json::from_str(raw)?;
    for field in ["#meta", "vars", "states", "params"] {
        let mut null = base.clone();
        null[field] = serde_json::Value::Null;
        assert!(
            decode_itf(&null.to_string(), &schema, &BTreeMap::new(), 0).is_err(),
            "null {field}"
        );
        if field != "params" {
            let mut missing = base.clone();
            missing.as_object_mut().ok_or("expected object")?.remove(field);
            assert!(
                decode_itf(&missing.to_string(), &schema, &BTreeMap::new(), 0).is_err(),
                "missing {field}"
            );
        }
    }
    for extra in ["header", "extensions", "loop", "unknown"] {
        let mut unknown = base.clone();
        unknown[extra] = serde_json::json!({});
        assert!(
            decode_itf(&unknown.to_string(), &schema, &BTreeMap::new(), 0).is_err(),
            "extra {extra}"
        );
    }
    let positional = serde_json::json!([base["#meta"], base["vars"], base["states"]]);
    assert!(decode_itf(&positional.to_string(), &schema, &BTreeMap::new(), 0).is_err());
    for field in ["#meta", "vars", "states", "params"] {
        let value = if field == "params" { serde_json::json!([]) } else { base[field].clone() };
        let duplicate = format!("{{\"{field}\":{value},\"{field}\":{value},{}", &raw[1..]);
        assert!(decode_itf(&duplicate, &schema, &BTreeMap::new(), 0).is_err(), "duplicate {field}");
    }
    for (from, to) in [
        ("\"format\":\"ITF\"", "\"format\":\"ITF\",\"format\":\"ITF\""),
        ("\"format\":\"ITF\"", "\"format\":\"ITF\",\"unknown\":true"),
        ("\"format\":\"ITF\"", "\"format\":\"ITF\",\"description\":null"),
        ("\"format\":\"ITF\"", "\"format\":\"ITF\",\"format-description\":null"),
        ("\"format\":\"ITF\"", "\"format\":\"ITF\",\"varTypes\":null"),
        ("\"index\":0", "\"index\":0,\"index\":0"),
        ("\"index\":0", "\"index\":0,\"unknown\":true"),
        ("\"index\":0", "\"index\":null"),
        ("\"flag\":true", "\"flag\":null"),
        ("\"flag\":true", "\"flag\":1.5"),
    ] {
        let bad = raw.replace(from, to);
        assert_ne!(bad, raw);
        assert!(decode_itf(&bad, &schema, &BTreeMap::new(), 0).is_err(), "accepted {bad}");
    }
    Ok(())
}

#[test]
fn strict_envelope_world_and_state_bounds() -> Result<(), Box<dyn std::error::Error>> {
    use std::collections::BTreeMap;
    use umpire_apalache::decode_itf;
    use umpire_tla::{
        Type,
        values::{Schema, Value},
    };

    let schema = Schema {
        fields: BTreeMap::from([("flag".into(), Type::Bool)]),
        observers: BTreeMap::new(),
    };
    let parameters = BTreeMap::from([("world".into(), Type::Bool)]);
    let raw = r##"{"#meta":{"format":"ITF","description":"current tool witness","format-description":"ITF","varTypes":{"flag":"Bool"}},"vars":["flag"],"params":["world"],"states":[{"#meta":{"index":0},"flag":false,"world":true},{"#meta":{"index":1},"flag":true,"world":true}]}"##;
    let trace = decode_itf(raw, &schema, &parameters, 1)?;
    assert_eq!(trace.parameters["world"], Value::Bool(true));
    assert_eq!(trace.states.len(), 2);
    assert!(decode_itf(raw, &schema, &parameters, 0).is_err());
    let mut empty: serde_json::Value = serde_json::from_str(raw)?;
    empty["states"] = serde_json::json!([]);
    assert!(decode_itf(&empty.to_string(), &schema, &parameters, 1).is_err());
    for (from, to) in [
        (",\"params\":[\"world\"]", ""),
        ("\"flag\":true,\"world\":true", "\"flag\":true"),
        ("\"flag\":true,\"world\":true", "\"flag\":true,\"world\":false"),
        ("\"index\":1", "\"index\":2"),
    ] {
        let bad = raw.replace(from, to);
        assert_ne!(bad, raw);
        assert!(decode_itf(&bad, &schema, &parameters, 1).is_err(), "accepted {bad}");
    }
    Ok(())
}

#[test]
fn strict_envelope_resource_controls() -> Result<(), Box<dyn std::error::Error>> {
    use std::collections::BTreeMap;
    use umpire_apalache::decode_itf;
    use umpire_tla::{Type, values::Schema};

    let mut kind = Type::Bool;
    for _ in 0..61 {
        kind = Type::Seq(Box::new(kind));
    }
    let schema =
        Schema { fields: BTreeMap::from([("x".into(), kind)]), observers: BTreeMap::new() };
    let nested = format!("{}true{}", "[".repeat(61), "]".repeat(61));
    let raw = format!(
        r##"{{"#meta":{{"format":"ITF"}},"vars":["x"],"states":[{{"#meta":{{"index":0}},"x":{nested}}}]}}"##
    );
    decode_itf(&raw, &schema, &BTreeMap::new(), 0)?;
    let deeper = raw.replace(&nested, &format!("[{nested}]"));
    assert!(
        decode_itf(&deeper, &schema, &BTreeMap::new(), 0)
            .is_err_and(|e| e.0.contains("nesting limit"))
    );
    let padded = format!("{raw}{}", " ".repeat(4 * 1024 * 1024 - raw.len()));
    decode_itf(&padded, &schema, &BTreeMap::new(), 0)?;
    assert!(
        decode_itf(&(padded + " "), &schema, &BTreeMap::new(), 0)
            .is_err_and(|e| e.0.contains("byte limit"))
    );
    let many = format!(
        r##"{{"#meta":{{"format":"ITF"}},"vars":["x"],"states":[{}]}}"##,
        vec!["true"; 100_001].join(",")
    );
    assert!(
        decode_itf(&many, &schema, &BTreeMap::new(), 0)
            .is_err_and(|e| e.0.contains("aggregate item limit"))
    );
    Ok(())
}

#[allow(unreachable_pub, reason = "shared authoring fixture")]
#[path = "../../../dev/fixtures/fixtures/optional_facts/model.rs"]
mod fact_fixture;

#[test]
fn real_fact_witnesses_replay_exact_vectors_and_stutters() -> Result<(), Box<dyn std::error::Error>> {
    use umpire_dsl::__lifted::BindingOf;
    use umpire_tla::{Binding, values::Value};
    if !required() { return Ok(()); }
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..");
    let tool = Apalache::discover(&root)?;
    let world = fact_fixture::facts::World::new(false);
    let starts = world.starts();
    let catalog = world.checked_catalog(umpire_dsl::CATALOG_LIMIT)?;
    let binding = Binding::of(&world, &starts, catalog.actions());
    let source = include_str!("../../../dev/fixtures/fixtures/optional_facts/model.rs");
    let module = Tla::safety(&Lift::parse("facts.rs", source)?.bind(&binding)?)?;
    let replay = Replay { world: &world, starts: &starts, actions: catalog.actions(), negative: &[] };
    for (property, fact) in [("never_a", "A"), ("never_b", "B"), ("never_c", "C"), ("never_d", "D"), ("never_clear", "")] {
        let property = format!("Invariant_{property}");
        let length = if fact.is_empty() { 2 } else { 1 };
        let result = tool.check(&module, &property, length)?;
        retain(&format!("facts-{property}"), &module, &result)?;
        assert!(result.violated()?, "{}", result.output());
        assert!(!result.traces().is_empty());
        for raw in result.traces() {
            let confirmed = replay.violation(&module, &property, length, raw)?;
            let expected = if fact.is_empty() { vec![] } else { vec![Value::Text(format!("Fact::{fact}"))] };
            assert_eq!(confirmed.steps.last().ok_or("no replayed edge")?.facts, expected);
            let mut stutter: serde_json::Value = serde_json::from_str(raw)?;
            let rows = stutter["states"].as_array_mut().ok_or("no states")?;
            let mut last = rows.last().ok_or("no final state")?.clone();
            last["#meta"]["index"] = serde_json::json!(rows.len());
            last["__umpire_stutter"] = serde_json::json!(true);
            rows.push(last);
            assert_eq!(replay.violation(&module, &property, length + 1, &stutter.to_string())?.steps, confirmed.steps);
            for replacement in [serde_json::json!(["Fact::A", "Fact::A"]), serde_json::json!(["Fact::B"]), serde_json::json!([]), serde_json::json!([true])] {
                if replacement == serde_json::from_str::<serde_json::Value>(raw)?["states"][length]["__umpire_facts"] { continue; }
                let mut forged: serde_json::Value = serde_json::from_str(raw)?;
                forged["states"][length]["__umpire_facts"] = replacement;
                assert!(replay.violation(&module, &property, length, &forged.to_string()).is_err());
            }
            let mut missing: serde_json::Value = serde_json::from_str(raw)?;
            missing["states"][length].as_object_mut().ok_or("not an object")?.remove("__umpire_facts");
            assert!(replay.violation(&module, &property, length, &missing.to_string()).is_err());
        }
    }
    let mutant = Tla::safety(&Lift::parse("facts-mutant.rs", &source.replacen("Some(Fact::A)", "Some(Fact::B)", 1))?.bind(&binding)?)?;
    let result = tool.check(&mutant, "Invariant_never_a", 1)?;
    assert!(result.violated()?);
    assert!(!result.traces().is_empty());
    for raw in result.traces() {
        let error = replay.violation(&mutant, "Invariant_never_a", 1, raw).err().ok_or("source fact mutant replayed")?;
        assert!(error.to_string().contains("facts"), "{error}");
    }
    Ok(())
}
