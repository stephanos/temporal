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

/// Patterns the helper source scan treats as time, entropy, thread or
/// hash-iteration sites.
const SOURCE_PATTERNS: &[&str] = &[
    "Instant::now",
    "SystemTime",
    "/dev/urandom",
    "thread::spawn",
    "thread::Builder",
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
        let balanced = first.matches('{').count() == first.matches('}').count();
        let end = first.trim_end();
        if balanced && (end.ends_with(';') || end.ends_with(',')) {
            if let Some(name) = first.trim().strip_prefix("mod ") {
                test_modules.push(name.trim_end_matches(';').to_string());
            }
            index += 1;
            continue;
        }
        let mut depth = 0i64;
        let mut opened = false;
        loop {
            let line = strip_literals(
                lines
                    .get(index)
                    .ok_or_else(|| format!("cfg(test) item at line {} never closes", start + 1))?,
            );
            for c in line.chars() {
                match c {
                    '{' => {
                        depth += 1;
                        opened = true;
                    }
                    '}' => depth -= 1,
                    _ => {}
                }
            }
            index += 1;
            if opened && depth == 0 {
                let last = lines[index - 1];
                if index - 1 != start
                    && (!last.trim_start().starts_with('}')
                        || last.len() - last.trim_start().len() != indent)
                {
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
/// `file|pattern|count`. Production files are those reachable from `main.rs`
/// through `mod` declarations outside `#[cfg(test)]` items; test-only modules
/// are never reached.
fn source_sites(src: &std::path::Path) -> Result<BTreeSet<String>, String> {
    let mut sites = BTreeSet::new();
    let mut pending = vec!["main.rs".to_string()];
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
        let owns_directory = stem == "main" || stem.ends_with("/mod") || stem == "mod";
        for (index, (_, line)) in lines.iter().enumerate() {
            let trimmed = line.trim();
            let declared = ["mod ", "pub mod ", "pub(crate) mod "]
                .iter()
                .find_map(|prefix| trimmed.strip_prefix(prefix))
                .and_then(|rest| rest.strip_suffix(';'));
            let Some(name) = declared else {
                continue;
            };
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
        if !sites.contains(*row) {
            gaps.push(format!(
                "inventory row names helper source site {row} that the sources no longer have"
            ));
        }
    }
    for file in &allowed {
        if !sites
            .iter()
            .any(|site| site.starts_with(&format!("{file}|")))
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
}

const ENGINE: &str = include_str!("../src/engine.rs");
const MANIFEST: &str = include_str!("../Cargo.toml");

/// Derives every Wasmtime setting the helper engine applies: each builder
/// call on the `Config` in `engine::prepare` (with its arguments and any
/// `cfg` attribute) and the wasmtime Cargo feature set.
fn engine_settings(engine: &str, manifest: &str) -> Result<Vec<String>, String> {
    let start = engine
        .find("pub(crate) fn prepare(")
        .ok_or("engine.rs has no prepare")?;
    let body = &engine[start..];
    let end = body
        .find("Engine::new(&config)")
        .ok_or("prepare builds no Engine")?;
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
        return Err("prepare creates no Config".into());
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
        rows.difference(&derived).collect::<Vec<_>>(),
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
        ".wasm_multi_memory(false);",
        ".wasm_multi_memory(false)\n        .wasm_threads(true);",
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
