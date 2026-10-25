use super::*;
use crate::Files;
use umpire_tla::Literal;

fn bounded(name: &str, field: Option<&str>, low: i128) -> umpire_dsl::__lifted::Bounded {
    umpire_dsl::__lifted::Bounded {
        name: name.into(),
        field: field.map(str::to_owned),
        backing: "u8".into(),
        min: 0,
        max: 255,
        low,
        high: 3,
    }
}

#[test]
fn nominal_debug_preserves_authored_shapes_and_exact_optional_codes() -> crate::TestResult {
    let ticks = bounded("Ticks", None, 1);
    let remaining = bounded("Remaining", Some("ticks"), 0);
    for (ty, code, expected) in [
        (Type::Bounded(ticks.clone()), 1, "Ticks(1)"),
        (Type::Optional(ticks.clone()), 0, "None"),
        (Type::Optional(ticks.clone()), 2, "Some(Ticks(1))"),
        (Type::Optional(remaining.clone()), 1, "Some(Remaining { ticks: 0 })"),
    ] {
        assert_eq!(debug_value(&ty, &Value::Int(code))?, expected);
    }
    for value in [
        Value::Int(-1),
        Value::Int(1),
        Value::Int(5),
        Value::Bool(false),
        Value::Text("Some(Ticks(1))".into()),
        Value::Seq(Vec::new()),
    ] {
        assert!(debug_value(&Type::Optional(ticks.clone()), &value).is_err(), "{value:?}");
    }
    assert!(debug_value(&Type::Bounded(ticks.clone()), &Value::Int(0)).is_err());
    let mut malformed = remaining;
    malformed.max = 4;
    assert!(debug_value(&Type::Optional(malformed), &Value::Int(0)).is_err());
    let mut module = module(Vec::new(), Vec::new());
    module.metadata.fields[0].ty = Type::Optional(ticks);
    let field = module.metadata.fields[0].name.clone();
    assert!(state_text(&module, &State::from([(field, Value::Int(1))])).is_err());
    Ok(())
}

struct Invalid;

impl Machine for Invalid {
    type State = bool;
    type Action = u8;
    type Outcome = ();
    type Fact = ();

    fn starts(&self) -> Vec<bool> {
        vec![false]
    }
    fn actions(&self) -> Vec<u8> {
        vec![1]
    }
    fn internal(&self, _: &u8) -> bool {
        false
    }
    fn step(&self, _: &bool, _: u8) -> umpire_dsl::Steps<bool, (), ()> {
        panic!("unchecked invalid Model step")
    }
    fn checked_step(
        &self,
        _: &bool,
        _: u8,
    ) -> Result<umpire_dsl::Steps<bool, (), ()>, umpire_dsl::StepError> {
        Err(umpire_dsl::StepError::RejectedAfterMutation)
    }
}

#[test]
fn replay_uses_checked_steps_without_panicking() -> crate::TestResult {
    let result = std::panic::catch_unwind(|| native_steps(&Invalid, &false, 1));
    assert_eq!(
        result.map_err(|_| "backend replay bypassed checked_step")?,
        Err(umpire_checker::Refusal::Step {
            action: "1".into(),
            before: "false".into(),
            error: umpire_dsl::StepError::RejectedAfterMutation,
        })
    );
    Ok(())
}

struct RetainedStarts;

impl Machine for RetainedStarts {
    type State = String;
    type Action = u8;
    type Outcome = ();
    type Fact = ();

    fn starts(&self) -> Vec<String> {
        panic!("authored starts must not be enumerated during replay")
    }
    fn actions(&self) -> Vec<u8> {
        Vec::new()
    }
    fn internal(&self, _: &u8) -> bool {
        false
    }
    fn step(&self, _: &String, _: u8) -> umpire_dsl::Steps<String, (), ()> {
        umpire_dsl::Steps::new()
    }
}

