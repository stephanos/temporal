//! Reproducible activity line inventory: semantic Model, feature mapping and shared owners.
//!
//! Counts nonblank, non-comment lines outside `#[cfg(test)]` items. Registry metadata on
//! Model declarations is mapping; moving feature semantics into another file keeps them
//! charged to the feature because every file needs an explicit owner below.
use std::{
    collections::BTreeMap,
    fs,
    io::{self, Write},
    path::Path,
};

/// The owner charged with a counted line.
#[derive(Clone, Copy, Debug, PartialEq, Eq, PartialOrd, Ord)]
pub(crate) enum Owner {
    /// Handwritten semantic feature Model lines (the ratio denominator).
    Semantic,
    /// Handwritten feature mapping, including registry metadata (the ratio numerator).
    Mapping,
    /// Handwritten shared Temporal/realization infrastructure.
    Shared,
    /// The independent shared protocol Model and its contracts.
    Protocol,
    /// Feature- or protocol-owned proofs, checks and tests.
    Proof,
}

/// Path-prefix ownership; the longest matching prefix wins.
const OWNERS: &[(&str, Owner)] = &[
    ("feature/activity/model.rs", Owner::Semantic),
    ("feature/activity/", Owner::Mapping),
    ("feature/activity/tests.rs", Owner::Proof),
    ("feature/activity/policy_checks.rs", Owner::Proof),
    ("feature/activity/proof_checks.rs", Owner::Proof),
    ("feature/activity/deadline_frontier_mapping.rs", Owner::Proof),
    ("feature/activity/compatibility/", Owner::Proof),
    ("realization/src/live.rs", Owner::Mapping),
    ("realization/src/annotations.rs", Owner::Mapping),
    ("realization/src/probes.rs", Owner::Mapping),
    ("realization/src/probes/", Owner::Mapping),
    ("realization/src/lib.rs", Owner::Shared),
    ("realization/src/registry.rs", Owner::Shared),
    ("realization/macros/", Owner::Shared),
    ("realization/tests/", Owner::Proof),
    ("shared/", Owner::Shared),
    ("shared/protocol/", Owner::Protocol),
    ("shared/protocol/activity/tests.rs", Owner::Proof),
    ("shared/protocol/activity/mutants.rs", Owner::Proof),
    ("shared/protocol/activity/parameters_tests.rs", Owner::Proof),
    ("shared/protocol/activity/view_tests.rs", Owner::Proof),
    ("foundation/", Owner::Shared),
    // Conditional views consumed only by the activity claims and probes.
    ("foundation/views.rs", Owner::Mapping),
    ("guest/", Owner::Mapping),
    ("annotate/", Owner::Shared),
    ("api/", Owner::Shared),
    ("testing/", Owner::Proof),
    // Tests inside shared crates are proofs, like every other test tree.
    ("annotate/syntax/tests/", Owner::Proof),
    ("api/tests/", Owner::Proof),
    ("realization/macros/tests/", Owner::Proof),
    ("foundation/tests.rs", Owner::Proof),
    // The activity worker-program executors interpret the feature's programmed responses.
    ("hosts/go/reference/worker/", Owner::Mapping),
];

/// Roots whose every Rust file must have an owner.
const ROOTS: &[&str] = &[
    "feature/activity",
    "realization",
    "shared",
    "foundation",
    "annotate",
    "api",
    "guest",
    "testing",
    "hosts/go/reference/worker",
];

/// Model roots outside the Rust line inventory, with the reason each is excluded.
const EXCLUDED: &[(&str, &str)] = &[
    (
        "design",
        "design notes and the native-symbolic backend prototype crate, which has no activity code",
    ),
    ("dev", "build and gate tooling, independent of any feature"),
    ("docs", "prose"),
    (
        "hosts",
        "generic Go hosts, generated Go bindings and the committed guest binary; the activity worker executors under hosts/go/reference/worker are counted",
    ),
    ("proto", "protobuf schemas; their generated Rust lives in the build OUT_DIR"),
    ("scripts", "shell build helpers"),
];

/// Maximum mapping lines per semantic Model line.
const RATIO: usize = 2;

pub(crate) fn owner(path: &str) -> Option<Owner> {
    OWNERS
        .iter()
        .filter(|(prefix, _)| {
            path == *prefix || (prefix.ends_with('/') && path.starts_with(prefix))
        })
        .max_by_key(|(prefix, _)| prefix.len())
        .map(|(_, owner)| *owner)
}

