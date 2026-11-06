//! The relation descriptor subset linked into guests.
//!
//! Relation bindings resolve only their own RPCs, so guests embed the message closure of those
//! methods instead of the whole linked API. Every retained descriptor entry keeps its exact
//! source bytes, including custom options, so admission sees the same declarations.

use std::{
    collections::{BTreeMap, BTreeSet},
    io,
};

/// RPCs that carry production relation bindings.
const METHODS: [(&str, &str); 3] = [
    ("temporal.api.workflowservice.v1.WorkflowService", "StartActivityExecution"),
    ("temporal.api.workflowservice.v1.WorkflowService", "DescribeActivityExecution"),
    ("temporal.api.workflowservice.v1.WorkflowService", "RequestCancelActivityExecution"),
];

// FileDescriptorProto, DescriptorProto, FieldDescriptorProto and ServiceDescriptorProto tags.
const FILE: u64 = 1;
const PACKAGE: u64 = 2;
const MESSAGE: u64 = 4;
const ENUM: u64 = 5;
const SERVICE: u64 = 6;
const NAME: u64 = 1;
const FIELD: u64 = 2;
const NESTED: u64 = 3;
const NESTED_ENUM: u64 = 4;
const TYPE_NAME: u64 = 6;
const METHOD: u64 = 2;
const INPUT: u64 = 2;
const OUTPUT: u64 = 3;

struct Entry<'a> {
    tag: u64,
    wire: u64,
    raw: &'a [u8],
    payload: &'a [u8],
}

fn error(message: &str) -> io::Error {
    io::Error::other(format!("relation descriptor subset: {message}"))
}

fn varint(bytes: &[u8], index: &mut usize) -> io::Result<u64> {
    let mut value = 0u64;
    for shift in (0..64).step_by(7) {
        let byte = *bytes.get(*index).ok_or_else(|| error("truncated varint"))?;
        *index += 1;
        value |= u64::from(byte & 0x7f) << shift;
        if byte < 0x80 {
            return Ok(value);
        }
    }
    Err(error("overlong varint"))
}

fn entries(bytes: &[u8]) -> io::Result<Vec<Entry<'_>>> {
    let mut index = 0;
    let mut out = Vec::new();
    while index < bytes.len() {
        let start = index;
        let key = varint(bytes, &mut index)?;
        let wire = key & 7;
        let payload = match wire {
            0 => {
                varint(bytes, &mut index)?;
                &bytes[index..index]
            }
            1 | 5 => {
                let width = if wire == 1 { 8 } else { 4 };
                index += width;
                bytes.get(index - width..index).ok_or_else(|| error("truncated fixed"))?
            }
            2 => {
                let length = usize::try_from(varint(bytes, &mut index)?)
                    .map_err(|_| error("oversized field"))?;
                index += length;
                bytes.get(index - length..index).ok_or_else(|| error("truncated field"))?
            }
            _ => return Err(error("unsupported wire type")),
        };
        out.push(Entry { tag: key >> 3, wire, raw: &bytes[start..index], payload });
    }
    Ok(out)
}

