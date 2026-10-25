//! Independent coverage checks against the upstream protobuf descriptor set.

use std::collections::BTreeSet;

use crate::proto::{self, Graph, Kind};

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
    let graph = Graph::new(bytes, "")
        .map_err(|error| format!("invalid protobuf descriptor set: {error}"))?;
    admit(&graph, coverage)
}

/// [`check`] against an already decoded descriptor graph.
pub fn admit(graph: &Graph, coverage: &Coverage) -> Result<(), String> {
    let mut seen = BTreeSet::new();
    for message in &coverage.messages {
        let name = unique(&mut seen, "message", &message.name)?;
        let descriptor =
            graph.messages.get(name).ok_or_else(|| format!("stale message annotation: {name}"))?;
        exact(name, &message.fields, descriptor.fields.iter().map(|field| field.name.as_str()))?;
        check_nested(message, descriptor, coverage)?;
    }
    seen.clear();
    for enumeration in &coverage.enums {
        let name = unique(&mut seen, "enum", &enumeration.name)?;
        let descriptor =
            graph.enums.get(name).ok_or_else(|| format!("stale enum annotation: {name}"))?;
        exact(
            name,
            &enumeration.values,
            descriptor.values.iter().map(|value| value.name.as_str()),
        )?;
    }
    seen.clear();
    for service in &coverage.services {
        let name = unique(&mut seen, "service", &service.name)?;
        let descriptor =
            graph.services.get(name).ok_or_else(|| format!("stale service annotation: {name}"))?;
        let methods: Vec<_> = service.methods.iter().map(|method| method.name.clone()).collect();
        exact(name, &methods, descriptor.iter().map(|method| method.name.as_str()))?;
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
    descriptor: &proto::Message,
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
    for field in &descriptor.fields {
        let name = field.name.as_str();
        if opaque.contains(name) {
            continue;
        }
        let (Kind::Message { target } | Kind::Enum { target }) = &field.kind else {
            continue;
        };
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
    Ok(())
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

/// Native primitive carriers used by prost-build's default well-known-type mappings.
pub fn well_known_primitive(name: &str) -> Option<&'static str> {
    Some(match name {
        "google.protobuf.Empty" => "()",
        "google.protobuf.BoolValue" => "bool",
        "google.protobuf.BytesValue" => "::prost::alloc::vec::Vec<u8>",
        "google.protobuf.DoubleValue" => "f64",
        "google.protobuf.FloatValue" => "f32",
        "google.protobuf.Int32Value" => "i32",
        "google.protobuf.Int64Value" => "i64",
        "google.protobuf.StringValue" => "::prost::alloc::string::String",
        "google.protobuf.UInt32Value" => "u32",
        "google.protobuf.UInt64Value" => "u64",
        _ => return None,
    })
}
