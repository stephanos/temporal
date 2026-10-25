use crate::{Refusal, Scratch, pinned_java};
use sha2::{Digest, Sha256};
use std::{
    fs::{self, File},
    io::{ErrorKind, Read},
    path::{Component, Path, PathBuf},
};

/// A release jar admitted from a workspace cache, with the mise-pinned Java that runs it.
#[derive(Clone, Debug)]
pub struct PinnedJar {
    /// Canonical workspace root.
    pub root: PathBuf,
    /// Canonical admitted jar, contained in its cache.
    pub jar: PathBuf,
    /// The repository's pinned Temurin executable.
    pub java: PathBuf,
}

impl PinnedJar {
    /// Admit `root/cache/jar` before any use: no symlink or escape on its path, the exact
    /// SHA-256, the pinned Java, then `probe` runs the tool's version check in a cache scratch.
    pub fn admit(
        root: &Path,
        cache: &str,
        jar: &str,
        sha: &str,
        probe: impl FnOnce(&Self, &Path) -> Result<(), Refusal>,
    ) -> Result<Self, Refusal> {
        let root = root.canonicalize()?;
        let jar = contained(&root, &format!("{cache}/{jar}"))?;
        if digest(&jar)? != sha {
            return Err(Refusal(format!("pinned jar integrity mismatch: {}", jar.display())));
        }
        let scratch = Scratch::new_in(&root, cache)?;
        let admitted = (|| {
            let java = pinned_java(&root, scratch.path())?;
            let admitted = Self { root: root.clone(), jar, java };
            probe(&admitted, scratch.path())?;
            Ok(admitted)
        })();
        scratch.finish(admitted)
    }
}

/// Resolve a workspace-relative regular file, refusing symlinks and non-normal components.
fn contained(root: &Path, relative: &str) -> Result<PathBuf, Refusal> {
    let mut path = root.to_path_buf();
    for component in Path::new(relative).components() {
        let Component::Normal(component) = component else {
            return Err(Refusal(format!("pinned jar escapes the workspace cache: {relative}")));
        };
        path.push(component);
        match fs::symlink_metadata(&path) {
            Ok(meta) if meta.is_symlink() => {
                return Err(Refusal(format!(
                    "pinned jar path contains a symlink: {}",
                    path.display()
                )));
            }
            Ok(_) => {}
            Err(error) if error.kind() == ErrorKind::NotFound => {
                return Err(Refusal(format!(
                    "missing pinned jar {}",
                    root.join(relative).display()
                )));
            }
            Err(error) => return Err(error.into()),
        }
    }
    if !path.is_file() || path.canonicalize()? != path {
        return Err(Refusal(format!("pinned jar escapes the workspace cache: {}", path.display())));
    }
    Ok(path)
}

fn digest(path: &Path) -> Result<String, Refusal> {
    let mut file = File::open(path)?;
    let mut hasher = Sha256::new();
    let mut buffer = vec![0; 1 << 16];
    loop {
        let read = file.read(&mut buffer)?;
        if read == 0 {
            break;
        }
        hasher.update(&buffer[..read]);
    }
    let mut hex = String::with_capacity(64);
    for byte in hasher.finalize() {
        use std::fmt::Write;
        let _ = write!(hex, "{byte:02x}");
    }
    Ok(hex)
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::cell::Cell;

    const EMPTY: &str = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855";

    fn workspace() -> Result<Scratch, Refusal> {
        let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..").canonicalize()?;
        let scratch = Scratch::new(&root)?;
        fs::create_dir_all(scratch.path().join("model/build/tool"))?;
        fs::write(scratch.path().join("model/build/tool/tool.jar"), "")?;
        Ok(scratch)
    }

    fn admit(root: &Path, jar: &str, sha: &str, probed: &Cell<bool>) -> Result<PinnedJar, Refusal> {
        PinnedJar::admit(root, "model/build/tool", jar, sha, |_, _| {
            probed.set(true);
            Ok(())
        })
    }

    #[test]
    fn digest_is_in_process_sha256() -> Result<(), Refusal> {
        let scratch = workspace()?;
        assert_eq!(digest(&scratch.path().join("model/build/tool/tool.jar"))?, EMPTY);
        scratch.finish(Ok(()))
    }

    #[test]
    fn wrong_digest_wrong_java_and_cache_escape_refuse_before_use() -> Result<(), Refusal> {
        let scratch = workspace()?;
        let root = scratch.path();
        let probed = Cell::new(false);
        fs::write(root.join("mise.toml"), "java = \"temurin-27\"\n")?;
        let error = admit(root, "tool.jar", &"0".repeat(64), &probed).err();
        assert!(error.is_some_and(|e| e.0.contains("integrity")));

        fs::write(root.join("mise.toml"), "java = \"temurin-21\"\n")?;
        let error = admit(root, "tool.jar", EMPTY, &probed).err();
        assert!(error.is_some_and(|e| e.0.contains("Java pin")));

        fs::write(root.join("model/outside.jar"), "")?;
        let error = admit(root, "../../outside.jar", EMPTY, &probed).err();
        assert!(error.is_some_and(|e| e.0.contains("escapes")));
        std::os::unix::fs::symlink(
            root.join("model/outside.jar"),
            root.join("model/build/tool/link.jar"),
        )?;
        let error = admit(root, "link.jar", EMPTY, &probed).err();
        assert!(error.is_some_and(|e| e.0.contains("symlink")));
        fs::create_dir_all(root.join("model/build/tool/elsewhere"))?;
        fs::write(root.join("model/build/tool/elsewhere/tool.jar"), "")?;
        std::os::unix::fs::symlink(
            root.join("model/build/tool/elsewhere"),
            root.join("model/build/tool/lib"),
        )?;
        let error = admit(root, "lib/tool.jar", EMPTY, &probed).err();
        assert!(error.is_some_and(|e| e.0.contains("symlink")));
        let error = admit(root, "absent.jar", EMPTY, &probed).err();
        assert!(error.is_some_and(|e| e.0.contains("missing pinned jar")));

        assert!(!probed.get());
        scratch.finish(Ok(()))
    }

    #[test]
    fn exact_jar_and_pinned_java_reach_the_probe() -> Result<(), Refusal> {
        let workspace_root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..");
        let scratch = workspace()?;
        // Pinned Java is a backend prerequisite; without it only the refusals above are evidence.
        if let Err(error) = pinned_java(&workspace_root.canonicalize()?, scratch.path()) {
            let required = std::env::var("UMPIRE_BACKENDS").as_deref() == Ok("require");
            return scratch.finish(if required { Err(error) } else { Ok(()) });
        }
        let root = scratch.path();
        fs::copy(workspace_root.join("mise.toml"), root.join("mise.toml"))?;
        let probed = Cell::new(false);
        let admitted = admit(root, "tool.jar", EMPTY, &probed)?;
        assert!(probed.get());
        assert_eq!(admitted.jar, root.canonicalize()?.join("model/build/tool/tool.jar"));
        let error = PinnedJar::admit(root, "model/build/tool", "tool.jar", EMPTY, |_, _| {
            Err(Refusal("wrong version".into()))
        })
        .err();
        assert!(error.is_some_and(|e| e.0.contains("wrong version")));
        scratch.finish(Ok(()))
    }
}
