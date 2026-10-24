use super::Contract;
use crate::{
    campaign::{
        origin, preparation,
        prepare::{
            driver::{Recipe, shared::Shared},
            inventory,
        },
    },
    cli::run_guest::build_launcher,
    evidence::{canonical, symbolize::build_info},
    install::manifest,
};
use std::{
    collections::BTreeSet,
    ffi::{OsStr, OsString},
    fs,
    io::Write as _,
    os::unix::{
        ffi::OsStrExt as _,
        fs::{DirBuilderExt as _, OpenOptionsExt as _},
        process::CommandExt as _,
    },
    path::{Path, PathBuf},
    process::{Command, Stdio},
    time::{Duration, Instant},
};

fn shell_quote(value: &OsStr) -> Result<Vec<u8>, String> {
    if value.as_bytes().contains(&0) {
        return Err("launcher argument contains NUL".into());
    }
    let mut bytes = vec![b'\''];
    for &byte in value.as_bytes() {
        if byte == b'\'' {
            bytes.extend_from_slice(b"'\\''");
        } else {
            bytes.push(byte);
        }
    }
    bytes.push(b'\'');
    Ok(bytes)
}

fn script(
    binary: &Path,
    installation: &Path,
    module: &Path,
    receipt: &Path,
    digest: &str,
    contract: &Contract,
) -> Result<Vec<u8>, String> {
    let mut bytes = b"#!/bin/sh\n".to_vec();
    for (name, value) in [
        ("GOMAD_PREPARATION_RECEIPT", receipt.as_os_str()),
        ("GOMAD_PREPARATION_RECEIPT_SHA256", OsStr::new(digest)),
        ("GOMAD_SHIM_ARTIFACTS", contract.artifacts.as_os_str()),
        ("GOMAD_SHIM_SEEDS", OsStr::new(&contract.seeds)),
    ] {
        bytes.extend_from_slice(b"export ");
        bytes.extend_from_slice(name.as_bytes());
        bytes.push(b'=');
        bytes.extend(shell_quote(value)?);
        bytes.push(b'\n');
    }
    bytes.extend_from_slice(b"exec ");
    bytes.extend(shell_quote(binary.as_os_str())?);
    bytes.extend_from_slice(b" --installation ");
    bytes.extend(shell_quote(installation.as_os_str())?);
    bytes.extend_from_slice(b" run-guest -- ");
    bytes.extend(shell_quote(module.as_os_str())?);
    bytes.extend_from_slice(b" \"$@\"\n");
    Ok(bytes)
}

fn publish_pair(
    destination: &Path,
    module_path: &Path,
    module: &[u8],
    launcher: &[u8],
) -> Result<(), String> {
    for path in [destination, module_path] {
        match fs::symlink_metadata(path) {
            Ok(meta) if !meta.is_file() => {
                return Err("build output destination is not a regular file".into());
            }
            Ok(_) => (),
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => (),
            Err(error) => return Err(error.to_string()),
        }
    }
    let parent = destination.parent().ok_or("launcher parent missing")?;
    let staging = crate::campaign::prepare::process::Temporary::new(parent)?;
    for (name, bytes, mode) in [("module", module, 0o600), ("launcher", launcher, 0o700)] {
        let mut file = fs::OpenOptions::new()
            .write(true)
            .create_new(true)
            .mode(mode)
            .open(staging.0.join(name))
            .map_err(|e| e.to_string())?;
        file.write_all(bytes)
            .and_then(|()| file.sync_all())
            .map_err(|e| e.to_string())?;
    }
    fs::rename(staging.0.join("module"), module_path).map_err(|e| e.to_string())?;
    fs::rename(staging.0.join("launcher"), destination).map_err(|e| e.to_string())?;
    fs::File::open(parent)
        .and_then(|file| file.sync_all())
        .map_err(|e| e.to_string())
}

