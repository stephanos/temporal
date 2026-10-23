pub mod build;
pub mod cluster_driver;
pub mod evidence;
pub mod execution;
pub mod guest;
pub mod host;
pub mod inputs;
pub mod module;
pub mod namespace;
pub mod namespace_driver;
pub mod operations;
pub mod retention;
pub mod sqlitebusy;
pub mod statistics;
pub mod stress_driver;
pub mod warm;
pub mod warm_driver;
pub mod workloads;

use crate::evidence::canonical;
use gomad_wasmhost::linked::Conditions;
use serde_json::{Value, json};
use std::{io::Write as _, os::unix::fs::OpenOptionsExt};

#[must_use]
pub fn features() -> Vec<&'static str> {
    let mut result = Vec::new();
    if cfg!(feature = "helper-diagnostic-fuel-off") {
        result.push("diagnostic-fuel-off");
    }
    if cfg!(feature = "helper-diagnostic-profile") {
        result.push("diagnostic-profile");
    }
    if cfg!(feature = "helper-guest-profile") {
        result.push("guest-profile");
    }
    if cfg!(feature = "helper-stress-jitter") {
        result.push("stress-jitter");
    }
    if cfg!(feature = "helper-determinism-fixture") {
        result.push("determinism-fixture");
    }
    result.sort_unstable();
    result
}

/// # Errors
/// Refuses an unavailable executable identity rather than manufacturing a receipt.
pub fn identity() -> Result<Value, String> {
    let executable = std::env::current_exe().map_err(|error| error.to_string())?;
    let executable = std::fs::read(executable).map_err(|error| error.to_string())?;
    let role = match Conditions::compiled() {
        Conditions::Ordinary => "ordinary",
        Conditions::DiagnosticProfile => "diagnostic-profile",
        Conditions::GuestProfile => "guest-profile",
        Conditions::DiagnosticFuelOff => "diagnostic-fuel-off",
        Conditions::GuestProfileFuelOff => "guest-profile-fuel-off",
        Conditions::StressReplay => "stress-jitter",
        Conditions::DeterminismFixtureReplay => "determinism-fixture",
    };
    Ok(json!({
        "schema":"gomad.native-measurement-identity/v1",
        "role":role,"features":features(),
        "executable_sha256":canonical::digest(&executable),
        "host_implementation_sha256":env!("GOMAD_HOST_IMPLEMENTATION_SHA256"),
        "linked_implementation_sha256":gomad_wasmhost::linked::IMPLEMENTATION_SHA256,
        "runtime_implementation_sha256":env!("GOMAD_RUNTIME_IMPLEMENTATION_SHA256"),
        "configuration":gomad_wasmhost::linked::Conditions::compiled().configuration(),
        "source_inputs":{"host":inputs::SOURCE_INPUTS,"helper":gomad_wasmhost::linked::SOURCE_INPUTS}
    }))
}

/// # Errors
/// Refuses malformed binary records or unsupported-operation inputs with invalid objects/fds.
pub fn count_operations(data: &[u8], maximum: u64) -> Result<operations::Counts, String> {
    operations::count_with_unknown(data, maximum, |raw| {
        let value: Value = crate::qualify::spec::strict_with_nesting(raw, 32)?;
        let object = value.as_object().ok_or("retained input object")?;
        object
            .get("fd")
            .map(|value| {
                value
                    .as_u64()
                    .and_then(|value| u32::try_from(value).ok())
                    .ok_or_else(|| "retained fd type".into())
            })
            .transpose()
    })
}

fn root_variable(call: &inputs::Invocation) -> Result<&'static str, String> {
    inputs::inputs(&call.name)
        .and_then(|declared| {
            declared
                .iter()
                .find(|input| input.required && input.variable.ends_with("_ROOT"))
        })
        .map(|input| input.variable)
        .or_else(|| (call.name == "matched-suite").then_some("GOMAD_WASM_PROTOTYPE_ROOT"))
        .ok_or_else(|| "measurement producer has no declared output root".into())
}