fn text(entries: &[Entry<'_>], tag: u64) -> io::Result<String> {
    entries
        .iter()
        .find(|entry| entry.tag == tag && entry.wire == 2)
        .map(|entry| String::from_utf8(entry.payload.to_vec()).map_err(|_| error("invalid name")))
        .transpose()
        .map(Option::unwrap_or_default)
}

fn length_delimited(tag: u64, payload: &[u8], out: &mut Vec<u8>) {
    let mut put = |mut value: u64| {
        while value >= 0x80 {
            out.push(u8::try_from(value & 0x7f).unwrap_or_default() | 0x80);
            value >>= 7;
        }
        out.push(u8::try_from(value).unwrap_or_default());
    };
    put(tag << 3 | 2);
    put(payload.len() as u64);
    out.extend_from_slice(payload);
}

/// Index every message (top-level owner) and enum by fully qualified name.
#[derive(Default)]
struct Index {
    owners: BTreeMap<String, (usize, String)>,
    references: BTreeMap<String, BTreeSet<String>>,
    enums: BTreeMap<String, usize>,
}

impl Index {
    fn message(&mut self, file: usize, top: &str, scope: &str, message: &[u8]) -> io::Result<()> {
        let fields = entries(message)?;
        let name = format!("{scope}.{}", text(&fields, NAME)?);
        let top = if top.is_empty() { name.clone() } else { top.to_owned() };
        self.owners.insert(name.clone(), (file, top.clone()));
        for entry in &fields {
            match entry.tag {
                FIELD if entry.wire == 2 => {
                    let field = entries(entry.payload)?;
                    let referenced = text(&field, TYPE_NAME)?;
                    if !referenced.is_empty() {
                        self.references.entry(top.clone()).or_default().insert(referenced);
                    }
                }
                NESTED if entry.wire == 2 => self.message(file, &top, &name, entry.payload)?,
                NESTED_ENUM if entry.wire == 2 => {
                    let nested = format!("{name}.{}", text(&entries(entry.payload)?, NAME)?);
                    self.enums.insert(nested, file);
                }
                _ => {}
            }
        }
        Ok(())
    }
}

/// Retain the descriptor entries reachable from [`METHODS`], byte for byte.
pub(crate) fn subset(linked: &[u8]) -> io::Result<Vec<u8>> {
    let files = entries(linked)?;
    let mut index = Index::default();
    let mut roots = Vec::new();
    for (position, file) in files.iter().enumerate() {
        if file.tag != FILE {
            return Err(error("unexpected descriptor set field"));
        }
        let fields = entries(file.payload)?;
        let package = text(&fields, PACKAGE)?;
        let scope = if package.is_empty() { String::new() } else { format!(".{package}") };
        for entry in &fields {
            match entry.tag {
                MESSAGE => index.message(position, "", &scope, entry.payload)?,
                ENUM => {
                    let name = format!("{scope}.{}", text(&entries(entry.payload)?, NAME)?);
                    index.enums.insert(name, position);
                }
                SERVICE => {
                    let service = entries(entry.payload)?;
                    let name = format!("{package}.{}", text(&service, NAME)?);
                    for method in service.iter().filter(|method| method.tag == METHOD) {
                        let method = entries(method.payload)?;
                        let selected = text(&method, NAME)?;
                        if METHODS.contains(&(name.as_str(), selected.as_str())) {
                            roots.push(text(&method, INPUT)?);
                            roots.push(text(&method, OUTPUT)?);
                        }
                    }
                }
                _ => {}
            }
        }
    }
    if roots.len() != METHODS.len() * 2 {
        return Err(error("a relation method is missing from the linked descriptor"));
    }
    let mut kept = BTreeSet::new();
    let mut files_kept = BTreeSet::new();
    while let Some(name) = roots.pop() {
        if let Some(file) = index.enums.get(&name) {
            files_kept.insert(*file);
            continue;
        }
        let (file, top) = index.owners.get(&name).ok_or_else(|| error("unresolved type"))?;
        files_kept.insert(*file);
        if kept.insert(top.clone()) {
            roots.extend(index.references.get(top).into_iter().flatten().cloned());
        }
    }
    let mut out = Vec::new();
    for (position, file) in files.iter().enumerate() {
        let fields = entries(file.payload)?;
        let package = text(&fields, PACKAGE)?;
        let scope = if package.is_empty() { String::new() } else { format!(".{package}") };
        let mut body = Vec::new();
        let mut services = false;
        for entry in &fields {
            match entry.tag {
                MESSAGE => {
                    let name = format!("{scope}.{}", text(&entries(entry.payload)?, NAME)?);
                    if kept.contains(&name) {
                        body.extend_from_slice(entry.raw);
                    }
                }
                SERVICE => {
                    let service = entries(entry.payload)?;
                    let name = format!("{package}.{}", text(&service, NAME)?);
                    let mut retained = Vec::new();
                    for part in &service {
                        if part.tag != METHOD {
                            retained.extend_from_slice(part.raw);
                            continue;
                        }
                        let method = text(&entries(part.payload)?, NAME)?;
                        if METHODS.contains(&(name.as_str(), method.as_str())) {
                            retained.extend_from_slice(part.raw);
                            services = true;
                        }
                    }
                    if METHODS.iter().any(|(service, _)| *service == name) {
                        length_delimited(SERVICE, &retained, &mut body);
                    }
                }
                _ => body.extend_from_slice(entry.raw),
            }
        }
        if files_kept.contains(&position) || services {
            length_delimited(FILE, &body, &mut out);
        }
    }
    Ok(out)
}
