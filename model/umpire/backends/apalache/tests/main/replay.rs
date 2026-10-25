//! Counterexamples are executed from retained starts, without a reached graph.
#![forbid(unsafe_code)]

use crate::selected_kettle as selected;
use umpire_apalache::Replay;
use umpire_dsl::{Claim, Machine};
#[test]
fn replay_refuses_an_undeclared_property_before_reading_a_witness()
-> Result<(), Box<dyn std::error::Error>> {
    let world = selected::kettle::world();
    let starts = world.starts();
    let catalog = world.checked_catalog(umpire_dsl::CATALOG_LIMIT)?;
    let module = umpire_tla::Tla::safety(&umpire_tla::Lift::parse(
        "kettle.rs",
        include_str!("../../../../dev/fixtures/fixtures/sample_kettle/model.rs"),
    )?)?;
    let negatives: Vec<Claim<_>> = Vec::new();
    let replay =
        Replay { world: &world, starts: &starts, actions: catalog.actions(), negative: &negatives };
    let error = replay
        .violation(&module, "invented", 0, "{}")
        .err()
        .ok_or("accepted an undeclared property")?;
    assert!(error.to_string().contains("property"), "{error}");
    Ok(())
}

use crate::replay_fixture as authored;
use serde_json::{Value as Json, json};
use umpire_dsl::{__lifted::BindingOf, ActionValues, Value as NativeValue, Values};
use umpire_tla::{Binding, Literal, Module};
const SOURCE: &str = include_str!("../fixtures/replay.rs");
const HOOK: &str = r#"
umpire_testing::fixture!(inputs::World::new(2),100,Claim::new("never_two",|_,s:&inputs::Model|(s.value!=2).into()));
"#;
fn module() -> Result<Module, Box<dyn std::error::Error>> {
    let w = authored::inputs::World::new(2);
    Ok(umpire_tla::Tla::safety(
        &umpire_tla::Lift::parse("inputs.rs", &format!("{SOURCE}{HOOK}"))?.bind(&Binding::of(
            &w,
            &w.starts(),
            &w.actions(),
        ))?,
    )?)
}
fn default(ty: &umpire_tla::Type) -> Json {
    match ty {
        umpire_tla::Type::Bool => json!(false),
        umpire_tla::Type::Int { .. } | umpire_tla::Type::Optional(_) => json!({"#bigint":"0"}),
        umpire_tla::Type::Bounded(b) => json!({"#bigint": b.low.to_string()}),
        umpire_tla::Type::Text => json!(""),
        umpire_tla::Type::Enum { variants, .. } => json!(variants[0]),
        umpire_tla::Type::Seq(_) | umpire_tla::Type::Unit => json!([]),
        umpire_tla::Type::Tuple(v) => json!({"#tup":v.iter().map(default).collect::<Vec<_>>()}),
    }
}
fn initial(module: &Module, value: u8) -> Json {
    let mut row = serde_json::Map::new();
    row.insert("#meta".into(), json!({"index":0}));
    row.insert("value".into(), json!({"#bigint":value.to_string()}));
    for (name, ty) in &module.metadata.observer_types {
        row.insert(name.clone(), default(ty));
    }
    row.insert(umpire_tla::observers::STUTTER.into(), json!(true));
    for p in &module.metadata.parameters {
        row.insert(p.symbol.clone(), json!({"#bigint":"2"}));
    }
    Json::Object(row)
}
fn event(previous: &Json, value: u8, ordinal: usize, action: &str, outcome: &str) -> Json {
    let mut row = previous.clone();
    row["#meta"]["index"] = json!(previous["#meta"]["index"].as_u64().unwrap_or_default() + 1);
    row[umpire_tla::observers::before("value")] = previous["value"].clone();
    row["value"] = json!({"#bigint":value.to_string()});
    row[umpire_tla::observers::HAS_STEP] = json!(true);
    row[umpire_tla::observers::STUTTER] = json!(false);
    row[umpire_tla::observers::ACTION] = json!(action);
    row[umpire_tla::observers::INPUT] = json!({"#bigint":ordinal.to_string()});
    row[umpire_tla::observers::OUTCOME] = json!(outcome);
    row
}
fn witness(module: &Module, states: Vec<Json>) -> Json {
    let mut root = json!({
        "#meta":{"format":"ITF"},
        "vars":module.metadata.fields.iter().map(|f|f.name.clone()).chain(module.metadata.observer_fields.iter().cloned()).collect::<Vec<_>>(),
        "params":module.metadata.parameters.iter().map(|p|p.symbol.clone()).collect::<Vec<_>>(),
    });
    root["states"] = Json::Array(states);
    root
}

