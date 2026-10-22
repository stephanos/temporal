mod input;

use crate::{
    campaign::{launcher, origin, preparation, prepare::driver},
    evidence::canonical,
};
use std::{
    ffi::{OsStr, OsString},
    io::Write,
    path::{Path, PathBuf},
    time::Instant,
};

/// # Errors
/// Refuses missing or incompatible preparation bindings before guest admission.
pub fn run(arguments: &[OsString], installation: Option<&Path>) -> Result<i32, super::Error> {
    let parsed = parse(arguments)?;
    match execute(&parsed, installation) {
        Ok((outcome, installation)) => {
            launcher::report(
                &outcome,
                parsed.json,
                &installation,
                &mut std::io::stdout(),
                &mut std::io::stderr(),
            )
            .map_err(|message| super::Error { status: 3, message })?;
            Ok(outcome.status)
        }
        Err((status, message)) => {
            let mut stderr = std::io::stderr();
            writeln!(stderr, "gomad run-guest: {message}")
                .and_then(|()| {
                    writeln!(
                        stderr,
                        "gomad run-guest: no replayable artifact was retained"
                    )
                })
                .map_err(|e| super::Error {
                    status: 3,
                    message: e.to_string(),
                })?;
            Ok(status)
        }
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
    } = match admit(arguments, installation, module) {
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
    launcher::run(&options, &prepared)
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
) -> Result<Admitted, String> {
    arguments
        .module
        .to_str()
        .ok_or("guest module path must be UTF-8")?;
    if let Some(receipt_path) = std::env::var_os("GOMAD_PREPARATION_RECEIPT") {
        return admit_retained(arguments, installation, &module, &receipt_path);
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
    let mut recipe: driver::Recipe = serde_json::from_slice(&bytes).map_err(|e| e.to_string())?;
    if canonical::encode(&recipe)? != bytes {
        return Err("noncanonical driver preparation recipe".into());
    }
    let selected = crate::install::manifest::resolve(installation)?;
    if canonical::encode(&selected.manifest)? != canonical::encode(&recipe.installed)?
        || selected.sha256 != invocation.installation_sha256
    {
        return Err("launcher installation differs from preparation".into());
    }
    recipe.options.installation = Some(selected.path.clone());
    apply_options(&mut recipe.options, arguments)?;
    let options = recipe.options.clone();
    let deadline = Instant::now()
        .checked_add(options.overall_timeout)
        .ok_or("launcher preparation deadline exceeds clock range")?;
    let mut prepared =
        recipe.admit_external(module, arguments.argv.clone(), &invocation, deadline)?;
    let invocation_id = invocation.id.clone();
    origin::intercept(&mut prepared, invocation)?;
    let retained = preparation::retain_admitted(&options, prepared)?;
    let admission_root = options.artifacts.join("admissions").join(invocation_id);
    crate::evidence::store::durable_directory(&admission_root, |_| Ok(()))?;
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
    let prepared = retained.prepared;
    Ok(Admitted {
        prepared,
        options,
        installation: selected.path,
    })
}

fn admit_retained(
    arguments: &Arguments,
    installation: Option<&Path>,
    module: &[u8],
    receipt_path: &OsStr,
) -> Result<Admitted, String> {
    let digest = std::env::var("GOMAD_PREPARATION_RECEIPT_SHA256")
        .map_err(|_| "run-guest requires prepared receipt digest")?;
    let (_, prepared) = preparation::open_receipt(Path::new(receipt_path), &digest)?;
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
    let mut json = false;
    let mut options = super::explore::parse(&arguments[4..], &mut json)?;
    options.installation = Some(selected.path);
    let recipe = driver::prepare(&options, Instant::now() + options.overall_timeout)?;
    let bytes = canonical::encode(&recipe)?;
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
    let argv = recipe.driver_arguments(&std::env::current_exe().map_err(|e| e.to_string())?)?;
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
        os::unix::{
            fs::{DirBuilderExt as _, OpenOptionsExt as _},
            process::ExitStatusExt as _,
        },
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
    let root = path.parent().ok_or("invocation root missing")?;
    let recipe_bytes = origin::read_private(
        &root.join(&invocation.recipe_file),
        &invocation.recipe_sha256,
    )?;
    let recipe: driver::Recipe =
        serde_json::from_slice(&recipe_bytes).map_err(|e| e.to_string())?;
    let argv = recipe.driver_arguments(&std::env::current_exe().map_err(|e| e.to_string())?)?;
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
    let mut child = Command::new(&argv[0])
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
        .stderr(Stdio::inherit())
        .spawn()
        .map_err(|e| e.to_string())?;
    let registration = origin::DriverRegistration {
        schema: "gomad.driver-registration/v1".into(),
        process_id: child.id(),
        invocation_sha256: arguments[3].clone(),
        compiler_sha256: invocation.compiler_sha256.clone(),
        driver_argv_sha256: canonical::digest(&canonical::encode(&invocation.driver_argv)?),
        module_root,
    };
    let publication = (|| -> Result<(), String> {
        let mut file = fs::OpenOptions::new()
            .write(true)
            .create_new(true)
            .mode(0o600)
            .open(root.join("driver-registration.json"))
            .map_err(|e| e.to_string())?;
        file.write_all(&canonical::encode(&registration)?)
            .and_then(|()| file.sync_all())
            .map_err(|e| e.to_string())
    })();
    if let Err(error) = publication {
        let _ = child.kill();
        let _ = child.wait();
        return Err(error);
    }
    let status = child.wait().map_err(|e| e.to_string())?;
    std::process::exit(
        status
            .code()
            .unwrap_or_else(|| 128 + status.signal().unwrap_or(1)),
    );
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn module_boundary_preserves_guest_flag_lookalikes() {
        let args = ["--seeds", "7,8", "module.wasm", "--json", "--seeds=99"].map(OsString::from);
        let parsed = parse(&args).unwrap_or_else(|error| panic!("{}", error.message));
        assert_eq!(parsed.seeds.as_deref(), Some("7,8"));
        assert!(!parsed.json);
        assert_eq!(parsed.argv, ["--json", "--seeds=99"]);
    }
}
