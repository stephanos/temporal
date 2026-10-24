use serde_json::{Value, json};
use std::{
    collections::BTreeMap,
    ffi::{OsStr, OsString},
    fs,
    os::unix::ffi::{OsStrExt, OsStringExt},
    path::PathBuf,
    process::Command,
    sync::atomic::{AtomicU64, Ordering},
};
use xtask::{
    compiler::{Sidecar, prepare},
    support::{SCHEMA, canonical},
};

type TestResult<T = ()> = Result<T, Box<dyn std::error::Error>>;

type Environment = BTreeMap<OsString, OsString>;
struct Fixture {
    root: PathBuf,
    args: Vec<OsString>,
    env: Environment,
}
impl Fixture {
    fn new(host: &str) -> TestResult<Self> {
        static NEXT: AtomicU64 = AtomicU64::new(0);
        let root = std::env::temp_dir().join(format!(
            "umpire compiler {} {}",
            std::process::id(),
            NEXT.fetch_add(1, Ordering::Relaxed)
        ));
        fs::create_dir_all(&root)?;
        let mut fixture = Self { root, args: Vec::new(), env: BTreeMap::new() };
        fixture.relocate(host);
        fixture.lock(&json!({"example 1.0.0":"e".repeat(64), "example 2.0.0":"f".repeat(64), "another 1.0.0":"0".repeat(64)}))?;
        Ok(fixture)
    }
    fn relocate(&mut self, host: &str) {
        let home = self.root.join(host);
        let workspace = home.join("workspace");
        let target = home.join("target");
        self.args = vec![
            "rustc".into(),
            "--crate-name".into(),
            "example".into(),
            "--edition=2024".into(),
            "src/lib.rs".into(),
            "--crate-type=lib".into(),
            "--emit=dep-info,metadata,link".into(),
            "-Copt-level=3".into(),
            format!("-Cmetadata={host}").into(),
            "-C".into(),
            format!("extra-filename=-{host}").into(),
            "--out-dir".into(),
            target.join("deps").into_os_string(),
            "--target=wasm32-wasip1".into(),
            "--cfg".into(),
            "feature=\"example\"".into(),
        ];
        for (key, value) in [
            ("UMPIRE_GUEST_WORKSPACE", workspace.clone()),
            ("UMPIRE_GUEST_TARGET", target.clone()),
            ("UMPIRE_GUEST_SYSROOT", home.join("rust")),
            ("UMPIRE_GUEST_CARGO_HOME", home.join("cargo")),
            ("UMPIRE_GUEST_LOCK", self.root.join("lock.json")),
            ("CARGO_MANIFEST_DIR", workspace.join("crate")),
            ("OUT_DIR", target.join("build/out")),
        ] {
            self.env.insert(key.into(), value.into_os_string());
        }
        for (key, value) in [
            ("UMPIRE_GUEST_COMPILER", "rustc 1.99.1 pinned"),
            ("CARGO_PKG_NAME", "example"),
            ("CARGO_PKG_VERSION", "1.0.0"),
        ] {
            self.env.insert(key.into(), value.into());
        }
    }
    fn lock(&self, closures: &Value) -> TestResult {
        fs::write(
            self.root.join("lock.json"),
            canonical(&json!({"schema":SCHEMA,"closures":closures}))?,
        )?;
        Ok(())
    }
    fn record(&self) -> TestResult<Sidecar> {
        Ok(prepare(self.args.clone(), &self.env)?.sidecar.ok_or("expected sidecar")?.1)
    }
    fn dependency(&mut self, alias: &str, identity: &str, variants: &Value) -> TestResult<PathBuf> {
        let directory = PathBuf::from(&self.env[OsStr::new("UMPIRE_GUEST_TARGET")]).join("deps");
        fs::create_dir_all(&directory)?;
        let path = directory.join(format!("{alias}-host.umpire-id"));
        fs::write(
            &path,
            canonical(&json!({"schema":SCHEMA,"identity":identity,"host_variants":variants}))?,
        )?;
        let mut argument = format!("--extern={alias}=").into_bytes();
        argument.extend(directory.join(format!("lib{alias}-host.rlib")).as_os_str().as_bytes());
        self.args.push(OsString::from_vec(argument));
        Ok(path)
    }
    fn process(&self) -> Command {
        let mut command = Command::new(env!("CARGO_BIN_EXE_xtask"));
        command
            .arg("guest-rustc")
            .args(&self.args)
            .envs(&self.env)
            .env_remove("UMPIRE_GUEST_PREVIOUS_WRAPPER");
        command
    }
    fn child(&mut self) {
        self.args[0] = env!("CARGO_BIN_EXE_identity-compiler-fixture").into();
    }
}
impl Drop for Fixture {
    fn drop(&mut self) {
        let result = fs::remove_dir_all(&self.root);
        assert!(result.is_ok(), "fixture cleanup failed: {result:?}");
    }
}