fn module(actions: Vec<umpire_tla::Action>, catalog: Vec<Concrete>) -> Module {
    Module {
        name: "RetainedStarts".into(),
        tla: String::new(),
        cfg: String::new(),
        metadata: umpire_tla::Metadata {
            arithmetic: Vec::new(),
            state_name: "Model".into(),
            world_path: String::new(),
            parameters: Vec::new(),
            binding: None,
            fields: vec![umpire_tla::Field { name: "value".into(), ty: Type::Bool }],
            actions,
            catalog,
            invariants: Vec::new(),
            postconditions: Vec::new(),
            progress: Vec::new(),
            negative: Vec::new(),
            observer_fields: Vec::new(),
            observer_types: BTreeMap::new(),
            outcome_type: Type::Unit,
        },
    }
}

#[test]
fn replay_uses_retained_starts_without_enumerating_again() -> crate::TestResult {
    let module = module(Vec::new(), Vec::new());
    let starts = ["Model { value: false }".to_owned()];
    let native = starts.iter().map(|state| (state.clone(), state)).collect();
    let trace = umpire_tlc::Trace {
        states: vec![umpire_tlc::TraceState {
            state: State::from([("value".into(), Value::Bool(false))]),
            observers: State::new(),
        }],
        back_to: None,
    };
    std::panic::catch_unwind(|| replay(&RetainedStarts, &starts, &module, &native, &trace, None))
        .map_err(|_| "backend replay re-enumerated authored starts")??;
    let error = replay(&RetainedStarts, &[], &module, &native, &trace, None)
        .err()
        .ok_or("replay accepted a counterexample outside the retained starts")?;
    assert!(error.to_string().contains("does not begin at a Model start"), "{error}");
    Ok(())
}

#[derive(Clone, Debug, PartialEq, Eq, Hash)]
struct Polled(u8);

impl std::fmt::Display for Polled {
    fn fmt(&self, out: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(out, "Model {{ value: {} }}", self.0)
    }
}

/// `Poll(1)` advances from 0; `Poll(2)` is never enabled.
struct Polls;

impl Machine for Polls {
    type State = Polled;
    type Action = u8;
    type Outcome = ();
    type Fact = ();

    fn starts(&self) -> Vec<Polled> {
        vec![Polled(0)]
    }
    fn actions(&self) -> Vec<u8> {
        vec![1, 2]
    }
    fn internal(&self, _: &u8) -> bool {
        false
    }
    fn step(&self, state: &Polled, action: u8) -> umpire_dsl::Steps<Polled, (), ()> {
        let mut steps = umpire_dsl::Steps::new();
        if state.0 == 0 && action == 1 {
            steps.push(((), Vec::new(), Polled(1)));
        }
        steps
    }
}

fn poll(value: i128) -> Concrete {
    Concrete { action: "Poll".into(), fields: vec![("0".into(), Literal::Int(value))] }
}

fn polled(value: i128, tick: bool, action: &str, input: i128) -> umpire_tlc::TraceState {
    umpire_tlc::TraceState {
        state: State::from([("value".into(), Value::Int(value))]),
        observers: State::from([
            ("__umpire_edge_tick".into(), Value::Bool(tick)),
            ("__umpire_action".into(), Value::Text(action.into())),
            ("__umpire_choice".into(), Value::Int(0)),
            ("__umpire_outcome".into(), Value::Seq(Vec::new())),
            ("__umpire_facts".into(), Value::Seq(Vec::new())),
            (INPUT.into(), Value::Int(input)),
        ]),
    }
}