/// Count one source, charging `#[metadata(...)]` attributes of a semantic file to mapping.
pub(crate) fn count(source: &str, owner: Owner) -> BTreeMap<Owner, usize> {
    let mut counts = BTreeMap::new();
    let mut block_comment = false;
    let mut skip_depth: Option<i64> = None;
    let mut pending_test = false;
    let mut metadata_depth: i64 = 0;
    for line in source.lines() {
        let trimmed = line.trim();
        if block_comment {
            block_comment = !trimmed.contains("*/");
            continue;
        }
        if trimmed.starts_with("/*") {
            block_comment = !trimmed.contains("*/");
            continue;
        }
        if trimmed.is_empty() || trimmed.starts_with("//") {
            continue;
        }
        if let Some(depth) = skip_depth.as_mut() {
            *depth += braces(trimmed);
            if *depth <= 0 {
                skip_depth = None;
            }
            continue;
        }
        let item = if let Some(rest) = test_attribute(trimmed) {
            pending_test = true;
            rest
        } else {
            trimmed
        };
        if pending_test {
            if item.is_empty() || item.starts_with("#[") {
                continue;
            }
            let depth = braces(item);
            if depth > 0 {
                skip_depth = Some(depth);
                pending_test = false;
            } else if item.ends_with(';') || item.ends_with('}') {
                pending_test = false;
            }
            continue;
        }
        let charged = if owner == Owner::Semantic
            && (metadata_depth > 0 || trimmed.starts_with("#[metadata("))
        {
            metadata_depth += parens(trimmed);
            Owner::Mapping
        } else {
            owner
        };
        *counts.entry(charged).or_default() += 1;
    }
    counts
}

/// The remainder after a test-only `cfg` attribute, if the line starts with one.
fn test_attribute(line: &str) -> Option<&str> {
    ["#[cfg(test)]", "#[cfg(all(test"]
        .iter()
        .find(|prefix| line.starts_with(*prefix))
        .map(|_| line.find(")]").map_or("", |end| line[end + 2..].trim()))
}

fn braces(line: &str) -> i64 {
    line.chars().map(|c| i64::from(c == '{') - i64::from(c == '}')).sum()
}

fn parens(line: &str) -> i64 {
    line.chars().map(|c| i64::from(c == '(') - i64::from(c == ')')).sum()
}

fn sources(root: &Path, relative: &str, out: &mut Vec<String>) -> io::Result<()> {
    let path = root.join(relative);
    if path.is_file() {
        let extension = Path::new(relative).extension().and_then(|extension| extension.to_str());
        // Go tests are proofs outside the inventory; Go has no in-file test items.
        let go = extension == Some("go") && !relative.ends_with("_test.go");
        if extension == Some("rs") || go {
            out.push(relative.to_owned());
        }
        return Ok(());
    }
    let mut entries: Vec<_> = fs::read_dir(&path)?
        .map(|entry| entry.map(|entry| entry.file_name()))
        .collect::<io::Result<_>>()?;
    entries.sort();
    for name in entries {
        let name =
            name.to_str().ok_or_else(|| io::Error::other("source path must be UTF-8"))?.to_owned();
        if name == "target" || name.starts_with('.') {
            continue;
        }
        sources(root, &format!("{relative}/{name}"), out)?;
    }
    Ok(())
}

/// The complete inventory under `model`, refusing any unclassified Rust file.
pub(crate) fn inventory(model: &Path) -> io::Result<BTreeMap<String, BTreeMap<Owner, usize>>> {
    let mut files = Vec::new();
    for root in ROOTS {
        sources(model, root, &mut files)?;
    }
    let mut result = BTreeMap::new();
    for file in files {
        let owner =
            owner(&file).ok_or_else(|| io::Error::other(format!("unclassified source: {file}")))?;
        result.insert(file.clone(), count(&fs::read_to_string(model.join(&file))?, owner));
    }
    Ok(result)
}

