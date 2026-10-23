mod build;
pub(crate) mod classify;
mod guest_env;
mod passthrough;

use crate::{
    campaign::{
        origin,
        prepare::{driver, process},
    },
    evidence::canonical,
    install::manifest,
};
use std::io::Write as _;
use std::{
    ffi::{OsStr, OsString},
    fs,
    os::unix::fs::{DirBuilderExt as _, OpenOptionsExt as _},
    path::{Path, PathBuf},
    time::{Duration, Instant},
};

#[derive(serde::Deserialize)]
struct Package {
    #[serde(rename = "ImportPath")]
    import_path: String,
    #[serde(rename = "Dir")]
    directory: PathBuf,
    #[serde(rename = "Name")]
    name: String,
}

struct Contract {
    binary: PathBuf,
    compiler: PathBuf,
    installation: Option<PathBuf>,
    artifacts: PathBuf,
    seeds: String,
    run: String,
    directory: PathBuf,
}

impl Contract {
    fn read() -> Result<Self, String> {
        let path = |name: &str| -> Result<PathBuf, String> {
            let value = PathBuf::from(
                std::env::var_os(name).ok_or_else(|| format!("missing shim contract {name}"))?,
            );
            if !value.is_absolute() {
                return Err(format!("shim contract {name} must be absolute"));
            }
            Ok(value)
        };
        let run = std::env::var("GOMAD_SHIM_RUN").map_err(|_| "missing shim run identity")?;
        if !canonical::valid_digest(&run) {
            return Err("invalid shim run identity".into());
        }
        Ok(Self {
            binary: path("GOMAD_SHIM_BINARY")?,
            compiler: path("GOMAD_SHIM_GO")?,
            installation: std::env::var_os("GOMAD_SHIM_INSTALLATION")
                .filter(|v| !v.is_empty())
                .map(PathBuf::from),
            artifacts: path("GOMAD_SHIM_ARTIFACTS")?,
            seeds: std::env::var("GOMAD_SHIM_SEEDS").map_err(|_| "missing shim seeds")?,
            run,
            directory: path("GOMAD_SHIM_RUN_DIR")?,
        })
    }
}

/// Runs the wrapper's exact pinned compiler, admitting guest commands before compilation.
/// # Errors
/// Refuses absent wrapper contracts, unsupported guest flags or invalid preparation inputs.
pub fn run(arguments: &[OsString]) -> Result<i32, String> {
    if std::env::var_os("GOMAD_SHIM_BUILD_CHILD").is_some() {
        return build::exec_child(arguments);
    }
    let contract = Contract::read()?;
    let goflags = std::env::var_os("GOFLAGS").unwrap_or_default();
    let Some(request) = classify_request(&contract, arguments, &goflags)? else {
        return passthrough::run(&contract.compiler, arguments);
    };
    let mut recipes = prepare_guest(&contract, &request)?;
    let command = request.command;
    let command_index = request.command_index;
    let parsed = &request.parsed;
    let build_flags = &request.build_flags;
    let deadline = request.deadline;
    let first = recipes
        .first()
        .ok_or("Go command selected no concrete packages")?;
    let mut injected = first.flags[..first.flags.len() - build_flags.len()]
        .iter()
        .map(OsString::from)
        .collect::<Vec<_>>();
    injected.extend(["-trimpath".into(), "-buildvcs=false".into()]);
    if command == "test" && parsed.count.is_none() {
        injected.push("-count=1".into());
    }
    let caller_exec = parsed.caller_exec;
    if !caller_exec && command != "build" {
        injected.extend([
            "-exec".into(),
            OsString::from(format!("{} run-guest", quote_go(&contract.binary)?)),
        ]);
    }
    let mut argv = vec![contract.compiler.as_os_str().to_owned()];
    argv.extend_from_slice(&arguments[..=command_index]);
    argv.extend(parsed.rewritten_arguments(&arguments[command_index + 1..], &injected));
    let goflags = parsed.rewritten_goflags();
    // GOFLAGS is an exact build input, even though source inventory uses its expanded flags.
    for recipe in &mut recipes {
        let value = goflags
            .to_str()
            .ok_or("guest GOFLAGS is not UTF-8")?
            .to_owned();
        recipe.environment.insert("GOFLAGS".into(), value.clone());
        recipe.inputs["environment"]["GOFLAGS"] = serde_json::json!(value);
    }
    if command == "build" {
        return build::run(&contract, recipes, &argv, deadline);
    }
    execute_registered(&contract, recipes, &argv)
}

