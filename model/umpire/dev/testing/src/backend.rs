//! Independent backend execution and complete native agreement.

use std::{
    collections::{BTreeMap, BTreeSet},
    fmt::Write,
};
use umpire_dsl::Machine;
use umpire_tla::{Concrete, Literal, Module, Type};
use umpire_tlc::{State, Value};

fn debug_value(ty: &Type, value: &Value) -> Result<String, Box<dyn std::error::Error>> {
    match (ty, value) {
        (Type::Bool, Value::Bool(value)) => Ok(value.to_string()),
        (Type::Int { .. }, Value::Int(value)) => Ok(value.to_string()),
        (Type::Bounded(bounded), Value::Int(raw)) if bounded.contains(*raw) => {
            Ok(bounded_debug(bounded, *raw))
        }
        (Type::Optional(bounded), Value::Int(code)) => match bounded.decode(*code) {
            Some(None) => Ok("None".into()),
            Some(Some(raw)) => Ok(format!("Some({})", bounded_debug(bounded, raw))),
            None => Err(format!("invalid optional code for {}: {code}", bounded.name).into()),
        },
        (Type::Text, Value::Text(value)) => Ok(format!("{value:?}")),
        (Type::Enum { name, variants }, Value::Text(value)) if variants.contains(value) => {
            Ok(value.strip_prefix(&format!("{name}::")).unwrap_or(value).to_owned())
        }
        (Type::Unit, Value::Seq(values)) if values.is_empty() => Ok("()".into()),
        (Type::Seq(inner), Value::Seq(values)) => Ok(format!(
            "[{}]",
            values
                .iter()
                .map(|value| debug_value(inner, value))
                .collect::<Result<Vec<_>, _>>()?
                .join(", ")
        )),
        (Type::Tuple(types), Value::Seq(values)) if types.len() == values.len() => Ok(format!(
            "({}{})",
            types
                .iter()
                .zip(values)
                .map(|(ty, value)| debug_value(ty, value))
                .collect::<Result<Vec<_>, _>>()?
                .join(", "),
            if types.len() == 1 { "," } else { "" }
        )),
        _ => Err(format!("decoded value does not match authored type: {ty:?}: {value:?}").into()),
    }
}

fn bounded_debug(bounded: &umpire_dsl::__lifted::Bounded, raw: i128) -> String {
    bounded.field.as_ref().map_or_else(
        || format!("{}({raw})", bounded.name),
        |field| format!("{} {{ {field}: {raw} }}", bounded.name),
    )
}

fn state_text(module: &Module, state: &State) -> Result<String, Box<dyn std::error::Error>> {
    if state.len() != module.metadata.fields.len() {
        return Err("decoded state has different declared fields".into());
    }
    let fields = module
        .metadata
        .fields
        .iter()
        .map(|field| {
            let value = state.get(&field.name).ok_or("missing authored state field")?;
            Ok(format!("{}: {}", field.name, debug_value(&field.ty, value)?))
        })
        .collect::<Result<Vec<_>, Box<dyn std::error::Error>>>()?;
    Ok(format!("{} {{ {} }}", module.metadata.state_name, fields.join(", ")))
}

/// Execute the independently emitted module and replay every returned counterexample.
pub fn check_machine<M: Machine>(
    world: &M,
    module: &Module,
    folder: &std::path::Path,
    native: &crate::Files,
) -> crate::RunResult {
    let starts = world.starts();
    let native_graph = umpire_checker::Checker::explore_within(
        world,
        starts.clone(),
        1 << 20,
        umpire_checker::Checker::CATALOG_LIMIT,
    )?;
    check_exploration(world, &starts, &native_graph, module, folder, native, None)
}

