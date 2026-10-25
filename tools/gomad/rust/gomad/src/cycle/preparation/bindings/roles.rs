use crate::{
    cycle::{build::DryRun, custody, process::Runner},
    measurement::{
        build,
        inputs::{CheckoutProof, authenticate_checkout},
    },
};
use serde::{Deserialize, Serialize};
use serde_json::{Value, json};
use std::{
    collections::{BTreeMap, BTreeSet},
    io::Write,
    os::unix::fs::DirBuilderExt,
    path::{Component, Path, PathBuf},
    sync::atomic::AtomicBool,
};

pub(super) type Files = BTreeMap<String, String>;

pub(super) fn read(path: &Path) -> Result<Value, String> {
    custody::strict_json(&custody::read_file(path, 64 << 20)?)
}
pub(super) fn pin(files: &mut Files, path: &Path) -> Result<(), String> {
    let digest = custody::digest(&custody::read_file(path, 512 << 20)?);
    let name = path.to_str().ok_or("non-UTF8 binding path")?.to_owned();
    if files
        .insert(name, digest.clone())
        .is_some_and(|old| old != digest)
    {
        return Err("binding input changed during authentication".into());
    }
    Ok(())
}
pub(super) fn path<'a>(value: &'a Value, key: &str) -> Result<&'a Path, String> {
    let path = Path::new(
        value[key]
            .as_str()
            .ok_or_else(|| format!("missing binding path: {key}"))?,
    );
    custody::clean_absolute(path)?;
    Ok(path)
}
pub(super) fn fields(value: &Value, required: &[&str], optional: &[&str]) -> Result<(), String> {
    let fields = value.as_object().ok_or("binding object required")?;
    if required.iter().any(|name| !fields.contains_key(*name))
        || fields
            .keys()
            .any(|name| !required.contains(&name.as_str()) && !optional.contains(&name.as_str()))
    {
        return Err("missing or unknown binding fields".into());
    }
    Ok(())
}

fn native_input_path(input: &Path) -> Result<PathBuf, String> {
    if !input.is_absolute() {
        return Err("relative authenticated native input".into());
    }
    let mut physical = PathBuf::new();
    for part in input.components() {
        match part {
            Component::RootDir => physical.push(part.as_os_str()),
            Component::Normal(_) => {
                physical.push(part.as_os_str());
                if std::fs::symlink_metadata(&physical)
                    .map_err(|e| e.to_string())?
                    .file_type()
                    .is_symlink()
                {
                    return Err("aliased authenticated native input".into());
                }
            }
            Component::CurDir => {}
            Component::ParentDir if physical.file_name().is_some() => {
                physical.pop();
            }
            Component::ParentDir | Component::Prefix(_) => {
                return Err("unsafe authenticated native input".into());
            }
        }
    }
    custody::clean_absolute(&physical)?;
    if input.canonicalize().map_err(|e| e.to_string())? != physical {
        return Err("aliased authenticated native input".into());
    }
    Ok(physical)
}

