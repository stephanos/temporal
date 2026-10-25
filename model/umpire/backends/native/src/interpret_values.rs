use std::collections::{HashMap, HashSet};

use umpire_dsl::{
    __lifted::{Lifted, Literal, Type, value::Value},
    Value as Exported,
};

/// An exported Model value as the interpreter's value of its admitted type.
pub(super) fn import(ty: &Type, value: &Exported) -> Result<Value, String> {
    Literal::from(value.clone()).admit(ty).map_err(|error| error.to_string())
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
