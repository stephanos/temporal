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

/// One direct artifact whose comparison belongs to a named external test.
#[derive(Clone, Copy)]
pub struct ExternalConsumer {
    /// Live consumer that must execute and compare this artifact separately.
    pub name: &'static str,
    /// Known fixture owning the artifact.
    pub fixture: &'static str,
    /// Direct artifact basename, distinct from native snapshot files.
    pub expected: &'static str,
}

/// Native checker outputs, including the optional negative-control report.
pub const CHECKER: Consumer = Consumer {
    name: "checker",
    expected: &["report.txt", "states.txt", "rows.txt", "negative_report.txt", "paths/*.txt"],
    render: |produce, _| produce(),
};

/// Suite expansion and coverage outputs.
pub const SUITE: Consumer = Consumer {
    name: "suite",
    expected: &["expansion.txt", "expansion/*.txt", "coverage.txt"],
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

fn owners(file: &str, consumers: &[Consumer]) -> Vec<usize> {
    consumers
        .iter()
        .enumerate()
        .filter(|(_, consumer)| consumer.owns(file))
        .map(|(index, _)| index)
        .collect()
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
    run_at_skipping(root, fixtures, consumers, &[])
}

/// Validate selected optional backend artifacts without claiming execution evidence.
pub fn run_at_skipping(
    root: &Path,
    fixtures: Vec<crate::Fixture>,
    consumers: &[Consumer],
    skipped: &[&str],
) -> crate::TestResult {
    run_at_skipping_with_external(root, fixtures, consumers, skipped, &[])
}

/// Preserves externally owned artifacts without claiming their execution or comparison.
pub fn run_at_skipping_with_external(
    root: &Path,
    fixtures: Vec<crate::Fixture>,
    consumers: &[Consumer],
    skipped: &[&str],
    external: &[ExternalConsumer],
) -> crate::TestResult {
    check_skipped(consumers, skipped)?;
    let discovered: BTreeSet<_> = crate::discovery::discover(root, crate::discovery::MODEL)?
        .into_iter()
        .map(|fixture| fixture.name)
        .collect();
    let registered: BTreeSet<_> = fixtures.iter().map(|(name, _)| (*name).to_owned()).collect();
    if discovered != registered || fixtures.len() != registered.len() {
        return Err("discovered fixtures differ from generated registry; rebuild required".into());
    }
    let external_artifacts = external_artifacts(root, &registered, consumers, external)?;
    let mut actual = BTreeMap::new();
    let mut existing = Vec::new();
    let mut skipped_artifacts = BTreeSet::new();
    for (name, run) in fixtures {
        let folder = root.join(name);
        let expected = folder.join("expected");
        let mut files = Vec::new();
        collect_files(&expected, &mut files)?;
        files.sort();
        let mut selected = BTreeSet::new();
        for path in files.iter().filter(|path| !external_artifacts.contains(*path)) {
            let logical = logical_file(path, &expected)?;
            let owners = owners(&logical, consumers);
            if owners.is_empty() {
                return Err(format!("expected file has no consumer: {}", path.display()).into());
            }
            if owners.len() != 1 {
                return Err(
                    format!("expected file has multiple consumers: {}", path.display()).into()
                );
            }
            selected.insert(owners[0]);
        }
        if selected.is_empty() {
            return Err(format!(
                "fixture has no consumer-owned expected files: {}",
                folder.display()
            )
            .into());
        }
        for &index in &selected {
            let consumer = &consumers[index];
            if skipped.contains(&consumer.name) {
                skipped_artifacts.extend(skipped_files(&files, &expected, consumer)?);
                continue;
            }
            let output = (consumer.render)(run, &folder)?;
            if output != (consumer.render)(run, &folder)? {
                return Err(format!(
                    "nondeterministic {} producer: {}",
                    consumer.name,
                    folder.display()
                )
                .into());
            }
            let location = folder.display();
            for (file, content) in output {
                if external_artifacts.contains(&expected.join(&file)) {
                    return Err(format!(
                        "native producer emitted external artifact: {file} for {location}"
                    )
                    .into());
                }
                let owners = owners(&file, consumers);
                if owners.len() > 1 {
                    return Err(format!(
                        "produced output has multiple consumers: {file} for {location}"
                    )
                    .into());
                }
                let Some(&owner) = owners.first().filter(|owner| selected.contains(owner)) else {
                    return Err(format!(
                        "{} produced undeclared output {file} for {}",
                        consumer.name,
                        folder.display()
                    )
                    .into());
                };
                if owner != index {
                    continue;
                }
                actual.insert(expected.join(file), content);
            }
        }
        existing.extend(files);
    }
    let mut permitted: BTreeSet<_> = actual.keys().flat_map(|path| golden_files(path)).collect();
    permitted.extend(skipped_artifacts);
    permitted.extend(external_artifacts);
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

#[allow(clippy::print_stderr)]
fn check_skipped(consumers: &[Consumer], skipped: &[&str]) -> crate::TestResult {
    for name in skipped {
        if !consumers.iter().any(|consumer| consumer.name == *name) {
            return Err(format!("unknown skipped consumer: {name}").into());
        }
        eprintln!(
            "backend {name}: skipped; no execution or agreement evidence (UMPIRE_BACKENDS=require runs it)"
        );
    }
    Ok(())
}

#[allow(clippy::print_stderr)]
fn external_artifacts(
    root: &Path,
    registered: &BTreeSet<String>,
    consumers: &[Consumer],
    external: &[ExternalConsumer],
) -> Result<BTreeSet<PathBuf>, Box<dyn std::error::Error>> {
    let mut artifacts = BTreeSet::new();
    for declaration in external {
        if declaration.name.trim().is_empty() {
            return Err("invalid external consumer: empty identity".into());
        }
        if !registered.contains(declaration.fixture) {
            return Err(format!("unknown external fixture: {}", declaration.fixture).into());
        }
        let file = declaration.expected;
        if file.is_empty()
            || matches!(file, "." | "..")
            || file.contains(['/', '\\', '*', '?'])
            || Path::new(file)
                .extension()
                .is_some_and(|extension| extension.eq_ignore_ascii_case("snap"))
            || file.contains(".snap.")
        {
            return Err(format!("invalid external artifact: {file}").into());
        }
        if !owners(file, consumers).is_empty() {
            return Err(format!("external artifact has native consumer: {file}").into());
        }
        let expected = root.join(declaration.fixture).join("expected");
        let path = expected.join(file);
        if !artifacts.insert(path.clone()) {
            return Err(format!("duplicate external artifact: {}", path.display()).into());
        }
        if !fs::symlink_metadata(&expected)?.is_dir()
            || !fs::symlink_metadata(&path).is_ok_and(|metadata| metadata.is_file())
        {
            return Err(
                format!("missing external artifact or unsafe path: {}", path.display()).into()
            );
        }
        eprintln!(
            "external artifact {}: requires {}; no execution or comparison evidence from native runner",
            path.display(),
            declaration.name
        );
    }
    Ok(artifacts)
}

fn skipped_files(
    files: &[PathBuf],
    expected: &Path,
    consumer: &Consumer,
) -> Result<BTreeSet<PathBuf>, Box<dyn std::error::Error>> {
    let mut artifacts = BTreeSet::new();
    for path in files {
        let logical = logical_file(path, expected)?;
        if !consumer.owns(&logical) {
            continue;
        }
        let [metadata, payload, pending_metadata, pending_payload] =
            golden_files(&expected.join(&logical));
        if !metadata.is_file() || !payload.is_file() {
            return Err(format!("skipped backend has incomplete golden: {}", path.display()).into());
        }
        if pending_metadata.exists() || pending_payload.exists() {
            return Err(format!("skipped backend has pending golden: {}", path.display()).into());
        }
        artifacts.extend([metadata, payload]);
    }
    Ok(artifacts)
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

fn golden_files(path: &Path) -> [PathBuf; 4] {
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
        metadata.clone(),
        metadata.with_extension(format!("snap.{extension}")),
        metadata.with_extension("snap.new"),
        metadata.with_extension(format!("snap.new.{extension}")),
    ]
}

fn snapshot(path: &Path, content: &str) -> crate::TestResult {
    let parent = path.parent().ok_or("golden has no parent")?;
    let [metadata, payload, _, _] = golden_files(path);
    if std::env::var("INSTA_UPDATE").as_deref() == Ok("always") {
        fs::create_dir_all(parent)?;
    } else {
        fs::read(&payload).map_err(|error| {
            format!("fixture has no readable golden {}: {error}", payload.display())
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
