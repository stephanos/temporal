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
use sha2::{Digest, Sha256};
use std::collections::{BTreeMap, BTreeSet};

const INVENTORY: &str = include_str!("../../../internal/backends/wasm/determinism/inventory.json");
const ABI: &str = include_str!("../src/abi.rs");
const SCHEMA: &str = "gomad.wasm-determinism-inventory/v1";
const IDENTITY_HISTORY: &str =
    include_str!("../../../internal/backends/wasm/determinism/identity-history.json");
// Candidate 276-record trust prefix; independent fn232-determinism-reach review required.
const ACCEPTED_IDENTITY_COUNT: usize = 276;
const ACCEPTED_IDENTITY_SHA256: &str =
    "6ae785f83f0b8325df082e577699059728954b9b9efcf0c2637fca72aaf92f55";

#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
struct IdentityRecord {
    id: String,
    site: String,
}

fn check_identity_history(text: &str, entries: &[Entry]) -> Result<(), String> {
    let records: Vec<IdentityRecord> =
        serde_json::from_str(text).map_err(|error| error.to_string())?;
    if records.len() < ACCEPTED_IDENTITY_COUNT {
        return Err("historical identity ledger removed accepted records".into());
    }
    let mut hash = Sha256::new();
    for record in &records[..ACCEPTED_IDENTITY_COUNT] {
        hash.update(record.id.as_bytes());
        hash.update(b"\t");
        hash.update(record.site.as_bytes());
        hash.update(b"\n");
    }
    if format!("{:x}", hash.finalize()) != ACCEPTED_IDENTITY_SHA256 {
        return Err(
            "historical identity ledger changed accepted records; append new identities".into(),
        );
    }
    let mut by_id = BTreeMap::new();
    let mut sites = BTreeSet::new();
    for record in &records {
        if record.id.is_empty() || record.site.is_empty() {
            return Err("historical identity ledger has an invalid identity".into());
        }
        if by_id
            .insert(record.id.as_str(), record.site.as_str())
            .is_some()
            || !sites.insert(record.site.as_str())
        {
            return Err("historical identity ledger reuses id or site".into());
        }
    }
    for entry in entries {
        if by_id.remove(entry.id.as_str()) != Some(entry.site.as_str()) {
            return Err(format!(
                "historical identity {:?} must retain its site; append new identities to the ledger",
                entry.id
            ));
        }
    }
    if let Some((id, _)) = by_id.first_key_value() {
        return Err(format!(
            "historical identity {id:?} removed; retain its site with disposition rejected when retired"
        ));
    }
    Ok(())
}

fn retired_identity(entry: &Entry, history: &str) -> bool {
    entry.disposition == "rejected"
        && !entry.evidence.is_empty()
        && !entry.rule.is_empty()
        && serde_json::from_str::<Vec<IdentityRecord>>(history).is_ok_and(|records| {
            records
                .get(..ACCEPTED_IDENTITY_COUNT)
                .is_some_and(|accepted| {
                    accepted
                        .iter()
                        .any(|record| record.id == entry.id && record.site == entry.site)
                })
        })
}

#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
struct Inventory {
    schema: String,
    reviewed: Reviewed,
    entries: Vec<Entry>,
}