#[test]
fn typed_replay_identifies_the_exact_catalog_payload() -> crate::TestResult {
    let actions = vec![umpire_tla::Action {
        name: "Poll".into(),
        operator: "Poll".into(),
        internal: false,
        fields: vec![umpire_tla::Field { name: "0".into(), ty: Type::Int { min: 0, max: 3 } }],
    }];
    let mut valid = module(actions, vec![poll(1), poll(2)]);
    valid.metadata.fields[0].ty = Type::Int { min: 0, max: 1 };
    valid.metadata.observer_types.insert("__umpire_facts".into(), Type::Seq(Box::new(Type::Unit)));
    let starts = [Polled(0)];
    let native = BTreeMap::from([("Model { value: 0 }".to_owned(), &starts[0])]);
    let identity = |action: &u8| poll(i128::from(*action));
    let trace = |after: umpire_tlc::TraceState| umpire_tlc::Trace {
        states: vec![polled(0, false, "", 0), after],
        back_to: None,
    };
    let check = |module: &Module, after, identity: Option<&dyn Fn(&u8) -> Concrete>| {
        replay(&Polls, &starts, module, &native, &trace(after), identity)
    };
    check(&valid, polled(1, true, "Poll", 0), Some(&identity))?;
    let mut altered = valid.clone();
    altered.metadata.catalog[0] = poll(2);
    let mut untyped = valid.clone();
    untyped.metadata.catalog[0] = poll(9);
    let mut choice = polled(1, true, "Poll", 0);
    choice.observers.insert("__umpire_choice".into(), Value::Int(1));
    let mut unidentified = polled(1, true, "Poll", 0);
    unidentified.observers.remove(INPUT);
    for (module, after, typed, reason) in [
        (&valid, polled(1, true, "Poll", 2), true, "outside the catalog"),
        (&valid, polled(1, true, "Push", 0), true, "not the edge's action Push"),
        (&valid, polled(1, true, "Poll", 1), true, "choice is not emitted"),
        (&valid, polled(1, true, "Poll", 0), false, "requires the typed native identity"),
        (&valid, choice, true, "choice is not emitted"),
        (&valid, unidentified, true, "lacks input identity"),
        (&altered, polled(1, true, "Poll", 0), true, "differs from the TLC input"),
        (&untyped, polled(1, true, "Poll", 0), true, "does not match its declared fields"),
    ] {
        let identity: Option<&dyn Fn(&u8) -> Concrete> = typed.then_some(&identity);
        let error = check(module, after, identity).err().ok_or(reason)?;
        assert!(error.to_string().contains(reason), "{reason}: {error}");
    }
    let before = polled(1, true, "Poll", 0);
    let error = validate_projection(&valid, &before, &polled(1, true, "Poll", 1))
        .err()
        .ok_or("observer-only step changed the input identity")?;
    assert!(error.to_string().contains(INPUT), "{error}");
    Ok(())
}

/// Action `1` is a rejected self-loop from `0`.
struct Rejecting;

impl Machine for Rejecting {
    type State = Polled;
    type Action = u8;
    type Outcome = umpire_dsl::Outcome;
    type Fact = ();

    fn starts(&self) -> Vec<Polled> {
        vec![Polled(0)]
    }
    fn actions(&self) -> Vec<u8> {
        vec![1]
    }
    fn internal(&self, _: &u8) -> bool {
        false
    }
    fn step(&self, state: &Polled, _: u8) -> umpire_dsl::Steps<Polled, Self::Outcome, ()> {
        let rejected = umpire_dsl::Outcome::Rejected(umpire_dsl::Rejection::NotFound);
        umpire_dsl::Steps::from_iter([(rejected, Vec::new(), state.clone())])
    }
}

#[test]
fn replay_compares_the_typed_outcome_of_a_rejected_self_loop() -> crate::TestResult {
    let mut module = module(Vec::new(), Vec::new());
    module.metadata.fields[0].ty = Type::Int { min: 0, max: 1 };
    module.metadata.outcome_type = Type::outcome();
    module.metadata.observer_types.insert("__umpire_facts".into(), Type::Seq(Box::new(Type::Unit)));
    let starts = [Polled(0)];
    let native = BTreeMap::from([("Model { value: 0 }".to_owned(), &starts[0])]);
    let state = |tick: bool, outcome: Value| umpire_tlc::TraceState {
        state: State::from([("value".into(), Value::Int(0))]),
        observers: State::from([
            ("__umpire_edge_tick".into(), Value::Bool(tick)),
            ("__umpire_action".into(), Value::Text(if tick { "1" } else { "" }.into())),
            ("__umpire_choice".into(), Value::Int(0)),
            ("__umpire_outcome".into(), outcome),
            ("__umpire_facts".into(), Value::Seq(Vec::new())),
        ]),
    };
    let replayed = |outcome: &str| {
        let accepted = Value::Text("Outcome::Accepted".into());
        let trace = umpire_tlc::Trace {
            states: vec![state(false, accepted), state(true, Value::Text(outcome.into()))],
            back_to: None,
        };
        replay(&Rejecting, &starts, &module, &native, &trace, None)
    };
    replayed("Outcome::Rejected(NotFound)")?;
    for (outcome, reason) in [
        ("Outcome::Accepted", "outcome/facts differ"),
        ("Outcome::Rejected(Unavailable)", "outcome/facts differ"),
        ("Rejected(NotFound)", "does not match authored type"),
    ] {
        let error = replayed(outcome).err().ok_or(outcome)?;
        assert!(error.to_string().contains(reason), "{outcome}: {error}");
    }
    Ok(())
}

