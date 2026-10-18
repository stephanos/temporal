//! Registered behavior cases and their human-readable golden files.

mod cases;

use std::{collections::BTreeMap, error::Error, fs, path::Path};

#[test]
fn samples() -> Result<(), Box<dyn Error>> {
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("cases");
    let mut actual = BTreeMap::new();
    for (name, run) in cases::registered() {
        let output = run()?;
        if output != run()? {
            return Err(format!("nondeterministic golden producer: {name}").into());
        }
        for (file, content) in output {
            actual.insert(root.join(name).join("expected").join(file), content);
        }
    }
    let mut existing = Vec::new();
    collect_files(&root, &mut existing)?;
    for path in existing {
        if !actual.contains_key(&path) {
            return Err(format!("golden has no case: {}", path.display()).into());
        }
    }
    for (path, content) in actual {
        if std::env::var("INSTA_UPDATE").as_deref() == Ok("always") {
            fs::create_dir_all(path.parent().ok_or("golden has no parent")?)?;
            fs::write(path, content)?;
        } else {
            let expected = fs::read_to_string(&path).map_err(|error| {
                format!("case has no readable golden {}: {error}", path.display())
            })?;
            // The pinned insta adapter keeps the contract's exact .txt paths, without .snap headers.
            let inline_reference = format!("\n{expected}");
            insta::allow_duplicates! {
                insta::assert_snapshot!(insta::_macro_support::InlineValue(&inline_reference), content);
            }
            if expected != content {
                return Err(format!("golden differs: {}", path.display()).into());
            }
        }
    }
    Ok(())
}

fn collect_files(root: &Path, files: &mut Vec<std::path::PathBuf>) -> std::io::Result<()> {
    if !root.exists() {
        return Ok(());
    }
    for entry in fs::read_dir(root)? {
        let path = entry?.path();
        if path.is_dir() {
            collect_files(&path, files)?;
        } else {
            files.push(path);
        }
    }
    Ok(())
}