#[derive(Clone, Debug, Deserialize, PartialEq, Eq)]
#[serde(deny_unknown_fields)]
struct Reviewed {
    date: String,
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

fn calendar_date(text: &str) -> bool {
    let bytes = text.as_bytes();
    if bytes.len() != 10
        || bytes[4] != b'-'
        || bytes[7] != b'-'
        || bytes
            .iter()
            .enumerate()
            .any(|(index, byte)| index != 4 && index != 7 && !byte.is_ascii_digit())
    {
        return false;
    }
    let year: u32 = text[..4].parse().unwrap();
    let month: usize = text[5..7].parse().unwrap();
    let day: u32 = text[8..].parse().unwrap();
    let leap = year.is_multiple_of(4) && (!year.is_multiple_of(100) || year.is_multiple_of(400));
    let days = [
        31,
        if leap { 29 } else { 28 },
        31,
        30,
        31,
        30,
        31,
        31,
        30,
        31,
        30,
        31,
    ];
    (1..=12).contains(&month) && day > 0 && day <= days[month - 1]
}

fn reference_shape(reference: &str) -> bool {
    let Some((kind, tail)) = reference.split_once(':') else {
        return false;
    };
    if kind == "migration" {
        return tail.starts_with("go:internal/backends/wasm/wasi:") && reference_shape(tail);
    }
    let path = match kind {
        "go" | "rust" => {
            let Some((path, name)) = tail.split_once(':') else {
                return false;
            };
            if name.is_empty()
                || !name
                    .chars()
                    .all(|value| value.is_alphanumeric() || value == '_')
            {
                return false;
            }
            path
        }
        "corpus" => tail,
        _ => return false,
    };
    !path.is_empty() && !std::path::Path::new(path).is_absolute() && !path.contains("..")
}

fn parse(text: &str) -> Result<Inventory, String> {
    let inventory: Inventory = serde_json::from_str(text).map_err(|error| error.to_string())?;
    if inventory.schema != SCHEMA {
        return Err(format!("inventory schema {:?}", inventory.schema));
    }
    let reviewed = &inventory.reviewed;
    if !calendar_date(&reviewed.date) {
        return Err("inventory reviewed date must be a valid YYYY-MM-DD calendar day".into());
    }
    if reviewed.go_toolchain.is_empty() || reviewed.wasmtime.is_empty() || reviewed.rust.is_empty()
    {
        return Err("inventory reviewed versions incomplete".into());
    }
    if inventory.entries.is_empty() {
        return Err("inventory has no entries".into());
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
        validate_entry(entry)?;
    }
    check_identity_history(IDENTITY_HISTORY, &inventory.entries)?;
    Ok(inventory)
}

fn validate_entry(entry: &Entry) -> Result<(), String> {
    let required = [
        &entry.id,
        &entry.site,
        &entry.layer,
        &entry.class,
        &entry.disposition,
        &entry.owner,
        &entry.rule,
    ];
    if required.iter().any(|field| field.is_empty()) || entry.evidence.is_empty() {
        return Err(format!("inventory row {:?} incomplete", entry.id));
    }
    let mut parts = entry.id.split(['.', '-']);
    if !parts.next().is_some_and(|part| {
        !part.is_empty()
            && part
                .bytes()
                .all(|byte| byte.is_ascii_lowercase() || byte.is_ascii_digit())
    }) || parts.any(|part| {
        part.is_empty()
            || !part
                .bytes()
                .all(|byte| byte.is_ascii_lowercase() || byte.is_ascii_digit() || byte == b'_')
    }) {
        return Err(format!(
            "inventory row {:?} id is not a stable lower-case identifier",
            entry.id
        ));
    }
    let known = [
        (
            &entry.layer,
            &[
                "guest-runtime",
                "guest-stdlib",
                "helper",
                "host-go",
                "engine-config",
                "host-process",
            ][..],
        ),
        (
            &entry.class,
            &[
                "clock",
                "entropy",
                "scheduling",
                "gc-finalizer",
                "address",
                "fs-io",
                "network-io",
                "thread-order",
                "settings",
            ][..],
        ),
        (
            &entry.disposition,
            &["controlled", "recorded", "rejected", "escape"][..],
        ),
    ];
    if known
        .iter()
        .any(|(value, allowed)| !allowed.contains(&value.as_str()))
    {
        return Err(format!(
            "inventory row {:?} has an unknown enum value",
            entry.id
        ));
    }
    if entry
        .evidence
        .iter()
        .any(|reference| !reference_shape(reference))
    {
        return Err(format!("inventory row {:?} evidence kind", entry.id));
    }
    if if entry.disposition == "escape" {
        entry.justification.trim().is_empty() || entry.readme_anchor.trim().is_empty()
    } else {
        !entry.justification.is_empty() || !entry.readme_anchor.is_empty()
    } {
        return Err(format!("inventory row {:?} escape justification", entry.id));
    }
    Ok(())
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
fn import_gaps(entries: &[Entry], table: &[Operation], history: &str) -> Vec<String> {
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
            None if retired_identity(entry, history) => {}
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
        import_gaps(&inventory.entries, &table, IDENTITY_HISTORY),
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
        import_gaps(&without, &table, IDENTITY_HISTORY),
        vec!["operation ClockTimeGet has no inventory row".to_string()]
    );
    // An operation added to the table without a row fails by its name.
    let added = ABI.replacen(
        "\noperations! {\n",
        "\noperations! {\n    SockRecv: \"wasi_snapshot_preview1\", \"sock_recv\", \"sock_recv\", (p0: i32) -> i32;\n",
        1,
    );
    assert_eq!(
        import_gaps(
            &inventory.entries,
            &operations(&added).unwrap(),
            IDENTITY_HISTORY
        ),
        vec!["operation SockRecv has no inventory row".to_string()]
    );
    // A row whose import the table no longer has fails by its id.
    let mut stale = inventory.entries;
    stale[0].site = "import:wasi_snapshot_preview1::sock_recv".into();
    assert!(
        import_gaps(&stale, &table, IDENTITY_HISTORY)
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
    let mut document = base.clone();
    document["entries"][0]["class"] = "luck".into();
    assert!(parse(&document.to_string()).unwrap_err().contains("enum"));
    let mut document = base;
    document["entries"][0]["disposition"] = "escape".into();
    assert!(parse(&document.to_string()).unwrap_err().contains("escape"));
}

#[test]
fn inventory_review_date_requires_a_calendar_day() {
    let base: serde_json::Value = serde_json::from_str(INVENTORY).unwrap();
    for date in [
        "",
        "2026-02-30",
        "20 October 2026",
        "2026-02-29",
        "1900-02-29",
        "2026-00-01",
        "2026-13-01",
        "2026-01-00",
        "2026-04-31",
    ] {
        let mut document = base.clone();
        document["reviewed"]["date"] = date.into();
        assert!(
            parse(&document.to_string())
                .unwrap_err()
                .contains("reviewed date"),
            "{date}"
        );
    }
    for date in ["2024-02-29", "2000-02-29", "2026-02-28"] {
        let mut document = base.clone();
        document["reviewed"]["date"] = date.into();
        assert!(parse(&document.to_string()).is_ok(), "{date}");
    }
}

#[test]
fn inventory_loader_rejects_every_malformed_row_variant() {
    let base: serde_json::Value = serde_json::from_str(INVENTORY).unwrap();
    for (name, field, value, expected) in [
        ("empty id", "id", serde_json::json!(""), "incomplete"),
        ("empty site", "site", serde_json::json!(""), "incomplete"),
        (
            "no evidence",
            "evidence",
            serde_json::json!([]),
            "incomplete",
        ),
        (
            "bad evidence",
            "evidence",
            serde_json::json!(["not-a-reference"]),
            "evidence kind",
        ),
        (
            "empty reference",
            "evidence",
            serde_json::json!(["rust:"]),
            "evidence kind",
        ),
        (
            "justification outside escape",
            "justification",
            serde_json::json!("unreviewed exception"),
            "escape justification",
        ),
        (
            "anchor outside escape",
            "readme_anchor",
            serde_json::json!("#escape"),
            "escape justification",
        ),
        (
            "unknown layer",
            "layer",
            serde_json::json!("unknown"),
            "enum",
        ),
        (
            "unknown disposition",
            "disposition",
            serde_json::json!("unknown"),
            "enum",
        ),
    ] {
        let mut document = base.clone();
        document["entries"][0][field] = value;
        assert!(
            parse(&document.to_string()).unwrap_err().contains(expected),
            "{name}"
        );
    }
    let mut document = base.clone();
    document["entries"][1]["site"] = document["entries"][0]["site"].clone();
    assert!(
        parse(&document.to_string())
            .unwrap_err()
            .contains("two rows")
    );
    let mut document = base.clone();
    document["reviewed"]["unknown"] = true.into();
    assert!(
        parse(&document.to_string())
            .unwrap_err()
            .contains("unknown field")
    );
    let mut document = base;
    document["schema"] = "unknown".into();
    assert!(parse(&document.to_string()).unwrap_err().contains("schema"));
    assert!(parse(&format!("{INVENTORY} {{}}")).is_err());
}

#[test]
fn inventory_preserves_historical_identities() {
    let base: serde_json::Value = serde_json::from_str(INVENTORY).unwrap();
    for mutation in ["remove", "rename", "reuse", "swap"] {
        let mut document = base.clone();
        match mutation {
            "remove" => {
                document["entries"].as_array_mut().unwrap().remove(0);
            }
            "rename" => document["entries"][0]["id"] = "replacement.id".into(),
            "reuse" => document["entries"][0]["site"] = "import:replacement::site".into(),
            "swap" => {
                let previous = document["entries"][0]["id"].clone();
                document["entries"][0]["id"] = document["entries"][1]["id"].clone();
                document["entries"][1]["id"] = previous;
            }
            _ => unreachable!(),
        }
        assert!(
            parse(&document.to_string())
                .unwrap_err()
                .contains("historical identity"),
            "{mutation}"
        );
    }
}

#[test]
fn identity_history_preserves_accepted_prefix_and_allows_appends() {
    let inventory = parse(INVENTORY).unwrap();
    let base: serde_json::Value = serde_json::from_str(IDENTITY_HISTORY).unwrap();
    for mutation in ["remove", "reuse", "swap", "remove-and-append"] {
        let mut history = base.clone();
        match mutation {
            "remove" => {
                history.as_array_mut().unwrap().remove(0);
            }
            "reuse" => history[0]["site"] = "import:replacement::site".into(),
            "swap" => history.as_array_mut().unwrap().swap(0, 1),
            "remove-and-append" => {
                history.as_array_mut().unwrap().remove(0);
                history
                    .as_array_mut()
                    .unwrap()
                    .push(serde_json::json!({"id": "new.id", "site": "import:new::site"}));
            }
            _ => unreachable!(),
        }
        assert!(
            check_identity_history(&history.to_string(), &inventory.entries)
                .unwrap_err()
                .contains("accepted records"),
            "{mutation}"
        );
    }
    let mut history = base;
    let mut entries = inventory.entries;
    let mut appended = entries[0].clone();
    appended.id = "appended.id".into();
    appended.site = "retired:appended-site".into();
    history
        .as_array_mut()
        .unwrap()
        .push(serde_json::json!({"id": appended.id, "site": appended.site}));
    entries.push(appended);
    assert!(check_identity_history(&history.to_string(), &entries).is_ok());
    entries.last_mut().unwrap().disposition = "rejected".into();
    assert!(check_identity_history(&history.to_string(), &entries).is_ok());
    entries.pop();
    assert!(
        check_identity_history(&history.to_string(), &entries)
            .unwrap_err()
            .contains("removed")
    );
}

#[test]
fn retired_import_requires_historical_identity_and_refusal_evidence() {
    let inventory = parse(INVENTORY).unwrap();
    let row = inventory
        .entries
        .iter()
        .find(|entry| entry.site == "import:wasi_snapshot_preview1::path_symlink")
        .unwrap();
    let retired = row.clone();
    assert_eq!(retired.disposition, "rejected");
    assert!(retired.evidence.iter().any(|reference| reference
        == "rust:tests/core_runtime_preservation.rs:rejected_stock_import_inventory_has_an_actual_ordinary_refusal_for_every_site"));
    assert_eq!(
        import_gaps(std::slice::from_ref(&retired), &[], IDENTITY_HISTORY),
        Vec::<String>::new()
    );
    assert!(
        import_gaps(
            std::slice::from_ref(&retired),
            &[Operation {
                variant: "NewUnlisted".into(),
                module: "new".into(),
                name: "unlisted".into()
            }],
            IDENTITY_HISTORY
        )[0]
        .contains("has no inventory row")
    );
    for mutation in ["controlled", "id", "site", "evidence"] {
        let mut changed = retired.clone();
        match mutation {
            "controlled" => changed.disposition = "controlled".into(),
            "id" => changed.id = "invented.id".into(),
            "site" => changed.site = "import:invented::site".into(),
            "evidence" => changed.evidence.clear(),
            _ => unreachable!(),
        }
        assert!(
            import_gaps(&[changed], &[], IDENTITY_HISTORY)[0].contains("no longer has"),
            "{mutation}"
        );
    }
}

#[test]
fn completeness_rejects_unaccepted_retirement() {
    let mut inventory = parse(INVENTORY).unwrap();
    let mut history: serde_json::Value = serde_json::from_str(IDENTITY_HISTORY).unwrap();
    let mut appended = inventory
        .entries
        .iter()
        .find(|entry| entry.site == "import:wasi_snapshot_preview1::path_symlink")
        .unwrap()
        .clone();
    appended.id = "appended.rejected.id".into();
    appended.site = "import:invented::stale".into();
    history
        .as_array_mut()
        .unwrap()
        .push(serde_json::json!({"id": appended.id, "site": appended.site}));
    inventory.entries.push(appended);
    let history = history.to_string();
    assert!(check_identity_history(&history, &inventory.entries).is_ok());
    let mut table = operations(ABI).unwrap();
    assert!(
        import_gaps(&inventory.entries, &table, &history)
            .iter()
            .any(|gap| gap.contains("invented::stale") && gap.contains("no longer has"))
    );
    inventory.entries.last_mut().unwrap().disposition = "controlled".into();
    table.push(Operation {
        variant: "Appended".into(),
        module: "invented".into(),
        name: "stale".into(),
    });
    assert_eq!(
        import_gaps(&inventory.entries, &table, &history),
        Vec::<String>::new()
    );
}

/// Patterns the helper source scan treats as time, entropy, thread or
/// hash-iteration sites.
const SOURCE_PATTERNS: &[&str] = &[
    "Instant::now",
    "clock_gettime(",
    "SystemTime",
    "/dev/urandom",
    "thread::spawn",
    "thread::sleep",
    "thread::Builder",
    "thread::scope",
    "recv_timeout",
    "std::process::Command",
    "HashMap",
    "HashSet",
];

/// Drops a line comment. With `blank`, string and char literal contents are
/// blanked too, so braces inside them do not count.
fn strip_literals_mode(line: &str, blank: bool) -> String {
    let mut out = String::new();
    let mut chars = line.chars().peekable();
    while let Some(c) = chars.next() {
        match c {
            '/' if chars.peek() == Some(&'/') => break,
            '"' => {
                let mut literal = String::from('"');
                while let Some(c) = chars.next() {
                    literal.push(c);
                    if c == '\\' {
                        literal.extend(chars.next());
                    } else if c == '"' {
                        break;
                    }
                }
                out.push_str(if blank { "\"\"" } else { &literal });
            }
            '\'' => {
                // A char literal ('{', '\n'); a lifetime has no closing quote nearby.
                let rest: String = chars.clone().take(3).collect();
                if let Some(end) = rest.find('\'').filter(|&end| end > 0) {
                    let literal: String = std::iter::once(c)
                        .chain((0..=end).filter_map(|_| chars.next()))
                        .collect();
                    out.push_str(if blank { "' '" } else { &literal });
                } else {
                    out.push(c);
                }
            }
            _ => out.push(c),
        }
    }
    out
}

fn strip_literals(line: &str) -> String {
    strip_literals_mode(line, true)
}

/// Parses `[pub[(..)]] mod name;`. A line that declares a module in any other
/// form fails, so the reachability walk cannot silently miss a file.
fn module_declaration(trimmed: &str) -> Result<Option<String>, String> {
    let rest = if let Some(rest) = trimmed.strip_prefix("pub") {
        let rest = if rest.starts_with('(') {
            &rest[rest
                .find(')')
                .ok_or_else(|| format!("visibility not closed: {trimmed:?}"))?
                + 1..]
        } else {
            rest
        };
        rest.strip_prefix(' ').unwrap_or(rest)
    } else {
        trimmed
    };
    let Some(declaration) = rest.strip_prefix("mod ") else {
        if trimmed
            .split(|c: char| !c.is_alphanumeric() && c != '_')
            .any(|token| token == "mod")
        {
            return Err(format!("module declaration not understood: {trimmed:?}"));
        }
        return Ok(None);
    };
    let Some(name) = declaration.strip_suffix(';') else {
        let Some(name) = declaration.strip_suffix('{').map(str::trim) else {
            return Err(format!("module declaration not understood: {trimmed:?}"));
        };
        if name.is_empty() || !name.chars().all(|c| c.is_alphanumeric() || c == '_') {
            return Err(format!("module declaration not understood: {trimmed:?}"));
        }
        return Ok(None);
    };
    if name.is_empty() || !name.chars().all(|c| c.is_alphanumeric() || c == '_') {
        return Err(format!("module declaration not understood: {trimmed:?}"));
    }
    Ok(Some(name.to_string()))
}

/// Returns the production lines of one source file and the child modules it
/// declares as test-only. A `#[cfg(test)]` item is skipped: a one-line item,
/// or a braced block that must close on a `}` line at the item's indentation.
/// Numbered production lines and the test-only child modules of a file.
type ProductionLines = (Vec<(usize, String)>, Vec<String>);

fn mask_multiline_literals(text: &str) -> String {
    let bytes = text.as_bytes();
    let mut masked = bytes.to_vec();
    let mut index = 0;
    while index < bytes.len() {
        if bytes[index..].starts_with(b"//") {
            while index < bytes.len() && bytes[index] != b'\n' {
                masked[index] = b' ';
                index += 1;
            }
        } else if bytes[index..].starts_with(b"/*") {
            let mut depth = 1;
            masked[index..index + 2].fill(b' ');
            index += 2;
            while index < bytes.len() && depth > 0 {
                if bytes[index..].starts_with(b"/*") {
                    depth += 1;
                    masked[index..index + 2].fill(b' ');
                    index += 2;
                } else if bytes[index..].starts_with(b"*/") {
                    depth -= 1;
                    masked[index..index + 2].fill(b' ');
                    index += 2;
                } else {
                    if bytes[index] != b'\n' {
                        masked[index] = b' ';
                    }
                    index += 1;
                }
            }
        } else if bytes[index] == b'\'' && {
            let end = if bytes.get(index + 1) == Some(&b'\\') {
                index + 3
            } else {
                index + 1 + text[index + 1..].chars().next().map_or(0, char::len_utf8)
            };
            bytes.get(end) == Some(&b'\'')
        } {
            let end = if bytes.get(index + 1) == Some(&b'\\') {
                index + 4
            } else {
                index + 2 + text[index + 1..].chars().next().map_or(0, char::len_utf8)
            };
            masked[index..end].fill(b' ');
            index = end;
        } else {
            let mut quote = index;
            if bytes[index] == b'r' {
                quote += 1;
                while bytes.get(quote) == Some(&b'#') {
                    quote += 1;
                }
            }
            if bytes.get(quote) == Some(&b'"') {
                let raw = quote != index;
                let hashes = quote.saturating_sub(index + 1);
                masked[index..=quote].fill(b' ');
                index = quote + 1;
                while index < bytes.len() {
                    let closes = bytes[index] == b'"'
                        && (!raw
                            || bytes
                                .get(index + 1..index + 1 + hashes)
                                .is_some_and(|tail| tail.iter().all(|byte| *byte == b'#')));
                    if closes {
                        let end = index + 1 + if raw { hashes } else { 0 };
                        masked[index..end].fill(b' ');
                        index = end;
                        break;
                    }
                    if !raw && bytes[index] == b'\\' {
                        masked[index] = b' ';
                        index += 1;
                    }
                    if index < bytes.len() {
                        if bytes[index] != b'\n' {
                            masked[index] = b' ';
                        }
                        index += 1;
                    }
                }
            } else {
                index += 1;
            }
        }
    }
    String::from_utf8(masked).expect("masking preserves UTF-8 outside literals")
}

fn production_lines(text: &str) -> Result<ProductionLines, String> {
    let lines: Vec<&str> = text.lines().collect();
    let masked = mask_multiline_literals(text);
    let masked_lines: Vec<&str> = masked.lines().collect();
    let (mut kept, mut test_modules) = (Vec::new(), Vec::new());
    let mut index = 0;
    while index < lines.len() {
        if masked_lines[index].trim() != "#[cfg(test)]"
            && !masked_lines[index]
                .split_whitespace()
                .collect::<String>()
                .starts_with("#[cfg(all(test,")
        {
            kept.push((index + 1, strip_literals_mode(lines[index], false)));
            index += 1;
            continue;
        }
        index += 1;
        while index < lines.len() {
            let trimmed = lines[index].trim();
            if trimmed.starts_with("#[") || trimmed.starts_with("///") {
                index += 1;
            } else {
                break;
            }
        }
        let start = index;
        let item = lines.get(start).ok_or("cfg(test) at end of file")?;
        let indent = item.len() - item.trim_start().len();
        let first = strip_literals(item);
        if first.trim().ends_with(';')
            && let Some(name) = module_declaration(first.trim())?
        {
            test_modules.push(name);
            index += 1;
            continue;
        }
        // The item ends on the first line where every bracket is closed and
        // the line ends the item: `;` or `,` for a statement, field or
        // expression, `}` for a block.
        let mut depth = 0i64;
        loop {
            let line = strip_literals(
                masked_lines
                    .get(index)
                    .ok_or_else(|| format!("cfg(test) item at line {} never closes", start + 1))?,
            );
            for c in line.chars() {
                match c {
                    '{' | '(' | '[' => depth += 1,
                    '}' | ')' | ']' => depth -= 1,
                    _ => {}
                }
            }
            index += 1;
            let end = line.trim_end();
            if depth < 0 {
                return Err(format!("cfg(test) item at line {} unbalanced", start + 1));
            }
            if depth == 0 && (end.ends_with(';') || end.ends_with(',') || end.ends_with('}')) {
                let last = lines[index - 1];
                let closing = last.len() - last.trim_start().len();
                // A block closes at the item's indentation; a statement may end
                // on a deeper continuation line, never a shallower one.
                let aligned = if end.ends_with('}') {
                    closing == indent
                } else {
                    closing >= indent
                };
                if index - 1 != start && !aligned {
                    return Err(format!(
                        "cfg(test) item at line {} closes at line {index} out of indentation",
                        start + 1
                    ));
                }
                break;
            }
        }
    }
    Ok((kept, test_modules))
}

/// Counts pattern hits in the helper's production sources, keyed
/// `file|pattern|count`. Production files are the build script and those reachable from `main.rs`/`lib.rs`
/// through `mod` declarations outside `#[cfg(test)]` items; test-only modules
/// are never reached.
fn source_sites(src: &std::path::Path) -> Result<BTreeSet<String>, String> {
    source_scan(src, &["main.rs", "lib.rs", "../build.rs"]).map(|(sites, _)| sites)
}

fn source_scan(
    src: &std::path::Path,
    roots: &[&str],
) -> Result<(BTreeSet<String>, BTreeSet<String>), String> {
    let mut sites = BTreeSet::new();
    let mut pending: Vec<String> = roots.iter().map(|root| (*root).to_string()).collect();
    let mut seen = BTreeSet::new();
    while let Some(relative) = pending.pop() {
        if !seen.insert(relative.clone()) {
            continue;
        }
        let text = std::fs::read_to_string(src.join(&relative))
            .map_err(|error| format!("{relative}: {error}"))?;
        let (lines, _) = production_lines(&text).map_err(|error| format!("{relative}: {error}"))?;
        let masked = mask_multiline_literals(&text);
        let masked_lines: Vec<&str> = masked.lines().collect();
        let directory = relative
            .rsplit_once('/')
            .map_or("", |(directory, _)| directory);
        let stem = relative.strip_suffix(".rs").unwrap();
        let owns_directory =
            roots.contains(&relative.as_str()) || stem == "mod" || stem.ends_with("/mod");
        for (index, (number, line)) in lines.iter().enumerate() {
            let trimmed = masked_lines[number - 1].trim();
            let Some(name) =
                module_declaration(trimmed).map_err(|error| format!("{relative}: {error}"))?
            else {
                continue;
            };
            let name = name.as_str();
            if line.len() != line.trim_start().len() {
                return Err(format!("{relative}: nested module declaration {trimmed:?}"));
            }
            let join = |base: &str, path: &str| {
                if base.is_empty() {
                    path.to_string()
                } else {
                    format!("{base}/{path}")
                }
            };
            // The scanned text has string literals blanked; read the attribute
            // from the original line.
            let attribute = index
                .checked_sub(1)
                .map(|previous| lines[previous].0)
                .and_then(|number| text.lines().nth(number - 1))
                .map(str::trim)
                .and_then(|line| line.strip_prefix("#[path = \""))
                .and_then(|line| line.strip_suffix("\"]"));
            let child = if let Some(path) = attribute {
                join(directory, path)
            } else {
                let base = if owns_directory {
                    directory.to_string()
                } else {
                    stem.to_string()
                };
                let file = join(&base, &format!("{name}.rs"));
                if src.join(&file).is_file() {
                    file
                } else {
                    join(&base, &format!("{name}/mod.rs"))
                }
            };
            if !src.join(&child).is_file() {
                return Err(format!(
                    "{relative}: module {name} resolves to missing {child}"
                ));
            }
            pending.push(child);
        }
        for pattern in SOURCE_PATTERNS {
            let count: usize = lines
                .iter()
                .map(|(_, line)| line.matches(pattern).count())
                .sum();
            if count > 0 {
                sites.insert(format!("{relative}|{pattern}|{count}"));
            }
        }
    }
    Ok((sites, seen))
}

fn source_gaps(entries: &[Entry], sites: &BTreeSet<String>) -> Vec<String> {
    let migrations =
        serde_json::from_str::<Vec<host_inventory::Migration>>(host_inventory::MIGRATIONS).unwrap();
    let (retired, migration_error) = match host_inventory::migrated_identities(
        entries,
        sites,
        IDENTITY_HISTORY,
        &migrations,
        "helper-source:",
    ) {
        Ok(retired) => (retired, None),
        Err(error) => (BTreeSet::new(), Some(error)),
    };
    let rows: BTreeSet<&str> = entries
        .iter()
        .filter_map(|entry| entry.site.strip_prefix("helper-source:"))
        .collect();
    let active_rows: BTreeSet<&str> = entries
        .iter()
        .filter(|entry| entry.disposition != "rejected")
        .filter_map(|entry| entry.site.strip_prefix("helper-source:"))
        .collect();
    let allowed: BTreeSet<&str> = entries
        .iter()
        .filter(|entry| entry.disposition != "rejected")
        .filter_map(|entry| entry.site.strip_prefix("helper-allow:"))
        .collect();
    let all_allowlists: BTreeSet<&str> = entries
        .iter()
        .filter_map(|entry| entry.site.strip_prefix("helper-allow:"))
        .collect();
    let mut gaps: Vec<String> = migration_error.into_iter().collect();
    for site in sites {
        let file = site.split('|').next().unwrap();
        if !allowed.contains(file) && !active_rows.contains(site.as_str()) {
            gaps.push(format!("helper source site {site} has no inventory row"));
        }
    }
    for row in &rows {
        if !sites.contains(*row)
            && !entries.iter().any(|entry| {
                entry.site == format!("helper-source:{row}")
                    && (retired_identity(entry, IDENTITY_HISTORY) || retired.contains(&entry.id))
            })
        {
            gaps.push(format!(
                "inventory row names helper source site {row} that the sources no longer have"
            ));
        }
    }
    for file in &all_allowlists {
        if !sites
            .iter()
            .any(|site| site.starts_with(&format!("{file}|")))
            && !entries.iter().any(|entry| {
                entry.site == format!("helper-allow:{file}")
                    && retired_identity(entry, IDENTITY_HISTORY)
            })
        {
            gaps.push(format!("allowlisted helper file {file} has no site"));
        }
    }
    gaps
}

#[test]
fn helper_source_scan_sites_are_inventoried() {
    let inventory = parse(INVENTORY).unwrap();
    let src = std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("src");
    let sites = source_sites(&src).unwrap();
    assert_eq!(
        source_gaps(&inventory.entries, &sites),
        Vec::<String>::new()
    );
    // A new site in a file without an allowlist row fails by name.
    let mut added = sites;
    added.insert("engine.rs|SystemTime|1".into());
    assert_eq!(
        source_gaps(&inventory.entries, &added),
        vec!["helper source site engine.rs|SystemTime|1 has no inventory row".to_string()]
    );
}

#[test]
fn retired_helper_allowlist_does_not_cover_new_sites() {
    let inventory = parse(INVENTORY).unwrap();
    let mut retired = inventory
        .entries
        .iter()
        .find(|entry| entry.site == "helper-allow:diagnostic.rs")
        .unwrap()
        .clone();
    retired.disposition = "rejected".into();
    assert!(retired_identity(&retired, IDENTITY_HISTORY));
    assert_eq!(
        source_gaps(std::slice::from_ref(&retired), &BTreeSet::new()),
        Vec::<String>::new()
    );
    let sites = BTreeSet::from(["diagnostic.rs|/dev/urandom|1".into()]);
    assert_eq!(
        source_gaps(std::slice::from_ref(&retired), &sites),
        vec!["helper source site diagnostic.rs|/dev/urandom|1 has no inventory row".to_string()]
    );
    retired.id = "invented.allowlist".into();
    assert_eq!(
        source_gaps(&[retired], &BTreeSet::new()),
        vec!["allowlisted helper file diagnostic.rs has no site".to_string()]
    );
}

#[test]
fn helper_source_scan_skips_only_test_items() {
    let text = "use std::time::Instant;\n#[cfg(test)]\nmod tests;\n#[cfg(test)]\nmod inline {\n    fn f() { let _ = \"}\"; Instant::now(); }\n}\nfn g() {\n    #[cfg(test)]\n    if x {\n        Instant::now();\n    }\n    Instant::now();\n}\n";
    let (lines, modules) = production_lines(text).unwrap();
    assert_eq!(modules, vec!["tests".to_string()]);
    let hits: usize = lines
        .iter()
        .map(|(_, line)| line.matches("Instant::now").count())
        .sum();
    assert_eq!(hits, 1);
    assert!(production_lines("#[cfg(test)]\nmod m {\n    fn f() {\n").is_err());
    // A multi-line test-only statement ends at its own `;`, not at a later block.
    let statement = "fn h() {\n    #[cfg(test)]\n    self.counter\n        .fetch_add(1, Ordering::Relaxed);\n    let head = Bounded {\n        at: Instant::now(),\n    };\n}\n";
    let (lines, _) = production_lines(statement).unwrap();
    let hits: usize = lines
        .iter()
        .map(|(_, line)| line.matches("Instant::now").count())
        .sum();
    assert_eq!(hits, 1);
    assert_eq!(
        module_declaration("pub(super) mod inner;").unwrap(),
        Some("inner".to_string())
    );
    assert!(module_declaration("pub mod a::b;").is_err());
}

#[test]
fn source_scan_skips_multiline_raw_and_combined_test_items() {
    let source = r###"#[cfg(all(test, feature = "determinism-fixture"))]
mod tests {
    const SOURCE: &str = r##" unmatched { ( [
        fn fake() { Instant::now(); }
    "##;
    const QUOTE: char = '\'';
    const DOUBLE: char = '"';
    /* outer { /* nested ( */ } */
}
fn production<'a>(value: &'a str) { thread::scope(|_| {}); }
"###;
    let (lines, modules) = production_lines(source).unwrap();
    assert_eq!(modules, Vec::<String>::new());
    assert_eq!(lines.len(), 1);
    assert!(lines[0].1.starts_with("fn production"));
    assert!(production_lines("#[cfg(test)]\nmod tests {\n").is_err());
    let source = "/*\n#[cfg(test)]\n*/\nmod production;\nconst RAW: &str = r#\"\n#[cfg(test)]\n\"#;\nmod receiving;\n";
    let (lines, _) = production_lines(source).unwrap();
    assert!(lines.iter().any(|(_, line)| line == "mod production;"));
    assert!(lines.iter().any(|(_, line)| line == "mod receiving;"));
}

