use std::{
    collections::{BTreeMap, BTreeSet},
    error::Error,
    path::Path,
    time::Duration,
};
use umpire::dsl::{IntoValue, Machine, Value as NativeValue, Values};
use umpire_checker::Checker;
use umpire_tla::{
    Module, Type, observers,
    values::{State, Value},
};

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

fn value(input: NativeValue) -> Checked<Value> {
    Ok(match input {
        NativeValue::Bool(v) => Value::Bool(v),
        NativeValue::Int(v) | NativeValue::Bounded { raw: v, .. } => Value::Int(v),
        NativeValue::Enum(v) => Value::Text(v.into()),
        NativeValue::Sequence(v) => Value::Seq(v.into_iter().map(value).collect::<Checked<_>>()?),
        NativeValue::Optional(None) => Value::Int(0),
        NativeValue::Optional(Some(v)) => match *v {
            NativeValue::Bounded { raw, .. } if raw >= 0 && raw < i128::from(i32::MAX) => {
                Value::Int(raw + 1)
            }
            _ => return Err("unsupported optional payload".into()),
        },
    })
}

fn state(state: &impl Values) -> Checked<State> {
    state
        .values()
        .into_iter()
        .map(|(name, v)| Ok((name.into(), value(v)?)))
        .collect()
}

pub fn native<M>(world: &M, identity: impl Fn(&M::Action) -> String) -> Checked<Snapshot>
where
    M: Machine<State: Values, Outcome: IntoValue, Fact: IntoValue>,
{
    native_with_cap(world, identity, STATES)
}

fn native_with_cap<M>(
    world: &M,
    identity: impl Fn(&M::Action) -> String,
    cap: usize,
) -> Checked<Snapshot>
where
    M: Machine<State: Values, Outcome: IntoValue, Fact: IntoValue>,
{
    let starts = world
        .starts()
        .iter()
        .map(state)
        .collect::<Checked<BTreeSet<_>>>()?;
    if starts.is_empty() {
        return Err("empty starts are not completed agreement".into());
    }
    let graph = Checker::explore(world, cap)?;
    if graph.counts.cut || graph.counts.emitted > EDGES {
        return Err("generated graph exceeds work cap".into());
    }
    let mut states = BTreeSet::new();
    let mut edges = BTreeSet::new();
    for (id, before) in graph.states() {
        states.insert(state(before)?);
        for edge in graph.rows(id)? {
            edges.insert(Edge {
                before: state(before)?,
                action: identity(&edge.action),
                choice: edge.choice,
                after: state(&edge.emitted.2)?,
                outcome: value(edge.emitted.0.value())?,
                facts: edge
                    .emitted
                    .1
                    .iter()
                    .map(|f| value(f.value()))
                    .collect::<Checked<_>>()?,
            });
        }
    }
    let verdicts = graph
        .verify_all()?
        .claims
        .into_iter()
        .map(|claim| (claim.name, format!("{:?}", claim.outcome)))
        .collect();
    println!(
        "graph starts={} states={} edges={} verdicts={:?}",
        starts.len(),
        states.len(),
        edges.len(),
        verdicts
    );
    Ok(Snapshot {
        starts,
        states,
        edges,
        verdicts,
    })
}

pub fn agree(compiled: Checked<Snapshot>, interpreted: Checked<Snapshot>) -> Checked<Snapshot> {
    // Each result must succeed separately; matching errors never become empty agreement.
    let compiled = compiled?;
    let interpreted = interpreted?;
    if compiled != interpreted {
        return Err(
            format!("complete labeled graphs differ:\n{compiled:#?}\n{interpreted:#?}").into(),
        );
    }
    Ok(compiled)
}