struct GuestRequest<'a> {
    command: &'a str,
    command_index: usize,
    original: &'a [OsString],
    cwd: PathBuf,
    repo: Option<crate::config::repo::Repo>,
    parsed: classify::Invocation,
    build_flags: Vec<String>,
    packages: Vec<Package>,
    deadline: Instant,
}

fn classify_request<'a>(
    contract: &Contract,
    arguments: &'a [OsString],
    goflags: &OsStr,
) -> Result<Option<GuestRequest<'a>>, String> {
    classify_request_with_environment(contract, arguments, goflags, &std::env::vars_os().collect())
}

fn classify_request_with_environment<'a>(
    contract: &Contract,
    arguments: &'a [OsString],
    goflags: &OsStr,
    inherited: &std::collections::BTreeMap<OsString, OsString>,
) -> Result<Option<GuestRequest<'a>>, String> {
    let original_cwd = std::env::current_dir().map_err(|e| e.to_string())?;
    let Some((command_index, cwd)) = command_location(arguments, &original_cwd) else {
        return Ok(None);
    };
    let Some(command) = arguments[command_index].to_str() else {
        return Ok(None);
    };
    let kind = match command {
        "test" => classify::Kind::Test,
        "run" => classify::Kind::Run,
        "build" => classify::Kind::Build,
        _ => return Ok(None),
    };
    let Ok(cwd) = cwd.canonicalize() else {
        return Ok(None);
    };
    let repo = crate::config::repo::Repo::optional(&cwd)?;
    if command != "test" && repo.as_ref().is_none_or(|repo| repo.guest.is_empty()) {
        return Ok(None);
    }
    let Ok(parsed) = classify::inspect_unchecked(&arguments[command_index + 1..], goflags, kind)
    else {
        return Ok(None);
    };
    if command == "test" {
        classify::inspect(&arguments[command_index + 1..], goflags, kind)?;
        guest_env::validate()?;
        if let Some(installation) = &contract.installation {
            let selected = manifest::resolve(Some(installation))?;
            crate::install::tags::admit(&selected.manifest.excluded_build_tags, &parsed.tags)?;
        }
    }
    let required_tags = if command != "test" {
        Vec::new()
    } else if let Some(installation) = &contract.installation {
        manifest::resolve(Some(installation))?.manifest.build_tags
    } else if repo
        .as_ref()
        .is_some_and(|repo| repo.derive == crate::config::repo::Derivation::Temporal)
    {
        crate::install::config::temporal_build_tags()
    } else {
        Vec::new()
    };
    let deadline = Instant::now() + Duration::from_secs(600);
    let build_flags = parsed.build_arguments(&arguments[command_index + 1..])?;
    let packages = packages(
        contract,
        &cwd,
        &parsed,
        &arguments[command_index + 1..],
        &build_flags,
        &required_tags,
        deadline,
        inherited,
        command == "test",
    )?;
    if command != "test" {
        let repo = repo.as_ref().ok_or("guest run/build requires gomad.toml")?;
        if packages.iter().any(|package| {
            package.name != "main"
                || !repo.guest.iter().any(|guest| {
                    repo.module_root.join(guest).canonicalize().ok().as_ref()
                        == Some(&package.directory)
                })
        }) {
            return Ok(None);
        }
    }
    parsed.admit()?;
    guest_env::validate()?;
    Ok(Some(GuestRequest {
        command,
        command_index,
        original: &arguments[command_index + 1..],
        cwd,
        repo,
        parsed,
        build_flags,
        packages,
        deadline,
    }))
}

