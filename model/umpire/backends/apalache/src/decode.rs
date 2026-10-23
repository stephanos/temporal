//! Bounded conversion of pinned ITF wire values into the shared typed assignment contract.
use serde::de::{self, Deserialize, Deserializer, MapAccess, SeqAccess, Visitor};
use std::{
    collections::{BTreeMap, BTreeSet},
    fmt,
};
use umpire_tlc::{Refusal, Schema, State, TraceState, Type, Value};

const BYTES: usize = 4 * 1024 * 1024;
const ITEMS: usize = 100_000;
const DEPTH: usize = 64;

/// A complete ITF trace with fixed, explicitly declared World parameters.
#[derive(Clone, Debug)]
pub struct Itf {
    /// Selected World parameters, constant throughout the trace.
    pub parameters: State,
    /// Complete typed states and exact emitted observers.
    pub states: Vec<TraceState>,
}

// An object visitor must see keys before a JSON map could silently overwrite them.
struct Json(serde_json::Value);
impl<'de> Deserialize<'de> for Json {
    fn deserialize<D: Deserializer<'de>>(decoder: D) -> Result<Self, D::Error> {
        struct Strict;
        impl<'de> Visitor<'de> for Strict {
            type Value = Json;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("strict JSON")
            }
            fn visit_bool<E: de::Error>(self, v: bool) -> Result<Json, E> {
                Ok(Json(v.into()))
            }
            fn visit_i64<E: de::Error>(self, v: i64) -> Result<Json, E> {
                Ok(Json(v.into()))
            }
            fn visit_u64<E: de::Error>(self, v: u64) -> Result<Json, E> {
                Ok(Json(v.into()))
            }
            fn visit_f64<E: de::Error>(self, _: f64) -> Result<Json, E> {
                Err(E::custom("ITF floating point value"))
            }
            fn visit_str<E: de::Error>(self, v: &str) -> Result<Json, E> {
                Ok(Json(v.into()))
            }
            fn visit_unit<E: de::Error>(self) -> Result<Json, E> {
                Err(E::custom("ITF null value"))
            }
            fn visit_seq<A: SeqAccess<'de>>(self, mut seq: A) -> Result<Json, A::Error> {
                let mut values = Vec::new();
                while let Some(Json(value)) = seq.next_element()? {
                    if values.len() >= ITEMS {
                        return Err(de::Error::custom("ITF collection limit"));
                    }
                    values.push(value);
                }
                Ok(Json(values.into()))
            }
            fn visit_map<A: MapAccess<'de>>(self, mut map: A) -> Result<Json, A::Error> {
                let mut values = serde_json::Map::new();
                while let Some(key) = map.next_key::<String>()? {
                    if values.len() >= ITEMS || values.contains_key(&key) {
                        return Err(de::Error::custom("ITF duplicate key or collection limit"));
                    }
                    let Json(value) = map.next_value()?;
                    values.insert(key, value);
                }
                Ok(Json(values.into()))
            }
        }
        decoder.deserialize_any(Strict)
    }
}

fn refuse(message: impl Into<String>) -> Refusal {
    Refusal(format!("ITF: {}", message.into()))
}
fn object(v: &serde_json::Value) -> Result<&serde_json::Map<String, serde_json::Value>, Refusal> {
    v.as_object().ok_or_else(|| refuse("expected object"))
}
fn array(v: &serde_json::Value) -> Result<&[serde_json::Value], Refusal> {
    v.as_array().map(Vec::as_slice).ok_or_else(|| refuse("expected array"))
}
fn get<'a>(v: &'a serde_json::Value, key: &str) -> Result<&'a serde_json::Value, Refusal> {
    object(v)?.get(key).ok_or_else(|| refuse(format!("missing {key}")))
}
fn keys(v: &serde_json::Value, required: &[&str], optional: &[&str]) -> Result<(), Refusal> {
    let map = object(v)?;
    if required.iter().any(|key| !map.contains_key(*key))
        || map
            .keys()
            .any(|key| !required.contains(&key.as_str()) && !optional.contains(&key.as_str()))
    {
        return Err(refuse("missing or undeclared object field"));
    }
    Ok(())
}
fn names(v: &serde_json::Value) -> Result<BTreeSet<&str>, Refusal> {
    let values = array(v)?;
    let names: BTreeSet<_> = values
        .iter()
        .map(|v| v.as_str().ok_or_else(|| refuse("non-string declaration")))
        .collect::<Result<_, _>>()?;
    if names.len() != values.len() {
        return Err(refuse("duplicate declaration"));
    }
    Ok(names)
}

