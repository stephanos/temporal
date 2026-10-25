use std::collections::{BTreeMap, BTreeSet};

use umpire_tla::{Literal, Module, Type, identifier};

use crate::{Refusal, Schema, Value, runner::split_cfg};

/// The module's decoding contract, after its World binding and property declarations agree
/// with its configuration.
pub(crate) fn schema(module: &Module) -> Result<Schema, Refusal> {
    let schema = module.schema()?;
    validate_binding(module)?;
    validate_properties(module)?;
    Ok(schema)
}

fn binding_value(ty: &Type, literal: &Literal) -> Result<Value, Refusal> {
    let admitted = literal.admit(ty).map_err(|error| {
        Refusal(format!("selected World literal differs from its declared type: {error}"))
    })?;
    let value = Value::admitted(&admitted, ty);
    if matches!(&value, Value::Int(raw) if !(0..=i128::from(i32::MAX)).contains(raw)) {
        return Err(Refusal("selected World value is outside its declared CFG domain".into()));
    }
    Ok(value)
}

fn validate_binding(module: &Module) -> Result<(), Refusal> {
    let parameters = &module.metadata.parameters;
    let constants = constant_bindings(&module.cfg)?;
    if parameters.is_empty() {
        if !constants.is_empty()
            || module.metadata.binding.as_ref().is_some_and(|binding| !binding.world.is_empty())
        {
            return Err(Refusal("unexpected selected World bindings".into()));
        }
        return Ok(());
    }
    let binding = module
        .metadata
        .binding
        .as_ref()
        .ok_or_else(|| Refusal("missing selected World binding".into()))?;
    if binding.world.len() != parameters.len() {
        return Err(Refusal("selected World fields differ from parameter declarations".into()));
    }
    let mut fields = BTreeMap::new();
    let mut expected = BTreeMap::new();
    let mut names = BTreeSet::new();
    for parameter in parameters {
        identifier(&parameter.symbol)?;
        if !names.insert(&parameter.field)
            || fields.insert(parameter.symbol.clone(), parameter.ty.clone()).is_some()
        {
            return Err(Refusal("duplicate selected World parameter".into()));
        }
        let literal = binding
            .world
            .iter()
            .find(|(name, _)| name == &parameter.field)
            .map(|(_, value)| value)
            .ok_or_else(|| Refusal(format!("missing selected World field {}", parameter.field)))?;
        expected.insert(parameter.symbol.clone(), binding_value(&parameter.ty, literal)?);
    }
    let mut assignment = String::new();
    for constant in constants {
        let body = constant
            .strip_prefix("CONSTANT ")
            .ok_or_else(|| Refusal("invalid selected World assignment".into()))?;
        assignment.push_str("/\\ ");
        assignment.push_str(body);
        assignment.push('\n');
    }
    let schema = Schema { fields, observers: BTreeMap::new() };
    if crate::decode_state(&assignment, &schema)?.state != expected {
        return Err(Refusal("selected World bindings differ in main CFG".into()));
    }
    Ok(())
}

fn constant_bindings(cfg: &str) -> Result<Vec<&str>, Refusal> {
    let mut constants = Vec::new();
    for line in cfg.lines().map(str::trim).filter(|line| !line.is_empty()) {
        let (keyword, value) = line
            .split_once(char::is_whitespace)
            .ok_or_else(|| Refusal(format!("noncanonical config line: {line}")))?;
        match keyword {
            "CONSTANT" => constants.push(line),
            "SPECIFICATION" | "INIT" | "NEXT" | "INVARIANT" | "PROPERTY" => {
                identifier(value.trim())?;
            }
            "CHECK_DEADLOCK" if value.trim() == "TRUE" => {}
            _ => return Err(Refusal(format!("noncanonical config line: {line}"))),
        }
    }
    Ok(constants)
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
        if constant_bindings(cfg)? != constant_bindings(&module.cfg)? {
            return Err(Refusal(format!(
                "selected World bindings differ in isolated CFG for {name}"
            )));
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