#[test]
fn probes_pass_through_without_guest_environment() -> TestResult {
    for args in [
        vec!["rustc", "-V"],
        vec!["rustc", "--crate-name", "___", "--print=file-names", "--target=wasm32-wasip1", "-"],
    ] {
        let args: Vec<OsString> = args.into_iter().map(Into::into).collect();
        let actual = prepare(args.clone(), &BTreeMap::new())?;
        assert_eq!(actual.command, args);
        assert!(actual.sidecar.is_none());
    }
    Ok(())
}

#[test]
fn portable_identity_replaces_guest_metadata_and_remaps_generated_paths() -> TestResult {
    let mut fixture = Fixture::new("linux")?;
    let first = fixture.record()?;
    fixture.relocate("macos");
    fixture.args.extend([
        "-Cincremental=/arbitrary/host/cache".into(),
        "-C".into(),
        "metadata=second".into(),
    ]);
    let actual = prepare(fixture.args.clone(), &fixture.env)?;
    assert_eq!(first, actual.sidecar.ok_or("expected sidecar")?.1);
    assert!(actual.command.contains(&OsString::from(format!("metadata={}", first.identity))));
    assert!(!actual.command.iter().any(|a| a.as_bytes().starts_with(b"-Cmetadata=")));
    assert!(
        actual
            .command
            .iter()
            .any(|a| a.as_bytes().ends_with(format!("=/generated/{}", first.identity).as_bytes()))
    );
    assert_eq!(fixture.record()?, first);
    Ok(())
}

#[test]
fn package_features_profile_compiler_and_closure_remain_distinct() -> TestResult {
    let mut fixture = Fixture::new("linux")?;
    let baseline = fixture.record()?;
    for (key, value) in [
        ("CARGO_PKG_NAME", "another"),
        ("CARGO_PKG_VERSION", "2.0.0"),
        ("UMPIRE_GUEST_COMPILER", "rustc another"),
    ] {
        let old = fixture.env.insert(key.into(), value.into()).ok_or("existing fixture env")?;
        assert_ne!(fixture.record()?, baseline);
        fixture.env.insert(key.into(), old);
    }
    for flag in ["--cfg=feature=\"other\"", "-Coverflow-checks=on", "-Copt-level=1"] {
        fixture.args.push(flag.into());
        assert_ne!(fixture.record()?, baseline);
        fixture.args.pop();
    }
    fixture.lock(&json!({"example 1.0.0":"1".repeat(64)}))?;
    assert_ne!(fixture.record()?, baseline);
    Ok(())
}

#[test]
fn dependency_identity_and_transitive_variants_propagate() -> TestResult {
    let mut fixture = Fixture::new("linux")?;
    let baseline = fixture.record()?;
    let path = fixture.dependency(
        "dependency",
        &"a".repeat(64),
        &json!({"b".repeat(64):"c".repeat(64)}),
    )?;
    let first = fixture.record()?;
    assert_ne!(first, baseline);
    assert_eq!(first.host_variants.get(&"b".repeat(64)), Some(&"c".repeat(64)));
    fs::write(path, canonical(&json!({"schema":SCHEMA,"identity":"d".repeat(64)}))?)?;
    assert_ne!(fixture.record()?, first);
    Ok(())
}

#[test]
fn missing_ambiguous_and_malformed_lock_maps_refuse() -> TestResult {
    let fixture = Fixture::new("linux")?;
    for closures in [json!({}), json!({"example 1.0.0":null}), json!({"example 1.0.0":"invalid"})] {
        fixture.lock(&closures)?;
        assert!(prepare(fixture.args.clone(), &fixture.env).is_err());
    }
    for bytes in [b"not json".as_slice(), br#"{"schema":1,"closures":{}}"#] {
        fs::write(fixture.root.join("lock.json"), bytes)?;
        assert!(prepare(fixture.args.clone(), &fixture.env).is_err());
    }
    Ok(())
}