fn produce(
    call: &inputs::Invocation,
    proof: &inputs::CheckoutProof,
    root: &std::path::Path,
) -> Result<std::path::PathBuf, String> {
    match call.name.as_str() {
        "diagnostic-helper-build" => {
            let features = call
                .inputs
                .get("GOMAD_WASM_DIAGNOSTIC_HELPER_FEATURES")
                .ok_or("diagnostic build features absent")?
                .split(',')
                .map(str::trim)
                .filter(|value| !value.is_empty())
                .map(str::to_owned)
                .collect();
            build::build_role(
                proof,
                root,
                &build::Role::Diagnostic(features),
                &call.environ,
            )
        }
        "terminal-exit-helper-build" => {
            build::build_role(proof, root, &build::Role::Release, &call.environ)
        }
        "warm-preparation-helper-build" => {
            build::build_role(proof, root, &build::Role::Warm, &call.environ)
        }
        "terminal-exit-helper-stage" => {
            let receipt = call
                .inputs
                .get("GOMAD_WASM_TERMINAL_EXIT_HELPER_BUILD_RECEIPT")
                .ok_or("native terminal build receipt required")?;
            build::stage_role(
                proof,
                root,
                std::path::Path::new(receipt),
                &build::Role::Release,
            )
        }
        "stress" => stress_driver::run(call, proof, root),
        "compiled-identity-owners" => build::compiled_owners(root),
        "namespace-activity" | "namespace-sample" | "operation-retention" => {
            namespace_driver::run(call, proof, root)
        }
        "cluster-owner-profile" | "matched-release-candidate" | "diagnostic-cluster-baseline" => {
            cluster_driver::run(call, proof, root)
        }
        "warm-preparation" | "warm-preparation-prime" | "terminal-exit" => {
            warm_driver::run(call, proof, root)
        }
        "runtime-cost-profile"
        | "host-owner-profile"
        | "host-call-profile"
        | "guest-profile"
        | "guest-variants"
        | "failing-suite-profile"
        | "legacy-workflow-profile"
        | "scratch-smoke-profile"
        | "matched-suite"
        | "migration" => workloads::run(call, proof, root),
        name => Err(format!(
            "refused: {name} requires its authenticated native workload preparation recipe"
        )),
    }
}

#[expect(
    clippy::needless_pass_by_value,
    reason = "Dispatch consumes the admitted invocation through the CLI callback contract"
)]
pub fn dispatch(
    call: inputs::Invocation,
    proof: inputs::CheckoutProof,
    stdout: &mut dyn std::io::Write,
    stderr: &mut dyn std::io::Write,
) -> i32 {
    let mut run = || -> Result<(), String> {
        let variable = root_variable(&call)?;
        let requested = call
            .inputs
            .get(variable)
            .ok_or_else(|| format!("missing --{}", inputs::flag(variable)))?;
        if call.name == "namespace-sample" {
            let _lease = retention::ChildLease::for_path(std::path::Path::new(requested))?
                .ok_or("namespace sample requires its authenticated parent child lease")?;
            let result =
                namespace_driver::run_sample(&call, &proof, std::path::Path::new(requested))?;
            writeln!(
                stdout,
                "{}",
                json!({"result":result,"requested_root":requested})
            )
            .map_err(|error| error.to_string())?;
            return Ok(());
        }
        let retained = retention::RetainedRun::begin(
            std::path::Path::new(requested),
            &call.name,
            &call.inputs,
            call.keep,
            &|| false,
        )?;
        let executable = std::env::current_exe().map_err(|error| error.to_string())?;
        let rerun = inputs::rerun_record(
            &executable.to_string_lossy(),
            &call.repo.to_string_lossy(),
            &call.name,
            &call.inputs,
            false,
        );
        let result = (|| {
            let mut file = std::fs::OpenOptions::new()
                .write(true)
                .create_new(true)
                .mode(0o600)
                .open(retained.path().join("rerun.json"))
                .map_err(|error| error.to_string())?;
            file.write_all(&canonical::encode(&rerun)?)
                .and_then(|()| file.sync_all())
                .map_err(|error| error.to_string())?;
            let result = produce(&call, &proof, retained.path())?;
            writeln!(
                stdout,
                "{}",
                json!({"result":result,"requested_root":requested})
            )
            .map_err(|error| error.to_string())
        })();
        let finish = retained.finish(result.is_ok(), &|| false);
        combine(result, finish)
    };
    match run() {
        Ok(()) => 0,
        Err(error) => {
            if writeln!(stderr, "gomad-measure: {error}").is_err() {
                return inputs::EXIT_FAILED;
            }
            if error.starts_with("refused:") {
                inputs::EXIT_REFUSED
            } else {
                inputs::EXIT_FAILED
            }
        }
    }
}

pub(crate) fn combine(first: Result<(), String>, second: Result<(), String>) -> Result<(), String> {
    match (first, second) {
        (Ok(()), Ok(())) => Ok(()),
        (Err(error), Ok(())) | (Ok(()), Err(error)) => Err(error),
        (Err(first), Err(second)) => Err(format!("{first}; {second}")),
    }
}

/// Internal native stress child entry; the stress owner validates compiled role and source.
/// # Errors
/// Refuses unadmitted stress roles or incomplete source and workload inputs.
pub fn stress_run(arguments: &[String], output: &mut dyn std::io::Write) -> Result<i32, String> {
    crate::stress::run(arguments, output)
}