#[test]
fn exact_payload_outcome_history_and_specific_claim_are_replayed()
-> Result<(), Box<dyn std::error::Error>> {
    let world = authored::inputs::World::new(2);
    let starts = world.starts();
    let actions = world.actions();
    let module = module()?;
    let negatives =
        [Claim::new("never_two", |_, s: &authored::inputs::Model| (s.value != 2).into())];
    let replay = Replay { world: &world, starts: &starts, actions: &actions, negative: &negatives };
    let first = initial(&module, 1);
    let rejected = event(&first, 1, 0, "User::Set", "Outcome::Rejected(NotFound)");
    let mut stutter = rejected.clone();
    stutter["#meta"]["index"] = json!(2);
    stutter[umpire_tla::observers::STUTTER] = json!(true);
    let final_state = event(&stutter, 2, 2, "User::Set", "Outcome::Accepted");
    let raw = witness(&module, vec![first, rejected, stutter, final_state]);
    let verified = replay.violation(&module, "Negative_never_two", 3, &raw.to_string())?;
    assert_eq!(verified.confirmation, umpire_apalache::Confirmation::Violation);
    assert!(
        verified.path.contains("input=0 choice=0 outcome=Text(\"Outcome::Rejected(NotFound)\")")
    );
    assert!(verified.path.contains("input=2"));
    assert_eq!(verified.steps.len(), 2);
    assert_eq!(verified.steps[0].action.fields[0].1, Literal::Int(1));
    assert_eq!(verified.steps[1].action.fields[0].1, Literal::Int(2));
    let mut duplicate_ordinal = raw.clone();
    duplicate_ordinal["states"][1][umpire_tla::observers::INPUT] = json!({"#bigint":"1"});
    duplicate_ordinal["states"][2][umpire_tla::observers::INPUT] = json!({"#bigint":"1"});
    assert!(
        replay.violation(&module, "Negative_never_two", 3, &duplicate_ordinal.to_string()).is_err(),
        "accepted a noncanonical duplicate ordinal"
    );
    for (position, field, bad) in [
        (0, "value", json!({"#bigint":"9"})),
        (0, &umpire_tla::observers::before("value"), json!({"#bigint":"1"})),
        (0, umpire_tla::observers::HAS_STEP, json!(true)),
        (1, "__umpire_world_limit", json!({"#bigint":"3"})),
        (1, umpire_tla::observers::ACTION, json!("User::Reset")),
        (1, umpire_tla::observers::INPUT, json!({"#bigint":"2"})),
        (1, umpire_tla::observers::CHOICE, json!({"#bigint":"1"})),
        (1, umpire_tla::observers::OUTCOME, json!("Outcome::Accepted")),
        (1, umpire_tla::observers::FACTS, json!([[]])),
        (1, &umpire_tla::observers::before("value"), json!({"#bigint":"0"})),
        (1, umpire_tla::observers::HAS_STEP, json!(false)),
        (1, umpire_tla::observers::STUTTER, json!(true)),
        (2, umpire_tla::observers::ACTION, json!("User::Reset")),
        (2, &umpire_tla::observers::before("value"), json!({"#bigint":"0"})),
        (2, "value", json!({"#bigint":"2"})),
        (3, "value", json!({"#bigint":"1"})),
        (3, umpire_tla::observers::INPUT, json!({"#bigint":"0"})),
        (3, umpire_tla::observers::ERROR, json!(true)),
    ] {
        let mut tampered = raw.clone();
        tampered["states"][position][field] = bad;
        assert!(
            replay.violation(&module, "Negative_never_two", 3, &tampered.to_string()).is_err(),
            "accepted {position}/{field}"
        );
    }
    for property in
        ["Invariant_bounded", "Post_no_reset", "ArithmeticSafe", "ValidStep", "Negative_other"]
    {
        assert!(
            replay.violation(&module, property, 3, &raw.to_string()).is_err(),
            "accepted unrelated property {property}"
        );
    }
    assert!(replay.violation(&module, "Negative_never_two", 2, &raw.to_string()).is_err());
    let wrong = authored::inputs::World::new(3);
    assert!(
        Replay { world: &wrong, starts: &starts, actions: &actions, negative: &negatives }
            .violation(&module, "Negative_never_two", 3, &raw.to_string())
            .is_err()
    );
    let mut reversed = actions.clone();
    reversed.swap(0, 2);
    assert!(
        Replay { world: &world, starts: &starts, actions: &reversed, negative: &negatives }
            .violation(&module, "Negative_never_two", 3, &raw.to_string())
            .is_err()
    );
    assert!(
        Replay { world: &world, starts: &starts[..1], actions: &actions, negative: &negatives }
            .violation(&module, "Negative_never_two", 3, &raw.to_string())
            .is_err()
    );
    let first = initial(&module, 1);
    let reset = event(&first, 0, 3, "User::Reset", "Outcome::Accepted");
    replay.violation(
        &module,
        "Post_no_reset",
        1,
        &witness(&module, vec![first, reset]).to_string(),
    )?;
    Ok(())
}

