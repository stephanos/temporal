use super::*;
use std::{fs, os::unix::fs::PermissionsExt as _};

struct Fixture {
    root: PathBuf,
    driver: Value,
    receipt: Value,
    config: FileBinding,
    executable: FileBinding,
    installation: PathBuf,
}
fn write(path: &Path, bytes: &[u8]) -> String {
    fs::create_dir_all(path.parent().unwrap()).unwrap();
    fs::write(path, bytes).unwrap();
    fs::set_permissions(path, fs::Permissions::from_mode(0o600)).unwrap();
    canonical::digest(bytes)
}
fn prepared(root: &Path, provenance: &Value) -> PathBuf {
    use crate::evidence::{
        identity::Identity,
        record::{Limits, Outcome, Record},
        store::{Publication, Store},
    };
    let identity = Identity {
        target: canonical::digest(b"synthetic module"),
        toolchain: canonical::digest(b"synthetic build"),
        platform: canonical::digest(b"synthetic platform"),
        implementation: canonical::digest(b"synthetic implementation"),
        configuration: canonical::digest(b"synthetic configuration"),
    };
    let record = Record {
        schema: "gomad.execution-record/v2".into(),
        execution_identity: identity.digest().unwrap(),
        identity,
        profile: "gomad.wasi-cooperative/v1".into(),
        seed: 0,
        arguments: vec![],
        environment: BTreeMap::new(),
        limits: Limits {
            fuel: 1,
            memory_bytes: 65536,
            wall_millis: 1,
            output_bytes: 1,
            evidence_bytes: 1024,
        },
        virtual_time_nanos: 0,
        outcome: Outcome {
            class: "prepared".into(),
            termination: "prepared".into(),
            exit_code: None,
            reason: String::new(),
        },
        streams: BTreeMap::new(),
    };
    let payloads = BTreeMap::from([
        ("record.json".into(), record.bytes().unwrap()),
        (
            "target/provenance.json".into(),
            canonical::encode(provenance).unwrap(),
        ),
    ]);
    Store {
        root: root.join("prepared-store"),
        maximum_bytes: 1 << 20,
        target_pool: None,
        #[cfg(feature = "determinism-fixture")]
        fault: None,
    }
    .publish(Publication { record, payloads })
    .unwrap()
    .path
}
impl Fixture {
    #[expect(
        clippy::too_many_lines,
        reason = "The bounded synthetic fixture binds one complete installation and source inventory without running a guest"
    )]
    fn new() -> Self {
        let root = std::env::temp_dir().join(format!(
            "gomad-inspection-driver-{}",
            crate::campaign::origin::identity().unwrap()
        ));
        fs::create_dir(&root).unwrap();
        fs::set_permissions(&root, fs::Permissions::from_mode(0o700)).unwrap();
        let source_root = root.join("source");
        let source = source_root.join("schema/sqlite/schema.sql");
        let source_sha = write(&source, b"synthetic schema");
        let manifest = root.join("source.sha256");
        let manifest_sha = write(
            &manifest,
            format!("{source_sha}  {}\n", source.display()).as_bytes(),
        );
        let executable = FileBinding {
            path: root.join("gomad"),
            sha256: write(&root.join("gomad"), b"synthetic executable; never run"),
        };
        let compiler_root = root.join("compiler");
        let compiler = compiler_root.join("bin/go");
        let runtime = root.join("runtime");
        let cache = root.join("cache");
        let generated = cache.join("runtime-overlay").join("c".repeat(64));
        let mut inputs = BTreeMap::new();
        for (path, bytes) in [
            (compiler.clone(), b"synthetic Go".as_slice()),
            (compiler_root.join("VERSION"), b"go1.27.1".as_slice()),
            (
                compiler_root.join("src/runtime/proc.go"),
                b"original proc".as_slice(),
            ),
            (
                compiler_root.join("src/runtime/unused.go"),
                b"unused frozen compiler input".as_slice(),
            ),
            (
                runtime.join("gomad_semantics.go.txt"),
                b"runtime semantics".as_slice(),
            ),
            (
                runtime.join("gomad_choicewire_generated.go.txt"),
                b"runtime choice".as_slice(),
            ),
            (generated.join("proc.go"), b"generated proc".as_slice()),
        ] {
            inputs.insert(path.clone(), write(&path, bytes));
        }
        let descriptor = json!({"Replace":{compiler_root.join("src/runtime/proc.go").to_str().unwrap(): generated.join("proc.go")}});
        let descriptor = canonical::encode(&descriptor).unwrap();
        inputs.insert(
            generated.join("overlay.json"),
            write(&generated.join("overlay.json"), &descriptor),
        );
        let configured =
            crate::install::config::build("gomad.wasi-cooperative/v1", None, None).unwrap();
        let installation = crate::install::manifest::Installation {
            schema: crate::install::manifest::SCHEMA.into(),
            compiler_path: compiler.clone(),
            compiler_sha256: inputs[&compiler].clone(),
            go_version: "go1.27.1".into(),
            runtime_root: runtime.clone(),
            overlay_module_root: None,
            cache_root: cache.clone(),
            profile: "gomad.wasi-cooperative/v1".into(),
            fuel: 1 << 50,
            memory_bytes: 4 << 30,
            helper_sha256: gomad_wasmhost::linked::IMPLEMENTATION_SHA256.into(),
            host_sha256: env!("GOMAD_HOST_IMPLEMENTATION_SHA256").into(),
            runtime_implementation_sha256: Some(env!("GOMAD_RUNTIME_IMPLEMENTATION_SHA256").into()),
            model: configured.model,
            namespace: configured.namespace,
            storage: configured.storage,
            network: configured.network,
            build_tags: vec![],
            excluded_build_tags: vec![],
            admission: Some(crate::install::manifest::Admission {
                inputs_sha256: digest(&inputs).unwrap(),
                compiler_inputs: inputs.clone(),
                configuration_sha256: digest(&configured.original).unwrap(),
                guest_configuration: configured.original,
                host_executable_sha256: executable.sha256.clone(),
                explicit_helper: None,
                environment: configured.environment,
                arguments: configured.arguments,
                compile_cache_root: None,
            }),
        };
        crate::fork_order::make_executable(&compiler, 0o755);
        fs::set_permissions(&source, fs::Permissions::from_mode(0o644)).unwrap();
        let installation_path = root.join("installation.json");
        let installation_sha = write(
            &installation_path,
            &canonical::encode(&installation).unwrap(),
        );
        let mut files = inputs;
        files.insert(source, source_sha);
        files.insert(manifest.clone(), manifest_sha.clone());
        files.insert(executable.path.clone(), executable.sha256.clone());
        files.insert(installation_path.clone(), installation_sha);
        let config = FileBinding {
            path: root.join("driver.json"),
            sha256: "d".repeat(64),
        };
        let alias_root = root.join("private/compiler");
        let alias_overlay = root
            .join("private-cache/runtime-overlay")
            .join("c".repeat(64))
            .join("overlay.json");
        let descriptor = canonical::encode(&json!({"Replace":{
            alias_root.join("src/runtime/proc.go").to_str().unwrap(): alias_overlay.parent().unwrap().join("proc.go")
        }})).unwrap();
        let mut receipt = json!({"private_observation_config":{"path":config.path,"sha256":config.sha256},
            "private_inputs_detached":root.join("private-unavailable"), "compiler_overlay":{
                "path":alias_overlay,"sha256":canonical::digest(&descriptor),
                "bytes_base64":crate::campaign::exploration::base64(&descriptor)}});
        let original_inputs = &installation.admission.as_ref().unwrap().compiler_inputs;
        let mut observed = BTreeMap::new();
        for (member, sha) in original_inputs {
            if member.file_name().is_some_and(|name| name == "unused.go") {
                continue;
            }
            let alias = if let Ok(relative) = member.strip_prefix(&compiler_root) {
                alias_root.join(relative)
            } else if let Ok(relative) = member.strip_prefix(&runtime) {
                root.join("private/runtime").join(relative)
            } else if let Ok(relative) = member.strip_prefix(&generated) {
                alias_overlay.parent().unwrap().join(relative)
            } else {
                member.clone()
            };
            observed.insert(
                alias,
                if member == &generated.join("overlay.json") {
                    canonical::digest(&descriptor)
                } else {
                    sha.clone()
                },
            );
        }
        let artifact = prepared(
            &root,
            &json!({"inputs":{"compiler":observed,"compiler_path":alias_root.join("bin/go"),"runtime":"c".repeat(64)}}),
        );
        receipt["preparation"] = json!({"artifact":artifact});
        let driver = json!({"schema":"gomad.rust-qualification-driver/v1","features":[],
            "binary":executable.path,"binary_sha256":executable.sha256,
            "identity":{"host":env!("GOMAD_HOST_IMPLEMENTATION_SHA256"),"helper":gomad_wasmhost::linked::IMPLEMENTATION_SHA256},
            "repo":source_root,"source_manifest":manifest,"source_sha256":manifest_sha,
            "primary":{"compiler":compiler,"runtime_root":runtime,"cache":cache},
            "installations":{"onebox-cooperative":installation_path},"files":files});
        Self {
            root,
            driver,
            receipt,
            config,
            executable,
            installation: installation_path,
        }
    }
    fn candidate(&self) -> Result<Candidate, String> {
        candidate(&self.driver, &self.receipt, &self.config, &self.executable)
    }
}
impl Drop for Fixture {
    fn drop(&mut self) {
        fs::remove_dir_all(&self.root).unwrap();
    }
}

