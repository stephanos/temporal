use super::*;
use std::{fs, path::Path};

const EFFECTS: &[&str] = &[
    "Instant::now",
    ".elapsed()",
    "SystemTime",
    "/dev/urandom",
    "thread::spawn",
    "thread::sleep",
    "thread::scope",
    "thread::Builder",
    ".spawn(",
    ".spawn_scoped(",
    "recv_timeout",
    "Command::new",
    "HashMap",
    "HashSet",
    ".sort_unstable",
];

fn host_sites() -> Result<BTreeSet<String>, String> {
    let helper = Path::new(env!("CARGO_MANIFEST_DIR")).join("src");
    let operator = helper.join("../../gomad/src");
    let mut sites = BTreeSet::new();
    for (owner, root, roots) in [
        ("wasmhost", helper, vec!["lib.rs"]),
        ("gomad", operator, vec!["lib.rs", "main.rs"]),
    ] {
        let (_, files) = source_scan(&root, &roots)?;
        for file in files {
            let source = fs::read_to_string(root.join(&file)).map_err(|e| e.to_string())?;
            let (lines, _) = production_lines(&source)?;
            let masked = mask_multiline_literals(&source);
            let masked_lines: Vec<&str> = masked.lines().collect();
            let mut function = "package".to_string();
            let mut counts = BTreeMap::new();
            let mut functions = BTreeMap::new();
            let mut indent = None;
            for (number, line) in lines {
                let blank = strip_literals(masked_lines[number - 1]);
                if indent.is_some_and(|start| {
                    line.trim() == "}" && line.len() - line.trim_start().len() == start
                }) {
                    function = "package".into();
                    indent = None;
                }
                if let Some((_, rest)) = blank.split_once("fn ") {
                    function = rest
                        .trim_start_matches('$')
                        .chars()
                        .take_while(|c| c.is_alphanumeric() || *c == '_')
                        .collect();
                    if function.is_empty() {
                        return Err(format!("{file}: unrecognized function"));
                    }
                    let ordinal = functions.entry(function.clone()).or_insert(0usize);
                    *ordinal += 1;
                    if *ordinal > 1 {
                        function = format!("{function}[{ordinal}]");
                    }
                    indent = Some(line.len() - line.trim_start().len());
                }
                for pattern in EFFECTS {
                    let count = if *pattern == "/dev/urandom" {
                        line.matches(pattern).count()
                    } else {
                        blank.matches(pattern).count()
                    };
                    if count > 0 {
                        *counts.entry((function.clone(), *pattern)).or_insert(0usize) += count;
                    }
                }
            }
            for ((function, pattern), count) in counts {
                sites.insert(format!("{owner}/src/{file}|{function}|{pattern}|{count}"));
            }
        }
    }
    Ok(sites)
}

fn gaps(entries: &[Entry], derived: &BTreeSet<String>) -> Vec<String> {
    let rows: BTreeMap<&str, &Entry> = entries
        .iter()
        .filter_map(|entry| {
            entry
                .site
                .strip_prefix("host-rust:")
                .map(|site| (site, entry))
        })
        .collect();
    let mut gaps = Vec::new();
    for site in derived {
        if rows
            .get(site.as_str())
            .is_none_or(|row| row.disposition == "rejected")
        {
            gaps.push(format!("host effect {site:?} has no inventory row"));
        }
    }
    for (site, row) in rows {
        if !derived.contains(site) && !retired_identity(row, IDENTITY_HISTORY) {
            gaps.push(format!("host effect {site:?} no longer has a source"));
        }
    }
    gaps
}

#[test]
fn receiving_host_effect_sites_are_inventoried() {
    let sites = host_sites().unwrap();
    let inventory = parse(INVENTORY).unwrap();
    assert_eq!(gaps(&inventory.entries, &sites), Vec::<String>::new());
    for required in [
        "wasmhost/src/linked.rs|",
        "wasmhost/src/linked_cluster.rs|",
        "gomad/src/campaign/prepare/process.rs|",
        "gomad/src/campaign/mod.rs|",
        "gomad/src/cli/run_guest.rs|",
        "gomad/src/evidence/store/publish.rs|",
        "gomad/src/slots.rs|",
        "gomad/src/qualify/cells.rs|",
        "gomad/src/sweep/process.rs|",
        "gomad/src/backstop.rs|",
        "gomad/src/runtime_qualification.rs|",
    ] {
        assert!(
            sites.iter().any(|site| site.starts_with(required)),
            "missing original receiving scope {required}"
        );
    }
    for site in &sites {
        let entries: Vec<Entry> = inventory
            .entries
            .iter()
            .filter(|entry| entry.site != format!("host-rust:{site}"))
            .cloned()
            .collect();
        assert!(
            gaps(&entries, &sites).iter().any(|gap| gap.contains(site)),
            "{site}"
        );
        let mut added = sites.clone();
        let (prefix, count) = site.rsplit_once('|').unwrap();
        let changed = format!("{prefix}|{}", count.parse::<u64>().unwrap() + 1);
        added.insert(changed.clone());
        assert!(
            gaps(&inventory.entries, &added)
                .iter()
                .any(|gap| gap.contains(&changed)),
            "{changed}"
        );
    }
}
