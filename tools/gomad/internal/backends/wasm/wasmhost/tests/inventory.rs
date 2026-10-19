//! Rust reader and completeness checks for the WASM determinism inventory
//! (`../determinism/inventory.json`, schema
//! `gomad.wasm-determinism-inventory/v1`).
//!
//! The import table is derived from the `operations!` invocation in
//! `src/abi.rs`, which is the single source of the helper's imports. The
//! check lives here rather than in `src/` because any byte change under
//! `src/`, including a `cfg(test)` module, changes the release helper and
//! therefore its identity.

use serde::Deserialize;
use std::collections::BTreeSet;

const INVENTORY: &str = include_str!("../../determinism/inventory.json");
const ABI: &str = include_str!("../src/abi.rs");
const SCHEMA: &str = "gomad.wasm-determinism-inventory/v1";

#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
struct Inventory {
    schema: String,
    reviewed: Reviewed,
    entries: Vec<Entry>,
}

#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
struct Reviewed {
    go_toolchain: String,
    wasmtime: String,
    rust: String,
}

#[derive(Clone, Debug, Deserialize)]
#[serde(deny_unknown_fields)]
struct Entry {
    id: String,
    layer: String,
    class: String,
    site: String,
    disposition: String,
    owner: String,
    rule: String,
    evidence: Vec<String>,
    #[serde(default)]
    justification: String,
    #[serde(default)]
    readme_anchor: String,
}

fn parse(text: &str) -> Result<Inventory, String> {
    let inventory: Inventory = serde_json::from_str(text).map_err(|error| error.to_string())?;
    if inventory.schema != SCHEMA {
        return Err(format!("inventory schema {:?}", inventory.schema));
    }
    let reviewed = &inventory.reviewed;
    if reviewed.go_toolchain.is_empty() || reviewed.wasmtime.is_empty() || reviewed.rust.is_empty()
    {
        return Err("inventory reviewed versions incomplete".into());
    }
    let mut ids = BTreeSet::new();
    let mut sites = BTreeSet::new();
    for entry in &inventory.entries {
        if !ids.insert(entry.id.as_str()) {
            return Err(format!("inventory id {:?} reused", entry.id));
        }
        if !sites.insert(entry.site.as_str()) {
            return Err(format!("inventory site {:?} has two rows", entry.site));
        }
        let required = [
            &entry.layer,
            &entry.class,
            &entry.disposition,
            &entry.owner,
            &entry.rule,
        ];
        if required.iter().any(|field| field.is_empty()) || entry.evidence.is_empty() {
            return Err(format!("inventory row {:?} incomplete", entry.id));
        }
        if (entry.disposition == "escape")
            != (!entry.justification.is_empty() && !entry.readme_anchor.is_empty())
        {
            return Err(format!("inventory row {:?} escape justification", entry.id));
        }
    }
    Ok(inventory)
}

/// One row of the `operations!` table: variant, module and import name.
#[derive(Debug, PartialEq, Eq)]
struct Operation {
    variant: String,
    module: String,
    name: String,
}

/// Parses the `operations! { ... }` invocation. Every non-blank line inside
/// the block must be a table row; anything else fails, so a format change
/// cannot silently drop operations.
fn operations(source: &str) -> Result<Vec<Operation>, String> {
    let start = source
        .find("\noperations! {\n")
        .ok_or("abi.rs has no operations! invocation")?;
    let body = &source[start + "\noperations! {\n".len()..];
    let end = body
        .find("\n}\n")
        .ok_or("operations! invocation not closed")?;
    let mut table = Vec::new();
    for line in body[..end]
        .lines()
        .map(str::trim)
        .filter(|line| !line.is_empty())
    {
        let malformed = || format!("operations! row not understood: {line:?}");
        let (variant, rest) = line.split_once(": ").ok_or_else(malformed)?;
        let mut quoted = rest.split('"');
        let (Some(""), Some(module), Some(", "), Some(name), Some(", "), Some(_callback)) = (
            quoted.next(),
            quoted.next(),
            quoted.next(),
            quoted.next(),
            quoted.next(),
            quoted.next(),
        ) else {
            return Err(malformed());
        };
        let signature = quoted.next().ok_or_else(malformed)?;
        if quoted.next().is_some()
            || !signature.starts_with(", (")
            || !signature.ends_with(';')
            || !variant.chars().all(char::is_alphanumeric)
        {
            return Err(malformed());
        }
        table.push(Operation {
            variant: variant.into(),
            module: module.into(),
            name: name.into(),
        });
    }
    if table.is_empty() {
        return Err("operations! table is empty".into());
    }
    Ok(table)
}