#[test]
fn standalone_candidate_admits_frozen_inputs_without_any_result() {
    let fixture = Fixture::new();
    let value = fixture.candidate().unwrap();
    assert_eq!(value.executable, fixture.executable);
    assert_eq!(
        value.integrated_source_sha256,
        fixture.driver["source_sha256"]
    );
    assert_eq!(
        value.inventory.compiler_path,
        PathBuf::from(fixture.driver["primary"]["compiler"].as_str().unwrap())
    );
    assert!(
        !value
            .inventory
            .raw
            .keys()
            .any(|path| path.file_name().is_some_and(|name| name == "unused.go"))
    );
    assert!(!fixture.root.join("result.json").exists());
}

#[test]
fn standalone_candidate_refuses_foreign_executable_source_installation_and_driver() {
    for mutation in 0..8 {
        let mut fixture = Fixture::new();
        match mutation {
            0 => fixture.executable.sha256 = "1".repeat(64),
            1 => fixture.executable.path = fixture.root.join("other-gomad"),
            2 => fixture.driver["identity"]["host"] = json!("2".repeat(64)),
            3 => fixture.driver["features"] = json!(["diagnostic-profile"]),
            4 => fixture.driver["source_sha256"] = json!("3".repeat(64)),
            5 => {
                fs::write(&fixture.installation, b"{}").unwrap();
            }
            6 => fixture.receipt["private_observation_config"]["sha256"] = json!("4".repeat(64)),
            _ => {
                let compiler = fixture.driver["primary"]["compiler"]
                    .as_str()
                    .unwrap()
                    .to_owned();
                fixture.driver["files"]
                    .as_object_mut()
                    .unwrap()
                    .remove(&compiler);
            }
        }
        assert!(fixture.candidate().is_err(), "mutation {mutation}");
    }
}