#[test]
fn source_scan_explicit_roots_own_their_sibling_module_directory() {
    let root = std::env::temp_dir().join(format!("gomad-explicit-roots-{}", std::process::id()));
    for directory in ["src/entry", "src/lib", "build_identity"] {
        std::fs::create_dir_all(root.join(directory)).unwrap();
    }
    for (name, text) in [
        ("src/main.rs", "mod entry;\nmod lib;\n"),
        ("src/entry.rs", "mod leaf;\n"),
        ("src/entry/leaf.rs", "fn clock() { Instant::now(); }\n"),
        (
            "src/leaf.rs",
            "fn wrong_directory() { SystemTime::now(); }\n",
        ),
        ("src/lib.rs", "mod child;\n"),
        ("src/lib/child.rs", "fn clock() { Instant::now(); }\n"),
        (
            "src/child.rs",
            "fn wrong_directory() { SystemTime::now(); }\n",
        ),
        ("src/custom_root.rs", "mod neighbor;\n"),
        ("src/neighbor.rs", "fn clock() { Instant::now(); }\n"),
        ("build.rs", "mod build_identity;\n"),
        ("build_identity.rs", "mod nested;\n"),
        (
            "build_identity/nested.rs",
            "fn clock() { Instant::now(); }\n",
        ),
    ] {
        std::fs::write(root.join(name), text).unwrap();
    }
    let roots = ["main.rs", "custom_root.rs", "../build.rs"];
    let actual = source_scan(&root.join("src"), &roots);
    std::fs::remove_file(root.join("src/entry/leaf.rs")).unwrap();
    let missing = source_scan(&root.join("src"), &roots);
    std::fs::remove_dir_all(root).unwrap();
    let (sites, files) = actual.unwrap();
    assert_eq!(
        files,
        BTreeSet::from(
            [
                "main.rs",
                "entry.rs",
                "entry/leaf.rs",
                "lib.rs",
                "lib/child.rs",
                "custom_root.rs",
                "neighbor.rs",
                "../build.rs",
                "../build_identity.rs",
                "../build_identity/nested.rs",
            ]
            .map(str::to_owned)
        )
    );
    assert_eq!(
        sites,
        BTreeSet::from(
            [
                "entry/leaf.rs|Instant::now|1",
                "lib/child.rs|Instant::now|1",
                "neighbor.rs|Instant::now|1",
                "../build_identity/nested.rs|Instant::now|1",
            ]
            .map(str::to_owned)
        )
    );
    assert!(
        missing
            .unwrap_err()
            .contains("entry.rs: module leaf resolves to missing entry/leaf/mod.rs")
    );
}

