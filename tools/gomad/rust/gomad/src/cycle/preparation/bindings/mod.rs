#![cfg_attr(
    not(test),
    allow(dead_code, reason = "Cycle route integration belongs to B15")
)]
pub(crate) mod cells;
#[cfg(test)]
mod owner_cases;
pub(crate) mod public;
mod publication;
pub(crate) mod roles;
#[cfg(test)]
mod tests;

pub(crate) use super::inputs::PreparedInputs;
use super::{composition::StageBindings, inputs};
use crate::cycle::{
    config::{Configured, TrustedContext},
    custody::{self, Snapshot},
    model::{Adapter, Mode, Plan, PlanConfig},
    plan,
    process::Runner,
};
pub(crate) use cells::cell_spec;
pub(crate) use roles::CandidateRoles;
use serde::{Deserialize, Serialize};
use serde_json::{Value, json};
use std::{collections::BTreeMap, path::Path, sync::atomic::AtomicBool};

const BUNDLE_SCHEMA: &str = "gomad.cycle-frozen-bindings/v1";
const MAX_BUNDLE: u64 = 64 << 20;

/// Trusted paths and process authority come from startup, never from a decoded configuration.
pub(crate) struct FreezeContext<'a> {
    pub output: &'a Path,
    pub trusted: TrustedContext<'a>,
    pub runner: &'a mut Runner,
    pub cancelled: &'a AtomicBool,
}
#[derive(Debug, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
struct Bundle {
    schema: String,
    prepared: PreparedInputs,
    roles: CandidateRoles,
    stages: StageBindings,
    source: Snapshot,
}

