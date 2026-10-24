use std::collections::{BTreeMap, BTreeSet};

use umpire_tla::{Module, Type as LiftType};

use crate::{Refusal, Schema, Type, runner::split_cfg};

pub(crate) fn schema(module: &Module) -> Result<Schema, Refusal> {
    identifier(&module.name)?;
    for parameter in &module.metadata.parameters {
        backend_type(&parameter.ty, 0)?;
    }
    let mut fields = BTreeMap::new();
    for field in &module.metadata.fields {
        identifier(&field.name)?;
        if fields.insert(field.name.clone(), backend_type(&field.ty, 0)?).is_some() {
            return Err(Refusal(format!("duplicate authored field {}", field.name)));
        }
    }
    let mut observers = BTreeSet::new();
    for name in &module.metadata.observer_fields {
        identifier(name)?;
        if !observers.insert(name) {
            return Err(Refusal(format!("duplicate observer {name}")));
        }
    }
    if observers != module.metadata.observer_types.keys().collect() {
        return Err(Refusal("observer declarations differ".into()));
    }
    let observers = module
        .metadata
        .observer_types
        .iter()
        .map(|(name, ty)| Ok((name.clone(), backend_type(ty, 0)?)))
        .collect::<Result<BTreeMap<_, _>, Refusal>>()?;
    let schema = Schema { fields, observers };
    schema.validate()?;
    validate_properties(module)?;
    Ok(schema)
}

fn backend_type(ty: &LiftType, depth: usize) -> Result<Type, Refusal> {
    if depth >= 64 {
        return Err(Refusal("type declaration nesting exceeds backend limit".into()));
    }
    Ok(match ty {
        LiftType::Bool => Type::Bool,
        LiftType::Int { min, max } => {
            if min > max {
                return Err(Refusal("invalid declared integer bounds".into()));
            }
            Type::Int { min: *min, max: *max }
        }
        LiftType::Bounded(bound) => {
            identifier(&bound.name)?;
            if let Some(field) = &bound.field {
                identifier(field)?;
            }
            if !bound.valid() {
                return Err(Refusal("invalid declared bounded integer".into()));
            }
            Type::Int { min: bound.low, max: bound.high }
        }
        LiftType::Optional(bound) => {
            identifier(&bound.name)?;
            if let Some(field) = &bound.field {
                identifier(field)?;
            }
            if !bound.optional_valid() {
                return Err(Refusal("invalid declared optional integer".into()));
            }
            Type::OptionalInt { low: bound.low, high: bound.high }
        }
        LiftType::Text => Type::Text,
        LiftType::Enum { name, variants } => {
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
            Type::Enum(variants.clone())
        }
        LiftType::Unit => Type::Tuple(Vec::new()),
        LiftType::Tuple(types) => Type::Tuple(
            types.iter().map(|ty| backend_type(ty, depth + 1)).collect::<Result<_, _>>()?,
        ),
        LiftType::Seq(inner) => Type::Seq(Box::new(backend_type(inner, depth + 1)?)),
    })
}

/// A variant name, or a variant carrying one fieldless variant, such as `Rejected(NotFound)`.
fn variant(name: &str) -> Result<(), Refusal> {
    match name.strip_suffix(')').and_then(|name| name.split_once('(')) {
        Some((variant, payload)) => identifier(variant).and_then(|()| identifier(payload)),
        None => identifier(name),
    }
}

fn identifier(name: &str) -> Result<(), Refusal> {
    let mut bytes = name.bytes();
    if !bytes.next().is_some_and(|byte| byte.is_ascii_alphabetic() || byte == b'_')
        || !bytes.all(|byte| byte.is_ascii_alphanumeric() || byte == b'_')
    {
        return Err(Refusal(format!("invalid declared TLA+ identifier {name:?}")));
    }
    Ok(())
}

