use std::collections::{HashMap, HashSet};

use umpire_dsl::{
    __lifted::{Lifted, Type, value::Value},
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

pub(super) fn world_values(
    model: &Lifted,
    values: &[(&'static str, Exported)],
) -> Result<HashMap<String, Value>, String> {
    let mut seen = HashSet::new();
    for (name, _) in values {
        if !seen.insert(name) {
            return Err(format!("duplicate World field `{name}`"));
        }
        if !model.world_fields.iter().any(|field| field.name == *name) {
            return Err(format!("undeclared World field `{name}`"));
        }
    }
    let mut world = HashMap::new();
    for field in &model.world_fields {
        let (_, value) = values
            .iter()
            .find(|(name, _)| *name == field.name)
            .ok_or_else(|| format!("the World exports no field {}", field.name))?;
        let imported = import(&field.ty, value)
            .map_err(|error| format!("World field {}: {error}", field.name))?;
        if let (Type::Int { min, max }, Value::Int(raw)) = (&field.ty, &imported)
            && (raw < min || raw > max)
        {
            return Err(format!(
                "World field {} value is outside its admitted integer range",
                field.name
            ));
        }
        world.insert(field.name.clone(), imported);
    }
    Ok(world)
}

#[cfg(test)]
mod world_validation_tests {
    use super::*;

    #[umpire_dsl::machine]
    mod exported {
        use umpire_dsl::{Action, State, World};
        #[derive(World)]
        pub(super) struct World {
            maximum: u8,
            enabled: bool,
        }
        #[derive(State)]
        pub(super) struct Model {
            done: bool,
        }
        #[derive(Action, Copy)]
        pub(super) enum User {
            Complete,
        }
        impl Model {
            #[on(User)]
            fn user(&mut self, action: User) {
                match action {
                    User::Complete => self.done = true,
                }
            }
        }
    }

    #[test]
    fn primitive_world_import_refuses_missing_duplicate_extra_kinds_and_ranges()
    -> Result<(), Box<dyn std::error::Error>> {
        use umpire_dsl::__lifted::Lifts;
        let model =
            exported::World::__umpire_lifted().model.map_err(|error| format!("{error:?}"))?;
        let valid = vec![("maximum", Exported::Int(1)), ("enabled", Exported::Bool(true))];
        assert!(world_values(&model, &valid).is_ok());
        for mutation in ["missing", "duplicate", "undeclared", "mistyped", "range"] {
            let mut values = valid.clone();
            match mutation {
                "missing" => {
                    values.pop();
                }
                "duplicate" => values.push(values[0].clone()),
                "undeclared" => values.push(("extra", Exported::Int(1))),
                "mistyped" => values[0].1 = Exported::Bool(true),
                _ => values[0].1 = Exported::Int(256),
            }
            let error = world_values(&model, &values).err().ok_or(mutation)?;
            assert!(error.contains("World"), "{mutation}: {error}");
        }
        Ok(())
    }
}