pub(crate) struct FrozenConfiguration {
    pub configured: Configured,
    pub plan: Plan,
    bundle: Bundle,
}
impl FrozenConfiguration {
    /// Independently reopens the persisted configuration, descriptors, source and every role.
    pub(crate) fn open(
        path: &Path,
        expected_sha256: &str,
        context: &mut FreezeContext<'_>,
    ) -> Result<Self, String> {
        let configured = Configured::read_with_context(path, expected_sha256, &context.trusted)?;
        let value = configured.value();
        let descriptor = roles::path(value, "frozen_bindings")?;
        let bytes = custody::read_file(descriptor, MAX_BUNDLE)?;
        require(
            value["files"][descriptor.to_str().ok_or("binding descriptor path UTF8")?]
                == custody::digest(&bytes),
            "frozen descriptor is not sealed",
        )?;
        let bundle: Bundle =
            serde_json::from_value(custody::strict_json(&bytes)?).map_err(|e| e.to_string())?;
        authenticate_bundle(&bundle, value, context)?;
        let plan = plan::derive(
            &PlanConfig {
                binary: roles::path(value, "binary")?.into(),
                config_path: path.into(),
                config_sha256: expected_sha256.into(),
            },
            Mode::Parallel,
        )?;
        Ok(Self {
            configured,
            plan,
            bundle,
        })
    }
    pub(crate) fn recheck(&self, context: &mut FreezeContext<'_>) -> Result<(), String> {
        self.configured.recheck()?;
        authenticate_bundle(&self.bundle, self.configured.value(), context)
    }
}
fn private_context(runner: &Runner) -> Result<String, String> {
    runner
        .private_context()
        .filter(|value| !value.is_empty())
        .map(str::to_owned)
        .ok_or_else(|| "binding operations require a bound private observation context".into())
}
fn require(ok: bool, error: &str) -> Result<(), String> {
    if ok { Ok(()) } else { Err(error.into()) }
}
fn merge(
    files: &mut BTreeMap<String, String>,
    more: &BTreeMap<String, String>,
) -> Result<(), String> {
    for (name, sha) in more {
        if files
            .insert(name.clone(), sha.clone())
            .is_some_and(|old| old != *sha)
        {
            return Err("conflicting frozen input digest".into());
        }
    }
    Ok(())
}
fn check_prepared(prepared: &PreparedInputs, config: &Value) -> Result<(), String> {
    require(
        !prepared.compiler.is_empty() && !prepared.sources.is_empty(),
        "empty prepared compiler/source inventory",
    )?;
    custody::check_files(&prepared.compiler)?;
    let physical: BTreeMap<_, _> = prepared
        .sources
        .iter()
        .filter(|(name, _)| Path::new(name).is_absolute())
        .map(|(n, s)| (n.clone(), s.clone()))
        .collect();
    if !physical.is_empty() {
        custody::check_files(&physical)?;
    }
    let virtual_sources = prepared.sources.len() != physical.len();
    if let Some(receiving) = config.get("preparation_receiving") {
        let mut compiler = BTreeMap::new();
        let mut sources = BTreeMap::new();
        let receiving = receiving
            .as_array()
            .filter(|v| !v.is_empty())
            .ok_or("empty preparation receiving descriptors")?;
        for row in receiving {
            roles::fields(row, &["artifact", "installation", "inputs"], &[])?;
            let installation = roles::read(roles::path(row, "installation")?)?;
            let actual = inputs::verify_preparation(
                config,
                roles::path(row, "artifact")?,
                &installation,
                &row["inputs"],
                None,
            )?;
            merge(&mut compiler, &actual.compiler)?;
            merge(&mut sources, &actual.sources)?;
        }
        require(
            compiler == prepared.compiler && sources == prepared.sources,
            "independently reopened preparation inventory differs",
        )?;
    } else {
        require(
            !virtual_sources,
            "virtual prepared source requires independent preparation receiving",
        )?;
    }
    let sealed: BTreeMap<String, String> =
        serde_json::from_value(config["files"].clone()).map_err(|e| e.to_string())?;
    require(
        prepared
            .compiler
            .iter()
            .chain(physical.iter())
            .all(|(n, s)| sealed.get(n) == Some(s)),
        "prepared input is not sealed",
    )
}
fn check_source(
    source: &Snapshot,
    config: &Value,
    context: &mut FreezeContext<'_>,
) -> Result<(), String> {
    require(
        config["repo"].as_str() == context.trusted.repo_root.to_str(),
        "wrong freeze checkout",
    )?;
    require(
        config["head"] == source.head
            && config["symlinks"] == json!(source.symlinks)
            && config["source_sha256"] == custody::digest(&source.manifest()),
        "frozen source identity differs",
    )?;
    let private = private_context(context.runner)?;
    source.verify(
        context.trusted.repo_root,
        context.runner,
        context.cancelled,
        &private,
    )?;
    let manifest = custody::read_file(roles::path(config, "source_manifest")?, 128 << 20)?;
    require(
        manifest == source.manifest(),
        "frozen source manifest differs",
    )
}
const CONSUMED_LIVE_ROLES: [(plan::leaf::Binding, &str); 3] = [
    (plan::leaf::Binding::HostTests, "ordinary-host-test"),
    (plan::leaf::Binding::HelperTests, "ordinary-helper-test"),
    (plan::leaf::Binding::HostFixture, "host-fixture"),
];
fn check_consumed_roles(stages: &StageBindings, roles: &CandidateRoles) -> Result<(), String> {
    for (binding, role) in CONSUMED_LIVE_ROLES {
        let consumed = stages
            .inputs
            .leaves
            .paths
            .get(&binding)
            .ok_or("consumed role executable binding missing")?;
        let authenticated = roles::path(&roles.config()["live_corpus"]["roles"][role], "binary")?;
        require(
            authenticated.to_str() == Some(consumed.as_str()),
            &format!("consumed role executable differs from authenticated {role}"),
        )?;
        let authenticated_digest = roles
            .files()
            .get(consumed)
            .ok_or("authenticated consumed role executable is not sealed")?;
        require(
            stages.files.get(consumed) == Some(authenticated_digest),
            "consumed role executable composition seal differs from authenticated role",
        )?;
    }
    Ok(())
}
fn cell_inputs(config: &Value, files: &mut BTreeMap<String, String>) -> Result<(), String> {
    let root = roles::path(config, "root")?;
    for (label, entry) in config["stages"].as_object().ok_or("missing fixed stages")? {
        if entry["adapter"] != json!(Adapter::Qualify) {
            continue;
        }
        let path = roles::path(entry, "spec")?;
        let spec = roles::read(path)?;
        let manifest = roles::path(&spec["Manifest"], "IntegratedSourceManifestPath")?;
        let bytes = custody::read_file(manifest, 128 << 20)?;
        require(
            config["source_sha256"] == custody::digest(&bytes),
            "cell source manifest differs from frozen source",
        )?;
        let expected = cell_spec(
            label,
            &root.join("cells").join(label),
            manifest,
            config["source_sha256"]
                .as_str()
                .ok_or("source identity missing")?,
            config,
        )?;
        require(spec == expected, "changed fixed cell specification")?;
        roles::pin(files, path)?;
        roles::pin(files, manifest)?;
    }
    if let Some(support) = config.get("fn222") {
        let argv: Vec<String> =
            serde_json::from_value(support["argv"].clone()).map_err(|e| e.to_string())?;
        for flag in ["--support", "--captured-config"] {
            let i = argv
                .iter()
                .position(|value| value == flag)
                .ok_or("fn222 receiver and captured policy inputs must be explicit")?;
            let path = Path::new(argv.get(i + 1).ok_or("missing fn222 receiver input")?);
            roles::pin(files, path)?;
        }
    }
    Ok(())
}
fn terminal_request(config: &Value) -> Result<Value, String> {
    let primary = &config["primary"];
    let compiler = roles::path(primary, "compiler")?;
    let cache = roles::path(primary, "cache")?;
    let runtime = roles::path(primary, "runtime_root")?;
    require(
        runtime == roles::path(config, "repo")?.join("tools/gomad/internal/model/runtimesource"),
        "terminal runtime differs from candidate",
    )?;
    require(
        !cache.starts_with(roles::path(config, "repo")?) && cache.is_dir(),
        "terminal compiler cache must be external",
    )?;
    let mut shared_inventory = None;
    for name in ["cooperative-runtime", "onebox-cooperative"] {
        let installed = roles::read(roles::path(&config["installations"], name)?)?;
        require(
            installed["compiler_path"].as_str() == compiler.to_str()
                && installed["cache_root"].as_str() == cache.to_str()
                && installed["runtime_root"].as_str() == runtime.to_str()
                && installed["profile"] == "gomad.wasi-cooperative/v1"
                && installed["go_version"] == "go1.27.1",
            "terminal installation differs from shared compiler/cache/runtime",
        )?;
        let inventory = &installed["admission"]["compiler_inputs"];
        require(
            inventory.as_object().is_some_and(|v| !v.is_empty()),
            "terminal compiler inventory required",
        )?;
        if let Some(prior) = shared_inventory {
            require(prior == *inventory, "terminal compiler closures differ")?;
        }
        shared_inventory = Some(inventory.clone());
    }
    Ok(
        json!({"schema":"gomad.rust-terminal-preparation/v1","repo":config["repo"],"stock_go":compiler,"toolchain_cache":cache,"runtime_root":runtime}),
    )
}
fn authenticate_bundle(
    bundle: &Bundle,
    config: &Value,
    context: &mut FreezeContext<'_>,
) -> Result<(), String> {
    require(
        bundle.schema == BUNDLE_SCHEMA,
        "unknown frozen bindings schema",
    )?;
    require(
        context.output == roles::path(config, "configuration_path")?,
        "frozen configuration publication differs",
    )?;
    require(
        config.get("stress").is_none() && config.get("stress_cycle").is_none(),
        "stress matrix belongs to post-freeze window",
    )?;
    bundle.stages.validate()?;
    check_consumed_roles(&bundle.stages, &bundle.roles)?;
    for name in ["stages", "final_sections", "final_reader"] {
        require(
            config[name] == bundle.stages.config()[name],
            "frozen endpoint bindings differ from current composition",
        )?;
    }
    let files: BTreeMap<String, String> =
        serde_json::from_value(config["files"].clone()).map_err(|e| e.to_string())?;
    require(
        bundle
            .source
            .files
            .iter()
            .chain(bundle.stages.files.iter())
            .chain(bundle.roles.files().iter())
            .all(|(n, s)| files.get(n) == Some(s)),
        "incomplete frozen source/role/stage closure",
    )?;
    custody::check_files(&files)?;
    let mut extra = BTreeMap::new();
    cell_inputs(config, &mut extra)?;
    require(
        extra.iter().all(|(n, s)| files.get(n) == Some(s)),
        "unsealed cell/support input",
    )?;
    let terminal = roles::path(config, "terminal_module_spec")?;
    require(
        roles::read(terminal)? == terminal_request(config)?,
        "changed terminal preparation request",
    )?;
    check_prepared(&bundle.prepared, config)?;
    check_source(&bundle.source, config, context)?;
    bundle
        .roles
        .reauthenticate(config, context.runner, context.cancelled)?;
    Ok(())
}

