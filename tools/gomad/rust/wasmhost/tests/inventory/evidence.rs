use super::*;
use std::{fs, path::Path};

fn test_attribute(lines: &[&str]) -> bool {
    let mut depth = 0_i64;
    for line in lines.iter().rev() {
        let blank = strip_literals(line);
        let blank = blank.trim();
        if depth == 0 && blank == "#[test]" {
            return true;
        }
        if blank.is_empty() {
            continue;
        }
        for token in blank.chars().rev() {
            match token {
                ']' => depth += 1,
                '[' => depth -= 1,
                _ => {}
            }
            if depth < 0 {
                return false;
            }
        }
        if depth == 0 && !blank.starts_with("#[") {
            return false;
        }
    }
    false
}

pub(super) fn check_reference(module: &Path, reference: &str) -> Result<(), String> {
    let missing = || {
        format!(
            "inventory evidence {reference:?} does not name an existing test or corpus contract"
        )
    };
    let (kind, tail) = reference.split_once(':').ok_or_else(missing)?;
    if kind == "migration" {
        return super::evidence_migrations::check_reference(module, tail);
    }
    if kind == "corpus" {
        let root = module.join("rust").canonicalize().map_err(|_| missing())?;
        let path = root.join(tail).canonicalize().map_err(|_| missing())?;
        if !path.starts_with(root) || !path.is_file() {
            return Err(missing());
        }
        return Ok(());
    }
    let (path, function) = tail.rsplit_once(':').ok_or_else(missing)?;
    if function.is_empty() || !function.chars().all(|c| c.is_alphanumeric() || c == '_') {
        return Err(missing());
    }
    let found = match kind {
        "rust" => {
            let root = module.join("rust").canonicalize().map_err(|_| missing())?;
            // Existing helper references are crate relative. Operator references name
            // their workspace crate explicitly so they cannot resolve accidentally
            // to a similarly named helper test.
            let relative = Path::new(path);
            let source = if relative.starts_with("gomad") {
                root.join(relative)
            } else {
                root.join("wasmhost").join(relative)
            }
            .canonicalize()
            .map_err(|_| missing())?;
            if !source.starts_with(root)
                || source.extension().is_none_or(|extension| extension != "rs")
            {
                return Err(missing());
            }
            let text = fs::read_to_string(source).map_err(|_| missing())?;
            let masked = mask_multiline_literals(&text);
            let lines: Vec<&str> = masked.lines().collect();
            lines.iter().enumerate().any(|(index, line)| {
                strip_literals(line).contains(&format!("fn {function}("))
                    && test_attribute(&lines[..index])
            })
        }
        "go" => {
            let root = module.canonicalize().map_err(|_| missing())?;
            let directory = root.join(path).canonicalize().map_err(|_| missing())?;
            if !directory.starts_with(&root) {
                return Err(missing());
            }
            let mut found = false;
            for entry in fs::read_dir(directory).map_err(|_| missing())? {
                let path = entry.map_err(|_| missing())?.path();
                if !path.to_string_lossy().ends_with("_test.go") {
                    continue;
                }
                let text = fs::read_to_string(path).map_err(|_| missing())?;
                if text.contains(&format!("func {function}(")) {
                    found = true;
                }
            }
            found
        }
        _ => false,
    };
    if !found {
        return Err(missing());
    }
    Ok(())
}

#[test]
fn inventory_native_and_retained_model_evidence_resolves() {
    let module = Path::new(env!("CARGO_MANIFEST_DIR")).join("../..");
    let inventory = parse(INVENTORY).unwrap();
    let references: BTreeSet<&str> = inventory
        .entries
        .iter()
        .flat_map(|row| row.evidence.iter().map(String::as_str))
        .filter(|reference| {
            reference.starts_with("rust:")
                || reference.starts_with("corpus:")
                || reference.starts_with("go:internal/model/")
        })
        .collect();
    for reference in references {
        check_reference(&module, reference).unwrap();
    }
    check_reference(
        &module,
        "rust:gomad/src/stress/reach_tests.rs:stress_host_perturbations_do_not_change_declared_guest_inputs",
    )
    .unwrap();
    for removed in [
        "rust:tests/inventory.rs:removed_rust_test",
        "go:internal/model/choice:TestRemovedEvidence",
        "corpus:wasmhost/tests/testdata/removed.json",
    ] {
        assert!(
            check_reference(&module, removed)
                .unwrap_err()
                .contains(removed)
        );
    }
}

#[test]
fn inventory_all_evidence_references_resolve() {
    let module = Path::new(env!("CARGO_MANIFEST_DIR")).join("../..");
    let inventory = parse(INVENTORY).unwrap();
    let references: BTreeSet<&str> = inventory
        .entries
        .iter()
        .flat_map(|row| row.evidence.iter().map(String::as_str))
        .collect();
    let unresolved: Vec<String> = references
        .into_iter()
        .filter_map(|reference| check_reference(&module, reference).err())
        .collect();
    assert!(
        unresolved.is_empty(),
        "unresolved inventory evidence:\n{}",
        unresolved.join("\n")
    );
}

#[test]
fn evidence_requires_an_adjacent_test_attribute_across_closed_multiline_attributes() {
    for source in [
        "#[test]\n#[expect(\nclippy::too_many_lines,\nreason = \"bounded test\"\n)]\n#[ignore = \"explicit resource slot\"]\n",
        "#[test]\n// rationale\n#[cfg(all(test, feature = \"fixture\"))]\n",
    ] {
        assert!(test_attribute(&source.lines().collect::<Vec<_>>()));
    }
    for source in [
        "#[test]\nfn previous() {}\n",
        "#[test]\nconst VALUE: &[u8] = &[1, 2];\n",
        "#[test]\n#[expect(\n",
        "#[test]\n)]\n",
        "#[test]\nlet text = \"#[test]\";\n",
    ] {
        assert!(!test_attribute(&source.lines().collect::<Vec<_>>()));
    }
}
