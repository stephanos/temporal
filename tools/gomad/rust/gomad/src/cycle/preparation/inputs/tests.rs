use super::*;
use std::{collections::BTreeMap, fs, os::unix::fs::PermissionsExt, path::PathBuf};

fn fixture() -> (PathBuf, PathBuf, PathBuf, BTreeMap<String, String>) {
    let base = std::env::temp_dir().join(format!(
        "gomad-inputs-{}-{}",
        std::process::id(),
        std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_nanos()
    ));
    let root = base.join("producer");
    let repo = base.join("repo");
    fs::create_dir_all(&root).unwrap();
    fs::set_permissions(&root, fs::Permissions::from_mode(0o700)).unwrap();
    fs::create_dir_all(repo.join("tools/gomad/internal/backends/wasm/testdata/temporal_storage"))
        .unwrap();
    let inputs = [
        ("go.mod", b"module example.test/fixture\n".as_slice()),
        ("go.sum", b"sum\n"),
        (
            "tools/gomad/internal/backends/wasm/testdata/temporal_storage/recovery_test.go",
            b"package fixture\n",
        ),
    ];
    let mut pins = BTreeMap::new();
    let mut manifest = String::new();
    for (name, bytes) in inputs {
        let path = repo.join(name);
        fs::write(&path, bytes).unwrap();
        pins.insert(path.to_str().unwrap().into(), digest(bytes));
        manifest.push_str(&format!("{}  {}\n", digest(bytes), path.display()));
    }
    let manifest_path = base.join("sources.manifest");
    fs::write(&manifest_path, &manifest).unwrap();
    pins.insert(
        manifest_path.to_str().unwrap().into(),
        digest(manifest.as_bytes()),
    );
    (root, repo, manifest_path, pins)
}

#[test]
fn source_custody_reopens_original_and_derived_bytes() {
    let (root, repo, manifest, pins) = fixture();
    let record = source::stage(&root, &repo, &pins, "recovery", &manifest).unwrap();
    source::verify(
        &root,
        &repo,
        &pins,
        "recovery",
        &record.source,
        &record.working_dir,
        Some(&manifest),
    )
    .unwrap();
    let target = record.working_dir.join("go.mod");
    fs::write(&target, b"module substituted\n").unwrap();
    assert!(
        source::verify(
            &root,
            &repo,
            &pins,
            "recovery",
            &record.source,
            &record.working_dir,
            Some(&manifest)
        )
        .unwrap_err()
        .contains("derived source changed")
    );
    fs::remove_dir_all(root.parent().unwrap()).unwrap();
}

#[test]
fn source_custody_refuses_ambient_replacement() {
    let (root, repo, manifest, mut pins) = fixture();
    let path = repo.join("go.mod");
    fs::write(
        &path,
        b"module example.test/fixture\nreplace foo => /ambient\n",
    )
    .unwrap();
    let mut text = fs::read_to_string(&manifest).unwrap();
    text = text.replace(
        &pins[path.to_str().unwrap()],
        &digest(&fs::read(&path).unwrap()),
    );
    fs::write(&manifest, &text).unwrap();
    pins.insert(
        path.to_str().unwrap().into(),
        digest(&fs::read(&path).unwrap()),
    );
    pins.insert(manifest.to_str().unwrap().into(), digest(text.as_bytes()));
    assert!(
        source::stage(&root, &repo, &pins, "recovery", &manifest)
            .unwrap_err()
            .contains("ambient module replacement")
    );
    fs::remove_dir_all(root.parent().unwrap()).unwrap();
}

#[test]
fn independent_source_inventory_parses_stream_and_reopens_pinned_bytes() {
    let (root, repo, manifest, pins) = fixture();
    let config = serde_json::json!({"files":pins});
    let inventory = Inventory::new(&config).unwrap();
    let raw = format!(
        "{{\"Dir\":{},\"GoFiles\":[\"go.mod\"]}}\n{{\"Dir\":{},\"GoFiles\":[\"go.sum\"]}}",
        serde_json::to_string(&repo).unwrap(),
        serde_json::to_string(&repo).unwrap()
    );
    let selected = inventory
        .sources(raw.as_bytes(), &repo, &BTreeMap::new())
        .unwrap();
    assert_eq!(selected.len(), 2);
    fs::write(repo.join("go.sum"), b"substituted\n").unwrap();
    assert!(
        inventory
            .file_bytes(&repo.join("go.sum"))
            .unwrap_err()
            .contains("source bytes changed")
    );
    assert!(
        inventory
            .sources(
                b"{\"Dir\":\"/tmp\",\"Dir\":\"/elsewhere\"}",
                &repo,
                &BTreeMap::new()
            )
            .is_err()
    );
    assert!(
        inventory
            .sources(
                format!(
                    "{{\"Dir\":{},\"GoFiles\":[\"../escape\"]}}",
                    serde_json::to_string(&repo).unwrap()
                )
                .as_bytes(),
                &repo,
                &BTreeMap::new()
            )
            .unwrap_err()
            .contains("unsafe native package filename")
    );
    let _ = manifest;
    fs::remove_dir_all(root.parent().unwrap()).unwrap();
}

#[test]
fn source_custody_exact_tree_modes_aliases_and_frozen_originals() {
    for mutation in [
        "extra-file",
        "extra-dir",
        "missing",
        "symlink",
        "hardlink",
        "file-mode",
        "dir-mode",
        "receipt",
        "original",
        "foreign-manifest",
        "target",
    ] {
        let (root, repo, manifest, pins) = fixture();
        let record = source::stage(&root, &repo, &pins, "recovery", &manifest).unwrap();
        source::stage(&root, &repo, &pins, "recovery", &manifest).unwrap();
        let target = record.working_dir.join("go.sum");
        let mut source = record.source.clone();
        let mut frozen = manifest.clone();
        match mutation {
            "extra-file" => fs::write(record.working_dir.join("foreign"), b"x").unwrap(),
            "extra-dir" => fs::create_dir(record.working_dir.join("foreign")).unwrap(),
            "missing" => fs::remove_file(&target).unwrap(),
            "symlink" => {
                fs::remove_file(&target).unwrap();
                std::os::unix::fs::symlink(repo.join("go.sum"), &target).unwrap();
            }
            "hardlink" => fs::hard_link(&target, root.join("alias")).unwrap(),
            "file-mode" => fs::set_permissions(&target, fs::Permissions::from_mode(0o644)).unwrap(),
            "dir-mode" => {
                fs::set_permissions(&record.working_dir, fs::Permissions::from_mode(0o755)).unwrap()
            }
            "receipt" => {
                let receipt = root.join("source-custody.json");
                let mut raw = fs::read(&receipt).unwrap();
                raw.push(b' ');
                fs::write(receipt, raw).unwrap();
            }
            "original" => fs::write(repo.join("go.sum"), b"changed original").unwrap(),
            "foreign-manifest" => {
                frozen = root.join("foreign.manifest");
                fs::copy(&manifest, &frozen).unwrap();
            }
            "target" => source.push_str("-substituted"),
            _ => unreachable!(),
        }
        assert!(
            source::verify(
                &root,
                &repo,
                &pins,
                "recovery",
                &source,
                &record.working_dir,
                Some(&frozen)
            )
            .is_err(),
            "accepted {mutation}"
        );
        fs::remove_dir_all(root.parent().unwrap()).unwrap();
    }
}