#[test]
fn retained_pinned_kettle_witness_replays_deterministically()
-> Result<(), Box<dyn std::error::Error>> {
    let world = selected::kettle::world();
    let starts = world.starts();
    let actions = world.actions();
    let source = format!(
        r#"{}
        umpire_testing::fixture!(kettle::world(),1000,Claim::new("never_beeps",|_,s:&kettle::Model|(!s.beeping).into()));
    "#,
        include_str!("../../../../dev/fixtures/fixtures/sample_kettle/model.rs")
    );
    let module = umpire_tla::Tla::safety(&umpire_tla::Lift::parse("kettle.rs", &source)?)?;
    let negatives =
        [Claim::new("never_beeps", |_, s: &selected::kettle::Model| (!s.beeping).into())];
    let replay = Replay { world: &world, starts: &starts, actions: &actions, negative: &negatives };
    let result = replay.violation(
        &module,
        "Negative_never_beeps",
        63,
        include_str!("../fixtures/kettle-0.62.4.itf.json"),
    )?;
    assert_eq!(result.to_string(), include_str!("../expected/kettle-negative.txt"));
    Ok(())
}

#[derive(Clone, PartialEq, Eq, Hash)]
struct Colliding(authored::inputs::Model);
impl std::fmt::Display for Colliding {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str("same state")
    }
}
impl Values for Colliding {
    fn values(&self) -> Vec<(&'static str, NativeValue)> {
        self.0.values()
    }
}
struct Retained {
    world: authored::inputs::World,
    calls: std::cell::Cell<usize>,
}
impl Values for Retained {
    fn values(&self) -> Vec<(&'static str, NativeValue)> {
        self.world.values()
    }
}
#[derive(Clone, PartialEq, Eq, Hash)]
struct CollidingAction(authored::inputs::Action);
impl std::fmt::Display for CollidingAction {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str("same action")
    }
}
impl ActionValues<Retained> for CollidingAction {
    fn action_value(&self) -> (&'static str, Vec<(&'static str, NativeValue)>) {
        ActionValues::<authored::inputs::World>::action_value(&self.0)
    }
    fn domains(w: &Retained) -> Vec<(&'static str, &'static str, Vec<NativeValue>)> {
        <authored::inputs::Action as ActionValues<authored::inputs::World>>::domains(&w.world)
    }
}
impl Machine for Retained {
    type State = Colliding;
    type Action = CollidingAction;
    type Outcome = umpire_dsl::Outcome;
    type Fact = ();
    fn starts(&self) -> Vec<Self::State> {
        panic!("retained starts must not be enumerated during replay")
    }
    fn actions(&self) -> Vec<Self::Action> {
        panic!("retained catalog must not be enumerated during replay")
    }
    fn internal(&self, a: &Self::Action) -> bool {
        self.world.internal(&a.0)
    }
    fn step(
        &self,
        s: &Self::State,
        a: Self::Action,
    ) -> umpire_dsl::Steps<Self::State, Self::Outcome, Self::Fact> {
        self.calls.set(self.calls.get() + 1);
        self.world.step(&s.0, a.0).into_iter().map(|(o, f, s)| (o, f, Colliding(s))).collect()
    }
    fn checked_step(
        &self,
        s: &Self::State,
        a: Self::Action,
    ) -> Result<umpire_dsl::Steps<Self::State, Self::Outcome, Self::Fact>, umpire_dsl::StepError>
    {
        self.calls.set(self.calls.get() + 1);
        Ok(self
            .world
            .checked_step(&s.0, a.0)?
            .into_iter()
            .map(|(o, f, s)| (o, f, Colliding(s)))
            .collect())
    }
    fn invariants(&self) -> Vec<Claim<Self>> {
        vec![Claim::<Self>::new("bounded", |w, s| {
            w.world.invariants()[0].evaluate(&w.world, &s.0).verdict
        })]
    }
    fn postconditions(&self) -> Vec<umpire_dsl::StepClaim<Self>> {
        vec![
            umpire_dsl::StepClaim::<Self>::after(
                "no_reset",
                CollidingAction(authored::inputs::User::Reset.into()),
                |w, b, s, a, o, f| {
                    w.world.postconditions()[0].holds(
                        &w.world,
                        &b.0,
                        &a.0,
                        &(*o, f.to_vec(), s.0.clone()),
                    )
                },
            ),
            umpire_dsl::StepClaim::<Self>::new("no_rejection", |w, b, s, a, o, f| {
                w.world.postconditions()[1].holds(
                    &w.world,
                    &b.0,
                    &a.0,
                    &(*o, f.to_vec(), s.0.clone()),
                )
            }),
        ]
    }
}
#[test]
fn colliding_renderings_cannot_replace_typed_replay_or_reenumerate_native_inputs()
-> Result<(), Box<dyn std::error::Error>> {
    let world = Retained { world: authored::inputs::World::new(2), calls: std::cell::Cell::new(0) };
    let starts = world.world.starts().into_iter().map(Colliding).collect::<Vec<_>>();
    assert_eq!(starts[0].to_string(), starts[1].to_string());
    let actions = world.world.actions().into_iter().map(CollidingAction).collect::<Vec<_>>();
    assert_eq!(actions[0].to_string(), actions[2].to_string());
    let module = module()?;
    let negatives = [Claim::new("never_two", |_, s: &Colliding| (s.0.value != 2).into())];
    let replay = Replay { world: &world, starts: &starts, actions: &actions, negative: &negatives };
    let first = initial(&module, 1);
    let after = event(&first, 2, 2, "User::Set", "Outcome::Accepted");
    let mut raw = witness(&module, vec![first, after]);
    replay.violation(&module, "Negative_never_two", 1, &raw.to_string())?;
    assert_eq!(world.calls.get(), 1);
    raw["states"][1]["value"] = json!({"#bigint":"1"});
    assert!(replay.violation(&module, "Negative_never_two", 1, &raw.to_string()).is_err());
    assert_eq!(world.calls.get(), 2);
    Ok(())
}