pub(crate) fn native_role_inputs(
    proof: &CheckoutProof,
    entry: &Value,
    expected: &build::Role,
) -> Result<Files, String> {
    fields(entry, &["binary", "build_receipt"], &["admission"])?;
    let binary = path(entry, "binary")?;
    let receipt_path = path(entry, "build_receipt")?;
    let receipt_value = read(receipt_path)?;
    let receipt: build::BuildReceipt =
        serde_json::from_value(receipt_value).map_err(|e| e.to_string())?;
    build::validate_build_receipt(proof, &receipt, expected)?;
    let mut files = Files::new();
    pin(&mut files, receipt_path)?;
    pin(&mut files, binary)?;
    pin(&mut files, &receipt.executable)?;
    for (input, expected) in &receipt.inputs {
        let physical = native_input_path(input)?;
        pin(&mut files, &physical)?;
        if files[physical.to_str().ok_or("non-UTF8 native input")?] != *expected {
            return Err("changed native role input".into());
        }
    }
    if entry.get("admission").is_some() {
        let admission_path = path(entry, "admission")?;
        let admission = build::validate_admission(proof, admission_path, expected)?;
        if admission.build_receipt != receipt_path || admission.executable != binary {
            return Err("native admission does not publish this exact build/binary".into());
        }
        pin(&mut files, admission_path)?;
        for name in ["artifact-identity", "staged-identity"] {
            for suffix in ["stdout", "stderr", "json"] {
                pin(
                    &mut files,
                    &admission_path
                        .parent()
                        .ok_or("admission parent missing")?
                        .join(format!("{name}.{suffix}")),
                )
                .map_err(|e| {
                    format!("missing/nonregular static native role input: {name}.{suffix}: {e}")
                })?;
            }
        }
    } else if binary != receipt.executable {
        return Err("published native role requires its actual admission".into());
    }
    Ok(files)
}
fn admitted_role(
    proof: &CheckoutProof,
    admission: &Path,
    role: &build::Role,
) -> Result<Files, String> {
    let value = read(admission)?;
    native_role_inputs(
        proof,
        &json!({"binary":value["executable"],"build_receipt":value["build_receipt"],"admission":admission}),
        role,
    )
}
pub(crate) fn native_bootstrap_inputs(
    proof: &CheckoutProof,
    config: &Value,
) -> Result<Files, String> {
    fields(&config["native_measurement"], &["admission"], &[])?;
    admitted_role(
        proof,
        path(&config["native_measurement"], "admission")?,
        &build::Role::Release,
    )
}
pub(crate) fn native_profile_inputs(
    proof: &CheckoutProof,
    config: &Value,
) -> Result<Files, String> {
    let profiles = &config["final_profiles"];
    let mut files = Files::new();
    for (name, role) in [
        ("measure", build::Role::Release),
        ("ordinary", build::Role::Release),
        (
            "diagnostic",
            build::Role::Diagnostic(vec!["diagnostic-profile".into()]),
        ),
    ] {
        merge(
            &mut files,
            native_role_inputs(proof, &profiles[name], &role)?,
        )?;
    }
    for (name, role) in [
        ("terminal_helper_receipt", build::Role::Release),
        (
            "terminal_diagnostic_helper_receipt",
            build::Role::Diagnostic(vec!["diagnostic-profile".into()]),
        ),
    ] {
        if profiles.get(name).is_some() {
            merge(
                &mut files,
                admitted_role(proof, path(profiles, name)?, &role)?,
            )?;
        }
    }
    Ok(files)
}
pub(super) fn merge(files: &mut Files, incoming: Files) -> Result<(), String> {
    for (name, sha) in incoming {
        if files
            .insert(name, sha.clone())
            .is_some_and(|old| old != sha)
        {
            return Err("conflicting role input digest".into());
        }
    }
    Ok(())
}
pub(super) fn candidate_role(
    config: &Value,
    entry: &Value,
    role: &str,
    runner: &mut Runner,
    cancelled: &AtomicBool,
) -> Result<(Value, Files), String> {
    fields(entry, &["binary", "build_receipt"], &[])?;
    let receipt_path = path(entry, "build_receipt")?;
    let binary = path(entry, "binary")?;
    let mut dry = DryRun::default();
    let mut selected = config.clone();
    selected["dry_run"] = json!(true);
    dry.enable(&selected);
    let private = super::private_context(runner)?;
    let receipt = dry.read_record(receipt_path, None, Some(role), runner, cancelled, &private)?;
    if receipt["binary"].as_str() != binary.to_str() {
        return Err("candidate role binary differs from authenticated build publication".into());
    }
    let mut files: Files =
        serde_json::from_value(receipt["inputs"].clone()).map_err(|e| e.to_string())?;
    pin(&mut files, receipt_path)?;
    Ok((receipt, files))
}
pub(crate) fn live_role_inputs(
    config: &Value,
    runner: &mut Runner,
    cancelled: &AtomicBool,
) -> Result<Files, String> {
    let roles = &config["live_corpus"]["roles"];
    fields(
        roles,
        &[
            "release",
            "host-fixture",
            "ordinary-helper-test",
            "ordinary-host-test",
        ],
        &[
            "counterfactual-fixture",
            "counterfactual-harness",
            "reference-transport",
        ],
    )?;
    let mut files = Files::new();
    for (role, entry) in roles.as_object().ok_or("live roles missing")? {
        merge(
            &mut files,
            candidate_role(config, entry, role, runner, cancelled)?.1,
        )?;
    }
    Ok(files)
}
pub(crate) fn runtime_harness_inputs(
    config: &Value,
    runner: &mut Runner,
    cancelled: &AtomicBool,
) -> Result<Files, String> {
    let mut bindings = BTreeSet::new();
    for index in 0..32 {
        let entry = &config["stages"][format!("runtime-{index}")];
        if entry["adapter"] != "runtime" {
            return Err("runtime stage requires its exact release-library gate".into());
        }
        bindings.insert((
            path(entry, "harness")?.to_owned(),
            path(entry, "harness_build_receipt")?.to_owned(),
        ));
    }
    if bindings.len() != 1 {
        return Err("all 32 original shards require one frozen runtime harness".into());
    }
    let (binary, receipt) = bindings
        .into_iter()
        .next()
        .ok_or("runtime harness missing")?;
    Ok(candidate_role(
        config,
        &json!({"binary":binary,"build_receipt":receipt}),
        "runtime-harness",
        runner,
        cancelled,
    )?
    .1)
}