pub(crate) fn run_cases(owner: &crate::cycle::process::CycleOwner) -> Result<(), String> {
    actual_preparation(owner)?;
    gnu::run_cases(owner)?;
    Ok(())
}
fn actual_preparation(owner: &crate::cycle::process::CycleOwner) -> Result<(), String> {
    actual_profile(owner, false, false)?;
    actual_profile(owner, true, false)?;
    actual_profile(owner, false, true)
}
fn actual_profile(
    _owner: &crate::cycle::process::CycleOwner,
    cooperative: bool,
    go_test: bool,
) -> Result<(), String> {
    use serde_json::{Value, json};
    use std::time::Duration;
    let compiler = PathBuf::from(
        "/Users/stephan/Workspace/skunkworks/gomad/.build-cache/fn232-stock-go1.27.1/bin/go",
    );
    let compiler = fs::canonicalize(compiler).map_err(|e| e.to_string())?;
    let base = PathBuf::from(format!(
        "/home/agent/wt/evidence/fn242-b01-custody-inputs/prep/actual-{}-{}",
        std::process::id(),
        if cooperative {
            "cooperative"
        } else if go_test {
            "stock-test"
        } else {
            "stock"
        }
    ));
    fs::create_dir_all(&base).map_err(|e| e.to_string())?;
    let source_dir = base.join("source");
    fs::create_dir(&source_dir).map_err(|e| e.to_string())?;
    fs::write(
        source_dir.join("go.mod"),
        b"module preparation.inputs.fixture\n\ngo 1.27.1\n",
    )
    .map_err(|e| e.to_string())?;
    fs::write(
        source_dir.join("main.go"),
        b"package main\nfunc main(){println(\"authenticated-inputs\")}\n",
    )
    .map_err(|e| e.to_string())?;
    if go_test {
        fs::write(source_dir.join("main_test.go"),b"package main\nimport \"testing\"\nfunc TestPrepared(t *testing.T){ t.Log(\"authenticated-test-inputs\") }\n").map_err(|e|e.to_string())?;
    }
    let installation_path = base.join("installation.json");
    let mut args = vec![
        "--compiler".into(),
        compiler.to_string_lossy().into_owned(),
        "--cache".into(),
        base.join("cache").to_string_lossy().into_owned(),
        "--go-version=go1.27.1".into(),
        "--compile-cache".into(),
        base.join("compilation").to_string_lossy().into_owned(),
        "--output".into(),
        installation_path.to_string_lossy().into_owned(),
    ];
    if cooperative {
        let runtime = PathBuf::from(env!("CARGO_MANIFEST_DIR"))
            .join("../../internal/model/runtimesource")
            .canonicalize()
            .map_err(|e| e.to_string())?;
        args.extend([
            "--profile=gomad.wasi-cooperative/v1".into(),
            "--runtime-root".into(),
            runtime.to_string_lossy().into_owned(),
        ]);
    }
    crate::cli::install::run(&args, &mut Vec::new()).map_err(|e| e.message)?;
    let installed = crate::install::manifest::resolve(Some(&installation_path))?;
    let installation = serde_json::to_value(installed.manifest).map_err(|e| e.to_string())?;
    let mut options = crate::cli::explore::parse(
        &[
            if go_test {
                "go-test".into()
            } else {
                "go-run".into()
            },
            ".".into(),
        ],
        &mut false,
    )?;
    if go_test {
        options.target_args = vec!["-test.run=^TestPrepared$".into(), "-test.v".into()];
    }
    options.choices = cooperative;
    options.working_dir = source_dir.clone();
    if cooperative {
        options.working_dir = PathBuf::from(env!("CARGO_MANIFEST_DIR"))
            .join("../..")
            .canonicalize()
            .map_err(|e| e.to_string())?;
        options.source = "./internal/backends/wasm/testdata/runtime_probes".into();
        options.choice_bytes = 1 << 30;
    }
    options.installation = Some(installation_path);
    options.overall_timeout = Duration::from_secs(600);
    let (prepared, _) = crate::campaign::prepare::prepare_with_compiler_controls(
        &options,
        &crate::campaign::prepare::CompilerControls {
            parallelism: Some(1),
            environment: BTreeMap::from([("GOMAXPROCS".into(), "1".into())]),
        },
    )?;
    let provenance: Value = strict_json(&prepared.provenance)?;
    let inputs = &provenance["inputs"];
    let artifact = base.join("observation");
    fs::create_dir(&artifact).map_err(|e| e.to_string())?;
    for (name, raw) in &prepared.auxiliary {
        let path = artifact.join(name);
        fs::create_dir_all(path.parent().ok_or("observation parent")?)
            .map_err(|e| e.to_string())?;
        fs::write(&path, raw).map_err(|e| e.to_string())?;
    }
    let mut files: BTreeMap<String, String> =
        serde_json::from_value(inputs["compiler"].clone()).map_err(|e| e.to_string())?;
    files.extend(
        serde_json::from_value::<BTreeMap<String, String>>(inputs["sources"].clone())
            .map_err(|e| e.to_string())?
            .into_iter()
            .filter(|(path, _)| !path.starts_with("/$GOCACHE/")),
    );
    files.insert(
        compiler.to_string_lossy().into_owned(),
        inputs["compiler_sha256"]
            .as_str()
            .ok_or("compiler digest")?
            .to_owned(),
    );
    let config = json!({"files":files});
    let inventory = Inventory::new(&config)?;
    let control = inventory.project(
        &artifact,
        &installation,
        inputs,
        Some(u64::from(std::process::id())),
    )?;
    assert!(!control.compiler.is_empty() && !control.sources.is_empty());
    let completion = crate::campaign::execution::run(
        &prepared,
        &crate::campaign::ledger::Job {
            ordinal: 0,
            seed: if cooperative { 0 } else { 7 },
        },
        Duration::from_secs(60),
        &gomad_wasmhost::linked::Cancellation::default(),
    )?
    .finalized()
    .map_err(|e| e.message())?;
    require(
        completion.publication.record.outcome.exit_code == Some(0),
        "genuine guest did not exit zero",
    )?;
    let meta = artifact.join("target/preparation-inputs/compile/metadata.json");
    let original = fs::read(&meta).map_err(|e| e.to_string())?;
    let baseline: Value = strict_json(&original)?;
    if go_test {
        for (id, command, predicate) in [
            (
                "orchestration:preparation:299",
                "compile-module",
                "actual compile command shape changed",
            ),
            (
                "orchestration:preparation:280",
                "selected-source-closure",
                "actual preparation command argv changed",
            ),
        ] {
            let mut changed = baseline.clone();
            let row = changed["commands"]
                .as_array_mut()
                .unwrap()
                .iter_mut()
                .find(|row| row["name"] == command)
                .unwrap();
            if command == "compile-module" {
                assert_eq!(row["argv"][1], "test");
                assert_eq!(row["argv"][2], "-c");
                row["argv"][1] = json!("build");
            } else {
                let argv = row["argv"].as_array_mut().unwrap();
                let index = argv
                    .iter()
                    .position(|arg| arg == "-test")
                    .ok_or("go-test original closure flag missing")?;
                argv.remove(index);
            }
            fs::write(&meta, canonical(&changed)?).map_err(|e| e.to_string())?;
            let refused = inventory
                .project(&artifact, &installation, inputs, None)
                .unwrap_err();
            preparation_proof(
                id,
                json!({"kind":"go-test","inputs":inputs,"metadata":baseline}),
                json!({"kind":"go-test","metadata":changed}),
                predicate,
                &refused,
            );
            fs::write(&meta, &original).map_err(|e| e.to_string())?;
        }
        let generated_key = inputs["sources"]
            .as_object()
            .ok_or("source inventory map missing")?
            .keys()
            .find(|key| key.starts_with("/$GOCACHE/"))
            .ok_or("genuine generated test cache identity missing")?;
        for missing in [false, true] {
            let mut changed = inputs.clone();
            if missing {
                changed["sources"]
                    .as_object_mut()
                    .unwrap()
                    .remove(generated_key);
            } else {
                let expected = changed["sources"][generated_key]
                    .as_str()
                    .ok_or("generated test pin missing")?;
                changed["sources"][generated_key] = json!(format!(
                    "{}{}",
                    if expected.starts_with('0') { '1' } else { '0' },
                    &expected[1..]
                ));
            }
            let predicate = if missing {
                "native generated test source pin missing"
            } else {
                "native generated test source bytes changed"
            };
            let mut key_inputs = json!({"schema":"gomad.native-compilation-inputs/v1","profile":installation["profile"]});
            for field in [
                "compiler",
                "compiler_path",
                "compiler_sha256",
                "go_version",
                "sources",
                "runtime",
                "runtime_implementation",
                "kind",
                "source",
                "directory",
                "tags",
                "build_flags",
                "environment",
            ] {
                key_inputs[field] = changed[field].clone();
            }
            let cache_key = digest(&canonical(&key_inputs)?);
            let mut metadata = baseline.clone();
            for row in metadata["commands"].as_array_mut().unwrap() {
                if row["name"] == "compile-module" {
                    row["environment"]["GOCACHE"] = json!(
                        Path::new(installation["cache_root"].as_str().unwrap())
                            .join("compilation-v1")
                            .join(cache_key.as_str())
                            .join("objects")
                    );
                }
            }
            fs::write(&meta, canonical(&metadata)?).map_err(|e| e.to_string())?;
            let refused = inventory
                .project(&artifact, &installation, &changed, None)
                .unwrap_err();
            preparation_proof(
                "orchestration:preparation:67",
                json!({"kind":"go-test","config":config,"inputs":inputs}),
                json!({"case":"generated-go-test-cache-binding","inputs":changed,"metadata":metadata}),
                predicate,
                &refused,
            );
            fs::write(&meta, &original).map_err(|e| e.to_string())?;
        }
        let closure_row = baseline["commands"]
            .as_array()
            .unwrap()
            .iter()
            .find(|row| row["name"] == "selected-source-closure")
            .unwrap();
        let closure_identity = closure_row["stdout"].as_str().unwrap();
        let raw = fs::read(
            artifact.join(
                baseline["blobs"][closure_identity]["path"]
                    .as_str()
                    .unwrap(),
            ),
        )
        .map_err(|e| e.to_string())?;
        let mut documents = serde_json::Deserializer::from_slice(&raw)
            .into_iter::<Value>()
            .collect::<Result<Vec<_>, _>>()
            .map_err(|e| e.to_string())?;
        let generated = documents
            .iter_mut()
            .find(|package| {
                package["ImportPath"]
                    .as_str()
                    .is_some_and(|name| name.ends_with(".test"))
                    && package["Name"] == "main"
            })
            .ok_or("generated test package missing")?;
        generated["GoFiles"][0] = json!("/foreign/cache/generated-d");
        let mut changed_raw = Vec::new();
        for document in documents {
            changed_raw.extend(canonical(&document)?);
            changed_raw.push(b'\n');
        }
        let mut changed = baseline.clone();
        replace_observation_stream(
            &artifact,
            "compile",
            &mut changed,
            "selected-source-closure",
            &changed_raw,
        )?;
        fs::write(&meta, canonical(&changed)?).map_err(|e| e.to_string())?;
        let refused = inventory
            .project(&artifact, &installation, inputs, None)
            .unwrap_err();
        preparation_proof(
            "orchestration:preparation:67",
            json!({"kind":"go-test","metadata":baseline,"inputs":inputs}),
            json!({"case":"generated-go-test-cache-path","metadata":changed,"raw":String::from_utf8_lossy(&changed_raw)}),
            "unsafe native generated test cache member",
            &refused,
        );
        fs::write(&meta, &original).map_err(|e| e.to_string())?;
        inventory.project(&artifact, &installation, inputs, None)?;
        fs::write(base.join("proof.json"),canonical(&json!({"control":"normal-return","kind":"go-test","guest_exit":0,"inputs":inputs,"raw_metadata":baseline,"mutations":5}))?).map_err(|e|e.to_string())?;
        println!(
            "PREPARATION_INPUTS_ACTUAL {} control=normal-return kind=go-test guest_exit=0 mutations=5",
            base.display()
        );
        return Ok(());
    }
    original_observation_boundaries(&inventory, &artifact, &baseline)?;
    original_project_boundaries(&inventory, &artifact, &installation, inputs, cooperative)?;
    let mut observations = Vec::new();
    for (kind, suffix) in [
        ("compiler", "/compile"),
        ("compiler", "/link"),
        ("sources", "/main.go"),
    ] {
        let mut changed = inputs.clone();
        let table = changed[kind].as_object_mut().ok_or("inventory map")?;
        let key = table
            .keys()
            .find(|name| name.ends_with(suffix))
            .ok_or("selected genuine inventory member")?
            .clone();
        table.remove(&key);
        let fields = [
            "compiler",
            "compiler_path",
            "compiler_sha256",
            "go_version",
            "sources",
            "runtime",
            "runtime_implementation",
            "kind",
            "source",
            "directory",
            "tags",
            "build_flags",
            "environment",
        ];
        let mut key_inputs = json!({"schema":"gomad.native-compilation-inputs/v1","profile":installation["profile"]});
        for field in fields {
            key_inputs[field] = changed[field].clone();
        }
        let key = digest(&canonical(&key_inputs)?);
        let mut metadata = baseline.clone();
        for row in metadata["commands"].as_array_mut().unwrap() {
            if row["name"] == "compile-module" {
                row["environment"]["GOCACHE"] = json!(
                    Path::new(installation["cache_root"].as_str().unwrap())
                        .join("compilation-v1")
                        .join(&key)
                        .join("objects")
                );
            }
        }
        let raw = canonical(&metadata)?;
        fs::write(&meta, &raw).map_err(|e| e.to_string())?;
        let refused = inventory
            .project(&artifact, &installation, &changed, None)
            .unwrap_err();
        require(
            refused.contains("membership differs"),
            &format!("resealed omission caught wrong predicate: {refused}"),
        )?;
        preparation_proof(
            "orchestration:preparation:313",
            serde_json::json!({"artifact":artifact,"inputs":inputs}),
            serde_json::json!({"inputs":changed,"metadata":metadata}),
            "complete source/compiler membership differs",
            &refused,
        );
        observations.push(json!({"case":format!("resealed-{kind}-{suffix}"),"input":changed,"raw_metadata":metadata,"input_sha256":digest(&canonical(&changed)?),"refusal":refused,"intended_predicate":"complete source/compiler membership differs"}));
        fs::write(&meta, &original).map_err(|e| e.to_string())?;
    }
    if !cooperative {
        let path = source_dir.join("main.go");
        let raw = fs::read(&path).map_err(|e| e.to_string())?;
        fs::write(&path, b"package main\nfunc main(){}\n").map_err(|e| e.to_string())?;
        let refused = inventory
            .project(&artifact, &installation, inputs, None)
            .unwrap_err();
        require(
            refused.contains("source bytes changed"),
            "actual source substitution did not reach source byte predicate",
        )?;
        fs::write(&path, raw).map_err(|e| e.to_string())?;
    }
    inventory.project(&artifact, &installation, inputs, None)?;
    fs::write(base.join("proof.json"),canonical(&json!({"schema":"gomad.preparation-inputs-proof/v1","control":{"result":"normal-return","compiler":control.compiler,"sources":control.sources,"provenance_sha256":digest(&prepared.provenance),"module_sha256":digest(&prepared.module),"guest_exit":0},"observations":observations}))?).map_err(|e|e.to_string())?;
    println!(
        "PREPARATION_INPUTS_ACTUAL {} control=normal-return guest_exit=0 mutations={} ",
        base.display(),
        observations.len()
    );
    Ok(())
}