#[test]
fn source_scan_refuses_inline_modules_that_hide_external_children() {
    let root = std::env::temp_dir().join(format!("gomad-inline-source-{}", std::process::id()));
    std::fs::create_dir(&root).unwrap();
    std::fs::create_dir(root.join("inner")).unwrap();
    std::fs::write(root.join("lib.rs"), "mod inner { pub mod hidden; }\n").unwrap();
    std::fs::write(
        root.join("inner/hidden.rs"),
        "pub fn hidden() { let _ = std::time::Instant::now(); }\n",
    )
    .unwrap();
    let result = source_scan(&root, &["lib.rs"]);
    std::fs::remove_dir_all(root).unwrap();
    assert!(result.unwrap_err().contains("module declaration"));
    assert_eq!(module_declaration("mod platform {").unwrap(), None);
    assert_eq!(module_declaration("pub(crate) mod phases {").unwrap(), None);
    assert!(module_declaration("#[cfg(unix)] mod hidden;").is_err());
    assert!(module_declaration("fn wrapper() { mod hidden; }").is_err());
}

const ENGINE: &str = include_str!("../src/engine.rs");
const MANIFEST: &str = include_str!("../Cargo.toml");

/// Derives every Wasmtime setting the helper engine applies: each builder
/// call on the shared `Config` in `engine::configuration` (with its arguments and any
/// `cfg` attribute) and the wasmtime Cargo feature set.
fn engine_settings(engine: &str, manifest: &str) -> Result<Vec<String>, String> {
    let start = engine
        .find("pub(crate) fn configuration(")
        .ok_or("engine.rs has no configuration")?;
    let body = &engine[start..];
    let end = body
        .find("\n    config\n}")
        .ok_or("configuration returns no Config")?;
    let mut settings = Vec::new();
    let mut attribute = None;
    let mut created = false;
    for line in body[..end].lines().map(str::trim) {
        if line.starts_with("#[cfg(") {
            attribute = Some(line.to_string());
            continue;
        }
        if line == "let mut config = Config::new();" {
            created = true;
            settings.push("Config::new".to_string());
            continue;
        }
        let call = line
            .strip_prefix("config.")
            .or_else(|| line.strip_prefix('.'))
            .map(|call| call.trim_end_matches(';'));
        match (call, attribute.take()) {
            (Some(call), Some(cfg)) => settings.push(format!("{cfg} {call}")),
            (Some(call), None) => settings.push(call.to_string()),
            (None, Some(cfg)) if !line.starts_with("let ") && line != "config" => {
                return Err(format!("unexpected item under {cfg}: {line:?}"));
            }
            _ if created && line != "config" && !line.is_empty() && !line.starts_with("let ") => {
                return Err(format!("prepare line not understood: {line:?}"));
            }
            _ => {}
        }
    }
    if !created {
        return Err("configuration creates no Config".into());
    }
    let features = manifest
        .lines()
        .find(|line| line.starts_with("wasmtime = "))
        .ok_or("Cargo.toml has no wasmtime dependency")?;
    settings.push(format!("cargo {}", features.trim()));
    Ok(settings)
}