#[derive(Clone, Debug, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct CandidateRoles {
    config: Value,
    files: Files,
}
impl CandidateRoles {
    #[expect(
        clippy::too_many_lines,
        reason = "All role groups and installation closures form one authenticated candidate boundary"
    )]
    pub(crate) fn authenticate(
        config: &Value,
        runner: &mut Runner,
        cancelled: &AtomicBool,
    ) -> Result<Self, String> {
        let mut files = candidate_role(
            config,
            &json!({"binary":config["binary"],"build_receipt":config["build_receipt"]}),
            "release",
            runner,
            cancelled,
        )?
        .1;
        if config["dry_run"] != true || config.get("live_corpus").is_some() {
            merge(&mut files, live_role_inputs(config, runner, cancelled)?)?;
        }
        merge(
            &mut files,
            runtime_harness_inputs(config, runner, cancelled)?,
        )?;
        if config["dry_run"] != true || config.get("sqlite_oracle").is_some() {
            merge(&mut files, sqlite_oracle_inputs(config, runner, cancelled)?)?;
        }
        if config["dry_run"] != true || config.get("public_verbs").is_some() {
            let (_, public_files) = super::public::public_inputs(
                config,
                path(&config["public_verbs"], "bindings")?,
                runner,
                cancelled,
            )?;
            merge(&mut files, public_files)?;
        }
        if config["dry_run"] != true
            || config.get("native_measurement").is_some()
            || config.get("final_profiles").is_some()
        {
            let proof = authenticate_checkout(path(config, "repo")?)?;
            if config["dry_run"] != true || config.get("native_measurement").is_some() {
                merge(&mut files, native_bootstrap_inputs(&proof, config)?)?;
            }
            if config["dry_run"] != true || config.get("final_profiles").is_some() {
                merge(&mut files, native_profile_inputs(&proof, config)?)?;
            }
        }
        let empty = serde_json::Map::new();
        let installations = if config["dry_run"] == true && config.get("installations").is_none() {
            &empty
        } else {
            config["installations"]
                .as_object()
                .ok_or("installation bindings missing")?
        };
        if installations.is_empty() && config["dry_run"] != true {
            return Err("installation bindings empty".into());
        }
        let mut checked_installations = BTreeSet::new();
        for value in installations.values() {
            let installation_path =
                PathBuf::from(value.as_str().ok_or("installation path missing")?);
            if !checked_installations.insert(installation_path.clone()) {
                continue;
            }
            let value = read(&installation_path)?;
            let installation: crate::install::manifest::Installation =
                serde_json::from_value(value).map_err(|e| e.to_string())?;
            let repo = path(config, "repo")?;
            if config["identity"]["host"] != installation.host_sha256
                || config["identity"]["helper"] != installation.helper_sha256
                || installation.compiler_path != path(&config["primary"], "compiler")?
                || installation
                    .overlay_module_root
                    .as_ref()
                    .is_some_and(|root| *root != repo.join("tools/gomad"))
            {
                return Err("installation differs from candidate identity/source/compiler".into());
            }
            crate::install::manifest::validate_for_executable(
                &installation,
                config["binary_sha256"]
                    .as_str()
                    .ok_or("candidate binary digest missing")?,
            )?;
            crate::install::manifest::executable(
                &installation.compiler_path,
                &installation.compiler_sha256,
            )?;
            let expected_compiler = crate::install::toolchain::pin(&installation)?;
            let admission = installation
                .admission
                .ok_or("actual installation compiler inventory required")?;
            if expected_compiler != admission.compiler_inputs {
                return Err("installation compiler closure is incomplete or foreign".into());
            }
            for (name, sha) in admission.compiler_inputs {
                pin(&mut files, &name)?;
                if files[name.to_str().ok_or("non-UTF8 compiler input")?] != sha {
                    return Err("changed installation compiler input".into());
                }
            }
            pin(&mut files, &installation_path)?;
        }
        custody::check_files(&files)?;
        Ok(Self {
            config: config.clone(),
            files,
        })
    }
    pub(crate) fn config(&self) -> &Value {
        &self.config
    }
    pub(crate) fn files(&self) -> &Files {
        &self.files
    }
    pub(crate) fn reauthenticate(
        &self,
        config: &Value,
        runner: &mut Runner,
        cancelled: &AtomicBool,
    ) -> Result<(), String> {
        custody::check_files(&self.files)?;
        let reopened = Self::authenticate(config, runner, cancelled)?;
        if reopened.files != self.files {
            return Err("candidate role closure changed".into());
        }
        Ok(())
    }
}