fn merge_roles(config: &mut Value, roles: &CandidateRoles) -> Result<(), String> {
    for name in [
        "repo",
        "binary",
        "identity",
        "build_receipt",
        "binary_sha256",
        "live_corpus",
        "native_measurement",
        "final_profiles",
        "installations",
        "primary",
        "environment",
        "public_verbs",
        "sqlite_oracle",
    ] {
        if let Some(value) = roles.config().get(name) {
            if let Some(selected) = config.get(name) {
                require(
                    selected == value,
                    "candidate role binding differs from stage composition",
                )?;
            }
            config[name] = value.clone();
        }
    }
    Ok(())
}
/// Freeze one candidate using the post-preparation `PreparedInputs` inventory.
/// The returned hash must be retained outside the writable configuration directory by admission.
pub(crate) fn freeze(
    prepared: PreparedInputs,
    roles: CandidateRoles,
    stages: StageBindings,
    context: &mut FreezeContext<'_>,
) -> Result<FrozenConfiguration, String> {
    stages.validate()?;
    check_consumed_roles(&stages, &roles)?;
    let mut config = stages.config().clone();
    merge_roles(&mut config, &roles)?;
    config["schema"] = json!("gomad.rust-qualification-driver/v1");
    config["configuration_path"] = json!(context.output);
    require(
        roles::path(&config, "repo")? == context.trusted.repo_root,
        "wrong freeze checkout",
    )?;
    require(
        context.output
            == Path::new(
                stages
                    .inputs
                    .leaves
                    .paths
                    .get(&plan::leaf::Binding::Config)
                    .ok_or("missing configuration output binding")?,
            ),
        "freeze output differs from stage binding",
    )?;
    let parent = context
        .output
        .parent()
        .ok_or("freeze output parent missing")?;
    let stem = context
        .output
        .file_name()
        .ok_or("freeze output filename missing")?
        .to_str()
        .ok_or("freeze output filename UTF8")?;
    let manifest = parent.join(format!("{stem}.source.manifest"));
    let descriptors = parent.join(format!("{stem}.bindings.json"));
    let companion = parent.join(format!("{stem}.sha256.json"));
    for path in [&manifest, &descriptors, &companion, context.output] {
        publication::fresh(path)?;
    }
    let private = private_context(context.runner)?;
    let source = Snapshot::observe(
        context.trusted.repo_root,
        context.runner,
        context.cancelled,
        &private,
    )?;
    let source_bytes = source.manifest();
    config["source_manifest"] = json!(manifest);
    config["source_sha256"] = json!(custody::digest(&source_bytes));
    config["head"] = json!(source.head);
    config["symlinks"] = json!(source.symlinks);
    let mut files = source.files.clone();
    merge(&mut files, roles.files())?;
    merge(&mut files, &stages.files)?;
    merge(&mut files, &prepared.compiler)?;
    for (name, sha) in &prepared.sources {
        if Path::new(name).is_absolute() {
            merge(&mut files, &BTreeMap::from([(name.clone(), sha.clone())]))?;
        }
    }
    cell_inputs(&config, &mut files)?;
    let terminal = std::path::PathBuf::from(
        stages
            .inputs
            .leaves
            .paths
            .get(&plan::leaf::Binding::TerminalSpec)
            .ok_or("missing terminal preparation binding")?,
    );
    publication::fresh(&terminal)?;
    publication::write(&terminal, &inputs::canonical(&terminal_request(&config)?)?)?;
    roles::pin(&mut files, &terminal)?;
    config["terminal_module_spec"] = json!(terminal);
    config["files"] = json!(files);
    roles.reauthenticate(&config, context.runner, context.cancelled)?;
    check_prepared(&prepared, &config)?;
    let bundle = Bundle {
        schema: BUNDLE_SCHEMA.into(),
        prepared,
        roles,
        stages,
        source,
    };
    publish_bundle(
        &bundle,
        config,
        files,
        (&manifest, &descriptors, &companion),
        context,
    )
}
fn publish_bundle(
    bundle: &Bundle,
    mut config: Value,
    mut files: BTreeMap<String, String>,
    paths: (&Path, &Path, &Path),
    context: &mut FreezeContext<'_>,
) -> Result<FrozenConfiguration, String> {
    let (manifest, descriptors, companion) = paths;
    let source_bytes = bundle.source.manifest();
    publication::write(manifest, &source_bytes)?;
    files.insert(
        manifest.to_str().ok_or("manifest path UTF8")?.into(),
        custody::digest(&source_bytes),
    );
    let descriptor_bytes =
        inputs::canonical(&serde_json::to_value(bundle).map_err(|e| e.to_string())?)?;
    require(
        descriptor_bytes.len() as u64 <= MAX_BUNDLE,
        "frozen descriptor byte bound",
    )?;
    publication::write(descriptors, &descriptor_bytes)?;
    files.insert(
        descriptors.to_str().ok_or("descriptor path UTF8")?.into(),
        custody::digest(&descriptor_bytes),
    );
    for stat in ["/usr/bin/stat", "/usr/bin/gnustat"] {
        if let Ok(raw) = custody::read_file(Path::new(stat), 16 << 20) {
            files.insert(stat.into(), custody::digest(&raw));
            break;
        }
    }
    config["frozen_bindings"] = json!(descriptors);
    config["files"] = json!(files);
    authenticate_bundle(bundle, &config, context)?;
    let bytes = inputs::canonical(&config)?;
    require(bytes.len() <= 16 << 20, "frozen configuration byte bound")?;
    let sha = custody::digest(&bytes);
    publication::write(context.output, &bytes)?;
    publication::write(
        companion,
        &inputs::canonical(&json!({"path":context.output,"sha256":sha}))?,
    )?;
    FrozenConfiguration::open(context.output, &sha, context)
}

#[cfg(test)]
pub(crate) fn run_cases(owner: &crate::cycle::process::CycleOwner) -> Result<(), String> {
    owner_cases::run_cases(owner)
}