#[test]
fn engine_settings_are_inventoried() {
    let inventory = parse(INVENTORY).unwrap();
    let settings = engine_settings(ENGINE, MANIFEST).unwrap();
    let rows: BTreeSet<&str> = inventory
        .entries
        .iter()
        .filter_map(|entry| entry.site.strip_prefix("engine-setting:"))
        .collect();
    let derived: BTreeSet<&str> = settings.iter().map(String::as_str).collect();
    assert_eq!(
        derived.difference(&rows).collect::<Vec<_>>(),
        Vec::<&&str>::new(),
        "engine settings without an inventory row"
    );
    assert_eq!(
        rows.difference(&derived)
            .filter(|site| !inventory
                .entries
                .iter()
                .any(|entry| entry.site == format!("engine-setting:{site}")
                    && retired_identity(entry, IDENTITY_HISTORY)))
            .collect::<Vec<_>>(),
        Vec::<&&str>::new(),
        "inventory rows naming settings the engine no longer applies"
    );
    // A new setting, or a changed value, fails.
    let changed = ENGINE.replacen(".wasm_relaxed_simd(false)", ".wasm_relaxed_simd(true)", 1);
    assert!(
        engine_settings(&changed, MANIFEST)
            .unwrap()
            .iter()
            .any(|setting| !rows.contains(setting.as_str()))
    );
    let added = ENGINE.replacen(
        ".wasm_multi_memory(false)",
        ".wasm_multi_memory(false)\n        .wasm_threads(true)",
        1,
    );
    assert!(
        engine_settings(&added, MANIFEST)
            .unwrap()
            .contains(&"wasm_threads(true)".to_string())
    );
}