#[test]
fn build_std_requires_wasm_marker_and_trusted_source_or_rust_src_coordinate() -> TestResult {
    let mut fixture = Fixture::new("linux")?;
    fixture.lock(&json!({}))?;
    let library = PathBuf::from(&fixture.env[OsStr::new("UMPIRE_GUEST_SYSROOT")])
        .join("lib/rustlib/src/rust/library");
    fs::create_dir_all(&library)?;
    fs::write(
        library.join("Cargo.lock"),
        "version=4\n[[package]]\nname='example'\nversion='1.0.0'\n",
    )?;
    let registry = PathBuf::from(&fixture.env[OsStr::new("UMPIRE_GUEST_CARGO_HOME")])
        .join("registry/src/index/example");
    fixture.args.push("-Zforce-unstable-if-unmarked".into());
    for manifest in [library.join("std"), registry.clone()] {
        fixture.env.insert("CARGO_MANIFEST_DIR".into(), manifest.into());
        assert!(prepare(fixture.args.clone(), &fixture.env).is_ok());
    }
    fixture.args.pop();
    assert!(prepare(fixture.args.clone(), &fixture.env).is_err());
    fixture.args.push("-Zforce-unstable-if-unmarked".into());
    fixture.env.insert("CARGO_PKG_VERSION".into(), "2.0.0".into());
    assert!(prepare(fixture.args.clone(), &fixture.env).is_err());
    fixture
        .env
        .insert("CARGO_MANIFEST_DIR".into(), fixture.root.join("linux/workspace/crate").into());
    assert!(prepare(fixture.args.clone(), &fixture.env).is_err());
    fixture.env.insert("CARGO_MANIFEST_DIR".into(), registry.into());
    fs::write(library.join("Cargo.lock"), "[broken")?;
    assert!(prepare(fixture.args.clone(), &fixture.env).is_err());
    Ok(())
}

#[test]
fn invalid_or_missing_dependency_sidecars_and_macro_collisions_refuse() -> TestResult {
    let mut fixture = Fixture::new("linux")?;
    let path = fixture.dependency("first", &"a".repeat(64), &json!({}))?;
    for data in [
        json!({}),
        json!({"schema":1,"identity":"a".repeat(64)}),
        json!({"schema":SCHEMA,"identity":"bad"}),
        json!({"schema":SCHEMA,"identity":"a".repeat(64),"host_variants":{"bad":"bad"}}),
    ] {
        fs::write(&path, canonical(&data)?)?;
        assert!(prepare(fixture.args.clone(), &fixture.env).is_err());
    }
    fs::write(&path, b"not json")?;
    assert!(prepare(fixture.args.clone(), &fixture.env).is_err());
    fs::remove_file(&path)?;
    assert!(prepare(fixture.args.clone(), &fixture.env).is_err());
    fs::write(
        path,
        canonical(
            &json!({"schema":SCHEMA,"identity":"a".repeat(64),"host_variants":{"b".repeat(64):"c".repeat(64)}}),
        )?,
    )?;
    fixture.dependency("other", &"a".repeat(64), &json!({"b".repeat(64):"d".repeat(64)}))?;
    assert!(
        prepare(fixture.args.clone(), &fixture.env)
            .err()
            .ok_or("expected refusal")?
            .to_string()
            .contains("conflicting native macro")
    );
    Ok(())
}

#[test]
fn native_macro_identity_ignores_host_configuration_but_retains_argv_and_collisions() -> TestResult
{
    let mut fixture = Fixture::new("linux")?;
    let make_host = |args: &mut Vec<OsString>| -> TestResult {
        args.retain(|a| !a.as_bytes().starts_with(b"--target="));
        *args
            .iter_mut()
            .find(|a| a.as_bytes().starts_with(b"--crate-type="))
            .ok_or("crate type fixture flag")? = "--crate-type=proc-macro".into();
        Ok(())
    };
    make_host(&mut fixture.args)?;
    let first = prepare(fixture.args.clone(), &fixture.env)?;
    assert_eq!(first.command, fixture.args);
    let first = first.sidecar.ok_or("expected sidecar")?.1;
    fixture.relocate("macos");
    make_host(&mut fixture.args)?;
    fixture.args.extend([
        "--cfg=native_macos".into(),
        "-Clinker=/host/linker".into(),
        "-Zhost-only".into(),
    ]);
    let second = prepare(fixture.args.clone(), &fixture.env)?;
    assert_eq!(second.command, fixture.args);
    let second = second.sidecar.ok_or("expected sidecar")?.1;
    assert_eq!(first.identity, second.identity);
    assert_ne!(first.host_variants, second.host_variants);
    Ok(())
}

