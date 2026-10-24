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

/// Original `wasi.TestTemporalClockConfigurationIdentity`: the frozen default
/// and the `read-step-1us-v1` clock selections are distinct installation
/// model identities.
#[test]
fn temporal_clock_selection_binds_configuration_identity() {
    let identities = [0_u64, 1000].map(|read_step| {
        let mut original = default_value(COOPERATIVE);
        original["Clock"]["ReadStepNanos"] = read_step.into();
        let configured = convert(original, COOPERATIVE).unwrap();
        assert_eq!(configured.model["read_step_nanos"], read_step);
        canonical::digest(&canonical::encode(&configured.model).unwrap())
    });
    assert_ne!(
        identities[0], identities[1],
        "clock selection was omitted from the configuration identity"
    );
}

/// Original `TestProviderAdmitsPinnedCooperativeProfile`: the cooperative
/// runtime is admitted only with its pinned go1.27.1 compiler, while the stock
/// profile keeps both supported compiler versions.
#[test]
fn cooperative_installation_refuses_a_compiler_outside_its_pinned_version() {
    let root = temporary();
    let path = root.0.join("installation.json");
    for (profile, version, admitted) in [
        (COOPERATIVE, "go1.27.1", true),
        (COOPERATIVE, "go1.27.0", false),
        (STOCK, "go1.27.0", true),
        (STOCK, "go1.27.1", true),
    ] {
        let mut installed = installation(&root.0, profile, default_value(profile));
        installed.go_version = version.into();
        fs::write(&path, serde_json::to_vec(&installed).unwrap()).unwrap();
        let resolved = manifest::resolve(Some(&path));
        assert_eq!(resolved.is_ok(), admitted, "{profile} {version}");
        if !admitted {
            assert!(
                resolved
                    .err()
                    .unwrap()
                    .contains("unsupported runtime profile or implementation")
            );
        }
    }
}

/// Original `TestDiscoverAdvertisesOnlyExecutableScenarioSupport`: discovery
/// reports the resolved installation and never overstates capabilities; only
/// the cooperative profile advertises typed scenarios, exact replay and a
/// choice profile, and neither advertises simulation.
#[test]
fn discover_advertises_only_executable_scenario_support() {
    for profile in [STOCK, COOPERATIVE] {
        let root = temporary();
        let path = root.0.join("installation.json");
        let installed = installation(&root.0, profile, default_value(profile));
        fs::write(&path, serde_json::to_vec(&installed).unwrap()).unwrap();
        let mut output = Vec::new();
        assert_eq!(
            crate::cli::discover::run(&[], &mut output, Some(&path))
                .unwrap_or_else(|error| panic!("{}", error.message)),
            0,
            "{profile}"
        );
        let response: Value = serde_json::from_slice(&output).unwrap();
        let cooperative = profile == COOPERATIVE;
        let capabilities = &response["capabilities"];
        assert_eq!(capabilities["Name"], "wasm");
        assert_eq!(capabilities["Simulation"], false);
        assert_eq!(capabilities["TypedScenarios"], cooperative, "{profile}");
        assert_eq!(
            capabilities["ReplayModes"],
            if cooperative {
                serde_json::json!(["exact", "observed"])
            } else {
                serde_json::json!(["observed"])
            }
        );
        assert_eq!(capabilities["ChoiceProfiles"].is_null(), !cooperative);
        let resolved = manifest::resolve(Some(&path)).unwrap().path;
        assert_eq!(
            response["path"],
            resolved.to_string_lossy().as_ref(),
            "{profile}: resolved installation path missing"
        );
        assert!(
            crate::cli::discover::run(&["extra".into()], &mut Vec::new(), Some(&path)).is_err()
        );
    }
}

/// Original `TestResolveInstallationPrecedenceAndSourceLessReplayInputs`: an
/// explicit manifest wins over the environment, the environment over the
/// executable sibling; the selected manifest resolves without its compiler,
/// and an invalid explicit selection never falls back to the environment.
#[test]
fn installation_precedence_selects_one_manifest_and_resolves_source_less() {
    let root = temporary();
    let path = root.0.join("gomad-wasm.json");
    let installed = installation(&root.0, STOCK, default_value(STOCK));
    fs::write(&path, serde_json::to_vec(&installed).unwrap()).unwrap();
    fs::remove_file(&installed.compiler_path).unwrap();
    let executable = |beside_manifest: bool| {
        let path = if beside_manifest {
            root.0.join("gomad")
        } else {
            PathBuf::from("/missing/bin/gomad")
        };
        move || -> std::io::Result<PathBuf> { Ok(path) }
    };
    let manifest_text = path.to_str().unwrap();
    for (explicit, environment, beside_manifest, source) in [
        (
            Some(path.as_path()),
            Some("/missing/environment"),
            false,
            "explicit",
        ),
        (None, Some(manifest_text), false, "environment"),
        (
            Some(Path::new("")),
            Some(manifest_text),
            false,
            "environment",
        ),
        (None, None, true, "executable-sibling"),
        (None, Some(""), true, "executable-sibling"),
    ] {
        let (selected, actual) = manifest::select(
            explicit,
            environment.map(Into::into),
            executable(beside_manifest),
        )
        .unwrap();
        assert_eq!((selected.as_path(), actual), (path.as_path(), source));
        let description = manifest::resolve_selected(selected, actual).unwrap();
        assert_eq!(description.path, path);
        assert!(canonical::valid_digest(&description.sha256));
    }
    let (selected, source) = manifest::select(
        Some(Path::new("/missing/explicit")),
        Some(path.clone().into_os_string()),
        executable(false),
    )
    .unwrap();
    assert_eq!(source, "explicit");
    assert!(
        manifest::resolve_selected(selected, source).is_err(),
        "invalid explicit manifest fell back to environment"
    );
}
