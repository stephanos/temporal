use super::roles::{Files, candidate_role, fields, merge, native_role_inputs, path, pin, read};
use crate::{
    cycle::{build::Role, custody, process::Runner},
    measurement::{build, inputs::authenticate_checkout},
};
use serde_json::{Value, json};
use std::{
    collections::BTreeSet,
    fs::{self, OpenOptions},
    io::Write,
    path::{Component, Path, PathBuf},
    sync::atomic::AtomicBool,
};

fn absolute(base: &Path, value: &Value) -> Result<PathBuf, String> {
    let text = value.as_str().ok_or("public binding path missing")?;
    let path = Path::new(text);
    if path
        .components()
        .any(|part| matches!(part, Component::ParentDir))
    {
        return Err("public binding parent traversal".into());
    }
    let path = if path.is_absolute() {
        path.to_owned()
    } else {
        base.join(path)
    };
    custody::clean_absolute(&path)?;
    if path.canonicalize().map_err(|e| e.to_string())? != path {
        return Err("noncanonical public binding path".into());
    }
    Ok(path)
}
fn expected_file(files: &mut Files, path: &Path, expected: &Value) -> Result<(), String> {
    pin(files, path)?;
    if expected.as_str()
        != files
            .get(path.to_str().ok_or("non-UTF8 public path")?)
            .map(String::as_str)
    {
        return Err("changed public binding input".into());
    }
    Ok(())
}
fn tree(root: &Path, files: &mut Files) -> Result<(), String> {
    for entry in fs::read_dir(root).map_err(|e| e.to_string())? {
        let entry = entry.map_err(|e| e.to_string())?;
        if matches!(
            entry.file_name().to_str(),
            Some(".git" | ".flow" | "__pycache__")
        ) {
            continue;
        }
        let kind = entry.file_type().map_err(|e| e.to_string())?;
        if kind.is_symlink() {
            return Err("symlink in public input tree".into());
        }
        if kind.is_dir() {
            tree(&entry.path(), files)?;
        } else {
            pin(files, &entry.path())?;
        }
    }
    Ok(())
}

fn static_inputs(template: &Path) -> Result<(Value, Files), String> {
    let mut files = Files::new();
    pin(&mut files, template)?;
    let mut declared = read(template)?;
    fields(&declared, &["schema", "files", "trees"], &[])?;
    if declared["schema"] != "gomad.rust-public-bindings/v1" {
        return Err("foreign public verb binding template".into());
    }
    if declared["trees"].get("goldens").is_none() {
        return Err("public raw golden inventory missing".into());
    }
    let base = template.parent().ok_or("public binding parent missing")?;
    for binding in declared["files"]
        .as_object_mut()
        .ok_or("public file bindings missing")?
        .values_mut()
    {
        fields(binding, &["path", "sha256"], &[])?;
        let path = absolute(base, &binding["path"])?;
        expected_file(&mut files, &path, &binding["sha256"])?;
        binding["path"] = json!(path);
    }
    for binding in declared["trees"]
        .as_object_mut()
        .ok_or("public tree bindings missing")?
        .values_mut()
    {
        fields(binding, &["path", "inventory", "sha256"], &[])?;
        let root = absolute(base, &binding["path"])?;
        let inventory = absolute(base, &binding["inventory"])?;
        expected_file(&mut files, &inventory, &binding["sha256"])?;
        let bytes = custody::read_file(&inventory, 64 << 20)?;
        let mut listed = Files::new();
        for line in std::str::from_utf8(&bytes)
            .map_err(|e| e.to_string())?
            .lines()
        {
            let (sha, name) = line
                .split_once("  ")
                .ok_or("malformed public tree inventory")?;
            let relative = Path::new(name);
            if !custody::valid_digest(sha)
                || relative.is_absolute()
                || relative
                    .components()
                    .any(|p| !matches!(p, Component::Normal(_)))
                || name.is_empty()
            {
                return Err("unsafe public tree inventory member".into());
            }
            let member = root.join(relative);
            let key = member
                .to_str()
                .ok_or("non-UTF8 inventory member")?
                .to_owned();
            if listed.insert(key, sha.to_owned()).is_some() {
                return Err("duplicate public tree inventory member".into());
            }
        }
        if listed.is_empty() {
            return Err("empty public tree inventory".into());
        }
        custody::check_files(&listed)?;
        let mut actual = Files::new();
        tree(&root, &mut actual)?;
        if actual != listed {
            return Err("incomplete public tree inventory".into());
        }
        merge(&mut files, listed)?;
        binding["path"] = json!(root);
        binding["inventory"] = json!(inventory);
    }
    Ok((declared, files))
}

