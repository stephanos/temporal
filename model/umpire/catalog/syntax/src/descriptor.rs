//! Independent coverage checks against the upstream protobuf descriptor set.

use std::collections::{BTreeMap, BTreeSet};

use prost::Message;
use umpire_runtime::google::protobuf::{DescriptorProto, FileDescriptorSet};

/// Descriptor obligations projected from the executable annotation declarations.
#[derive(Clone, Debug, Default)]
pub struct Coverage {
    /// Annotated protobuf messages.
    pub messages: Vec<MessageCoverage>,
    /// Annotated protobuf enums.
    pub enums: Vec<EnumCoverage>,
    /// Annotated protobuf services.
    pub services: Vec<ServiceCoverage>,
}

/// Every protobuf source field of a named message, including oneof alternatives.
#[derive(Clone, Debug)]
pub struct MessageCoverage {
    /// Fully qualified protobuf name.
    pub name: String,
    /// Original protobuf field names.
    pub fields: Vec<String>,
    /// Explicitly ignored opaque fields exempt from nested type obligations.
    pub opaque_fields: Vec<String>,
}

/// Every known value of a named protobuf enum.
#[derive(Clone, Debug)]
pub struct EnumCoverage {
    /// Fully qualified protobuf name.
    pub name: String,
    /// Original protobuf enum value names.
    pub values: Vec<String>,
}

/// Explicit dispositions for every method of a named service.
#[derive(Clone, Debug)]
pub struct ServiceCoverage {
    /// Fully qualified protobuf name.
    pub name: String,
    /// Method declarations.
    pub methods: Vec<MethodCoverage>,
}

/// One RPC and its declared relationship to the Model.
#[derive(Clone, Debug)]
pub struct MethodCoverage {
    /// Original protobuf method name.
    pub name: String,
    /// The method's declared role.
    pub disposition: Disposition,
}

/// An RPC's declared relationship to the Model.
#[derive(Clone, Debug)]
pub enum Disposition {
    /// Callable Model action name.
    Action(String),
    /// An observation of behavior.
    Observed,
    /// An intentionally excluded RPC with an explanation.
    OutOfScope(String),
}

/// Refuse omitted, duplicate, stale or unexplained declarations.
pub fn check(bytes: &[u8], coverage: &Coverage) -> Result<(), String> {
    let descriptors = FileDescriptorSet::decode(bytes)
        .map_err(|error| format!("invalid protobuf descriptor set: {error}"))?;
    let mut messages = BTreeMap::new();
    let mut enums = BTreeMap::new();
    let mut services = BTreeMap::new();
    for file in &descriptors.file {
        let package = file.package.as_deref().unwrap_or_default();
        collect_messages(package, &file.message_type, &mut messages, &mut enums);
        for enumeration in &file.enum_type {
            enums.insert(qualified(package, enumeration.name.as_deref()), enumeration);
        }
        for service in &file.service {
            services.insert(qualified(package, service.name.as_deref()), service);
        }
    }
    let mut seen = BTreeSet::new();
    for message in &coverage.messages {
        let name = unique(&mut seen, "message", &message.name)?;
        let descriptor =
            messages.get(name).ok_or_else(|| format!("stale message annotation: {name}"))?;
        exact(
            name,
            &message.fields,
            descriptor.field.iter().map(|field| field.name.as_deref().unwrap_or_default()),
        )?;
        check_nested(message, descriptor, coverage)?;
    }
    seen.clear();
    for enumeration in &coverage.enums {
        let name = unique(&mut seen, "enum", &enumeration.name)?;
        let descriptor = enums.get(name).ok_or_else(|| format!("stale enum annotation: {name}"))?;
        exact(
            name,
            &enumeration.values,
            descriptor.value.iter().map(|value| value.name.as_deref().unwrap_or_default()),
        )?;
    }
    seen.clear();
    for service in &coverage.services {
        let name = unique(&mut seen, "service", &service.name)?;
        let descriptor =
            services.get(name).ok_or_else(|| format!("stale service annotation: {name}"))?;
        let methods: Vec<_> = service.methods.iter().map(|method| method.name.clone()).collect();
        exact(
            name,
            &methods,
            descriptor.method.iter().map(|method| method.name.as_deref().unwrap_or_default()),
        )?;
        for method in &service.methods {
            match &method.disposition {
                Disposition::Action(action) if action.trim().is_empty() => {
                    return Err(format!("missing action for {name}.{}", method.name));
                }
                Disposition::OutOfScope(reason) if reason.trim().is_empty() => {
                    return Err(format!("missing OutOfScope reason for {name}.{}", method.name));
                }
                _ => {}
            }
        }
    }
    Ok(())
}

