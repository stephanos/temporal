use super::*;
use crate::campaign::prepare::process::Temporary;
use std::{fs, os::unix::fs::PermissionsExt as _, path::PathBuf};

struct Cached {
    root: Temporary,
    path: PathBuf,
    module: Vec<u8>,
    invocation: Invocation,
    registration: DriverRegistration,
}

impl Cached {
    fn new() -> Self {
        let root = Temporary::new(&std::env::temp_dir()).unwrap();
        let invocation = Invocation {
            schema: "gomad.intercepted-invocation/v1".into(),
            wrapper_run_id: "1".repeat(64),
            id: "2".repeat(64),
            original_argv: ["go", "run", "./main"]
                .map(|value| argument(OsStr::new(value)))
                .to_vec(),
            driver_argv: ["/pinned/go", "run", "./main"]
                .map(|value| argument(OsStr::new(value)))
                .to_vec(),
            compiler_sha256: "3".repeat(64),
            installation_sha256: "4".repeat(64),
            recipe_sha256: "5".repeat(64),
            recipe_file: "driver-recipe.json".into(),
        };
        let module = b"already captured module snapshot".to_vec();
        let digest = canonical::digest(&module);
        let cache = root.0.join("cache");
        let path = cache
            .join(&digest[..2])
            .join(format!("{digest}-d"))
            .join("main");
        fs::create_dir_all(path.parent().unwrap()).unwrap();
        for directory in [&cache, &cache.join(&digest[..2]), path.parent().unwrap()] {
            fs::set_permissions(directory, fs::Permissions::from_mode(0o755)).unwrap();
        }
        fs::write(&path, &module).unwrap();
        fs::set_permissions(&path, fs::Permissions::from_mode(0o755)).unwrap();
        let private = root.0.join("go-build");
        fs::create_dir(&private).unwrap();
        fs::set_permissions(&private, fs::Permissions::from_mode(0o700)).unwrap();
        let registration = DriverRegistration {
            schema: "gomad.driver-registration/v1".into(),
            process_id: u32::try_from(rustix::process::getppid().unwrap().as_raw_pid()).unwrap(),
            invocation_sha256: canonical::digest(&canonical::encode(&invocation).unwrap()),
            compiler_sha256: invocation.compiler_sha256.clone(),
            driver_argv_sha256: canonical::digest(
                &canonical::encode(&invocation.driver_argv).unwrap(),
            ),
            module_root: private,
            module_cache_root: Some(cache),
        };
        Self {
            root,
            path,
            module,
            invocation,
            registration,
        }
    }

    fn authenticate(&self, bytes: &[u8]) -> Result<(), String> {
        let path = self.root.0.join("driver-registration.json");
        fs::write(&path, canonical::encode(&self.registration).unwrap()).unwrap();
        fs::set_permissions(&path, fs::Permissions::from_mode(0o600)).unwrap();
        authenticate_driver(
            &self.root.0.join("invocation.json"),
            &canonical::digest(&canonical::encode(&self.invocation).unwrap()),
            &self.invocation,
            &self.path,
            bytes,
        )
    }
}

#[test]
fn cached_run_module_is_admitted_under_its_registered_content_address() {
    let fixture = Cached::new();
    assert_eq!(fixture.authenticate(&fixture.module), Ok(()));
}

#[test]
fn cached_module_uses_the_captured_bytes_not_a_reopened_path() {
    let fixture = Cached::new();
    let mut changed = fixture.module.clone();
    changed[0] ^= 1;
    assert!(
        fixture
            .authenticate(&changed)
            .unwrap_err()
            .contains("cache address")
    );
    fs::write(&fixture.path, [fixture.module.as_slice(), b"x"].concat()).unwrap();
    assert!(
        fixture
            .authenticate(&fixture.module)
            .unwrap_err()
            .contains("changed after capture")
    );
}

#[test]
fn cached_module_requires_registered_run_and_preserves_global_directory_flags() {
    for command in ["test", "build", "run"] {
        for directory_flag in [false, true] {
            let mut fixture = Cached::new();
            let words = if directory_flag {
                vec!["/pinned/go", "-C", "/workspace", command, "./main"]
            } else {
                vec!["/pinned/go", "-C=/workspace", command, "./main"]
            };
            fixture.invocation.driver_argv = words
                .iter()
                .map(|word| argument(OsStr::new(word)))
                .collect();
            fixture.registration.invocation_sha256 =
                canonical::digest(&canonical::encode(&fixture.invocation).unwrap());
            fixture.registration.driver_argv_sha256 =
                canonical::digest(&canonical::encode(&fixture.invocation.driver_argv).unwrap());
            let result = fixture.authenticate(&fixture.module);
            if command == "run" {
                assert_eq!(result, Ok(()));
            } else {
                assert!(result.unwrap_err().contains("requires registered go run"));
            }
        }
    }
}