/// Execute an independently emitted module against a retained native exploration and starts.
/// An input catalog replays edges by catalog ordinal: the native action at that ordinal must
/// project, through the typed `identity`, to exactly the emitted concrete identity.
#[allow(clippy::print_stderr)]
#[expect(clippy::too_many_arguments, reason = "The typed identity extends the shared route")]
pub fn check_exploration<M: Machine>(
    world: &M,
    starts: &[M::State],
    native_graph: &umpire_checker::Result<M>,
    module: &Module,
    folder: &std::path::Path,
    native: &crate::Files,
    identity: Option<&dyn Fn(&M::Action) -> Concrete>,
) -> crate::RunResult {
    let root = folder
        .ancestors()
        .find(|path| path.join("mise.toml").is_file())
        .ok_or("backend repository root not found")?;
    if native_graph.counts.cut {
        return Err("native exploration is cut; no complete agreement".into());
    }
    let native_states = native_graph
        .states()
        .map(|(_, state)| (state.to_string(), state))
        .collect::<BTreeMap<_, _>>();
    if native_states.len() != native_graph.counts.reached {
        return Err("native state Display collision prevents exact backend agreement".into());
    }
    let complete = umpire_tlc::Tlc::discover(root)?.check_module(module)?;
    eprintln!("TLC raw output for {}:\n{}", folder.display(), complete.output);
    let states = complete
        .states
        .iter()
        .map(|state| state_text(module, state))
        .collect::<Result<BTreeSet<_>, _>>()?;
    let mut verdicts = BTreeMap::new();
    let mut replayed = 0;
    for (operator, property) in complete.properties {
        let name = module
            .metadata
            .invariants
            .iter()
            .chain(&module.metadata.postconditions)
            .chain(&module.metadata.negative)
            .find(|claim| claim.operator == operator)
            .map(|claim| claim.name.as_str())
            .or_else(|| {
                module
                    .metadata
                    .progress
                    .iter()
                    .find(|claim| claim.operator == operator)
                    .map(|claim| claim.name.as_str())
            })
            .ok_or_else(|| format!("TLC returned undeclared property {operator}"))?;
        verdicts.insert(name.to_owned(), format!("{:?}", property.verdict));
        if let Some(trace) = property.trace {
            replay(world, starts, module, &native_states, &trace, identity)?;
            replayed += 1;
        }
    }
    let mut report = String::from("TLC 1.7.4; complete source-lifted exploration\n");
    report.push_str(&agreement(native, &states, &verdicts)?);
    writeln!(report, "counterexamples replayed: {replayed}")?;
    Ok(crate::Files::from([("tlc.txt".into(), report)]))
}

fn replay<M: Machine>(
    world: &M,
    starts: &[M::State],
    module: &Module,
    native: &BTreeMap<String, &M::State>,
    trace: &umpire_tlc::Trace,
    identity: Option<&dyn Fn(&M::Action) -> Concrete>,
) -> crate::TestResult {
    if trace.states.is_empty() {
        return Err("empty TLC counterexample".into());
    }
    let first = state_text(module, &trace.states[0].state)?;
    if !starts.iter().any(|state| state.to_string() == first) {
        return Err("TLC counterexample does not begin at a Model start".into());
    }
    let mut pairs = trace.states.windows(2).map(|pair| (&pair[0], &pair[1])).collect::<Vec<_>>();
    if let Some(index) = trace.back_to {
        pairs.push((
            trace.states.last().ok_or("missing trace end")?,
            trace.states.get(index).ok_or("invalid TLC cycle index")?,
        ));
    }
    for (before, after) in pairs {
        let before_text = state_text(module, &before.state)?;
        let after_text = state_text(module, &after.state)?;
        let tick_before = before.observers.get("__umpire_edge_tick").ok_or("missing edge tick")?;
        let tick_after =
            after.observers.get("__umpire_edge_tick").ok_or("missing next edge tick")?;
        if tick_before == tick_after {
            validate_projection(module, before, after)?;
            continue;
        }
        let Some(Value::Text(action_name)) = after.observers.get("__umpire_action") else {
            return Err("counterexample lacks action identity".into());
        };
        let Some(Value::Int(choice)) = after.observers.get("__umpire_choice") else {
            return Err("counterexample lacks choice identity".into());
        };
        let catalog = world.checked_catalog(umpire_dsl::CATALOG_LIMIT)?.into_actions();
        let action = if module.metadata.catalog.is_empty() {
            catalog
                .into_iter()
                .find(|action| action.to_string() == *action_name)
                .ok_or("TLC action is not in Model catalog")?
        } else {
            typed_action(module, catalog, after, action_name, identity)?
        };
        let state =
            native.get(&before_text).ok_or("TLC trace predecessor is not reachable natively")?;
        let steps = native_steps(world, state, action)?;
        let step = steps
            .get(usize::try_from(*choice)?)
            .ok_or("TLC trace choice is not emitted by Model")?;
        if step.2.to_string() != after_text {
            return Err(format!("TLC trace replay differs: {action_name} choice {choice}").into());
        }
        let outcome = after.observers.get("__umpire_outcome").ok_or("missing TLC outcome")?;
        let facts = after.observers.get("__umpire_facts").ok_or("missing TLC facts")?;
        let fact_type =
            module.metadata.observer_types.get("__umpire_facts").ok_or("missing TLC fact type")?;
        if format!("{:?}", step.0) != debug_value(&module.metadata.outcome_type, outcome)?
            || format!("{:?}", step.1) != debug_value(fact_type, facts)?
        {
            return Err("TLC counterexample outcome/facts differ from Model replay".into());
        }
    }
    Ok(())
}