#[test]
fn os_bytes_and_lexical_root_boundaries_are_preserved() -> TestResult {
    let mut fixture = Fixture::new("linux")?;
    fixture.args[0] = OsString::from_vec(b"-Cmetadata=compiler-\xfe".to_vec());
    assert_eq!(prepare(fixture.args.clone(), &fixture.env)?.command[0], fixture.args[0]);
    fixture.args[4] = OsString::from_vec(b"src/nonutf-\xff.rs".to_vec());
    fixture.args.push(OsString::from_vec(b"--cfg=raw_\xfe".to_vec()));
    let baseline = fixture.record()?;
    let mut raw_root = fixture.root.as_os_str().as_bytes().to_vec();
    raw_root.extend(b"/raw-root-\xff");
    let raw_root = PathBuf::from(OsString::from_vec(raw_root));
    fixture.env.insert("UMPIRE_GUEST_WORKSPACE".into(), raw_root.clone().into());
    fixture.env.insert("CARGO_MANIFEST_DIR".into(), raw_root.join("crate").into());
    assert_eq!(fixture.record()?, baseline);
    fixture.args[4] = OsString::from_vec(b"src/nonutf-\xfe.rs".to_vec());
    assert_ne!(fixture.record()?, baseline);
    fixture.args[4] = "src/./nested/../lib.rs".into();
    let lexical = fixture.record()?;
    fixture.args[4] = "src/lib.rs".into();
    assert_eq!(fixture.record()?, lexical);
    let workspace = PathBuf::from(&fixture.env[OsStr::new("UMPIRE_GUEST_WORKSPACE")]);
    let mut escaped = workspace.into_os_string();
    escaped.push("-sibling/crate");
    fixture.env.insert("CARGO_MANIFEST_DIR".into(), escaped);
    assert!(prepare(fixture.args.clone(), &fixture.env).is_err());
    Ok(())
}

#[test]
fn missing_required_inputs_and_option_values_refuse() -> TestResult {
    assert!(prepare(Vec::new(), &BTreeMap::new()).is_err());
    let fixture = Fixture::new("linux")?;
    for flag in ["--crate-name", "--target", "--out-dir", "--extern", "--cfg", "-C", "-Z"] {
        let mut args = fixture.args.clone();
        args.push(flag.into());
        assert!(prepare(args, &fixture.env).is_err());
    }
    for key in [
        "CARGO_MANIFEST_DIR",
        "UMPIRE_GUEST_WORKSPACE",
        "UMPIRE_GUEST_COMPILER",
        "CARGO_PKG_VERSION",
        "UMPIRE_GUEST_LOCK",
    ] {
        let mut env = fixture.env.clone();
        env.remove(OsStr::new(key));
        assert!(prepare(fixture.args.clone(), &env).is_err());
    }
    let mut args = fixture.args.clone();
    args.push("--extern=no_path".into());
    assert!(prepare(args, &fixture.env).is_err());
    Ok(())
}

#[test]
fn sidecar_precedes_child_and_cleanup_preserves_exit_launch_and_signal_outcomes() -> TestResult {
    let mut fixture = Fixture::new("linux")?;
    fixture.child();
    let path = prepare(fixture.args.clone(), &fixture.env)?.sidecar.ok_or("expected sidecar")?.0;
    for (mode, expected) in [("success", 0), ("failure", 23), ("signal", 143), ("launch", 1)] {
        let mut command = fixture.process();
        command.env("FIXTURE_SIDECAR", &path);
        match mode {
            "failure" => {
                command.env("FIXTURE_EXIT", "23");
            }
            "signal" => {
                command.env("FIXTURE_SIGNAL", "yes");
            }
            "launch" => {
                fixture.args[0] = fixture.root.join("absent").into();
                command = fixture.process();
            }
            _ => (),
        }
        let output = command.output()?;
        assert_eq!(
            output.status.code(),
            Some(expected),
            "{mode}: {}",
            String::from_utf8_lossy(&output.stderr)
        );
        assert_eq!(path.exists(), mode == "success");
    }
    Ok(())
}