fn preparation_proof(
    id: &str,
    control: serde_json::Value,
    mutation: serde_json::Value,
    predicate: &str,
    refusal: &str,
) {
    assert!(
        refusal.contains(predicate),
        "{id}: expected {predicate}, got {refusal}"
    );
    println!(
        "PREPARATION_PROOF {}",
        serde_json::json!({"original_id":id,"original_source_sha256":"1bf8d8dbfddf8de652aad3cfa26b762ed4736ed6aaa0b98de9a7d356812553d7","current_source_sha256":digest(include_bytes!("inventory.rs")),"current_tests_sha256":digest(include_bytes!("tests.rs")),"control_return":"Ok","control_input_sha256":digest(&canonical(&control).unwrap()),"control_input":control,"mutation_input_sha256":digest(&canonical(&mutation).unwrap()),"mutation_input":mutation,"intended_predicate":predicate,"function_return":"Err","refusal":refusal})
    );
}
#[test]
fn original_inventory_source_and_scalar_predicates() {
    use serde_json::json;
    let (root, repo, manifest, pins) = fixture();
    let mut pins = pins;
    let local = repo.join("input.s");
    let header = repo.join("input.h");
    fs::write(&local, b"#include \"input.h\"\n").unwrap();
    fs::write(&header, b"#define INPUT 1\n").unwrap();
    for path in [&local, &header] {
        pins.insert(
            path.to_str().unwrap().into(),
            digest(&fs::read(path).unwrap()),
        );
    }
    let control = json!({"Dir":repo,"GoFiles":["go.mod"]});
    let raw = canonical(&control).unwrap();
    let inventory = Inventory::new(&json!({"files":pins})).unwrap();
    inventory.sources(&raw, &repo, &BTreeMap::new()).unwrap();
    for (id, mutant, predicate) in [
        (
            "orchestration:preparation:43",
            json!([]),
            "requires objects",
        ),
        (
            "orchestration:preparation:45",
            json!(null),
            "empty native package stream",
        ),
        (
            "orchestration:preparation:136",
            json!({"Dir":"relative","GoFiles":["go.mod"]}),
            "directory is not absolute",
        ),
        (
            "orchestration:preparation:139",
            json!({"Dir":repo,"Module":{"Replace":{}} ,"GoFiles":["go.mod"]}),
            "ambient module replacement",
        ),
        (
            "orchestration:preparation:145",
            json!({"Dir":repo,"GoFiles":[1]}),
            "invalid type",
        ),
        (
            "orchestration:preparation:147",
            json!({"Dir":repo,"GoFiles":["../foreign"]}),
            "unsafe native package filename",
        ),
        (
            "orchestration:preparation:157",
            json!({"Dir":repo}),
            "empty native complete selected inventory",
        ),
    ] {
        let bytes = if mutant.is_null() {
            vec![]
        } else {
            canonical(&mutant).unwrap()
        };
        let error = inventory
            .sources(&bytes, &repo, &BTreeMap::new())
            .unwrap_err();
        preparation_proof(id, control.clone(), json!({"raw":bytes}), predicate, &error);
    }
    let duplicate = format!(
        "{{\"Dir\":{},\"Dir\":{}}}",
        serde_json::to_string(&repo).unwrap(),
        serde_json::to_string(&repo).unwrap()
    );
    let error = inventory
        .sources(duplicate.as_bytes(), &repo, &BTreeMap::new())
        .unwrap_err();
    preparation_proof(
        "orchestration:preparation:22",
        control.clone(),
        json!({"raw":duplicate}),
        "duplicate",
        &error,
    );
    for name in ["relative", "/unfrozen", "/frozen/../unfrozen"] {
        let error = inventory.pin(Path::new(name)).unwrap_err();
        inventory.pin(&repo.join("go.mod")).unwrap();
        preparation_proof(
            "orchestration:preparation:56",
            json!({"path":repo.join("go.mod")}),
            json!({"path":name}),
            "not frozen",
            &error,
        );
    }
    let mut replacements = BTreeMap::new();
    replacements.insert(
        repo.join("go.mod").to_string_lossy().into_owned(),
        String::new(),
    );
    let error = inventory.sources(&raw, &repo, &replacements).unwrap_err();
    preparation_proof(
        "orchestration:preparation:151",
        control.clone(),
        json!({"package":control,"replacements":replacements}),
        "overlay removed selected input",
        &error,
    );
    let assembly = json!({"Dir":repo,"SFiles":["input.s"]});
    inventory
        .sources(&canonical(&assembly).unwrap(), &repo, &BTreeMap::new())
        .unwrap();
    for (id, body, predicate) in [
        (
            "orchestration:preparation:188",
            b"#include <unsupported>\n".as_slice(),
            "JSON",
        ),
        (
            "orchestration:preparation:192",
            b"#include \"missing.h\"\n".as_slice(),
            "assembler include missing",
        ),
        (
            "orchestration:preparation:196",
            b"/* unclosed\n".as_slice(),
            "unterminated native assembler comment",
        ),
    ] {
        fs::write(&local, body).unwrap();
        let mut nextpins = pins.clone();
        nextpins.insert(local.to_str().unwrap().into(), digest(body));
        let next = Inventory::new(&json!({"files":nextpins})).unwrap();
        let error = next
            .sources(&canonical(&assembly).unwrap(), &repo, &BTreeMap::new())
            .unwrap_err();
        preparation_proof(
            id,
            json!({"raw":b"#include \"input.h\"\n".as_slice()}),
            json!({"raw":body}),
            predicate,
            &error,
        );
    }
    let path = repo.join("go.sum");
    inventory.file_bytes(&path).unwrap();
    let body = fs::read(&path).unwrap();
    fs::remove_file(&path).unwrap();
    std::os::unix::fs::symlink(repo.join("go.mod"), &path).unwrap();
    let error = inventory.file_bytes(&path).unwrap_err();
    preparation_proof(
        "orchestration:preparation:63",
        json!({"path":path,"bytes":body}),
        json!({"path":path,"symlink":repo.join("go.mod")}),
        "custody file",
        &error,
    );
    fs::remove_file(&path).unwrap();
    fs::write(&path, &body).unwrap();
    fs::OpenOptions::new()
        .write(true)
        .open(&path)
        .unwrap()
        .set_len((512 << 20) + 1)
        .unwrap();
    let error = inventory.file_bytes(&path).unwrap_err();
    preparation_proof(
        "orchestration:preparation:65",
        json!({"bytes":body}),
        json!({"sparse_bytes":(512_u64<<20)+1}),
        "byte bound",
        &error,
    );
    fs::write(&path, b"substituted").unwrap();
    let error = inventory.file_bytes(&path).unwrap_err();
    preparation_proof(
        "orchestration:preparation:67",
        json!({"bytes":body}),
        json!({"bytes":b"substituted".as_slice()}),
        "source bytes changed",
        &error,
    );
    assert_eq!(strict_json(b"{\"finite\":1}").unwrap()["finite"], 1);
    for raw in [b"NaN".as_slice(), b"Infinity", b"-Infinity", b"{\"n\":NaN}"] {
        let refused = strict_json(raw).unwrap_err();
        println!(
            "PREPARATION_PROOF {}",
            json!({"original_id":"workload-callback:native_observations:callback_throw:44:51","original_source_sha256":"495b07b0954b409fbb5f2d54bfdb6d3ae39674ed818947c3fbd754bb8e8ac95b","current_source_sha256":digest(include_bytes!("../../custody/mod.rs")),"control_return":"Ok","control_input":b"{\"finite\":1}".as_slice(),"mutation_input":raw,"mutation_input_sha256":digest(raw),"intended_predicate":"strict JSON refuses nonfinite constants","function_return":"Err","refusal":refused})
        );
    }
    let _ = manifest;
    fs::remove_dir_all(root.parent().unwrap()).unwrap();
}