#[derive(Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct SqliteBuild {
    schema: String,
    repo: PathBuf,
    compiler: PathBuf,
    binary: PathBuf,
    binary_sha256: String,
    sources: Files,
    compiler_inputs: Files,
    command: Vec<String>,
    environment: BTreeMap<String, String>,
    completed: crate::cycle::process::Completed,
    build_info: String,
}
fn sqlite_environment(
    config: &Value,
    root: &Path,
    compiler: &Path,
) -> Result<BTreeMap<String, String>, String> {
    let sdk = compiler
        .parent()
        .and_then(Path::parent)
        .ok_or("Go compiler distribution missing")?;
    let mut env = BTreeMap::from([
        (
            "PATH".into(),
            format!("{}:/usr/bin:/bin", sdk.join("bin").display()),
        ),
        ("GOROOT".into(), sdk.display().to_string()),
        ("CGO_ENABLED".into(), "0".into()),
        ("GOENV".into(), "off".into()),
        ("GOWORK".into(), "off".into()),
        ("GOTOOLCHAIN".into(), "local".into()),
        ("GOPROXY".into(), "off".into()),
        ("GOSUMDB".into(), "off".into()),
        ("GOFLAGS".into(), String::new()),
        ("GOCACHEPROG".into(), String::new()),
        ("GOMAXPROCS".into(), "1".into()),
        ("GOTMPDIR".into(), root.join("tmp").display().to_string()),
    ]);
    for key in ["GOCACHE", "GOMODCACHE"] {
        let value = config["environment"][key]
            .as_str()
            .ok_or("SQLite requires assigned Go caches")?;
        custody::clean_absolute(Path::new(value))?;
        if Path::new(value).starts_with(path(config, "repo")?) {
            return Err("SQLite Go caches must be external".into());
        }
        env.insert(key.into(), value.into());
    }
    Ok(env)
}
fn sqlite_command(compiler: &Path, binary: &Path) -> Vec<String> {
    vec![
        compiler.display().to_string(),
        "build".into(),
        "-mod=readonly".into(),
        "-trimpath".into(),
        "-buildvcs=false".into(),
        "-tags=test_dep".into(),
        "-o".into(),
        binary.display().to_string(),
        "./internal/model/cmd/sqlite-native".into(),
    ]
}
fn sqlite_compiler(
    config: &Value,
    root: &Path,
    compiler: &Path,
    runner: &mut Runner,
    cancelled: &AtomicBool,
) -> Result<Files, String> {
    let observed = sqlite_capture(
        config,
        root,
        compiler,
        vec![
            "env".into(),
            "-json".into(),
            "GOROOT".into(),
            "GOTOOLDIR".into(),
        ],
        runner,
        cancelled,
    )?;
    let environment = custody::strict_json(&observed.stdout)?;
    let sdk = path(&environment, "GOROOT")?;
    if compiler.parent().and_then(Path::parent) != Some(sdk) {
        return Err("SQLite compiler distribution differs from actual Go environment".into());
    }
    let tools = path(&environment, "GOTOOLDIR")?;
    let mut paths = BTreeSet::from([
        sdk.join("VERSION"),
        tools.join("asm"),
        tools.join("compile"),
        tools.join("link"),
    ]);
    for item in std::fs::read_dir(sdk.join("pkg/include")).map_err(|e| e.to_string())? {
        let path = item.map_err(|e| e.to_string())?.path();
        if path.extension().is_some_and(|e| e == "h") {
            paths.insert(path);
        }
    }
    let mut pins = Files::new();
    for path in &paths {
        pin(&mut pins, path)?;
    }
    let inventory = crate::cycle::preparation::inputs::Inventory::new(&json!({"files":pins}))?;
    let snapshot = crate::evidence::canonical::encode(&json!(paths))?;
    let (mut pins, actual) = inventory.compiler(&observed.stdout, &snapshot)?;
    if actual != sdk {
        return Err("SQLite compiler inventory root differs".into());
    }
    pin(&mut pins, compiler)?;
    Ok(pins)
}
fn sqlite_capture(
    config: &Value,
    root: &Path,
    compiler: &Path,
    arguments: Vec<String>,
    runner: &mut Runner,
    cancelled: &AtomicBool,
) -> Result<custody::Capture, String> {
    let mut argv = vec![compiler.display().to_string()];
    argv.extend(arguments);
    let private = super::private_context(runner)?;
    let result = custody::observe_command(
        runner,
        &private,
        &path(config, "repo")?.join("tools/gomad"),
        argv,
        sqlite_environment(config, root, compiler)?,
        std::time::Duration::from_mins(15),
        cancelled,
    )?;
    if !result.completed.accepted() || result.completed.exit != 0 {
        return Err("SQLite Go command failed".into());
    }
    Ok(result)
}
fn sqlite_sources(
    config: &Value,
    root: &Path,
    compiler: &Path,
    runner: &mut Runner,
    cancelled: &AtomicBool,
) -> Result<Files, String> {
    let listed = sqlite_capture(
        config,
        root,
        compiler,
        vec![
            "list".into(),
            "-mod=readonly".into(),
            "-deps".into(),
            "-json".into(),
            "-tags=test_dep".into(),
            "./internal/model/cmd/sqlite-native".into(),
        ],
        runner,
        cancelled,
    )?;
    let mut files = Files::new();
    let mut main = false;
    for package in serde_json::Deserializer::from_slice(&listed.stdout).into_iter::<Value>() {
        let package = package.map_err(|e| e.to_string())?;
        if package.get("Error").is_some() || package["Incomplete"] == true {
            return Err("incomplete SQLite source graph".into());
        }
        main |= package["ImportPath"]
            == "go.temporal.io/server/tools/gomad/internal/model/cmd/sqlite-native";
        let directory = path(&package, "Dir")?;
        for field in [
            "GoFiles",
            "CgoFiles",
            "CFiles",
            "HFiles",
            "SFiles",
            "SysoFiles",
            "EmbedFiles",
        ] {
            if let Some(names) = package[field].as_array() {
                for name in names {
                    let relative =
                        Path::new(name.as_str().ok_or("SQLite package source name missing")?);
                    if relative.is_absolute()
                        || relative
                            .components()
                            .any(|p| !matches!(p, std::path::Component::Normal(_)))
                    {
                        return Err("unsafe SQLite package source name".into());
                    }
                    pin(&mut files, &directory.join(relative))?;
                }
            }
        }
        if let Some(module) = package.get("Module") {
            if module.get("Replace").is_some() {
                return Err("SQLite oracle module replacement forbidden".into());
            }
            if let Some(sum) = module["GoMod"].as_str() {
                pin(&mut files, Path::new(sum))?;
            }
            if module["Main"] != true {
                let name = module["Path"]
                    .as_str()
                    .ok_or("SQLite dependency path missing")?;
                let version = module["Version"]
                    .as_str()
                    .ok_or("SQLite dependency version missing")?;
                let sum = module["Sum"]
                    .as_str()
                    .ok_or("SQLite dependency checksum missing")?;
                let bytes = custody::read_file(
                    &path(config, "repo")?.join("tools/gomad/go.sum"),
                    16 << 20,
                )?;
                let wanted = format!("{name} {version} {sum}");
                if !std::str::from_utf8(&bytes)
                    .map_err(|e| e.to_string())?
                    .lines()
                    .any(|line| line == wanted)
                {
                    return Err("SQLite dependency absent from candidate Go sums".into());
                }
            }
        }
    }
    if !main {
        return Err("SQLite oracle package omitted from Go graph".into());
    }
    for module in [
        path(config, "repo")?.to_owned(),
        path(config, "repo")?.join("tools/gomad"),
    ] {
        for name in ["go.mod", "go.sum"] {
            pin(&mut files, &module.join(name))?;
        }
    }
    Ok(files)
}
impl SqliteBuild {
    pub(crate) fn build(
        config: &Value,
        root: &Path,
        runner: &mut Runner,
        cancelled: &AtomicBool,
    ) -> Result<PathBuf, String> {
        custody::clean_absolute(root)?;
        std::fs::create_dir(root).map_err(|e| e.to_string())?;
        std::fs::create_dir(root.join("tmp")).map_err(|e| e.to_string())?;
        let compiler = path(&config["primary"], "compiler")?;
        let compiler_inputs = sqlite_compiler(config, root, compiler, runner, cancelled)?;
        let sources = sqlite_sources(config, root, compiler, runner, cancelled)?;
        let binary = root.join("sqlite-native");
        let command = sqlite_command(compiler, &binary);
        let built = sqlite_capture(
            config,
            root,
            compiler,
            command[1..].to_vec(),
            runner,
            cancelled,
        )?;
        let info = sqlite_capture(
            config,
            root,
            compiler,
            vec!["version".into(), "-m".into(), binary.display().to_string()],
            runner,
            cancelled,
        )?;
        let record = Self {
            schema: "gomad.rust-sqlite-go-build/v1".into(),
            repo: path(config, "repo")?.to_owned(),
            compiler: compiler.to_owned(),
            binary: binary.clone(),
            binary_sha256: custody::digest(&custody::read_file(&binary, 512 << 20)?),
            sources,
            compiler_inputs,
            command,
            environment: sqlite_environment(config, root, compiler)?,
            completed: built.completed,
            build_info: String::from_utf8(info.stdout).map_err(|e| e.to_string())?,
        };
        let output = root.join("build.json");
        std::fs::OpenOptions::new()
            .create_new(true)
            .write(true)
            .open(&output)
            .map_err(|e| e.to_string())?
            .write_all(&crate::evidence::canonical::encode(
                &serde_json::to_value(&record).map_err(|e| e.to_string())?,
            )?)
            .map_err(|e| e.to_string())?;
        Ok(output)
    }
}
pub(crate) fn sqlite_oracle_inputs(
    config: &Value,
    runner: &mut Runner,
    cancelled: &AtomicBool,
) -> Result<Files, String> {
    struct Rebuild(std::path::PathBuf);
    impl Drop for Rebuild {
        fn drop(&mut self) {
            let _ = std::fs::remove_dir_all(&self.0);
        }
    }

    if config["stages"]["sqlite-catalogue"].get("oracle").is_some() {
        return Err("legacy SQLite oracle argv is forbidden".into());
    }
    fields(&config["sqlite_oracle"], &["build_receipt"], &[])?;
    let receipt = path(&config["sqlite_oracle"], "build_receipt")?;
    let raw = read(receipt)?;
    let build: SqliteBuild = serde_json::from_value(raw).map_err(|e| e.to_string())?;
    let root = receipt.parent().ok_or("SQLite build parent missing")?;
    if build.schema != "gomad.rust-sqlite-go-build/v1"
        || build.repo != path(config, "repo")?
        || build.compiler != path(&config["primary"], "compiler")?
        || build.binary != root.join("sqlite-native")
        || build.command != sqlite_command(&build.compiler, &build.binary)
        || build.environment != sqlite_environment(config, root, &build.compiler)?
        || !build.completed.accepted()
        || build.completed.exit != 0
        || build.completed.pid == 0
        || build.completed.resources.waited_pid != build.completed.pid
        || build.completed.resources.reaps != 1
        || build.completed.started_monotonic_nanos >= build.completed.reaped_monotonic_nanos
    {
        return Err("SQLite build source/compiler/command custody differs".into());
    }
    let compiler_inputs = sqlite_compiler(config, root, &build.compiler, runner, cancelled)?;
    if compiler_inputs != build.compiler_inputs
        || sqlite_sources(config, root, &build.compiler, runner, cancelled)? != build.sources
    {
        return Err("SQLite source/compiler closure changed".into());
    }
    let info = sqlite_capture(
        config,
        root,
        &build.compiler,
        vec![
            "version".into(),
            "-m".into(),
            build.binary.display().to_string(),
        ],
        runner,
        cancelled,
    )?;
    if info.stdout != build.build_info.as_bytes()
        || !build.build_info.contains(
            "\tpath\tgo.temporal.io/server/tools/gomad/internal/model/cmd/sqlite-native\n",
        )
        || !build.build_info.contains("\tbuild\t-tags=test_dep\n")
        || !build.build_info.contains("\tbuild\tCGO_ENABLED=0\n")
        || custody::digest(&custody::read_file(&build.binary, 512 << 20)?) != build.binary_sha256
    {
        return Err("SQLite executable changed fixed source/module/compiler flags".into());
    }
    let unique = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map_err(|e| e.to_string())?
        .as_nanos();
    let scratch = root.join(format!("reopen-{}-{unique}", std::process::id()));
    std::fs::DirBuilder::new()
        .mode(0o700)
        .create(&scratch)
        .map_err(|e| e.to_string())?;
    let _cleanup = Rebuild(scratch.clone());
    let output = scratch.join("sqlite-native");
    let command = sqlite_command(&build.compiler, &output);
    sqlite_capture(
        config,
        root,
        &build.compiler,
        command[1..].to_vec(),
        runner,
        cancelled,
    )?;
    if custody::digest(&custody::read_file(&output, 512 << 20)?) != build.binary_sha256 {
        return Err(
            "SQLite executable does not reproduce from current pinned source/compiler".into(),
        );
    }
    custody::check_files(&build.sources)?;
    custody::check_files(&compiler_inputs)?;
    let mut files = build.sources;
    merge(&mut files, compiler_inputs)?;
    pin(&mut files, &build.binary)?;
    pin(&mut files, receipt)?;
    Ok(files)
}