/// This private dispatch records its own PID before replacing itself with the pinned compiler.
pub(super) fn exec_child(arguments: &[OsString]) -> Result<i32, String> {
    let path = PathBuf::from(
        std::env::var_os("GOMAD_SHIM_BUILD_CHILD").ok_or("missing private build invocation")?,
    );
    let digest = std::env::var("GOMAD_SHIM_BUILD_CHILD_SHA256")
        .map_err(|_| "missing private build invocation digest")?;
    let bytes = origin::read_private(&path, &digest)?;
    let invocation: origin::Invocation =
        serde_json::from_slice(&bytes).map_err(|e| e.to_string())?;
    invocation.validate()?;
    if canonical::encode(&invocation)? != bytes
        || arguments
            .iter()
            .map(|a| origin::argument(a))
            .collect::<Vec<_>>()
            != invocation.driver_argv[1..]
    {
        return Err("private build child arguments differ from registered invocation".into());
    }
    let root = path
        .parent()
        .ok_or("private build invocation has no parent")?;
    let bytes = origin::read_private(
        &root.join(&invocation.recipe_file),
        &invocation.recipe_sha256,
    )?;
    let shared: Shared = serde_json::from_slice(&bytes).map_err(|e| e.to_string())?;
    if canonical::encode(&shared)? != bytes
        || shared.installation_commitment()? != invocation.installation_sha256
        || shared.driver_argv != invocation.driver_argv
    {
        return Err("private build catalog commitment mismatch".into());
    }
    for recipe in &shared.packages {
        manifest::admit_build(&recipe.installed)?;
        inventory::verify(&recipe.compiler)?;
        inventory::verify(&recipe.sources)?;
    }
    let first = shared
        .packages
        .first()
        .ok_or("empty private build catalog")?;
    let mut command = super::guest_env::command(
        &first.installed.compiler_path,
        arguments,
        &first.environment,
    );
    command.env("GOTMPDIR", root.join("go-build"));
    let registration = origin::DriverRegistration {
        schema: "gomad.driver-registration/v1".into(),
        process_id: std::process::id(),
        invocation_sha256: digest,
        compiler_sha256: invocation.compiler_sha256.clone(),
        driver_argv_sha256: canonical::digest(&canonical::encode(&invocation.driver_argv)?),
        module_root: root.join("outputs"),
        module_cache_root: None,
    };
    crate::cli::run_guest::registered_exec(root, &registration, &mut command)?;
    Err("registered build exec unexpectedly returned".into())
}

fn build_index(argv: &[OsString]) -> Result<usize, String> {
    let mut index = 1;
    while let Some(argument) = argv.get(index) {
        if argument == "-C" {
            index += 2;
        } else if argument.as_bytes().starts_with(b"-C=") {
            index += 1;
        } else {
            return if argument == "build" {
                Ok(index)
            } else {
                Err("private build driver has no build subcommand".into())
            };
        }
    }
    Err("private build driver has no build subcommand".into())
}

fn output_arguments(
    argv: &[OsString],
    private: &Path,
    goflags: &OsStr,
) -> Result<(Vec<OsString>, Option<PathBuf>), String> {
    let offset = build_index(argv)? + 1;
    let parsed =
        super::classify::inspect_unchecked(&argv[offset..], goflags, super::classify::Kind::Build)?;
    let output = parsed
        .effective
        .get("o")
        .filter(|value| !value.is_empty())
        .map(PathBuf::from);
    let mut removed = BTreeSet::new();
    for range in &parsed.driver_ranges {
        let value = argv[range.start + offset].as_bytes();
        if value == b"-o"
            || value.starts_with(b"-o=")
            || value == b"--o"
            || value.starts_with(b"--o=")
        {
            removed.extend(range.clone());
        }
    }
    let mut result = argv[..offset].to_vec();
    result.extend([OsString::from("-o"), private.as_os_str().to_owned()]);
    result.extend(
        argv[offset..]
            .iter()
            .enumerate()
            .filter(|(index, _)| !removed.contains(index))
            .map(|(_, value)| value.clone()),
    );
    Ok((result, output))
}

fn registration(
    root: &Path,
    invocation: &origin::Invocation,
    digest: &str,
    pid: u32,
) -> Result<origin::DriverRegistration, String> {
    let bytes = manifest::read(&root.join("driver-registration.json"), 16384)?;
    let registered: origin::DriverRegistration =
        serde_json::from_slice(&bytes).map_err(|e| e.to_string())?;
    if canonical::encode(&registered)? != bytes
        || registered.schema != "gomad.driver-registration/v1"
        || registered.process_id != pid
        || registered.invocation_sha256 != digest
        || registered.compiler_sha256 != invocation.compiler_sha256
        || registered.driver_argv_sha256
            != canonical::digest(&canonical::encode(&invocation.driver_argv)?)
        || registered.module_root != root.join("outputs")
    {
        return Err("completed compiler differs from private build registration".into());
    }
    Ok(registered)
}

