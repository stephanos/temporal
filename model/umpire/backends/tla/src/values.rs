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
    /// An optional bounded integer, encoded as zero or payload plus one.
    OptionalInt {
        /// Inclusive raw payload lower bound.
        low: i128,
        /// Inclusive raw payload upper bound.
        high: i128,
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

impl Value {
    /// An admitted value of `ty` on the wire: enum variants by their qualified name, unit and
    /// tuples as sequences.
    ///
    /// # Panics
    /// If `value` is not a value of `ty`, as [`Literal::admit`](crate::Literal::admit) ensures.
    #[must_use]
    pub fn admitted(value: &umpire_front::value::Value, ty: &crate::Type) -> Self {
        use crate::Type as Source;
        use umpire_front::value::Value as Admitted;
        match (value, ty) {
            (Admitted::Bool(value), _) => Self::Bool(*value),
            (Admitted::Int(value), _) => Self::Int(*value),
            (Admitted::Text(value), _) => Self::Text(value.clone()),
            (Admitted::Enum(index), Source::Enum { variants, .. }) if *index < variants.len() => {
                Self::Text(variants[*index].clone())
            }
            (Admitted::Unit, _) => Self::Seq(Vec::new()),
            (Admitted::Tuple(values), Source::Tuple(types)) if values.len() == types.len() => {
                Self::Seq(
                    types.iter().zip(values).map(|(ty, value)| Self::admitted(value, ty)).collect(),
                )
            }
            (Admitted::Seq(values), Source::Seq(inner)) => {
                Self::Seq(values.iter().map(|value| Self::admitted(value, inner)).collect())
            }
            _ => panic!("{value:?} is not a value of {ty:?}"),
        }
    }
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
        self.validate_types()
    }

    fn validate_types(&self) -> Result<(), Refusal> {
        for (name, kind) in self.fields.iter().chain(&self.observers) {
            if !kind.valid() {
                return Err(Refusal(format!("invalid declared type for {name}")));
            }
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
        self.validate_types()
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
    fn valid(&self) -> bool {
        match self {
            Self::Int { min, max } => min <= max,
            Self::OptionalInt { low, high } => {
                *low >= 0 && low <= high && *high < i128::from(i32::MAX)
            }
            Self::Seq(inner) | Self::Set(inner) => inner.valid(),
            Self::Tuple(types) => types.iter().all(Self::valid),
            Self::Record(fields) => fields.values().all(Self::valid),
            Self::Bool | Self::Text | Self::Enum(_) => true,
        }
    }
    /// Whether a structurally decoded value inhabits this exact declared type.
    pub fn accepts(&self, value: &Value) -> bool {
        match (self, value) {
            (Self::Bool, Value::Bool(_)) | (Self::Text, Value::Text(_)) => true,
            (Self::Int { min, max }, Value::Int(number)) => (*min..=*max).contains(number),
            (Self::OptionalInt { low, high }, Value::Int(number)) => {
                *low >= 0
                    && low <= high
                    && *high < i128::from(i32::MAX)
                    && (*number == 0 || (*low + 1..=*high + 1).contains(number))
            }
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
