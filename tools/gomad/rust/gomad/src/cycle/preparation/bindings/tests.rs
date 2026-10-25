use super::*;
use std::{fs, os::unix::fs::symlink, path::PathBuf};
fn root() -> PathBuf {
    let p = std::env::temp_dir().canonicalize().unwrap().join(format!(
        "gomad-binding-{}-{}",
        std::process::id(),
        std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_nanos()
    ));
    fs::create_dir(&p).unwrap();
    p
}
pub(super) fn report(ids: &[&str], case: &str, input: &Value, result: &str) {
    eprintln!(
        "B03_PROOF {}",
        json!({"ids":ids,"case":case,"input":input,"input_sha256":custody::digest(&serde_json::to_vec(input).unwrap()),"result":result,"source_sha256":custody::digest(include_bytes!("mod.rs")),"test_sha256":custody::digest(include_bytes!("tests.rs"))})
    );
}
#[test]
fn fresh_publication_reopens_exact_bytes_and_refuses_aliases() {
    let root = root();
    let p = root.join("config");
    publication::fresh(&p).unwrap();
    publication::write(&p, b"actual current bytes").unwrap();
    assert_eq!(custody::read_file(&p, 32).unwrap(), b"actual current bytes");
    let error = publication::fresh(&p).unwrap_err();
    assert!(error.contains("fresh"));
    report(
        &[
            "orchestration:configure:161",
            "orchestration:configure:182",
            "orchestration:configure:259",
        ],
        "fresh output",
        &json!({"path":p,"bytes":"actual current bytes"}),
        &error,
    );
    assert!(publication::write(&p, b"replacement").is_err());
    let alias = root.join("alias");
    symlink(&root, &alias).unwrap();
    assert!(publication::write(&alias.join("other"), b"bytes").is_err());
    fs::remove_dir_all(root).unwrap();
}
#[test]
fn prepared_inventory_reopens_physical_compiler_and_source() {
    let root = root();
    let compiler = root.join("compiler");
    let source = root.join("source.go");
    fs::write(&compiler, b"genuine fixture tool bytes").unwrap();
    fs::write(&source, b"package main\n").unwrap();
    let prepared = PreparedInputs {
        compiler: BTreeMap::from([(
            compiler.to_str().unwrap().into(),
            custody::digest(&fs::read(&compiler).unwrap()),
        )]),
        sources: BTreeMap::from([(
            source.to_str().unwrap().into(),
            custody::digest(&fs::read(&source).unwrap()),
        )]),
    };
    let mut files = prepared.compiler.clone();
    merge(&mut files, &prepared.sources).unwrap();
    let config = json!({"files":files});
    check_prepared(&prepared, &config).unwrap();
    for (p, bytes) in [
        (&compiler, b"substituted compiler".as_slice()),
        (&source, b"package replaced\n".as_slice()),
    ] {
        let old = fs::read(p).unwrap();
        fs::write(p, bytes).unwrap();
        let error = check_prepared(&prepared, &config).unwrap_err();
        assert!(error.contains("frozen input changed"));
        report(
            &["orchestration:configure:207", "orchestration:configure:297"],
            "changed frozen physical input",
            &json!({"control":config,"path":p,"old_hex":old,"mutant_hex":bytes}),
            &error,
        );
        fs::write(p, &old).unwrap();
        check_prepared(&prepared, &config).unwrap();
    }
    let mut unsealed = config.clone();
    unsealed["files"]
        .as_object_mut()
        .unwrap()
        .remove(source.to_str().unwrap());
    assert!(
        check_prepared(&prepared, &unsealed)
            .unwrap_err()
            .contains("not sealed")
    );
    let empty = PreparedInputs {
        compiler: BTreeMap::new(),
        sources: prepared.sources.clone(),
    };
    assert!(
        check_prepared(&empty, &config)
            .unwrap_err()
            .contains("empty")
    );
    fs::remove_dir_all(root).unwrap();
}
#[test]
fn fixed_cells_rederive_durable_source_and_support_inputs() {
    let root = root();
    let manifest = root.join("source.manifest");
    fs::write(&manifest, b"actual current source manifest\n").unwrap();
    let source_sha = custody::digest(&fs::read(&manifest).unwrap());
    let config = json!({"repo":root,"root":root.join("cycle")});
    let label = "workload-sqlite-cooperative-success";
    let control = cell_spec(
        label,
        &root.join("evidence"),
        &manifest,
        &source_sha,
        &config,
    )
    .unwrap();
    assert_eq!(
        control["Manifest"]["IntegratedSourceSHA256"],
        format!("sha256:{source_sha}")
    );
    let error = cell_spec(
        label,
        &root.join("evidence"),
        &manifest,
        &"a".repeat(64),
        &config,
    )
    .unwrap_err();
    assert!(error.contains("source manifest differs"));
    report(
        &["orchestration:configure:34"],
        "durable cell source mismatch",
        &json!({"control":control,"mutant_source_sha256":"a".repeat(64)}),
        &error,
    );
    let error = cell_spec(
        label,
        &root.join("evidence"),
        &manifest,
        &source_sha,
        &json!({}),
    )
    .unwrap_err();
    assert!(error.contains("candidate repo missing"));
    report(
        &["orchestration:configure:18"],
        "durable cell missing actual binding",
        &json!({"control":config,"mutant":{}}),
        &error,
    );
    let cell_path = root.join("cell.json");
    let current_cell = cell_spec(
        label,
        &root.join("cycle/cells").join(label),
        &manifest,
        &source_sha,
        &config,
    )
    .unwrap();
    fs::write(&cell_path, inputs::canonical(&current_cell).unwrap()).unwrap();
    let mut mixed = config.clone();
    mixed["source_sha256"] = json!(source_sha);
    mixed["stages"] = json!({
        "ordinary-workload-tools-run": {"adapter":crate::cycle::model::Adapter::CheckedCommand},
        "workload-corpus": {"adapter":crate::cycle::model::Adapter::Corpus},
        (label): {"adapter":crate::cycle::model::Adapter::Qualify,"spec":cell_path}
    });
    let mut cell_files = BTreeMap::new();
    cell_inputs(&mixed, &mut cell_files).unwrap();
    assert!(cell_files.contains_key(cell_path.to_str().unwrap()));
    assert!(cell_files.contains_key(manifest.to_str().unwrap()));
    mixed["stages"][label]
        .as_object_mut()
        .unwrap()
        .remove("spec");
    assert!(
        cell_inputs(&mixed, &mut BTreeMap::new())
            .unwrap_err()
            .contains("missing binding path: spec")
    );
    let support = root.join("support.json");
    let captured = root.join("captured.json");
    fs::write(&support, b"{}").unwrap();
    fs::write(&captured, b"{}").unwrap();
    let selected = json!({"root":root,"stages":{},"fn222":{"argv":["current-receiver","--support",support,"--captured-config",captured]}});
    cell_inputs(&selected, &mut BTreeMap::new()).unwrap();
    for flag in ["--support", "--captured-config"] {
        let mut bad = selected.clone();
        let argv = bad["fn222"]["argv"].as_array_mut().unwrap();
        let i = argv.iter().position(|v| v == flag).unwrap();
        argv.remove(i);
        let error = cell_inputs(&bad, &mut BTreeMap::new()).unwrap_err();
        assert!(error.contains("must be explicit"));
        report(
            &["orchestration:configure:238"],
            "explicit support and captured policy",
            &bad,
            &error,
        );
    }
    fs::remove_dir_all(root).unwrap();
}
#[test]
fn frozen_input_alias_and_relative_names_refuse_at_receiving() {
    let root = root();
    let input = root.join("input");
    let alias = root.join("alias");
    fs::write(&input, b"actual immutable bytes").unwrap();
    let mut files = BTreeMap::from([(
        input.to_str().unwrap().into(),
        custody::digest(b"actual immutable bytes"),
    )]);
    custody::check_files(&files).unwrap();
    symlink(&input, &alias).unwrap();
    files.clear();
    files.insert(
        alias.to_str().unwrap().into(),
        custody::digest(b"actual immutable bytes"),
    );
    let error = custody::check_files(&files).unwrap_err();
    report(
        &["orchestration:configure:297"],
        "frozen input symlink",
        &json!({"control":input,"mutant":alias}),
        &error,
    );
    files.clear();
    files.insert(
        "relative-input".into(),
        custody::digest(b"actual immutable bytes"),
    );
    let error = custody::check_files(&files).unwrap_err();
    assert!(error.contains("absolute clean"));
    report(
        &["orchestration:configure:297"],
        "frozen input relative path",
        &json!(files),
        &error,
    );
    fs::remove_dir_all(root).unwrap();
}

