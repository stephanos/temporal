use super::*;
use crate::campaign::prepare::process::Temporary;

fn repository() -> (Temporary, Repo) {
    let root =
        Temporary::new(&Path::new(env!("CARGO_MANIFEST_DIR")).join("target/derive-tests")).unwrap();
    fs::write(root.0.join("go.mod"), "module example.test/repo\n").unwrap();
    for directory in ["runtime", "schema/sqlite", "pkg", "pkg2"] {
        fs::create_dir_all(root.0.join(directory)).unwrap();
    }
    fs::write(root.0.join("runtime/runtime.go"), "package runtime\n").unwrap();
    fs::write(
        root.0.join("schema/sqlite/schema.sql"),
        "CREATE TABLE example(id INTEGER);\n",
    )
    .unwrap();
    fs::write(
        root.0.join("gomad.toml"),
        "runtime_root='runtime'\nderive='temporal'\n",
    )
    .unwrap();
    let repo = Repo::discover(&root.0).unwrap();
    (root, repo)
}

// These unit artifacts exercise cache authentication without launching a compiler.
// Pinned-Go derivation and guest acceptance are separate integration gates.
fn publish_candidate(root: &Path, repo: &Repo) -> (PathBuf, PathBuf) {
    let compiler = std::env::current_exe().unwrap();
    let (directory, _handle) = directory(root).unwrap();
    let mut candidate = candidate(&compiler, repo, "pkg", &directory).unwrap();
    candidate.admission.compiler_inputs.insert(
        compiler.clone(),
        candidate.installed.compiler_sha256.clone(),
    );
    candidate.admission.inputs_sha256 =
        canonical::digest(&canonical::encode(&candidate.admission.compiler_inputs).unwrap());
    candidate.installed.admission = Some(candidate.admission);
    let path = directory.join(format!("{}.json", candidate.key));
    manifest::write(&path, &candidate.installed).unwrap();
    (compiler, path)
}

#[test]
fn derive_cache_reuses_authenticated_unchanged_configuration_and_isolates_packages() {
    let (root, repo) = repository();
    let cache = root.0.join("cache");
    let (compiler, path) = publish_candidate(&cache, &repo);
    let before = fs::metadata(&path).unwrap();
    let original = manifest::read(&path, 16 << 20).unwrap();
    for _ in 0..2 {
        assert_eq!(derive(&compiler, &repo, "pkg", &cache).unwrap(), path);
    }
    assert_eq!(manifest::read(&path, 16 << 20).unwrap(), original);
    assert_eq!(fs::metadata(&path).unwrap().ino(), before.ino());
    let installed = manifest::resolve(Some(&path)).unwrap().manifest;
    let explicit = config::build(PROFILE, None, Some((&repo.module_root, "pkg"))).unwrap();
    assert_eq!(
        canonical::encode(&installed.admission.unwrap().guest_configuration).unwrap(),
        canonical::encode(&explicit.original).unwrap()
    );
    let owner = path.parent().unwrap();
    assert_ne!(
        candidate(&compiler, &repo, "pkg", owner).unwrap().key,
        candidate(&compiler, &repo, "pkg2", owner).unwrap().key
    );
    assert!(candidate(&compiler, &repo, ".", owner).is_err());
}

#[test]
fn derive_cache_captures_schema_before_lookup_and_refuses_absent_schema() {
    let (root, repo) = repository();
    let cache = root.0.join("cache");
    let (compiler, path) = publish_candidate(&cache, &repo);
    let first = candidate(&compiler, &repo, "pkg", path.parent().unwrap()).unwrap();
    fs::write(
        root.0.join("schema/sqlite/schema.sql"),
        "CREATE TABLE changed(id INTEGER);\n",
    )
    .unwrap();
    let changed = candidate(&compiler, &repo, "pkg", path.parent().unwrap()).unwrap();
    assert_ne!(first.key, changed.key);
    assert_ne!(
        first.admission.configuration_sha256,
        changed.admission.configuration_sha256
    );
    assert!(
        !path
            .parent()
            .unwrap()
            .join(format!("{}.json", changed.key))
            .exists()
    );
    assert!(admit(&path, &changed).is_err());
    fs::remove_dir_all(root.0.join("schema/sqlite")).unwrap();
    assert!(derive(&compiler, &repo, "pkg", &cache).is_err());
    assert!(path.is_file());
}

#[test]
fn derive_cache_refuses_changed_implementation_executable_and_unpinned_inputs() {
    for field in [
        "host_sha256",
        "helper_sha256",
        "host_executable_sha256",
        "compiler_inputs",
    ] {
        let (root, repo) = repository();
        let cache = root.0.join("cache");
        let (compiler, path) = publish_candidate(&cache, &repo);
        let raw = manifest::read(&path, 16 << 20).unwrap();
        let mut value: serde_json::Value = serde_json::from_slice(&raw).unwrap();
        if matches!(field, "host_sha256" | "helper_sha256") {
            value[field] = serde_json::json!("0".repeat(64));
        } else if field == "host_executable_sha256" {
            value["admission"][field] = serde_json::json!("0".repeat(64));
        } else {
            let input = root.0.join("pinned-input");
            fs::write(&input, "current").unwrap();
            let inputs = BTreeMap::from([(input, canonical::digest(b"previous"))]);
            value["admission"][field] = serde_json::to_value(&inputs).unwrap();
            value["admission"]["inputs_sha256"] =
                serde_json::json!(canonical::digest(&canonical::encode(&inputs).unwrap()));
        }
        fs::write(&path, canonical::encode(&value).unwrap()).unwrap();
        assert!(
            derive(&compiler, &repo, "pkg", &cache).is_err(),
            "accepted changed {field}"
        );
        assert_eq!(
            manifest::read(&path, 16 << 20).unwrap(),
            canonical::encode(&value).unwrap()
        );
    }
}

#[test]
fn derive_cache_serializes_concurrent_hits_and_refuses_unsafe_owners() {
    let (root, repo) = repository();
    let cache = root.0.join("cache");
    let (compiler, path) = publish_candidate(&cache, &repo);
    std::thread::scope(|scope| {
        let handles = (0..4)
            .map(|_| scope.spawn(|| derive(&compiler, &repo, "pkg", &cache).unwrap()))
            .collect::<Vec<_>>();
        for handle in handles {
            assert_eq!(handle.join().unwrap(), path);
        }
    });
    let changed = root.0.join("runtime/changed.go");
    fs::write(&changed, "package runtime\n").unwrap();
    let new = candidate(&compiler, &repo, "pkg", path.parent().unwrap()).unwrap();
    assert_ne!(path.file_stem().unwrap(), new.key.as_str());
    fs::remove_file(changed).unwrap();
    fs::set_permissions(path.parent().unwrap(), fs::Permissions::from_mode(0o755)).unwrap();
    assert!(derive(&compiler, &repo, "pkg", &cache).is_err());
}