#[test]
fn serde_json_objects_serialize_in_sorted_key_order() {
    // Without serde_json's preserve_order feature, objects are BTreeMaps.
    for line in MANIFEST
        .lines()
        .filter(|line| line.starts_with("serde_json"))
    {
        assert!(!line.contains("preserve_order"), "{line}");
    }
    let value = serde_json::json!({"b": 1, "a": {"d": 2, "c": 3}});
    assert_eq!(value.to_string(), r#"{"a":{"c":3,"d":2},"b":1}"#);
}

const REVIEW_HISTORY: &str =
    include_str!("../../../internal/backends/wasm/determinism/review-history.json");

#[derive(Clone, Debug, Deserialize)]
#[serde(deny_unknown_fields)]
struct ReviewRecord {
    timestamp: String,
    reviewed: Reviewed,
}

fn timestamp_key(text: &str) -> Result<[u32; 7], String> {
    let invalid = || "review history requires an explicit UTC timestamp".to_string();
    let Some(body) = text.strip_suffix('Z') else {
        return Err(invalid());
    };
    let bytes = body.as_bytes();
    if bytes.len() < 19
        || !bytes[..19].is_ascii()
        || !calendar_date(&body[..10])
        || bytes[10] != b'T'
        || bytes[13] != b':'
        || bytes[16] != b':'
        || bytes[11..19]
            .iter()
            .enumerate()
            .any(|(i, byte)| i != 2 && i != 5 && !byte.is_ascii_digit())
    {
        return Err(invalid());
    }
    let hour: u32 = body[11..13].parse().map_err(|_| invalid())?;
    let minute: u32 = body[14..16].parse().map_err(|_| invalid())?;
    let second: u32 = body[17..19].parse().map_err(|_| invalid())?;
    if hour > 23 || minute > 59 || second > 59 {
        return Err(invalid());
    }
    let mut nanos = String::new();
    if bytes.len() > 19 {
        let fraction = body[19..].strip_prefix('.').ok_or_else(invalid)?;
        if fraction.is_empty() || !fraction.bytes().all(|byte| byte.is_ascii_digit()) {
            return Err(invalid());
        }
        nanos.extend(fraction.chars().take(9));
    }
    while nanos.len() < 9 {
        nanos.push('0');
    }
    Ok([
        body[..4].parse().unwrap(),
        body[5..7].parse().unwrap(),
        body[8..10].parse().unwrap(),
        hour,
        minute,
        second,
        nanos.parse().unwrap(),
    ])
}

fn check_review_history(text: &str, reviewed: &Reviewed) -> Result<(), String> {
    let records: Vec<ReviewRecord> =
        serde_json::from_str(text).map_err(|error| error.to_string())?;
    let first = records
        .first()
        .ok_or("review history removed an accepted review")?;
    if first.timestamp != "2026-10-20T15:00:00Z"
        || first.reviewed
            != (Reviewed {
                date: "2026-10-20".into(),
                go_toolchain: "go1.27.1".into(),
                wasmtime: "47.0.3".into(),
                rust: "1.99.1".into(),
            })
    {
        return Err("review history changed accepted review; append a new dated record".into());
    }
    let mut previous = [0; 7];
    for record in &records {
        let key = timestamp_key(&record.timestamp)?;
        if key <= previous {
            return Err("review history timestamp must advance".into());
        }
        if record.timestamp.get(..10) != Some(record.reviewed.date.as_str()) {
            return Err("review history date differs from timestamp".into());
        }
        if [
            &record.reviewed.go_toolchain,
            &record.reviewed.wasmtime,
            &record.reviewed.rust,
        ]
        .iter()
        .any(|version| version.trim().is_empty())
        {
            return Err("review history has incomplete versions".into());
        }
        previous = key;
    }
    if &records.last().unwrap().reviewed != reviewed {
        return Err("append a new dated review record for inventory versions or date".into());
    }
    Ok(())
}

#[test]
fn inventory_pins_require_append_only_dated_reviews() {
    let inventory = parse(INVENTORY).unwrap();
    check_review_history(REVIEW_HISTORY, &inventory.reviewed).unwrap();
    let mut changed = inventory.reviewed;
    changed.wasmtime.push_str(".new");
    assert!(
        check_review_history(REVIEW_HISTORY, &changed)
            .unwrap_err()
            .contains("append a new dated review record")
    );
    let mut history: serde_json::Value = serde_json::from_str(REVIEW_HISTORY).unwrap();
    history[0]["reviewed"]["wasmtime"] = changed.wasmtime.clone().into();
    assert!(
        check_review_history(&history.to_string(), &changed)
            .unwrap_err()
            .contains("changed accepted review")
    );
    assert!(check_review_history("[]", &changed).is_err());
    assert!(check_review_history(&format!("{REVIEW_HISTORY} {{}}"), &changed).is_err());
}

#[test]
fn review_history_rejects_stale_dates_and_non_utc_timestamps() {
    let mut base: serde_json::Value = serde_json::from_str(REVIEW_HISTORY).unwrap();
    base.as_array_mut().unwrap().truncate(1);
    for (timestamp, expected) in [
        ("2026-10-20T15:00:00Z", "timestamp must advance"),
        ("2026-10-20T16:00:00Z", ""),
        ("2026-10-21T15:00:00Z", "date differs"),
        ("2026-10-20T16:00:00+00:00", "explicit UTC"),
        ("2026-10-20T16:00:00.1Z", ""),
        ("2026-10-20T25:00:00Z", "explicit UTC"),
        ("2026-02-30T16:00:00Z", "explicit UTC"),
    ] {
        let mut history = base.clone();
        let mut next = history[0].clone();
        next["timestamp"] = timestamp.into();
        next["reviewed"]["wasmtime"] = "47.0.4".into();
        let reviewed: Reviewed = serde_json::from_value(next["reviewed"].clone()).unwrap();
        history.as_array_mut().unwrap().push(next);
        let result = check_review_history(&history.to_string(), &reviewed);
        if expected.is_empty() {
            result.unwrap();
        } else {
            assert!(result.unwrap_err().contains(expected), "{timestamp}");
        }
    }
}

fn one_pin<'a>(source: &'a str, prefix: &str, suffix: &str) -> Result<&'a str, String> {
    let values = source
        .match_indices(prefix)
        .map(|(at, _)| {
            source[at + prefix.len()..]
                .split_once(suffix)
                .map(|(value, _)| value)
        })
        .collect::<Option<Vec<_>>>()
        .ok_or("unclosed version pin")?;
    if values.len() != 1 {
        return Err(format!("expected one version pin, found {}", values.len()));
    }
    Ok(values[0])
}

