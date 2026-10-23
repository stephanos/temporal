use serde::Serialize;
use std::{
    collections::BTreeMap,
    io::Write,
    path::{Path, PathBuf},
};

pub const EXIT_FAILED: i32 = 1;
pub const EXIT_REFUSED: i32 = 2;
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct Input {
    pub variable: &'static str,
    pub required: bool,
}

#[must_use]
pub fn names() -> Vec<&'static str> {
    let mut result: Vec<_> = PRODUCERS.iter().map(|(name, _)| *name).collect();
    result.sort_unstable();
    result
}
#[must_use]
pub fn inputs(name: &str) -> Option<&'static [Input]> {
    PRODUCERS
        .iter()
        .find(|(candidate, _)| *candidate == name)
        .map(|(_, inputs)| *inputs)
}
#[must_use]
pub fn flag(variable: &str) -> String {
    if variable == "GOMAD_WASM_DIAGNOSTIC_HELPER_FEATURES" {
        return "features".into();
    }
    let name = variable.strip_prefix("GOMAD_WASM_").unwrap_or(variable);
    name.strip_prefix("GOMAD_")
        .unwrap_or(name)
        .to_lowercase()
        .replace('_', "-")
}
#[must_use]
pub fn command(
    executable: &str,
    repo: &str,
    name: &str,
    values: &BTreeMap<String, String>,
) -> Vec<String> {
    let mut result = vec![executable.into(), name.into(), "--repo".into(), repo.into()];
    for (key, value) in values {
        if key.starts_with("GOMAD_") && !value.is_empty() {
            result.extend([format!("--{}", flag(key)), value.clone()]);
        }
    }
    result
}
#[must_use]
pub fn environment(values: &BTreeMap<String, String>) -> BTreeMap<String, String> {
    values
        .iter()
        .filter(|(key, _)| !key.starts_with("GOMAD_"))
        .map(|(key, value)| (key.clone(), value.clone()))
        .collect()
}
#[must_use]
pub fn shell(command: &[String]) -> String {
    command
        .iter()
        .map(|word| {
            if !word.is_empty()
                && word
                    .bytes()
                    .all(|c| c.is_ascii_alphanumeric() || b"/-_.=:,".contains(&c))
            {
                word.clone()
            } else {
                format!("'{}'", word.replace('\'', "'\\''"))
            }
        })
        .collect::<Vec<_>>()
        .join(" ")
}
#[derive(Debug, Serialize, PartialEq, Eq)]
#[serde(untagged)]
pub enum RerunCommand {
    Argv(Vec<String>),
    Shell(String),
}
#[derive(Debug, Serialize, PartialEq, Eq)]
pub struct RerunRecord {
    pub working_directory: PathBuf,
    pub environment: BTreeMap<String, String>,
    pub command: RerunCommand,
}
#[must_use]
pub fn rerun_record(
    executable: &str,
    repo: &str,
    name: &str,
    values: &BTreeMap<String, String>,
    as_shell: bool,
) -> RerunRecord {
    let mut env = environment(values);
    for (key, value) in [
        ("GOENV", "off"),
        ("GOTOOLCHAIN", "local"),
        ("GOWORK", "off"),
        ("GOROOT", ""),
    ] {
        env.entry(key.into()).or_insert_with(|| value.into());
    }
    let argv = command(executable, repo, name, values);
    RerunRecord {
        working_directory: Path::new(repo).join("tools/gomad"),
        environment: env,
        command: if as_shell {
            RerunCommand::Shell(format!(
                "env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off {}",
                shell(&argv)
            ))
        } else {
            RerunCommand::Argv(argv)
        },
    }
}
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Invocation {
    pub name: String,
    pub repo: PathBuf,
    pub go_command: String,
    pub inputs: BTreeMap<String, String>,
    pub environ: Vec<String>,
    pub keep: bool,
}

fn status(output: &mut dyn Write, code: i32, message: &str) -> i32 {
    if output.write_all(message.as_bytes()).is_err() {
        EXIT_FAILED
    } else {
        code
    }
}
/// # Errors
/// Returns an error if writing usage to the output fails.
///
/// # Panics
/// Panics if the producer registry returns a name without its input specification.
pub fn usage(output: &mut dyn Write) -> std::io::Result<()> {
    writeln!(
        output,
        "usage: gomad-measure <measurement> --repo <absolute checkout> [--go <go>] [inputs]\nmeasurements:\n"
    )?;
    for name in names() {
        let required = inputs(name)
            .unwrap()
            .iter()
            .filter(|input| input.required)
            .map(|input| format!("--{}", flag(input.variable)))
            .collect::<Vec<_>>()
            .join(" ");
        writeln!(output, "  {name:<30} {required}")?;
    }
    Ok(())
}

