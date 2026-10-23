use super::{
    BTreeMap, BTreeSet, Deserialize, Path, PathBuf, Serialize, Value, absolute, canonical, read,
    spec,
};

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(super) struct Overlay {
    pub path: PathBuf,
    pub sha256: String,
    pub bytes_base64: String,
}
#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(super) struct Inventory {
    pub compiler_path: PathBuf,
    pub runtime_root: PathBuf,
    pub runtime_identity: String,
    pub raw: BTreeMap<PathBuf, String>,
    pub overlay: Overlay,
}
#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Descriptor {
    #[serde(rename = "Replace")]
    replace: BTreeMap<PathBuf, PathBuf>,
}
#[derive(Debug, Eq, PartialEq)]
pub(super) struct View {
    entries: BTreeMap<String, String>,
    replacements: BTreeMap<PathBuf, PathBuf>,
    runtime_identity: String,
}

fn runtime_root(raw: &BTreeMap<PathBuf, String>) -> Result<PathBuf, String> {
    let roots: Vec<_> = raw
        .keys()
        .filter(|path| {
            path.file_name()
                .is_some_and(|name| name == "gomad_semantics.go.txt")
        })
        .filter_map(|path| path.parent().map(Path::to_path_buf))
        .collect();
    let [root] = <[PathBuf; 1]>::try_from(roots)
        .map_err(|_| "support runtime source root must be unique")?;
    if !raw.contains_key(&root.join("gomad_choicewire_generated.go.txt")) {
        return Err("support runtime schema source missing".into());
    }
    Ok(root)
}

pub(super) fn from_provenance(
    provenance: &Value,
    overlay: Option<Overlay>,
) -> Result<Inventory, String> {
    let inputs = &provenance["inputs"];
    let raw: BTreeMap<PathBuf, String> =
        serde_json::from_value(inputs["compiler"].clone()).map_err(|error| error.to_string())?;
    let compiler_path = PathBuf::from(
        inputs["compiler_path"]
            .as_str()
            .ok_or("support compiler executable path missing")?,
    );
    let runtime_identity = inputs["runtime"]
        .as_str()
        .filter(|digest| canonical::valid_digest(digest))
        .ok_or("support generated runtime identity missing")?
        .to_owned();
    let paths: Vec<_> = raw
        .keys()
        .filter(|path| path.file_name().is_some_and(|name| name == "overlay.json"))
        .cloned()
        .collect();
    let [path] =
        <[PathBuf; 1]>::try_from(paths).map_err(|_| "support overlay descriptor must be unique")?;
    let overlay = if let Some(overlay) = overlay {
        overlay
    } else {
        let bytes = read(&path, 16 << 20)?;
        Overlay {
            path: path.clone(),
            sha256: canonical::digest(&bytes),
            bytes_base64: crate::campaign::exploration::base64(&bytes),
        }
    };
    if overlay.path != path {
        return Err("support overlay descriptor reference changed".into());
    }
    let inventory = Inventory {
        compiler_path,
        runtime_root: runtime_root(&raw)?,
        runtime_identity,
        raw,
        overlay,
    };
    view(&inventory)?;
    Ok(inventory)
}