#[cfg(test)]
fn caught<T>(result: Result<T, String>, predicate: &str) -> Result<String, String> {
    match result {
        Err(error) if error.contains(predicate) => Ok(error),
        Err(error) => Err(format!("wrong catcher; expected {predicate}: {error}")),
        Ok(_) => Err(format!("mutation missed {predicate}")),
    }
}

#[cfg(test)]
pub(crate) fn run_owner_tests(
    config: &Value,
    runner: &mut Runner,
    cancelled: &AtomicBool,
) -> Result<Value, String> {
    let release = json!({"binary":config["binary"],"build_receipt":config["build_receipt"]});
    let (_, control) = candidate_role(config, &release, "release", runner, cancelled)?;
    let harness_control = runtime_harness_inputs(config, runner, cancelled)?;
    let mut observations = Vec::new();
    let mut extra = release.clone();
    extra["caller_claim"] = json!(true);
    observations.push(json!({"id":"orchestration:configure:114","mutation":extra,"refusal":caught(candidate_role(config,&extra,"release",runner,cancelled),"unknown binding fields")?}));
    let mut relative = release.clone();
    relative["binary"] = json!("relative");
    observations.push(json!({"id":"orchestration:configure:63","mutation":relative,"refusal":caught(candidate_role(config,&relative,"release",runner,cancelled),"absolute clean custody path")?}));
    observations.push(json!({"id":"orchestration:configure:117","mutation":"release receipt claimed as runtime-harness","refusal":caught(candidate_role(config,&release,"runtime-harness",runner,cancelled),"foreign dry-run build role")?}));
    let mut wrong_source = config.clone();
    wrong_source["head"] = json!("0".repeat(40));
    observations.push(json!({"id":"orchestration:configure:120","mutation":{"head":"0000000000000000000000000000000000000000"},"refusal":caught(candidate_role(&wrong_source,&release,"release",runner,cancelled),"source custody differs")?}));
    let mut wrong_adapter = config.clone();
    wrong_adapter["stages"]["runtime-13"]["adapter"] = json!("primary");
    observations.push(json!({"id":"orchestration:configure:138","mutation":{"stage":"runtime-13","adapter":"primary"},"refusal":caught(runtime_harness_inputs(&wrong_adapter,runner,cancelled),"runtime stage requires")?}));
    let mut mixed = config.clone();
    mixed["stages"]["runtime-13"]["harness"] = config["binary"].clone();
    observations.push(json!({"id":"orchestration:configure:140","mutation":{"stage":"runtime-13","harness":config["binary"]},"refusal":caught(runtime_harness_inputs(&mixed,runner,cancelled),"one frozen runtime harness")?}));
    let mut wrong_binary = config.clone();
    for index in 0..32 {
        wrong_binary["stages"][format!("runtime-{index}")]["harness"] = config["binary"].clone();
    }
    observations.push(json!({"id":"orchestration:configure:143","mutation":"all harness binaries replaced with release binary","refusal":caught(runtime_harness_inputs(&wrong_binary,runner,cancelled),"binary differs from authenticated build publication")?}));
    let receipt = path(config, "build_receipt")?;
    let original = custody::read_file(receipt, 64 << 20)?;
    let original_value = custody::strict_json(&original)?;
    for (field, value, catcher) in [
        ("schema", json!("foreign"), "foreign dry-run build role"),
        (
            "binary_sha256",
            json!("0".repeat(64)),
            "binary changed after its build",
        ),
        (
            "features",
            json!(["diagnostic-profile"]),
            "compiled role features differ",
        ),
    ] {
        let mut mutant = original_value.clone();
        mutant[field] = value;
        let raw = crate::evidence::canonical::encode(&mutant)?;
        let mutant_path = receipt
            .parent()
            .ok_or("receipt parent missing")?
            .join(format!("bindings-{field}-mutant.json"));
        let mut handle = std::fs::OpenOptions::new()
            .create_new(true)
            .write(true)
            .open(&mutant_path)
            .map_err(|e| e.to_string())?;
        handle.write_all(&raw).map_err(|e| e.to_string())?;
        drop(handle);
        let entry = json!({"binary":config["binary"],"build_receipt":mutant_path});
        let result = candidate_role(config, &entry, "release", runner, cancelled);
        std::fs::remove_file(&mutant_path).map_err(|e| e.to_string())?;
        observations.push(json!({"id":"current-role-build-integrity","field":field,"input":mutant,"input_sha256":custody::digest(&raw),"refusal":caught(result,catcher)?}));
    }
    Ok(
        json!({"schema":"gomad.bindings-role-controls/v1","release_control":control,"runtime_control":harness_control,"original_receipt_sha256":custody::digest(&original),"mutations":observations}),
    )
}