#[test]
fn original_compiler_environment_and_complete_header_inventory_predicates() {
    use serde_json::json;
    let compiler = fs::canonicalize(
        "/Users/stephan/Workspace/skunkworks/gomad/.build-cache/fn232-stock-go1.27.1/bin/go",
    )
    .unwrap();
    let sdk = compiler.parent().unwrap().parent().unwrap();
    let tools = fs::read_dir(sdk.join("pkg/tool"))
        .unwrap()
        .map(|entry| entry.unwrap().path())
        .find(|path| path.is_dir())
        .unwrap();
    let mut paths = std::collections::BTreeSet::from([
        sdk.join("VERSION"),
        tools.join("asm"),
        tools.join("compile"),
        tools.join("link"),
    ]);
    for entry in fs::read_dir(sdk.join("pkg/include")).unwrap() {
        let path = entry.unwrap().path();
        if path.extension().is_some_and(|ext| ext == "h") {
            paths.insert(path);
        }
    }
    let pins: BTreeMap<String, String> = paths
        .iter()
        .map(|path| {
            (
                path.to_string_lossy().into_owned(),
                digest(&read(path, 512 << 20).unwrap()),
            )
        })
        .collect();
    let inventory = Inventory::new(&json!({"files":pins})).unwrap();
    let environment = json!({"GOROOT":sdk,"GOTOOLDIR":tools});
    let snapshot = serde_json::to_value(&paths).unwrap();
    let raw = canonical(&environment).unwrap();
    let frozen = canonical(&snapshot).unwrap();
    inventory.compiler(&raw, &frozen).unwrap();
    for (id, changed, predicate) in [
        (
            "orchestration:preparation:118",
            json!({"GOROOT":sdk,"GOTOOLDIR":tools,"FOREIGN":1}),
            "env output shape changed",
        ),
        (
            "orchestration:preparation:120",
            json!({"GOROOT":"relative","GOTOOLDIR":tools}),
            "distribution is not absolute",
        ),
    ] {
        let error = inventory
            .compiler(&canonical(&changed).unwrap(), &frozen)
            .unwrap_err();
        preparation_proof(
            id,
            json!({"environment":environment,"snapshot":snapshot}),
            changed,
            predicate,
            &error,
        );
    }
    let error = inventory.compiler(&raw, b"[]").unwrap_err();
    preparation_proof(
        "orchestration:preparation:125",
        json!({"environment":environment,"snapshot":snapshot}),
        json!({"environment":environment,"snapshot":[]}),
        "complete compiler directory inventory changed",
        &error,
    );
    let (root, repo, _, _) = fixture();
    fs::create_dir_all(repo.join("pkg/include")).unwrap();
    let changed = json!({"GOROOT":repo,"GOTOOLDIR":tools});
    let error = inventory
        .compiler(&canonical(&changed).unwrap(), &frozen)
        .unwrap_err();
    preparation_proof(
        "orchestration:preparation:123",
        environment,
        changed,
        "assembler headers missing",
        &error,
    );
    fs::remove_dir_all(root.parent().unwrap()).unwrap();
}

