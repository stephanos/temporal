use super::*;
use std::os::unix::fs::PermissionsExt;
struct Temporary(PathBuf);
impl Temporary {
    fn new() -> Self {
        let root = std::env::temp_dir().join(format!(
            "gomad-native-module-{}-{}",
            std::process::id(),
            std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_nanos()
        ));
        fs::create_dir(&root).unwrap();
        Self(root)
    }
}
impl Drop for Temporary {
    fn drop(&mut self) {
        let _ = fs::remove_dir_all(&self.0);
    }
}
fn strings(values: &[&str]) -> Vec<String> {
    values.iter().map(|value| (*value).into()).collect()
}
#[test]
fn diagnostic_module_build_commands() {
    for (kind, tags, list, build) in [
        (
            "test",
            strings(&["test_dep"]),
            strings(&[
                "/go",
                "list",
                "-p=1",
                "-deps",
                "-test",
                "-json",
                "-overlay",
                "/overlay",
                "-tags",
                "test_dep",
                "./fixture",
            ]),
            strings(&[
                "/go",
                "test",
                "-c",
                "-p=1",
                "-trimpath",
                "-overlay",
                "/overlay",
                "-o",
                "/module",
                "-tags",
                "test_dep",
                "./fixture",
            ]),
        ),
        (
            "build",
            vec![],
            strings(&[
                "/go",
                "list",
                "-p=1",
                "-deps",
                "-json",
                "-overlay",
                "/overlay",
                "./fixture",
            ]),
            strings(&[
                "/go",
                "build",
                "-p=1",
                "-trimpath",
                "-buildvcs=false",
                "-overlay",
                "/overlay",
                "-o",
                "/module",
                "./fixture",
            ]),
        ),
    ] {
        assert_eq!(
            diagnostic_module_commands("/go", "/overlay", "/module", "./fixture", kind, &tags)
                .unwrap(),
            (list, build)
        );
    }
    assert!(
        diagnostic_module_commands("/go", "/overlay", "/module", "./fixture", "run", &[]).is_err()
    );
}
#[test]
fn diagnostic_module_source_bindings() {
    let temp = Temporary::new();
    let compiler = temp.0.join("stock/bin/go");
    let overlay = temp.0.join("overlay.json");
    let mut paths = vec![
        compiler.clone(),
        temp.0.join("stock/VERSION"),
        temp.0.join("go.mod"),
        temp.0.join("go.sum"),
        temp.0.join("fixture.go"),
        temp.0.join("fixture.h"),
        temp.0.join("fixture.syso"),
        temp.0.join("replacement.go"),
        temp.0.join("added-replacement.go"),
        temp.0.join("embedded.txt"),
    ];
    let platform = format!(
        "{}_{}",
        if std::env::consts::OS == "macos" {
            "darwin"
        } else {
            std::env::consts::OS
        },
        match std::env::consts::ARCH {
            "x86_64" => "amd64",
            "aarch64" => "arm64",
            other => other,
        }
    );
    for tool in ["compile", "link", "asm"] {
        paths.push(temp.0.join("stock/pkg/tool").join(&platform).join(tool));
    }
    for path in &paths {
        fs::create_dir_all(path.parent().unwrap()).unwrap();
        fs::write(path, b"original").unwrap();
    }
    let added = temp.0.join("added.go");
    fs::write(&overlay,serde_json::to_vec(&json!({"Replace":{temp.0.join("fixture.go").to_string_lossy().as_ref():temp.0.join("replacement.go"),added.to_string_lossy().as_ref():temp.0.join("added-replacement.go")}})).unwrap()).unwrap();
    let listed=serde_json::to_vec(&json!({"Dir":temp.0,"GoFiles":["fixture.go","added.go"],"HFiles":["fixture.h"],"SysoFiles":["fixture.syso"],"TestEmbedFiles":["embedded.txt"]})).unwrap();
    let inputs = source_bindings(&listed, &compiler, &overlay, &temp.0).unwrap();
    assert!(!inputs.contains_key(&added));
    paths.push(overlay);
    for path in paths {
        let original = read(&path, 1 << 20).unwrap();
        assert_eq!(inputs[&path], canonical::digest(&original));
        fs::write(&path, b"changed").unwrap();
        assert!(verify_inputs(&inputs).is_err(), "{}", path.display());
        fs::write(&path, &original).unwrap();
    }
}
#[test]
fn diagnostic_recovery_compiler_environment() {
    let call = Invocation {
        name: "fixture".into(),
        repo: "/repo".into(),
        go_command: "go".into(),
        inputs: BTreeMap::new(),
        environ: strings(&["GOMAXPROCS=9", "GOTMPDIR=/private/compiler-temp"]),
        keep: false,
    };
    let controls = compiler_controls(&call).unwrap();
    assert_eq!(controls.parallelism, Some(1));
    let mut environment = preparation::process::environment().unwrap();
    environment.extend(controls.environment);
    for (key, want) in [
        ("GOMAXPROCS", "1"),
        ("GOTMPDIR", "/private/compiler-temp"),
        ("GOENV", "off"),
        ("GOTOOLCHAIN", "local"),
        ("GOWORK", "off"),
        ("GOFLAGS", ""),
    ] {
        assert_eq!(environment[key], want);
    }
    let mut invalid = call;
    invalid.environ = strings(&["GOTMPDIR=relative"]);
    assert!(compiler_controls(&invalid).is_err());
}
#[test]
fn prototype_temporal_configuration_preserves_complete_capture() {
    let temp = Temporary::new();
    let schema = temp.0.join("schema/sqlite");
    fs::create_dir_all(&schema).unwrap();
    fs::set_permissions(&schema, fs::Permissions::from_mode(0o700)).unwrap();
    let data = b"CREATE TABLE example(id INTEGER);";
    fs::write(schema.join("schema.sql"), data).unwrap();
    fs::set_permissions(schema.join("schema.sql"), fs::Permissions::from_mode(0o600)).unwrap();
    let arguments = strings(&[
        "-test.run=^TestActivityTestSuite$",
        "-persistenceType=sql",
        "-persistenceDriver=sqlite",
    ]);
    let actual = temporal_configuration(&temp.0, "/workspace/tests", &arguments).unwrap();
    assert_eq!(
        actual["Args"],
        json!([
            "temporal.test",
            "-test.v",
            "-test.count=1",
            "-test.timeout=72h",
            "-test.run=^TestActivityTestSuite$",
            "-persistenceType=sql",
            "-persistenceDriver=sqlite"
        ])
    );
    assert_eq!(
        actual["Environment"],
        json!([
            "PWD=/workspace/tests",
            "TEMPORAL_ROOT=/workspace",
            "TMPDIR=/tmp",
            "TZ=UTC",
            "TEMPORAL_TEST_LOG_LEVEL=warn"
        ])
    );
    assert_eq!(actual["Profile"], COOPERATIVE);
    assert_eq!(actual["WorkingDirectory"], "/workspace/tests");
    assert_eq!(
        actual["WritableDirectories"],
        json!(["/workspace", "/tmp", "/workspace/tests"])
    );
    use sha2::Digest;
    assert_eq!(
        actual["EntropyKey"],
        json!(sha2::Sha256::digest(b"gomad-wasm-prototype/v1").to_vec())
    );
    assert_eq!(
        actual["Clock"],
        json!({"EpochNanos":946684800000000000_u64,"ReadStepNanos":0})
    );
    assert_eq!(
        actual["Limits"],
        json!({"OutputBytes":256<<20,"TranscriptBytes":1<<30,"Calls":1<<28,"Descriptors":4096,"Files":100000,"FilesystemBytes":512<<20,"PendingEvents":4096})
    );
    let captured = &actual["CapturedInputs"];
    let descriptor =
        crate::qualify::cells::base64(captured["Descriptor"].as_str().unwrap()).unwrap();
    assert_eq!(
        captured["Manifest"]["SHA256"],
        format!("sha256:{}", canonical::digest(&descriptor))
    );
    assert_eq!(captured["Manifest"]["Bytes"], descriptor.len());
    assert_eq!(captured["Manifest"]["TotalBytes"], data.len());
    let descriptor: Value = serde_json::from_slice(&descriptor).unwrap();
    assert_eq!(descriptor["mappings"], json!(["/workspace/schema/sqlite"]));
    let entries = descriptor["entries"].as_array().unwrap();
    assert_eq!(entries.len(), 2);
    let file = entries
        .iter()
        .find(|entry| entry["path"] == "/workspace/schema/sqlite/schema.sql")
        .unwrap();
    assert_eq!(
        file["sha256"],
        format!("sha256:{}", canonical::digest(data))
    );
    assert_eq!(file["mode"], "0600");
    assert_eq!(file["size"], data.len().to_string());
    let payload = captured["Payloads"][file["payload"].as_str().unwrap()]
        .as_str()
        .unwrap();
    assert_eq!(crate::qualify::cells::base64(payload).unwrap(), data);
}
#[test]
fn prototype_temporal_configuration_preserves_capture_error() {
    let temp = Temporary::new();
    let expected = crate::install::config::build(COOPERATIVE, None, Some((&temp.0, "tests")))
        .err()
        .unwrap();
    assert_eq!(
        temporal_configuration(&temp.0, "/workspace/tests", &[]).unwrap_err(),
        expected
    );
}
#[test]
fn runtime_selection_retains_actual_preparation_bounds() {
    let temp = Temporary::new();
    let (options, arguments, scenario_path) = options(
        &temp.0,
        &temp.0,
        &temp.0.join("install.json"),
        "go-run",
        "./runtime_cost",
        &[],
        &[],
        true,
        Duration::from_secs(180),
        None,
    )
    .unwrap();
    assert_eq!(options.execution_timeout, Duration::from_secs(180));
    let reparsed = crate::cli::explore::parse(&arguments, &mut false).unwrap();
    assert_eq!(reparsed.execution_timeout, options.execution_timeout);
    assert_eq!(reparsed.target_kind, options.target_kind);
    assert_eq!(reparsed.source, options.source);
    assert!(scenario_path.is_none());
    assert_eq!(options.choice_bytes, 64 << 20);
    assert!(!options.diagnostics);
    assert_eq!(options.target_kind, "go-run");
}

#[test]
fn runtime_configuration_preserves_original_request_without_widening() {
    let config = runtime_configuration().unwrap();
    assert_eq!(
        resource_bounds(&Selection::Runtime),
        Bounds {
            fuel: 1 << 40,
            memory_bytes: 256 << 20,
            timeout: Duration::from_secs(180),
            output_bytes: 1 << 20
        }
    );
    assert_eq!(config["Args"], json!(["guest", "literal $arg"]));
    assert_eq!(
        config["Environment"],
        json!(["PWD=/workspace", "TZ=UTC", "ONLY=captured"])
    );
    let mut entropy = vec![0_u8; 32];
    entropy[..3].copy_from_slice(&[1, 2, 3]);
    assert_eq!(config["EntropyKey"], json!(entropy));
    assert_eq!(
        config["Clock"],
        json!({"EpochNanos":946684800000000000_u64,"ReadStepNanos":0})
    );
    assert_eq!(
        config["Limits"],
        json!({"OutputBytes":1<<20,"TranscriptBytes":64<<20,"Calls":1<<20,"Descriptors":64,"Files":1000,"FilesystemBytes":8<<20,"PendingEvents":4096})
    );
}