#[test]
fn cached_module_preserves_original_driver_identity_checks() {
    for field in ["process", "invocation", "compiler", "argv"] {
        let mut fixture = Cached::new();
        match field {
            "process" => fixture.registration.process_id = 0,
            "invocation" => fixture.registration.invocation_sha256 = "6".repeat(64),
            "compiler" => fixture.registration.compiler_sha256 = "7".repeat(64),
            "argv" => fixture.registration.driver_argv_sha256 = "8".repeat(64),
            _ => unreachable!(),
        }
        assert_eq!(
            fixture.authenticate(&fixture.module).unwrap_err(),
            "module is not associated with the registered live pinned Go driver"
        );
    }
}

#[test]
fn cached_module_rejects_wrong_root_shard_address_and_layout() {
    for field in ["root", "shard", "address", "layout"] {
        let mut fixture = Cached::new();
        let cache = fixture
            .registration
            .module_cache_root
            .as_ref()
            .unwrap()
            .clone();
        let digest = canonical::digest(&fixture.module);
        if field == "root" {
            fixture.registration.module_cache_root = Some(fixture.root.0.join("other"));
        } else {
            let (shard, address) = match field {
                "shard" => (
                    if &digest[..2] == "00" { "01" } else { "00" }.to_owned(),
                    format!("{digest}-d"),
                ),
                "address" => (digest[..2].to_owned(), format!("{digest}-a")),
                _ => (format!("extra/{}", &digest[..2]), format!("{digest}-d")),
            };
            fixture.path = cache.join(shard).join(address).join("main");
            fs::create_dir_all(fixture.path.parent().unwrap()).unwrap();
            fs::write(&fixture.path, &fixture.module).unwrap();
        }
        assert!(fixture.authenticate(&fixture.module).is_err(), "{field}");
    }
}

#[test]
fn cached_module_requires_one_payload_and_refuses_removed_files() {
    let fixture = Cached::new();
    let extra = fixture.path.parent().unwrap().join("extra");
    fs::write(&extra, b"extra").unwrap();
    assert!(
        fixture
            .authenticate(&fixture.module)
            .unwrap_err()
            .contains("contain one file")
    );
    fs::remove_file(extra).unwrap();
    fs::remove_file(&fixture.path).unwrap();
    assert!(fixture.authenticate(&fixture.module).is_err());
}

#[test]
fn cached_module_refuses_symlinks_and_writable_cache_components() {
    use std::os::unix::fs::symlink;
    for component in ["root", "shard", "address", "file"] {
        for link in [false, true] {
            let fixture = Cached::new();
            let cache = fixture.registration.module_cache_root.as_ref().unwrap();
            let selected = match component {
                "root" => cache.clone(),
                "shard" => fixture
                    .path
                    .parent()
                    .unwrap()
                    .parent()
                    .unwrap()
                    .to_path_buf(),
                "address" => fixture.path.parent().unwrap().to_path_buf(),
                _ => fixture.path.clone(),
            };
            if link {
                let target = selected.with_file_name(format!(
                    "{}-real",
                    selected.file_name().unwrap().to_string_lossy()
                ));
                fs::rename(&selected, &target).unwrap();
                symlink(target, selected).unwrap();
            } else {
                fs::set_permissions(selected, fs::Permissions::from_mode(0o777)).unwrap();
            }
            assert!(
                fixture.authenticate(&fixture.module).is_err(),
                "{component} link={link}"
            );
        }
    }
}

#[test]
fn legacy_registration_keeps_its_encoding_and_private_module_admission() {
    let mut fixture = Cached::new();
    fixture.registration.module_cache_root = None;
    let legacy = serde_json::json!({
        "schema":fixture.registration.schema,
        "process_id":fixture.registration.process_id,
        "invocation_sha256":fixture.registration.invocation_sha256,
        "compiler_sha256":fixture.registration.compiler_sha256,
        "driver_argv_sha256":fixture.registration.driver_argv_sha256,
        "module_root":fixture.registration.module_root,
    });
    let bytes = canonical::encode(&legacy).unwrap();
    let decoded: DriverRegistration = serde_json::from_slice(&bytes).unwrap();
    assert!(decoded.module_cache_root.is_none());
    assert_eq!(canonical::encode(&decoded).unwrap(), bytes);
    fixture.path = fixture.registration.module_root.join("module.wasm");
    fs::write(&fixture.path, &fixture.module).unwrap();
    assert_eq!(fixture.authenticate(&fixture.module), Ok(()));
}
