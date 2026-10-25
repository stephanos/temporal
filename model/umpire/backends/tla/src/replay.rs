//! Typed edge replay independent of native authoring and backend history encodings.
use crate::{
    Concrete, Literal, Module, Refusal, Type,
    observers::EdgeObservers,
    values::{State, TraceState, Value},
};
use std::collections::BTreeMap;

/// The typed projection of one native step.
pub struct EdgeValues {
    /// Complete successor assignment.
    pub state: State,
    /// Exact outcome.
    pub outcome: Value,
    /// Ordered facts, including duplicates.
    pub facts: Vec<Value>,
}
/// The chosen native step and its admitted trace identity.
pub struct ReplayedEdge<'a, A, S> {
    /// The selected native catalog action.
    pub action: &'a A,
    /// Its complete concrete identity.
    pub identity: Concrete,
    /// The emitted input ordinal.
    pub ordinal: usize,
    /// The emitted choice ordinal.
    pub choice: usize,
    /// The native step, evaluated once.
    pub step: S,
    /// Its typed projection.
    pub values: EdgeValues,
}
/// Native evaluation failures stay available to the backend's arithmetic policy.
#[derive(Debug)]
pub enum ReplayError<E> {
    /// The trace differs from the native execution.
    Divergence(Refusal),
    /// Native execution refused or panicked.
    Native(E),
}
impl<E> From<Refusal> for ReplayError<E> {
    fn from(error: Refusal) -> Self {
        Self::Divergence(error)
    }
}