fn prepare_guest(
    contract: &Contract,
    request: &GuestRequest<'_>,
) -> Result<Vec<driver::Recipe>, String> {
    let cwd = &request.cwd;
    let repo = request.repo.as_ref();
    let parsed = &request.parsed;
    let packages = &request.packages;
    let build_flags = &request.build_flags;
    let command = request.command;
    let deadline = request.deadline;
    let mut recipes = Vec::new();
    for package in packages {
        let installation = if let Some(explicit) = &contract.installation {
            explicit.clone()
        } else {
            let discovered;
            let repo = if let Some(repo) = repo {
                repo
            } else {
                discovered = crate::config::repo::Repo::discover(cwd)?;
                &discovered
            };
            let relative = package
                .directory
                .strip_prefix(&repo.module_root)
                .map_err(|_| "guest package escapes Go module")?;
            let source = if relative.as_os_str().is_empty() {
                ".".into()
            } else {
                format!(
                    "./{}",
                    relative.to_str().ok_or("guest package path is not UTF-8")?
                )
            };
            crate::install::derive_cache::derive(
                &contract.compiler,
                repo,
                &source,
                &cache_root()?,
                deadline,
            )?
        };
        let selected = manifest::resolve(Some(&installation))?;
        if selected.manifest.compiler_path != contract.compiler {
            return Err("shim compiler differs from installation".into());
        }
        let mut options =
            guest_options(contract, cwd, command, package, &selected.manifest.profile)?;
        options.installation = Some(installation);
        let source_arguments = if package.import_path == "command-line-arguments" {
            request
                .parsed
                .package_indices
                .iter()
                .map(|&index| {
                    request.original[index]
                        .to_str()
                        .map(str::to_owned)
                        .ok_or("Go source path is not UTF-8")
                })
                .collect::<Result<Vec<_>, _>>()?
        } else {
            Vec::new()
        };
        if let Some(first) = source_arguments.first() {
            options.source.clone_from(first);
        }
        options.build_tags = parsed
            .tags
            .iter()
            .map(|tag| {
                tag.to_str()
                    .map(str::to_owned)
                    .ok_or("build tag is not UTF-8")
            })
            .collect::<Result<_, _>>()?;
        crate::install::tags::admit(&selected.manifest.excluded_build_tags, &parsed.tags)?;
        let recipe = driver::prepare_sources(&options, build_flags, &source_arguments, deadline)?;
        recipes.push(recipe);
    }
    Ok(recipes)
}

fn guest_options(
    contract: &Contract,
    cwd: &Path,
    command: &str,
    package: &Package,
    profile: &str,
) -> Result<crate::cli::explore::ExploreOptions, String> {
    let mut ignored = false;
    let mut arguments = vec![
        "--seeds".into(),
        contract.seeds.clone(),
        "--execution-timeout".into(),
        "5m".into(),
        "--overall-timeout".into(),
        "10m".into(),
        "--artifacts".into(),
        contract
            .artifacts
            .to_str()
            .ok_or("artifact path is not UTF-8")?
            .into(),
        "--working-dir".into(),
        cwd.to_str().ok_or("working directory is not UTF-8")?.into(),
        if command == "test" {
            "go-test".into()
        } else {
            "go-run".into()
        },
        package.import_path.clone(),
    ];
    if profile == "gomad.wasi-cooperative/v1" {
        arguments.insert(0, "--choices".into());
    }
    crate::cli::explore::parse(&arguments, &mut ignored)
}

fn command_location(arguments: &[OsString], cwd: &Path) -> Option<(usize, PathBuf)> {
    use std::os::unix::ffi::{OsStrExt as _, OsStringExt as _};
    let first = arguments.first()?;
    let (index, directory) = if first == "-C" {
        let directory = arguments.get(1)?;
        (2, Some(PathBuf::from(directory)))
    } else if let Some(directory) = first.as_bytes().strip_prefix(b"-C=") {
        (
            1,
            Some(PathBuf::from(OsString::from_vec(directory.to_vec()))),
        )
    } else {
        (0, None)
    };
    if index >= arguments.len() {
        return None;
    }
    let cwd = if let Some(directory) = directory {
        cwd.join(directory)
    } else {
        cwd.to_path_buf()
    };
    Some((index, cwd))
}

