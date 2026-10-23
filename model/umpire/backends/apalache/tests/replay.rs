//! Counterexamples are executed from retained starts, without a reached graph.
use umpire_apalache::Replay;
use umpire_dsl::{Claim, Machine};
#[test]
fn replay_refuses_an_undeclared_property_before_reading_a_witness()
-> Result<(), Box<dyn std::error::Error>> {
    let world = authored::inputs::World::new(2);
    let starts = world.starts();
    let catalog = world.checked_catalog(umpire_dsl::CATALOG_LIMIT)?;
    let module = module()?;
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

#[path = "fixtures/replay.rs"]
mod authored;
use serde_json::{Value as Json, json};
use umpire_dsl::{ActionValues, Value as NativeValue, Values};
use umpire_lift::{Binding, Concrete, Literal, Module};
const SOURCE: &str = include_str!("fixtures/replay.rs");
const HOOK: &str = r#"
umpire_testing::fixture!(inputs::World::new(2),100,Claim::new("never_two",|_,s:&inputs::Model|(s.value!=2).into()));
"#;
fn literal(v: NativeValue) -> Literal {
    match v {
        NativeValue::Bool(v) => Literal::Bool(v),
        NativeValue::Int(v) => Literal::Int(v),
        NativeValue::Enum(v) => Literal::Enum(v.into()),
        NativeValue::Sequence(v) => Literal::Sequence(v.into_iter().map(literal).collect()),
    }
}
fn exported(v: Vec<(&'static str, NativeValue)>) -> Vec<(String, Literal)> {
    v.into_iter().map(|(n, v)| (n.into(), literal(v))).collect()
}
fn concrete<M: Machine>(a: &M::Action) -> Concrete
where
    M::Action: ActionValues<M>,
{
    let (action, fields) = ActionValues::<M>::action_value(a);
    Concrete { action: action.into(), fields: exported(fields) }
}
fn module() -> Result<Module, Box<dyn std::error::Error>> {
    let w = authored::inputs::World::new(2);
    Ok(umpire_lift::Tla::safety(
        &umpire_lift::Lift::parse("inputs.rs", &format!("{SOURCE}{HOOK}"))?.bind(&Binding {
            world: exported(w.values()),
            starts: w.starts().iter().map(|s| exported(s.values())).collect(),
            actions: w.actions().iter().map(concrete::<authored::inputs::World>).collect(),
        })?,
    )?)
}
fn default(ty: &umpire_lift::Type) -> Json {
    match ty {
        umpire_lift::Type::Bool => json!(false),
        umpire_lift::Type::Int { .. } => json!({"#bigint":"0"}),
        umpire_lift::Type::Text => json!(""),
        umpire_lift::Type::Enum { variants, .. } => json!(variants[0]),
        umpire_lift::Type::Seq(_) | umpire_lift::Type::Unit => json!([]),
        umpire_lift::Type::Tuple(v) => json!({"#tup":v.iter().map(default).collect::<Vec<_>>()}),
    }
}
fn initial(module: &Module, value: u8) -> Json {
    let mut row = serde_json::Map::new();
    row.insert("#meta".into(), json!({"index":0}));
    row.insert("value".into(), json!({"#bigint":value.to_string()}));
    for (name, ty) in &module.metadata.observer_types {
        row.insert(name.clone(), default(ty));
    }
    row.insert("__umpire_stutter".into(), json!(true));
    for p in &module.metadata.parameters {
        row.insert(p.symbol.clone(), json!({"#bigint":"2"}));
    }
    Json::Object(row)
}
fn event(previous: &Json, value: u8, ordinal: usize, action: &str, outcome: &str) -> Json {
    let mut row = previous.clone();
    row["#meta"]["index"] = json!(previous["#meta"]["index"].as_u64().unwrap_or_default() + 1);
    row["__umpire_before_value"] = previous["value"].clone();
    row["value"] = json!({"#bigint":value.to_string()});
    row["__umpire_has_step"] = json!(true);
    row["__umpire_stutter"] = json!(false);
    row["__umpire_action"] = json!(action);
    row["__umpire_input"] = json!({"#bigint":ordinal.to_string()});
    row["__umpire_outcome"] = json!(outcome);
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
    stutter["__umpire_stutter"] = json!(true);
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
    duplicate_ordinal["states"][1]["__umpire_input"] = json!({"#bigint":"1"});
    duplicate_ordinal["states"][2]["__umpire_input"] = json!({"#bigint":"1"});
    assert!(
        replay.violation(&module, "Negative_never_two", 3, &duplicate_ordinal.to_string()).is_err(),
        "accepted a noncanonical duplicate ordinal"
    );
    for (position, field, bad) in [
        (0, "value", json!({"#bigint":"9"})),
        (0, "__umpire_before_value", json!({"#bigint":"1"})),
        (0, "__umpire_has_step", json!(true)),
        (1, "__umpire_world_limit", json!({"#bigint":"3"})),
        (1, "__umpire_action", json!("User::Reset")),
        (1, "__umpire_input", json!({"#bigint":"2"})),
        (1, "__umpire_choice", json!({"#bigint":"1"})),
        (1, "__umpire_outcome", json!("Outcome::Accepted")),
        (1, "__umpire_facts", json!([[]])),
        (1, "__umpire_before_value", json!({"#bigint":"0"})),
        (1, "__umpire_has_step", json!(false)),
        (1, "__umpire_stutter", json!(true)),
        (2, "__umpire_action", json!("User::Reset")),
        (2, "__umpire_before_value", json!({"#bigint":"0"})),
        (2, "value", json!({"#bigint":"2"})),
        (3, "value", json!({"#bigint":"1"})),
        (3, "__umpire_input", json!({"#bigint":"0"})),
        (3, "__umpire_arithmetic_error", json!(true)),
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

#[path = "fixtures/arithmetic_prefix.rs"]
mod prefix_fixture;
#[test]
fn arithmetic_refusal_is_distinct_and_requires_the_exact_native_boundary()
-> Result<(), Box<dyn std::error::Error>> {
    let world = prefix_fixture::prefix::World::new();
    let starts = world.starts();
    let actions = world.actions();
    let module = umpire_lift::Tla::safety(
        &umpire_lift::Lift::parse("prefix.rs", include_str!("fixtures/arithmetic_prefix.rs"))?
            .bind(&Binding {
                world: vec![],
                starts: vec![vec![("value".into(), Literal::Int(254))]],
                actions: vec![Concrete { action: "User::Advance".into(), fields: vec![] }],
            })?,
    )?;
    let first = initial(&module, 254);
    let mut next = event(&first, 255, 0, "User::Advance", "");
    next["__umpire_outcome"] = json!([]);
    let mut failure = event(&next, 255, 0, "User::Advance", "");
    failure["__umpire_outcome"] = json!([]);
    failure["__umpire_arithmetic_error"] = json!(true);
    let raw = witness(&module, vec![first, next, failure]);
    let replay = Replay { world: &world, starts: &starts, actions: &actions, negative: &[] };
    let confirmed = replay.violation(&module, "ArithmeticSafe", 2, &raw.to_string())?;
    assert!(matches!(
        confirmed.confirmation,
        umpire_apalache::Confirmation::Arithmetic { index: 2, .. }
    ));
    assert_eq!(confirmed.steps.len(), 1);
    assert_eq!(confirmed.steps[0].state["value"], umpire_tlc::Value::Int(255));
    for (position, field, bad) in [
        (1, "__umpire_arithmetic_error", json!(true)),
        (2, "__umpire_arithmetic_error", json!(false)),
        (2, "value", json!({"#bigint":"0"})),
        (2, "__umpire_action", json!("User::Other")),
        (2, "__umpire_stutter", json!(true)),
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
