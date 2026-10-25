//! Admitted tool roots and the raw operating-system guest compilation context.
use crate::{Error, Result};
use std::{
    collections::BTreeMap,
    env,
    ffi::{OsStr, OsString},
    fs,
    os::unix::ffi::OsStrExt,
    path::{Component, Path, PathBuf},
    process::Command,
};

/// One locally configured framework tool root, independent of the caller workspace.
#[derive(Debug, Clone)]
pub struct ToolRoot(PathBuf);
impl ToolRoot {
    /// Resolve the explicit override or the root that compiled this tool.
    pub fn local() -> Result<Self> {
        Self::admit(
            &env::var_os("UMPIRE_TOOL_ROOT").map_or_else(
                || PathBuf::from(env!("CARGO_MANIFEST_DIR")).join("../.."),
                PathBuf::from,
            ),
        )
    }
    /// Require local configuration; never search parent directories.
    pub fn admit(path: &Path) -> Result<Self> {
        let root = fs::canonicalize(path)?;
        for name in ["Cargo.toml", "mise.toml", "rust-toolchain.toml"] {
            let file = root.join(name);
            if !file.is_file() || !fs::canonicalize(&file)?.starts_with(&root) {
                return Err(Error::Invalid(format!(
                    "missing or escaping framework configuration: {name}"
                )));
            }
        }
        Ok(Self(root))
    }
    /// The admitted canonical framework root.
    pub fn path(&self) -> &Path {
        &self.0
    }
}

pub(crate) type Environment = BTreeMap<OsString, OsString>;

pub(crate) fn value<'a>(environment: &'a Environment, name: &str) -> Result<&'a OsStr> {
    environment
        .get(OsStr::new(name))
        .map(OsString::as_os_str)
        .ok_or_else(|| Error::Invalid(format!("missing {name}")))
}

pub(crate) fn lexical(path: &Path) -> Result<PathBuf> {
    let path = if path.is_absolute() { path.to_owned() } else { env::current_dir()?.join(path) };
    let mut normalized = PathBuf::new();
    for component in path.components() {
        match component {
            Component::ParentDir => {
                normalized.pop();
            }
            Component::CurDir => (),
            other => normalized.push(other.as_os_str()),
        }
    }
    Ok(normalized)
}

pub(crate) fn caller_path(path: PathBuf, base: &Path) -> PathBuf {
    if path.is_absolute() { path } else { base.join(path) }
}

fn physical(path: &Path) -> Result<PathBuf> {
    if path.exists() { Ok(fs::canonicalize(path)?) } else { lexical(path) }
}

