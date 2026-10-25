//! Values a compiled Model exports for binding, and their one admission into an admitted type.
use std::fmt;

use crate::{Type, value::Value};

/// A value projected from compiler-checked authored fields, before admission into a type.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum Literal {
    /// Boolean value.
    Bool(bool),
    /// Integer checked against the independently admitted field type.
    Int(i128),
    /// Nominal integer with its authored wrapper identity.
    Bounded {
        /// Authored wrapper name.
        name: String,
        /// Raw backing integer.
        raw: i128,
    },
    /// Absent or present nominal payload.
    Optional(Option<Box<Literal>>),
    /// Qualified variant of an independently admitted fieldless enum.
    Enum(String),
    /// Ordered elements of a sequence, a tuple or unit.
    Sequence(Vec<Literal>),
}

/// One concrete catalog action: its qualified variant and ordered named payload values.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Concrete {
    /// Qualified authored variant name.
    pub action: String,
    /// Payload fields in declaration order; tuple fields are named by position.
    pub fields: Vec<(String, Literal)>,
}

/// One selected World and its complete authored start states.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Binding {
    /// Named World fields.
    pub world: Vec<(String, Literal)>,
    /// Complete named state fields for each authored start, in enumeration order.
    pub starts: Vec<Vec<(String, Literal)>>,
    /// The checked native catalog of the same World, in catalog order.
    pub actions: Vec<Concrete>,
}

/// A literal outside the type it was admitted as.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Mismatch {
    /// The refused literal.
    pub literal: Box<Literal>,
    /// The type it does not inhabit.
    pub ty: Box<Type>,
}
impl fmt::Display for Mismatch {
    fn fmt(&self, out: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(out, "{:?} is not a value of {:?}", self.literal, self.ty)
    }
}
impl std::error::Error for Mismatch {}

impl Literal {
    /// This literal as a value of `ty`: integers within their range, nominal values of the same
    /// wrapper within its domain, optional values as their exact code, enum variants by
    /// position, and sequences as unit, tuples of the same length or sequences.
    ///
    /// # Errors
    /// Refuses a literal that does not inhabit `ty`, naming the innermost refused part.
    pub fn admit(&self, ty: &Type) -> Result<Value, Mismatch> {
        let refuse = || Mismatch { literal: Box::new(self.clone()), ty: Box::new(ty.clone()) };
        Ok(match (self, ty) {
            (Self::Bool(value), Type::Bool) => Value::Bool(*value),
            (Self::Int(value), Type::Int { min, max }) if (*min..=*max).contains(value) => {
                Value::Int(*value)
            }
            (Self::Bounded { name, raw }, Type::Bounded(bounded))
                if *name == bounded.name && bounded.contains(*raw) =>
            {
                Value::Int(*raw)
            }
            (Self::Optional(None), Type::Optional(bounded)) if bounded.optional_valid() => {
                Value::Int(0)
            }
            (Self::Optional(Some(payload)), Type::Optional(bounded)) => match payload.as_ref() {
                Self::Bounded { name, raw } if *name == bounded.name => {
                    Value::Int(bounded.encode(*raw).ok_or_else(refuse)?)
                }
                _ => return Err(refuse()),
            },
            (Self::Enum(name), Type::Enum { variants, .. }) => {
                Value::Enum(variants.iter().position(|variant| variant == name).ok_or_else(refuse)?)
            }
            (Self::Sequence(values), Type::Unit) if values.is_empty() => Value::Unit,
            (Self::Sequence(values), Type::Tuple(types)) if values.len() == types.len() => {
                Value::Tuple(
                    values
                        .iter()
                        .zip(types)
                        .map(|(value, ty)| value.admit(ty))
                        .collect::<Result<_, _>>()?,
                )
            }
            (Self::Sequence(values), Type::Seq(inner)) => {
                Value::Seq(values.iter().map(|value| value.admit(inner)).collect::<Result<_, _>>()?)
            }
            _ => return Err(refuse()),
        })
    }
}

#[cfg(test)]
mod tests {
    use super::{Literal, Mismatch};
    use crate::{Bounded, Type, value::Value};