#[test]
fn inventory_reviewed_versions_match_exact_candidate_pins() {
    let inventory = parse(INVENTORY).unwrap();
    let overlay = include_str!("../../../internal/backends/wasm/toolchain/overlay.go");
    assert_eq!(
        inventory.reviewed.go_toolchain,
        one_pin(overlay, "string(version), \"\\n\", 2)[0] != \"", "\"").unwrap()
    );
    let lock = include_str!("../../Cargo.lock");
    assert_eq!(
        inventory.reviewed.wasmtime,
        one_pin(lock, "name = \"wasmtime\"\nversion = \"", "\"").unwrap()
    );
    let rust = include_str!("../../rust-toolchain.toml");
    assert_eq!(
        inventory.reviewed.rust,
        one_pin(rust, "channel = \"", "\"").unwrap()
    );
    assert!(one_pin("channel = \"1\"\nchannel = \"2\"", "channel = \"", "\"").is_err());
}

#[test]
fn every_escape_resolves_a_readme_heading() {
    let inventory = parse(INVENTORY).unwrap();
    let readme = include_str!("../../../internal/backends/wasm/README.md");
    let headings: BTreeSet<String> = readme
        .lines()
        .filter_map(|line| line.trim().strip_prefix('#'))
        .map(|title| {
            format!(
                "README.md#{}",
                title
                    .trim_start_matches('#')
                    .trim()
                    .to_lowercase()
                    .replace(' ', "-")
            )
        })
        .collect();
    for row in inventory
        .entries
        .iter()
        .filter(|row| row.disposition == "escape")
    {
        assert!(
            headings.contains(&row.readme_anchor),
            "{}: {}",
            row.id,
            row.readme_anchor
        );
    }
}