pub fn tlc(root: &Path, module: &Module) -> Checked<Snapshot> {
    let mut tlc = umpire_tlc::Tlc::discover(root)?;
    tlc.property_workers = 1;
    tlc.limits.timeout = Duration::from_secs(10);
    tlc.limits.output_bytes = 8 * 1024 * 1024;
    tlc.limits.dump_bytes = 16 * 1024 * 1024;
    let ordinary = tlc.check_module(module)?;
    if ordinary.states.len() > STATES {
        return Err("TLC state cap".into());
    }
    let verdicts = ordinary
        .properties
        .into_iter()
        .map(|(operator, result)| {
            let claim = module
                .metadata
                .invariants
                .iter()
                .find(|claim| claim.operator == operator)
                .ok_or("unexpected generated property")?;
            Ok((claim.name.clone(), format!("{:?}", result.verdict)))
        })
        .collect::<Checked<BTreeMap<_, _>>>()?;
    let (tla, schema) = extraction(module)?;
    if let Ok(folder) = std::env::var("FN255_EVIDENCE") {
        std::fs::write(Path::new(&folder).join("extracted.tla"), &tla)?;
    }
    // Independent full exploration; no native-derived starts, states or edge expectations.
    let extracted = tlc.check(
        &module.name,
        &tla,
        "INIT SpikeInit\nNEXT SpikeNext\nCHECK_DEADLOCK TRUE\n",
        &schema,
    )?;
    if extracted.raw_state_count > 290 {
        return Err("augmented assignment cap".into());
    }
    let fields: Vec<_> = module
        .metadata
        .fields
        .iter()
        .map(|field| field.name.clone())
        .collect();
    let mut starts = BTreeSet::new();
    let mut states = BTreeSet::new();
    let mut edges = BTreeSet::new();
    for assignment in extracted.states {
        let after: State = fields
            .iter()
            .map(|name| {
                Ok((
                    name.clone(),
                    assignment
                        .get(name)
                        .ok_or("missing authored field")?
                        .clone(),
                ))
            })
            .collect::<Checked<_>>()?;
        states.insert(after.clone());
        match assignment.get("spike_has_step") {
            Some(Value::Bool(false)) => {
                starts.insert(after.clone());
            }
            Some(Value::Bool(true)) => {
                let before = fields
                    .iter()
                    .map(|name| {
                        Ok((
                            name.clone(),
                            assignment
                                .get(&format!("spike_before_{name}"))
                                .ok_or("missing predecessor")?
                                .clone(),
                        ))
                    })
                    .collect::<Checked<_>>()?;
                let trace = umpire_tla::values::TraceState {
                    state: after.clone(),
                    observers: module
                        .metadata
                        .observer_fields
                        .iter()
                        .map(|name| {
                            Ok((
                                name.clone(),
                                assignment.get(name).ok_or("missing observer")?.clone(),
                            ))
                        })
                        .collect::<Checked<_>>()?,
                };
                let edge = observers::EdgeObservers::decode(&trace)?;
                edges.insert(Edge {
                    before,
                    action: edge.action.into(),
                    choice: edge.choice,
                    after,
                    outcome: edge.outcome.clone(),
                    facts: edge.facts.to_vec(),
                });
            }
            _ => return Err("missing extraction step marker".into()),
        }
    }
    println!("tlc raw_assignments={}", extracted.raw_state_count);
    if starts.is_empty() || states != ordinary.states || edges.len() > EDGES {
        return Err("incomplete or over-cap TLC extraction".into());
    }
    println!(
        "graph starts={} states={} edges={} verdicts={:?}",
        starts.len(),
        states.len(),
        edges.len(),
        verdicts
    );
    Ok(Snapshot {
        starts,
        states,
        edges,
        verdicts,
    })
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
    let (body, _) = module
        .tla
        .rsplit_once("====")
        .ok_or("missing module terminator")?;
    let tla = format!(
        "{body}\n{declarations}\nSpikeInit == {}\nSpikeNext == {}\n====\n",
        init.join(" /\\ "),
        next.join(" /\\ ")
    );
    Ok((
        tla,
        umpire_tlc::Schema {
            fields,
            observers: BTreeMap::new(),
        },
    ))
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
    M::State: Values,
    M::Outcome: IntoValue,
    M::Fact: IntoValue,
{
    use umpire::dsl::__lifted::expr::{Expr, ExprKind, Op};
    let identity = |action: &M::Action| {
        umpire::dsl::ActionValues::<M>::action_value(action)
            .0
            .into()
    };
    let compiled = native(&world(), identity)?;
    let healthy_interpreter = umpire_checker::Interpreted::new(world())?;
    let left_refusal = native_with_cap(&world(), identity, 1)
        .err()
        .ok_or("compiled cap refusal absent")?;
    let right_refusal = native_with_cap(&healthy_interpreter, identity, 1)
        .err()
        .ok_or("interpreted cap refusal absent")?;
    if left_refusal.to_string() != right_refusal.to_string() {
        return Err("refusal control reasons differ".into());
    }
    println!("matching_backend_refusal={left_refusal}");
    if agree(Err(left_refusal), Err(right_refusal)).is_ok() {
        return Err("matching backend refusals passed".into());
    }
    let original = M::__umpire_lifted()
        .model
        .map_err(|e| format!("generator refusal: {e:?}"))?;
    fn swap(expr: &mut Expr) -> bool {
        if let ExprKind::Binary(op @ Op::Lt, _, _) = &mut expr.kind {
            *op = Op::Gt;
            return true;
        }
        expr.kind.children_mut().any(swap)
    }
    let mut changed = original.clone();
    let planted = changed
        .actions
        .iter_mut()
        .flat_map(|a| &mut a.branches)
        .any(|branch| swap(&mut branch.guard));
    if !planted {
        return Err("comparison control was not planted".into());
    }
    let interpreter = umpire_checker::Interpreted::from_tree(world(), &changed)?;
    let swapped = native(&interpreter, identity)?;
    if agree(Ok(compiled.clone()), Ok(swapped)).is_ok() {
        return Err("swapped comparison survived".into());
    }
    let mut changed = original.clone();
    for branch in changed.actions.iter_mut().flat_map(|a| &mut a.branches) {
        branch.facts.kind = ExprKind::Sequence(Vec::new());
    }
    let interpreter = umpire_checker::Interpreted::from_tree(world(), &changed)?;
    let mutated = native(&interpreter, identity)?;
    if compiled.states != mutated.states || compiled.verdicts != mutated.verdicts {
        return Err("fact-only control changed states or verdicts".into());
    }
    if let Ok(folder) = std::env::var("FN255_EVIDENCE") {
        std::fs::write(
            Path::new(&folder).join("fact-mutant.txt"),
            format!("{mutated:#?}"),
        )?;
        std::fs::write(
            Path::new(&folder).join("control-original.txt"),
            format!("{compiled:#?}"),
        )?;
    }
    if agree(Ok(compiled.clone()), Ok(mutated)).is_ok() {
        return Err("fact-only mutation survived".into());
    }
    let mut outcome_tree = original;
    let mut planted_outcome = false;
    for branch in outcome_tree
        .actions
        .iter_mut()
        .flat_map(|a| &mut a.branches)
    {
        if let ExprKind::Enum(outcome) = &mut branch.outcome.kind
            && outcome == "Outcome::Rejected(NotFound)"
        {
            *outcome = "Outcome::Rejected(Unavailable)".into();
            planted_outcome = true;
        }
    }
    if !planted_outcome {
        return Err("outcome mutation was not planted".into());
    }
    let interpreter = umpire_checker::Interpreted::from_tree(world(), &outcome_tree)?;
    let mutated = native(&interpreter, identity)?;
    if compiled.states != mutated.states
        || compiled.verdicts != mutated.verdicts
        || agree(Ok(compiled), Ok(mutated)).is_ok()
    {
        return Err("outcome-only control failed".into());
    }
    println!("controls outcome_only=caught states_equal=true verdicts_equal=true");
    println!(
        "controls comparison=caught fact_only=caught states_equal=true verdicts_equal=true refusal=rejected"
    );
    controls()
}

pub fn batch(root: &Path, modules: &[Module], expected: &[Snapshot]) -> Checked<()> {
    if modules.len() != expected.len() || modules.is_empty() || modules.len() > 4 {
        return Err("invalid batch size".into());
    }
    let mut fields = modules[0].schema()?.fields;
    fields.extend(modules[0].schema()?.observers);
    let shared: Vec<_> = fields.keys().cloned().collect();
    let mut tla = String::from("---- MODULE SpikeBatch ----\nEXTENDS Integers, TLC, Sequences\n");
    let mut initial = Vec::new();
    let mut next = Vec::new();
    let mut invariant = Vec::new();
    for (index, module) in modules.iter().enumerate() {
        if module.schema()?.fields != modules[0].schema()?.fields
            || module.schema()?.observers != modules[0].schema()?.observers
        {
            return Err("batch schemas differ".into());
        }
        let nested = module.tla.replacen(
            &format!("MODULE {}", module.name),
            &format!("MODULE SpikeCase{index}"),
            1,
        );
        tla.push_str(&nested);
        tla.push('\n');
        initial.push(format!("(spike_case = {index} /\\ C{index}!Init)"));
        next.push(format!(
            "(spike_case = {index} /\\ C{index}!Edge /\\ UNCHANGED spike_case)"
        ));
        for claim in &module.metadata.invariants {
            invariant.push(format!(
                "(spike_case = {index} => C{index}!{})",
                claim.operator
            ));
        }
    }
    tla.push_str(&format!(
        "VARIABLES {}, spike_case, spike_has_step\n",
        shared.join(", ")
    ));
    for field in &modules[0].metadata.fields {
        tla.push_str(&format!("VARIABLE spike_before_{}\n", field.name));
        fields.insert(format!("spike_before_{}", field.name), field.ty.clone());
    }
    fields.insert(
        "spike_case".into(),
        Type::Int {
            min: 0,
            max: modules.len() as i128 - 1,
        },
    );
    fields.insert("spike_has_step".into(), Type::Bool);
    for index in 0..modules.len() {
        tla.push_str(&format!("C{index} == INSTANCE SpikeCase{index}\n"));
    }
    let before_init: Vec<_> = modules[0]
        .metadata
        .fields
        .iter()
        .map(|f| format!("spike_before_{} = {}", f.name, f.name))
        .collect();
    let before_next: Vec<_> = modules[0]
        .metadata
        .fields
        .iter()
        .map(|f| format!("spike_before_{}' = {}", f.name, f.name))
        .collect();
    tla.push_str(&format!(
        "SpikeInit == ({}) /\\ spike_has_step = FALSE /\\ {}\n",
        initial.join(" \\/ "),
        before_init.join(" /\\ ")
    ));
    tla.push_str(&format!(
        "SpikeNext == ({}) /\\ spike_has_step' = TRUE /\\ {}\n",
        next.join(" \\/ "),
        before_next.join(" /\\ ")
    ));
    tla.push_str(&format!(
        "BatchInvariant == {}\n====\n",
        invariant.join(" /\\ ")
    ));
    if let Ok(folder) = std::env::var("FN255_EVIDENCE") {
        std::fs::write(Path::new(&folder).join("batch.tla"), &tla)?;
    }
    let mut tlc = umpire_tlc::Tlc::discover(root)?;
    tlc.property_workers = 1;
    tlc.limits.timeout = Duration::from_secs(10);
    tlc.limits.output_bytes = 8 * 1024 * 1024;
    tlc.limits.dump_bytes = 16 * 1024 * 1024;
    let result = tlc.check(
        "SpikeBatch",
        &tla,
        "INIT SpikeInit\nNEXT SpikeNext\nCHECK_DEADLOCK TRUE\nINVARIANT BatchInvariant\n",
        &umpire_tlc::Schema {
            fields,
            observers: BTreeMap::new(),
        },
    )?;
    if result.raw_state_count > 290 * modules.len()
        || result
            .properties
            .get("BatchInvariant")
            .is_none_or(|p| p.verdict != umpire_tlc::Verdict::Verified)
    {
        return Err("batch incomplete, over cap or refuted".into());
    }
    for (index, (module, expected)) in modules.iter().zip(expected).enumerate() {
        let mut snapshot = Snapshot {
            starts: BTreeSet::new(),
            states: BTreeSet::new(),
            edges: BTreeSet::new(),
            verdicts: module
                .metadata
                .invariants
                .iter()
                .map(|c| (c.name.clone(), "Verified".into()))
                .collect(),
        };
        for assignment in result
            .states
            .iter()
            .filter(|s| s.get("spike_case") == Some(&Value::Int(index as i128)))
        {
            let after: State = module
                .metadata
                .fields
                .iter()
                .map(|f| {
                    Ok((
                        f.name.clone(),
                        assignment
                            .get(&f.name)
                            .ok_or("batch authored field missing")?
                            .clone(),
                    ))
                })
                .collect::<Checked<_>>()?;
            snapshot.states.insert(after.clone());
            match assignment.get("spike_has_step") {
                Some(Value::Bool(false)) => {
                    snapshot.starts.insert(after);
                }
                Some(Value::Bool(true)) => {
                    let before = module
                        .metadata
                        .fields
                        .iter()
                        .map(|f| {
                            Ok((
                                f.name.clone(),
                                assignment
                                    .get(&format!("spike_before_{}", f.name))
                                    .ok_or("batch predecessor missing")?
                                    .clone(),
                            ))
                        })
                        .collect::<Checked<_>>()?;
                    let trace = umpire_tla::values::TraceState {
                        state: after.clone(),
                        observers: module
                            .metadata
                            .observer_fields
                            .iter()
                            .map(|n| {
                                Ok((
                                    n.clone(),
                                    assignment.get(n).ok_or("batch observer missing")?.clone(),
                                ))
                            })
                            .collect::<Checked<_>>()?,
                    };
                    let edge = observers::EdgeObservers::decode(&trace)?;
                    snapshot.edges.insert(Edge {
                        before,
                        action: edge.action.into(),
                        choice: edge.choice,
                        after,
                        outcome: edge.outcome.clone(),
                        facts: edge.facts.to_vec(),
                    });
                }
                _ => return Err("batch extraction marker missing".into()),
            }
        }
        agree(Ok(expected.clone()), Ok(snapshot.clone()))?;
        println!(
            "batch_case={index} starts={} states={} edges={} complete",
            snapshot.starts.len(),
            snapshot.states.len(),
            snapshot.edges.len()
        );
        if let Ok(folder) = std::env::var("FN255_EVIDENCE") {
            std::fs::write(
                Path::new(&folder).join(format!("batch-{index}.txt")),
                format!("{snapshot:#?}"),
            )?;
        }
    }
    println!(
        "batch_models={} raw_assignments={} verified",
        modules.len(),
        result.raw_state_count
    );
    Ok(())
}