fn known_role(name: &str) -> bool {
    name == "diagnostic" || Role::parse(name).is_ok()
}
fn check_roles(config: &Value, declared: &Value, inventory: &Value) -> Result<(), String> {
    fields(&config["public_verbs"], &["bindings"], &["required_roles"])?;
    let entries = inventory["roles"]
        .as_object()
        .ok_or("public role inventory missing")?;
    let default = json!(["release"]);
    let selected = config["public_verbs"]
        .get("required_roles")
        .unwrap_or(&default)
        .as_array()
        .ok_or("public required roles must be an array")?;
    let mut required = BTreeSet::from(["release"]);
    let mut seen = BTreeSet::new();
    for value in selected {
        let name = value
            .as_str()
            .filter(|name| known_role(name))
            .ok_or("unknown required public role")?;
        if !seen.insert(name) {
            return Err("duplicate required public role".into());
        }
        required.insert(name);
    }
    if required.iter().any(|name| !entries.contains_key(*name)) {
        return Err("missing consumed public role".into());
    }
    for (name, entry) in entries {
        if !known_role(name) {
            return Err("unknown supplied public role".into());
        }
        if declared["files"].get(name).is_none() {
            return Err("public role has no executable file binding".into());
        }
        if name == "diagnostic" {
            fields(entry, &["kind", "build_receipt", "sha256"], &["admission"])?;
            if entry["kind"] != "native" {
                return Err("public diagnostic requires native measurement role".into());
            }
        } else {
            fields(entry, &["kind", "build_receipt", "sha256"], &[])?;
            if entry["kind"] != "cycle" {
                return Err("public role requires current cycle build".into());
            }
        }
    }
    for name in declared["files"]
        .as_object()
        .ok_or("public file bindings missing")?
        .keys()
    {
        if known_role(name) && !entries.contains_key(name) {
            return Err("public executable has no authenticated role".into());
        }
    }
    Ok(())
}

fn authenticate_goldens(
    snapshot: &custody::Snapshot,
    repo: &Path,
    root: &Path,
) -> Result<Files, String> {
    let corpus = repo.join("tools/gomad/testdata/parity");
    if !root.starts_with(&corpus) {
        return Err("public raw golden tree is outside candidate corpus".into());
    }
    let authoritative: Files = snapshot
        .files
        .iter()
        .filter(|(name, _)| {
            Path::new(name).strip_prefix(&corpus).is_ok_and(|relative| {
                relative.components().any(|part| {
                    part.as_os_str()
                        .to_str()
                        .is_some_and(|name| name.starts_with("golden"))
                })
            })
        })
        .map(|(name, sha)| (name.clone(), sha.clone()))
        .collect();
    let mut selected = Files::new();
    tree(root, &mut selected)?;
    if authoritative.is_empty()
        || selected.is_empty()
        || selected
            .iter()
            .any(|(name, sha)| authoritative.get(name) != Some(sha))
    {
        return Err("public raw golden bytes are not authenticated candidate corpus".into());
    }
    custody::check_files(&authoritative)?;
    Ok(authoritative)
}

