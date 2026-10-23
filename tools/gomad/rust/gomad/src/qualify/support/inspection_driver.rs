use super::{
    BTreeMap, BTreeSet, Candidate, FileBinding, MAXIMUM_LIST, Path, PathBuf, Value, absolute,
    bound_file, canonical, digest, executable_digest, inspection, inventory, json, spec,
};

fn text(value: &Value) -> Result<&str, String> {
    value
        .as_str()
        .ok_or_else(|| "inspection driver binding missing".into())
}
fn path(value: &Value) -> Result<PathBuf, String> {
    let path = PathBuf::from(text(value)?);
    absolute(&path)?;
    Ok(path)
}
fn pinned(binding: &FileBinding, maximum: u64) -> Result<Vec<u8>, String> {
    absolute(&binding.path)?;
    let bytes = crate::install::manifest::read(&binding.path, maximum)?;
    if !canonical::valid_digest(&binding.sha256) || canonical::digest(&bytes) != binding.sha256 {
        return Err("inspection frozen input digest changed".into());
    }
    Ok(bytes)
}
fn frozen(
    path: PathBuf,
    files: &BTreeMap<PathBuf, String>,
    maximum: u64,
) -> Result<Vec<u8>, String> {
    pinned(
        &FileBinding {
            sha256: files
                .get(&path)
                .ok_or("inspection input is not frozen")?
                .clone(),
            path,
        },
        maximum,
    )
}

pub(super) fn validate(
    receipt: &FileBinding,
    config: &FileBinding,
    executable: &FileBinding,
) -> Result<Value, String> {
    let receipt_value: Value = spec::strict(&bound_file(receipt, MAXIMUM_LIST)?)?;
    let driver: Value = spec::strict(&pinned(config, MAXIMUM_LIST)?)?;
    let candidate = candidate(&driver, &receipt_value, config, executable)?;
    inspection::validate(&receipt.path, &receipt.sha256, &candidate)?;
    Ok(json!({"validated":true,"receipt_sha256":receipt.sha256,
        "candidate_source_sha256":candidate.integrated_source_sha256,"admission":"frozen-driver"}))
}

