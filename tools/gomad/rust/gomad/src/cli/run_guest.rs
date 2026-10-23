mod input;

#[cfg(feature = "determinism-fixture")]
use std::io::Write as _;

use crate::{
    campaign::{launcher, origin, preparation, prepare::driver},
    evidence::canonical,
};
use std::{
    ffi::{OsStr, OsString},
    path::{Path, PathBuf},
    time::Instant,
};

/// Refuses incompatible preparation bindings and bounds all final output.
#[must_use]
pub fn run(arguments: &[OsString], installation: Option<&Path>) -> i32 {
    let mut deadline = Deadline::new();
    let parsed = match parse(arguments) {
        Ok(parsed) => parsed,
        Err(error) => {
            return if launcher::refusal_process(&error.message, deadline.at).is_ok() {
                error.status
            } else {
                3
            };
        }
    };
    match execute(&parsed, installation, &mut deadline) {
        Ok((outcome, installation)) => {
            if launcher::report_process(&outcome, parsed.json, &installation, deadline.at).is_err()
            {
                return 3;
            }
            outcome.status
        }
        Err((status, message)) => {
            if launcher::refusal_process(&message, deadline.at).is_err() {
                3
            } else {
                status
            }
        }
    }
}

struct Deadline {
    started: Instant,
    at: Instant,
    guard: crate::backstop::Guard,
}

impl Deadline {
    fn new() -> Self {
        let started = Instant::now();
        let overall = std::time::Duration::from_secs(600);
        Self {
            started,
            at: started + overall,
            guard: crate::backstop::arm_until(started + overall),
        }
    }
    fn limit(&mut self, overall: std::time::Duration) -> Result<(), String> {
        let at = self
            .started
            .checked_add(overall)
            .ok_or("launcher deadline exceeds clock range")?;
        if at <= Instant::now() {
            return Err("overall_timeout: launcher preparation deadline exceeded".into());
        }
        self.guard = crate::backstop::arm_until(at);
        self.at = at;
        Ok(())
    }
}

struct Arguments {
    module: PathBuf,
    argv: Vec<String>,
    seeds: Option<String>,
    artifacts: Option<PathBuf>,
    json: bool,
}

fn parse(arguments: &[OsString]) -> Result<Arguments, super::Error> {
    let mut index = 0;
    let mut seeds = None;
    let mut artifacts = None;
    let mut json = false;
    while let Some(value) = arguments.get(index).and_then(|v| v.to_str()) {
        if value == "--" {
            index += 1;
            break;
        }
        if !value.starts_with('-') {
            break;
        }
        if value == "--json" {
            json = true;
            index += 1;
            continue;
        }
        let (name, supplied) = value
            .split_once('=')
            .map_or((value, None), |(a, b)| (a, Some(b)));
        if !matches!(name, "--seeds" | "--artifacts") {
            return Err(format!("flag provided but not defined: {name}").into());
        }
        index += 1;
        let supplied = if let Some(value) = supplied {
            OsStr::new(value)
        } else {
            let value = arguments
                .get(index)
                .ok_or_else(|| format!("flag needs an argument: {name}"))?;
            index += 1;
            value
        };
        match name {
            "--seeds" => {
                seeds = Some(
                    supplied
                        .to_str()
                        .ok_or("seed selection is not UTF-8")?
                        .into(),
                );
            }
            _ => artifacts = Some(supplied.into()),
        }
    }
    let module = arguments
        .get(index)
        .ok_or("run-guest requires MODULE [ARG ...]")?
        .into();
    let arguments = arguments[index + 1..]
        .iter()
        .map(|v| {
            v.to_str()
                .map(str::to_owned)
                .ok_or_else(|| super::Error::from("guest arguments must be UTF-8"))
        })
        .collect::<Result<Vec<_>, _>>()?;
    Ok(Arguments {
        module,
        argv: arguments,
        seeds,
        artifacts,
        json,
    })
}

