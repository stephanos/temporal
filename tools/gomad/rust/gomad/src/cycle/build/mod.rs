#![cfg_attr(
    not(test),
    allow(
        dead_code,
        reason = "Cycle preparation integration belongs to successor tasks"
    )
)]
use super::{
    custody::{Snapshot, digest, observe_command, read_file, strict_json},
    process::Runner,
};
use serde_json::{Value, json};
use sha2::{Digest, Sha256};
use std::{
    collections::BTreeMap,
    fs,
    io::Write,
    os::unix::fs::MetadataExt,
    path::{Path, PathBuf},
    sync::atomic::AtomicBool,
    time::Duration,
};

const SCHEMA: &str = "gomad.dry-run-build/v1";
const JSON_BOUND: u64 = 64 << 20;
const FILE_BOUND: u64 = 512 << 20;
const ROLE_NAMES: &[&str] = &[
    "release",
    "host-fixture",
    "ordinary-helper-test",
    "ordinary-host-test",
    "stress",
    "stress-release",
    "conformance-reference",
    "reference-transport",
    "runtime-harness",
    "interception-harness",
    "fn222-support",
    "counterfactual-fixture",
    "counterfactual-harness",
];

#[derive(Clone, Copy, Debug)]
pub(crate) struct Role(&'static str);
impl Role {
    pub fn parse(name: &str) -> Result<Self, String> {
        ROLE_NAMES
            .iter()
            .find(|value| **value == name)
            .map(|name| Self(name))
            .ok_or_else(|| "unknown role".into())
    }
    fn features(self) -> Vec<&'static str> {
        match self.0 {
            "host-fixture" => vec!["determinism-fixture"],
            "stress" => vec!["determinism-fixture", "helper-stress-jitter"],
            "stress-release" => vec!["helper-stress-jitter"],
            "conformance-reference" => vec!["conformance", "reference-helper"],
            "reference-transport" => vec!["reference-helper"],
            "counterfactual-fixture" | "counterfactual-harness" => {
                vec!["determinism-fixture", "helper-determinism-fixture"]
            }
            _ => vec![],
        }
    }
    fn helper_features(self) -> Vec<&'static str> {
        if matches!(self.0, "conformance-reference" | "reference-transport") {
            return self.features();
        }
        self.features()
            .iter()
            .filter_map(|v| v.strip_prefix("helper-"))
            .collect()
    }
    fn helper(self) -> bool {
        matches!(
            self.0,
            "conformance-reference" | "ordinary-helper-test" | "reference-transport"
        )
    }
    fn libtest(self) -> bool {
        matches!(
            self.0,
            "counterfactual-harness" | "ordinary-host-test" | "ordinary-helper-test"
        )
    }
    fn example(self) -> Option<&'static str> {
        match self.0 {
            "runtime-harness" => Some("runtime-qualification-replay"),
            "interception-harness" => Some("interception-streaming-replay"),
            "fn222-support" => Some("fn222-support"),
            _ => None,
        }
    }
    fn target(self) -> &'static str {
        self.example().unwrap_or(if self.helper() {
            if self.libtest() {
                "gomad_wasmhost"
            } else {
                "gomad-wasmhost"
            }
        } else {
            "gomad"
        })
    }
    fn binary_name(self) -> &'static str {
        self.example().unwrap_or(match self.0 {
            "ordinary-helper-test" => "gomad-wasmhost-libtest",
            "ordinary-host-test" => "gomad-ordinary-libtest",
            "counterfactual-harness" => "gomad-fixture-libtest",
            "conformance-reference" | "reference-transport" => "gomad-wasmhost",
            _ => "gomad",
        })
    }
}
pub(crate) fn cargo_command(cargo: &Path, repo: &Path, role: Role) -> Vec<String> {
    let mut args = vec![
        cargo.display().to_string(),
        if role.libtest() { "test" } else { "build" }.into(),
        "--release".into(),
        "--locked".into(),
        "--no-default-features".into(),
        "-p".into(),
        if role.helper() {
            "gomad-wasmhost"
        } else {
            "gomad"
        }
        .into(),
    ];
    if role.libtest() {
        args.extend(["--lib".into(), "--no-run".into()]);
    } else if let Some(example) = role.example() {
        args.extend(["--example".into(), example.into()]);
    } else {
        args.extend(["--bin".into(), role.target().into()]);
    }
    args.extend([
        "--message-format=json".into(),
        "--manifest-path".into(),
        repo.join("tools/gomad/rust/Cargo.toml")
            .display()
            .to_string(),
    ]);
    let features = if role.0 == "conformance-reference" {
        vec!["conformance"]
    } else {
        role.features()
    };
    if !features.is_empty() {
        args.extend(["--features".into(), features.join(",")]);
    }
    args
}
fn default_repo() -> Result<PathBuf, String> {
    Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("../../../..")
        .canonicalize()
        .map_err(|e| e.to_string())
}
fn require(condition: bool, reason: &str) -> Result<(), String> {
    if condition {
        Ok(())
    } else {
        Err(reason.into())
    }
}
fn text<'a>(value: &'a Value, key: &str) -> Result<&'a str, String> {
    value
        .get(key)
        .and_then(Value::as_str)
        .ok_or_else(|| format!("missing/string build field {key}"))
}
fn file_json(path: &Path) -> Result<Value, String> {
    strict_json(&read_file(path, JSON_BOUND)?)
}
fn frames(raw: &[u8]) -> Result<Vec<Value>, String> {
    require(
        raw.len() as u64 <= JSON_BOUND,
        "Cargo output exceeds original byte bound",
    )?;
    std::str::from_utf8(raw)
        .map_err(|e| e.to_string())?
        .lines()
        .filter(|line| !line.trim().is_empty())
        .map(|line| strict_json(line.as_bytes()))
        .collect()
}
fn env_pairs(value: &Value) -> Result<BTreeMap<String, String>, String> {
    if value["reason"] != "build-script-executed" {
        return Ok(BTreeMap::new());
    }
    let mut env = BTreeMap::new();
    for pair in value["env"]
        .as_array()
        .ok_or("invalid Cargo build environment")?
    {
        let pair = pair
            .as_array()
            .filter(|p| p.len() == 2)
            .ok_or("invalid Cargo environment pair")?;
        let name = pair[0].as_str().ok_or("invalid Cargo environment name")?;
        let value = pair[1].as_str().ok_or("invalid Cargo environment value")?;
        require(
            env.insert(name.into(), value.into()).is_none(),
            "duplicate Cargo environment binding",
        )?;
    }
    Ok(env)
}
fn package(row: &Value, repo: &Path, helper: bool) -> bool {
    let root = repo
        .join("tools/gomad/rust")
        .join(if helper { "wasmhost" } else { "gomad" });
    let prefix = format!("path+file://{}#", root.display());
    row["package_id"]
        .as_str()
        .and_then(|v| v.strip_prefix(&prefix))
        .is_some_and(|v| {
            if helper {
                v == "gomad-wasmhost@0.1.0"
            } else {
                v == "0.1.0" || v == "gomad@0.1.0"
            }
        })
}
fn selected<'a>(frames: &'a [Value], role: Role, repo: &Path) -> Vec<&'a Value> {
    let manifest = repo
        .join("tools/gomad/rust")
        .join(if role.helper() { "wasmhost" } else { "gomad" })
        .join("Cargo.toml");
    frames
        .iter()
        .filter(|row| {
            row["reason"] == "compiler-artifact"
                && row["executable"].as_str().is_some()
                && row["manifest_path"].as_str() == manifest.to_str()
                && row["target"]["name"] == role.target()
                && package(row, repo, role.helper())
                && row["target"]["kind"]
                    == json!([if role.libtest() {
                        "lib"
                    } else if role.example().is_some() {
                        "example"
                    } else {
                        "bin"
                    }])
                && row["target"]["crate_types"]
                    == json!([if role.libtest() { "lib" } else { "bin" }])
                && row["target"]["src_path"]
                    == json!(manifest.parent().unwrap_or(repo).join(
                        if let Some(example) = role.example() {
                            format!("examples/{example}.rs")
                        } else if role.libtest() {
                            "src/lib.rs".into()
                        } else {
                            "src/main.rs".into()
                        }
                    ))
        })
        .collect()
}
fn source_digest(snapshot: &Snapshot) -> String {
    let mut raw = String::new();
    for (name, sha) in &snapshot.files {
        raw.push_str(sha);
        raw.push_str("  ");
        raw.push_str(name);
        raw.push('\n');
    }
    digest(raw.as_bytes())
}
pub(crate) struct DryRun {
    config: Option<Value>,
    enabled: bool,
}
impl Default for DryRun {
    fn default() -> Self {
        Self {
            config: None,
            enabled: std::env::var("GOMAD_DRIVER_DRY_RUN").is_ok_and(|v| v == "1"),
        }
    }
}
impl DryRun {
    pub fn enable(&mut self, config: &Value) -> bool {
        self.config = (config.get("dry_run") == Some(&Value::Bool(true))).then(|| config.clone());
        self.enabled = self.config.is_some();
        self.enabled()
    }
    pub fn enabled(&self) -> bool {
        self.enabled
    }
    #[expect(
        clippy::too_many_lines,
        reason = "One transaction retains all parsed input pins until final source and byte revalidation"
    )]
    pub fn read_record(
        &self,
        path: &Path,
        frozen: Option<&Value>,
        expected_role: Option<&str>,
        runner: &mut Runner,
        cancelled: &AtomicBool,
        context: &str,
    ) -> Result<Value, String> {
        require(
            self.enabled(),
            "dry-run build record outside a dry-run configuration",
        )?;
        let record_bytes = read_file(path, JSON_BOUND)?;
        let record = strict_json(&record_bytes)?;
        let mut retained = BTreeMap::from([(path.to_owned(), digest(&record_bytes))]);
        let role = record
            .get("role")
            .and_then(Value::as_str)
            .and_then(|v| Role::parse(v).ok());
        require(
            record["schema"] == SCHEMA
                && role.is_some()
                && expected_role.is_none_or(|v| record["role"] == v),
            "foreign dry-run build role",
        )?;
        let role = role.ok_or("foreign dry-run build role")?;
        let binary = PathBuf::from(text(&record, "binary")?);
        let binary_digest = digest(&read_file(&binary, FILE_BOUND)?);
        retained.insert(binary, binary_digest.clone());
        require(
            binary_digest == record["binary_sha256"],
            "dry-run binary changed after its build",
        )?;
        require(
            record["features"] == json!(role.features())
                && record["helper_features"] == json!(role.helper_features()),
            "dry-run compiled role features differ",
        )?;
        let cargo = path
            .parent()
            .ok_or("build record parent missing")?
            .join("cargo.stdout");
        let cargo_bytes = read_file(&cargo, JSON_BOUND)?;
        retained.insert(cargo, digest(&cargo_bytes));
        let frames = frames(&cargo_bytes)?;
        let finished: Vec<_> = frames
            .iter()
            .filter(|v| v["reason"] == "build-finished")
            .collect();
        require(
            finished.len() == 1
                && finished[0]["success"] == true
                && frames.last() == Some(finished[0]),
            "dry-run Cargo build did not complete",
        )?;
        let empty = Value::Null;
        let candidate = frozen.or(self.config.as_ref()).unwrap_or(&empty);
        let repo = candidate
            .get("repo")
            .and_then(Value::as_str)
            .map_or_else(default_repo, |v| Ok(PathBuf::from(v)))?;
        let mut helpers = Vec::new();
        for row in &frames {
            let env = env_pairs(row)?;
            if env
                .get("GOMAD_LINKED_IMPLEMENTATION_SHA256")
                .is_some_and(|v| record["helper_sha256"] == *v)
            {
                helpers.push(row);
            }
        }
        require(
            helpers.len() == 1 && package(helpers[0], &repo, true),
            "dry-run linked build identity missing or ambiguous",
        )?;
        let out = PathBuf::from(text(helpers[0], "out_dir")?);
        let compiler = out.join("identity-compiler.txt");
        let bindings = out.join("identity-bindings.tsv");
        let compiler_text =
            String::from_utf8(read_file(&compiler, JSON_BOUND)?).map_err(|e| e.to_string())?;
        let binding_text =
            String::from_utf8(read_file(&bindings, JSON_BOUND)?).map_err(|e| e.to_string())?;
        let mut observed = BTreeMap::new();
        let mut unique = true;
        for line in binding_text.lines() {
            if let Some((key, value)) = line.split_once('\t') {
                unique &= observed.insert(key.to_owned(), value.to_owned()).is_none();
            } else {
                unique = false;
            }
        }
        retained.insert(compiler, digest(compiler_text.as_bytes()));
        retained.insert(bindings, digest(binding_text.as_bytes()));
        let target = observed.get("TARGET").cloned().unwrap_or_default();
        require(
            unique
                && !target.is_empty()
                && observed == feature_bindings(&role.helper_features(), &target)
                && compiler_text.starts_with("rustc 1.99.1 "),
            "dry-run compiler/feature context differs",
        )?;
        let artifacts = selected(&frames, role, &repo);
        require(
            artifacts.len() == 1 && {
                let row = artifacts[0];
                let mut got: Vec<_> = row["features"]
                    .as_array()
                    .map(|a| a.iter().filter_map(Value::as_str).collect())
                    .unwrap_or_default();
                got.sort_unstable();
                let mut want = role.features();
                want.sort_unstable();
                row["features"]
                    .as_array()
                    .is_some_and(|v| v.iter().all(Value::is_string))
                    && got == want
                    && row["profile"]["opt_level"] == "3"
                    && row["profile"]["debug_assertions"] == false
                    && row["profile"]["test"] == role.libtest()
            },
            "dry-run Cargo artifact role differs",
        )?;
        let artifact = PathBuf::from(text(artifacts[0], "executable")?);
        require(
            digest(&read_file(&artifact, FILE_BOUND)?) == binary_digest,
            "dry-run published binary differs from actual Cargo artifact",
        )?;
        retained.insert(artifact, binary_digest);
        let snapshot = Snapshot::observe(&repo, runner, cancelled, context)?;
        let (identity, runtime) = identities(&repo, &snapshot, role, &compiler_text, &target)?;
        require(
            record["host_sha256"] == identity["host"]
                && record["helper_sha256"] == identity["helper"]
                && record["runtime_implementation_sha256"] == runtime,
            "dry-run compiled implementation differs from current source/features",
        )?;
        if !role.helper() {
            let hosts = frames
                .iter()
                .map(|row| Ok((package(row, &repo, false), env_pairs(row)?)))
                .collect::<Result<Vec<_>, String>>()?
                .into_iter()
                .filter_map(|(own, env)| {
                    env.contains_key("GOMAD_HOST_IMPLEMENTATION_SHA256")
                        .then_some((own, env))
                })
                .collect::<Vec<_>>();
            require(
                hosts.len() == 1
                    && hosts[0].0
                    && hosts[0]
                        .1
                        .get("GOMAD_HOST_IMPLEMENTATION_SHA256")
                        .is_some_and(|v| identity["host"] == *v)
                    && hosts[0]
                        .1
                        .get("GOMAD_RUNTIME_IMPLEMENTATION_SHA256")
                        .is_some_and(|v| runtime == *v),
                "dry-run host build identity differs",
            )?;
        }
        let mut inputs = snapshot.files.clone();
        for (name, sha) in &retained {
            inputs.insert(name.display().to_string(), sha.clone());
        }
        let source = source_digest(&snapshot);
        if !candidate.is_null() {
            require(
                candidate["source_sha256"] == source
                    && candidate["head"] == snapshot.head
                    && candidate["symlinks"] == json!(snapshot.symlinks),
                "dry-run source custody differs from current candidate",
            )?;
            if role.helper_features().is_empty() {
                require(
                    candidate["identity"]["helper"] == record["helper_sha256"],
                    "dry-run helper differs from the candidate helper",
                )?;
            }
            if matches!(
                role.0,
                "release"
                    | "runtime-harness"
                    | "fn222-support"
                    | "interception-harness"
                    | "ordinary-host-test"
            ) {
                require(
                    candidate["identity"]["host"] == record["host_sha256"],
                    "dry-run host differs from the candidate host",
                )?;
            }
            if frozen.is_some() {
                require(
                    inputs
                        .iter()
                        .all(|(name, sha)| candidate["files"][name] == *sha),
                    "dry-run build custody is incompletely sealed",
                )?;
                if role.0 == "release" {
                    require(
                        candidate["binary"] == record["binary"]
                            && candidate["binary_sha256"] == record["binary_sha256"],
                        "dry-run release differs from the admitted candidate",
                    )?;
                }
            }
        }
        snapshot.verify(&repo, runner, cancelled, context)?;
        for (path, sha) in retained {
            require(
                digest(&read_file(&path, FILE_BOUND)?) == sha,
                "dry-run build inputs changed during authentication",
            )?;
        }
        let mut receipt = record;
        for (key,value) in json!({"exit":0,"profile":"release","inputs":inputs,"root":path.parent(),"source_sha256":source,"head":snapshot.head,"symlinks":snapshot.symlinks,"tools_plan":null,"gnu_plan":null,"context":{"compiler_utf8":compiler_text,"bindings":{"PROFILE":"release","TARGET":target}}}).as_object().ok_or("invalid receipt fields")?{receipt[key]=value.clone();}
        Ok(receipt)
    }
    #[expect(
        clippy::too_many_arguments,
        reason = "Build role, paths and process context bindings remain explicit"
    )]
    pub fn build(
        name: &str,
        repo: &Path,
        root: &Path,
        cargo: &Path,
        target: &Path,
        runner: &mut Runner,
        cancelled: &AtomicBool,
        context: &str,
    ) -> Result<PathBuf, String> {
        let role = Role::parse(name)?;
        require(
            root.is_absolute() && target.is_absolute(),
            "build paths must be absolute",
        )?;
        let directory = BuildDirectory::create(root, role.0)?;
        let output = directory.path.clone();
        let before = Snapshot::observe(repo, runner, cancelled, context)?;
        let status = observe_command(
            runner,
            context,
            repo,
            vec![
                "/usr/bin/git".into(),
                "-C".into(),
                repo.display().to_string(),
                "status".into(),
                "--porcelain".into(),
                "--untracked-files=no".into(),
            ],
            BTreeMap::from([
                ("PATH".into(), "/usr/bin".into()),
                ("GIT_CONFIG_NOSYSTEM".into(), "1".into()),
                ("GIT_CONFIG_GLOBAL".into(), "/dev/null".into()),
                ("GIT_OPTIONAL_LOCKS".into(), "0".into()),
            ]),
            Duration::from_secs(30),
            cancelled,
        )?;
        require(
            status.completed.accepted(),
            "could not observe actual build source status",
        )?;
        let dirty = !status.stdout.is_empty();
        let argv = cargo_command(cargo, repo, role);
        let environment = super::process::Environment::clean(
            std::env::vars().collect(),
            &BTreeMap::from([
                ("CARGO_TARGET_DIR".into(), target.display().to_string()),
                ("RUSTUP_TOOLCHAIN".into(), "1.99.1".into()),
            ]),
            &BTreeMap::new(),
        )
        .values()
        .clone();
        let captured = observe_command(
            runner,
            context,
            repo,
            argv.clone(),
            environment,
            Duration::from_secs(7200),
            cancelled,
        )?;
        directory.write("cargo.stdout", &captured.stdout, 0o600)?;
        directory.write("cargo.stderr", &captured.stderr, 0o600)?;
        require(
            captured.completed.exit == 0 && captured.completed.accepted(),
            &format!(
                "{name}: dry-run cargo build failed (exit {}); see {}",
                captured.completed.exit,
                output.join("cargo.stderr").display()
            ),
        )?;
        let frames = frames(&captured.stdout)?;
        let (executable, observed) = build_observation(&frames, role, repo)?;
        let binary = output.join(role.binary_name());
        directory.write(
            role.binary_name(),
            &read_file(Path::new(&executable), FILE_BOUND)?,
            0o755,
        )?;
        before.verify(repo, runner, cancelled, context)?;
        let record = json!({"schema":SCHEMA,"role":role.0,"binary":binary,"binary_sha256":digest(&read_file(&binary,FILE_BOUND)?),"build_head":before.head,"build_dirty":dirty,"features":role.features(),"helper_features":role.helper_features(),"host_sha256":if role.helper(){None}else{observed.get("GOMAD_HOST_IMPLEMENTATION_SHA256")},"helper_sha256":observed.get("GOMAD_LINKED_IMPLEMENTATION_SHA256"),"runtime_implementation_sha256":if role.helper(){None}else{observed.get("GOMAD_RUNTIME_IMPLEMENTATION_SHA256")},"argv":argv,"scope":"current-source cycle preparation; no qualification pass"});
        let path = output.join("dry-build.json");
        directory.write(
            "dry-build.json",
            &serde_json::to_vec(&record).map_err(|e| e.to_string())?,
            0o600,
        )?;
        directory.check()?;
        Ok(path)
    }
}
fn build_observation(
    frames: &[Value],
    role: Role,
    repo: &Path,
) -> Result<(String, BTreeMap<String, String>), String> {
    let name = role.0;
    let rows = selected(frames, role, repo);
    require(
        rows.len() == 1,
        &format!("{name}: dry-run artifact missing or ambiguous"),
    )?;
    let mut observed = BTreeMap::new();
    for row in frames {
        for (key, value) in env_pairs(row)?.into_iter().filter(|(key, _)| {
            matches!(
                key.as_str(),
                "GOMAD_LINKED_IMPLEMENTATION_SHA256"
                    | "GOMAD_HOST_IMPLEMENTATION_SHA256"
                    | "GOMAD_RUNTIME_IMPLEMENTATION_SHA256"
            )
        }) {
            require(
                observed.insert(key, value).is_none(),
                "duplicate build identity observation",
            )?;
        }
    }
    require(
        observed
            .get("GOMAD_LINKED_IMPLEMENTATION_SHA256")
            .is_some_and(|v| !v.is_empty())
            && (role.helper()
                || observed
                    .get("GOMAD_HOST_IMPLEMENTATION_SHA256")
                    .is_some_and(|v| !v.is_empty())),
        &format!("{name}: build identities not observed"),
    )?;
    Ok((text(rows[0], "executable")?.into(), observed))
}
pub(crate) fn is_record(path: &Path) -> bool {
    path.canonicalize().is_ok_and(|path| {
        file_json(&path).is_ok_and(|value| value.get("schema") == Some(&json!(SCHEMA)))
    })
}
struct BuildDirectory {
    path: PathBuf,
    handles: Vec<(PathBuf, fs::File)>,
}
impl BuildDirectory {
    fn create(root: &Path, role: &str) -> Result<Self, String> {
        use rustix::fs::{Mode, OFlags, mkdirat, open, openat};
        super::custody::clean_absolute(root)?;
        let flags = OFlags::RDONLY | OFlags::DIRECTORY | OFlags::NOFOLLOW | OFlags::CLOEXEC;
        let mut handles = vec![(
            PathBuf::from("/"),
            fs::File::from(open("/", flags, Mode::empty()).map_err(|e| e.to_string())?),
        )];
        for part in root.components().filter_map(|v| {
            if let std::path::Component::Normal(v) = v {
                Some(v)
            } else {
                None
            }
        }) {
            let (path, parent) = handles.last().ok_or("build directory missing")?;
            match mkdirat(parent, part, Mode::from_raw_mode(0o700)) {
                Ok(()) | Err(rustix::io::Errno::EXIST) => {}
                Err(e) => return Err(e.to_string()),
            }
            let next = fs::File::from(
                openat(parent, part, flags, Mode::empty()).map_err(|e| e.to_string())?,
            );
            handles.push((path.join(part), next));
        }
        let (path, parent) = handles.last().ok_or("build directory missing")?;
        mkdirat(parent, role, Mode::from_raw_mode(0o700))
            .map_err(|e| format!("dry-run role output must be fresh: {e}"))?;
        let output = path.join(role);
        let handle =
            fs::File::from(openat(parent, role, flags, Mode::empty()).map_err(|e| e.to_string())?);
        handles.push((output.clone(), handle));
        let value = Self {
            path: output,
            handles,
        };
        value.check()?;
        Ok(value)
    }
    fn check(&self) -> Result<(), String> {
        for (path, file) in &self.handles {
            let opened = file.metadata().map_err(|e| e.to_string())?;
            let named = fs::symlink_metadata(path).map_err(|e| e.to_string())?;
            require(
                named.is_dir() && opened.dev() == named.dev() && opened.ino() == named.ino(),
                "build output directory identity changed",
            )?;
        }
        Ok(())
    }
    fn write(&self, name: &str, bytes: &[u8], mode: u32) -> Result<(), String> {
        use rustix::fs::{Mode, OFlags, fchmod, openat};
        self.check()?;
        let directory = &self.handles.last().ok_or("build directory missing")?.1;
        let mut file = fs::File::from(
            openat(
                directory,
                name,
                OFlags::WRONLY | OFlags::CREATE | OFlags::EXCL | OFlags::NOFOLLOW | OFlags::CLOEXEC,
                Mode::from_raw_mode(mode),
            )
            .map_err(|e| e.to_string())?,
        );
        file.write_all(bytes).map_err(|e| e.to_string())?;
        fchmod(&file, Mode::from_raw_mode(mode)).map_err(|e| e.to_string())?;
        file.sync_all().map_err(|e| e.to_string())?;
        directory.sync_all().map_err(|e| e.to_string())?;
        self.check()
    }
}
fn feature_bindings(features: &[&str], target: &str) -> BTreeMap<String, String> {
    let mut values = BTreeMap::from([
        ("PROFILE".into(), "release".into()),
        ("TARGET".into(), target.into()),
    ]);
    for feature in features {
        values.insert(
            format!("CARGO_FEATURE_{}", feature.replace('-', "_").to_uppercase()),
            "1".into(),
        );
    }
    values
}
mod identity;
use identity::identities;
#[cfg(test)]
mod tests;
#[cfg(test)]
pub(crate) use tests::run_cases;
