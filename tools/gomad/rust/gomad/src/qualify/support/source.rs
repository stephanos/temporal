use super::{BTreeMap, BTreeSet, Candidate, Path, PathBuf, canonical};

pub(super) fn inventory(candidate: &Candidate) -> Result<BTreeMap<PathBuf, String>, String> {
    if canonical::digest(candidate.integrated_source_manifest.as_bytes())
        != candidate.integrated_source_sha256
    {
        return Err("support integrated source preimage changed".into());
    }
    candidate
        .integrated_source_manifest
        .lines()
        .map(|line| {
            let (sha256, path) = line
                .split_once("  ")
                .ok_or("support integrated source row malformed")?;
            Ok((PathBuf::from(path), sha256.to_owned()))
        })
        .collect()
}

pub(super) fn schema(
    namespace: &gomad_wasmhost::linked::NamespaceImage,
    candidate: &Candidate,
) -> Result<(), String> {
    let sources = inventory(candidate)?;
    let source_root = candidate.source_root.join("schema/sqlite");
    let mut expected = BTreeMap::new();
    for (path, sha256) in sources {
        if let Ok(relative) = path.strip_prefix(&source_root) {
            expected.insert(Path::new("/workspace/schema/sqlite").join(relative), sha256);
        }
    }
    let mut directories = BTreeSet::new();
    let mut paths: Vec<_> = expected.keys().cloned().collect();
    for path in &namespace.writable_directories {
        paths.push(PathBuf::from(
            std::str::from_utf8(path).map_err(|_| "support writable path is not UTF-8")?,
        ));
    }
    for path in &paths {
        let start = if expected.contains_key(path) {
            path.parent()
        } else {
            Some(path.as_path())
        };
        if let Some(start) = start {
            for parent in start
                .ancestors()
                .take_while(|parent| *parent != Path::new("/"))
            {
                directories.insert(parent.to_path_buf());
            }
        }
    }
    let mut observed = BTreeMap::new();
    let mut observed_directories = BTreeSet::new();
    for entry in &namespace.entries {
        let path = PathBuf::from(
            std::str::from_utf8(&entry.path).map_err(|_| "support namespace path is not UTF-8")?,
        );
        if entry.directory {
            let readonly = path.starts_with("/workspace/schema");
            if !entry.data.is_empty()
                || entry.readonly != readonly
                || !directories.contains(&path)
                || !observed_directories.insert(path)
            {
                return Err("support schema namespace directory policy changed".into());
            }
        } else if !entry.readonly
            || !path.starts_with("/workspace/schema/sqlite")
            || observed
                .insert(path, canonical::digest(&entry.data))
                .is_some()
        {
            return Err("support namespace contains foreign or duplicate initial file".into());
        }
    }
    if expected.is_empty() || expected != observed || directories != observed_directories {
        return Err("support schema capture differs from integrated source inventory".into());
    }
    Ok(())
}

pub(super) fn inspection_staged(
    provenance: &serde_json::Value,
    staged: &Path,
    candidate: &Candidate,
) -> Result<(), String> {
    let template_root = candidate
        .source_root
        .join("tools/gomad/internal/backends/wasm/testdata/prototype_temporal");
    let mut expected = BTreeMap::new();
    for (path, sha256) in inventory(candidate)? {
        if path.parent() == Some(template_root.as_path())
            && path.extension().is_some_and(|extension| extension == "go")
        {
            expected.insert(
                path.file_name()
                    .ok_or("inspection template filename missing")?
                    .to_owned(),
                sha256,
            );
        }
    }
    let raw: BTreeMap<PathBuf, String> =
        serde_json::from_value(provenance["inputs"]["sources"].clone())
            .map_err(|error| error.to_string())?;
    let mut observed = BTreeMap::new();
    for (path, sha256) in raw {
        if path.starts_with(staged) {
            if path.parent() != Some(staged) {
                return Err("inspection staged source contains nested inputs".into());
            }
            observed.insert(
                path.file_name()
                    .ok_or("inspection staged filename missing")?
                    .to_owned(),
                sha256,
            );
        }
    }
    if expected.is_empty() || expected != observed {
        return Err("inspection sources differ from unchanged original Go templates".into());
    }
    Ok(())
}
