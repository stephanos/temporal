use umpire_dsl::{
    __lifted::{Type, value::Value},
    Value as Exported,
};

/// An exported Model value as the interpreter's value of its admitted type.
pub(super) fn import(ty: &Type, value: &Exported) -> Result<Value, String> {
    Ok(match (ty, value) {
        (Type::Bool, Exported::Bool(value)) => Value::Bool(*value),
        (Type::Int { .. }, Exported::Int(value)) => Value::Int(*value),
        (Type::Bounded(bounded), Exported::Bounded { name, raw })
            if bounded.name == *name && bounded.contains(*raw) =>
        {
            Value::Int(*raw)
        }
        (Type::Optional(bounded), Exported::Optional(None)) if bounded.optional_valid() => {
            Value::Int(0)
        }
        (Type::Optional(bounded), Exported::Optional(Some(payload))) => {
            let Value::Int(raw) = import(&Type::Bounded(bounded.clone()), payload)? else {
                unreachable!("a bounded value imports as an integer")
            };
            Value::Int(
                bounded.encode(raw).ok_or_else(|| format!("{value:?} is not a value of {ty:?}"))?,
            )
        }
        (Type::Enum { variants, .. }, Exported::Enum(name)) => Value::Enum(
            variants
                .iter()
                .position(|variant| variant == name)
                .ok_or_else(|| format!("{name} is not a variant of {ty:?}"))?,
        ),
        (Type::Unit, Exported::Sequence(values)) if values.is_empty() => Value::Unit,
        (Type::Seq(inner), Exported::Sequence(values)) => {
            Value::Seq(values.iter().map(|value| import(inner, value)).collect::<Result<_, _>>()?)
        }
        _ => return Err(format!("{value:?} is not a value of {ty:?}")),
    })
}