fn validate_properties(module: &Module) -> Result<(), Refusal> {
    let metadata = &module.metadata;
    let properties = metadata
        .invariants
        .iter()
        .map(|claim| {
            ("INVARIANT", claim.name.as_str(), claim.operator.as_str(), claim.cfg.as_str())
        })
        .chain(metadata.postconditions.iter().map(|claim| {
            ("PROPERTY", claim.name.as_str(), claim.operator.as_str(), claim.cfg.as_str())
        }))
        .chain(metadata.negative.iter().map(|claim| {
            ("INVARIANT", claim.name.as_str(), claim.operator.as_str(), claim.cfg.as_str())
        }))
        .chain(metadata.progress.iter().map(|claim| {
            ("PROPERTY", claim.name.as_str(), claim.operator.as_str(), claim.cfg.as_str())
        }));
    let mut names = BTreeSet::new();
    let mut operators = BTreeSet::new();
    for (kind, name, operator, cfg) in properties {
        identifier(operator)?;
        if name.is_empty() || !names.insert(name) || !operators.insert(operator) {
            return Err(Refusal(format!("duplicate property declaration {name:?}/{operator}")));
        }
        let (_, selected) = split_cfg(cfg)?;
        if selected != [(kind.to_owned(), operator.to_owned())] {
            return Err(Refusal(format!("isolated property declarations differ for {name}")));
        }
    }
    let (_, selected) = split_cfg(&module.cfg)?;
    let expected = metadata
        .invariants
        .iter()
        .map(|claim| ("INVARIANT".into(), claim.operator.clone()))
        .chain(
            metadata.postconditions.iter().map(|claim| ("PROPERTY".into(), claim.operator.clone())),
        )
        .collect::<Vec<_>>();
    if selected != expected {
        return Err(Refusal("module property declarations differ from registered claims".into()));
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn nominal_schema_validates_descriptor_and_retains_exact_code_gap() {
        let bound = umpire_dsl::__lifted::Bounded {
            name: "Ticks".into(),
            field: None,
            backing: "u8".into(),
            min: 0,
            max: 255,
            low: 1,
            high: 3,
        };
        let raw = backend_type(&LiftType::Bounded(bound.clone()), 0);
        assert_eq!(raw.ok(), Some(Type::Int { min: 1, max: 3 }));
        let optional = backend_type(&LiftType::Optional(bound.clone()), 0);
        assert_eq!(optional.ok(), Some(Type::OptionalInt { low: 1, high: 3 }));
        for field in ["backing", "range", "name", "member", "width"] {
            let mut bad = bound.clone();
            match field {
                "backing" => bad.backing = "u32".into(),
                "range" => bad.low = 4,
                "name" => bad.name = "not a name".into(),
                "member" => bad.field = Some("not a member".into()),
                _ => bad.max = 256,
            }
            assert!(backend_type(&LiftType::Bounded(bad.clone()), 0).is_err());
            assert!(backend_type(&LiftType::Optional(bad), 0).is_err());
        }
        let mut signed = bound;
        signed.backing = "i32".into();
        signed.min = i128::from(i32::MIN);
        signed.max = i128::from(i32::MAX);
        for (low, high) in [(-1, 3), (0, i128::from(i32::MAX))] {
            signed.low = low;
            signed.high = high;
            assert!(backend_type(&LiftType::Bounded(signed.clone()), 0).is_ok());
            assert!(backend_type(&LiftType::Optional(signed.clone()), 0).is_err());
        }
    }

    #[test]
    fn enum_variants_admit_one_fieldless_payload_only() {
        assert!(backend_type(&LiftType::outcome(), 0).is_ok());
        for variant in ["Rejected()", "Rejected(Not Found)", "Rejected(A)(B)", "(A)", "Rejected(A"]
        {
            let ty = LiftType::Enum {
                name: "Outcome".into(),
                variants: vec![format!("Outcome::{variant}")],
            };
            assert!(backend_type(&ty, 0).is_err(), "{variant}");
        }
    }
}