fn execute(
    arguments: &Arguments,
    installation: Option<&Path>,
    deadline: &mut Deadline,
) -> Result<(launcher::Outcome, PathBuf), (i32, String)> {
    let artifacts = arguments
        .artifacts
        .clone()
        .or_else(|| std::env::var_os("GOMAD_SHIM_ARTIFACTS").map(PathBuf::from))
        .unwrap_or_else(|| ".gomad/artifacts".into());
    let mut staged =
        input::Input::capture(&arguments.module, &artifacts).map_err(|message| (2, message))?;
    let module = std::mem::take(&mut staged.bytes);
    let Admitted {
        prepared,
        options,
        installation,
    } = match admit(arguments, installation, module, deadline) {
        Ok(result) => {
            staged.accepted().map_err(|message| (3, message))?;
            result
        }
        Err(message) => {
            let status = 2;
            let directory = staged.directory.clone();
            staged.reject(status, &message).map_err(|error| {
                (
                    3,
                    format!("{message}; unaccepted input diagnostic publication failed: {error}"),
                )
            })?;
            return Err((
                status,
                format!(
                    "{message}; unaccepted input retained at {}",
                    directory.display()
                ),
            ));
        }
    };
    launcher::run(&options, &prepared, deadline.at)
        .map(|outcome| (outcome, installation))
        .map_err(|message| (3, message))
}

struct Admitted {
    prepared: crate::campaign::prepare::Prepared,
    options: super::explore::ExploreOptions,
    installation: PathBuf,
}

fn admit(
    arguments: &Arguments,
    installation: Option<&Path>,
    module: Vec<u8>,
    deadline: &mut Deadline,
) -> Result<Admitted, String> {
    arguments
        .module
        .to_str()
        .ok_or("guest module path must be UTF-8")?;
    if let Some(receipt_path) = std::env::var_os("GOMAD_PREPARATION_RECEIPT") {
        return admit_retained(arguments, installation, &module, &receipt_path, deadline);
    }
    let invocation_path = std::env::var_os("GOMAD_SHIM_INVOCATION")
        .ok_or("run-guest requires authenticated preparation metadata")?;
    let invocation_digest = std::env::var("GOMAD_SHIM_INVOCATION_SHA256")
        .map_err(|_| "run-guest requires preparation metadata digest")?;
    let path = Path::new(&invocation_path);
    let bytes = origin::read_private(path, &invocation_digest)?;
    let invocation: origin::Invocation =
        serde_json::from_slice(&bytes).map_err(|e| e.to_string())?;
    invocation.validate()?;
    if canonical::encode(&invocation)? != bytes {
        return Err("noncanonical intercepted invocation".into());
    }
    origin::authenticate_driver(path, &invocation_digest, &invocation, &arguments.module)?;
    let recipe_path = path
        .parent()
        .ok_or("invocation owner missing")?
        .join(&invocation.recipe_file);
    let bytes = origin::read_private(&recipe_path, &invocation.recipe_sha256)?;
    let mut recipes: driver::collection::Recipes =
        serde_json::from_slice(&bytes).map_err(|e| e.to_string())?;
    if canonical::encode(&recipes)? != bytes {
        return Err("noncanonical driver preparation recipe".into());
    }
    recipes.validate()?;
    let selected = crate::install::manifest::resolve(installation)?;
    for recipe in recipes.packages_mut() {
        if canonical::encode(&selected.manifest)? != canonical::encode(&recipe.installed)?
            || selected.sha256 != invocation.installation_sha256
        {
            return Err("launcher installation differs from preparation".into());
        }
        recipe.options.installation = Some(selected.path.clone());
        apply_options(&mut recipe.options, arguments)?;
    }
    let options = recipes.first()?.options.clone();
    deadline.limit(options.overall_timeout)?;
    let mut prepared =
        recipes.admit_external(module, arguments.argv.clone(), &invocation, deadline.at)?;
    origin::intercept(&mut prepared, invocation)?;
    let prepared = retain_admission(&options, prepared)?;
    Ok(Admitted {
        prepared,
        options,
        installation: selected.path,
    })
}

fn retain_admission(
    options: &super::explore::ExploreOptions,
    prepared: crate::campaign::prepare::Prepared,
) -> Result<crate::campaign::prepare::Prepared, String> {
    let bound =
        origin::from_preparation(&prepared)?.ok_or("admission requires intercepted origin")?;
    let retained = preparation::retain_admitted(options, prepared)?;
    let admission_root = admission_attempt(&options.artifacts, &bound.invocation.id)?;
    retained.publish(
        &admission_root.join("plan.json"),
        std::env::args_os()
            .map(|argument| {
                argument
                    .into_string()
                    .map_err(|_| "portable preparation command requires UTF-8 arguments")
            })
            .collect::<Result<Vec<_>, _>>()?,
    )?;
    Ok(retained.prepared)
}

fn admission_attempt(artifacts: &Path, invocation_id: &str) -> Result<PathBuf, String> {
    use std::{fs, os::unix::fs::DirBuilderExt as _};
    if !canonical::valid_digest(invocation_id) {
        return Err("invalid admission invocation identity".into());
    }
    let parent = artifacts.join("admissions").join(invocation_id);
    crate::evidence::store::durable_directory(&parent, |_| Ok(()))?;
    let path = parent.join(origin::identity()?);
    fs::DirBuilder::new()
        .mode(0o700)
        .create(&path)
        .map_err(|error| error.to_string())?;
    fs::File::open(&parent)
        .and_then(|directory| directory.sync_all())
        .map_err(|error| error.to_string())?;
    Ok(path)
}