/// The native action at the decoded catalog ordinal, after validating the emitted identity
/// against the typed metadata and the generated native projection; never by display text.
fn typed_action<A>(
    module: &Module,
    catalog: Vec<A>,
    after: &umpire_tlc::TraceState,
    action_name: &str,
    identity: Option<&dyn Fn(&A) -> Concrete>,
) -> Result<A, Box<dyn std::error::Error>> {
    let Some(Value::Int(ordinal)) = after.observers.get(INPUT) else {
        return Err("counterexample lacks input identity".into());
    };
    let ordinal = usize::try_from(*ordinal)?;
    let entry =
        module.metadata.catalog.get(ordinal).ok_or("TLC input ordinal is outside the catalog")?;
    if entry.action != action_name {
        return Err(format!(
            "TLC input ordinal {ordinal} is {}, not the edge's action {action_name}",
            entry.action
        )
        .into());
    }
    let declared = module
        .metadata
        .actions
        .iter()
        .find(|action| action.name == entry.action)
        .ok_or("TLC input names an undeclared action")?;
    if entry.fields.len() != declared.fields.len()
        || !entry
            .fields
            .iter()
            .zip(&declared.fields)
            .all(|((name, value), field)| *name == field.name && admits(&field.ty, value))
    {
        return Err(
            format!("TLC input ordinal {ordinal} does not match its declared fields").into()
        );
    }
    let identity = identity.ok_or("an input catalog requires the typed native identity")?;
    let action = catalog
        .into_iter()
        .nth(ordinal)
        .ok_or("TLC input ordinal is outside the native catalog")?;
    if identity(&action) != *entry {
        return Err(format!("native catalog ordinal {ordinal} differs from the TLC input").into());
    }
    Ok(action)
}

fn admits(ty: &Type, value: &Literal) -> bool {
    match (ty, value) {
        (Type::Bool, Literal::Bool(_)) => true,
        (Type::Int { min, max }, Literal::Int(value)) => (*min..=*max).contains(value),
        (Type::Enum { variants, .. }, Literal::Enum(value)) => variants.contains(value),
        (Type::Bounded(bounded), Literal::Bounded { name, raw }) => {
            *name == bounded.name && bounded.contains(*raw)
        }
        (Type::Optional(bounded), Literal::Optional(value)) if bounded.optional_valid() => {
            value.as_ref().is_none_or(|value| admits(&Type::Bounded(bounded.clone()), value))
        }
        _ => false,
    }
}

const INPUT: &str = "__umpire_input";

fn validate_projection(
    module: &Module,
    before: &umpire_tlc::TraceState,
    after: &umpire_tlc::TraceState,
) -> crate::TestResult {
    if before.state != after.state {
        return Err("observer-only TLC step changes Model state".into());
    }
    let input = (!module.metadata.catalog.is_empty()).then_some(INPUT);
    for name in ["__umpire_action", "__umpire_choice", "__umpire_outcome", "__umpire_facts"]
        .into_iter()
        .chain(input)
    {
        if before.observers.get(name) != after.observers.get(name) {
            return Err(format!("observer-only TLC step changes edge field {name}").into());
        }
    }
    let unchanged = before.observers == after.observers;
    let activation = before.observers.get("__umpire_pending") == Some(&Value::Int(0))
        && after.observers.get("__umpire_pending") == Some(&Value::Int(1))
        && before.observers.get("__umpire_steps") == Some(&Value::Int(0))
        && after.observers.get("__umpire_steps") == Some(&Value::Int(0));
    if !unchanged && !activation {
        return Err("counterexample has an undeclared observer-only step".into());
    }
    Ok(())
}

fn native_steps<M: Machine>(
    world: &M,
    state: &M::State,
    action: M::Action,
) -> Result<umpire_dsl::Steps<M::State, M::Outcome, M::Fact>, umpire_checker::Refusal> {
    let action_name = action.to_string();
    world.checked_step(state, action).map_err(|error| umpire_checker::Refusal::Step {
        action: action_name,
        before: state.to_string(),
        error,
    })
}