/// Raw OS values cross the compiler boundary without Unicode conversion.
#[derive(Debug, Clone)]
pub(crate) struct Context {
    pub tool: PathBuf,
    pub workspace: PathBuf,
    pub sysroot: PathBuf,
    pub cargo_home: PathBuf,
    pub target: PathBuf,
    pub lock: PathBuf,
    pub compiler: OsString,
    pub previous: OsString,
}
impl Context {
    fn entries(&self) -> [(&'static str, &OsStr); 8] {
        [
            ("UMPIRE_GUEST_TOOL_ROOT", self.tool.as_os_str()),
            ("UMPIRE_GUEST_WORKSPACE", self.workspace.as_os_str()),
            ("UMPIRE_GUEST_SYSROOT", self.sysroot.as_os_str()),
            ("UMPIRE_GUEST_CARGO_HOME", self.cargo_home.as_os_str()),
            ("UMPIRE_GUEST_TARGET", self.target.as_os_str()),
            ("UMPIRE_GUEST_LOCK", self.lock.as_os_str()),
            ("UMPIRE_GUEST_COMPILER", &self.compiler),
            ("UMPIRE_GUEST_PREVIOUS_WRAPPER", &self.previous),
        ]
    }
    pub(crate) fn encode(&self, command: &mut Command) {
        command.envs(self.entries());
    }
    pub(crate) fn decode(environment: &Environment) -> Result<Self> {
        Ok(Self {
            tool: value(environment, "UMPIRE_GUEST_TOOL_ROOT")?.into(),
            workspace: value(environment, "UMPIRE_GUEST_WORKSPACE")?.into(),
            sysroot: value(environment, "UMPIRE_GUEST_SYSROOT")?.into(),
            cargo_home: value(environment, "UMPIRE_GUEST_CARGO_HOME")?.into(),
            target: value(environment, "UMPIRE_GUEST_TARGET")?.into(),
            lock: value(environment, "UMPIRE_GUEST_LOCK")?.into(),
            compiler: value(environment, "UMPIRE_GUEST_COMPILER")?.into(),
            previous: environment
                .get(OsStr::new("UMPIRE_GUEST_PREVIOUS_WRAPPER"))
                .cloned()
                .unwrap_or_default(),
        })
    }
    pub(crate) fn remaps(&self) -> Result<Remaps> {
        let mut roots = vec![
            (physical(&self.workspace)?, b"/workspace".as_slice()),
            (physical(&self.sysroot)?, b"/rust/sysroot".as_slice()),
            (physical(&self.cargo_home.join("registry"))?, b"/cargo/registry".as_slice()),
            (physical(&self.target)?, b"/target".as_slice()),
        ];
        let tool = physical(&self.tool)?;
        if tool != physical(&self.workspace)? {
            roots.push((tool, b"/framework"));
        }
        roots.sort_by_key(|(path, _)| std::cmp::Reverse(path.as_os_str().as_bytes().len()));
        Ok(Remaps(roots))
    }
}

pub(crate) struct Remaps(Vec<(PathBuf, &'static [u8])>);
impl Remaps {
    pub(crate) fn new(environment: &Environment) -> Result<Self> {
        Context::decode(environment)?.remaps()
    }
    pub(crate) fn path(&self, path: &Path) -> Result<Vec<u8>> {
        let path = physical(path)?;
        for (root, logical) in &self.0 {
            if let Ok(relative) = path.strip_prefix(root) {
                let mut result = logical.to_vec();
                if !relative.as_os_str().is_empty() {
                    result.push(b'/');
                    result.extend(relative.as_os_str().as_bytes());
                }
                return Ok(result);
            }
        }
        Err(Error::Invalid("guest source is outside the remapped roots".into()))
    }
    pub(crate) fn token(&self, token: &[u8]) -> Vec<u8> {
        let mut result = Vec::new();
        let mut index = 0;
        while index < token.len() {
            let found = self.0.iter().find(|(root, _)| {
                let root = root.as_os_str().as_bytes();
                (index == 0 || token[index - 1] == b'=')
                    && token[index..].starts_with(root)
                    && (index + root.len() == token.len() || token[index + root.len()] == b'/')
            });
            if let Some((root, logical)) = found {
                result.extend(*logical);
                index += root.as_os_str().as_bytes().len();
            } else {
                result.push(token[index]);
                index += 1;
            }
        }
        result
    }
    pub(crate) fn flags(&self) -> impl Iterator<Item = OsString> + '_ {
        self.0.iter().rev().map(|(path, logical)| {
            let mut flag = OsString::from("--remap-path-prefix=");
            flag.push(path);
            flag.push("=");
            flag.push(OsStr::from_bytes(logical));
            flag
        })
    }
}

#[cfg(test)]
#[allow(clippy::unwrap_used, clippy::expect_used)]
mod tests {
    use super::*;
    use std::os::unix::ffi::OsStringExt;

    #[test]
    fn caller_paths_preserve_relative_and_absolute_os_bytes() {
        let base = PathBuf::from(OsString::from_vec(b"/caller-\xff".to_vec()));
        let relative = PathBuf::from(OsString::from_vec(b"out-\xfe".to_vec()));
        assert_eq!(caller_path(relative.clone(), &base), base.join(&relative));
        let absolute = PathBuf::from(OsString::from_vec(b"/target-\xfd".to_vec()));
        assert_eq!(caller_path(absolute.clone(), &base), absolute);
    }

    #[test]
    fn raw_context_roundtrips_and_longest_root_drives_flags_and_identity() {
        let raw = PathBuf::from(OsString::from_vec(b"/checkout-\xff/framework".to_vec()));
        let context = Context {
            tool: raw.clone(),
            workspace: PathBuf::from("/checkout-\u{fffd}"),
            sysroot: "/rust".into(),
            cargo_home: "/cargo".into(),
            target: raw.join("target"),
            lock: raw.join("lock"),
            compiler: OsString::from_vec(b"rustc-\xff".to_vec()),
            previous: OsString::from_vec(b"wrapper-\xff".to_vec()),
        };
        let mut command = Command::new("unused");
        context.encode(&mut command);
        let encoded = command
            .get_envs()
            .map(|(key, value)| (key.to_owned(), value.unwrap().to_owned()))
            .collect();
        let decoded = Context::decode(&encoded).unwrap();
        assert_eq!(decoded.compiler, context.compiler);
        assert_eq!(decoded.previous, context.previous);
        assert_eq!(decoded.tool, raw);
        let maps = decoded.remaps().unwrap();
        assert_eq!(maps.path(&raw.join("src/lib.rs")).unwrap(), b"/framework/src/lib.rs");
        assert_eq!(maps.path(&raw.join("target/out")).unwrap(), b"/target/out");
        assert!(maps.path(Path::new("/undeclared/file")).is_err());
        let mut token = b"root=".to_vec();
        token.extend(raw.join("target/out").as_os_str().as_bytes());
        assert_eq!(maps.token(&token), b"root=/target/out");
        let flags: Vec<_> = maps.flags().collect();
        assert!(flags.last().unwrap().as_bytes().ends_with(b"=/target"));
    }

    #[test]
    fn local_tool_admission_refuses_missing_configuration_and_escaping_links() {
        let fixture = tempfile::tempdir().unwrap();
        let root = fixture.path().join("framework");
        fs::create_dir(&root).unwrap();
        assert!(ToolRoot::admit(&root).is_err());
        for name in ["Cargo.toml", "mise.toml", "rust-toolchain.toml"] {
            fs::write(root.join(name), "").unwrap();
        }
        assert_eq!(ToolRoot::admit(&root).unwrap().path(), root.canonicalize().unwrap());
        fs::remove_file(root.join("mise.toml")).unwrap();
        fs::write(fixture.path().join("outside"), "").unwrap();
        std::os::unix::fs::symlink(fixture.path().join("outside"), root.join("mise.toml")).unwrap();
        assert!(ToolRoot::admit(&root).is_err());
    }

    #[test]
    fn driver_lock_and_compiler_each_invalidate_cache() {
        let baseline = crate::guest::cache_key(b"driver", b"lock", "compiler").unwrap();
        for (driver, lock, compiler) in [
            (b"changed".as_slice(), b"lock".as_slice(), "compiler"),
            (b"driver", b"changed", "compiler"),
            (b"driver", b"lock", "changed"),
        ] {
            assert_ne!(baseline, crate::guest::cache_key(driver, lock, compiler).unwrap());
        }
        assert_eq!(baseline, crate::guest::cache_key(b"driver", b"lock", "compiler").unwrap());
    }
}