#[derive(Debug)]
enum Fact {
    A,
    B,
    C,
    D,
}

#[derive(Clone, Debug, PartialEq, Eq, Hash)]
struct FactAction(u8);

impl std::fmt::Display for FactAction {
    fn fmt(&self, out: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        out.write_str(&fact_input(self).action)
    }
}

struct FactEdges;

impl Machine for FactEdges {
    type State = Polled;
    type Action = FactAction;
    type Outcome = ();
    type Fact = Fact;

    fn starts(&self) -> Vec<Polled> {
        vec![Polled(0)]
    }
    fn actions(&self) -> Vec<FactAction> {
        (1..=5).map(FactAction).collect()
    }
    fn internal(&self, _: &FactAction) -> bool {
        false
    }
    fn step(&self, state: &Polled, action: FactAction) -> umpire_dsl::Steps<Polled, (), Fact> {
        let action = action.0;
        let mut steps = umpire_dsl::Steps::new();
        if state.0 == 0 {
            let fact = match action {
                1 => Some(Fact::A),
                2 => Some(Fact::B),
                3 => Some(Fact::C),
                4 => Some(Fact::D),
                _ => None,
            };
            if let Some(fact) = fact {
                steps.push(((), vec![fact], Polled(action)));
            }
        } else if state.0 < 5 && action == 5 {
            steps.push(((), Vec::new(), Polled(5)));
        }
        steps
    }
}

fn fact_input(action: &FactAction) -> Concrete {
    let variant = match action.0 {
        1 => "A",
        2 => "B",
        3 => "C",
        4 => "D",
        _ => "Clear",
    };
    Concrete { action: format!("Signal::{variant}"), fields: Vec::new() }
}

const FACT_SOURCE: &str = r"
            #[machine]
            mod facts {
                #[derive(World)] struct World {}
                #[derive(Debug)] enum Fact { A, B, C, D }
                #[derive(Action, Copy)] enum Signal { A, B, C, D, Clear }
                #[derive(State)] struct Model { value: u8 }
                impl Model {
                    #[on(Signal)]
                    fn signal(&mut self, action: Signal) -> Option<Fact> {
                        match action {
                            Signal::A if self.value == 0 => { self.value = 1; Some(Fact::A) }
                            Signal::B if self.value == 0 => { self.value = 2; Some(Fact::B) }
                            Signal::C if self.value == 0 => { self.value = 3; Some(Fact::C) }
                            Signal::D if self.value == 0 => { self.value = 4; Some(Fact::D) }
                            Signal::Clear if self.value > 0 && self.value < 5 => {
                                self.value = 5; None
                            }
                            _ => None,
                        }
                    }
                    #[invariant] fn never_a(&self) { check!(self.value != 1); }
                    #[invariant] fn never_b(&self) { check!(self.value != 2); }
                    #[invariant] fn never_c(&self) { check!(self.value != 3); }
                    #[invariant] fn never_d(&self) { check!(self.value != 4); }
                    #[invariant] fn never_clear(&self) { check!(self.value != 5); }
                }
            }
    ";