fn prune_observation_blobs(value: &mut serde_json::Value) {
    let used: std::collections::BTreeSet<String> = value["commands"]
        .as_array()
        .unwrap()
        .iter()
        .flat_map(|row| ["stdout", "stderr"].map(|kind| row[kind].as_str().unwrap().to_owned()))
        .chain(
            value["snapshots"]
                .as_object()
                .unwrap()
                .values()
                .map(|v| v.as_str().unwrap().to_owned()),
        )
        .collect();
    value["blobs"]
        .as_object_mut()
        .unwrap()
        .retain(|identity, _| used.contains(identity));
}
fn replace_observation_stream(
    artifact: &Path,
    phase: &str,
    value: &mut serde_json::Value,
    name: &str,
    raw: &[u8],
) -> Result<(), String> {
    let identity = digest(raw);
    let path = format!("target/preparation-inputs/{phase}/{identity}.bin");
    fs::write(artifact.join(&path), raw).map_err(|e| e.to_string())?;
    value["blobs"][&identity] =
        serde_json::json!({"path":path,"bytes":raw.len(),"sha256":identity});
    let row = value["commands"]
        .as_array_mut()
        .unwrap()
        .iter_mut()
        .find(|row| row["name"] == name)
        .ok_or("observed command missing")?;
    row["stdout"] = serde_json::json!(identity);
    prune_observation_blobs(value);
    Ok(())
}
fn original_observation_boundaries(
    inventory: &Inventory,
    artifact: &Path,
    baseline: &serde_json::Value,
) -> Result<(), String> {
    use serde_json::{Value, json};
    let meta = artifact.join("target/preparation-inputs/compile/metadata.json");
    let original = fs::read(&meta).map_err(|e| e.to_string())?;
    inventory.observed(artifact, "compile")?;
    let mut cases: Vec<(&str, &str, Value, &str)> = Vec::new();
    let mut changed = baseline.clone();
    changed["schema"] = json!("foreign");
    cases.push((
        "orchestration:preparation:77",
        "schema",
        changed,
        "wrong/noncanonical",
    ));
    let mut changed = baseline.clone();
    changed["phase"] = json!("admission");
    cases.push((
        "orchestration:preparation:77",
        "phase",
        changed,
        "wrong/noncanonical",
    ));
    let mut changed = baseline.clone();
    for index in 0..33 {
        let id = format!("{index:064x}");
        changed["blobs"][&id] = json!({"path":"unused","bytes":0,"sha256":id});
    }
    cases.push((
        "orchestration:preparation:80",
        "blob-count",
        changed,
        "blob table exceeds bound",
    ));
    let identity = baseline["blobs"]
        .as_object()
        .unwrap()
        .keys()
        .next()
        .unwrap()
        .clone();
    for (field, value) in [
        ("path", json!("../foreign")),
        ("bytes", json!((128u64 << 20) + 1)),
        ("sha256", json!("foreign")),
    ] {
        let mut changed = baseline.clone();
        changed["blobs"][&identity][field] = value;
        cases.push((
            "orchestration:preparation:83",
            field,
            changed,
            "invalid native preparation blob metadata",
        ));
    }
    let mut changed = baseline.clone();
    changed["commands"][0]["argv"] = json!(["/unfrozen"]);
    cases.push((
        "orchestration:preparation:101",
        "compiler",
        changed,
        "actual compiler/environment is not frozen",
    ));
    let mut changed = baseline.clone();
    changed["commands"][0]["stdout"] = json!("f".repeat(64));
    cases.push((
        "orchestration:preparation:105",
        "stream",
        changed,
        "command stream absent",
    ));
    let mut changed = baseline.clone();
    let first = changed["commands"][0]["owner_pid"].as_u64().unwrap();
    changed["commands"][0]["owner_pid"] = json!(first + 1);
    cases.push((
        "orchestration:preparation:109",
        "borrowed-owner",
        changed,
        "borrowed another owner",
    ));
    let mut changed = baseline.clone();
    changed["snapshots"]["compiler-base-paths"] = json!("f".repeat(64));
    cases.push((
        "orchestration:preparation:111",
        "snapshot",
        changed,
        "snapshot absent",
    ));
    let extra = b"unreferenced genuine retained stream";
    let id = digest(extra);
    let path = format!("target/preparation-inputs/compile/{id}.bin");
    fs::write(artifact.join(&path), extra).map_err(|e| e.to_string())?;
    let mut changed = baseline.clone();
    changed["blobs"][&id] = json!({"path":path,"bytes":extra.len(),"sha256":id});
    cases.push((
        "orchestration:preparation:113",
        "unreferenced",
        changed,
        "unreferenced/missing raw blobs",
    ));
    for (field, value) in [
        ("pid", json!(0)),
        ("owner_pid", json!(0)),
        ("exit", json!(1)),
        ("start_ns", json!(0)),
        ("end_ns", json!(u64::MAX)),
    ] {
        let mut changed = baseline.clone();
        changed["commands"][0][field] = value;
        cases.push((
            "orchestration:preparation:94",
            field,
            changed,
            "command missing/duplicate/failed",
        ));
    }
    for (id, name, value, predicate) in cases {
        fs::write(&meta, canonical(&value)?).map_err(|e| e.to_string())?;
        let refused = inventory
            .observed(artifact, "compile")
            .err()
            .ok_or("observation mutant accepted")?;
        preparation_proof(
            id,
            json!({"raw_metadata":baseline,"artifact":artifact}),
            json!({"case":name,"raw_metadata":value,"artifact":artifact}),
            predicate,
            &refused,
        );
        fs::write(&meta, &original).map_err(|e| e.to_string())?;
    }
    let displaced = meta.with_extension("saved");
    fs::rename(&meta, &displaced).map_err(|e| e.to_string())?;
    let refused = inventory
        .observed(artifact, "compile")
        .err()
        .ok_or("missing observation accepted")?;
    preparation_proof(
        "orchestration:preparation:73",
        json!({"path":meta,"raw":original}),
        json!({"path":meta,"absent":true}),
        "unexecuted:",
        &refused,
    );
    std::os::unix::fs::symlink(&displaced, &meta).map_err(|e| e.to_string())?;
    let refused = inventory
        .observed(artifact, "compile")
        .err()
        .ok_or("aliased observation accepted")?;
    preparation_proof(
        "orchestration:preparation:74",
        json!({"path":meta,"raw":original}),
        json!({"path":meta,"symlink":displaced}),
        "nonregular or linked source",
        &refused,
    );
    fs::remove_file(&meta).map_err(|e| e.to_string())?;
    fs::rename(&displaced, &meta).map_err(|e| e.to_string())?;
    fs::OpenOptions::new()
        .write(true)
        .open(&meta)
        .map_err(|e| e.to_string())?
        .set_len((16 << 20) + 1)
        .map_err(|e| e.to_string())?;
    let refused = inventory
        .observed(artifact, "compile")
        .err()
        .ok_or("oversized metadata accepted")?;
    preparation_proof(
        "orchestration:preparation:74",
        json!({"path":meta,"raw":original}),
        json!({"sparse_bytes":(16u64<<20)+1}),
        "input byte bound",
        &refused,
    );
    fs::write(&meta, &original).map_err(|e| e.to_string())?;
    let (identity, entry) = baseline["blobs"]
        .as_object()
        .unwrap()
        .iter()
        .find(|(_, entry)| entry["bytes"].as_u64().unwrap() > 0)
        .unwrap();
    let path = artifact.join(entry["path"].as_str().unwrap());
    let raw = fs::read(&path).map_err(|e| e.to_string())?;
    let moved = path.with_extension("saved");
    fs::rename(&path, &moved).map_err(|e| e.to_string())?;
    let refused = inventory
        .observed(artifact, "compile")
        .err()
        .ok_or("missing blob accepted")?;
    preparation_proof(
        "orchestration:preparation:87",
        json!({"blob":entry,"raw":raw}),
        json!({"blob":entry,"absent":true}),
        "No such file",
        &refused,
    );
    fs::rename(&moved, &path).map_err(|e| e.to_string())?;
    let mut corrupt = raw.clone();
    corrupt[0] ^= 1;
    fs::write(&path, &corrupt).map_err(|e| e.to_string())?;
    let refused = inventory
        .observed(artifact, "compile")
        .err()
        .ok_or("substituted raw blob accepted")?;
    preparation_proof(
        "orchestration:preparation:89",
        json!({"identity":identity,"raw":raw}),
        json!({"identity":identity,"raw":corrupt}),
        "raw command stream changed",
        &refused,
    );
    fs::write(&path, &raw).map_err(|e| e.to_string())?;
    inventory.observed(artifact, "compile")?;
    Ok(())
}