use crate::prefix_fixture;
#[test]
fn arithmetic_refusal_is_distinct_and_requires_the_exact_native_boundary()
-> Result<(), Box<dyn std::error::Error>> {
    let world = prefix_fixture::prefix::World;
    let starts = world.starts();
    let actions = world.actions();
    let module = umpire_tla::Tla::safety(&umpire_tla::Lift::parse(
        "prefix.rs",
        include_str!("../fixtures/arithmetic_prefix.rs"),
    )?)?;
    let first = initial(&module, 254);
    let mut next = event(&first, 255, 0, "User::Advance", "");
    next[umpire_tla::observers::OUTCOME] = json!([]);
    let mut failure = event(&next, 255, 0, "User::Advance", "");
    failure[umpire_tla::observers::OUTCOME] = json!([]);
    failure[umpire_tla::observers::ERROR] = json!(true);
    let raw = witness(&module, vec![first, next, failure]);
    let replay = Replay { world: &world, starts: &starts, actions: &actions, negative: &[] };
    let confirmed = replay.violation(&module, "ArithmeticSafe", 2, &raw.to_string())?;
    assert!(matches!(
        confirmed.confirmation,
        umpire_apalache::Confirmation::Arithmetic { index: 2, .. }
    ));
    assert_eq!(confirmed.steps.len(), 1);
    assert_eq!(confirmed.steps[0].state["value"], umpire_tla::values::Value::Int(255));
    for (position, field, bad) in [
        (1, umpire_tla::observers::ERROR, json!(true)),
        (2, umpire_tla::observers::ERROR, json!(false)),
        (2, "value", json!({"#bigint":"0"})),
        (2, umpire_tla::observers::ACTION, json!("User::Other")),
        (2, umpire_tla::observers::STUTTER, json!(true)),
    ] {
        let mut tampered = raw.clone();
        tampered["states"][position][field] = bad;
        assert!(
            replay.violation(&module, "ArithmeticSafe", 2, &tampered.to_string()).is_err(),
            "accepted {position}/{field}"
        );
    }
    Ok(())
}

#[path = "../fixtures/categories.rs"]
mod category_fixture;
#[test]
fn selected_property_category_cannot_alias_authored_names() -> Result<(), Box<dyn std::error::Error>>
{
    use category_fixture::categories;
    let world = categories::World;
    let starts = world.starts();
    let actions = world.actions();
    let source = format!(
        "{}\numpire_testing::fixture!(categories::world(),10,Claim::new(\"shared\",|_,s:&categories::Model|(s.value!=0).into()));",
        include_str!("../fixtures/categories.rs")
    );
    let module = umpire_tla::Tla::safety(&umpire_tla::Lift::parse("categories.rs", &source)?)?;
    let negative = [Claim::new("shared", |_, s: &categories::Model| (s.value != 0).into())];
    let replay = Replay { world: &world, starts: &starts, actions: &actions, negative: &negative };
    for (property, value, accepted) in [
        ("ArithmeticSafe", 0, false),
        ("ValidStep", 0, false),
        ("Invariant_arithmetic", 0, true),
        ("Invariant_valid_step", 0, true),
        ("Invariant_shared", 0, false),
        ("Negative_shared", 0, true),
        ("Invariant_shared", 1, true),
        ("Negative_shared", 1, false),
    ] {
        let raw = witness(&module, vec![initial(&module, value)]).to_string();
        assert_eq!(
            replay.violation(&module, property, 0, &raw).is_ok(),
            accepted,
            "{property} value={value}"
        );
    }
    Ok(())
}