pub(crate) fn public_inputs(
    config: &Value,
    template: &Path,
    runner: &mut Runner,
    cancelled: &AtomicBool,
) -> Result<(Value, Files), String> {
    let (declared, mut files) = static_inputs(template)?;
    let release = declared["files"]["release"]["path"]
        .as_str()
        .ok_or("public release binding missing")?;
    if config["binary"] != release
        || declared["files"]["release"]["sha256"] != config["binary_sha256"]
    {
        return Err("public release differs from candidate".into());
    }
    let builds = Path::new(
        declared["files"]["build-receipts"]["path"]
            .as_str()
            .ok_or("public build receipts missing")?,
    );
    let inventory = read(builds)?;
    fields(
        &inventory,
        &[
            "schema",
            "source_inventory",
            "source_inventory_sha256",
            "roles",
        ],
        &[],
    )?;
    if inventory["schema"] != "gomad.rust-public-builds/v1" {
        return Err("foreign public build inventory".into());
    }
    check_roles(config, &declared, &inventory)?;
    let base = builds.parent().ok_or("build inventory parent missing")?;
    for (name, entry) in inventory["roles"]
        .as_object()
        .ok_or("public role inventory missing")?
    {
        let receipt = absolute(base, &entry["build_receipt"])?;
        expected_file(&mut files, &receipt, &entry["sha256"])?;
        let mut binding = json!({"binary":declared["files"][name]["path"],"build_receipt":receipt});
        let closure = if name == "diagnostic" {
            if let Some(admission) = entry.get("admission") {
                binding["admission"] = json!(absolute(base, admission)?);
            }
            let proof = authenticate_checkout(path(config, "repo")?)?;
            native_role_inputs(
                &proof,
                &binding,
                &build::Role::Diagnostic(vec!["diagnostic-profile".into()]),
            )?
        } else {
            candidate_role(config, &binding, name, runner, cancelled)?.1
        };
        merge(&mut files, closure)?;
    }
    let source = absolute(
        builds.parent().ok_or("build inventory parent missing")?,
        &inventory["source_inventory"],
    )?;
    expected_file(&mut files, &source, &inventory["source_inventory_sha256"])?;
    let expected: custody::Snapshot =
        serde_json::from_value(read(&source)?).map_err(|e| e.to_string())?;
    let private = super::private_context(runner)?;
    let snapshot = custody::Snapshot::observe(path(config, "repo")?, runner, cancelled, &private)?;
    if expected != snapshot {
        return Err("public source inventory incomplete or foreign".into());
    }
    merge(
        &mut files,
        authenticate_goldens(
            &snapshot,
            path(config, "repo")?,
            path(&declared["trees"]["goldens"], "path")?,
        )?,
    )?;
    merge(&mut files, snapshot.files.clone())?;
    snapshot.verify(path(config, "repo")?, runner, cancelled, &private)?;
    custody::check_files(&files)?;
    Ok((declared, files))
}