#[expect(
    clippy::too_many_lines,
    reason = "Keep CLI parsing and validation precedence together."
)]
/// # Panics
/// Panics if a validated producer flag has no matching input specification.
pub fn run<F>(
    args: &[String],
    environ: &[String],
    stdout: &mut dyn Write,
    stderr: &mut dyn Write,
    start: F,
) -> i32
where
    F: FnOnce(Invocation, &mut dyn Write, &mut dyn Write) -> i32,
{
    if args.is_empty() || matches!(args[0].as_str(), "-h" | "--help" | "help") {
        return if usage(stdout).is_err() {
            EXIT_FAILED
        } else if args.is_empty() {
            EXIT_REFUSED
        } else {
            0
        };
    }
    let name = &args[0];
    let Some(spec) = inputs(name) else {
        if status(
            stderr,
            EXIT_REFUSED,
            &format!("gomad-measure: unknown measurement {name:?}\n"),
        ) == EXIT_FAILED
            || usage(stderr).is_err()
        {
            return EXIT_FAILED;
        }
        return EXIT_REFUSED;
    };
    for entry in environ {
        let key = entry.split('=').next().unwrap_or("");
        if key.starts_with("GOMAD_") {
            return status(
                stderr,
                EXIT_REFUSED,
                &format!("gomad-measure: unset {key}; pass inputs as flags\n"),
            );
        }
    }
    let mut values = BTreeMap::new();
    let mut repo = String::new();
    let mut go_command = "go".to_string();
    let mut keep = false;
    let mut index = 1;
    while index < args.len() {
        let argument = &args[index];
        if argument == "--" {
            index += 1;
            break;
        }
        if argument == "-" || !argument.starts_with('-') {
            break;
        }
        if argument.starts_with("---") {
            return status(stderr, EXIT_REFUSED, "bad flag syntax\n");
        }
        let option = argument.trim_start_matches('-');
        let (key, inline) = option
            .split_once('=')
            .map_or((option, None), |(key, value)| (key, Some(value)));
        if key == "keep" {
            keep = match inline {
                None | Some("true" | "1" | "t" | "TRUE" | "True" | "T") => true,
                Some("false" | "0" | "f" | "FALSE" | "False" | "F") => false,
                _ => return status(stderr, EXIT_REFUSED, "invalid value for flag -keep\n"),
            };
            index += 1;
            continue;
        }
        if key != "repo" && key != "go" && !spec.iter().any(|input| flag(input.variable) == key) {
            return status(
                stderr,
                EXIT_REFUSED,
                &format!("flag provided but not defined: -{key}\n"),
            );
        }
        let value = if let Some(value) = inline {
            value.to_owned()
        } else {
            index += 1;
            let Some(value) = args.get(index) else {
                return status(
                    stderr,
                    EXIT_REFUSED,
                    &format!("flag needs an argument: -{key}\n"),
                );
            };
            value.clone()
        };
        match key {
            "repo" => repo = value,
            "go" => go_command = value,
            _ => {
                let variable = spec
                    .iter()
                    .find(|input| flag(input.variable) == key)
                    .unwrap()
                    .variable;
                values.insert(variable.to_owned(), value);
            }
        }
        index += 1;
    }
    if index < args.len() {
        return status(
            stderr,
            EXIT_REFUSED,
            &format!("gomad-measure: unexpected arguments {:?}\n", &args[index..]),
        );
    }
    let path = Path::new(&repo);
    if !path.is_absolute() || clean_path(path).as_os_str() != std::ffi::OsStr::new(&repo) {
        return status(
            stderr,
            EXIT_REFUSED,
            "gomad-measure: --repo must be an absolute clean checkout path\n",
        );
    }
    if !path.join("tools/gomad/rust/gomad/src/measurement").is_dir() {
        return status(
            stderr,
            EXIT_REFUSED,
            &format!(
                "gomad-measure: {repo} is not a gomad checkout with native measurement producers\n"
            ),
        );
    }
    let missing: Vec<_> = spec
        .iter()
        .filter(|input| input.required && values.get(input.variable).is_none_or(String::is_empty))
        .map(|input| format!("--{}", flag(input.variable)))
        .collect();
    if !missing.is_empty() {
        return status(
            stderr,
            EXIT_REFUSED,
            &format!(
                "gomad-measure {name}: missing required {}\n",
                missing.join(", ")
            ),
        );
    }
    values.retain(|_, value| !value.is_empty());
    start(
        Invocation {
            name: name.clone(),
            repo: path.into(),
            go_command,
            inputs: values,
            environ: environ.to_vec(),
            keep,
        },
        stdout,
        stderr,
    )
}
fn clean_path(path: &Path) -> PathBuf {
    let mut result = PathBuf::new();
    for part in path.components() {
        match part {
            std::path::Component::CurDir => {}
            std::path::Component::ParentDir => {
                result.pop();
            }
            part => result.push(part.as_os_str()),
        }
    }
    result
}