fn packages(
    contract: &Contract,
    cwd: &Path,
    parsed: &classify::Invocation,
    arguments: &[OsString],
    flags: &[String],
    required_tags: &[String],
    deadline: Instant,
    inherited: &std::collections::BTreeMap<OsString, OsString>,
    guest: bool,
) -> Result<Vec<Package>, String> {
    let scratch = process::Temporary::new(&cache_root()?)?;
    let mut list = vec!["list".into(), "-e".into(), "-json".into(), "-find".into()];
    list.extend(flags.iter().cloned());
    if !parsed.tags.is_empty() || !required_tags.is_empty() {
        list.extend([
            "-tags".into(),
            parsed
                .merged_tags(required_tags)
                .into_string()
                .map_err(|_| "tag is not UTF-8")?,
        ]);
    }
    list.push("--".into());
    if parsed.package_indices.is_empty() {
        list.push(".".into());
    } else {
        for &index in &parsed.package_indices {
            list.push(
                arguments[index]
                    .to_str()
                    .ok_or("guest package is not UTF-8")?
                    .into(),
            );
        }
    }
    let raw = process::command(
        &contract.compiler,
        &list,
        cwd,
        &packages_environment(inherited, guest)?,
        &scratch.0,
        128 << 20,
        deadline,
    )?;
    let mut values = Vec::new();
    let mut names = std::collections::BTreeSet::new();
    for package in serde_json::Deserializer::from_slice(&raw).into_iter::<Package>() {
        let mut package = package.map_err(|e| e.to_string())?;
        package.directory = package
            .directory
            .canonicalize()
            .map_err(|e| e.to_string())?;
        if names.insert(package.import_path.clone()) {
            values.push(package);
        }
    }
    Ok(values)
}

fn packages_environment(
    inherited: &std::collections::BTreeMap<OsString, OsString>,
    guest: bool,
) -> Result<std::collections::BTreeMap<OsString, OsString>, String> {
    if guest {
        return Ok(process::environment()?
            .into_iter()
            .map(|(name, value)| (name.into(), value.into()))
            .collect());
    }
    let mut environment = inherited.clone();
    for (name, value) in [
        ("GOENV", "off"),
        ("GOFLAGS", ""),
        ("GOTOOLCHAIN", "local"),
        ("GOPROXY", "off"),
        ("GOSUMDB", "off"),
        ("GOPRIVATE", ""),
        ("GONOPROXY", "none"),
        ("GOVCS", "*:off"),
    ] {
        environment.insert(name.into(), value.into());
    }
    Ok(environment)
}

fn cache_root() -> Result<PathBuf, String> {
    if let Some(path) = std::env::var_os("XDG_CACHE_HOME").filter(|p| !p.is_empty()) {
        return std::path::absolute(PathBuf::from(path).join("gomad")).map_err(|e| e.to_string());
    }
    std::env::var_os("HOME")
        .map(|home| PathBuf::from(home).join(".cache/gomad"))
        .ok_or_else(|| "HOME or XDG_CACHE_HOME is required".into())
}

fn quote_go(path: &Path) -> Result<String, String> {
    let path = path.to_str().ok_or("launcher path is not UTF-8")?;
    if !path.contains('\'') {
        Ok(format!("'{path}'"))
    } else if !path.contains('"') {
        Ok(format!("\"{path}\""))
    } else {
        Err("launcher path contains both Go quote kinds".into())
    }
}

fn publish(root: &Path, name: &str, bytes: &[u8]) -> Result<(), String> {
    let mut file = fs::OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .open(root.join(name))
        .map_err(|e| e.to_string())?;
    file.write_all(bytes)
        .and_then(|()| file.sync_all())
        .map_err(|e| e.to_string())
}