#[expect(
    clippy::too_many_lines,
    reason = "Standalone admission binds the frozen driver, installation inventory and source preimage before the shared inspection validator"
)]
fn candidate(
    driver: &Value,
    receipt: &Value,
    config: &FileBinding,
    executable: &FileBinding,
) -> Result<Candidate, String> {
    if driver["schema"] != "gomad.rust-qualification-driver/v1"
        || driver["features"] != json!([])
        || driver["binary"] != executable.path.to_string_lossy().as_ref()
        || driver["binary_sha256"] != executable.sha256
        || driver["identity"]["host"] != env!("GOMAD_HOST_IMPLEMENTATION_SHA256")
        || driver["identity"]["helper"] != gomad_wasmhost::linked::IMPLEMENTATION_SHA256
        || receipt["private_observation_config"]
            != json!({"path":config.path,"sha256":config.sha256})
        || executable_digest(&executable.path)? != executable.sha256
    {
        return Err("inspection driver is not bound to this frozen ordinary candidate".into());
    }
    let files: BTreeMap<PathBuf, String> =
        serde_json::from_value(driver["files"].clone()).map_err(|error| error.to_string())?;
    if files.get(&executable.path) != Some(&executable.sha256) {
        return Err("inspection candidate executable is not frozen".into());
    }
    let source_root = path(&driver["repo"])?;
    let manifest_path = path(&driver["source_manifest"])?;
    let manifest = frozen(manifest_path, &files, MAXIMUM_LIST)?;
    let source_sha = canonical::digest(&manifest);
    if driver["source_sha256"] != source_sha {
        return Err("inspection driver source manifest changed".into());
    }
    let manifest =
        String::from_utf8(manifest).map_err(|_| "inspection source manifest is not UTF-8")?;
    let mut source_paths = BTreeSet::new();
    for line in manifest.lines() {
        let (sha256, name) = line
            .split_once("  ")
            .ok_or("inspection source row malformed")?;
        let member = PathBuf::from(name);
        if !member.starts_with(&source_root)
            || !source_paths.insert(member.clone())
            || files.get(&member).map(String::as_str) != Some(sha256)
            || canonical::digest(&frozen(member, &files, 512 << 20)?) != sha256
        {
            return Err("inspection source inventory differs from frozen driver".into());
        }
    }
    if source_paths.is_empty() {
        return Err("inspection source inventory is empty".into());
    }
    let installation_path = path(&driver["installations"]["onebox-cooperative"])?;
    let installation: crate::install::manifest::Installation =
        spec::strict(&frozen(installation_path, &files, 512 << 20)?)?;
    crate::install::manifest::validate_for_executable(&installation, &executable.sha256)?;
    let admission = installation
        .admission
        .as_ref()
        .ok_or("inspection installation admission absent")?;
    if installation.schema != crate::install::manifest::SCHEMA
        || installation.profile != "gomad.wasi-cooperative/v1"
        || installation.go_version != "go1.27.1"
        || installation.host_sha256 != env!("GOMAD_HOST_IMPLEMENTATION_SHA256")
        || installation.helper_sha256 != gomad_wasmhost::linked::IMPLEMENTATION_SHA256
        || installation.runtime_implementation_sha256.as_deref()
            != Some(env!("GOMAD_RUNTIME_IMPLEMENTATION_SHA256"))
        || installation.compiler_path != path(&driver["primary"]["compiler"])?
        || installation.runtime_root != path(&driver["primary"]["runtime_root"])?
        || installation.cache_root != path(&driver["primary"]["cache"])?
        || admission.host_executable_sha256 != executable.sha256
        || admission.compiler_inputs.is_empty()
        || digest(&admission.compiler_inputs)? != admission.inputs_sha256
        || digest(&admission.guest_configuration)? != admission.configuration_sha256
        || admission.compiler_inputs.get(&installation.compiler_path)
            != Some(&installation.compiler_sha256)
    {
        return Err(
            "inspection frozen installation candidate or compiler admission changed".into(),
        );
    }
    for (member, sha256) in &admission.compiler_inputs {
        if files.get(member) != Some(sha256)
            || canonical::digest(&frozen(member.clone(), &files, 512 << 20)?) != *sha256
        {
            return Err("inspection original compiler/runtime inventory is not frozen".into());
        }
    }
    let alias: inventory::Overlay = serde_json::from_value(receipt["compiler_overlay"].clone())
        .map_err(|error| error.to_string())?;
    let private = text(&receipt["private_inputs_detached"])?
        .strip_suffix("-unavailable")
        .ok_or("inspection compiler alias reference invalid")?;
    let prepared =
        crate::evidence::store::Opened::open(&path(&receipt["preparation"]["artifact"])?)?;
    prepared.authenticate_all()?;
    let provenance: Value = spec::strict(&prepared.payload("target/provenance.json", 16 << 20)?)?;
    let observed_alias = inventory::from_provenance(&provenance, Some(alias))?;
    let inventory = inventory::from_frozen_inputs(
        admission.compiler_inputs.clone(),
        installation.compiler_path.clone(),
        &observed_alias,
        &Path::new(private).join("compiler"),
    )?;
    if inventory.runtime_root != installation.runtime_root {
        return Err("inspection original runtime inventory root differs from installation".into());
    }
    Ok(Candidate {
        integrated_source_sha256: source_sha,
        integrated_source_manifest: manifest,
        source_root,
        executable: executable.clone(),
        host_sha256: installation.host_sha256,
        helper_sha256: installation.helper_sha256.clone(),
        model_implementation_owner_sha256: installation.helper_sha256,
        compiler_sha256: installation.compiler_sha256,
        runtime_inventory_sha256: digest(&inventory.raw)?,
        inventory,
        runtime_implementation_sha256: installation
            .runtime_implementation_sha256
            .ok_or("inspection runtime implementation absent")?,
        engine_sha256: canonical::digest(
            gomad_wasmhost::linked::Conditions::compiled()
                .configuration()
                .as_bytes(),
        ),
        host_os: std::env::consts::OS.into(),
        host_arch: std::env::consts::ARCH.into(),
    })
}

#[cfg(test)]
#[path = "inspection_driver_tests.rs"]
mod tests;