#[derive(Debug)]
pub struct CheckoutProof {
    repo: PathBuf,
}
impl CheckoutProof {
    #[must_use]
    pub fn repo(&self) -> &Path {
        &self.repo
    }
}
#[cfg(test)]
pub(super) fn fixture_checkout(repo: PathBuf) -> CheckoutProof {
    CheckoutProof { repo }
}

/// # Errors
/// Refuses a checkout with a different executor source identity or source closure.
pub fn authenticate_checkout(repo: &Path) -> Result<CheckoutProof, String> {
    let compiled = Path::new(env!("CARGO_MANIFEST_DIR")).join("src/measurement/inputs.rs");
    let selected = repo.join("tools/gomad/rust/gomad/src/measurement/inputs.rs");
    let compiled = std::fs::metadata(compiled).map_err(|_| "executor was not built from --repo")?;
    let selected = std::fs::metadata(selected).map_err(|_| "executor was not built from --repo")?;
    #[cfg(unix)]
    {
        use std::os::unix::fs::MetadataExt;
        if compiled.dev() != selected.dev() || compiled.ino() != selected.ino() {
            return Err("executor was not built from --repo".into());
        }
    }
    #[cfg(not(unix))]
    {
        return Err(
            "native measurement checkout authentication requires Unix file identity".into(),
        );
    }
    authenticate_source_closure(&repo.join("tools/gomad/rust/gomad"), SOURCE_INPUTS)?;
    authenticate_source_closure(
        &repo.join("tools/gomad/rust/wasmhost"),
        gomad_wasmhost::linked::SOURCE_INPUTS,
    )?;
    Ok(CheckoutProof { repo: repo.into() })
}

pub const SOURCE_INPUTS: &str =
    include_str!(concat!(env!("OUT_DIR"), "/measurement-source-inputs.tsv"));
fn rust_sources(
    root: &Path,
    directory: &Path,
    result: &mut std::collections::BTreeSet<String>,
) -> Result<(), String> {
    for entry in std::fs::read_dir(directory).map_err(|error| error.to_string())? {
        let entry = entry.map_err(|error| error.to_string())?;
        if entry
            .file_type()
            .map_err(|error| error.to_string())?
            .is_symlink()
        {
            return Err("native measurement Rust source symlink refused".into());
        }
        let path = entry.path();
        if path.is_dir() {
            rust_sources(root, &path, result)?;
        } else if path.extension().is_some_and(|extension| extension == "rs") {
            result.insert(
                path.strip_prefix(root)
                    .map_err(|error| error.to_string())?
                    .to_string_lossy()
                    .replace('\\', "/"),
            );
        }
    }
    Ok(())
}
/// # Errors
/// Refuses malformed manifests, unreadable sources, or changed source membership or bytes.
pub fn authenticate_source_closure(root: &Path, manifest: &str) -> Result<(), String> {
    let mut expected = std::collections::BTreeSet::new();
    let mut listed = std::collections::BTreeSet::new();
    if manifest.is_empty() {
        return Err("empty native measurement source closure".into());
    }
    for line in manifest.lines() {
        let (digest, name) = line
            .split_once('\t')
            .ok_or("malformed native measurement source closure")?;
        if digest.len() != 64
            || !digest.bytes().all(|byte| byte.is_ascii_hexdigit())
            || !listed.insert(name)
        {
            return Err("malformed native measurement source closure".into());
        }
        let bytes = std::fs::read(root.join(name))
            .map_err(|error| format!("native measurement source closure {name}: {error}"))?;
        if crate::evidence::canonical::digest(&bytes) != digest {
            return Err(format!("native measurement source closure changed: {name}"));
        }
        if name.starts_with("src/") && name.as_bytes().ends_with(b".rs") {
            expected.insert(name.to_string());
        }
    }
    let mut actual = std::collections::BTreeSet::new();
    rust_sources(root, &root.join("src"), &mut actual)?;
    if actual != expected {
        return Err("native measurement Rust source membership changed".into());
    }
    Ok(())
}