#[test]
fn actual_tlc_fact_traces_replay_exact_vectors() -> crate::TestResult {
    if std::env::var("UMPIRE_BACKENDS").as_deref() != Ok("require") {
        return Ok(());
    }
    let source = FACT_SOURCE;
    let binding = umpire_tla::Binding {
        world: Vec::new(),
        starts: vec![vec![("value".into(), Literal::Int(0))]],
        actions: FactEdges.actions().iter().map(fact_input).collect(),
    };
    let emit = |source: &str| -> Result<Module, Box<dyn std::error::Error>> {
        Ok(umpire_tla::Tla::emit(&umpire_tla::Lift::parse("facts.rs", source)?.bind(&binding)?)?)
    };
    let module = emit(source)?;
    let root = std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..");
    let tlc = umpire_tlc::Tlc::discover(&root)?;
    let result = tlc.check_module(&module)?;
    let starts = FactEdges.starts();
    let reached: Vec<_> = (0..=5).map(Polled).collect();
    let native = reached.iter().map(|state| (state.to_string(), state)).collect();
    for (name, expected) in [
        ("never_a", "A"),
        ("never_b", "B"),
        ("never_c", "C"),
        ("never_d", "D"),
        ("never_clear", ""),
    ] {
        let property = &result.properties[&format!("Invariant_{name}")];
        assert_eq!(property.verdict, umpire_tlc::Verdict::Refuted);
        let trace = property.trace.as_ref().ok_or("missing fact witness")?;
        replay(&FactEdges, &starts, &module, &native, trace, Some(&fact_input))?;
        let edge = trace.states.last().ok_or("empty fact witness")?;
        let facts = if expected.is_empty() {
            Vec::new()
        } else {
            vec![Value::Text(format!("Fact::{expected}"))]
        };
        assert_eq!(edge.observers["__umpire_facts"], Value::Seq(facts));
        let replacements = if expected.is_empty() {
            vec![Value::Seq(vec![Value::Text("Fact::A".into())])]
        } else {
            let wrong = if expected == "A" { "B" } else { "A" };
            vec![
                Value::Seq(vec![Value::Text(format!("Fact::{wrong}"))]),
                Value::Seq(Vec::new()),
                Value::Seq(vec![Value::Text(format!("Fact::{expected}")); 2]),
            ]
        };
        for replacement in replacements {
            let mut forged = trace.clone();
            forged
                .states
                .last_mut()
                .ok_or("empty forged trace")?
                .observers
                .insert("__umpire_facts".into(), replacement);
            let error = replay(&FactEdges, &starts, &module, &native, &forged, Some(&fact_input))
                .err()
                .ok_or("forged facts replayed")?;
            assert!(error.to_string().contains("outcome/facts differ"), "{error}");
        }
        let mut omitted = trace.clone();
        omitted.states.last_mut().ok_or("empty omitted trace")?.observers.remove("__umpire_facts");
        let error = replay(&FactEdges, &starts, &module, &native, &omitted, Some(&fact_input))
            .err()
            .ok_or("missing fact observer replayed")?;
        assert!(error.to_string().contains("missing TLC facts"), "{error}");
    }
    let mutant = emit(&source.replacen("Some(Fact::A)", "Some(Fact::B)", 1))?;
    assert_ne!(module.tla, mutant.tla);
    let changed = tlc.check_module(&mutant)?;
    let trace = changed.properties["Invariant_never_a"]
        .trace
        .as_ref()
        .ok_or("mutant has no A-target witness")?;
    let error = replay(&FactEdges, &starts, &mutant, &native, trace, Some(&fact_input))
        .err()
        .ok_or("source-only fact mutant replayed")?;
    assert!(error.to_string().contains("outcome/facts differ"), "{error}");
    Ok(())
}

#[test]
fn nonempty_facts_are_typed_and_preserved_on_observer_activation() -> crate::TestResult {
    let variants = ["A", "B", "C", "D"].map(|value| format!("Fact::{value}")).to_vec();
    let schema = umpire_tlc::Schema {
        fields: BTreeMap::from([("value".into(), umpire_tlc::Type::Bool)]),
        observers: BTreeMap::from([(
            "__umpire_facts".into(),
            umpire_tlc::Type::Seq(Box::new(umpire_tlc::Type::Enum(variants))),
        )]),
    };
    umpire_tlc::decode_state("value = FALSE /\\ __umpire_facts = <<\"Fact::A\">>", &schema)?;
    for invalid in ["<<\"Fact::Unknown\">>", "<<TRUE>>", "\"Fact::A\""] {
        assert!(
            umpire_tlc::decode_state(
                &format!("value = FALSE /\\ __umpire_facts = {invalid}"),
                &schema
            )
            .is_err(),
            "{invalid}"
        );
    }
    let mut before = polled(1, true, "Signal::A", 0);
    before
        .observers
        .insert("__umpire_facts".into(), Value::Seq(vec![Value::Text("Fact::A".into())]));
    before.observers.insert("__umpire_pending".into(), Value::Int(0));
    before.observers.insert("__umpire_steps".into(), Value::Int(0));
    let mut after = before.clone();
    after.observers.insert("__umpire_pending".into(), Value::Int(1));
    let module = module(Vec::new(), Vec::new());
    validate_projection(&module, &before, &after)?;
    after.observers.insert("__umpire_facts".into(), Value::Seq(Vec::new()));
    let error =
        validate_projection(&module, &before, &after).err().ok_or("activation cleared facts")?;
    assert!(error.to_string().contains("__umpire_facts"), "{error}");
    Ok(())
}