    fn ticks(low: i128) -> Bounded {
        Bounded {
            name: "Ticks".into(),
            field: None,
            backing: "u8".into(),
            min: 0,
            max: 255,
            low,
            high: 3,
        }
    }
    fn raw(name: &str, raw: i128) -> Literal {
        Literal::Bounded { name: name.into(), raw }
    }
    fn some(value: Literal) -> Literal {
        Literal::Optional(Some(Box::new(value)))
    }
    fn refused(literal: &Literal, ty: &Type) -> bool {
        literal.admit(ty).is_err()
    }

    #[test]
    fn nominal_and_optional_values_admit_exactly() {
        let ty = Type::Bounded(ticks(1));
        assert_eq!(raw("Ticks", 1).admit(&ty), Ok(Value::Int(1)));
        for value in [raw("Other", 1), raw("Ticks", 0), raw("Ticks", 4), Literal::Int(1)] {
            assert!(refused(&value, &ty), "{value:?}");
        }
        let ty = Type::Optional(ticks(1));
        for (value, code) in
            [(Literal::Optional(None), 0), (some(raw("Ticks", 1)), 2), (some(raw("Ticks", 3)), 4)]
        {
            assert_eq!(value.admit(&ty), Ok(Value::Int(code)));
        }
        for value in [
            some(raw("Other", 1)),
            some(raw("Ticks", 0)),
            some(raw("Ticks", 4)),
            some(Literal::Int(1)),
            some(Literal::Optional(None)),
            some(some(raw("Ticks", 1))),
            raw("Ticks", 1),
            Literal::Int(2),
        ] {
            assert!(refused(&value, &ty), "{value:?}");
        }
        assert_eq!(some(raw("Ticks", 0)).admit(&Type::Optional(ticks(0))), Ok(Value::Int(1)));
        let mut invalid = ticks(1);
        invalid.max = 256;
        assert!(refused(&raw("Ticks", 1), &Type::Bounded(invalid.clone())));
        assert!(refused(&Literal::Optional(None), &Type::Optional(invalid)));
    }

    #[test]
    fn integers_enums_and_sequences_refuse_outside_their_type() {
        let byte = Type::Int { min: 0, max: 255 };
        assert_eq!(Literal::Int(255).admit(&byte), Ok(Value::Int(255)));
        for value in [Literal::Int(-1), Literal::Int(256), Literal::Bool(true)] {
            assert!(refused(&value, &byte), "{value:?}");
        }
        let phase = Type::Enum {
            name: "Phase".into(),
            variants: vec!["Phase::Open".into(), "Phase::Closed".into()],
        };
        assert_eq!(Literal::Enum("Phase::Closed".into()).admit(&phase), Ok(Value::Enum(1)));
        assert!(refused(&Literal::Enum("Phase::Gone".into()), &phase));
        assert!(refused(&Literal::Enum("Other::Open".into()), &phase));
        assert_eq!(Literal::Sequence(vec![]).admit(&Type::Unit), Ok(Value::Unit));
        assert!(refused(&Literal::Sequence(vec![Literal::Bool(true)]), &Type::Unit));
        let pair = Type::Tuple(vec![Type::Bool, byte.clone()]);
        assert_eq!(
            Literal::Sequence(vec![Literal::Bool(true), Literal::Int(2)]).admit(&pair),
            Ok(Value::Tuple(vec![Value::Bool(true), Value::Int(2)]))
        );
        for value in [
            Literal::Sequence(vec![Literal::Bool(true)]),
            Literal::Sequence(vec![Literal::Bool(true), Literal::Int(2), Literal::Int(3)]),
            Literal::Sequence(vec![Literal::Int(2), Literal::Bool(true)]),
        ] {
            assert!(refused(&value, &pair), "{value:?}");
        }
        let bytes = Type::Seq(Box::new(byte));
        assert_eq!(
            Literal::Sequence(vec![Literal::Int(1), Literal::Int(2)]).admit(&bytes),
            Ok(Value::Seq(vec![Value::Int(1), Value::Int(2)]))
        );
        let error = Literal::Sequence(vec![Literal::Int(1), Literal::Int(300)])
            .admit(&bytes)
            .err()
            .map(|error: Mismatch| error.to_string());
        assert_eq!(error.as_deref(), Some("Int(300) is not a value of Int { min: 0, max: 255 }"));
        assert!(refused(&Literal::Bool(false), &Type::Text));
    }
}
