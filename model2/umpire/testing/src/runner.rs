use std::{
    collections::{BTreeMap, BTreeSet},
    fs,
    path::{Path, PathBuf},
};

/// A backend's declared logical expected files and deterministic renderer.
pub struct Consumer {
    /// Backend name used in diagnostics.
    pub name: &'static str,
    /// Logical filenames; `paths/*.txt` also admits named counterexample paths.
    pub expected: &'static [&'static str],
    /// Render this fixture, with its source folder available to source-based backends.
    pub render: fn(fn() -> crate::RunResult, &Path) -> crate::RunResult,
}

/// Native checker outputs, including the optional negative-control report.
pub const CHECKER: Consumer = Consumer {
    name: "checker",
    expected: &["report.txt", "states.txt", "rows.txt", "negative_report.txt", "paths/*.txt"],
    render: |produce, _| produce(),
};

impl Consumer {
    fn owns(&self, file: &str) -> bool {
        self.expected.iter().any(|pattern| {
            pattern.split_once('*').map_or_else(
                || *pattern == file,
                |(prefix, suffix)| {
                    file.strip_prefix(prefix)
                        .and_then(|rest| rest.strip_suffix(suffix))
                        .is_some_and(|name| !name.is_empty() && !name.contains('/'))
                },
            )
        })
    }
}

/// Run crate-local native checker fixtures.
pub fn run(manifest: &str, fixtures: Vec<crate::Fixture>) -> crate::TestResult {
    run_at(&Path::new(manifest).join("tests/fixtures"), fixtures, &[CHECKER])
}

/// Run selected consumers at an explicit fixture root, in registry order.
pub fn run_at(
    root: &Path,
    fixtures: Vec<crate::Fixture>,
    consumers: &[Consumer],
) -> crate::TestResult {
    let discovered: BTreeSet<_> =
        crate::discovery::discover(root)?.into_iter().map(|(name, _)| name).collect();
    let registered: BTreeSet<_> = fixtures.iter().map(|(name, _)| (*name).to_owned()).collect();
    if discovered != registered || fixtures.len() != registered.len() {
        return Err("discovered fixtures differ from generated registry; rebuild required".into());
    }
    let mut actual = BTreeMap::new();
    let mut existing = Vec::new();
    for (name, run) in fixtures {
        let folder = root.join(name);
        let expected = folder.join("expected");
        let mut files = Vec::new();
        collect_files(&expected, &mut files)?;
        files.sort();
        let mut selected = BTreeSet::new();
        for path in &files {
            let logical = logical_file(path, &expected)?;
            let owners: Vec<_> =
                consumers.iter().enumerate().filter(|(_, c)| c.owns(&logical)).collect();
            if owners.is_empty() {
                return Err(format!("expected file has no consumer: {}", path.display()).into());
            }
            if owners.len() != 1 {
                return Err(
                    format!("expected file has multiple consumers: {}", path.display()).into()
                );
            }
            selected.insert(owners[0].0);
        }
        if selected.is_empty() {
            return Err(format!(
                "fixture has no consumer-owned expected files: {}",
                folder.display()
            )
            .into());
        }
        for index in selected {
            let consumer = &consumers[index];
            let output = (consumer.render)(run, &folder)?;
            if output != (consumer.render)(run, &folder)? {
                return Err(format!(
                    "nondeterministic {} producer: {}",
                    consumer.name,
                    folder.display()
                )
                .into());
            }
            for (file, content) in output {
                if !consumer.owns(&file) {
                    return Err(format!(
                        "{} produced undeclared output {file} for {}",
                        consumer.name,
                        folder.display()
                    )
                    .into());
                }
                actual.insert(expected.join(file), content);
            }
        }
        existing.extend(files);
    }
    let permitted: BTreeSet<_> = actual.keys().flat_map(|path| golden_files(path)).collect();
    for path in existing {
        if !permitted.contains(&path) {
            return Err(format!("golden has no fixture: {}", path.display()).into());
        }
    }
    for (path, content) in actual {
        snapshot(&path, &content)?;
    }
    Ok(())
}