fn check_nested(
    message: &MessageCoverage,
    descriptor: &DescriptorProto,
    coverage: &Coverage,
) -> Result<(), String> {
    let mut opaque = BTreeSet::new();
    for field in &message.opaque_fields {
        if !opaque.insert(field.as_str()) {
            return Err(format!("duplicate opaque field annotation: {}.{field}", message.name));
        }
        if !message.fields.contains(field) {
            return Err(format!("stale opaque field annotation: {}.{field}", message.name));
        }
    }
    for field in &descriptor.field {
        let name = field.name.as_deref().unwrap_or_default();
        if opaque.contains(name) {
            continue;
        }
        if let Some(target) = &field.type_name {
            let target = target.trim_start_matches('.');
            let covered = coverage
                .messages
                .iter()
                .any(|candidate| candidate.name.trim_start_matches('.') == target)
                || coverage
                    .enums
                    .iter()
                    .any(|candidate| candidate.name.trim_start_matches('.') == target);
            if !covered {
                return Err(format!(
                    "uncovered nested protobuf type: {}.{name} -> {target}",
                    message.name
                ));
            }
        }
    }
    Ok(())
}

fn qualified(parent: &str, name: Option<&str>) -> String {
    let name = name.unwrap_or_default();
    if parent.is_empty() { name.to_owned() } else { format!("{parent}.{name}") }
}

fn collect_messages<'a>(
    parent: &str,
    descriptors: &'a [DescriptorProto],
    messages: &mut BTreeMap<String, &'a DescriptorProto>,
    enums: &mut BTreeMap<String, &'a umpire_runtime::google::protobuf::EnumDescriptorProto>,
) {
    for message in descriptors {
        let name = qualified(parent, message.name.as_deref());
        messages.insert(name.clone(), message);
        for enumeration in &message.enum_type {
            enums.insert(qualified(&name, enumeration.name.as_deref()), enumeration);
        }
        collect_messages(&name, &message.nested_type, messages, enums);
    }
}

fn unique<'a>(seen: &mut BTreeSet<String>, kind: &str, name: &'a str) -> Result<&'a str, String> {
    let name = name.strip_prefix('.').unwrap_or(name);
    if !seen.insert(name.to_owned()) {
        return Err(format!("duplicate {kind} annotation: {name}"));
    }
    Ok(name)
}

fn exact<'a>(
    owner: &str,
    declared: &[String],
    actual: impl Iterator<Item = &'a str>,
) -> Result<(), String> {
    let mut names = BTreeSet::new();
    for name in declared {
        if !names.insert(name.as_str()) {
            return Err(format!("duplicate annotation: {owner}.{name}"));
        }
    }
    let ordered: Vec<_> = actual.collect();
    let actual: BTreeSet<_> = ordered.iter().copied().collect();
    for name in &actual {
        if !names.contains(name) {
            return Err(format!("uncovered protobuf symbol: {owner}.{name}"));
        }
    }
    for name in names {
        if !actual.contains(name) {
            return Err(format!("stale annotation: {owner}.{name}"));
        }
    }
    for (found, expected) in declared.iter().zip(ordered) {
        if found != expected {
            return Err(format!(
                "out-of-order annotation: {owner}.{expected} (found {owner}.{found})"
            ));
        }
    }
    Ok(())
}
