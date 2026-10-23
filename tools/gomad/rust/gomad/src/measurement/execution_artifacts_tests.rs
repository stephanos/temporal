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