/// Compare complete reached sets and every registered property, retaining differences.
fn agreement(
    native: &crate::Files,
    states: &BTreeSet<String>,
    verdicts: &BTreeMap<String, String>,
) -> Result<String, Box<dyn std::error::Error>> {
    let mut expected_claims = BTreeMap::new();
    for name in ["report.txt", "negative_report.txt"] {
        if let Some(report) = native.get(name) {
            if !report.lines().any(|line| line == "cut: false") {
                return Err("native exploration is cut or has no completeness evidence".into());
            }
            for line in report.lines().filter_map(|line| line.strip_prefix("claim ")) {
                let (name, verdict) = line.split_once(": ").ok_or("malformed native claim")?;
                if !matches!(verdict, "Verified" | "Refuted") {
                    return Err(format!("native claim {name} is inconclusive: {verdict}").into());
                }
                if let Some(prior) = expected_claims.insert(name.to_owned(), verdict.to_owned())
                    && prior != verdict
                {
                    return Err(format!("inconsistent native verdict for {name}").into());
                }
            }
        }
    }
    if !native.contains_key("report.txt") {
        return Err("missing native report".into());
    }
    let native_states: BTreeSet<_> = native
        .get("states.txt")
        .ok_or("missing native states")?
        .lines()
        .map(|line| {
            line.split_once(": ").map(|(_, state)| state.to_owned()).ok_or("malformed native state")
        })
        .collect::<Result<_, _>>()?;
    let mut report = String::new();
    writeln!(
        report,
        "states: {}; native: {}; agreement: {}",
        states.len(),
        native_states.len(),
        if *states == native_states { "yes" } else { "no" }
    )?;
    let mut differs = false;
    for state in native_states.difference(states) {
        writeln!(report, "missing TLC state: {state}")?;
        differs = true;
    }
    for state in states.difference(&native_states) {
        writeln!(report, "extra TLC state: {state}")?;
        differs = true;
    }
    for (name, expected) in &expected_claims {
        match verdicts.get(name) {
            Some(actual) if actual == expected => {
                writeln!(report, "claim {name}: {actual}; native: {expected}; agreement: yes")?;
            }
            Some(actual) => {
                writeln!(report, "verdict difference {name}: TLC {actual}; native {expected}")?;
                differs = true;
            }
            None => {
                writeln!(report, "missing TLC claim: {name}")?;
                differs = true;
            }
        }
    }
    for name in verdicts.keys().filter(|name| !expected_claims.contains_key(*name)) {
        writeln!(report, "extra TLC claim: {name}")?;
        differs = true;
    }
    if differs {
        return Err(report.into());
    }
    Ok(report)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::Files;

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

    #[test]
    fn nominal_action_literals_require_exact_typed_identity() {
        let ticks = bounded("Ticks", None, 1);
        let ty = Type::Optional(ticks.clone());
        assert!(admits(&ty, &Literal::Optional(None)));
        assert!(admits(
            &ty,
            &Literal::Optional(Some(Box::new(Literal::Bounded { name: "Ticks".into(), raw: 1 })))
        ));
        for value in [
            Literal::Int(2),
            Literal::Optional(Some(Box::new(Literal::Int(1)))),
            Literal::Optional(Some(Box::new(Literal::Bounded {
                name: "Remaining".into(),
                raw: 1,
            }))),
            Literal::Optional(Some(Box::new(Literal::Bounded { name: "Ticks".into(), raw: 0 }))),
            Literal::Optional(Some(Box::new(Literal::Optional(None)))),
        ] {
            assert!(!admits(&ty, &value), "{value:?}");
        }
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
        std::panic::catch_unwind(|| {
            replay(&RetainedStarts, &starts, &module, &native, &trace, None)
        })
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
        valid
            .metadata
            .observer_types
            .insert("__umpire_facts".into(), Type::Seq(Box::new(Type::Unit)));
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
        module
            .metadata
            .observer_types
            .insert("__umpire_facts".into(), Type::Seq(Box::new(Type::Unit)));
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
            Ok(umpire_tla::Tla::emit(
                &umpire_tla::Lift::parse("facts.rs", source)?.bind(&binding)?,
            )?)
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
                let error =
                    replay(&FactEdges, &starts, &module, &native, &forged, Some(&fact_input))
                        .err()
                        .ok_or("forged facts replayed")?;
                assert!(error.to_string().contains("outcome/facts differ"), "{error}");
            }
            let mut omitted = trace.clone();
            omitted
                .states
                .last_mut()
                .ok_or("empty omitted trace")?
                .observers
                .remove("__umpire_facts");
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
        let error = validate_projection(&module, &before, &after)
            .err()
            .ok_or("activation cleared facts")?;
        assert!(error.to_string().contains("__umpire_facts"), "{error}");
        Ok(())
    }

    #[test]
    fn agreement_compares_complete_sets_and_every_verdict() -> crate::TestResult {
        let native = Files::from([
            ("states.txt".into(), "0: State { value: false }\n1: State { value: true }\n".into()),
            ("report.txt".into(), "cut: false\nclaim safe: Verified\n".into()),
        ]);
        let states =
            BTreeSet::from(["State { value: true }".into(), "State { value: false }".into()]);
        let verdicts = BTreeMap::from([("safe".into(), "Verified".into())]);
        assert!(
            agreement(&native, &states, &verdicts)?
                .contains("states: 2; native: 2; agreement: yes")
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
            assert!(
                validate_projection(&module, &before, &after).is_err(),
                "projected away {name}"
            );
        }
        Ok(())
    }
}