#[cfg(test)]
pub(crate) fn native_owner_cases(
    config: &Value,
    root: &Path,
    release_receipt: &Path,
    diagnostic_receipt: &Path,
) -> Result<Value, String> {
    let proof = authenticate_checkout(path(config, "repo")?)?;
    let release_role = build::Role::Release;
    let diagnostic_role = build::Role::Diagnostic(vec!["diagnostic-profile".into()]);
    let release: build::BuildReceipt =
        serde_json::from_value(read(release_receipt)?).map_err(|e| e.to_string())?;
    let diagnostic: build::BuildReceipt =
        serde_json::from_value(read(diagnostic_receipt)?).map_err(|e| e.to_string())?;
    build::validate_build_receipt(&proof, &release, &release_role)?;
    build::validate_build_receipt(&proof, &diagnostic, &diagnostic_role)?;
    let release_admission = build::stage_role(&proof, root, release_receipt, &release_role)?;
    let second_admission = build::stage_role(&proof, root, release_receipt, &release_role)?;
    let diagnostic_admission =
        build::stage_role(&proof, root, diagnostic_receipt, &diagnostic_role)?;
    let release_admitted = read(&release_admission)?;
    let diagnostic_admitted = read(&diagnostic_admission)?;
    let release_raw = json!({"binary":release.executable,"build_receipt":release_receipt});
    let release_entry = json!({"binary":release_admitted["executable"],"build_receipt":release_receipt,"admission":release_admission});
    let diagnostic_entry = json!({"binary":diagnostic_admitted["executable"],"build_receipt":diagnostic_receipt,"admission":diagnostic_admission});
    let raw_control = native_role_inputs(&proof, &release_raw, &release_role)?;
    let staged_control = native_role_inputs(&proof, &release_entry, &release_role)?;
    let selected = json!({"native_measurement":{"admission":release_admission},"final_profiles":{"measure":release_entry,"ordinary":release_raw,"diagnostic":diagnostic_entry,"terminal_helper_receipt":release_admission,"terminal_diagnostic_helper_receipt":diagnostic_admission}});
    let bootstrap_control = native_bootstrap_inputs(&proof, &selected)?;
    let profile_control = native_profile_inputs(&proof, &selected)?;
    let mut rows = Vec::new();
    let mut extra = release_entry.clone();
    extra["caller_claim"] = json!(true);
    rows.push(json!({"id":"orchestration:configure:61","input":extra,"refusal":caught(native_role_inputs(&proof,&extra,&release_role),"unknown binding fields")?}));
    let mut relative = release_entry.clone();
    relative["build_receipt"] = json!("relative-receipt.json");
    rows.push(json!({"id":"orchestration:configure:63","input":relative,"refusal":caught(native_role_inputs(&proof,&relative,&release_role),"absolute clean custody path")?}));
    let mut schema = read(release_receipt)?;
    schema["schema"] = json!("foreign-native-build");
    let bytes = crate::evidence::canonical::encode(&schema)?;
    let mutant_path = root.join("native-schema-mutant.json");
    std::fs::OpenOptions::new()
        .create_new(true)
        .write(true)
        .open(&mutant_path)
        .map_err(|e| e.to_string())?
        .write_all(&bytes)
        .map_err(|e| e.to_string())?;
    let mutant = json!({"binary":release.executable,"build_receipt":mutant_path});
    let schema_result = native_role_inputs(&proof, &mutant, &release_role);
    std::fs::remove_file(&mutant_path).map_err(|e| e.to_string())?;
    rows.push(json!({"id":"orchestration:configure:67","input":schema,"input_sha256":custody::digest(&bytes),"refusal":caught(schema_result,"receipt does not bind pinned release command")?}));
    let unpublished =
        json!({"binary":release_admitted["executable"],"build_receipt":release_receipt});
    rows.push(json!({"id":"orchestration:configure:70","input":unpublished,"refusal":caught(native_role_inputs(&proof,&unpublished,&release_role),"published native role requires its actual admission")?}));
    let mut foreign = release_entry.clone();
    foreign["admission"] = json!(second_admission);
    rows.push(json!({"id":"orchestration:configure:74","input":foreign,"refusal":caught(native_role_inputs(&proof,&foreign,&release_role),"admission does not publish this exact build/binary")?}));
    let staged_stream = release_admission
        .parent()
        .ok_or("native admission parent missing")?
        .join("staged-identity.stderr");
    let saved_stream = staged_stream.with_extension("stderr.saved");
    let stream_bytes = custody::read_file(&staged_stream, 64 << 20)?;
    std::fs::rename(&staged_stream, &saved_stream).map_err(|e| e.to_string())?;
    let missing = native_role_inputs(&proof, &release_entry, &release_role);
    std::fs::rename(&saved_stream, &staged_stream).map_err(|e| e.to_string())?;
    rows.push(json!({"id":"orchestration:configure:80","mutation":{"missing_static_input":staged_stream,"original_sha256":custody::digest(&stream_bytes),"original_bytes":stream_bytes},"refusal":caught(missing,"missing/nonregular static native role input")?}));
    let mut bootstrap = selected.clone();
    bootstrap["native_measurement"]["caller_claim"] = json!(true);
    rows.push(json!({"id":"orchestration:configure:87","input":bootstrap["native_measurement"],"refusal":caught(native_bootstrap_inputs(&proof,&bootstrap),"unknown binding fields")?}));
    Ok(
        json!({"schema":"gomad.bindings-native-controls/v1","release_receipt":release_receipt,"release_receipt_sha256":custody::digest(&custody::read_file(release_receipt,64<<20)?),"diagnostic_receipt":diagnostic_receipt,"diagnostic_receipt_sha256":custody::digest(&custody::read_file(diagnostic_receipt,64<<20)?),"selected":selected,"raw_control":raw_control,"staged_control":staged_control,"bootstrap_control":bootstrap_control,"profile_control":profile_control,"mutations":rows}),
    )
}

#[cfg(test)]
mod path_tests {
    use super::*;
    #[test]
    fn authenticated_native_parent_references_retain_nofollow_custody() {
        let root =
            std::env::temp_dir().join(format!("gomad-native-input-path-{}", std::process::id()));
        std::fs::create_dir(&root).unwrap();
        std::fs::create_dir(root.join("parent")).unwrap();
        std::fs::write(root.join("input"), b"actual native input").unwrap();
        assert_eq!(
            native_input_path(&root.join("parent/../input")).unwrap(),
            root.join("input")
        );
        std::os::unix::fs::symlink(root.join("parent"), root.join("alias")).unwrap();
        for input in [
            PathBuf::from("relative"),
            PathBuf::from("/../escape"),
            root.join("alias/../input"),
        ] {
            assert!(native_input_path(&input).is_err());
        }
        std::fs::remove_dir_all(root).unwrap();
    }
}