#[path = "inventory/runtime.rs"]
mod runtime_inventory;

#[path = "inventory/host.rs"]
mod host_inventory;

fn diagnostic_streams() -> BTreeSet<String> {
    let wire = include_str!("../../../internal/model/choice/internal/wire/wire_generated.go");
    let model = include_str!("../src/model.rs");
    let body = wire
        .split_once("type DiagnosticRecord struct {")
        .unwrap()
        .1
        .split_once("\n}")
        .unwrap()
        .0;
    let state = [
        "Ordinal",
        "VirtualTime",
        "Allocations",
        "GCCycle",
        "GCPhase",
        "RunQueueLength",
    ];
    let mut streams = BTreeSet::new();
    for line in body.lines().filter(|line| !line.trim().is_empty()) {
        let field = line.split_whitespace().next().unwrap();
        if state.contains(&field) {
            continue;
        }
        assert!(
            field.ends_with("Draws"),
            "unclassified diagnostic record field {field}"
        );
        let mut name = String::new();
        for (index, c) in field.chars().enumerate() {
            if index > 0 && c.is_ascii_uppercase() {
                name.push('_');
            }
            name.extend(c.to_lowercase());
        }
        let native_name = name.replace("cheap_rand", "cheaprand");
        let encoded = wire
            .lines()
            .find(|line| line.contains(&format!(", value.{field})")))
            .unwrap();
        let offset = encoded
            .split_once("record[")
            .unwrap()
            .1
            .split_once(':')
            .unwrap()
            .0;
        assert!(
            model.contains(&format!("\"{native_name}\":get64(state,{offset})")),
            "draw-counter event omits or misdecodes {field}"
        );
        assert!(streams.insert(name));
    }
    assert!(!streams.is_empty());
    streams
}

fn stream_gaps(entries: &[Entry], streams: &BTreeSet<String>) -> Vec<String> {
    let rows: BTreeSet<&str> = entries
        .iter()
        .filter_map(|entry| entry.site.strip_prefix("stream:"))
        .collect();
    streams
        .iter()
        .filter(|stream| !rows.contains(stream.as_str()))
        .map(|stream| format!("stream {stream:?} has no inventory row"))
        .chain(
            rows.iter()
                .filter(|row| !streams.contains(**row))
                .map(|row| format!("inventory stream {row} no longer has a source")),
        )
        .collect()
}

#[test]
fn inventory_covers_every_diagnostic_draw_stream() {
    let streams = diagnostic_streams();
    let inventory = parse(INVENTORY).unwrap();
    assert_eq!(
        stream_gaps(&inventory.entries, &streams),
        Vec::<String>::new()
    );
    for stream in &streams {
        let entries: Vec<Entry> = inventory
            .entries
            .iter()
            .filter(|entry| entry.site != format!("stream:{stream}"))
            .cloned()
            .collect();
        assert!(
            stream_gaps(&entries, &streams)
                .iter()
                .any(|gap| gap.contains(stream))
        );
        let mut removed = streams.clone();
        removed.remove(stream);
        assert!(
            stream_gaps(&inventory.entries, &removed)
                .iter()
                .any(|gap| gap.contains("no longer has"))
        );
    }
    let mut added = streams;
    added.insert("gc_pacer_draws".into());
    assert!(
        stream_gaps(&inventory.entries, &added)
            .iter()
            .any(|gap| gap.contains("gc_pacer_draws"))
    );
}

#[test]
fn appended_retired_identity_cannot_reuse_an_existing_site() {
    let mut inventory = parse(INVENTORY).unwrap();
    let mut history: serde_json::Value = serde_json::from_str(IDENTITY_HISTORY).unwrap();
    let mut appended = inventory.entries[0].clone();
    appended.id = "appended.retired".into();
    appended.site = "retired:appended".into();
    appended.disposition = "rejected".into();
    history
        .as_array_mut()
        .unwrap()
        .push(serde_json::json!({"id":appended.id,"site":appended.site}));
    inventory.entries.push(appended);
    check_identity_history(&history.to_string(), &inventory.entries).unwrap();
    let last = history.as_array().unwrap().len() - 1;
    history[last]["site"] = history[0]["site"].clone();
    inventory.entries.last_mut().unwrap().site = inventory.entries[0].site.clone();
    assert!(
        check_identity_history(&history.to_string(), &inventory.entries)
            .unwrap_err()
            .contains("reuses id or site")
    );
}

#[path = "inventory/evidence.rs"]
mod evidence_inventory;
#[path = "inventory/evidence_migrations.rs"]
mod evidence_migrations;

#[test]
fn inventory_admits_only_retired_go_migration_reference_shapes() {
    parse(INVENTORY).unwrap();
    assert!(reference_shape(
        "migration:go:internal/backends/wasm/wasi:TestDeterminismRefuseOffPinStarted"
    ));
    for reference in [
        "migration:migration:go:internal/backends/wasm/wasi:TestMissing",
        "migration:rust:tests/inventory.rs:test",
        "migration:go:internal/model/wasi:TestMissing",
        "migration:go:internal/backends/wasm/wasi:",
        "migration:go:internal/backends/wasm/wasi:../TestMissing",
    ] {
        assert!(!reference_shape(reference), "{reference}");
    }
}

#[test]
fn inventory_rejects_invalid_stable_ids_and_reference_paths() {
    let base: serde_json::Value = serde_json::from_str(INVENTORY).unwrap();
    for id in [
        "Uppercase",
        "two..parts",
        "_first",
        "trailing-",
        "white space",
    ] {
        let mut document = base.clone();
        document["entries"][0]["id"] = id.into();
        assert!(
            parse(&document.to_string())
                .unwrap_err()
                .contains("stable lower-case"),
            "{id}"
        );
    }
    for reference in [
        "rust:tests/inventory.rs",
        "rust:/absolute.rs:test",
        "rust:../escape.rs:test",
        "rust:tests/inventory.rs:",
        "go::TestMissing",
        "corpus:../escape.json",
        "corpus:/absolute.json",
    ] {
        let mut document = base.clone();
        document["entries"][0]["evidence"] = serde_json::json!([reference]);
        assert!(
            parse(&document.to_string())
                .unwrap_err()
                .contains("evidence kind"),
            "{reference}"
        );
    }
    let mut document = base;
    document["entries"] = serde_json::json!([]);
    assert!(
        parse(&document.to_string())
            .unwrap_err()
            .contains("no entries")
    );
}

#[path = "inventory/architecture.rs"]
mod architecture_inventory;
