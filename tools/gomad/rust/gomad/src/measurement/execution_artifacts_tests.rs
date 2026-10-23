use super::*;

#[test]
fn actual_collection_and_catalog_publication_failures_are_both_returned() {
    let root = std::env::temp_dir().join(format!(
        "gomad-catalog-failure-{}-{}",
        std::process::id(),
        std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_nanos()
    ));
    std::fs::DirBuilder::new()
        .mode(0o700)
        .create(&root)
        .unwrap();
    let missing = root.join("missing-module.wasm");
    let expected_primary = std::fs::read(&missing).unwrap_err().to_string();
    write_raw(
        &root.join("variant-artifacts.json"),
        b"prior receipt must remain intact",
    )
    .unwrap();
    let raw=serde_json::to_vec(&json!({"kind":"standalone","guest":{"module":missing,"module_sha256":"a".repeat(64),
        "limits":{"memory_bytes":65536,"fuel":10000,"wall_millis":5000,"output_bytes":1024,"evidence_bytes":1<<20},
        "model":null,"namespace":null,"storage":null},"network":null})).unwrap();
    let request: Input = serde_json::from_slice(&raw).unwrap();
    let error = execute(&raw, request, &root, None).unwrap_err();
    assert!(error.starts_with(&expected_primary), "{error}");
    assert!(
        error[expected_primary.len()..].starts_with("; "),
        "catalog publication failure was lost: {error}"
    );
    assert_eq!(
        std::fs::read(root.join("variant-artifacts.json")).unwrap(),
        b"prior receipt must remain intact"
    );
    assert!(root.join("request.json").is_file());
    std::fs::remove_dir_all(root).unwrap();
}

#[test]
fn actual_digest_only_descriptor_refusal_publishes_without_inventing_model_evidence() {
    let root = std::env::temp_dir().join(format!(
        "gomad-digest-admission-{}-{}",
        std::process::id(),
        std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_nanos()
    ));
    std::fs::DirBuilder::new()
        .mode(0o700)
        .create(&root)
        .unwrap();
    let module = b"\0asm\x01\0\0\0\x01\x04\x01\x60\0\0\x03\x02\x01\0\x05\x03\x01\0\x02\x07\x13\x02\x06memory\x02\0\x06_start\0\0\x0a\x04\x01\x02\0\x0b";
    let module_path = root.join("input.wasm");
    write_raw(&module_path, module).unwrap();
    let configuration =
        crate::install::config::build("gomad.wasi-cooperative/v1", None, None).unwrap();
    let mut model = configuration.model;
    model["digest_only"] = json!(true);
    let environment = configuration
        .environment
        .into_iter()
        .map(|(name, value)| format!("{name}={value}"))
        .collect::<Vec<_>>();
    let raw = serde_json::to_vec(&json!({
        "kind":"standalone", "network":null,
        "guest":{
            "module":module_path,"module_sha256":canonical::digest(module),
            "limits":{"memory_bytes":65536,"fuel":10000,"wall_millis":5000,
                "output_bytes":configuration.namespace.limits.output_bytes,"evidence_bytes":16<<20},
            "namespace":configuration.namespace,"storage":null,
            "model":{"configuration":model,"arguments":configuration.arguments,
                "environment":environment,"replay_trace":null}
        }
    }))
    .unwrap();
    let result = execute(&raw, serde_json::from_slice(&raw).unwrap(), &root, None);
    let report = std::fs::read(root.join("report.json"));
    let outcome: Value =
        serde_json::from_slice(&std::fs::read(root.join("outcome.json")).unwrap()).unwrap();
    #[cfg(feature = "helper-diagnostic-profile")]
    let profile: Value =
        serde_json::from_slice(&std::fs::read(root.join("profile.json")).unwrap()).unwrap();
    let canonical_receipt = root.join("canonical-streams.json").exists();
    let fabricated = ["wasi", "terminal", "stdout", "stderr", "network"]
        .into_iter()
        .any(|name| root.join(name).exists());
    std::fs::remove_dir_all(&root).unwrap();
    assert!(
        result.is_ok(),
        "actual pre-instantiation refusal publication failed: {result:?}"
    );
    let report: Value = serde_json::from_slice(&report.unwrap()).unwrap();
    assert_eq!(outcome["termination"], "capacity");
    assert_eq!(outcome["message"], "initial guest memory exceeds limit");
    assert!(outcome["local_model"].is_null());
    assert!(outcome["metrics"]["prepare_nanos"].as_u64().unwrap() > 0);
    assert!(outcome["metrics"]["compile_nanos"].as_u64().unwrap() > 0);
    assert_eq!(outcome["metrics"]["instantiate_nanos"], 0);
    assert_eq!(outcome["metrics"]["execute_nanos"], 0);
    assert_eq!(
        report["admission_refusal"]["stage"],
        "loaded-module-descriptor"
    );
    assert_eq!(report["admission_refusal"]["outcome"], outcome);
    assert!(!canonical_receipt && !fabricated);
    #[cfg(feature = "helper-diagnostic-profile")]
    {
        assert_eq!(profile["phases"]["compile"]["count"], 1);
        assert_eq!(profile["phases"]["instantiate"]["count"], 0);
        assert_eq!(profile["phases"]["execute"]["count"], 0);
    }
}