#[test]
fn file_boundary_retains_count_digest_and_per_file_limits() {
    let root = root();
    let file = root.join("limit-input");
    std::fs::write(&file, b"actual control").unwrap();
    let valid = BTreeMap::from([(
        file.to_str().unwrap().to_owned(),
        custody::digest(b"actual control"),
    )]);
    custody::check_files(&valid).unwrap();
    let too_many: BTreeMap<_, _> = (0..50_001)
        .map(|n| (format!("/unused-count-{n}"), "0".repeat(64)))
        .collect();
    assert!(
        custody::check_files(&too_many)
            .unwrap_err()
            .contains("count bound")
    );
    for expected in ["malformed".into(), "0".repeat(64)] {
        let mutant = BTreeMap::from([(file.to_str().unwrap().to_owned(), expected)]);
        assert!(custody::check_files(&mutant).is_err());
    }
    let huge = root.join("per-file-bound");
    std::fs::File::create(&huge)
        .unwrap()
        .set_len((128 << 20) + 1)
        .unwrap();
    let oversized = BTreeMap::from([(huge.to_str().unwrap().to_owned(), "0".repeat(64))]);
    assert!(
        custody::check_files(&oversized)
            .unwrap_err()
            .contains("bound")
    );
    custody::check_files(&valid).unwrap();
    std::fs::remove_dir_all(root).unwrap();
}
