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
// Matches the reviewed prefix in determinism/identity_history.go.
const ACCEPTED_IDENTITY_COUNT: usize = 263;
const ACCEPTED_IDENTITY_SHA256: &str =
    "571a435d3eed2e5533450ef8210e3ad0f1d4e779ffc0853287d31775a1b66475";

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

#[derive(Debug, Deserialize)]
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

fn parse(text: &str) -> Result<Inventory, String> {
    let inventory: Inventory = serde_json::from_str(text).map_err(|error| error.to_string())?;
    if inventory.schema != SCHEMA {
        return Err(format!("inventory schema {:?}", inventory.schema));
    }
    let reviewed = &inventory.reviewed;
    let date = reviewed.date.as_bytes();
    if date.len() != 10
        || date[4] != b'-'
        || date[7] != b'-'
        || date
            .iter()
            .enumerate()
            .any(|(index, byte)| index != 4 && index != 7 && !byte.is_ascii_digit())
    {
        return Err("inventory reviewed date must be YYYY-MM-DD".into());
    }
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
        if entry.evidence.iter().any(|reference| {
            !["go:", "rust:", "corpus:"]
                .iter()
                .any(|kind| reference.starts_with(kind))
        }) {
            return Err(format!("inventory row {:?} evidence kind", entry.id));
        }
        if (entry.disposition == "escape")
            != (!entry.justification.is_empty() && !entry.readme_anchor.is_empty())
        {
            return Err(format!("inventory row {:?} escape justification", entry.id));
        }
    }
    check_identity_history(IDENTITY_HISTORY, &inventory.entries)?;
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
        == "go:internal/backends/wasm/wasi:TestDeterminismInventoryRejectedImportsAreRefused"));
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
        return Ok(None);
    };
    let Some(name) = declaration.strip_suffix(';') else {
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

fn production_lines(text: &str) -> Result<ProductionLines, String> {
    let lines: Vec<&str> = text.lines().collect();
    let (mut kept, mut test_modules) = (Vec::new(), Vec::new());
    let mut index = 0;
    while index < lines.len() {
        if lines[index].trim() != "#[cfg(test)]" {
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
        if let Some(name) = module_declaration(first.trim())? {
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
                lines
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
    let mut sites = BTreeSet::new();
    let mut pending = vec![
        "main.rs".to_string(),
        "lib.rs".to_string(),
        "../build.rs".to_string(),
    ];
    let mut seen = BTreeSet::new();
    while let Some(relative) = pending.pop() {
        if !seen.insert(relative.clone()) {
            continue;
        }
        let text = std::fs::read_to_string(src.join(&relative))
            .map_err(|error| format!("{relative}: {error}"))?;
        let (lines, _) = production_lines(&text).map_err(|error| format!("{relative}: {error}"))?;
        let directory = relative
            .rsplit_once('/')
            .map_or("", |(directory, _)| directory);
        let stem = relative.strip_suffix(".rs").unwrap();
        let owns_directory = matches!(stem, "main" | "lib" | "mod") || stem.ends_with("/mod");
        for (index, (_, line)) in lines.iter().enumerate() {
            let trimmed = line.trim();
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
    Ok(sites)
}

fn source_gaps(entries: &[Entry], sites: &BTreeSet<String>) -> Vec<String> {
    let rows: BTreeSet<&str> = entries
        .iter()
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
    let mut gaps = Vec::new();
    for site in sites {
        let file = site.split('|').next().unwrap();
        if !allowed.contains(file) && !rows.contains(site.as_str()) {
            gaps.push(format!("helper source site {site} has no inventory row"));
        }
    }
    for row in &rows {
        if !sites.contains(*row)
            && !entries.iter().any(|entry| {
                entry.site == format!("helper-source:{row}")
                    && retired_identity(entry, IDENTITY_HISTORY)
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