pub(super) fn view(inventory: &Inventory) -> Result<View, String> {
    absolute(&inventory.compiler_path)?;
    absolute(&inventory.runtime_root)?;
    absolute(&inventory.overlay.path)?;
    let compiler_root = inventory
        .compiler_path
        .parent()
        .filter(|parent| parent.file_name().is_some_and(|name| name == "bin"))
        .and_then(Path::parent)
        .ok_or("support compiler must be a stock distribution bin/go")?;
    if inventory
        .compiler_path
        .file_name()
        .is_none_or(|name| name != "go")
    {
        return Err("support compiler executable is not bin/go".into());
    }
    let overlay_root = inventory
        .overlay
        .path
        .parent()
        .ok_or("support overlay parent missing")?;
    if overlay_root.file_name().and_then(|name| name.to_str())
        != Some(inventory.runtime_identity.as_str())
        || !inventory.raw.contains_key(&compiler_root.join("VERSION"))
    {
        return Err("support compiler distribution or overlay runtime identity missing".into());
    }
    let bytes = crate::qualify::cells::base64(&inventory.overlay.bytes_base64)
        .ok_or("support overlay descriptor base64 invalid")?;
    if bytes.len() > 16 << 20
        || canonical::digest(&bytes) != inventory.overlay.sha256
        || inventory.raw.get(&inventory.overlay.path) != Some(&inventory.overlay.sha256)
    {
        return Err("support raw overlay descriptor digest changed".into());
    }
    let descriptor: Descriptor = spec::strict(&bytes)?;
    if descriptor.replace.is_empty() {
        return Err("support overlay replacements missing".into());
    }
    let mut replacements = BTreeMap::new();
    let mut outputs = BTreeSet::new();
    for (source, target) in descriptor.replace {
        absolute(&source)?;
        absolute(&target)?;
        let source = source
            .strip_prefix(compiler_root)
            .map_err(|_| "support overlay source escapes compiler distribution")?
            .to_path_buf();
        let relative = target
            .strip_prefix(overlay_root)
            .map_err(|_| "support overlay replacement escapes its descriptor root")?
            .to_path_buf();
        if source.components().count() < 2
            || !source.starts_with("src")
            || relative.components().count() != 1
            || !inventory.raw.contains_key(&target)
            || !outputs.insert(target)
            || replacements.insert(source, relative).is_some()
        {
            return Err("support overlay replacement membership is invalid".into());
        }
    }
    let mut entries = BTreeMap::new();
    for (path, sha256) in &inventory.raw {
        absolute(path)?;
        if !canonical::valid_digest(sha256) {
            return Err("support compiler inventory member digest invalid".into());
        }
        if path == &inventory.overlay.path {
            continue;
        }
        let name = if let Ok(relative) = path.strip_prefix(overlay_root) {
            if !outputs.contains(path) {
                return Err("support undeclared generated overlay member".into());
            }
            format!("overlay/{}", relative.display())
        } else if let Ok(relative) = path.strip_prefix(compiler_root) {
            format!("compiler/{}", relative.display())
        } else if let Ok(relative) = path.strip_prefix(&inventory.runtime_root) {
            format!("runtime/{}", relative.display())
        } else {
            format!("shared/{}", path.display())
        };
        if entries.insert(name, sha256.clone()).is_some() {
            return Err("support aliased inventory member collision".into());
        }
    }
    Ok(View {
        entries,
        replacements,
        runtime_identity: inventory.runtime_identity.clone(),
    })
}

pub(super) fn from_alias(
    provenance: &Value,
    alias: &Overlay,
    alias_compiler_root: &Path,
) -> Result<Inventory, String> {
    absolute(alias_compiler_root)?;
    let raw: BTreeMap<PathBuf, String> =
        serde_json::from_value(provenance["inputs"]["compiler"].clone())
            .map_err(|error| error.to_string())?;
    let compiler = PathBuf::from(
        provenance["inputs"]["compiler_path"]
            .as_str()
            .ok_or("candidate compiler path missing")?,
    );
    let overlay = alias_overlay(&raw, &compiler, alias, alias_compiler_root)?;
    from_provenance(provenance, Some(overlay))
}