/// Decode a pinned-release witness, rejecting undeclared data and excessive input before parsing.
pub fn decode_itf(
    raw: &str,
    schema: &Schema,
    parameters: &BTreeMap<String, Type>,
    length: usize,
) -> Result<Itf, Refusal> {
    budget(raw)?;
    schema.validate_assignment()?;
    if parameters
        .keys()
        .any(|key| schema.fields.contains_key(key) || schema.observers.contains_key(key))
    {
        return Err(refuse("parameter collides with variable"));
    }
    let Json(root) = serde_json::from_str::<Json>(raw).map_err(|e| refuse(e.to_string()))?;
    keys(&root, &["#meta", "vars", "states"], &["params"])?;
    let meta = get(&root, "#meta")?;
    keys(meta, &["format"], &["format-description", "description", "varTypes", "version"])?;
    if get(meta, "format")?.as_str() != Some("ITF") {
        return Err(refuse("unknown format"));
    }
    if object(meta)?.contains_key("version") {
        return Err(refuse("pinned ITF declares no version"));
    }
    for key in ["format-description", "description"] {
        if let Some(v) = meta.get(key)
            && !v.is_string()
        {
            return Err(refuse("invalid metadata text"));
        }
    }
    let expected: BTreeSet<_> =
        schema.fields.keys().chain(schema.observers.keys()).map(String::as_str).collect();
    if names(get(&root, "vars")?)? != expected {
        return Err(refuse("variable declarations differ"));
    }
    if let Some(types) = meta.get("varTypes")
        && (object(types)?.keys().map(String::as_str).collect::<BTreeSet<_>>() != expected
            || schema.fields.iter().chain(&schema.observers).any(|(name, kind)| {
                types.get(name).and_then(serde_json::Value::as_str)
                    != Some(wire_type(kind).as_str())
            }))
    {
        return Err(refuse("variable type declarations differ"));
    }
    let declared = root.get("params").map(names).transpose()?.unwrap_or_default();
    if declared != parameters.keys().map(String::as_str).collect() {
        return Err(refuse("parameter declarations differ"));
    }
    let rows = array(get(&root, "states")?)?;
    if rows.is_empty() || rows.len() > 10_000 || rows.len().saturating_sub(1) > length {
        return Err(refuse("state count outside bound"));
    }
    let mut fixed = None;
    let mut states = Vec::new();
    for (index, row) in rows.iter().enumerate() {
        let meta = get(row, "#meta")?;
        keys(meta, &["index"], &[])?;
        if get(meta, "index")?.as_u64().and_then(|i| usize::try_from(i).ok()) != Some(index) {
            return Err(refuse("nonconsecutive state positions"));
        }
        let mut assignment = State::new();
        let mut selected = State::new();
        for (name, raw) in object(row)? {
            if name == "#meta" {
                continue;
            }
            let kind = parameters
                .get(name)
                .or_else(|| schema.fields.get(name))
                .or_else(|| schema.observers.get(name))
                .ok_or_else(|| refuse(format!("undeclared state field {name}")))?;
            let value = value(raw, kind)?;
            if parameters.contains_key(name) {
                selected.insert(name.clone(), value);
            } else {
                assignment.insert(name.clone(), value);
            }
        }
        if selected.len() != parameters.len() {
            return Err(refuse("missing World parameter"));
        }
        if fixed.as_ref().is_some_and(|p| p != &selected) {
            return Err(refuse("World parameter changed"));
        }
        fixed = Some(selected);
        states.push(schema.project_assignment(assignment)?);
    }
    Ok(Itf { parameters: fixed.unwrap_or_default(), states })
}