/// Select an exact catalog action, execute it once and compare the chosen typed edge.
/// Native callbacks keep DSL and checker dependencies out of this module.
///
/// # Errors
/// Refuses malformed identities, disabled choices and typed successor/outcome/fact differences.
pub fn step<'a, A, S, E, I: IntoIterator<Item = S>>(
    module: &Module,
    after: &TraceState,
    actions: &'a [A],
    identity: impl Fn(&A) -> Option<Concrete>,
    execute: impl FnOnce(&A) -> Result<I, E>,
    project: impl FnOnce(&S) -> Result<EdgeValues, Refusal>,
) -> Result<ReplayedEdge<'a, A, S>, ReplayError<E>> {
    let edge = EdgeObservers::decode(after)?;
    let (action, concrete, ordinal) = action(module, &edge, actions, identity)?;
    if !crate::values::accepts(&module.metadata.outcome_type, edge.outcome) {
        return Err(Refusal("outcome does not match authored type".into()).into());
    }
    if let Some(Type::Seq(fact_type)) = module.metadata.observer_types.get(crate::observers::FACTS)
    {
        if edge.facts.iter().any(|fact| !crate::values::accepts(fact_type, fact)) {
            return Err(Refusal("facts do not match authored type".into()).into());
        }
    } else {
        return Err(Refusal("missing facts observer type".into()).into());
    }
    let steps = execute(action).map_err(ReplayError::Native)?;
    let native = steps.into_iter().nth(edge.choice).ok_or_else(|| {
        Refusal("trace choice is not emitted by Model (disabled action or unknown choice)".into())
    })?;
    let values = project(&native)?;
    for field in &module.metadata.fields {
        if values.state.get(&field.name) != after.state.get(&field.name) {
            return Err(Refusal(format!("native successor differs at field {}", field.name)).into());
        }
    }
    if values.state.len() != after.state.len() {
        return Err(Refusal("native successor fields differ".into()).into());
    }
    if values.outcome != *edge.outcome {
        return Err(Refusal("native outcome/facts differ at outcome".into()).into());
    }
    if values.facts != edge.facts {
        return Err(Refusal("native outcome/facts differ at facts".into()).into());
    }
    Ok(ReplayedEdge {
        action,
        identity: concrete,
        ordinal,
        choice: edge.choice,
        step: native,
        values,
    })
}
fn action<'a, A>(
    module: &Module,
    edge: &EdgeObservers<'_>,
    actions: &'a [A],
    identity: impl Fn(&A) -> Option<Concrete>,
) -> Result<(&'a A, Concrete, usize), Refusal> {
    let catalog = if edge.input.is_some() {
        module
            .metadata
            .binding
            .as_ref()
            .map_or(module.metadata.catalog.as_slice(), |b| b.actions.as_slice())
    } else {
        module.metadata.catalog.as_slice()
    };
    let ordinal = edge.input.unwrap_or(0);
    let action = if catalog.is_empty() {
        if ordinal != 0 {
            return Err(Refusal("unit action ordinal is not zero".into()));
        }
        actions
            .iter()
            .find(|a| identity(a).is_some_and(|a| a.action == edge.action))
            .ok_or_else(|| Refusal("action is not in Model catalog".into()))?
    } else {
        edge.input.ok_or_else(|| Refusal("counterexample lacks input identity".into()))?;
        let entry = catalog
            .get(ordinal)
            .ok_or_else(|| Refusal("input ordinal is outside the catalog".into()))?;
        if entry.action != edge.action {
            return Err(Refusal(format!(
                "input ordinal {ordinal} is {}, not the edge's action {}",
                entry.action, edge.action
            )));
        }
        let declaration = module
            .metadata
            .actions
            .iter()
            .find(|a| a.name == entry.action)
            .ok_or_else(|| Refusal("input names an undeclared action".into()))?;
        if entry.fields.len() != declaration.fields.len()
            || entry
                .fields
                .iter()
                .zip(&declaration.fields)
                .any(|((name, _), field)| *name != field.name)
        {
            return Err(Refusal(format!(
                "input ordinal {ordinal} does not match its declared field order"
            )));
        }
        fields(entry.fields.clone(), declaration.fields.iter().map(|f| (f.name.as_str(), &f.ty)))
            .map_err(|e| {
            Refusal(format!("input ordinal {ordinal} does not match its declared fields: {e}"))
        })?;
        let action = actions
            .get(ordinal)
            .ok_or_else(|| Refusal("input ordinal is outside the native catalog".into()))?;
        let concrete = identity(action)
            .ok_or_else(|| Refusal("an input catalog requires the typed native identity".into()))?;
        if concrete != *entry {
            return Err(Refusal(format!(
                "native catalog ordinal {ordinal} differs from the emitted input"
            )));
        }
        if catalog.iter().position(|entry| entry == &concrete) != Some(ordinal) {
            return Err(Refusal("input ordinal is not the emitted first concrete identity".into()));
        }
        action
    };
    let concrete =
        identity(action).ok_or_else(|| Refusal("missing native action identity".into()))?;
    Ok((action, concrete, ordinal))
}

/// Admit one exported literal into its typed wire representation.
///
/// # Errors
/// Returns its type mismatch.
pub fn admitted(value: impl Into<Literal>, ty: &Type) -> Result<Value, Refusal> {
    let admitted = value.into().admit(ty).map_err(|e| Refusal(e.to_string()))?;
    Ok(Value::admitted(&admitted, ty))
}
/// Admit exactly the declared fields, retaining each field's refusal context.
///
/// # Errors
/// Refuses unknown, duplicate, missing or mistyped fields.
pub fn fields<'a, N: AsRef<str>, V: Into<Literal>>(
    values: impl IntoIterator<Item = (N, V)>,
    types: impl Iterator<Item = (&'a str, &'a Type)>,
) -> Result<State, Refusal> {
    let types: BTreeMap<_, _> = types.collect();
    let mut result = State::new();
    for (name, value) in values {
        let name = name.as_ref();
        let ty =
            types.get(name).ok_or_else(|| Refusal(format!("undeclared native field: {name}")))?;
        let value = admitted(value, ty)
            .map_err(|e| Refusal(format!("typed native field differs: {name}: {e}")))?;
        if result.insert(name.into(), value).is_some() {
            return Err(Refusal(format!("duplicate native field: {name}")));
        }
    }
    if let Some(name) = types.keys().find(|name| !result.contains_key(**name)) {
        return Err(Refusal(format!("missing native field: {name}")));
    }
    Ok(result)
}