#[test]
fn previous_wrapper_and_loader_environment_keep_raw_os_bytes() -> TestResult {
    let mut fixture = Fixture::new("linux")?;
    fixture.child();
    fixture.args.push(OsString::from_vec(b"untouched-\xff".to_vec()));
    let marker = fixture.root.join("previous");
    let output = fixture.root.join("argv.json");
    let raw = OsString::from_vec(b"raw-\xfe".to_vec());
    let result = fixture
        .process()
        .env("UMPIRE_GUEST_PREVIOUS_WRAPPER", env!("CARGO_BIN_EXE_identity-compiler-fixture"))
        .env("FIXTURE_PREVIOUS", &marker)
        .env("FIXTURE_OUTPUT", &output)
        .env("FIXTURE_RAW", &raw)
        .env("DYLD_FALLBACK_LIBRARY_PATH", &raw)
        .output()?;
    assert!(result.status.success(), "{}", String::from_utf8_lossy(&result.stderr));
    assert_eq!(fs::read(marker)?, b"called");
    let observed: Value = serde_json::from_slice(&fs::read(output)?)?;
    assert_eq!(observed["raw"], json!(raw.as_bytes()));
    assert_eq!(observed["loader"], json!(raw.as_bytes()));
    assert!(
        observed["argv"]
            .as_array()
            .ok_or("expected argv array")?
            .contains(&json!(b"untouched-\xff".as_slice()))
    );
    Ok(())
}

#[test]
fn inherited_jobserver_descriptor_reaches_direct_compiler_child() -> TestResult {
    let fixture = Fixture::new("linux")?;
    let token = fixture.root.join("jobserver");
    fs::write(&token, b"jobserver")?;
    let result = Command::new("/bin/sh")
        .args([
            OsStr::new("-c"),
            OsStr::new("exec 9<\"$1\"; shift; exec \"$@\""),
            OsStr::new("fixture"),
            token.as_os_str(),
            OsStr::new(env!("CARGO_BIN_EXE_xtask")),
            OsStr::new("guest-rustc"),
            OsStr::new(env!("CARGO_BIN_EXE_identity-compiler-fixture")),
            OsStr::new("-V"),
        ])
        .env("FIXTURE_FD", "yes")
        .env("CARGO_MAKEFLAGS", "--jobserver-auth=9,9")
        .output()?;
    assert!(result.status.success(), "{}", String::from_utf8_lossy(&result.stderr));
    Ok(())
}

#[test]
fn actual_rustc_compiles_portable_native_macro_sidecars() -> TestResult {
    let mut fixture = Fixture::new("first")?;
    let mut records = Vec::new();
    for host in ["first", "second"] {
        fixture.relocate(host);
        fixture.args.retain(|a| !a.as_bytes().starts_with(b"--target="));
        *fixture
            .args
            .iter_mut()
            .find(|a| a.as_bytes().starts_with(b"--crate-type="))
            .ok_or("crate type fixture flag")? = "--crate-type=proc-macro".into();
        let manifest = PathBuf::from(&fixture.env[OsStr::new("CARGO_MANIFEST_DIR")]);
        fs::create_dir_all(manifest.join("src"))?;
        fs::write(
            manifest.join("src/lib.rs"),
            "extern crate proc_macro; #[proc_macro] pub fn pass(input: proc_macro::TokenStream) -> proc_macro::TokenStream { input }",
        )?;
        let (path, expected) =
            prepare(fixture.args.clone(), &fixture.env)?.sidecar.ok_or("expected sidecar")?;
        let output = fixture.process().current_dir(manifest).output()?;
        assert!(output.status.success(), "{}", String::from_utf8_lossy(&output.stderr));
        let actual: Sidecar = serde_json::from_slice(&fs::read(path)?)?;
        assert_eq!(actual, expected);
        records.push(actual);
    }
    assert_eq!(records[0].identity, records[1].identity);
    assert_ne!(records[0].host_variants, records[1].host_variants);
    Ok(())
}