fn wire_type(kind: &Type) -> String {
    match kind {
        Type::Bool => "Bool".into(),
        Type::Int { .. } => "Int".into(),
        Type::Text | Type::Enum(_) => "Str".into(),
        Type::Tuple(types) if types.is_empty() => "Seq(Int)".into(),
        Type::Tuple(types) => {
            format!("<<{}>>", types.iter().map(wire_type).collect::<Vec<_>>().join(", "))
        }
        Type::Seq(inner) => format!("Seq({})", wire_type(inner)),
        Type::Set(inner) => format!("Set({})", wire_type(inner)),
        Type::Record(fields) => format!(
            "{{ {} }}",
            fields
                .iter()
                .map(|(name, kind)| format!("{name}: {}", wire_type(kind)))
                .collect::<Vec<_>>()
                .join(", ")
        ),
    }
}

fn budget(raw: &str) -> Result<(), Refusal> {
    if raw.len() > BYTES {
        return Err(refuse("byte limit"));
    }
    let (mut depth, mut items, mut string, mut escaped) = (0usize, 0usize, false, false);
    for b in raw.bytes() {
        if string {
            if escaped {
                escaped = false;
            } else if b == b'\\' {
                escaped = true;
            } else if b == b'"' {
                string = false;
            }
        } else {
            match b {
                b'"' => {
                    string = true;
                    items += 1;
                }
                b'[' | b'{' => {
                    depth += 1;
                    items += 1;
                    if depth > DEPTH {
                        return Err(refuse("nesting limit"));
                    }
                }
                b']' | b'}' => depth = depth.saturating_sub(1),
                b',' | b':' => items += 1,
                _ => {}
            }
            if items > ITEMS {
                return Err(refuse("aggregate item limit"));
            }
        }
    }
    Ok(())
}
fn value(raw: &serde_json::Value, kind: &Type) -> Result<Value, Refusal> {
    let converted = match kind {
        Type::Bool => Value::Bool(raw.as_bool().ok_or_else(|| refuse("expected bool"))?),
        Type::Text | Type::Enum(_) => {
            Value::Text(raw.as_str().ok_or_else(|| refuse("expected text/enum"))?.into())
        }
        Type::Int { .. } => {
            keys(raw, &["#bigint"], &[])?;
            let s = get(raw, "#bigint")?.as_str().ok_or_else(|| refuse("bigint must be text"))?;
            if s.is_empty()
                || s.bytes().enumerate().any(|(i, b)| !(b.is_ascii_digit() || i == 0 && b == b'-'))
            {
                return Err(refuse("invalid bigint syntax"));
            }
            let n = s.parse::<i128>().map_err(|_| refuse("integer range"))?;
            Value::Int(n)
        }
        Type::Seq(inner) => {
            Value::Seq(array(raw)?.iter().map(|v| value(v, inner)).collect::<Result<_, _>>()?)
        }
        Type::Tuple(types) => {
            // Apalache encodes the empty/unit tuple as [] and nonempty tuples with #tup.
            let values = if types.is_empty() {
                array(raw)?
            } else {
                keys(raw, &["#tup"], &[])?;
                array(get(raw, "#tup")?)?
            };
            if types.len() != values.len() {
                return Err(refuse("tuple arity"));
            }
            Value::Seq(
                values.iter().zip(types).map(|(v, t)| value(v, t)).collect::<Result<_, _>>()?,
            )
        }
        Type::Set(inner) => {
            keys(raw, &["#set"], &[])?;
            let values = array(get(raw, "#set")?)?;
            let set: BTreeSet<_> =
                values.iter().map(|v| value(v, inner)).collect::<Result<_, _>>()?;
            if set.len() != values.len() {
                return Err(refuse("duplicate set member"));
            }
            Value::Set(set)
        }
        Type::Record(types) => {
            if object(raw)?.len() != types.len() {
                return Err(refuse("record fields differ"));
            }
            Value::Record(
                types
                    .iter()
                    .map(|(n, t)| Ok((n.clone(), value(get(raw, n)?, t)?)))
                    .collect::<Result<_, Refusal>>()?,
            )
        }
    };
    if !kind.accepts(&converted) {
        return Err(refuse("value differs from source type or integer range"));
    }
    Ok(converted)
}