fn execute_registered(
    contract: &Contract,
    recipes: Vec<driver::Recipe>,
    argv: &[OsString],
) -> Result<i32, String> {
    let invocation_id = origin::identity()?;
    let root = contract
        .directory
        .join(format!("invocation-{invocation_id}"));
    fs::DirBuilder::new()
        .mode(0o700)
        .create(&root)
        .map_err(|e| e.to_string())?;
    let encoded = argv.iter().map(|arg| origin::argument(arg)).collect();
    let shared = driver::shared::Shared::new(recipes, encoded)?;
    let raw = canonical::encode(&shared)?;
    if raw.len() > 32 << 20 {
        return Err("shared Go driver metadata exceeds existing bound".into());
    }
    publish(&root, "driver-recipe.json", &raw)?;
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
    publish(&root, "invocation.json", &bytes)?;
    let module_root = root.join("go-build");
    fs::DirBuilder::new()
        .mode(0o700)
        .create(&module_root)
        .map_err(|e| e.to_string())?;
    let mut command = guest_env::command(
        &contract.compiler,
        &argv[1..],
        &shared.packages[0].environment,
    );
    command
        .env("GOTMPDIR", &module_root)
        .env("GOMAD_SHIM_INVOCATION", root.join("invocation.json"))
        .env("GOMAD_SHIM_INVOCATION_SHA256", canonical::digest(&bytes));
    let registration = origin::DriverRegistration {
        schema: "gomad.driver-registration/v1".into(),
        process_id: std::process::id(),
        invocation_sha256: canonical::digest(&bytes),
        compiler_sha256: invocation.compiler_sha256.clone(),
        driver_argv_sha256: canonical::digest(&canonical::encode(&invocation.driver_argv)?),
        module_root,
    };
    crate::cli::run_guest::registered_exec(&root, &registration, &mut command)?;
    Err("registered Go exec unexpectedly returned".into())
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::os::unix::fs::PermissionsExt as _;

    #[test]
    fn global_directory_flag_keeps_the_real_driver_prefix_and_effective_source_directory() {
        let root = process::Temporary::new(&std::env::temp_dir()).unwrap();
        fs::create_dir(root.0.join("child")).unwrap();
        for arguments in [
            vec!["-C", "child", "test", "./pkg"],
            vec!["-C=child", "test", "./pkg"],
        ] {
            let arguments = arguments
                .into_iter()
                .map(OsString::from)
                .collect::<Vec<_>>();
            let (index, cwd) = command_location(&arguments, &root.0).unwrap();
            assert_eq!(arguments[index], "test");
            assert_eq!(cwd, root.0.join("child").canonicalize().unwrap());
        }
    }

    #[test]
    fn excluded_effective_tags_refuse_before_any_metadata_compiler_process() {
        let root = process::Temporary::new(&std::env::temp_dir()).unwrap();
        let compiler = root.0.join("go");
        let counter = root.0.join("compiler-calls");
        let metadata =
            serde_json::json!({"ImportPath":"example.test/fixture","Dir":root.0,"Name":"fixture"});
        fs::write(
            &compiler,
            format!(
                "#!/bin/sh\nprintf x >> '{}'\n/bin/cat <<'GOMAD_METADATA'\n{}\nGOMAD_METADATA\n",
                counter.display(),
                metadata
            ),
        )
        .unwrap();
        fs::set_permissions(&compiler, fs::Permissions::from_mode(0o700)).unwrap();
        let config =
            crate::install::config::build("gomad.wasi-stock-exploratory/v1", None, None).unwrap();
        let installed = manifest::Installation {
            schema: manifest::SCHEMA.into(),
            admission: None,
            compiler_path: compiler.clone(),
            compiler_sha256: canonical::digest(&fs::read(&compiler).unwrap()),
            go_version: "go1.27.1".into(),
            runtime_root: PathBuf::new(),
            overlay_module_root: None,
            cache_root: root.0.clone(),
            profile: "gomad.wasi-stock-exploratory/v1".into(),
            fuel: 1 << 50,
            memory_bytes: 1 << 32,
            helper_sha256: gomad_wasmhost::linked::IMPLEMENTATION_SHA256.into(),
            host_sha256: env!("GOMAD_HOST_IMPLEMENTATION_SHA256").into(),
            runtime_implementation_sha256: None,
            model: config.model,
            namespace: config.namespace,
            storage: config.storage,
            network: config.network,
            build_tags: config.build_tags,
            excluded_build_tags: vec!["blocked".into()],
        };
        let installation = root.0.join("installation.json");
        manifest::write(&installation, &installed).unwrap();
        let contract = Contract {
            binary: std::env::current_exe().unwrap(),
            compiler,
            installation: Some(installation),
            artifacts: root.0.join("artifacts"),
            seeds: "1".into(),
            run: "1".repeat(64),
            directory: root.0.clone(),
        };
        for (arguments, goflags) in [
            (vec!["test", "-tags=blocked", "./pkg"], ""),
            (vec!["test", "./pkg"], "-tags=blocked"),
        ] {
            let arguments = arguments
                .into_iter()
                .map(OsString::from)
                .collect::<Vec<_>>();
            let error = classify_request(&contract, &arguments, OsStr::new(goflags))
                .err()
                .unwrap();
            assert_eq!(error, "gomad: unsupported guest build tag \"blocked\"");
            assert!(!counter.exists());
        }
        let arguments =
            ["test", "-tags=admitted", "./pkg", "-args", "-tags=blocked"].map(OsString::from);
        let request = classify_request(&contract, &arguments, OsStr::new("-tags=blocked"))
            .unwrap()
            .unwrap();
        assert_eq!(request.parsed.tags, [OsString::from("admitted")]);
        assert_eq!(fs::read(&counter).unwrap(), b"x");
    }

    fn fixture_contract(root: &Path, compiler: PathBuf) -> Contract {
        Contract {
            binary: std::env::current_exe().unwrap(),
            compiler,
            installation: None,
            artifacts: root.join("artifacts"),
            seeds: "1".into(),
            run: "1".repeat(64),
            directory: root.to_path_buf(),
        }
    }

    fn fixture_installation(root: &Path, compiler: &Path, profile: &str) -> PathBuf {
        let config = crate::install::config::build(profile, None, None).unwrap();
        let cooperative = profile == "gomad.wasi-cooperative/v1";
        let installed = manifest::Installation {
            schema: manifest::SCHEMA.into(),
            admission: None,
            compiler_path: compiler.to_path_buf(),
            compiler_sha256: canonical::digest(&fs::read(compiler).unwrap()),
            go_version: "go1.27.1".into(),
            runtime_root: if cooperative {
                root.to_path_buf()
            } else {
                PathBuf::new()
            },
            overlay_module_root: None,
            cache_root: root.to_path_buf(),
            profile: profile.into(),
            fuel: 1 << 50,
            memory_bytes: 1 << 32,
            helper_sha256: gomad_wasmhost::linked::IMPLEMENTATION_SHA256.into(),
            host_sha256: env!("GOMAD_HOST_IMPLEMENTATION_SHA256").into(),
            runtime_implementation_sha256: cooperative
                .then(|| env!("GOMAD_RUNTIME_IMPLEMENTATION_SHA256").into()),
            model: config.model,
            namespace: config.namespace,
            storage: config.storage,
            network: config.network,
            build_tags: config.build_tags,
            excluded_build_tags: Vec::new(),
        };
        let path = root.join("installation.json");
        manifest::write(&path, &installed).unwrap();
        path
    }

    #[test]
    fn shim_options_match_stock_and_cooperative_driver_profile_admission() {
        let root = process::Temporary::new(&std::env::temp_dir()).unwrap();
        fs::write(root.0.join("go.mod"), "module example.test/fixture\n").unwrap();
        let compiler = root.0.join("go");
        fs::write(
            &compiler,
            "#!/bin/sh\nprintf 'reached-version-after-profile-admission' >&2\nexit 1\n",
        )
        .unwrap();
        fs::set_permissions(&compiler, fs::Permissions::from_mode(0o700)).unwrap();
        let contract = fixture_contract(&root.0, compiler.clone());
        let package = Package {
            import_path: "example.test/fixture".into(),
            directory: root.0.clone(),
            name: "main".into(),
        };
        for profile in [
            "gomad.wasi-stock-exploratory/v1",
            "gomad.wasi-cooperative/v1",
        ] {
            let mut options = guest_options(&contract, &root.0, "test", &package, profile).unwrap();
            options.installation = Some(fixture_installation(&root.0, &compiler, profile));
            let error = driver::prepare(&options, Instant::now() + Duration::from_secs(10))
                .err()
                .unwrap();
            assert!(
                error.contains("reached-version-after-profile-admission"),
                "{profile}: {error}"
            );
            options.choices = !options.choices;
            let error = driver::prepare(&options, Instant::now() + Duration::from_secs(10))
                .err()
                .unwrap();
            assert!(
                error.contains("choices/scenario do not match installed profile"),
                "{profile}: {error}"
            );
        }
    }

    #[test]
    fn unmatched_race_run_and_build_use_host_metadata_then_preserve_passthrough() {
        use std::os::unix::ffi::OsStringExt as _;
        let root = process::Temporary::new(&std::env::temp_dir()).unwrap();
        for directory in ["host", "opted"] {
            fs::create_dir(root.0.join(directory)).unwrap();
        }
        fs::write(root.0.join("go.mod"), "module example.test/fixture\n").unwrap();
        fs::write(
            root.0.join("gomad.toml"),
            "runtime_root = \".\"\nderive = \"default\"\nguest = [\"./opted\"]\n",
        )
        .unwrap();
        let host = serde_json::json!({"ImportPath":"example.test/fixture/host","Dir":root.0.join("host"),"Name":"main"});
        let opted = serde_json::json!({"ImportPath":"example.test/fixture/opted","Dir":root.0.join("opted"),"Name":"main"});
        let compiler = root.0.join("go");
        fs::write(
            &compiler,
            format!(
                r#"#!/bin/sh
[ "$1" = list ] || exit 90
race=0
for arg do
    case "$arg" in -race=true) race=$((race + 1));; esac
    last="$arg"
done
[ "$race" = 1 ] || {{ printf 'race must be expanded exactly once' >&2; exit 91; }}
[ -z "$GOFLAGS" ] || {{ printf 'GOFLAGS would expand twice' >&2; exit 92; }}
if [ "$GOOS" = wasip1 ]; then printf 'go: -race is not supported on wasip1/wasm' >&2; exit 2; fi
[ "$GOOS/$GOARCH/$CGO_ENABLED" = linux/arm64/1 ] || exit 93
[ "$GOTOOLCHAIN/$GOPROXY/$GOSUMDB/$GOVCS" = 'local/off/off/*:off' ] || exit 94
case "$last" in ./opted) printf '%s' '{opted}';; *) printf '%s' '{host}';; esac
"#
            ),
        )
        .unwrap();
        fs::set_permissions(&compiler, fs::Permissions::from_mode(0o700)).unwrap();
        let contract = fixture_contract(&root.0, compiler);
        let mut inherited = std::env::vars_os().collect::<std::collections::BTreeMap<_, _>>();
        for (name, value) in [
            ("GOOS", "linux"),
            ("GOARCH", "arm64"),
            ("CGO_ENABLED", "1"),
            ("GOFLAGS", "-race"),
        ] {
            inherited.insert(name.into(), value.into());
        }
        let opaque = OsString::from_vec(vec![b'x', 0xff]);
        inherited.insert("OPAQUE_HOST_VALUE".into(), opaque.clone());
        for command in ["run", "build"] {
            let arguments = vec![
                "-C".into(),
                root.0.as_os_str().to_owned(),
                command.into(),
                "./host".into(),
            ];
            let original = arguments.clone();
            assert!(
                classify_request_with_environment(
                    &contract,
                    &arguments,
                    OsStr::new("-race"),
                    &inherited
                )
                .unwrap()
                .is_none()
            );
            assert_eq!(arguments, original);
            assert_eq!(inherited[OsStr::new("GOFLAGS")], "-race");
            let opted = vec![
                "-C".into(),
                root.0.as_os_str().to_owned(),
                command.into(),
                "./opted".into(),
            ];
            let error = classify_request_with_environment(
                &contract,
                &opted,
                OsStr::new("-race"),
                &inherited,
            )
            .err()
            .unwrap();
            assert_eq!(error, "gomad: unsupported guest option -race");
        }
        let environment = packages_environment(&inherited, false).unwrap();
        assert_eq!(environment[OsStr::new("OPAQUE_HOST_VALUE")], opaque);
        let guest = packages_environment(&inherited, true).unwrap();
        assert_eq!(guest[OsStr::new("GOOS")], "wasip1");
        assert_eq!(guest[OsStr::new("GOARCH")], "wasm");
        assert_eq!(guest[OsStr::new("CGO_ENABLED")], "0");
    }
}