pub(super) fn from_frozen_inputs(
    raw: BTreeMap<PathBuf, String>,
    compiler_path: PathBuf,
    observed_alias: &Inventory,
    alias_compiler_root: &Path,
) -> Result<Inventory, String> {
    if observed_alias.compiler_path != alias_compiler_root.join("bin/go")
        || observed_alias.raw.get(&observed_alias.compiler_path) != raw.get(&compiler_path)
        || !raw.contains_key(&compiler_path)
    {
        return Err(
            "inspection actual compiler executable differs from frozen installation".into(),
        );
    }
    let overlay = alias_overlay(
        &raw,
        &compiler_path,
        &observed_alias.overlay,
        alias_compiler_root,
    )?;
    let runtime_identity = overlay
        .path
        .parent()
        .and_then(Path::file_name)
        .and_then(|name| name.to_str())
        .filter(|name| canonical::valid_digest(name))
        .ok_or("frozen installation runtime identity missing")?
        .to_owned();
    let mut inventory = Inventory {
        runtime_root: runtime_root(&raw)?,
        raw,
        compiler_path,
        runtime_identity,
        overlay,
    };
    let full = view(&inventory)?;
    let observed = view(observed_alias)?;
    if full.runtime_identity != observed.runtime_identity
        || full.replacements != observed.replacements
        || observed
            .entries
            .iter()
            .any(|(name, sha)| full.entries.get(name) != Some(sha))
    {
        return Err(
            "inspection actual compiler closure is not a subset of frozen installation inputs"
                .into(),
        );
    }
    let compiler_root = inventory
        .compiler_path
        .parent()
        .and_then(Path::parent)
        .ok_or("frozen compiler distribution missing")?;
    let overlay_root = inventory
        .overlay
        .path
        .parent()
        .ok_or("frozen overlay root missing")?;
    let mut selected = BTreeMap::from([(
        inventory.overlay.path.clone(),
        inventory.overlay.sha256.clone(),
    )]);
    for (path, sha256) in &inventory.raw {
        if path == &inventory.overlay.path {
            continue;
        }
        let name = if let Ok(relative) = path.strip_prefix(overlay_root) {
            format!("overlay/{}", relative.display())
        } else if let Ok(relative) = path.strip_prefix(compiler_root) {
            format!("compiler/{}", relative.display())
        } else if let Ok(relative) = path.strip_prefix(&inventory.runtime_root) {
            format!("runtime/{}", relative.display())
        } else {
            format!("shared/{}", path.display())
        };
        if observed.entries.contains_key(&name) {
            selected.insert(path.clone(), sha256.clone());
        }
    }
    inventory.raw = selected;
    if view(&inventory)? != observed {
        return Err(
            "inspection selected original inventory differs from actual alias closure".into(),
        );
    }
    Ok(inventory)
}

fn alias_overlay(
    raw: &BTreeMap<PathBuf, String>,
    compiler: &Path,
    alias: &Overlay,
    alias_compiler_root: &Path,
) -> Result<Overlay, String> {
    absolute(alias_compiler_root)?;
    let compiler_root = compiler
        .parent()
        .and_then(Path::parent)
        .ok_or("candidate compiler distribution missing")?;
    let overlays: Vec<_> = raw
        .keys()
        .filter(|path| path.file_name().is_some_and(|name| name == "overlay.json"))
        .cloned()
        .collect();
    let [path] = <[PathBuf; 1]>::try_from(overlays)
        .map_err(|_| "candidate overlay descriptor must be unique")?;
    let original_root = path.parent().ok_or("candidate overlay parent missing")?;
    let alias_root = alias
        .path
        .parent()
        .ok_or("inspection overlay parent missing")?;
    let alias_bytes = crate::qualify::cells::base64(&alias.bytes_base64)
        .ok_or("inspection overlay base64 invalid")?;
    if alias_bytes.len() > 16 << 20 || canonical::digest(&alias_bytes) != alias.sha256 {
        return Err("inspection raw overlay descriptor digest changed".into());
    }
    let descriptor: Descriptor = spec::strict(&alias_bytes)?;
    let mut replace = BTreeMap::new();
    for (source, target) in descriptor.replace {
        let source = source
            .strip_prefix(alias_compiler_root)
            .map_err(|_| "inspection overlay source escaped declared compiler alias")?;
        let target = target
            .strip_prefix(alias_root)
            .map_err(|_| "inspection replacement escaped declared cache alias")?;
        replace.insert(compiler_root.join(source), original_root.join(target));
    }
    // This reconstructs the original Go descriptor bytes and checks their
    // retained raw hash. It never edits either inventory or descriptor.
    let bytes = serde_json::to_string(&serde_json::json!({"Replace":replace}))
        .map_err(|error| error.to_string())?
        .replace('&', "\\u0026")
        .replace('<', "\\u003c")
        .replace('>', "\\u003e")
        .replace('\u{2028}', "\\u2028")
        .replace('\u{2029}', "\\u2029")
        .into_bytes();
    let sha256 = canonical::digest(&bytes);
    if raw.get(&path) != Some(&sha256) {
        return Err(
            "reconstructed original overlay bytes differ from candidate raw inventory".into(),
        );
    }
    Ok(Overlay {
        path,
        sha256,
        bytes_base64: crate::campaign::exploration::base64(&bytes),
    })
}
