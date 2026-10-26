use std::{
    fs, io,
    io::Write,
    path::{Component, Path, PathBuf},
    process::{Command, ExitStatus},
    sync::atomic::AtomicBool,
    time::Duration,
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

fn supervised(
    root: &Path,
    command: &mut Command,
    label: &str,
    limits: crate::Limits,
    cancelled: &AtomicBool,
) -> io::Result<(ExitStatus, String, String)> {
    let scratch = crate::Scratch::new_in(root, "build/mise-jars/runs")
        .map_err(io::Error::other)?;
    let result = (|| {
        let (status, output) = crate::run_process_cancelled(
            command, scratch.path(), label, limits, None, cancelled,
        ).map_err(|failure| crate::Refusal(format!("{}; {}", failure.refusal, failure.output)))?;
        let stdout = crate::bounded_text(&scratch.path().join(format!("{label}.stdout")), limits.output_bytes)?;
        Ok::<_, crate::Refusal>((status, stdout, output))
    })();
    scratch.finish(result).map_err(io::Error::other)
}

/// Require mise to resolve the same exact-version installation under process limits.
pub fn resolve(root: &Path, tool: JarTool) -> io::Result<PathBuf> {
    resolve_cancelled(root, tool, crate::Limits::default(), &AtomicBool::new(false))
}

/// Resolve with the caller's cancellation token and process limits.
pub fn resolve_cancelled(
    root: &Path,
    tool: JarTool,
    limits: crate::Limits,
    cancelled: &AtomicBool,
) -> io::Result<PathBuf> {
    let root = root.canonicalize()?;
    let jar = installed_jar(&root, tool)?;
    let relative = Path::new("build/mise-jars/data/installs").join(tool.installation());
    let mut command = command(&root, tool)?;
    command.args(["where", tool.selector()]);
    let (status, location, output) = supervised(&root, &mut command, "mise-where", limits, cancelled)?;
    if !status.success() || Path::new(location.trim()) != root.join(relative) {
        return Err(refusal(format!("mise did not resolve the exact pinned installation: {status}; {output}")));
    }
    Ok(jar)
}

/// Explicit provisioning with a bounded subprocess; mise owns extraction and install locking.
pub fn install(root: &Path, tool: JarTool) -> io::Result<PathBuf> {
    install_cancelled(root, tool,
        crate::Limits { timeout: Duration::from_secs(180), ..crate::Limits::default() },
        &AtomicBool::new(false))
}

/// Provision using existing process supervision and a caller-owned cancellation token.
pub fn install_cancelled(
    root: &Path,
    tool: JarTool,
    limits: crate::Limits,
    cancelled: &AtomicBool,
) -> io::Result<PathBuf> {
    let mut command = command(root, tool)?;
    command.args(["install", tool.selector()]);
    let (status, _, output) = supervised(root, &mut command, "mise-install", limits, cancelled)?;
    io::stderr().write_all(output.as_bytes())?;
    if !status.success() { return Err(refusal(format!("mise jar installation failed: {status}"))); }
    resolve_cancelled(root, tool,
        crate::Limits { timeout: limits.timeout.min(Duration::from_secs(60)), ..limits }, cancelled)
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

    fn fixture(root: &Path, script: &str) -> Command {
        let mut command = Command::new("/bin/sh");
        command.args(["-c", script]).current_dir(root);
        command
    }

    fn cleaned(root: &Path) -> io::Result<()> {
        assert_eq!(fs::read_dir(root.join("build/mise-jars/runs"))?.count(), 0);
        Ok(())
    }

    #[test]
    fn cancellation_and_timeout_refuse_and_clean_owned_capture() -> io::Result<()> {
        let scratch = tempfile::tempdir()?;
        let root = scratch.path().canonicalize()?;
        let mut command = fixture(&root, "touch should-not-run");
        let error = supervised(&root, &mut command, "cancelled", crate::Limits::default(),
            &AtomicBool::new(true)).err().ok_or_else(|| refusal("cancelled process ran"))?;
        assert!(error.to_string().contains("cancelled"));
        assert!(!root.join("should-not-run").exists());
        cleaned(&root)?;

        let mut command = fixture(&root, "sleep 30");
        let limits = crate::Limits { timeout: Duration::from_millis(20), ..crate::Limits::default() };
        let error = supervised(&root, &mut command, "timeout", limits, &AtomicBool::new(false))
            .err().ok_or_else(|| refusal("timed process completed"))?;
        assert!(error.to_string().contains("limit"));
        cleaned(&root)
    }

    #[test]
    fn interrupted_partial_install_never_reaches_resolution() -> io::Result<()> {
        let scratch = tempfile::tempdir()?;
        let root = scratch.path().canonicalize()?;
        let destination = root.join("build/mise-jars/data/installs/http-tlc/1.7.4");
        fs::create_dir_all(&destination)?;
        let mut command = fixture(&root, "printf partial > partial-download; sleep 30");
        command.current_dir(&destination);
        let limits = crate::Limits { timeout: Duration::from_millis(30), ..crate::Limits::default() };
        assert!(supervised(&root, &mut command, "interrupted", limits, &AtomicBool::new(false)).is_err());
        assert!(installed_jar(&root, JarTool::Tlc).is_err());
        // Installer leftovers are mise-owned; our cleanup must not delete a shared installation.
        assert!(destination.is_dir());
        cleaned(&root)
    }

    #[test]
    fn in_flight_cancellation_uses_existing_supervisor() -> io::Result<()> {
        use std::sync::atomic::Ordering;
        let scratch = tempfile::tempdir()?;
        let root = scratch.path().canonicalize()?;
        let cancelled = AtomicBool::new(false);
        std::thread::scope(|scope| {
            let worker = scope.spawn(|| {
                let mut command = fixture(&root, "printf ready > started; sleep 30");
                supervised(&root, &mut command, "cancel-running", crate::Limits::default(), &cancelled)
            });
            let started = std::time::Instant::now();
            while !root.join("started").exists() && started.elapsed() < Duration::from_secs(10) {
                std::thread::yield_now();
            }
            cancelled.store(true, Ordering::Relaxed);
            let result = worker.join().map_err(|_| refusal("supervision worker panicked"))?;
            let error = result.err().ok_or_else(|| refusal("cancelled process completed"))?;
            assert!(error.to_string().contains("cancelled"));
            cleaned(&root)
        })
    }

    #[test]
    fn concurrent_provider_capture_and_failure_propagation_are_independent() -> io::Result<()> {
        let scratch = tempfile::tempdir()?;
        let root = scratch.path().canonicalize()?;
        std::thread::scope(|scope| {
            let workers: Vec<_> = [0, 42].into_iter().map(|code| {
                let root = &root;
                scope.spawn(move || {
                    let mut command = fixture(root, &format!("printf provider-output; exit {code}"));
                    supervised(root, &mut command, "mise-install", crate::Limits::default(), &AtomicBool::new(false))
                })
            }).collect();
            for (worker, expected) in workers.into_iter().zip([0, 42]) {
                let (status, stdout, output) = worker.join()
                    .map_err(|_| refusal("concurrent supervision worker panicked"))??;
                assert_eq!(status.code(), Some(expected));
                assert_eq!(stdout, "provider-output");
                assert_eq!(output, "provider-output");
            }
            cleaned(&root)
        })
    }

}