fn logical_file(path: &Path, root: &Path) -> Result<String, Box<dyn std::error::Error>> {
    let relative = path.strip_prefix(root)?.to_string_lossy().replace('\\', "/");
    let (stem, extension) = if let Some(stem) =
        relative.strip_suffix(".snap.new").or_else(|| relative.strip_suffix(".snap"))
    {
        let metadata = fs::read_to_string(path)?;
        let extension = metadata
            .lines()
            .find_map(|line| line.strip_prefix("extension: "))
            .ok_or_else(|| format!("invalid binary snapshot metadata: {}", path.display()))?;
        (stem.to_owned(), extension.to_owned())
    } else if let Some((stem, extension)) =
        relative.split_once(".snap.new.").or_else(|| relative.split_once(".snap."))
    {
        (stem.to_owned(), extension.to_owned())
    } else {
        return Ok(relative);
    };
    let stem = if extension == "txt" {
        &stem
    } else {
        stem.strip_suffix(&format!("@{extension}")).unwrap_or(&stem)
    };
    Ok(format!("{stem}.{extension}"))
}

fn golden_files(path: &Path) -> [PathBuf; 5] {
    let extension = path.extension().and_then(|s| s.to_str()).unwrap_or("txt");
    let metadata = if extension == "txt" {
        path.with_extension("snap")
    } else {
        path.with_file_name(format!(
            "{}@{extension}.snap",
            path.file_stem().unwrap_or_default().to_string_lossy()
        ))
    };
    [
        path.into(),
        metadata.clone(),
        metadata.with_extension(format!("snap.{extension}")),
        metadata.with_extension("snap.new"),
        metadata.with_extension(format!("snap.new.{extension}")),
    ]
}

fn snapshot(path: &Path, content: &str) -> crate::TestResult {
    let parent = path.parent().ok_or("golden has no parent")?;
    let [_, metadata, payload, _, _] = golden_files(path);
    let target = Path::new(payload.file_name().ok_or("golden has no file name")?);
    if std::env::var("INSTA_UPDATE").as_deref() == Ok("always") {
        fs::create_dir_all(parent)?;
        if fs::read_link(path).ok().as_deref() != Some(target) {
            if fs::symlink_metadata(path).is_ok() {
                fs::remove_file(path)?;
            }
            #[cfg(unix)]
            std::os::unix::fs::symlink(target, path)?;
            #[cfg(windows)]
            std::os::windows::fs::symlink_file(target, path)?;
        }
    } else {
        if fs::read_link(path).ok().as_deref() != Some(target) {
            return Err(format!("fixture has no golden alias: {}", path.display()).into());
        }
        fs::read(path).map_err(|error| {
            format!("fixture has no readable golden {}: {error}", path.display())
        })?;
        if !metadata.is_file() {
            return Err(format!("fixture has no snapshot metadata: {}", path.display()).into());
        }
    }
    let mut settings = insta::Settings::clone_current();
    settings.set_snapshot_path(parent);
    settings.set_prepend_module_to_snapshot(false);
    settings.set_omit_expression(true);
    let file = path.file_name().and_then(|name| name.to_str()).ok_or("invalid golden file name")?;
    let extension =
        path.extension().and_then(|name| name.to_str()).ok_or("golden has no extension")?;
    if extension != "txt" {
        settings.set_snapshot_suffix(extension);
    }
    let name = file;
    settings.bind(|| {
        insta::assert_binary_snapshot!(&name, content.as_bytes().to_vec());
    });
    Ok(())
}

fn collect_files(root: &Path, files: &mut Vec<PathBuf>) -> std::io::Result<()> {
    if !root.exists() {
        return Ok(());
    }
    for entry in fs::read_dir(root)? {
        let entry = entry?;
        let path = entry.path();
        if entry.file_type()?.is_dir() {
            collect_files(&path, files)?;
        } else {
            files.push(path);
        }
    }
    Ok(())
}
