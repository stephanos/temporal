use std::{collections::{BTreeMap, BTreeSet}, error::Error, path::Path, time::Duration};
use umpire::dsl::{Machine, Values, IntoValue, Value as NativeValue};
use umpire_checker::Checker;
use umpire_tla::{Module, Type, observers, values::{State, Value}};

type Checked<T> = Result<T, Box<dyn Error>>;
pub const STATES: usize = 36;
pub const EDGES: usize = 144;

#[derive(Clone, Debug, PartialEq, Eq, PartialOrd, Ord)]
pub struct Edge {
    before: State,
    action: String,
    choice: usize,
    after: State,
    outcome: Value,
    facts: Vec<Value>,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Snapshot {
    starts: BTreeSet<State>,
    states: BTreeSet<State>,
    edges: BTreeSet<Edge>,
    verdicts: BTreeMap<String, String>,
}

fn value(value: NativeValue) -> Checked<Value> {
    Ok(match value {
        NativeValue::Bool(v) => Value::Bool(v),
        NativeValue::Int(v) | NativeValue::Bounded { raw: v, .. } => Value::Int(v),
        NativeValue::Enum(v) => Value::Text(v.into()),
        NativeValue::Sequence(v) => Value::Seq(v.into_iter().map(value).collect::<Checked<_>>()?),
        NativeValue::Optional(None) => Value::Int(0),
        NativeValue::Optional(Some(v)) => match *v {
            NativeValue::Bounded { raw, .. } if raw >= 0 && raw < i128::from(i32::MAX) => Value::Int(raw + 1),
            _ => return Err("unsupported optional payload".into()),
        },
    })
}

fn state(state: &impl Values) -> Checked<State> {
    state.values().into_iter().map(|(name, v)| Ok((name.into(), value(v)?))).collect()
}

pub fn native<M>(world: &M, identity: impl Fn(&M::Action) -> String) -> Checked<Snapshot>
where M: Machine<State: Values, Outcome: IntoValue, Fact: IntoValue> {
    let starts = world.starts().iter().map(state).collect::<Checked<BTreeSet<_>>>()?;
    if starts.is_empty() { return Err("empty starts are not completed agreement".into()); }
    let graph = Checker::explore(world, STATES)?;
    if graph.counts.cut || graph.counts.emitted > EDGES { return Err("generated graph exceeds work cap".into()); }
    let mut states = BTreeSet::new();
    let mut edges = BTreeSet::new();
    for (id, before) in graph.states() {
        states.insert(state(before)?);
        for edge in graph.rows(id)? {
            edges.insert(Edge {
                before: state(before)?, action: identity(&edge.action), choice: edge.choice,
                after: state(&edge.emitted.2)?, outcome: value(edge.emitted.0.value())?,
                facts: edge.emitted.1.iter().map(|f| value(f.value())).collect::<Checked<_>>()?,
            });
        }
    }
    let verdicts = graph.verify_all()?.claims.into_iter()
        .map(|claim| (claim.name, format!("{:?}", claim.outcome))).collect();
    Ok(Snapshot { starts, states, edges, verdicts })
}

pub fn agree(compiled: Checked<Snapshot>, interpreted: Checked<Snapshot>) -> Checked<Snapshot> {
    // Each result must succeed separately; matching errors never become empty agreement.
    let compiled = compiled?;
    let interpreted = interpreted?;
    if compiled != interpreted { return Err(format!("complete labeled graphs differ:\n{compiled:#?}\n{interpreted:#?}").into()); }
    Ok(compiled)
}

pub fn tlc(root: &Path, module: &Module) -> Checked<Snapshot> {
    let mut tlc = umpire_tlc::Tlc::discover(root)?;
    tlc.property_workers = 1;
    tlc.limits.timeout = Duration::from_secs(10);
    tlc.limits.output_bytes = 8 * 1024 * 1024;
    tlc.limits.dump_bytes = 16 * 1024 * 1024;
    let ordinary = tlc.check_module(module)?;
    if ordinary.states.len() > STATES { return Err("TLC state cap".into()); }
    let verdicts = ordinary.properties.into_iter().map(|(operator, result)| {
        let claim = module.metadata.invariants.iter().find(|claim| claim.operator == operator)
            .ok_or("unexpected generated property")?;
        Ok((claim.name.clone(), format!("{:?}", result.verdict)))
    }).collect::<Checked<BTreeMap<_, _>>>()?;
    let (tla, schema) = extraction(module)?;
    // Independent full exploration; no native-derived starts, states or edge expectations.
    let extracted = tlc.check(&module.name, &tla,
        "INIT SpikeInit\nNEXT SpikeNext\nCHECK_DEADLOCK TRUE\n", &schema)?;
    if extracted.raw_state_count > 8192 { return Err("augmented assignment cap".into()); }
    let fields: Vec<_> = module.metadata.fields.iter().map(|field| field.name.clone()).collect();
    let mut starts = BTreeSet::new();
    let mut states = BTreeSet::new();
    let mut edges = BTreeSet::new();
    for assignment in extracted.states {
        let after: State = fields.iter().map(|name| Ok((name.clone(), assignment.get(name).ok_or("missing authored field")?.clone())))
            .collect::<Checked<_>>()?;
        states.insert(after.clone());
        match assignment.get("spike_has_step") {
            Some(Value::Bool(false)) => { starts.insert(after.clone()); }
            Some(Value::Bool(true)) => {
                let before = fields.iter().map(|name| {
                    Ok((name.clone(), assignment.get(&format!("spike_before_{name}")).ok_or("missing predecessor")?.clone()))
                }).collect::<Checked<_>>()?;
                let trace = umpire_tla::values::TraceState {
                    state: after.clone(), observers: module.metadata.observer_fields.iter().map(|name| {
                        Ok((name.clone(), assignment.get(name).ok_or("missing observer")?.clone()))
                    }).collect::<Checked<_>>()?,
                };
                let edge = observers::EdgeObservers::decode(&trace)?;
                edges.insert(Edge { before, action: edge.action.into(), choice: edge.choice,
                    after, outcome: edge.outcome.clone(), facts: edge.facts.to_vec() });
            }
            _ => return Err("missing extraction step marker".into()),
        }
    }
    if starts.is_empty() || states != ordinary.states || edges.len() > EDGES {
        return Err("incomplete or over-cap TLC extraction".into());
    }
    Ok(Snapshot { starts, states, edges, verdicts })
}

fn extraction(module: &Module) -> Checked<(String, umpire_tlc::Schema)> {
    let base = module.schema()?;
    let mut fields = base.fields;
    fields.extend(base.observers);
    let mut declarations = String::from("VARIABLE spike_has_step\n");
    let mut init = vec!["Init".to_owned(), "spike_has_step = FALSE".into()];
    // Only Edge is a model edge. Complete is TLC's synthetic completion transition.
    let mut next = vec!["Edge".to_owned(), "spike_has_step' = TRUE".into()];
    for field in &module.metadata.fields {
        let before = format!("spike_before_{}", field.name);
        declarations.push_str(&format!("VARIABLE {before}\n"));
        fields.insert(before.clone(), field.ty.clone());
        init.push(format!("{before} = {}", field.name));
        next.push(format!("{before}' = {}", field.name));
    }
    fields.insert("spike_has_step".into(), Type::Bool);
    let (body, _) = module.tla.rsplit_once("====").ok_or("missing module terminator")?;
    let tla = format!("{body}\n{declarations}\nSpikeInit == {}\nSpikeNext == {}\n====\n", init.join(" /\\ "), next.join(" /\\ "));
    Ok((tla, umpire_tlc::Schema { fields, observers: BTreeMap::new() }))
}

pub fn controls() -> Checked<()> {
    if agree(Err("refused".into()), Err("refused".into())).is_ok() {
        return Err("matching refusals passed".into());
    }
    Ok(())
}

pub fn lowering_controls<M>(world: impl Fn() -> M) -> Checked<()>
where
    M: umpire::dsl::__lifted::Interprets + Values,
    M::Action: umpire::dsl::ActionValues<M>,
    M::State: Values, M::Outcome: IntoValue, M::Fact: IntoValue,
{
    use umpire::dsl::__lifted::{Lifts, expr::{Expr, ExprKind, Op}};
    let identity = |action: &M::Action| umpire::dsl::ActionValues::<M>::action_value(action).0.into();
    let compiled = native(&world(), identity)?;
    let original = M::__umpire_lifted().model.map_err(|e| format!("generator refusal: {e:?}"))?;
    fn swap(expr: &mut Expr) -> bool {
        if let ExprKind::Binary(op @ Op::Lt, _, _) = &mut expr.kind {
            *op = Op::Gt;
            return true;
        }
        expr.children_mut().any(swap)
    }
    let mut changed = original.clone();
    let planted = changed.actions.iter_mut().flat_map(|a| &mut a.branches)
        .any(|branch| swap(&mut branch.guard));
    if !planted { return Err("comparison control was not planted".into()); }
    let interpreter = umpire_checker::Interpreted::from_tree(world(), &changed)?;
    if agree(Ok(compiled.clone()), native(&interpreter, identity)).is_ok() {
        return Err("swapped comparison survived".into());
    }
    let mut changed = original;
    for branch in changed.actions.iter_mut().flat_map(|a| &mut a.branches) {
        branch.facts.kind = ExprKind::Sequence(Vec::new());
    }
    let interpreter = umpire_checker::Interpreted::from_tree(world(), &changed)?;
    let mutated = native(&interpreter, identity)?;
    if compiled.states != mutated.states || compiled.verdicts != mutated.verdicts {
        return Err("fact-only control changed states or verdicts".into());
    }
    if agree(Ok(compiled), Ok(mutated)).is_ok() {
        return Err("fact-only mutation survived".into());
    }
    controls()
}
