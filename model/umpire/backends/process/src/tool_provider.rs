use std::{
    fs, io,
    path::{Component, Path, PathBuf},
    process::Command,
};

/// Official release identities configured by the isolated mise jar environment.
#[derive(Clone, Copy)]
pub enum JarTool {
    /// Required TLC release.
    Tlc,
    /// Explicitly selected Apalache release.
    Apalache,
}

impl JarTool {
    /// Exact version selector, never an alias.
    pub fn selector(self) -> &'static str {
        match self { Self::Tlc => "http:tlc@1.7.4", Self::Apalache => "http:apalache@0.62.4" }
    }

    fn installation(self) -> &'static str {
        match self { Self::Tlc => "http-tlc/1.7.4", Self::Apalache => "http-apalache/0.62.4" }
    }

    fn jar(self) -> &'static str {
        match self { Self::Tlc => "tla2tools.jar", Self::Apalache => "lib/apalache.jar" }
    }
}

fn refusal(message: impl Into<String>) -> io::Error {
    io::Error::other(message.into())
}

/// Admit a relative path without symlinks or non-normal components.
pub fn contained(root: &Path, relative: &Path) -> io::Result<PathBuf> {
    let mut path = root.to_path_buf();
    for component in relative.components() {
        let Component::Normal(component) = component else {
            return Err(refusal("pinned jar escapes the mise install root"));
        };
        path.push(component);
        let meta = fs::symlink_metadata(&path).map_err(|error| {
            if error.kind() == io::ErrorKind::NotFound {
                refusal(format!("missing pinned jar {}", path.display()))
            } else { error }
        })?;
        if meta.is_symlink() {
            return Err(refusal(format!("pinned jar path contains a symlink: {}", path.display())));
        }
    }
    if path.canonicalize()? != path {
        return Err(refusal("pinned jar escapes the mise install root"));
    }
    Ok(path)
}

fn directory(root: &Path, relative: &Path) -> io::Result<PathBuf> {
    let mut path = root.to_path_buf();
    for component in relative.components() {
        let Component::Normal(component) = component else {
            return Err(refusal("mise data directory escapes framework"));
        };
        path.push(component);
        match fs::create_dir(&path) {
            Ok(()) => (),
            Err(error) if error.kind() == io::ErrorKind::AlreadyExists => (),
            Err(error) => return Err(error),
        }
        let metadata = fs::symlink_metadata(&path)?;
        if !metadata.is_dir() || path.canonicalize()? != path {
            return Err(refusal("mise data directory contains a symlink or escapes framework"));
        }
    }
    Ok(path)
}

/// Construct a jar-only mise command without inherited configuration selectors.
pub fn command(root: &Path, tool: JarTool) -> io::Result<Command> {
    let root = root.canonicalize()?;
    let context = contained(&root, Path::new("mise-tools"))?;
    for config in ["mise.toml", "mise.apalache.toml", "empty.toml"] {
        if !contained(&context, Path::new(config))?.is_file() {
            return Err(refusal("mise jar configuration is not a regular file"));
        }
    }
    let owned = directory(&root, Path::new("build/mise-jars"))?;
    for relative in ["data/installs", "cache", "state", "config"] {
        directory(&owned, Path::new(relative))?;
    }
    let mut command = Command::new("mise");
    // Inherited mise selectors must not admit Temporal or user configuration.
    for (key, _) in std::env::vars_os() {
        if key.to_string_lossy().starts_with("MISE_") { command.env_remove(key); }
    }
    command.current_dir(&context)
        .env("MISE_DATA_DIR", owned.join("data"))
        .env("MISE_CACHE_DIR", owned.join("cache"))
        .env("MISE_STATE_DIR", owned.join("state"))
        .env("MISE_CONFIG_DIR", owned.join("config"))
        .env("MISE_GLOBAL_CONFIG_FILE", context.join("empty.toml"))
        .env("MISE_SYSTEM_CONFIG_FILE", context.join("empty.toml"))
        .env("MISE_CEILING_PATHS", &root)
        .env("MISE_TRUSTED_CONFIG_PATHS", &context)
        .env("MISE_AUTO_INSTALL", "false")
        .env("MISE_EXEC_AUTO_INSTALL", "false")
        .env("MISE_NOT_FOUND_AUTO_INSTALL", "false")
        .env("MISE_YES", "1")
        .env("MISE_COLOR", "0");
    if matches!(tool, JarTool::Apalache) { command.args(["--env", "apalache"]); }
    Ok(command)
}

/// Admit the expected exact-version regular file before resolver invocation.
pub fn installed_jar(root: &Path, tool: JarTool) -> io::Result<PathBuf> {
    let root = root.canonicalize()?;
    let relative = Path::new("build/mise-jars/data/installs").join(tool.installation());
    let jar = contained(&root, &relative.join(tool.jar()))?;
    if !fs::symlink_metadata(&jar)?.is_file() {
        return Err(refusal("pinned jar is not a regular file"));
    }
    Ok(jar)
}

/// Require mise to resolve the same exact-version installation.
pub fn resolve(root: &Path, tool: JarTool) -> io::Result<PathBuf> {
    let root = root.canonicalize()?;
    let jar = installed_jar(&root, tool)?;
    let relative = Path::new("build/mise-jars/data/installs").join(tool.installation());
    let output = command(&root, tool)?.args(["where", tool.selector()]).output()?;
    let location = String::from_utf8(output.stdout).map_err(|_| refusal("mise install path is not UTF-8"))?;
    if !output.status.success() || Path::new(location.trim()) != root.join(relative) {
        return Err(refusal(format!("mise did not resolve the exact pinned installation: {}; {}",
            output.status, String::from_utf8_lossy(&output.stderr))));
    }
    Ok(jar)
}

/// Explicit provisioning; mise owns downloads, extraction and installation locking.
pub fn install(root: &Path, tool: JarTool) -> io::Result<PathBuf> {
    let status = command(root, tool)?.args(["install", tool.selector()])
        .stdout(std::process::Stdio::from(std::io::stderr())).stderr(std::process::Stdio::inherit()).status()?;
    if !status.success() { return Err(refusal(format!("mise jar installation failed: {status}"))); }
    resolve(root, tool)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn scoped_commands_only_select_optional_apalache_explicitly() -> io::Result<()> {
        let scratch = tempfile::tempdir()?;
        let root = scratch.path().canonicalize()?;
        fs::create_dir(root.as_path().join("mise-tools"))?;
        for file in ["mise.toml", "mise.apalache.toml", "empty.toml"] {
            fs::write(root.as_path().join("mise-tools").join(file), "")?;
        }
        for tool in [JarTool::Tlc, JarTool::Apalache] {
            let command = command(root.as_path(), tool)?;
            let args: Vec<_> = command.get_args().collect();
            if matches!(tool, JarTool::Tlc) { assert!(args.is_empty()); }
            else { assert_eq!(args, ["--env", "apalache"]); }
            let environment: std::collections::BTreeMap<_, _> = command.get_envs().collect();
            for key in ["MISE_AUTO_INSTALL", "MISE_EXEC_AUTO_INSTALL", "MISE_NOT_FOUND_AUTO_INSTALL"] {
                assert_eq!(environment.get(std::ffi::OsStr::new(key)),
                    Some(&Some(std::ffi::OsStr::new("false"))));
            }
            assert_eq!(command.get_current_dir(), Some(root.as_path().join("mise-tools").as_path()));
            assert_eq!(environment.get(std::ffi::OsStr::new("MISE_CEILING_PATHS")),
                Some(&Some(root.as_path().as_os_str())));
        }
        Ok(())
    }
}