struct Child {
    process: std::process::Child,
    deadline: Instant,
}

impl Drop for Child {
    fn drop(&mut self) {
        if matches!(self.process.try_wait(), Ok(Some(_))) {
            return;
        }
        let cleanup_start = Instant::now().min(self.deadline);
        let _backstop = crate::backstop::arm_until(cleanup_start);
        let cleanup_end = cleanup_start + crate::backstop::CLEANUP_GRACE;
        let group = i32::try_from(self.process.id())
            .ok()
            .and_then(rustix::process::Pid::from_raw);
        if let Some(pid) = group {
            let _ = rustix::process::kill_process_group(pid, rustix::process::Signal::TERM);
        }
        loop {
            if matches!(self.process.try_wait(), Ok(Some(_))) {
                return;
            }
            let remaining = cleanup_end.saturating_duration_since(Instant::now());
            if remaining.is_zero() {
                break;
            }
            std::thread::sleep(Duration::from_millis(10).min(remaining));
        }
        if let Some(pid) = group {
            let _ = rustix::process::kill_process_group(pid, rustix::process::Signal::KILL);
        }
        let _ = self.process.kill();
        let _ = self.process.wait();
    }
}

#[expect(
    clippy::too_many_lines,
    reason = "Private build orders immutable registration, actual compiler lifetime and post-build admission before publication"
)]
pub(super) fn run(
    contract: &Contract,
    recipes: Vec<Recipe>,
    argv: &[OsString],
    deadline: Instant,
) -> Result<i32, String> {
    let invocation_id = origin::identity()?;
    let root = contract
        .directory
        .join(format!("invocation-{invocation_id}"));
    fs::DirBuilder::new()
        .mode(0o700)
        .create(&root)
        .map_err(|e| e.to_string())?;
    for name in ["outputs", "go-build"] {
        fs::DirBuilder::new()
            .mode(0o700)
            .create(root.join(name))
            .map_err(|e| e.to_string())?;
    }
    let first = recipes
        .first()
        .ok_or("guest build selected no concrete packages")?;
    let goflags = first
        .environment
        .get("GOFLAGS")
        .map_or(OsStr::new(""), |value| OsStr::new(value));
    // The caller argv was admitted before the shim injected its own recipe flags.
    let dry_run = super::classify::inspect_unchecked(
        &argv[build_index(argv)? + 1..],
        goflags,
        super::classify::Kind::Build,
    )?
    .effective
    .get("n")
    .is_some_and(|value| value != "false" && value != "0");
    let (argv, output) = output_arguments(argv, &root.join("outputs"), goflags)?;
    let output_base = first.directory.clone();
    let output = output.map(|path| {
        if path.is_absolute() {
            path
        } else {
            output_base.join(path)
        }
    });
    let directory_output = output
        .as_ref()
        .is_some_and(|path| path.is_dir() || path.as_os_str().as_bytes().ends_with(b"/"));
    if recipes.len() > 1 && output.is_some() && !directory_output {
        return Err("go: cannot write multiple packages to non-directory output".into());
    }
    let mut shared = Shared::new(
        recipes,
        argv.iter().map(|arg| origin::argument(arg)).collect(),
    )?;
    let raw = canonical::encode(&shared)?;
    if raw.len() > 32 << 20 {
        return Err("private build catalog exceeds existing bound".into());
    }
    super::publish(&root, "driver-recipe.json", &raw)?;
    let invocation = origin::Invocation {
        schema: "gomad.intercepted-invocation/v1".into(),
        wrapper_run_id: contract.run.clone(),
        id: invocation_id,
        original_argv: std::env::args_os()
            .map(|arg| origin::argument(&arg))
            .collect(),
        driver_argv: shared.driver_argv.clone(),
        compiler_sha256: shared.packages[0].installed.compiler_sha256.clone(),
        installation_sha256: shared.installation_commitment()?,
        recipe_sha256: canonical::digest(&raw),
        recipe_file: "driver-recipe.json".into(),
    };
    invocation.validate()?;
    let bytes = canonical::encode(&invocation)?;
    let digest = canonical::digest(&bytes);
    super::publish(&root, "invocation.json", &bytes)?;
    let mut command = Command::new(&contract.binary);
    let mut child = Child {
        process: crate::fork_order::spawning(|| {
            command
                .arg0("go")
                .args(&argv[1..])
                .env_clear()
                .envs(&shared.packages[0].environment)
                .env("GOMAD_SHIM_BUILD_CHILD", root.join("invocation.json"))
                .env("GOMAD_SHIM_BUILD_CHILD_SHA256", &digest)
                .stdin(Stdio::inherit())
                .stdout(Stdio::inherit())
                .stderr(Stdio::inherit())
                .process_group(0)
                .spawn()
        })
        .map_err(|e| e.to_string())?,
        deadline,
    };
    let pid = child.process.id();
    let status = loop {
        if let Some(status) = child.process.try_wait().map_err(|e| e.to_string())? {
            break status;
        }
        if Instant::now() >= deadline {
            return Err("overall_timeout: guest build exceeded preparation deadline".into());
        }
        std::thread::sleep(Duration::from_millis(10));
    };
    drop(child);
    let status = super::passthrough::status(status);
    if status != 0 {
        return Ok(status);
    }
    let registration = registration(&root, &invocation, &digest, pid)?;
    if origin::read_private(&root.join("invocation.json"), &digest)? != bytes
        || origin::read_private(&root.join("driver-recipe.json"), &invocation.recipe_sha256)? != raw
    {
        return Err("private build metadata changed during compiler execution".into());
    }
    if dry_run {
        return Ok(0);
    }
    for recipe in &mut shared.packages {
        recipe.inputs["build_driver_registration"] =
            serde_json::to_value(&registration).map_err(|e| e.to_string())?;
    }
    admit_outputs(
        contract,
        shared,
        &output_base,
        &invocation,
        &root,
        (output.as_deref(), directory_output),
        deadline,
    )?;
    Ok(0)
}

