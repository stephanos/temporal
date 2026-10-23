//! Compilation tuples and execution requests from the immutable pre-rewrite producers.
//! The terminal backend's preparation configuration is deliberately separate from its later request.
use serde_json::{Value, json};
use std::{
    path::{Path, PathBuf},
    time::Duration,
};

const COOPERATIVE: &str = "gomad.wasi-cooperative/v1";
const ORIGINAL_TEMPORAL_TAGS: [&str; 3] = ["disable_grpc_modules", "test_dep", "sqlite3_dotlk"];

pub(crate) struct ModuleRecipe {
    pub kind: &'static str,
    pub source: String,
    pub directory: PathBuf,
    pub build_tags: Vec<String>,
    pub target_arguments: Vec<String>,
}

pub(crate) struct PreparationRecipe {
    pub module: ModuleRecipe,
    pub configuration: Value,
    pub memory_bytes: u64,
    pub fuel: u64,
}

pub(crate) struct RequestRecipe {
    pub configuration: Value,
    pub memory_bytes: u64,
    pub fuel: u64,
    pub timeout: Duration,
    pub choice_bytes: u64,
    pub diagnostics: bool,
    pub digest_transcript: bool,
}

pub(crate) fn temporal_test(directory: &Path, source: &str) -> ModuleRecipe {
    ModuleRecipe {
        kind: "go-test",
        source: source.into(),
        directory: directory.into(),
        build_tags: ORIGINAL_TEMPORAL_TAGS
            .into_iter()
            .map(str::to_owned)
            .collect(),
        target_arguments: Vec::new(),
    }
}

pub(crate) fn derived_temporal_test(directory: &Path, source: &str) -> ModuleRecipe {
    let mut recipe = temporal_test(directory, source);
    recipe.build_tags = ["test_dep", "disable_grpc_modules", "sqlite3_dotlk"]
        .into_iter()
        .map(str::to_owned)
        .collect();
    recipe
}

pub(crate) fn slim_host_owner(repo: &Path) -> ModuleRecipe {
    let mut recipe = temporal_test(repo, "./tests");
    recipe.build_tags = [
        "disable_grpc_modules",
        "gomad_wasm_slim",
        "sqlite3_dotlk",
        "test_dep",
    ]
    .into_iter()
    .map(str::to_owned)
    .collect();
    recipe
}

fn integration_configuration() -> Result<Value, String> {
    let mut configuration = crate::install::config::build(COOPERATIVE, None, None)?.original;
    configuration["Limits"] = json!({"OutputBytes":1<<20,"TranscriptBytes":8<<20,"Calls":10_000,"Descriptors":64,"Files":1000,"FilesystemBytes":8<<20,"PendingEvents":4096});
    Ok(configuration)
}

/// The original TestPrepareTemporalTerminalExitModule used integrationProvider defaults,
/// overriding only profile, clock step, memory and fuel; its target Spec had no Args.
pub(crate) fn terminal_exit(repo: &Path) -> Result<PreparationRecipe, String> {
    let configuration = integration_configuration()?;
    Ok(PreparationRecipe {
        module: temporal_test(repo, "./tests"),
        configuration,
        memory_bytes: 4 << 30,
        fuel: 1 << 50,
    })
}

pub(crate) fn runtime_cost(repo: &Path) -> ModuleRecipe {
    ModuleRecipe {
        kind: "go-run",
        source: "./internal/model/wasi/testdata/runtime_cost".into(),
        directory: repo.join("tools/gomad"),
        build_tags: Vec::new(),
        target_arguments: Vec::new(),
    }
}

pub(crate) fn diagnostic_probe(repo: &Path) -> ModuleRecipe {
    ModuleRecipe {
        source: "./internal/backends/wasm/toolchain/testdata/choices".into(),
        ..runtime_cost(repo)
    }
}

/// Original prototypeRequest used the full Temporal prototype policy, not installation networking.
pub(crate) fn temporal_request(
    repo: &Path,
    package: &str,
    arguments: &[String],
    timeout: Duration,
) -> Result<RequestRecipe, String> {
    let mut configuration = crate::install::config::temporal_policy(
        repo,
        package,
        crate::install::config::TemporalPolicy::Qualification,
    )?;
    let mut args = vec![
        "temporal.test".into(),
        "-test.v".into(),
        "-test.count=1".into(),
        "-test.timeout=72h".into(),
    ];
    args.extend_from_slice(arguments);
    configuration["Args"] = json!(args);
    Ok(RequestRecipe {
        configuration,
        memory_bytes: 4 << 30,
        fuel: 1 << 50,
        timeout,
        choice_bytes: 1 << 30,
        diagnostics: false,
        digest_transcript: true,
    })
}

/// Original cooperativeModuleRequest, also used by the diagnostic choices feature probe.
pub(crate) fn diagnostic_probe_request() -> Result<RequestRecipe, String> {
    let mut configuration = integration_configuration()?;
    configuration["Args"] = json!(["guest", "literal $arg"]);
    configuration["Environment"] = json!(["PWD=/workspace", "TZ=UTC", "ONLY=captured"]);
    let mut entropy = [0_u8; 32];
    entropy[..3].copy_from_slice(&[1, 2, 3]);
    configuration["EntropyKey"] = json!(entropy);
    configuration["Clock"]["ReadStepNanos"] = json!(1000);
    Ok(RequestRecipe {
        configuration,
        memory_bytes: 256 << 20,
        fuel: 4_000_000_000,
        timeout: Duration::from_secs(60),
        choice_bytes: 1 << 20,
        diagnostics: true,
        digest_transcript: false,
    })
}

/// measureRuntimeCostProfile starts with cooperativeModuleRequest, then applies these overrides.
pub(crate) fn runtime_request() -> Result<RequestRecipe, String> {
    let mut request = diagnostic_probe_request()?;
    request.configuration["Clock"]["ReadStepNanos"] = json!(0);
    request.configuration["Limits"]["Calls"] = json!(1 << 20);
    request.configuration["Limits"]["TranscriptBytes"] = json!(64 << 20);
    request.fuel = 1 << 40;
    request.timeout = Duration::from_secs(180);
    request.choice_bytes = 64 << 20;
    request.diagnostics = false;
    Ok(request)
}

#[cfg(test)]
#[path = "module_recipes_tests.rs"]
mod tests;