fn admit_retained(
    arguments: &Arguments,
    installation: Option<&Path>,
    module: &[u8],
    receipt_path: &OsStr,
    deadline: &mut Deadline,
) -> Result<Admitted, String> {
    let digest = std::env::var("GOMAD_PREPARATION_RECEIPT_SHA256")
        .map_err(|_| "run-guest requires prepared receipt digest")?;
    admit_receipt(
        arguments,
        installation,
        module,
        receipt_path,
        &digest,
        deadline,
    )
}

fn admit_receipt(
    arguments: &Arguments,
    installation: Option<&Path>,
    module: &[u8],
    receipt_path: &OsStr,
    digest: &str,
    deadline: &mut Deadline,
) -> Result<Admitted, String> {
    let (_, mut prepared) = preparation::open_receipt(Path::new(receipt_path), digest)?;
    let origin = origin::from_preparation(&prepared)?
        .ok_or("run-guest requires intercepted preparation origin")?;
    let selected = crate::install::manifest::resolve(installation)?;
    if origin.invocation.installation_sha256 != selected.sha256
        || canonical::digest(module) != canonical::digest(&prepared.module)
        || prepared.configuration.arguments.get(1..) != Some(arguments.argv.as_slice())
    {
        return Err(
            "standalone module, arguments or installation differ from the prepared receipt".into(),
        );
    }
    prepared.compile_cache_root = crate::campaign::prepare::compile_cache_root(&selected.manifest)?;
    origin::renew_launch(&mut prepared, digest.to_owned())?;
    let mut ignored = false;
    let mut options = super::explore::parse(
        &[
            "--choices".into(),
            "--execution-timeout=5m".into(),
            "--overall-timeout=10m".into(),
            "go-test".into(),
            "retained".into(),
        ],
        &mut ignored,
    )?;
    options.installation = Some(selected.path.clone());
    apply_options(&mut options, arguments)?;
    deadline.limit(options.overall_timeout)?;
    let prepared = retain_admission(&options, prepared)?;
    Ok(Admitted {
        prepared,
        options,
        installation: selected.path,
    })
}

fn apply_options(
    options: &mut super::explore::ExploreOptions,
    arguments: &Arguments,
) -> Result<(), String> {
    options.seeds = arguments
        .seeds
        .clone()
        .or_else(|| std::env::var("GOMAD_SHIM_SEEDS").ok())
        .unwrap_or_else(|| "1".into());
    options.artifacts = arguments
        .artifacts
        .clone()
        .or_else(|| std::env::var_os("GOMAD_SHIM_ARTIFACTS").map(PathBuf::from))
        .unwrap_or_else(|| ".gomad/artifacts".into());
    options.artifacts = std::path::absolute(&options.artifacts).map_err(|e| e.to_string())?;
    options.parallel = 1;
    options.on_failure = "first".into();
    options.json = arguments.json;
    crate::campaign::seeds::SeedSelection::parse(&options.seeds)?;
    Ok(())
}

