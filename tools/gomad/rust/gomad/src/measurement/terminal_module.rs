//! Original terminal module compilation, separate from the execution request.
use super::{
    inputs::{self, CheckoutProof, Invocation},
    module, module_recipes,
};
use crate::evidence::canonical;
use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::{
    collections::BTreeMap,
    fs,
    io::Write,
    os::unix::fs::{DirBuilderExt, OpenOptionsExt},
    path::{Path, PathBuf},
};

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Spec {
    schema: String,
    repo: PathBuf,
    stock_go: PathBuf,
    toolchain_cache: PathBuf,
    #[serde(default)]
    temporary_directory: Option<PathBuf>,
}
#[derive(Serialize, Deserialize, PartialEq)]
#[serde(deny_unknown_fields)]
struct Recipe {
    kind: String,
    source: String,
    directory: PathBuf,
    build_tags: Vec<String>,
    target_arguments: Vec<String>,
}
impl From<module_recipes::ModuleRecipe> for Recipe {
    fn from(value: module_recipes::ModuleRecipe) -> Self {
        Self {
            kind: value.kind.into(),
            source: value.source,
            directory: value.directory,
            build_tags: value.build_tags,
            target_arguments: value.target_arguments,
        }
    }
}
#[derive(Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
struct Descriptor {
    schema: String,
    module: PathBuf,
    module_sha256: String,
    provenance: PathBuf,
    provenance_sha256: String,
    installation: PathBuf,
    installation_sha256: String,
    preparation_observations: PathBuf,
    preparation_observations_sha256: String,
    preparation_receipt: PathBuf,
    preparation_receipt_sha256: String,
    recipe: Recipe,
    configuration: Value,
    memory_bytes: u64,
    fuel: u64,
}
fn read(path: &Path, limit: u64) -> Result<Vec<u8>, String> {
    crate::qualify::spec::read_bounded(path, limit)
}
fn save(path: &Path, bytes: &[u8]) -> Result<(), String> {
    let mut file = fs::OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .open(path)
        .map_err(|e| e.to_string())?;
    file.write_all(bytes)
        .and_then(|()| file.sync_all())
        .map_err(|e| e.to_string())
}
fn fresh(root: &Path) -> Result<(), String> {
    if !root.is_absolute() {
        return Err("absolute fresh evidence root required".into());
    }
    fs::DirBuilder::new()
        .mode(0o700)
        .create(root)
        .map_err(|e| e.to_string())
}
/// # Errors
/// Refuses changed checkout/compiler inputs, an existing output root or malformed closed input.
pub fn prepare(spec: &Path, root: &Path) -> Result<PathBuf, String> {
    let value: Spec = serde_json::from_value(crate::qualify::spec::strict(&read(spec, 1 << 20)?)?)
        .map_err(|e| e.to_string())?;
    if value.schema != "gomad.native-terminal-module-preparation-input/v1"
        || [&value.repo, &value.stock_go, &value.toolchain_cache]
            .iter()
            .any(|p| !p.is_absolute())
        || value
            .temporary_directory
            .as_ref()
            .is_some_and(|p| !p.is_absolute())
    {
        return Err("invalid terminal module preparation input".into());
    }
    let proof = inputs::authenticate_checkout(&value.repo)?;
    let mut selected = BTreeMap::from([
        (
            "GOMAD_STOCK_GO".into(),
            value.stock_go.to_string_lossy().into_owned(),
        ),
        (
            "GOMAD_WASM_TOOLCHAIN_CACHE".into(),
            value.toolchain_cache.to_string_lossy().into_owned(),
        ),
    ]);
    if let Some(path) = value.temporary_directory {
        selected.insert("GOTMPDIR".into(), path.to_string_lossy().into_owned());
    }
    let mut environ = std::env::vars()
        .filter(|(key, _)| key != "GOTMPDIR")
        .map(|(k, v)| format!("{k}={v}"))
        .collect::<Vec<_>>();
    if let Some(directory) = selected.get("GOTMPDIR") {
        environ.push(format!("GOTMPDIR={directory}"));
    }
    let call = Invocation {
        name: "terminal-exit".into(),
        repo: value.repo,
        go_command: value.stock_go.to_string_lossy().into_owned(),
        inputs: selected,
        environ,
        keep: true,
    };
    fresh(root)?;
    let prepared = module::prepare_terminal(&proof, &call, root)?;
    let provenance = module::authenticate_prepared(&prepared)?;
    let installed = crate::install::manifest::resolve(Some(&prepared.installation_path))?;
    crate::install::manifest::admit_build(&installed.manifest)?;
    module::validate_terminal_binding(proof.repo(), root, &installed.manifest, &provenance)?;
    let original = module_recipes::terminal_exit(proof.repo())?;
    let observations = root.join("preparation-observations.json");
    let receipt = root.join("preparation-observations/receipt.json");
    let descriptor = Descriptor {
        schema: "gomad.native-terminal-module/v1".into(),
        module_sha256: canonical::digest(&read(&prepared.module_path, 512 << 20)?),
        module: prepared.module_path,
        provenance_sha256: canonical::digest(&read(&prepared.provenance_path, 16 << 20)?),
        provenance: prepared.provenance_path,
        installation_sha256: canonical::digest(&read(&prepared.installation_path, 16 << 20)?),
        installation: prepared.installation_path,
        preparation_observations_sha256: canonical::digest(&read(&observations, 16 << 20)?),
        preparation_observations: observations,
        preparation_receipt_sha256: canonical::digest(&read(&receipt, 16 << 20)?),
        preparation_receipt: receipt,
        recipe: original.module.into(),
        configuration: original.configuration,
        memory_bytes: original.memory_bytes,
        fuel: original.fuel,
    };
    let path = root.join("terminal-module.json");
    save(&path, &canonical::encode(&descriptor)?)?;
    Ok(path)
}