/// Reports every operation without a row and every import row that names no
/// operation.
fn import_gaps(entries: &[Entry], table: &[Operation]) -> Vec<String> {
    let mut gaps = Vec::new();
    let mut covered = BTreeSet::new();
    for entry in entries {
        let Some(site) = entry.site.strip_prefix("import:") else {
            continue;
        };
        if site == "*" {
            continue;
        }
        match table
            .iter()
            .find(|op| format!("{}::{}", op.module, op.name) == site)
        {
            Some(op) => {
                covered.insert(op.variant.as_str());
            }
            None => gaps.push(format!(
                "inventory row {:?} names import {site:?} that the operation table no longer has",
                entry.id
            )),
        }
    }
    for op in table {
        if !covered.contains(op.variant.as_str()) {
            gaps.push(format!("operation {} has no inventory row", op.variant));
        }
    }
    gaps
}

#[test]
fn inventory_covers_every_operation() {
    let inventory = parse(INVENTORY).unwrap();
    let table = operations(ABI).unwrap();
    assert_eq!(
        import_gaps(&inventory.entries, &table),
        Vec::<String>::new()
    );
    assert!(
        inventory
            .entries
            .iter()
            .any(|entry| entry.site == "import:*" && entry.disposition == "rejected"),
        "unknown imports need a rejected row"
    );
    // Removing a row fails by the operation's name.
    let without: Vec<Entry> = inventory
        .entries
        .iter()
        .filter(|entry| entry.site != "import:wasi_snapshot_preview1::clock_time_get")
        .cloned()
        .collect();
    assert_eq!(
        import_gaps(&without, &table),
        vec!["operation ClockTimeGet has no inventory row".to_string()]
    );
    // An operation added to the table without a row fails by its name.
    let added = ABI.replacen(
        "\noperations! {\n",
        "\noperations! {\n    SockRecv: \"wasi_snapshot_preview1\", \"sock_recv\", \"sock_recv\", (p0: i32) -> i32;\n",
        1,
    );
    assert_eq!(
        import_gaps(&inventory.entries, &operations(&added).unwrap()),
        vec!["operation SockRecv has no inventory row".to_string()]
    );
    // A row whose import the table no longer has fails by its id.
    let mut stale = inventory.entries;
    stale[0].site = "import:wasi_snapshot_preview1::sock_recv".into();
    assert!(
        import_gaps(&stale, &table)
            .iter()
            .any(|gap| gap.contains("no longer has"))
    );
}

#[test]
fn operation_table_parser_fails_closed() {
    let changed = ABI.replacen(
        "\noperations! {\n",
        "\noperations! {\n    // a comment the parser does not know\n",
        1,
    );
    assert!(operations(&changed).unwrap_err().contains("not understood"));
    assert!(operations("fn main() {}\n").is_err());
}

#[test]
fn inventory_loader_rejects_unknown_fields_and_reused_ids() {
    let base: serde_json::Value = serde_json::from_str(INVENTORY).unwrap();
    let mut document = base.clone();
    document["entries"][0]["note"] = "x".into();
    assert!(
        parse(&document.to_string())
            .unwrap_err()
            .contains("unknown field")
    );
    let mut document = base.clone();
    document["extra"] = true.into();
    assert!(
        parse(&document.to_string())
            .unwrap_err()
            .contains("unknown field")
    );
    let mut document = base.clone();
    document["entries"][1]["id"] = document["entries"][0]["id"].clone();
    assert!(parse(&document.to_string()).unwrap_err().contains("reused"));
    let mut document = base;
    document["entries"][0]["disposition"] = "escape".into();
    assert!(parse(&document.to_string()).unwrap_err().contains("escape"));
}
