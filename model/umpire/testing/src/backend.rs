//! Independent backend execution and complete native agreement.

use std::{
    collections::{BTreeMap, BTreeSet},
    fmt::Write,
};
use umpire_dsl::Machine;
use umpire_lift::{Module, Type};
use umpire_tlc::{State, Value};

fn debug_value(ty: &Type, value: &Value) -> Result<String, Box<dyn std::error::Error>> {
    match (ty, value) {
        (Type::Bool, Value::Bool(value)) => Ok(value.to_string()),
        (Type::Int { .. }, Value::Int(value)) => Ok(value.to_string()),
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

fn state_text(module: &Module, state: &State) -> Result<String, Box<dyn std::error::Error>> {
    if state.len() != module.metadata.fields.len() {
        return Err("decoded state has different declared fields".into());
    }
    let fields = module
        .metadata
        .fields
        .iter()
        .enumerate()
        .map(|(index, field)| {
            let value = state.get(&field.name).ok_or("missing authored state field")?;
            let label =
                if module.metadata.state_tuple { index.to_string() } else { field.name.clone() };
            Ok(format!("{label}: {}", debug_value(&field.ty, value)?))
        })
        .collect::<Result<Vec<_>, Box<dyn std::error::Error>>>()?;
    Ok(format!("{} {{ {} }}", module.metadata.state_name, fields.join(", ")))
}

/// Execute the independently emitted module and replay every returned counterexample.
#[allow(clippy::print_stderr)]
pub fn check_machine<M: Machine>(
    world: &M,
    module: &Module,
    folder: &std::path::Path,
    native: &crate::Files,
) -> crate::RunResult {
    let root = folder
        .ancestors()
        .find(|path| path.join("mise.toml").is_file())
        .ok_or("backend repository root not found")?;
    let native_graph = umpire_checker::Checker::explore(world, 1 << 20)?;
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
            replay(world, module, &native_states, &trace)?;
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
    module: &Module,
    native: &BTreeMap<String, &M::State>,
    trace: &umpire_tlc::Trace,
) -> crate::TestResult {
    if trace.states.is_empty() {
        return Err("empty TLC counterexample".into());
    }
    let first = state_text(module, &trace.states[0].state)?;
    if !world.starts().iter().any(|state| state.to_string() == first) {
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
            validate_projection(before, after)?;
            continue;
        }
        let Some(Value::Text(action_name)) = after.observers.get("__umpire_action") else {
            return Err("counterexample lacks action identity".into());
        };
        let Some(Value::Int(choice)) = after.observers.get("__umpire_choice") else {
            return Err("counterexample lacks choice identity".into());
        };
        let action = world
            .actions()
            .into_iter()
            .find(|action| action.to_string() == *action_name)
            .ok_or("TLC action is not in Model catalog")?;
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
        if format!("{:?}", step.0) != debug_value(&module.metadata.outcome_type, outcome)?
            || format!("{:?}", step.1)
                != debug_value(&Type::Seq(Box::new(module.metadata.fact_type.clone())), facts)?
        {
            return Err("TLC counterexample outcome/facts differ from Model replay".into());
        }
    }
    Ok(())
}

fn validate_projection(
    before: &umpire_tlc::TraceState,
    after: &umpire_tlc::TraceState,
) -> crate::TestResult {
    if before.state != after.state {
        return Err("observer-only TLC step changes Model state".into());
    }
    for name in ["__umpire_action", "__umpire_choice", "__umpire_outcome", "__umpire_facts"] {
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
        validate_projection(&before, &before)?;
        let mut activation = before.clone();
        activation.observers.insert("__umpire_pending".into(), Value::Int(1));
        validate_projection(&before, &activation)?;
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
            assert!(validate_projection(&before, &after).is_err(), "projected away {name}");
        }
        Ok(())
    }
}