#[cfg(feature = "determinism-fixture")]
pub(crate) fn fixture_prepare(arguments: &[String]) -> Result<(), String> {
    use std::{
        fs,
        os::unix::fs::{DirBuilderExt as _, OpenOptionsExt as _},
    };
    if arguments.len() < 5 || arguments[0] != "--output" || arguments[2] != "--installation" {
        return Err("fixture-driver-prepare requires --output DIR --installation FILE [PREPARATION OPTIONS] TARGET".into());
    }
    let root = std::path::absolute(&arguments[1]).map_err(|e| e.to_string())?;
    fs::DirBuilder::new()
        .mode(0o700)
        .create(&root)
        .map_err(|e| e.to_string())?;
    let selected = crate::install::manifest::resolve(Some(Path::new(&arguments[3])))?;
    let (second, preparation_arguments) =
        if arguments.get(4).map(String::as_str) == Some("--package") {
            (
                Some(
                    arguments
                        .get(5)
                        .ok_or("private second package operand missing")?,
                ),
                &arguments[6..],
            )
        } else {
            (None, &arguments[4..])
        };
    let mut json = false;
    let mut options = super::explore::parse(preparation_arguments, &mut json)?;
    options.installation = Some(selected.path);
    let deadline = Instant::now() + options.overall_timeout;
    let recipe = driver::prepare(&options, deadline)?;
    let recipes = if let Some(second) = second {
        options.source.clone_from(second);
        driver::collection::Recipes::new(vec![recipe, driver::prepare(&options, deadline)?])?
    } else {
        driver::collection::Recipes::Single(Box::new(recipe))
    };
    let bytes = canonical::encode(&recipes)?;
    let recipe_file = "driver-recipe.json";
    let publish = |name: &str, bytes: &[u8]| -> Result<(), String> {
        let mut file = fs::OpenOptions::new()
            .write(true)
            .create_new(true)
            .mode(0o600)
            .open(root.join(name))
            .map_err(|e| e.to_string())?;
        file.write_all(bytes)
            .and_then(|()| file.sync_all())
            .map_err(|e| e.to_string())
    };
    publish(recipe_file, &bytes)?;
    let argv = recipes.driver_arguments(&std::env::current_exe().map_err(|e| e.to_string())?)?;
    let recipe = recipes.first()?;
    let argv_bytes = argv
        .iter()
        .map(|a| origin::argument(OsStr::new(a)))
        .collect::<Vec<_>>();
    let invocation = origin::Invocation {
        schema: "gomad.intercepted-invocation/v1".into(),
        wrapper_run_id: origin::identity()?,
        id: origin::identity()?,
        original_argv: argv_bytes.clone(),
        driver_argv: argv_bytes,
        compiler_sha256: recipe.installed.compiler_sha256.clone(),
        installation_sha256: selected.sha256,
        recipe_sha256: canonical::digest(&bytes),
        recipe_file: recipe_file.into(),
    };
    let bytes = canonical::encode(&invocation)?;
    publish("invocation.json", &bytes)?;
    println!(
        "{}",
        serde_json::json!({"invocation":root.join("invocation.json"),"invocation_sha256":canonical::digest(&bytes),"driver_argv":argv,"environment":recipe.environment,"working_directory":recipe.directory,"guest_executions":0})
    );
    Ok(())
}

#[cfg(feature = "determinism-fixture")]
pub(crate) fn fixture_driver_run(arguments: &[String]) -> Result<(), String> {
    use std::{
        fs,
        os::unix::fs::DirBuilderExt as _,
        process::{Command, Stdio},
    };
    if arguments.len() != 4 || arguments[0] != "--invocation" || arguments[2] != "--sha256" {
        return Err("fixture-driver-run requires --invocation FILE --sha256 HASH".into());
    }
    let path = Path::new(&arguments[1]);
    let bytes = origin::read_private(path, &arguments[3])?;
    let invocation: origin::Invocation =
        serde_json::from_slice(&bytes).map_err(|e| e.to_string())?;
    invocation.validate()?;
    if canonical::encode(&invocation)? != bytes {
        return Err("noncanonical intercepted invocation".into());
    }
    let root = path.parent().ok_or("invocation root missing")?;
    let recipe_bytes = origin::read_private(
        &root.join(&invocation.recipe_file),
        &invocation.recipe_sha256,
    )?;
    let recipes: driver::collection::Recipes =
        serde_json::from_slice(&recipe_bytes).map_err(|e| e.to_string())?;
    if canonical::encode(&recipes)? != recipe_bytes {
        return Err("noncanonical driver preparation recipe".into());
    }
    let argv = recipes.driver_arguments(&std::env::current_exe().map_err(|e| e.to_string())?)?;
    let recipe = recipes.first()?;
    if argv
        .iter()
        .map(|a| origin::argument(OsStr::new(a)))
        .collect::<Vec<_>>()
        != invocation.driver_argv
    {
        return Err("fixture driver arguments differ from invocation".into());
    }
    let module_root = root.join("go-build");
    fs::DirBuilder::new()
        .mode(0o700)
        .create(&module_root)
        .map_err(|e| e.to_string())?;
    let mut command = Command::new(&argv[0]);
    command
        .args(&argv[1..])
        .current_dir(&recipe.directory)
        .env_clear()
        .envs(&recipe.environment)
        .env("GOTMPDIR", &module_root)
        .env(
            "GOMAD_WASM_INSTALLATION",
            recipe
                .options
                .installation
                .as_ref()
                .ok_or("driver installation missing")?,
        )
        .env("GOMAD_SHIM_INVOCATION", path)
        .env("GOMAD_SHIM_INVOCATION_SHA256", &arguments[3])
        .env("GOMAD_SHIM_ARTIFACTS", &recipe.options.artifacts)
        .env("GOMAD_SHIM_SEEDS", &recipe.options.seeds)
        .stdin(Stdio::inherit())
        .stdout(Stdio::inherit())
        .stderr(Stdio::inherit());
    let registration = origin::DriverRegistration {
        schema: "gomad.driver-registration/v1".into(),
        process_id: std::process::id(),
        invocation_sha256: arguments[3].clone(),
        compiler_sha256: invocation.compiler_sha256.clone(),
        driver_argv_sha256: canonical::digest(&canonical::encode(&invocation.driver_argv)?),
        module_root,
    };
    registered_exec(root, &registration, &mut command)
}

