use std::{
    collections::{BTreeMap, BTreeSet},
    fs,
    path::{Path, PathBuf},
};

/// Run every discovered fixture and compare all of its expected outputs.
pub fn run(manifest: &str, fixtures: Vec<crate::Fixture>) -> crate::TestResult {
    let root = Path::new(manifest).join("tests/fixtures");
    let discovered: BTreeSet<_> =
        crate::discovery::discover(&root)?.into_iter().map(|(name, _)| name).collect();
    let registered: BTreeSet<_> = fixtures.iter().map(|(name, _)| (*name).to_owned()).collect();
    if discovered != registered || fixtures.len() != registered.len() {
        return Err("discovered fixtures differ from generated registry; rebuild required".into());
    }
    let mut actual = BTreeMap::new();
    let mut compile_only = BTreeSet::new();
    for (name, run) in fixtures {
        let output = run()?;
        if output != run()? {
            return Err(format!("nondeterministic golden producer: {name}").into());
        }
        if output.is_empty() {
            compile_only.insert(root.join(name).join("expected/.gitkeep"));
        }
        for (file, content) in output {
            actual.insert(root.join(name).join("expected").join(file), content);
        }
    }
    let mut existing = Vec::new();
    for name in discovered {
        collect_files(&root.join(name).join("expected"), &mut existing)?;
    }
    let permitted: BTreeSet<_> =
        actual.keys().flat_map(|path| golden_files(path)).chain(compile_only).collect();
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

fn golden_files(path: &Path) -> [PathBuf; 5] {
    [
        path.into(),
        path.with_extension("snap"),
        path.with_extension("snap.txt"),
        path.with_extension("snap.new"),
        path.with_extension("snap.new.txt"),
    ]
}

fn snapshot(path: &Path, content: &str) -> crate::TestResult {
    let parent = path.parent().ok_or("golden has no parent")?;
    let payload = path.with_extension("snap.txt");
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
        if !path.with_extension("snap").is_file() {
            return Err(format!("fixture has no snapshot metadata: {}", path.display()).into());
        }
    }
    let mut settings = insta::Settings::clone_current();
    settings.set_snapshot_path(parent);
    settings.set_prepend_module_to_snapshot(false);
    settings.set_omit_expression(true);
    let name = path.file_name().and_then(|name| name.to_str()).ok_or("invalid golden file name")?;
    settings.bind(|| {
        insta::assert_binary_snapshot!(name, content.as_bytes().to_vec());
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
