use super::*;
use std::{fs, path::Path};

const EFFECTS: &[&str] = &[
    "Instant::now",
    "clock_gettime(",
    "linked::monotonic_nanos(",
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

#[derive(Clone, Deserialize)]
#[serde(deny_unknown_fields)]
pub(super) struct Migration {
    old_id: String,
    old_site: String,
    new_id: String,
    new_site: String,
    reason: String,
    kind: String,
}

pub(super) const MIGRATIONS: &str =
    include_str!("../../../../internal/backends/wasm/determinism/effect-migrations.json");

pub(super) fn migrated_identities(
    entries: &[Entry],
    derived: &BTreeSet<String>,
    history: &str,
    migrations: &[Migration],
    prefix: &str,
) -> Result<BTreeSet<String>, String> {
    let records: Vec<IdentityRecord> = serde_json::from_str(history).map_err(|e| e.to_string())?;
    let mut retired = BTreeSet::new();
    let mut destinations = BTreeSet::new();
    if !matches!(prefix, "host-rust:" | "helper-source:") {
        return Err("unsupported source migration scope".into());
    }
    for migration in migrations {
        if !migration.old_site.starts_with("host-rust:")
            && !migration.old_site.starts_with("helper-source:")
        {
            return Err("unsupported migration source kind".into());
        }
        if !migration.old_site.starts_with(prefix) {
            continue;
        }
        let old = entries.iter().find(|entry| entry.id == migration.old_id);
        let new = entries.iter().find(|entry| entry.id == migration.new_id);
        let Some(old) = old else {
            // Synthetic subset checks have no retirement to authorize. The full
            // inventory's immutable identity ledger independently forbids removal.
            continue;
        };
        let Some(new) = new else {
            return Err("migration removed its replacement identity".into());
        };
        let historical = |entry: &Entry| {
            records
                .iter()
                .any(|record| record.id == entry.id && record.site == entry.site)
        };
        let parts = if prefix == "host-rust:" { 4 } else { 3 };
        let count = |site: &str| {
            site.rsplit_once('|')
                .and_then(|(_, count)| count.parse::<u64>().ok())
        };
        let old_count = count(&old.site);
        let new_count = count(&new.site);
        let kind_valid = match migration.kind.as_str() {
            "relocation" => old_count == new_count,
            "multiplicity" => {
                old_count != new_count
                    && old.site.rsplit_once('|').map(|(site, _)| site)
                        == new.site.rsplit_once('|').map(|(site, _)| site)
            }
            _ => false,
        };
        let old_effect = old
            .site
            .rsplit_once('|')
            .and_then(|(site, _)| site.rsplit_once('|'));
        let new_effect = new
            .site
            .rsplit_once('|')
            .and_then(|(site, _)| site.rsplit_once('|'));
        if old.site != migration.old_site
            || new.site != migration.new_site
            || old.site.split('|').count() != parts
            || new.site.split('|').count() != parts
            || old_count.is_none_or(|count| count == 0)
            || new_count.is_none_or(|count| count == 0)
            || !kind_valid
            || !historical(old)
            || !historical(new)
            || !old.site.starts_with(prefix)
            || !new.site.starts_with(prefix)
            || old_effect.map(|(_, effect)| effect) != new_effect.map(|(_, effect)| effect)
            || old.layer != new.layer
            || old.class != new.class
            || old.disposition != "rejected"
            || new.disposition == "rejected"
            || old.rule != migration.reason
            || migration.reason.is_empty()
            || old.evidence.is_empty()
            || new.evidence.is_empty()
            || derived.contains(old.site.trim_start_matches(prefix))
            || !derived.contains(new.site.trim_start_matches(prefix))
            || !retired.insert(old.id.clone())
            || !destinations.insert(new.id.clone())
        {
            return Err(format!(
                "invalid effect migration {:?}: {} -> {}",
                migration.old_id, migration.old_site, migration.new_site
            ));
        }
    }
    if !retired.is_disjoint(&destinations) {
        return Err("host migration contains a chain or cycle".into());
    }
    Ok(retired)
}

fn gaps(entries: &[Entry], derived: &BTreeSet<String>) -> Vec<String> {
    let migrations: Vec<Migration> = serde_json::from_str(MIGRATIONS).unwrap();
    let (retired, migration_error) = match migrated_identities(
        entries,
        derived,
        IDENTITY_HISTORY,
        &migrations,
        "host-rust:",
    ) {
        Ok(retired) => (retired, None),
        Err(error) => (BTreeSet::new(), Some(error)),
    };
    let rows: BTreeMap<&str, &Entry> = entries
        .iter()
        .filter_map(|entry| {
            entry
                .site
                .strip_prefix("host-rust:")
                .map(|site| (site, entry))
        })
        .collect();
    let mut gaps: Vec<String> = migration_error.into_iter().collect();
    for site in derived {
        if rows
            .get(site.as_str())
            .is_none_or(|row| row.disposition == "rejected")
        {
            gaps.push(format!("host effect {site:?} has no inventory row"));
        }
    }
    for (site, row) in rows {
        if !derived.contains(site)
            && !retired_identity(row, IDENTITY_HISTORY)
            && !retired.contains(&row.id)
        {
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
        assert!(!gaps(&entries, &sites).is_empty(), "{site}");
        let mut added = sites.clone();
        let (prefix, count) = site.rsplit_once('|').unwrap();
        let changed = format!("{prefix}|{}", count.parse::<u64>().unwrap() + 1);
        added.insert(changed.clone());
        assert!(!gaps(&inventory.entries, &added).is_empty(), "{changed}");
    }
}

#[test]
fn host_migrations_preserve_exact_identity_and_active_receiving_sites() {
    let inventory = parse(INVENTORY).unwrap();
    let migrations: Vec<Migration> = serde_json::from_str(MIGRATIONS).unwrap();
    assert!(!migrations.is_empty());
    for migration in &migrations {
        assert!(
            inventory
                .entries
                .iter()
                .any(|entry| entry.id == migration.old_id)
        );
        assert!(
            inventory
                .entries
                .iter()
                .any(|entry| entry.id == migration.new_id)
        );
    }
    let helper = Path::new(env!("CARGO_MANIFEST_DIR")).join("src");
    for (prefix, sites) in [
        ("host-rust:", host_sites().unwrap()),
        (
            "helper-source:",
            source_scan(&helper, &["lib.rs"]).unwrap().0,
        ),
    ] {
        let migrations: Vec<Migration> = migrations
            .iter()
            .filter(|migration| migration.old_site.starts_with(prefix))
            .cloned()
            .collect();
        assert!(!migrations.is_empty());
        let validate =
            |entries: &[Entry], sites: &BTreeSet<String>, history: &str, changes: &[Migration]| {
                migrated_identities(entries, sites, history, changes, prefix)
            };
        validate(&inventory.entries, &sites, IDENTITY_HISTORY, &migrations).unwrap();
        for index in 0..migrations.len() {
            for field in [
                "old_site",
                "new_site",
                "new_id",
                "kind",
                "reason",
                "kind_swap",
            ] {
                let mut changed = migrations.clone();
                match field {
                    "old_site" => changed[index].old_site.push('x'),
                    "new_site" => changed[index].new_site.push('x'),
                    "new_id" => changed[index].new_id = changed[index].old_id.clone(),
                    "kind" => changed[index].kind = "unreviewed".into(),
                    "kind_swap" => {
                        changed[index].kind = if changed[index].kind == "relocation" {
                            "multiplicity".into()
                        } else {
                            "relocation".into()
                        }
                    }
                    _ => changed[index].reason.clear(),
                }
                assert!(validate(&inventory.entries, &sites, IDENTITY_HISTORY, &changed).is_err());
            }
            let mut missing = sites.clone();
            missing.remove(migrations[index].new_site.trim_start_matches(prefix));
            assert!(validate(&inventory.entries, &missing, IDENTITY_HISTORY, &migrations).is_err());
            let mut restored = sites.clone();
            restored.insert(migrations[index].old_site.trim_start_matches(prefix).into());
            assert!(
                validate(&inventory.entries, &restored, IDENTITY_HISTORY, &migrations).is_err()
            );
            let mut rejected = inventory.entries.clone();
            rejected
                .iter_mut()
                .find(|entry| entry.id == migrations[index].new_id)
                .unwrap()
                .disposition = "rejected".into();
            assert!(validate(&rejected, &sites, IDENTITY_HISTORY, &migrations).is_err());
            let forged = IDENTITY_HISTORY.replace(&migrations[index].new_id, "forged-id");
            assert!(validate(&inventory.entries, &sites, &forged, &migrations).is_err());
        }
        assert_multiplicity_cannot_change_coordinate(
            &inventory.entries,
            &sites,
            &migrations,
            prefix,
        );
        let mut duplicated = migrations.clone();
        duplicated.push(migrations[0].clone());
        assert!(validate(&inventory.entries, &sites, IDENTITY_HISTORY, &duplicated).is_err());
    }
}

fn assert_multiplicity_cannot_change_coordinate(
    entries: &[Entry],
    sites: &BTreeSet<String>,
    migrations: &[Migration],
    prefix: &str,
) {
    for migration in migrations
        .iter()
        .filter(|migration| migration.kind == "multiplicity")
    {
        let (old_coordinate, old_count) = migration.old_site.rsplit_once('|').unwrap();
        let effect = old_coordinate.rsplit_once('|').unwrap().1;
        let unrelated = entries
            .iter()
            .find(|entry| {
                let Some((coordinate, count)) = entry.site.rsplit_once('|') else {
                    return false;
                };
                entry.disposition != "rejected"
                    && entry.site.starts_with(prefix)
                    && coordinate != old_coordinate
                    && count != old_count
                    && coordinate
                        .rsplit_once('|')
                        .is_some_and(|(_, token)| token == effect)
                    && sites.contains(entry.site.trim_start_matches(prefix))
            })
            .unwrap();
        let mut changed = migration.clone();
        changed.new_id.clone_from(&unrelated.id);
        changed.new_site.clone_from(&unrelated.site);
        assert!(migrated_identities(entries, sites, IDENTITY_HISTORY, &[changed], prefix).is_err());
    }
}

#[test]
fn unlisted_appended_host_retirement_is_not_a_source_migration() {
    let inventory = parse(INVENTORY).unwrap();
    let mut unlisted = inventory.entries;
    let mut retired = unlisted
        .iter()
        .find(|entry| entry.site.starts_with("host-rust:"))
        .unwrap()
        .clone();
    retired.id = "host.unlisted.appended-retirement".into();
    retired.site = "host-rust:gomad/src/invented.rs|missing|Instant::now|1".into();
    retired.disposition = "rejected".into();
    unlisted.push(retired);
    assert_ne!(
        gaps(&unlisted, &host_sites().unwrap()),
        Vec::<String>::new()
    );
}
