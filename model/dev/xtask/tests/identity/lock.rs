use serde_json::{Value, json};
use std::fmt::Write;
use std::{
    collections::BTreeMap,
    fs,
    path::PathBuf,
    sync::atomic::{AtomicU64, Ordering},
};

struct Fixture {
    root: PathBuf,
    metadata: Value,
    lock: String,
}

impl Fixture {
    fn new() -> Self {
        static NEXT: AtomicU64 = AtomicU64::new(0);
        let root = std::env::temp_dir().join(format!(
            "xtask lock {} {}",
            std::process::id(),
            NEXT.fetch_add(1, Ordering::Relaxed)
        ));
        fs::create_dir_all(&root).unwrap();
        Self {
            root,
            metadata: json!({"packages":[
                {"id":"/checkout/guest", "name":"guest", "version":"1", "source":null},
                {"id":"/checkout/core", "name":"core", "version":"1", "source":null},
                {"id":"registry serde", "name":"serde", "version":"1", "source":"registry+example"},
                {"id":"registry windows", "name":"windows", "version":"1", "source":"registry+example"}
            ], "resolve":{"nodes":[
                {"id":"/checkout/guest", "deps":[{"pkg":"/checkout/core", "dep_kinds":[{"kind":null,"target":null}]}]},
                {"id":"/checkout/core", "deps":[
                    {"pkg":"registry serde", "dep_kinds":[{"kind":null,"target":null}]},
                    {"pkg":"registry windows", "dep_kinds":[{"kind":"build","target":"cfg(windows)"}]}
                ]},
                {"id":"registry serde", "deps":[]},
                {"id":"registry windows", "deps":[]}
            ]}}),
            lock: format!(
                "version = 4\n[[package]]\nname = 'serde'\nversion = '1'\nsource = 'registry+example'\nchecksum = '{}'\n[[package]]\nname = 'windows'\nversion = '1'\nsource = 'registry+example'\nchecksum = '{}'\n",
                "a".repeat(64),
                "b".repeat(64)
            ),
        }
    }

    fn publish(&self) -> xtask::Result<PathBuf> {
        let lockfile = self.root.join("Cargo.lock");
        fs::write(&lockfile, &self.lock).unwrap();
        xtask::lock::publish(
            &serde_json::to_vec(&self.metadata).unwrap(),
            &lockfile,
            &self.root.join("target"),
        )
    }

    fn map(&self) -> BTreeMap<String, Option<String>> {
        let value: Value =
            serde_json::from_slice(&fs::read(self.publish().unwrap()).unwrap()).unwrap();
        assert_eq!(value["schema"], 2);
        serde_json::from_value(value["closures"].clone()).unwrap()
    }
}

impl Drop for Fixture {
    fn drop(&mut self) {
        fs::remove_dir_all(&self.root).unwrap();
    }
}

#[test]
fn content_addressed_publication_is_repeatable_and_portable() {
    let first = Fixture::new();
    let baseline = first.map();
    let path = first.publish().unwrap();
    let bytes = fs::read(&path).unwrap();
    assert_eq!(path, first.publish().unwrap());
    assert_eq!(
        path.file_name().unwrap().to_str().unwrap(),
        format!("umpire-lock-{}.json", xtask::support::hash(&bytes))
    );
    assert!(!String::from_utf8(bytes).unwrap().contains("checkout"));
    let mut relocated = Fixture::new();
    relocated.metadata = serde_json::from_str(
        &serde_json::to_string(&relocated.metadata)
            .unwrap()
            .replace("/checkout", "/another checkout"),
    )
    .unwrap();
    relocated.metadata["packages"].as_array_mut().unwrap().reverse();
    relocated.metadata["resolve"]["nodes"].as_array_mut().unwrap().reverse();
    relocated.metadata["resolve"]["nodes"][2]["deps"].as_array_mut().unwrap().reverse();
    assert_eq!(relocated.map(), baseline);
}

#[test]
fn unrelated_and_dev_only_dependencies_do_not_change_guest_closures() {
    let fixture = Fixture::new();
    let baseline = fixture.map();
    for dev in [false, true] {
        let mut variant = Fixture::new();
        variant.metadata["packages"]
            .as_array_mut()
            .unwrap()
            .push(json!({"id":"tool", "name":"tool", "version":"1", "source":null}));
        variant.metadata["resolve"]["nodes"]
            .as_array_mut()
            .unwrap()
            .push(json!({"id":"tool", "deps":[]}));
        if dev {
            variant.metadata["resolve"]["nodes"][1]["deps"]
                .as_array_mut()
                .unwrap()
                .push(json!({"pkg":"tool", "dep_kinds":[{"kind":"dev"}]}));
        }
        let changed = variant.map();
        for (name, identity) in &baseline {
            assert_eq!(changed[name], *identity);
        }
    }
}

