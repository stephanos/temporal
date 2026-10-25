//! Shared typed assignments admitted independently of the backend wire format.
use std::collections::{BTreeMap, BTreeSet};

use crate::{Refusal, Type, identifier};

/// A complete assignment to explicitly declared fields.
pub type State = BTreeMap<String, Value>;

/// A decoded backend value with deterministic equality and ordering.
#[derive(Clone, Debug, PartialEq, Eq, PartialOrd, Ord)]
pub enum Value {
    /// Boolean.
    Bool(bool),
    /// Integer, including nominal payloads and optional codes.
    Int(i128),
    /// String or admitted enum variant.
    Text(String),
    /// Sequence, tuple or unit.
    Seq(Vec<Self>),
}

impl Value {
    /// An admitted value of `ty` on the wire: enum variants by their qualified name, unit and
    /// tuples as sequences.
    ///
    /// # Panics
    /// If `value` is not a value of `ty`, as [`Literal::admit`](crate::Literal::admit) ensures.
    #[must_use]
    pub fn admitted(value: &umpire_front::value::Value, ty: &Type) -> Self {
        use Type as Source;
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
            validate_type(kind).map_err(|Refusal(error)| {
                Refusal(format!("invalid declared type for {name}: {error}"))
            })?;
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
                if !accepts(kind, &value) {
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

/// Whether a structurally decoded value inhabits this exact declared type: integers within
/// their range, nominal payloads within their domain, optional values as an exact code, enum
/// variants by qualified name, and unit and tuples as sequences.
pub fn accepts(ty: &Type, value: &Value) -> bool {
    match (ty, value) {
        (Type::Bool, Value::Bool(_)) | (Type::Text, Value::Text(_)) => true,
        (Type::Int { min, max }, Value::Int(number)) => (*min..=*max).contains(number),
        (Type::Bounded(bounded), Value::Int(raw)) => bounded.contains(*raw),
        (Type::Optional(bounded), Value::Int(code)) => bounded.decode(*code).is_some(),
        (Type::Enum { variants, .. }, Value::Text(value)) => variants.contains(value),
        (Type::Unit, Value::Seq(values)) => values.is_empty(),
        (Type::Seq(inner), Value::Seq(values)) => values.iter().all(|value| accepts(inner, value)),
        (Type::Tuple(types), Value::Seq(values)) => {
            types.len() == values.len()
                && types.iter().zip(values).all(|(ty, value)| accepts(ty, value))
        }
        _ => false,
    }
}

/// Refuses a declared type that no backend can carry: inverted integer bounds, an invalid
/// nominal descriptor, enum variants that are empty, repeated or not qualified identifiers,
/// or nesting past the backend limit.
pub fn validate_type(ty: &Type) -> Result<(), Refusal> {
    nested(ty, 0)
}

fn nested(ty: &Type, depth: usize) -> Result<(), Refusal> {
    if depth >= 64 {
        return Err(Refusal("type declaration nesting exceeds backend limit".into()));
    }
    match ty {
        Type::Bool | Type::Text | Type::Unit => {}
        Type::Int { min, max } => {
            if min > max {
                return Err(Refusal("invalid declared integer bounds".into()));
            }
        }
        Type::Bounded(bound) | Type::Optional(bound) => {
            identifier(&bound.name)?;
            if let Some(field) = &bound.field {
                identifier(field)?;
            }
            if matches!(ty, Type::Bounded(_)) && !bound.valid() {
                return Err(Refusal("invalid declared bounded integer".into()));
            }
            if matches!(ty, Type::Optional(_)) && !bound.optional_valid() {
                return Err(Refusal("invalid declared optional integer".into()));
            }
        }
        Type::Enum { name, variants } => {
            identifier(name)?;
            let prefix = format!("{name}::");
            if variants.is_empty()
                || variants.iter().collect::<BTreeSet<_>>().len() != variants.len()
                || variants.iter().any(|variant| {
                    variant
                        .strip_prefix(&prefix)
                        .is_none_or(|variant| self::variant(variant).is_err())
                })
            {
                return Err(Refusal(format!("invalid declared enum variants for {name}")));
            }
        }
        Type::Tuple(types) => {
            for ty in types {
                nested(ty, depth + 1)?;
            }
        }
        Type::Seq(inner) => nested(inner, depth + 1)?,
    }
    Ok(())
}

/// A variant name, or a variant carrying one fieldless variant, such as `Rejected(NotFound)`.
fn variant(name: &str) -> Result<(), Refusal> {
    match name.strip_suffix(')').and_then(|name| name.split_once('(')) {
        Some((variant, payload)) => identifier(variant).and_then(|()| identifier(payload)),
        None => identifier(name),
    }
}

#[cfg(test)]
mod tests {
    use super::{Value, accepts, validate_type};
    use crate::Type;
    use umpire_front::Bounded;

    fn ticks() -> Bounded {
        Bounded {
            name: "Ticks".into(),
            field: None,
            backing: "u8".into(),
            min: 0,
            max: 255,
            low: 1,
            high: 3,
        }
    }

    #[test]
    fn nominal_types_validate_descriptor_and_accept_exact_codes() {
        let bound = ticks();
        assert!(validate_type(&Type::Bounded(bound.clone())).is_ok());
        assert!(validate_type(&Type::Optional(bound.clone())).is_ok());
        for raw in -1..=5 {
            assert_eq!(
                accepts(&Type::Bounded(bound.clone()), &Value::Int(raw)),
                (1..=3).contains(&raw)
            );
            assert_eq!(
                accepts(&Type::Optional(bound.clone()), &Value::Int(raw)),
                [0, 2, 3, 4].contains(&raw),
                "{raw}"
            );
        }
        for field in ["backing", "range", "name", "member", "width"] {
            let mut bad = bound.clone();
            match field {
                "backing" => bad.backing = "u32".into(),
                "range" => bad.low = 4,
                "name" => bad.name = "not a name".into(),
                "member" => bad.field = Some("not a member".into()),
                _ => bad.max = 256,
            }
            assert!(validate_type(&Type::Bounded(bad.clone())).is_err(), "{field}");
            assert!(validate_type(&Type::Optional(bad)).is_err(), "{field}");
        }
        let mut signed = bound;
        signed.backing = "i32".into();
        signed.min = i128::from(i32::MIN);
        signed.max = i128::from(i32::MAX);
        for (low, high) in [(-1, 3), (0, i128::from(i32::MAX))] {
            signed.low = low;
            signed.high = high;
            assert!(validate_type(&Type::Bounded(signed.clone())).is_ok());
            assert!(validate_type(&Type::Optional(signed.clone())).is_err());
        }
    }

    #[test]
    fn enum_variants_admit_one_fieldless_payload_only() {
        assert!(validate_type(&Type::outcome()).is_ok());
        for variant in ["Rejected()", "Rejected(Not Found)", "Rejected(A)(B)", "(A)", "Rejected(A"]
        {
            let ty = Type::Enum {
                name: "Outcome".into(),
                variants: vec![format!("Outcome::{variant}")],
            };
            assert!(validate_type(&ty).is_err(), "{variant}");
        }
        let empty = Type::Enum { name: "Empty".into(), variants: Vec::new() };
        assert!(validate_type(&empty).is_err());
        assert!(validate_type(&Type::Int { min: 3, max: 2 }).is_err());
    }

    #[test]
    fn composite_values_accept_exact_shapes_only() {
        let pair = Type::Tuple(vec![Type::Bool, Type::Int { min: 0, max: 2 }]);
        assert!(accepts(&pair, &Value::Seq(vec![Value::Bool(true), Value::Int(2)])));
        for value in [
            Value::Seq(vec![Value::Bool(true)]),
            Value::Seq(vec![Value::Bool(true), Value::Int(3)]),
            Value::Seq(vec![Value::Int(2), Value::Bool(true)]),
            Value::Text("pair".into()),
        ] {
            assert!(!accepts(&pair, &value), "{value:?}");
        }
        assert!(accepts(&Type::Unit, &Value::Seq(Vec::new())));
        assert!(!accepts(&Type::Unit, &Value::Seq(vec![Value::Bool(true)])));
        let outcome = Type::outcome();
        assert!(accepts(&outcome, &Value::Text("Outcome::Accepted".into())));
        assert!(!accepts(&outcome, &Value::Text("Outcome::Lost".into())));
        let nested = Type::Seq(Box::new(Type::Optional(ticks())));
        assert!(accepts(&nested, &Value::Seq(vec![Value::Int(0), Value::Int(4)])));
        assert!(!accepts(&nested, &Value::Seq(vec![Value::Int(1)])));
        let mut deep = Type::Bool;
        for _ in 0..64 {
            deep = Type::Seq(Box::new(deep));
        }
        assert!(validate_type(&deep).is_err());
    }
}
