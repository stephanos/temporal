//! Receiving tests for original `installation/installation_test.go` roots.
use super::super::manifest;
use super::*;
use crate::evidence::canonical;
use std::{fs, path::PathBuf};

fn temporary() -> crate::campaign::prepare::process::Temporary {
    crate::campaign::prepare::process::Temporary::new(&std::env::temp_dir()).unwrap()
}

fn installation(root: &Path, profile: &str, original: Value) -> manifest::Installation {
    let compiler = root.join("go");
    fs::write(&compiler, b"pinned compiler").unwrap();
    let config = convert(original, profile).unwrap();
    let cooperative = profile == COOPERATIVE;
    manifest::Installation {
        schema: manifest::SCHEMA.into(),
        admission: None,
        compiler_sha256: canonical::digest(&fs::read(&compiler).unwrap()),
        compiler_path: compiler,
        go_version: "go1.27.1".into(),
        runtime_root: if cooperative {
            root.to_path_buf()
        } else {
            PathBuf::new()
        },
        overlay_module_root: None,
        cache_root: root.to_path_buf(),
        profile: profile.into(),
        fuel: 4_000_000_000,
        memory_bytes: 256 << 20,
        helper_sha256: linked::IMPLEMENTATION_SHA256.into(),
        host_sha256: env!("GOMAD_HOST_IMPLEMENTATION_SHA256").into(),
        runtime_implementation_sha256: cooperative
            .then(|| env!("GOMAD_RUNTIME_IMPLEMENTATION_SHA256").into()),
        model: config.model,
        namespace: config.namespace,
        storage: config.storage,
        network: config.network,
        build_tags: config.build_tags,
        excluded_build_tags: Vec::new(),
    }
}

fn storage(capacity: u64) -> Value {
    serde_json::json!({"volumes":[{"ID":"data","Path":"/workspace","CapacityBytes":capacity}],
        "limits":{"PendingOperations":100,"Transitions":1000}})
}

fn network() -> Value {
    serde_json::json!({"profile":"gomad.simulation-network-stream/v3","limits":{"listeners":128,"connections":128,
        "bytes":8 << 20,"connection_bytes":4 << 20,"stall_nanos":10_000_000_000_u64,"transitions":10000}})
}

/// Original `TestInstallationAdmitsCooperativeStorageAndNetwork`.
#[test]
fn installation_admits_cooperative_storage_and_network() {
    let mut original = default_value(COOPERATIVE);
    original["storage"] = storage(1 << 20);
    original["network_v2"] = network();
    let configured = convert(original.clone(), COOPERATIVE).unwrap();
    assert!(configured.storage.is_some() && configured.network.is_some());
    original["storage"] = storage(1 << 30);
    assert!(
        convert(original, COOPERATIVE).is_err(),
        "invalid storage capacity admitted"
    );
}

/// Original `TestInstallationRejectsStockModelsBeforePreparation`: the stock
/// profile refuses operator network and storage at configuration admission.
#[test]
fn installation_rejects_stock_models_before_preparation() {
    for (field, value) in [("network_v2", network()), ("storage", storage(1 << 20))] {
        let mut original = default_value(STOCK);
        original[field] = value;
        let Err(error) = convert(original, STOCK) else {
            panic!("stock {field} admitted");
        };
        assert_eq!(
            error,
            "network and storage require cooperative profile and network requires frozen clock",
            "{field}"
        );
    }
}

/// Original `TestInstallationAndConfigRejectReferenceSelection`: reference
/// selectors fail strict decoding of configurations and manifests.
#[test]
fn installation_and_config_reject_reference_selection() {
    for field in ["GoBrokerReference", "remoteModel"] {
        let mut original = default_value(COOPERATIVE);
        original[field] = true.into();
        let error = read_config(&serde_json::to_vec(&original).unwrap()).unwrap_err();
        assert!(error.contains("unknown field"), "{field}: {error}");
        let root = temporary();
        let mut installed = serde_json::to_value(installation(
            &root.0,
            COOPERATIVE,
            default_value(COOPERATIVE),
        ))
        .unwrap();
        installed[field] = true.into();
        let path = root.0.join("installation.json");
        fs::write(&path, serde_json::to_vec(&installed).unwrap()).unwrap();
        assert!(
            manifest::resolve(Some(&path)).is_err(),
            "{field} manifest admitted"
        );
    }
}

/// Original `TestResolveRejectsChangedIdentityAndUnsafeManifest`.
#[test]
fn resolve_rejects_changed_identity_and_unsafe_manifest() {
    let root = temporary();
    let installed = installation(&root.0, STOCK, default_value(STOCK));
    let path = root.0.join("installation.json");
    let encode = |value: &manifest::Installation| serde_json::to_vec(value).unwrap();
    for mutate in [
        (|m: &mut manifest::Installation| m.schema = "unknown".into())
            as fn(&mut manifest::Installation),
        |m| m.helper_sha256 = "0".repeat(64),
        |m| m.host_sha256 = "0".repeat(64),
        |m| m.profile = "unknown".into(),
        |m| m.go_version = "go1.26.0".into(),
        |m| m.cache_root = "relative".into(),
        |m| m.compiler_sha256 = "invalid".into(),
    ] {
        let mut changed: manifest::Installation =
            serde_json::from_slice(&encode(&installed)).unwrap();
        mutate(&mut changed);
        fs::write(&path, encode(&changed)).unwrap();
        assert!(
            manifest::resolve(Some(&path)).is_err(),
            "accepted altered manifest"
        );
    }
    let data = encode(&installed);
    let mut duplicated = String::from_utf8(data.clone()).unwrap();
    duplicated.insert_str(1, r#""fuel":1,"#);
    for invalid in [
        [data.as_slice(), b" {}"].concat(),
        format!(r#"{{"schema":"{}","unknown":true}}"#, manifest::SCHEMA).into_bytes(),
        format!(r#"{{"schema":"unknown","Schema":"{}"}}"#, manifest::SCHEMA).into_bytes(),
        duplicated.into_bytes(),
    ] {
        fs::write(&path, invalid).unwrap();
        assert!(
            manifest::resolve(Some(&path)).is_err(),
            "accepted non-strict installation JSON"
        );
    }
    fs::write(&path, &data).unwrap();
    manifest::resolve(Some(&path)).unwrap();
    let link = root.0.join("manifest-link");
    std::os::unix::fs::symlink(&path, &link).unwrap();
    assert!(
        manifest::resolve(Some(&link)).is_err(),
        "accepted symbolic-link manifest"
    );
    fs::write(&installed.compiler_path, b"changed executable").unwrap();
    assert!(
        manifest::admit_build(&installed).is_err(),
        "accepted changed compiler"
    );
}

/// Original `TestWritePreservesExistingInstallation`.
#[test]
fn write_preserves_existing_installation() {
    let root = temporary();
    let installed = installation(&root.0, STOCK, default_value(STOCK));
    let path = root.0.join("installation.json");
    manifest::write(&path, &installed).unwrap();
    let before = fs::read(&path).unwrap();
    assert!(
        manifest::write(&path, &installed).is_err(),
        "overwrote existing manifest"
    );
    assert_eq!(fs::read(&path).unwrap(), before);
    manifest::resolve(Some(&path)).unwrap();
}