pub(crate) fn run(model: &Path) -> io::Result<i32> {
    let inventory = inventory(model)?;
    let mut totals: BTreeMap<Owner, usize> = BTreeMap::new();
    let mut stdout = io::stdout().lock();
    for (file, counts) in &inventory {
        for (owner, lines) in counts {
            *totals.entry(*owner).or_default() += lines;
            writeln!(stdout, "{owner:?}\t{lines}\t{file}")?;
        }
    }
    let semantic = totals.get(&Owner::Semantic).copied().unwrap_or(0);
    let mapping = totals.get(&Owner::Mapping).copied().unwrap_or(0);
    for owner in [Owner::Semantic, Owner::Mapping, Owner::Shared, Owner::Protocol, Owner::Proof] {
        writeln!(stdout, "total\t{owner:?}\t{}", totals.get(&owner).copied().unwrap_or(0))?;
    }
    writeln!(
        stdout,
        "generated\tnot counted: descriptor-generated Rust lives only in the build OUT_DIR"
    )?;
    for (root, reason) in EXCLUDED {
        writeln!(stdout, "excluded\t{root}\t{reason}")?;
    }
    let within = semantic > 0 && mapping <= semantic * RATIO;
    let hundredths = (mapping * 100).checked_div(semantic);
    writeln!(
        stdout,
        "ratio\tmapping {mapping} / semantic {semantic} = {} (target <= {RATIO}): {}",
        hundredths.map_or_else(
            || "undefined".to_owned(),
            |value| format!("{}.{:02}", value / 100, value % 100)
        ),
        if within { "within target" } else { "over target" }
    )?;
    Ok(i32::from(!within))
}

#[cfg(test)]
mod tests {
    use super::{BTreeMap, Owner, Path, count, inventory, io, owner};

    #[test]
    fn comments_blank_lines_and_test_items_are_not_counted() {
        let source = "//! doc\n\n/// item\nfn a() {}\n/* block\n still */\n#[cfg(test)]\n#[allow(x)]\nmod tests {\n    fn b() {\n    }\n}\n#[cfg(test)]\nmod external;\nfn c() {}\n";
        assert_eq!(count(source, Owner::Mapping), BTreeMap::from([(Owner::Mapping, 2)]));
        let same_line = "#[cfg(test)] mod external;\nfn a() {}\n#[cfg(all(test, unix))]\nfn helper(\n    value: u8,\n) {\n    let _ = value;\n}\nfn b() {}\n";
        assert_eq!(count(same_line, Owner::Shared), BTreeMap::from([(Owner::Shared, 2)]));
    }

    #[test]
    fn model_registry_metadata_is_charged_to_mapping() {
        let source = "enum Client {\n    #[metadata(temporal(rpc = A::B,\n        fill = f))]\n    Cancel,\n}\n";
        assert_eq!(
            count(source, Owner::Semantic),
            BTreeMap::from([(Owner::Semantic, 3), (Owner::Mapping, 2)])
        );
        assert_eq!(count(source, Owner::Mapping), BTreeMap::from([(Owner::Mapping, 5)]));
    }

    #[test]
    fn longest_prefix_owns_and_unknown_paths_are_unclassified() {
        assert_eq!(owner("feature/activity/model.rs"), Some(Owner::Semantic));
        assert_eq!(owner("feature/activity/temporal.rs"), Some(Owner::Mapping));
        assert_eq!(owner("feature/activity/compatibility/original/x.rs"), Some(Owner::Proof));
        assert_eq!(owner("shared/lens/mod.rs"), Some(Owner::Shared));
        assert_eq!(owner("shared/protocol/activity/model.rs"), Some(Owner::Protocol));
        assert_eq!(owner("realization/src/live.rs"), Some(Owner::Mapping));
        assert_eq!(owner("foundation/views.rs"), Some(Owner::Mapping));
        assert_eq!(owner("foundation/claims.rs"), Some(Owner::Shared));
        assert_eq!(owner("api/tests/codegen.rs"), Some(Owner::Proof));
        assert_eq!(owner("hosts/go/reference/worker/program.go"), Some(Owner::Mapping));
        assert_eq!(owner("feature/workflow/model.rs"), None);
        assert_eq!(owner("realization/src/new_feature_adapter.rs"), None);
    }

    #[test]
    fn current_tree_is_fully_classified() -> io::Result<()> {
        let model = Path::new(env!("CARGO_MANIFEST_DIR")).join("../..");
        let inventory = inventory(&model)?;
        assert!(inventory.contains_key("feature/activity/model.rs"));
        assert!(inventory.contains_key("hosts/go/reference/worker/program.go"));
        assert!(!inventory.keys().any(|file| file.ends_with("_test.go")));
        assert!(inventory.values().any(|counts| counts.contains_key(&Owner::Mapping)));
        Ok(())
    }
}