#[test]
fn checksum_changes_propagate_through_normal_and_all_platform_build_edges() {
    let baseline = Fixture::new().map();
    for (checksum, affected, unaffected) in
        [('a', "serde 1", "windows 1"), ('b', "windows 1", "serde 1")]
    {
        let mut changed = Fixture::new();
        changed.lock = changed.lock.replace(&checksum.to_string().repeat(64), &"c".repeat(64));
        let changed = changed.map();
        for name in ["guest 1", "core 1", affected] {
            assert_ne!(changed[name], baseline[name]);
        }
        assert_eq!(changed[unaffected], baseline[unaffected]);
    }
}

#[test]
fn unified_edges_and_edge_topology_change_ancestor_identity() {
    let baseline = Fixture::new().map();
    for node in [0, 2] {
        let mut changed = Fixture::new();
        changed.metadata["resolve"]["nodes"][node]["deps"]
            .as_array_mut()
            .unwrap()
            .push(json!({"pkg":"registry windows", "dep_kinds":[{"kind":null}]}));
        let changed = changed.map();
        assert_ne!(changed["guest 1"], baseline["guest 1"]);
        assert_eq!(changed["windows 1"], baseline["windows 1"]);
    }
}

#[test]
fn ambiguous_name_version_coordinates_are_marked_unavailable() {
    for source in [Value::Null, json!("git+https://example.invalid/serde#123")] {
        let mut fixture = Fixture::new();
        fixture.metadata["packages"]
            .as_array_mut()
            .unwrap()
            .push(json!({"id":"another serde", "name":"serde", "version":"1", "source":source}));
        fixture.metadata["resolve"]["nodes"]
            .as_array_mut()
            .unwrap()
            .push(json!({"id":"another serde", "deps":[]}));
        if let Some(source) = source.as_str() {
            write!(
                fixture.lock,
                "[[package]]\nname = 'serde'\nversion = '1'\nsource = '{source}'\n"
            )
            .unwrap();
        }
        let map = fixture.map();
        assert_eq!(map["serde 1"], None);
        assert!(map["guest 1"].is_some());
    }
}

#[test]
fn sourced_entries_are_required_but_git_checksums_are_optional() {
    let mut fixture = Fixture::new();
    fixture.lock = "version = 4\n".into();
    assert!(fixture.publish().unwrap_err().to_string().contains("no lockfile entry"));
    fixture.lock = "version = 4\n[[package]]\nname = 'serde'\nversion = '1'\nsource = 'git+https://example.invalid/serde#123'\n[[package]]\nname = 'windows'\nversion = '1'\nsource = 'registry+example'\n".into();
    fixture.metadata["packages"][2]["source"] = json!("git+https://example.invalid/serde#123");
    assert!(fixture.map()["guest 1"].is_some());
}

#[test]
fn malformed_native_documents_and_graphs_are_rejected() {
    let baseline = Fixture::new();
    let mut cases = Vec::new();
    let mut add = |label, metadata: Value| cases.push((label, metadata));
    let mut value = baseline.metadata.clone();
    value["resolve"] = Value::Null;
    add("missing resolve", value);
    let mut value = baseline.metadata.clone();
    value["resolve"]["nodes"][0]["deps"][0]["pkg"] = json!("missing");
    add("dangling edge", value);
    let mut value = baseline.metadata.clone();
    value["resolve"]["nodes"].as_array_mut().unwrap().pop();
    add("missing node", value);
    let mut value = baseline.metadata.clone();
    value["packages"][0] = value["packages"][1].clone();
    add("duplicate package", value);
    let mut value = baseline.metadata.clone();
    value["resolve"]["nodes"][0] = value["resolve"]["nodes"][1].clone();
    add("duplicate node", value);
    let mut value = baseline.metadata.clone();
    value["resolve"]["nodes"][0]["deps"][0]["dep_kinds"] = json!([]);
    add("empty kinds", value);
    let mut value = baseline.metadata.clone();
    value["resolve"]["nodes"][0]["deps"][0]["dep_kinds"][0]["kind"] = json!("unknown");
    add("unknown kind", value);
    let mut value = baseline.metadata.clone();
    value["packages"][0]["id"] = json!("");
    add("empty ID", value);
    let mut value = baseline.metadata.clone();
    value["resolve"]["nodes"][3]["deps"] =
        json!([{"pkg":"/checkout/core", "dep_kinds":[{"kind":null}]}]);
    add("cycle", value);
    let mut value = baseline.metadata.clone();
    value["packages"][0]["name"] = json!("core");
    add("duplicate coordinate", value);
    for (label, metadata) in cases {
        let mut fixture = Fixture::new();
        fixture.metadata = metadata;
        assert!(fixture.publish().is_err(), "{label}");
    }
    let mut fixture = Fixture::new();
    fixture.lock = "not toml".into();
    assert!(fixture.publish().is_err());
    fs::write(baseline.root.join("Cargo.lock"), &baseline.lock).unwrap();
    assert!(
        xtask::lock::publish(b"not json", &baseline.root.join("Cargo.lock"), &baseline.root)
            .is_err()
    );
}