#[expect(
    clippy::too_many_lines,
    reason = "All actual modules are admitted before publishing the independently receipt-bound launchers"
)]
fn admit_outputs(
    contract: &Contract,
    mut shared: Shared,
    output_base: &Path,
    invocation: &origin::Invocation,
    root: &Path,
    output: (Option<&Path>, bool),
    deadline: Instant,
) -> Result<(), String> {
    let (output, directory_output) = output;
    let bindings = shared.installation_bindings();
    let expected = shared.packages.len();
    let private_outputs = root
        .join("outputs")
        .canonicalize()
        .map_err(|e| e.to_string())?;
    let mut seen = BTreeSet::new();
    let mut outputs = fs::read_dir(root.join("outputs"))
        .map_err(|e| e.to_string())?
        .collect::<Result<Vec<_>, _>>()
        .map_err(|e| e.to_string())?;
    outputs.sort_by_key(std::fs::DirEntry::file_name);
    if outputs.len() != expected {
        return Err("guest build output count differs from concrete package catalog".into());
    }
    // Authenticate every output before any launcher becomes visible.
    let mut admitted = Vec::new();
    for entry in outputs {
        let path = entry.path();
        if !path
            .canonicalize()
            .map_err(|e| e.to_string())?
            .starts_with(&private_outputs)
        {
            return Err("guest build output escapes registered private root".into());
        }
        let module = manifest::read(&path, 512 << 20)?;
        let metadata = build_info::read(&module)?;
        let index = shared.select(&metadata.path)?;
        if !seen.insert(metadata.path.clone()) {
            return Err("duplicate guest build output package".into());
        }
        let installation = shared.installations[index].clone();
        let mut recipe = shared.packages.remove(index);
        recipe.inputs["shared_driver_installations"] = serde_json::json!(bindings);
        shared.installations.remove(index);
        let options = recipe.options.clone();
        let selected = manifest::resolve(options.installation.as_deref())?;
        let mut prepared = recipe.admit_registered(
            module,
            Vec::new(),
            invocation,
            &invocation.driver_argv,
            &installation,
            &metadata,
            deadline,
        )?;
        build_launcher::seal(&mut prepared, &installation, invocation)?;
        origin::intercept(&mut prepared, invocation.clone())?;
        let retained = preparation::retain_admitted(&options, prepared)?;
        let receipt_root = contract
            .artifacts
            .join("build-launchers")
            .join(&invocation.id)
            .join(origin::identity()?);
        crate::evidence::store::durable_directory(&receipt_root, |_| Ok(()))?;
        let (receipt, _, digest) = retained.publish(
            &receipt_root.join("plan.json"),
            invocation
                .original_argv
                .iter()
                .map(|encoded| format!("os-hex:{encoded}"))
                .collect(),
        )?;
        let destination = if directory_output {
            Some(
                output
                    .ok_or("missing output directory")?
                    .join(entry.file_name()),
            )
        } else if let Some(output) = output {
            Some(output.to_path_buf())
        } else if expected == 1 {
            Some(output_base.join(entry.file_name()))
        } else {
            None
        };
        admitted.push((
            path,
            canonical::digest(&retained.prepared.module),
            receipt,
            digest,
            selected.path,
            destination,
        ));
    }
    for (path, module_sha256, receipt, digest, installation, destination) in admitted {
        let Some(destination) = destination else {
            continue;
        };
        let module = manifest::read(&path, 512 << 20)?;
        if canonical::digest(&module) != module_sha256 {
            return Err("admitted private build module changed before launcher publication".into());
        }
        let destination = std::path::absolute(destination).map_err(|e| e.to_string())?;
        if directory_output {
            fs::create_dir_all(destination.parent().ok_or("launcher parent missing")?)
                .map_err(|e| e.to_string())?;
        }
        let mut module_path = destination.as_os_str().to_owned();
        module_path.push(".wasm");
        let module_path = PathBuf::from(module_path);
        let launcher = script(
            &contract.binary,
            &installation,
            &module_path,
            &receipt,
            &digest,
            contract,
        )?;
        let lock_root = contract.artifacts.join("build-launcher-locks");
        crate::evidence::store::durable_directory(&lock_root, |_| Ok(()))?;
        let lock_name = canonical::digest(destination.as_os_str().as_bytes());
        let lock = fs::OpenOptions::new()
            .read(true)
            .write(true)
            .create(true)
            .mode(0o600)
            .custom_flags(
                (rustix::fs::OFlags::NOFOLLOW | rustix::fs::OFlags::NONBLOCK)
                    .bits()
                    .cast_signed(),
            )
            .open(lock_root.join(lock_name))
            .map_err(|e| e.to_string())?;
        let meta = lock.metadata().map_err(|e| e.to_string())?;
        if !meta.is_file()
            || std::os::unix::fs::MetadataExt::uid(&meta) != rustix::process::geteuid().as_raw()
            || std::os::unix::fs::MetadataExt::nlink(&meta) != 1
            || std::os::unix::fs::PermissionsExt::mode(&meta.permissions()) & 0o777 != 0o600
        {
            return Err("unsafe build launcher publication lock".into());
        }
        rustix::fs::flock(&lock, rustix::fs::FlockOperation::LockExclusive)
            .map_err(|e| e.to_string())?;
        publish_pair(&destination, &module_path, &module, &launcher)?;
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn shell_quote_preserves_quotes_spaces_and_non_utf8() {
        assert_eq!(
            shell_quote(OsStr::from_bytes(b"a' \xff")).unwrap(),
            b"'a'\\'' \xff'"
        );
        assert!(shell_quote(OsStr::from_bytes(b"a\0b")).is_err());
    }
    #[test]
    fn private_output_rewrite_keeps_other_flag_values_and_original_bytes() {
        let argv = [
            "/stock/go",
            "build",
            "-ldflags",
            "-o=literal",
            "-o",
            "caller",
            "./cmd",
        ]
        .map(OsString::from);
        let (actual, selected) =
            output_arguments(&argv, Path::new("/private/outputs"), OsStr::new("")).unwrap();
        assert_eq!(selected, Some(PathBuf::from("caller")));
        let global =
            ["/stock/go", "-C", "nested", "build", "-o=caller", "./cmd"].map(OsString::from);
        let (actual_global, selected_global) =
            output_arguments(&global, Path::new("/private/outputs"), OsStr::new("")).unwrap();
        assert_eq!(selected_global, Some(PathBuf::from("caller")));
        assert_eq!(
            actual_global,
            [
                "/stock/go",
                "-C",
                "nested",
                "build",
                "-o",
                "/private/outputs",
                "./cmd"
            ]
            .map(OsString::from)
        );

        assert_eq!(
            actual,
            [
                "/stock/go",
                "build",
                "-o",
                "/private/outputs",
                "-ldflags",
                "-o=literal",
                "./cmd"
            ]
            .map(OsString::from)
        );
    }

    #[test]
    fn private_output_rewrite_accepts_the_shim_injected_recipe_overlay() {
        let argv = [
            "/stock/go",
            "build",
            "-overlay=/private/overlay.json",
            "-trimpath",
            "-buildvcs=false",
            "-o",
            "caller",
            ".",
        ]
        .map(OsString::from);
        let (actual, selected) =
            output_arguments(&argv, Path::new("/private/outputs"), OsStr::new("")).unwrap();
        assert_eq!(selected, Some(PathBuf::from("caller")));
        assert_eq!(
            actual,
            [
                "/stock/go",
                "build",
                "-o",
                "/private/outputs",
                "-overlay=/private/overlay.json",
                "-trimpath",
                "-buildvcs=false",
                "."
            ]
            .map(OsString::from)
        );
    }

    #[test]
    fn generated_launcher_passes_runtime_arguments_and_bound_receipt_bytes() {
        let root =
            crate::campaign::prepare::process::Temporary::new(&std::env::temp_dir()).unwrap();
        let binary = root.0.join("capture 'binary");
        fs::write(&binary,b"#!/bin/sh\nprintf '%s\\0' \"$GOMAD_PREPARATION_RECEIPT\" \"$GOMAD_PREPARATION_RECEIPT_SHA256\" \"$@\" > \"$RESULT\"\n").unwrap();
        crate::fork_order::make_executable(&binary, 0o700);
        let contract = Contract {
            binary: binary.clone(),
            compiler: "/stock/go".into(),
            installation: None,
            artifacts: root.0.join("artifacts"),
            seeds: "7,8".into(),
            run: "1".repeat(64),
            directory: root.0.clone(),
        };
        let receipt = root.0.join("receipt 'json");
        let installation = root.0.join("install.json");
        let module = root.0.join("module.wasm");
        let launcher = root.0.join("launcher");
        fs::write(
            &launcher,
            script(
                &binary,
                &installation,
                &module,
                &receipt,
                &"2".repeat(64),
                &contract,
            )
            .unwrap(),
        )
        .unwrap();
        let result = root.0.join("captured");
        assert!(
            crate::fork_order::status(
                Command::new("/bin/sh")
                    .arg(&launcher)
                    .arg("--json")
                    .arg(OsStr::from_bytes(b"a' \xff"))
                    .env("RESULT", &result)
            )
            .unwrap()
            .success()
        );
        let actual = fs::read(result).unwrap();
        let expected = [
            receipt.as_os_str(),
            OsStr::new(&"2".repeat(64)),
            OsStr::new("--installation"),
            installation.as_os_str(),
            OsStr::new("run-guest"),
            OsStr::new("--"),
            module.as_os_str(),
            OsStr::new("--json"),
            OsStr::from_bytes(b"a' \xff"),
        ]
        .into_iter()
        .flat_map(|value| value.as_bytes().iter().copied().chain(std::iter::once(0)))
        .collect::<Vec<_>>();
        assert_eq!(actual, expected);
    }

    #[test]
    fn cleanup_delivers_term_and_reaps_a_running_plain_shell() {
        use std::io::Read as _;
        let mut process = crate::fork_order::spawn(
            Command::new("/bin/sh")
                .args([
                    "-c",
                    "trap 'printf term; exit 0' TERM; printf ready; while :; do :; done",
                ])
                .process_group(0)
                .stdout(Stdio::piped()),
        )
        .unwrap();
        let mut output = process.stdout.take().unwrap();
        let mut ready = [0; 5];
        output.read_exact(&mut ready).unwrap();
        assert_eq!(&ready, b"ready");
        drop(Child {
            process,
            deadline: Instant::now() + Duration::from_secs(600),
        });
        let mut terminal = Vec::new();
        output.read_to_end(&mut terminal).unwrap();
        assert_eq!(terminal, b"term");
    }

    #[test]
    fn cleanup_accepts_an_already_reaped_plain_child() {
        let mut process = crate::fork_order::spawn(
            Command::new("/bin/sh")
                .args(["-c", "exit 0"])
                .process_group(0),
        )
        .unwrap();
        assert!(process.wait().unwrap().success());
        drop(Child {
            process,
            deadline: Instant::now().checked_sub(Duration::from_secs(10)).unwrap(),
        });
    }
}