fn authenticate_files(root: &Path, descriptor: &Descriptor) -> Result<(), String> {
    for (file, digest, name, maximum) in [
        (
            &descriptor.module,
            &descriptor.module_sha256,
            "module.wasm",
            512 << 20,
        ),
        (
            &descriptor.provenance,
            &descriptor.provenance_sha256,
            "module-provenance.json",
            16 << 20,
        ),
        (
            &descriptor.installation,
            &descriptor.installation_sha256,
            "installation.json",
            16 << 20,
        ),
        (
            &descriptor.preparation_observations,
            &descriptor.preparation_observations_sha256,
            "preparation-observations.json",
            16 << 20,
        ),
        (
            &descriptor.preparation_receipt,
            &descriptor.preparation_receipt_sha256,
            "preparation-observations/receipt.json",
            16 << 20,
        ),
    ] {
        if file != &root.join(name)
            || !file.is_absolute()
            || canonical::digest(&read(file, maximum)?) != *digest
        {
            return Err("terminal module descriptor artifact binding changed".into());
        }
    }
    Ok(())
}

pub(crate) fn load(proof: &CheckoutProof, path: &Path) -> Result<module::Prepared, String> {
    let descriptor: Descriptor =
        serde_json::from_value(crate::qualify::spec::strict(&read(path, 16 << 20)?)?)
            .map_err(|e| e.to_string())?;
    let expected = module_recipes::terminal_exit(proof.repo())?;
    if descriptor.schema != "gomad.native-terminal-module/v1"
        || descriptor.recipe != Recipe::from(expected.module)
        || descriptor.configuration != expected.configuration
        || descriptor.memory_bytes != expected.memory_bytes
        || descriptor.fuel != expected.fuel
    {
        return Err("terminal module original preparation recipe mismatch".into());
    }
    let root = path.parent().ok_or("terminal descriptor root absent")?;
    authenticate_files(root, &descriptor)?;
    let raw = read(&descriptor.provenance, 16 << 20)?;
    let provenance = crate::qualify::spec::strict(&raw)?;
    crate::campaign::prepare::typed_provenance(&provenance)?;
    let installed = crate::install::manifest::resolve(Some(&descriptor.installation))?;
    crate::install::manifest::admit_build(&installed.manifest)?;
    module::validate_terminal_binding(proof.repo(), root, &installed.manifest, &provenance)?;
    let mut result = module::Prepared {
        request: None,
        observation_receipt_sha256: Some(descriptor.preparation_receipt_sha256.clone()),
        guest: crate::campaign::prepare::Prepared {
            auxiliary: BTreeMap::new(),
            compile_cache_root: crate::campaign::prepare::compile_cache_root(&installed.manifest)?,
            module: read(&descriptor.module, 512 << 20)?,
            configuration: serde_json::from_value(provenance["configuration"].clone())
                .map_err(|e| e.to_string())?,
            toolchain: provenance["build_key"]
                .as_str()
                .ok_or("terminal module build key absent")?
                .into(),
            choice_identity: serde_json::from_value(provenance["choice_identity"].clone())
                .map_err(|e| e.to_string())?,
            scenario: None,
            network: serde_json::from_value(
                provenance.get("network").cloned().unwrap_or(Value::Null),
            )
            .map_err(|e| e.to_string())?,
            soft_assertion_report_only: provenance["inputs"]["soft_assertion_report_only"]
                .as_bool()
                .ok_or("terminal assertion policy absent")?,
            provenance: raw,
        },
        module_path: descriptor.module,
        provenance_path: descriptor.provenance,
        installation_path: descriptor.installation,
        module_metadata: provenance["module_metadata"].clone(),
        preparation_arguments: Vec::new(),
        scenario_path: None,
    };
    result.guest.auxiliary = crate::campaign::prepare::open_observations_from_receipt(
        root,
        &result.guest,
        &descriptor.provenance_sha256,
        &descriptor.preparation_receipt_sha256,
    )?;
    if result.guest.auxiliary.is_empty() {
        return Err("terminal preparation observations absent".into());
    }
    module::authenticate_prepared(&result)?;
    Ok(result)
}
/// # Errors
/// Returns actual host collection errors and refuses existing evidence roots.
pub fn capture_host(root: &Path) -> Result<PathBuf, String> {
    fresh(root)?;
    let environ = std::env::vars()
        .map(|(k, v)| format!("{k}={v}"))
        .collect::<Vec<_>>();
    let observed = super::host::current(&root.join("host"), &environ)?;
    let path = root.join("host.json");
    save(&path, &canonical::encode(&observed)?)?;
    Ok(path)
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn original_spec_is_closed_and_paths_are_not_inferred() {
        let valid = serde_json::json!({"schema":"gomad.native-terminal-module-preparation-input/v1","repo":"/repo","stock_go":"/stock/go","toolchain_cache":"/cache"});
        assert!(serde_json::from_value::<Spec>(valid.clone()).is_ok());
        let mut changed = valid.clone();
        changed["module_sha256"] = Value::String("invented-future-module".into());
        assert!(serde_json::from_value::<Spec>(changed).is_err());
        let mut changed = valid;
        changed.as_object_mut().unwrap().remove("toolchain_cache");
        assert!(serde_json::from_value::<Spec>(changed).is_err());
    }
    #[test]
    fn host_endpoint_retains_actual_current_host_and_refuses_overwrite() {
        let root = std::env::temp_dir().join(format!(
            "gomad-terminal-host-{}-{}",
            std::process::id(),
            std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_nanos()
        ));
        let path = capture_host(&root).unwrap();
        assert_eq!(path, root.join("host.json"));
        let observed: super::super::host::HostEvidence =
            serde_json::from_slice(&read(&path, 2 << 20).unwrap()).unwrap();
        assert_eq!(observed.schema, "gomad.native-measurement-host/v1");
        assert_ne!(observed.boot_id, "");
        assert!(!observed.inputs.is_empty());
        for (path, digest) in observed.inputs {
            assert!(path.starts_with(root.join("host")));
            assert_eq!(canonical::digest(&read(&path, 1 << 20).unwrap()), digest);
        }
        let original = read(&path, 2 << 20).unwrap();
        assert!(capture_host(&root).is_err());
        assert_eq!(read(&path, 2 << 20).unwrap(), original);
        fs::remove_dir_all(root).unwrap();
    }
}