#[test]
fn agreement_compares_complete_sets_and_every_verdict() -> crate::TestResult {
    let native = Files::from([
        ("states.txt".into(), "0: State { value: false }\n1: State { value: true }\n".into()),
        ("report.txt".into(), "cut: false\nclaim safe: Verified\n".into()),
    ]);
    let states = BTreeSet::from(["State { value: true }".into(), "State { value: false }".into()]);
    let verdicts = BTreeMap::from([("safe".into(), "Verified".into())]);
    assert!(
        agreement(&native, &states, &verdicts)?.contains("states: 2; native: 2; agreement: yes")
    );
    for (changed_native, changed_states, changed_verdicts, reason) in [
        (
            native.clone(),
            BTreeSet::from(["State { value: true }".into()]),
            verdicts.clone(),
            "missing TLC state",
        ),
        (
            native.clone(),
            states.clone(),
            BTreeMap::from([("safe".into(), "Refuted".into())]),
            "verdict difference",
        ),
        (
            Files::from([
                ("states.txt".into(), "0: State { value: false }\n".into()),
                ("report.txt".into(), "cut: true\nclaim safe: Verified\n".into()),
            ]),
            states.clone(),
            verdicts.clone(),
            "native exploration is cut",
        ),
        (native.clone(), states.clone(), BTreeMap::new(), "missing TLC claim"),
    ] {
        let error = agreement(&changed_native, &changed_states, &changed_verdicts)
            .err()
            .ok_or("difference passed")?;
        assert!(error.to_string().contains(reason), "{error}");
    }
    Ok(())
}

#[test]
fn projection_refuses_state_or_edge_changes_without_a_real_edge() -> crate::TestResult {
    let before = umpire_tlc::TraceState {
        state: State::from([("value".into(), Value::Bool(false))]),
        observers: State::from([
            ("__umpire_edge_tick".into(), Value::Bool(false)),
            ("__umpire_pending".into(), Value::Int(0)),
            ("__umpire_steps".into(), Value::Int(0)),
            ("__umpire_action".into(), Value::Text(String::new())),
            ("__umpire_choice".into(), Value::Int(0)),
            ("__umpire_outcome".into(), Value::Seq(Vec::new())),
            ("__umpire_facts".into(), Value::Seq(Vec::new())),
        ]),
    };
    let module = module(Vec::new(), Vec::new());
    validate_projection(&module, &before, &before)?;
    let mut activation = before.clone();
    activation.observers.insert("__umpire_pending".into(), Value::Int(1));
    validate_projection(&module, &before, &activation)?;
    for (name, value, is_state) in [
        ("value", Value::Bool(true), true),
        ("__umpire_action", Value::Text("User::Change".into()), false),
        ("__umpire_choice", Value::Int(1), false),
        ("__umpire_outcome", Value::Bool(false), false),
        ("__umpire_facts", Value::Seq(vec![Value::Text("forged".into())]), false),
        ("__umpire_pending", Value::Int(2), false),
    ] {
        let mut after = before.clone();
        if is_state {
            after.state.insert(name.into(), value);
        } else {
            after.observers.insert(name.into(), value);
        }
        assert!(validate_projection(&module, &before, &after).is_err(), "projected away {name}");
    }
    Ok(())
}