pub fn execute<F>(call: Invocation, stdout: &mut dyn Write, stderr: &mut dyn Write, start: F) -> i32
where
    F: FnOnce(Invocation, CheckoutProof, &mut dyn Write, &mut dyn Write) -> i32,
{
    match authenticate_checkout(&call.repo) {
        Ok(proof) => start(call, proof, stdout, stderr),
        Err(error) => status(stderr, EXIT_REFUSED, &format!("gomad-measure: {error}\n")),
    }
}

const PRODUCERS: &[(&str, &[Input])] = &[
    (
        "stress",
        &[
            Input {
                variable: "GOMAD_WASM_STRESS_ROOT",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_TEST_HELPER",
                required: true,
            },
            Input {
                variable: "GOMAD_STOCK_GO",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_STRESS_HOST_DESCRIPTION",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_STRESS_REPETITIONS",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_STRESS_PARALLEL",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_STRESS_CPU_WORKERS",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_STRESS_ONLY",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_STRESS_SEED",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_STRESS_AXIS",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_TOOLCHAIN_CACHE",
                required: false,
            },
        ],
    ),
    (
        "cluster-owner-profile",
        &[
            Input {
                variable: "GOMAD_STOCK_GO",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_CLUSTER_HELPER_ADMISSION",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_CLUSTER_OWNER_PROFILE_ROOT",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_CLUSTER_OWNER_REFERENCE_HELPER",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_DIAGNOSTIC_HELPER_ADMISSION",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_TEST_HELPER",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_CLUSTER_OWNER_REPETITIONS",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_CLUSTER_OWNER_SEEDS",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_TEMPORAL_RECOVERY_DIAGNOSTIC_BUDGET",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_TEMPORAL_RECOVERY_EXPECT_MODULE_SHA256",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_TOOLCHAIN_CACHE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_PARALLEL",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_HEAVY",
                required: false,
            },
        ],
    ),
    (
        "compiled-identity-owners",
        &[
            Input {
                variable: "GOMAD_WASM_DIAGNOSTIC_IDENTITY_ROOT",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_TOOLCHAIN_CACHE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_PARALLEL",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_HEAVY",
                required: false,
            },
        ],
    ),
    (
        "diagnostic-helper-build",
        &[
            Input {
                variable: "GOMAD_WASM_DIAGNOSTIC_HELPER_BUILD_ROOT",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_DIAGNOSTIC_HELPER_FEATURES",
                required: true,
            },
        ],
    ),
    (
        "diagnostic-cluster-baseline",
        &[
            Input {
                variable: "GOMAD_STOCK_GO",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_TEMPORAL_RECOVERY_ROOT",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_TEST_HELPER",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_DIAGNOSTIC_HELPER_ADMISSION",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_TEMPORAL_RECOVERY_COMPILE_CACHE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_TEMPORAL_RECOVERY_DIAGNOSTIC_BUDGET",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_TEMPORAL_RECOVERY_EXPECT_MODULE_SHA256",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_TOOLCHAIN_CACHE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_PARALLEL",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_HEAVY",
                required: false,
            },
        ],
    ),
    (
        "failing-suite-profile",
        &[
            Input {
                variable: "GOMAD_WASM_DIAGNOSTIC_PROFILE_ROOT",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_TEST_HELPER",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_MATCHED_CACHE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_MATCHED_MODULE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_MATCHED_TOOLCHAIN_ID",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROFILE_REFERENCE_CACHE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROFILE_REFERENCE_HELPER",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_TOOLCHAIN_CACHE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_PARALLEL",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_HEAVY",
                required: false,
            },
        ],
    ),
    (
        "guest-profile",
        &[
            Input {
                variable: "GOMAD_STOCK_GO",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_GUEST_PROFILE_ROOT",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_TEST_HELPER",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_GUEST_HELPER_ADMISSION",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_TOOLCHAIN_CACHE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_PARALLEL",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_HEAVY",
                required: false,
            },
        ],
    ),
    (
        "guest-variants",
        &[
            Input {
                variable: "GOMAD_STOCK_GO",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_GUEST_PROFILE_ROOT",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_TEST_HELPER",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_DIAGNOSTIC_HELPER_ADMISSION",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_FUEL_OFF_HELPER",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_FUEL_OFF_HELPER_ADMISSION",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_TOOLCHAIN_CACHE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_PARALLEL",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_HEAVY",
                required: false,
            },
        ],
    ),
    (
        "host-call-profile",
        &[
            Input {
                variable: "GOMAD_STOCK_GO",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_HOST_CALL_PROFILE_ROOT",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_TEST_HELPER",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_HOST_CALL_PROFILE_MODULE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_HOST_CALL_PROFILE_REPETITIONS",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_TOOLCHAIN_CACHE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_PARALLEL",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_HEAVY",
                required: false,
            },
        ],
    ),
    (
        "host-owner-profile",
        &[
            Input {
                variable: "GOMAD_STOCK_GO",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_HOST_OWNER_PROFILE_ROOT",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_HOST_OWNER_REFERENCE_HELPER",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_TEST_HELPER",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_HOST_OWNER_MODULE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_HOST_OWNER_REPETITIONS",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_HOST_OWNER_SEEDS",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_HOST_OWNER_SELECTION",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_TOOLCHAIN_CACHE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_PARALLEL",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_HEAVY",
                required: false,
            },
        ],
    ),
    (
        "legacy-workflow-profile",
        &[
            Input {
                variable: "GOMAD_WASM_TEST_HELPER",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_WORKFLOW_PROFILE_ROOT",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_MATCHED_CACHE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_MATCHED_MODULE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_MATCHED_TOOLCHAIN_ID",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_TOOLCHAIN_CACHE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_PARALLEL",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_HEAVY",
                required: false,
            },
        ],
    ),
    (
        "matched-release-candidate",
        &[
            Input {
                variable: "GOMAD_STOCK_GO",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_CLUSTER_MEASUREMENT_ROOT",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_TEST_HELPER",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_CLUSTER_HELPER_ADMISSION",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_TEMPORAL_RECOVERY_DIAGNOSTIC_BUDGET",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_TEMPORAL_RECOVERY_EXPECT_MODULE_SHA256",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_TOOLCHAIN_CACHE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_PARALLEL",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_HEAVY",
                required: false,
            },
        ],
    ),
    (
        "matched-suite",
        &[
            Input {
                variable: "GOMAD_STOCK_GO",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_MATCHED_BASELINE_HELPER",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_MATCHED_MODULE",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_TEST_HELPER",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_MATCHED_BASELINE_CACHE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_MATCHED_CACHE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_MATCHED_SUITE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_MATCHED_TOOLCHAIN_ID",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_MIRROR",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_ROOT",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_TOOLCHAIN_CACHE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_PARALLEL",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_HEAVY",
                required: false,
            },
        ],
    ),
    (
        "migration",
        &[
            Input {
                variable: "GOMAD_STOCK_GO",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_MIGRATION_BASELINE",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_ROOT",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_TEST_HELPER",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_MIGRATION_CACHE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_MIRROR",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_TOOLCHAIN_CACHE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_PARALLEL",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_HEAVY",
                required: false,
            },
        ],
    ),
    (
        "namespace-activity",
        &[
            Input {
                variable: "GOMAD_WASM_NAMESPACE_MEASUREMENT_ROOT",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_NAMESPACE_BASELINE_CACHE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_NAMESPACE_BASELINE_HELPER_RECEIPT",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_NAMESPACE_BASELINE_SOURCE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_NAMESPACE_CACHE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_NAMESPACE_HELPER_RECEIPT",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_NAMESPACE_MODULE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_NAMESPACE_MODULE_RECEIPT",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_NAMESPACE_STOCK_GO",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_TOOLCHAIN_CACHE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_PARALLEL",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_HEAVY",
                required: false,
            },
        ],
    ),
    (
        "namespace-sample",
        &[
            Input {
                variable: "GOMAD_WASM_NAMESPACE_SAMPLE_ROOT",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_NAMESPACE_CACHE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_NAMESPACE_COMMON",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_NAMESPACE_HELPER_RECEIPT",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_NAMESPACE_REPETITION",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_NAMESPACE_REPLAY_TAPE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_NAMESPACE_ROLE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_NAMESPACE_SAMPLE_INDEX",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_NAMESPACE_SEED",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_TOOLCHAIN_CACHE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_PARALLEL",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_HEAVY",
                required: false,
            },
        ],
    ),
    (
        "operation-retention",
        &[
            Input {
                variable: "GOMAD_WASM_OPERATION_COUNT_ROOT",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_TEST_HELPER",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_MATCHED_CACHE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_MATCHED_MODULE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_MATCHED_TOOLCHAIN_ID",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_OPERATION_CONTROL_ROOT",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_TOOLCHAIN_CACHE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_PARALLEL",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_HEAVY",
                required: false,
            },
        ],
    ),
    (
        "runtime-cost-profile",
        &[
            Input {
                variable: "GOMAD_STOCK_GO",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_PROFILE_ROOT",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_TEST_HELPER",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_TOOLCHAIN_CACHE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_PARALLEL",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_HEAVY",
                required: false,
            },
        ],
    ),
    (
        "scratch-smoke-profile",
        &[
            Input {
                variable: "GOMAD_STOCK_GO",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_ROOT",
                required: true,
            },
            Input {
                variable: "GOMAD_SCRATCH_HELPERS",
                required: false,
            },
            Input {
                variable: "GOMAD_SCRATCH_ONLY",
                required: false,
            },
            Input {
                variable: "GOMAD_SCRATCH_SKIP_PREPARE",
                required: false,
            },
            Input {
                variable: "GOMAD_SCRATCH_TRANSCRIPT",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_TOOLCHAIN_CACHE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_PARALLEL",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_HEAVY",
                required: false,
            },
        ],
    ),
    (
        "terminal-exit",
        &[
            Input {
                variable: "GOMAD_STOCK_GO",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_TERMINAL_EXIT_ROOT",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_TEST_HELPER",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_TERMINAL_EXIT_HELPER_RECEIPT",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_TERMINAL_EXIT_MODULE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_TERMINAL_EXIT_MODULE_RECEIPT",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_TOOLCHAIN_CACHE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_PARALLEL",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_HEAVY",
                required: false,
            },
        ],
    ),
    (
        "terminal-exit-helper-build",
        &[
            Input {
                variable: "GOMAD_WASM_TERMINAL_EXIT_HELPER_BUILD_ROOT",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_TOOLCHAIN_CACHE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_PARALLEL",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_HEAVY",
                required: false,
            },
        ],
    ),
    (
        "terminal-exit-helper-stage",
        &[
            Input {
                variable: "GOMAD_WASM_TERMINAL_EXIT_HELPER_STAGE_ROOT",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_TERMINAL_EXIT_HELPER_BUILD_RECEIPT",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_TOOLCHAIN_CACHE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_PARALLEL",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_HEAVY",
                required: false,
            },
        ],
    ),
    (
        "warm-preparation",
        &[
            Input {
                variable: "GOMAD_STOCK_GO",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_TEST_HELPER",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_WARM_PREPARATION_ROOT",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_WARM_BASELINE_RECEIPT",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_WARM_CACHE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_WARM_HELPER_RECEIPT",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_WARM_MODULE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_WARM_MODULE_RECEIPT",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_TOOLCHAIN_CACHE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_PARALLEL",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_HEAVY",
                required: false,
            },
        ],
    ),
    (
        "warm-preparation-helper-build",
        &[
            Input {
                variable: "GOMAD_WASM_WARM_HELPER_BUILD_ROOT",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_TOOLCHAIN_CACHE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_PARALLEL",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_HEAVY",
                required: false,
            },
        ],
    ),
    (
        "warm-preparation-prime",
        &[
            Input {
                variable: "GOMAD_STOCK_GO",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_TEST_HELPER",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_WARM_PRIME_ROOT",
                required: true,
            },
            Input {
                variable: "GOMAD_WASM_WARM_CACHE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_WARM_HELPER_RECEIPT",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_WARM_MODULE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_WARM_MODULE_RECEIPT",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_TOOLCHAIN_CACHE",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_PARALLEL",
                required: false,
            },
            Input {
                variable: "GOMAD_WASM_PROTOTYPE_HEAVY",
                required: false,
            },
        ],
    ),
];

#[cfg(test)]
#[path = "inputs_tests.rs"]
mod tests;