#[test]
fn standalone_entrypoint_reaches_shared_inspection_checks_without_result_dependency() {
    let mut fixture = Fixture::new();
    let raw = canonical::encode(&fixture.driver).unwrap();
    fixture.config.sha256 = write(&fixture.config.path, &raw);
    fixture.receipt["private_observation_config"]["sha256"] = json!(fixture.config.sha256);
    let receipt = FileBinding {
        path: fixture.root.join("inspection-receipt.json"),
        sha256: write(
            &fixture.root.join("inspection-receipt.json"),
            &canonical::encode(&fixture.receipt).unwrap(),
        ),
    };
    let arguments = vec![
        "validate-inspection-driver".into(),
        receipt.path.to_string_lossy().into_owned(),
        receipt.sha256,
        fixture.config.path.to_string_lossy().into_owned(),
        fixture.config.sha256.clone(),
        fixture.executable.path.to_string_lossy().into_owned(),
        fixture.executable.sha256.clone(),
    ];
    let error = super::super::run(&arguments, &mut Vec::new()).unwrap_err();
    assert!(
        error.contains("executed original onebox subgate"),
        "{error}"
    );
    assert!(!fixture.root.join("result.json").exists());
}

#[test]
fn actual_alias_closure_cannot_add_change_or_omit_frozen_compiler_members() {
    let fixture = Fixture::new();
    let installation: crate::install::manifest::Installation =
        spec::strict(&crate::install::manifest::read(&fixture.installation, 1 << 20).unwrap())
            .unwrap();
    let original = &installation.admission.as_ref().unwrap().compiler_inputs;
    let artifact = crate::evidence::store::Opened::open(Path::new(
        fixture.receipt["preparation"]["artifact"].as_str().unwrap(),
    ))
    .unwrap();
    let provenance: Value =
        spec::strict(&artifact.payload("target/provenance.json", 1 << 20).unwrap()).unwrap();
    let overlay: inventory::Overlay =
        serde_json::from_value(fixture.receipt["compiler_overlay"].clone()).unwrap();
    let observed = inventory::from_provenance(&provenance, Some(overlay)).unwrap();
    let alias = fixture.root.join("private/compiler");
    for mutation in 0..3 {
        let mut changed = observed.clone();
        match mutation {
            0 => {
                changed.raw.insert(alias.join("VERSION"), "f".repeat(64));
            }
            1 => {
                changed
                    .raw
                    .insert(alias.join("src/runtime/foreign.go"), "e".repeat(64));
            }
            _ => {
                changed.raw.remove(&changed.compiler_path);
            }
        }
        assert!(
            inventory::from_frozen_inputs(
                original.clone(),
                installation.compiler_path.clone(),
                &changed,
                &alias
            )
            .is_err(),
            "mutation {mutation}"
        );
    }
}
