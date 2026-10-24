//! Shared typed assignments admitted independently of the backend wire format.
use std::collections::{BTreeMap, BTreeSet};

use crate::Refusal;

/// A complete assignment to explicitly declared fields.
pub type State = BTreeMap<String, Value>;

/// Value kinds admitted by the source emitter.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum Type {
    /// Boolean.
    Bool,
    /// An integer within its authored Rust type's inclusive bounds.
    Int {
        /// Inclusive lower bound.
        min: i128,
        /// Inclusive upper bound.
        max: i128,
    },
    /// A string.
    Text,
    /// One declared enum variant, encoded as a string.
    Enum(Vec<String>),
    /// Homogeneous sequence.
    Seq(Box<Self>),
    /// Heterogeneous tuple, including unit.
    Tuple(Vec<Self>),
    /// Unordered set.
    Set(Box<Self>),
    /// Named fields.
    Record(BTreeMap<String, Self>),
}

/// A decoded backend value with deterministic equality and ordering.
#[derive(Clone, Debug, PartialEq, Eq, PartialOrd, Ord)]
pub enum Value {
    /// Boolean.
    Bool(bool),
    /// Integer.
    Int(i128),
    /// String or admitted enum variant.
    Text(String),
    /// Sequence or tuple.
    Seq(Vec<Self>),
    /// Set.
    Set(BTreeSet<Self>),
    /// Record.
    Record(State),
}

/// The exact model and private observer fields. Prefix-based projection is forbidden.
#[derive(Clone, Debug, Default)]
pub struct Schema {
    /// Authored state fields.
    pub fields: BTreeMap<String, Type>,
    /// Private fields explicitly introduced by this module.
    pub observers: BTreeMap<String, Type>,
}

/// A trace assignment before private observer projection.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct TraceState {
    /// Authored state.
    pub state: State,
    /// Exactly the declared private observers.
    pub observers: State,
}

impl Schema {
    /// Validate disjoint declarations with at least one authored state field.
    pub fn validate(&self) -> Result<(), Refusal> {
        if self.fields.keys().any(|name| self.observers.contains_key(name)) {
            return Err(Refusal("authored field collides with a declared observer".into()));
        }
        if self.fields.is_empty() {
            return Err(Refusal("empty state schema".into()));
        }
        Ok(())
    }

    /// Validate typed assignments, including safety modules with only declared observers.
    pub fn validate_assignment(&self) -> Result<(), Refusal> {
        if self.fields.keys().any(|name| self.observers.contains_key(name)) {
            return Err(Refusal("authored field collides with a declared observer".into()));
        }
        if self.fields.is_empty() && self.observers.is_empty() {
            return Err(Refusal("empty assignment schema".into()));
        }
        Ok(())
    }

    /// Validate and split a complete typed assignment, independent of its wire format.
    pub fn project_assignment(&self, mut assignment: State) -> Result<TraceState, Refusal> {
        self.validate_assignment()?;
        let mut state = State::new();
        let mut observers = State::new();
        for (fields, target) in [(&self.fields, &mut state), (&self.observers, &mut observers)] {
            for (name, kind) in fields {
                let value = assignment
                    .remove(name)
                    .ok_or_else(|| Refusal(format!("missing dump field {name}")))?;
                if !kind.accepts(&value) {
                    return Err(Refusal(format!(
                        "dump field {name} has wrong type or integer range: {value:?}"
                    )));
                }
                target.insert(name.clone(), value);
            }
        }
        if let Some(name) = assignment.keys().next() {
            return Err(Refusal(format!("undeclared dump field {name}")));
        }
        Ok(TraceState { state, observers })
    }
}

impl Type {
    /// Whether a structurally decoded value inhabits this exact declared type.
    pub fn accepts(&self, value: &Value) -> bool {
        match (self, value) {
            (Self::Bool, Value::Bool(_)) | (Self::Text, Value::Text(_)) => true,
            (Self::Int { min, max }, Value::Int(number)) => (*min..=*max).contains(number),
            (Self::Enum(variants), Value::Text(value)) => variants.contains(value),
            (Self::Seq(kind), Value::Seq(values)) => values.iter().all(|value| kind.accepts(value)),
            (Self::Tuple(kinds), Value::Seq(values)) => {
                kinds.len() == values.len()
                    && kinds.iter().zip(values).all(|(kind, value)| kind.accepts(value))
            }
            (Self::Set(kind), Value::Set(values)) => values.iter().all(|value| kind.accepts(value)),
            (Self::Record(kinds), Value::Record(values)) => {
                kinds.len() == values.len()
                    && kinds.iter().all(|(name, kind)| {
                        values.get(name).is_some_and(|value| kind.accepts(value))
                    })
            }
            _ => false,
        }
    }
}