pub(crate) fn freeze_public_binding(
    config: &mut Value,
    output: &Path,
    runner: &mut Runner,
    cancelled: &AtomicBool,
) -> Result<Files, String> {
    fields(&config["public_verbs"], &["bindings"], &["required_roles"])?;
    let template = Path::new(
        config["public_verbs"]["bindings"]
            .as_str()
            .ok_or("public binding missing")?,
    );
    let raw = custody::read_file(template, 64 << 20)?;
    if config["files"][template.to_str().ok_or("non-UTF8 public template")?]
        != custody::digest(&raw)
    {
        return Err("unsealed public verb binding template".into());
    }
    let (declared, mut files) = public_inputs(config, template, runner, cancelled)?;
    let destination = output
        .parent()
        .ok_or("freeze output parent missing")?
        .join("public-verbs-bindings.json");
    let bytes = crate::evidence::canonical::encode(&declared)?;
    OpenOptions::new()
        .write(true)
        .create_new(true)
        .open(&destination)
        .map_err(|e| e.to_string())?
        .write_all(&bytes)
        .map_err(|e| e.to_string())?;
    pin(&mut files, &destination)?;
    config["public_verbs"]["bindings"] = json!(destination);
    config["files"][destination.to_str().ok_or("non-UTF8 public destination")?] =
        json!(custody::digest(&bytes));
    Ok(files)
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn current_selected_public_roles_require_every_consumed_and_supplied_role() {
        let config = json!({"public_verbs":{"bindings":"/public.json","required_roles":["release","diagnostic"]}});
        let declared = json!({"files":{"release":{},"diagnostic":{}}});
        let inventory = json!({"roles":{"release":{"kind":"cycle","build_receipt":"/release.json","sha256":"a".repeat(64)},"diagnostic":{"kind":"native","build_receipt":"/diagnostic.json","sha256":"b".repeat(64)}}});
        check_roles(&config, &declared, &inventory).unwrap();
        let mut missing = inventory.clone();
        missing["roles"]
            .as_object_mut()
            .unwrap()
            .remove("diagnostic");
        assert!(
            check_roles(&config, &declared, &missing)
                .unwrap_err()
                .contains("missing consumed public role")
        );
        let mut duplicate = config.clone();
        duplicate["public_verbs"]["required_roles"] = json!(["release", "release"]);
        assert!(
            check_roles(&duplicate, &declared, &inventory)
                .unwrap_err()
                .contains("duplicate required public role")
        );
        let mut unknown = config.clone();
        unknown["public_verbs"]["required_roles"] = json!(["legacy-fixture"]);
        assert!(
            check_roles(&unknown, &declared, &inventory)
                .unwrap_err()
                .contains("unknown required public role")
        );
        let mut wrong_kind = inventory.clone();
        wrong_kind["roles"]["diagnostic"]["kind"] = json!("cycle");
        assert!(
            check_roles(&config, &declared, &wrong_kind)
                .unwrap_err()
                .contains("requires native measurement role")
        );
        let mut unbound = declared.clone();
        unbound["files"]
            .as_object_mut()
            .unwrap()
            .remove("diagnostic");
        assert!(
            check_roles(&config, &unbound, &inventory)
                .unwrap_err()
                .contains("no executable file binding")
        );
        let mut unreceipted = declared.clone();
        unreceipted["files"]["ordinary-host-test"] = json!({});
        assert!(
            check_roles(&config, &unreceipted, &inventory)
                .unwrap_err()
                .contains("no authenticated role")
        );
        let release_only = json!({"roles":{"release":inventory["roles"]["release"]}});
        check_roles(
            &json!({"public_verbs":{"bindings":"/public.json"}}),
            &json!({"files":{"release":{}}}),
            &release_only,
        )
        .unwrap();
    }
    #[test]
    fn relocation_preserves_static_file_tree_and_golden_input_resolution() {
        let root =
            std::env::temp_dir().join(format!("gomad-bindings-public-{}", std::process::id()));
        fs::create_dir(&root).unwrap();
        fs::create_dir(root.join("tree")).unwrap();
        fs::write(root.join("tree/golden.bin"), b"genuine static golden bytes").unwrap();
        fs::write(
            root.join("release"),
            b"static input; executable role authenticated separately",
        )
        .unwrap();
        let inventory = format!(
            "{}  golden.bin\n",
            custody::digest(b"genuine static golden bytes")
        );
        fs::write(root.join("tree.sha256"), &inventory).unwrap();
        let mut template = json!({"schema":"gomad.rust-public-bindings/v1","files":{"release":{"path":"release","sha256":custody::digest(&fs::read(root.join("release")).unwrap())}},"trees":{"goldens":{"path":"tree","inventory":"tree.sha256","sha256":custody::digest(inventory.as_bytes())}}});
        let path = root.join("template.json");
        let save = |value: &Value| fs::write(&path, serde_json::to_vec(value).unwrap()).unwrap();
        save(&template);
        let (relocated, files) = static_inputs(&path).unwrap();
        assert_eq!(
            relocated["files"]["release"]["path"],
            json!(root.join("release"))
        );
        assert_eq!(
            relocated["trees"]["goldens"]["inventory"],
            json!(root.join("tree.sha256"))
        );
        assert_eq!(files.len(), 4);
        for (field, value, error) in [
            (
                "schema",
                json!("gomad.all-verb-bindings/v1"),
                "foreign public verb binding",
            ),
            ("extra", json!(true), "unknown binding fields"),
        ] {
            let mut mutant = template.clone();
            mutant[field] = value;
            save(&mutant);
            assert!(static_inputs(&path).unwrap_err().contains(error));
        }
        save(&template);
        fs::write(root.join("tree/golden.bin"), b"changed golden bytes").unwrap();
        assert!(
            static_inputs(&path)
                .unwrap_err()
                .contains("frozen input changed")
        );
        fs::write(root.join("tree/golden.bin"), b"genuine static golden bytes").unwrap();
        fs::write(root.join("tree/omitted.bin"), b"unlisted").unwrap();
        assert!(
            static_inputs(&path)
                .unwrap_err()
                .contains("incomplete public tree inventory")
        );
        fs::remove_file(root.join("tree/omitted.bin")).unwrap();
        let duplicate = format!("{inventory}{inventory}");
        fs::write(root.join("tree.sha256"), &duplicate).unwrap();
        template["trees"]["goldens"]["sha256"] = json!(custody::digest(duplicate.as_bytes()));
        save(&template);
        assert!(
            static_inputs(&path)
                .unwrap_err()
                .contains("duplicate public tree inventory")
        );
        fs::remove_dir_all(&root).unwrap();
    }
}
