//! Independent backend execution and complete native agreement.

use std::{
    collections::{BTreeMap, BTreeSet},
    fmt::Write,
};
use umpire_dsl::{__lifted::ConcreteOf, ActionValues, IntoValue, Machine, Values};
use umpire_tla::{Concrete, Module, Type};
use umpire_tla::{
    observers::{FACTS, INPUT},
    replay::{self as core, EdgeValues, ReplayError},
};
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
    let name = bounded.name.split('<').next().unwrap_or(&bounded.name);
    let name = name.rsplit("::").next().unwrap_or(name);
    bounded
        .field
        .as_ref()
        .map_or_else(|| format!("{name}({raw})"), |field| format!("{name} {{ {field}: {raw} }}"))
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
pub fn check_machine<M: Machine<State: Values, Outcome: IntoValue, Fact: IntoValue>>(
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
pub fn check_exploration<M: Machine<State: Values, Outcome: IntoValue, Fact: IntoValue>>(
    world: &M,
    starts: &[M::State],
    native_graph: &umpire_checker::Result<M>,
    module: &Module,
    folder: &std::path::Path,
    native: &crate::Files,
    identity: Option<&dyn Fn(&M::Action) -> Concrete>,
) -> crate::RunResult {
    let root = std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("../..").canonicalize()?;
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
    let complete = umpire_tlc::Tlc::discover(&root)?.check_module(module)?;
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

/// A trace refusal, with the native edge projection available for semantic disagreements.
pub struct Divergence {
    /// One-based trace transition, or zero for a refusal before edge replay.
    pub transition: usize,
    /// The specific replay refusal.
    pub reason: String,
    /// The native successor, outcome and facts, when the selected edge could be evaluated.
    pub native: Option<EdgeValues>,
}

impl std::fmt::Debug for Divergence {
    fn fmt(&self, out: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        out.debug_struct("Divergence")
            .field("transition", &self.transition)
            .field("reason", &self.reason)
            .field(
                "native",
                &self.native.as_ref().map(|edge| (&edge.state, &edge.outcome, &edge.facts)),
            )
            .finish()
    }
}

impl std::fmt::Display for Divergence {
    fn fmt(&self, out: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        if self.transition == 0 {
            out.write_str(&self.reason)
        } else {
            write!(out, "TLC transition {}: {}", self.transition, self.reason)
        }
    }
}

impl std::error::Error for Divergence {}

/// Replay a typed TLC trace against the Model and the selected starts.
///
/// # Errors
/// Refuses incomplete native exploration, malformed traces and any state, outcome or fact mismatch.
pub fn replay_trace<
    M: Machine<State: Values, Action: ActionValues<M>, Outcome: IntoValue, Fact: IntoValue>,
>(
    world: &M,
    starts: &[M::State],
    module: &Module,
    trace: &umpire_tlc::Trace,
) -> Result<(), Divergence> {
    let graph = umpire_checker::Checker::explore_within(
        world,
        starts.to_vec(),
        1 << 20,
        umpire_checker::Checker::CATALOG_LIMIT,
    )
    .map_err(|error| Divergence { transition: 0, reason: error.to_string(), native: None })?;
    if graph.counts.cut {
        return Err(Divergence {
            transition: 0,
            reason: "native exploration is cut; no complete replay".into(),
            native: None,
        });
    }
    let native = graph.states().map(|(id, state)| (id.0.to_string(), state)).collect();
    replay(world, starts, module, &native, trace, Some(&Concrete::of::<M>))
}

fn replay<M: Machine<State: Values, Outcome: IntoValue, Fact: IntoValue>>(
    world: &M,
    starts: &[M::State],
    module: &Module,
    native: &BTreeMap<String, &M::State>,
    trace: &umpire_tlc::Trace,
    identity: Option<&dyn Fn(&M::Action) -> Concrete>,
) -> Result<(), Divergence> {
    let mut transition = 0;
    let mut native_edge = None;
    let result = (|| -> crate::TestResult {
        if trace.states.is_empty() {
            return Err("empty TLC counterexample".into());
        }
        let typed_state = |state: &M::State| {
            core::fields(
                state.values(),
                module.metadata.fields.iter().map(|f| (f.name.as_str(), &f.ty)),
            )
        };
        let native = native
            .values()
            .map(|state| typed_state(state).map(|value| (value, *state)))
            .collect::<Result<BTreeMap<_, _>, _>>()?;
        if !starts.iter().any(|state| typed_state(state).is_ok_and(|s| s == trace.states[0].state))
        {
            return Err("TLC counterexample does not begin at a Model start".into());
        }
        let mut pairs =
            trace.states.windows(2).map(|pair| (&pair[0], &pair[1])).collect::<Vec<_>>();
        if let Some(index) = trace.back_to {
            pairs.push((
                trace.states.last().ok_or("missing trace end")?,
                trace.states.get(index).ok_or("invalid TLC cycle index")?,
            ));
        }
        for (index, (before, after)) in pairs.into_iter().enumerate() {
            transition = index + 1;
            native_edge = None;
            let tick_before = before
                .observers
                .get(umpire_tla::observers::EDGE_TICK)
                .ok_or("missing edge tick")?;
            let tick_after = after
                .observers
                .get(umpire_tla::observers::EDGE_TICK)
                .ok_or("missing next edge tick")?;
            if tick_before == tick_after {
                validate_projection(module, before, after)?;
                continue;
            }
            let state = native
                .get(&before.state)
                .ok_or("TLC trace predecessor is not reachable natively")?;
            let catalog = world.checked_catalog(umpire_dsl::CATALOG_LIMIT)?.into_actions();
            let replayed = core::step(
                module,
                after,
                &catalog,
                |action| {
                    identity.map_or_else(
                        || {
                            module
                                .metadata
                                .catalog
                                .is_empty()
                                .then(|| Concrete { action: action.to_string(), fields: vec![] })
                        },
                        |identity| Some(identity(action)),
                    )
                },
                |action| native_steps(world, state, action.clone()),
                |step| {
                    let Some(Type::Seq(fact_type)) = module.metadata.observer_types.get(FACTS)
                    else {
                        return Err(umpire_tla::Refusal("missing facts observer type".into()));
                    };
                    let values = EdgeValues {
                        state: typed_state(&step.2)?,
                        outcome: core::admitted(step.0.value(), &module.metadata.outcome_type)?,
                        facts: step
                            .1
                            .iter()
                            .map(|fact| core::admitted(fact.value(), fact_type))
                            .collect::<Result<_, _>>()?,
                    };
                    native_edge = Some(EdgeValues {
                        state: values.state.clone(),
                        outcome: values.outcome.clone(),
                        facts: values.facts.clone(),
                    });
                    Ok(values)
                },
            );
            match replayed {
                Ok(_) => {}
                Err(ReplayError::Divergence(error)) => return Err(error.into()),
                Err(ReplayError::Native(error)) => return Err(error.into()),
            }
        }
        Ok(())
    })();
    result.map_err(|error| Divergence {
        transition,
        reason: error.to_string(),
        native: native_edge,
    })
}

fn validate_projection(
    module: &Module,
    before: &umpire_tlc::TraceState,
    after: &umpire_tlc::TraceState,
) -> crate::TestResult {
    if before.state != after.state {
        return Err("observer-only TLC step changes Model state".into());
    }
    let input = (!module.metadata.catalog.is_empty()).then_some(INPUT);
    for name in [
        umpire_tla::observers::ACTION,
        umpire_tla::observers::CHOICE,
        umpire_tla::observers::OUTCOME,
        FACTS,
    ]
    .into_iter()
    .chain(input)
    {
        if before.observers.get(name) != after.observers.get(name) {
            return Err(format!("observer-only TLC step changes edge field {name}").into());
        }
    }
    let unchanged = before.observers == after.observers;
    let activation = before.observers.get(umpire_tla::observers::PENDING) == Some(&Value::Int(0))
        && after.observers.get(umpire_tla::observers::PENDING) == Some(&Value::Int(1))
        && before.observers.get(umpire_tla::observers::STEPS) == Some(&Value::Int(0))
        && after.observers.get(umpire_tla::observers::STEPS) == Some(&Value::Int(0));
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