#[cfg(feature = "determinism-fixture")]
fn registered_exec(
    root: &Path,
    registration: &origin::DriverRegistration,
    command: &mut std::process::Command,
) -> Result<(), String> {
    use std::{
        fs,
        os::unix::{fs::OpenOptionsExt as _, process::CommandExt as _},
    };
    let mut file = fs::OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .open(root.join("driver-registration.json"))
        .map_err(|e| e.to_string())?;
    file.write_all(&canonical::encode(registration)?)
        .and_then(|()| file.sync_all())
        .map_err(|e| e.to_string())?;
    fs::File::open(root)
        .and_then(|directory| directory.sync_all())
        .map_err(|e| e.to_string())?;
    drop(file);
    Err(format!("exec pinned driver: {}", command.exec()))
}

#[cfg(test)]
mod admission_tests;

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn preparation_consumes_the_original_overall_deadline() {
        let mut deadline = Deadline::new();
        let duration = std::time::Duration::from_millis(100);
        std::thread::sleep(std::time::Duration::from_millis(10));
        deadline.limit(duration).unwrap();
        assert_eq!(deadline.at, deadline.started + duration);
        assert!(deadline.at.duration_since(Instant::now()) < duration);
        assert!(deadline.limit(std::time::Duration::from_nanos(1)).is_err());
    }

    #[test]
    fn module_boundary_preserves_guest_flag_lookalikes() {
        let args = ["--seeds", "7,8", "module.wasm", "--json", "--seeds=99"].map(OsString::from);
        let parsed = parse(&args).unwrap_or_else(|error| panic!("{}", error.message));
        assert_eq!(parsed.seeds.as_deref(), Some("7,8"));
        assert!(!parsed.json);
        assert_eq!(parsed.argv, ["--json", "--seeds=99"]);
    }
}

#[cfg(all(test, feature = "determinism-fixture"))]
mod synchronization_tests {
    use super::*;
    #[test]
    fn registration_is_visible_before_exec_and_has_the_preserved_child_pid() {
        use std::{fs, os::unix::fs::DirBuilderExt as _, process::Command};
        let root = std::env::temp_dir().join(format!(
            "gomad-registration-{}",
            origin::identity().unwrap()
        ));
        fs::DirBuilder::new().mode(0o700).create(&root).unwrap();
        let child = Command::new(std::env::current_exe().unwrap())
            .args([
                "--exact",
                "cli::run_guest::synchronization_tests::registration_exec_child",
            ])
            .env("GOMAD_TEST_EXEC_ROOT", &root)
            .stdout(std::process::Stdio::piped())
            .stderr(std::process::Stdio::piped())
            .spawn()
            .unwrap();
        let pid = child.id();
        let output = child.wait_with_output().unwrap();
        assert!(
            output.status.success(),
            "{}",
            String::from_utf8_lossy(&output.stderr)
        );
        let start = output
            .stdout
            .iter()
            .position(|byte| *byte == b'{')
            .expect("execed program did not observe registration");
        let registration: origin::DriverRegistration =
            serde_json::from_slice(&output.stdout[start..]).unwrap();
        assert_eq!(registration.process_id, pid);
        assert_eq!(registration.invocation_sha256, "1".repeat(64));
        fs::remove_dir_all(root).unwrap();
    }
    #[test]
    fn registration_exec_child() {
        let Some(root) = std::env::var_os("GOMAD_TEST_EXEC_ROOT") else {
            return;
        };
        let root = Path::new(&root);
        let registration = origin::DriverRegistration {
            schema: "gomad.driver-registration/v1".into(),
            process_id: std::process::id(),
            invocation_sha256: "1".repeat(64),
            compiler_sha256: "2".repeat(64),
            driver_argv_sha256: "3".repeat(64),
            module_root: root.into(),
        };
        let mut command = std::process::Command::new("/bin/cat");
        command.arg(root.join("driver-registration.json"));
        let error = registered_exec(root, &registration, &mut command).unwrap_err();
        panic!("{error}");
    }
}