fn replace_observation_snapshot(
    artifact: &Path,
    phase: &str,
    value: &mut serde_json::Value,
    name: &str,
    raw: &[u8],
) -> Result<(), String> {
    let identity = digest(raw);
    let path = format!("target/preparation-inputs/{phase}/{identity}.bin");
    fs::write(artifact.join(&path), raw).map_err(|e| e.to_string())?;
    value["blobs"][&identity] =
        serde_json::json!({"path":path,"bytes":raw.len(),"sha256":identity});
    value["snapshots"][name] = serde_json::json!(identity);
    prune_observation_blobs(value);
    Ok(())
}
fn original_project_boundaries(
    inventory: &Inventory,
    artifact: &Path,
    installation: &serde_json::Value,
    inputs: &serde_json::Value,
    cooperative: bool,
) -> Result<(), String> {
    use serde_json::{Value, json};
    let meta = artifact.join("target/preparation-inputs/compile/metadata.json");
    let raw = fs::read(&meta).map_err(|e| e.to_string())?;
    let baseline = strict_json(&raw)?;
    let control = json!({"artifact":artifact,"installation":installation,"inputs":inputs,"raw_metadata_sha256":digest(&raw)});
    inventory.project(artifact, installation, inputs, None)?;
    let compiler_row = baseline["commands"]
        .as_array()
        .unwrap()
        .iter()
        .find(|row| row["name"] == "compiler-environment")
        .unwrap();
    let compiler_raw = fs::read(
        artifact.join(
            baseline["blobs"][compiler_row["stdout"].as_str().unwrap()]["path"]
                .as_str()
                .unwrap(),
        ),
    )
    .map_err(|e| e.to_string())?;
    let snapshot_id = baseline["snapshots"]["compiler-base-paths"]
        .as_str()
        .unwrap();
    let snapshot =
        fs::read(artifact.join(baseline["blobs"][snapshot_id]["path"].as_str().unwrap()))
            .map_err(|e| e.to_string())?;
    let (before, root) = inventory.compiler(&compiler_raw, &snapshot)?;
    super::inventory::same_compiler_inventory(&before, &root, &before, &root)?;
    let mut after = before.clone();
    let selected = after.keys().next().unwrap().clone();
    after.remove(&selected);
    let refused =
        super::inventory::same_compiler_inventory(&before, &root, &after, &root).unwrap_err();
    preparation_proof(
        "orchestration:preparation:213",
        json!({"compiler":before,"root":root}),
        json!({"compiler":after,"root":root}),
        "compiler pre/post complete inventory changed",
        &refused,
    );
    let refused =
        super::inventory::same_compiler_inventory(&before, &root, &before, Path::new("/foreign"))
            .unwrap_err();
    preparation_proof(
        "orchestration:preparation:213",
        json!({"compiler":before,"root":root}),
        json!({"compiler":before,"root":"/foreign"}),
        "compiler pre/post complete inventory changed",
        &refused,
    );
    let refused = inventory
        .project(artifact, installation, inputs, Some(u64::MAX))
        .unwrap_err();
    preparation_proof(
        "orchestration:preparation:202",
        control.clone(),
        json!({"owner_pid":u64::MAX}),
        "borrowed another actual producer",
        &refused,
    );
    let mut input_cases: Vec<(&str, &str, Value, &str)> = Vec::new();
    for (id, name, field, value, predicate) in [
        (
            "orchestration:preparation:215",
            "compiler-digest",
            "compiler_sha256",
            json!("f".repeat(64)),
            "actual compiler differs",
        ),
        (
            "orchestration:preparation:217",
            "target-kind",
            "kind",
            json!("exec"),
            "compiler version or target kind changed",
        ),
        (
            "orchestration:preparation:233",
            "build-flags",
            "build_flags",
            json!(["-mod=readonly"]),
            "compiler flags changed",
        ),
        (
            "orchestration:preparation:261",
            "tags",
            "tags",
            json!(["test_dep", "test_dep"]),
            "build tags not canonical",
        ),
    ] {
        let mut changed = inputs.clone();
        changed[field] = value;
        input_cases.push((id, name, changed, predicate));
    }
    for (key, value) in [
        ("GOMAXPROCS", "2"),
        ("GOTMPDIR", "relative"),
        ("GOWASM", "future"),
        ("UNKNOWN", "foreign"),
    ] {
        let mut changed = inputs.clone();
        changed["environment"][key] = json!(value);
        input_cases.push((
            "orchestration:preparation:223",
            key,
            changed,
            "scoped compiler controls changed",
        ));
    }
    let mut changed = inputs.clone();
    changed["environment"]["GOENV"] = json!("on");
    input_cases.push((
        "orchestration:preparation:230",
        "forced-controls",
        changed,
        "original forced compiler controls changed",
    ));
    for (id, name, changed, predicate) in input_cases {
        let refused = inventory
            .project(artifact, installation, &changed, None)
            .unwrap_err();
        preparation_proof(
            id,
            control.clone(),
            json!({"case":name,"inputs":changed}),
            predicate,
            &refused,
        );
    }
    let mut metadata_cases: Vec<(&str, &str, Value, &str)> = Vec::new();
    let mut changed = baseline.clone();
    changed["commands"]
        .as_array_mut()
        .unwrap()
        .retain(|row| row["name"] != "selected-package");
    prune_observation_blobs(&mut changed);
    metadata_cases.push((
        "orchestration:preparation:206",
        "exact-coverage",
        changed,
        "exact phase/command/snapshot coverage changed",
    ));
    for (id, name, field, value, predicate) in [
        (
            "orchestration:preparation:276",
            "output-cap",
            "output_limit",
            json!(1),
            "original command output cap changed",
        ),
        (
            "orchestration:preparation:279",
            "cwd",
            "cwd",
            json!("/foreign"),
            "command source directory changed",
        ),
        (
            "orchestration:preparation:280",
            "argv",
            "argv",
            json!([inputs["compiler_path"], "foreign"]),
            "actual preparation command argv changed",
        ),
    ] {
        let mut changed = baseline.clone();
        let row = changed["commands"]
            .as_array_mut()
            .unwrap()
            .iter_mut()
            .find(|row| row["name"] == "selected-package")
            .unwrap();
        row[field] = value;
        metadata_cases.push((id, name, changed, predicate));
    }
    for (id, name, key, value, predicate) in [
        (
            "orchestration:preparation:283",
            "ambient-environment",
            "SECRET_TOKEN",
            json!("foreign"),
            "inherited unrelated host environment",
        ),
        (
            "orchestration:preparation:284",
            "inherited-cache",
            "GOCACHE",
            json!("/foreign"),
            "used an unbound inherited object cache",
        ),
        (
            "orchestration:preparation:286",
            "foreign-root",
            "GOROOT",
            json!("/foreign"),
            "inherited a foreign GOROOT",
        ),
        (
            "orchestration:preparation:292",
            "compiler-control",
            "GOENV",
            json!("on"),
            "preparation compiler controls changed",
        ),
    ] {
        let mut changed = baseline.clone();
        let row = changed["commands"]
            .as_array_mut()
            .unwrap()
            .iter_mut()
            .find(|row| row["name"] == "compiler-version")
            .unwrap();
        row["environment"][key] = value;
        metadata_cases.push((id, name, changed, predicate));
    }
    for (id, name, action, predicate) in [
        (
            "orchestration:preparation:299",
            "compile-shape",
            0,
            "actual compile command shape changed",
        ),
        (
            "orchestration:preparation:303",
            "compile-output",
            1,
            "outside actual owner private preparation",
        ),
        (
            "orchestration:preparation:309",
            "object-cache",
            2,
            "own original per-entry object cache",
        ),
    ] {
        let mut changed = baseline.clone();
        let row = changed["commands"]
            .as_array_mut()
            .unwrap()
            .iter_mut()
            .find(|row| row["name"] == "compile-module")
            .unwrap();
        match action {
            0 => row["argv"].as_array_mut().unwrap().remove(1),
            1 => {
                let n = row["argv"].as_array().unwrap().len();
                row["argv"][n - 2] = json!("/foreign/module.wasm");
                Value::Null
            }
            _ => {
                row["environment"]["GOCACHE"] = json!("/foreign");
                Value::Null
            }
        };
        metadata_cases.push((id, name, changed, predicate));
    }
    if cooperative {
        let mut changed = baseline.clone();
        let row = changed["commands"]
            .as_array_mut()
            .unwrap()
            .iter_mut()
            .find(|row| row["name"] == "runtime-overlay")
            .unwrap();
        row["environment"]["GOOS"] = json!("wasip1");
        metadata_cases.push((
            "orchestration:preparation:288",
            "overlay-host-controls",
            changed,
            "overlay host compiler controls changed",
        ));
        let mut changed = baseline.clone();
        let row = changed["commands"]
            .as_array_mut()
            .unwrap()
            .iter_mut()
            .find(|row| row["name"] == "selected-package")
            .unwrap();
        row["environment"]["GOROOT"] = json!("/foreign");
        metadata_cases.push((
            "orchestration:preparation:294",
            "cooperative-root",
            changed,
            "preparation GOROOT changed",
        ));
        let mut changed = baseline.clone();
        replace_observation_stream(
            artifact,
            "compile",
            &mut changed,
            "runtime-overlay",
            br#"{"path":"/foreign","identity":"foreign","implementation_sha256":"foreign"}"#,
        )?;
        metadata_cases.push((
            "orchestration:preparation:241",
            "overlay-identity",
            changed,
            "runtime overlay output identity changed",
        ));
        let mut changed = baseline.clone();
        replace_observation_stream(
            artifact,
            "compile",
            &mut changed,
            "compiler-module-cache",
            b"/\n",
        )?;
        metadata_cases.push((
            "orchestration:preparation:259",
            "module-cache",
            changed,
            "cooperative compiler is inside module cache",
        ));
    }
    for (id, name, changed, predicate) in metadata_cases {
        fs::write(&meta, canonical(&changed)?).map_err(|e| e.to_string())?;
        let refused = inventory
            .project(artifact, installation, inputs, None)
            .unwrap_err();
        preparation_proof(
            id,
            control.clone(),
            json!({"case":name,"metadata":changed}),
            predicate,
            &refused,
        );
        fs::write(&meta, &raw).map_err(|e| e.to_string())?;
    }
    if cooperative {
        let runtime = baseline["commands"]
            .as_array()
            .unwrap()
            .iter()
            .find(|row| row["name"] == "runtime-overlay")
            .unwrap();
        let generated_raw = fs::read(
            artifact.join(
                baseline["blobs"][runtime["stdout"].as_str().unwrap()]["path"]
                    .as_str()
                    .unwrap(),
            ),
        )
        .map_err(|e| e.to_string())?;
        let generated = strict_json(&generated_raw)?;
        let overlay_path = generated["path"].as_str().unwrap();
        let mut changed_installation = installation.clone();
        changed_installation["admission"]["compiler_inputs"][overlay_path] = json!("f".repeat(64));
        let refused = inventory
            .project(artifact, &changed_installation, inputs, None)
            .unwrap_err();
        preparation_proof(
            "orchestration:preparation:244",
            control.clone(),
            json!({"installation":changed_installation}),
            "raw overlay is not frozen/admitted",
            &refused,
        );
        let identity = baseline["snapshots"]["overlay-replacements"]
            .as_str()
            .unwrap();
        let overlay_raw =
            fs::read(artifact.join(baseline["blobs"][identity]["path"].as_str().unwrap()))
                .map_err(|e| e.to_string())?;
        let overlay = strict_json(&overlay_raw)?;
        let mut removed = overlay.clone();
        let first = removed["Replace"]
            .as_object()
            .unwrap()
            .keys()
            .next()
            .unwrap()
            .clone();
        removed["Replace"][&first] = json!("");
        for (id, name, changed, predicate) in [
            (
                "orchestration:preparation:247",
                "overlay-shape",
                json!({"foreign":{}}),
                "runtime overlay shape changed",
            ),
            (
                "orchestration:preparation:250",
                "removed-overlay-source",
                removed,
                "runtime overlay removed source",
            ),
        ] {
            let body = canonical(&changed)?;
            let mut metadata = baseline.clone();
            replace_observation_snapshot(
                artifact,
                "compile",
                &mut metadata,
                "overlay-replacements",
                &body,
            )?;
            let mut installed = installation.clone();
            installed["admission"]["compiler_inputs"][overlay_path] = json!(digest(&body));
            fs::write(&meta, canonical(&metadata)?).map_err(|e| e.to_string())?;
            let refused = inventory
                .project(artifact, &installed, inputs, None)
                .unwrap_err();
            preparation_proof(
                id,
                control.clone(),
                json!({"case":name,"overlay_snapshot":changed,"installation":installed,"metadata":metadata}),
                predicate,
                &refused,
            );
            fs::write(&meta, &raw).map_err(|e| e.to_string())?;
        }
    }
    inventory.project(artifact, installation, inputs, None)?;
    Ok(())
}
