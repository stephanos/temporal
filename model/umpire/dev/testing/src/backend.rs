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
#[path = "backend_tests.rs"]
mod tests;
