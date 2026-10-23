use crate::{
    campaign::{origin, seeds::SeedSelection},
    install::manifest,
};
use rustix::{fs::FlockOperation, io::FdFlags};
use std::{
    ffi::{OsStr, OsString},
    fs::{self, File, OpenOptions},
    os::unix::{
        ffi::OsStrExt,
        fs::{DirBuilderExt, MetadataExt, OpenOptionsExt, PermissionsExt},
        process::{CommandExt, ExitStatusExt},
    },
    path::{Path, PathBuf},
    process::{Command, Stdio},
};

struct Arguments {
    installation: Option<PathBuf>,
    seeds: String,
    artifacts: PathBuf,
    child: Vec<OsString>,
}

fn parse(arguments: &[OsString], installation: Option<&Path>) -> Result<Arguments, super::Error> {
    let separator = arguments
        .iter()
        .position(|arg| arg == "--")
        .ok_or("wrapper requires -- before the child command")?;
    if separator + 1 == arguments.len() || arguments[separator + 1].is_empty() {
        return Err("wrapper requires a child command after --".into());
    }
    let mut parsed = Arguments {
        installation: installation.map(Path::to_path_buf),
        seeds: "0".into(),
        artifacts: ".gomad/artifacts".into(),
        child: arguments[separator + 1..].to_vec(),
    };
    let mut index = 0;
    while index < separator {
        let argument = arguments[index].as_bytes();
        let (name, inline) = argument
            .iter()
            .position(|byte| *byte == b'=')
            .map_or((argument, None), |at| {
                (&argument[..at], Some(&argument[at + 1..]))
            });
        if !matches!(name, b"--installation" | b"--seeds" | b"--artifacts") {
            return Err(format!(
                "unsupported wrapper flag: {}",
                arguments[index].to_string_lossy()
            )
            .into());
        }
        let value = if let Some(value) = inline {
            OsStr::from_bytes(value)
        } else {
            index += 1;
            arguments
                .get(index)
                .filter(|_| index < separator)
                .map(OsString::as_os_str)
                .ok_or_else(|| {
                    format!("flag needs an argument: {}", String::from_utf8_lossy(name))
                })?
        };
        if value.is_empty() {
            return Err("wrapper flag value must not be empty".into());
        }
        match name {
            b"--installation" => parsed.installation = Some(value.into()),
            b"--artifacts" => parsed.artifacts = value.into(),
            _ => {
                value
                    .to_str()
                    .ok_or("seeds must be UTF-8")?
                    .clone_into(&mut parsed.seeds);
                SeedSelection::parse(&parsed.seeds)?;
            }
        }
        index += 1;
    }
    Ok(parsed)
}

fn filesystem(error: impl std::fmt::Display) -> super::Error {
    super::Error {
        status: 3,
        message: format!("wrapper filesystem: {error}"),
    }
}

fn executable(path: &Path) -> bool {
    fs::metadata(path).is_ok_and(|meta| meta.is_file() && meta.permissions().mode() & 0o111 != 0)
}

fn resolve(command: &OsStr, search: &OsStr) -> Result<PathBuf, super::Error> {
    let path = Path::new(command);
    if command.as_bytes().contains(&b'/') {
        if executable(path) {
            return std::path::absolute(path)
                .map_err(|error| super::Error::from(error.to_string()));
        }
    } else {
        for directory in std::env::split_paths(search) {
            let candidate = directory.join(path);
            if executable(&candidate) {
                return std::path::absolute(candidate)
                    .map_err(|error| super::Error::from(error.to_string()));
            }
        }
    }
    Err(format!("command not found: {}", command.to_string_lossy()).into())
}

fn private_directory(path: &Path) -> Result<(), super::Error> {
    let meta = fs::symlink_metadata(path).map_err(filesystem)?;
    if !meta.is_dir()
        || meta.uid() != rustix::process::geteuid().as_raw()
        || meta.permissions().mode() & 0o777 != 0o700
    {
        return Err(filesystem(
            "shim directory must be owned by the current user with mode 0700",
        ));
    }
    Ok(())
}

fn lock(path: &Path) -> Result<File, super::Error> {
    let file = OpenOptions::new()
        .read(true)
        .write(true)
        .create(true)
        .mode(0o600)
        .custom_flags(
            (rustix::fs::OFlags::NOFOLLOW | rustix::fs::OFlags::NONBLOCK)
                .bits()
                .cast_signed(),
        )
        .open(path)
        .map_err(filesystem)?;
    let meta = file.metadata().map_err(filesystem)?;
    if !meta.is_file()
        || meta.uid() != rustix::process::geteuid().as_raw()
        || meta.nlink() != 1
        || meta.permissions().mode() & 0o777 != 0o600
    {
        return Err(filesystem("unsafe wrapper lock"));
    }
    Ok(file)
}

struct Directory {
    root: PathBuf,
    path: PathBuf,
    identity: String,
    guard: File,
}

impl Directory {
    fn create(cache: &Path, binary: &Path) -> Result<Self, super::Error> {
        let root = cache.join("wrappers");
        fs::DirBuilder::new()
            .recursive(true)
            .mode(0o700)
            .create(&root)
            .map_err(filesystem)?;
        private_directory(&root)?;
        let registry = lock(&root.join(".lock"))?;
        rustix::fs::flock(&registry, FlockOperation::LockExclusive).map_err(filesystem)?;
        for entry in fs::read_dir(&root).map_err(filesystem)? {
            let entry = entry.map_err(filesystem)?;
            let name = entry.file_name();
            let Some(identity) = name.to_str().and_then(|name| name.strip_prefix("run-")) else {
                continue;
            };
            if !crate::evidence::canonical::valid_digest(identity) {
                continue;
            }
            let path = entry.path();
            private_directory(&path)?;
            let guard = lock(&path.join(".lock"))?;
            match rustix::fs::flock(&guard, FlockOperation::NonBlockingLockExclusive) {
                Ok(()) => fs::remove_dir_all(path).map_err(filesystem)?,
                Err(rustix::io::Errno::WOULDBLOCK) => (),
                Err(error) => return Err(filesystem(error)),
            }
        }
        let identity = origin::identity()?;
        let path = root.join(format!("run-{identity}"));
        fs::DirBuilder::new()
            .mode(0o700)
            .create(&path)
            .map_err(filesystem)?;
        let result = (|| {
            let guard = lock(&path.join(".lock"))?;
            rustix::fs::flock(&guard, FlockOperation::LockExclusive).map_err(filesystem)?;
            std::os::unix::fs::symlink(binary, path.join("go")).map_err(filesystem)?;
            Ok(Self {
                root,
                path: path.clone(),
                identity,
                guard,
            })
        })();
        if result.is_err() {
            fs::remove_dir_all(&path).map_err(filesystem)?;
        }
        result
    }

    fn clean(self) -> Result<(), super::Error> {
        private_directory(&self.root)?;
        let registry = lock(&self.root.join(".lock"))?;
        rustix::fs::flock(&registry, FlockOperation::LockExclusive).map_err(filesystem)?;
        private_directory(&self.path)?;
        let held = self.guard.metadata().map_err(filesystem)?;
        let current = fs::symlink_metadata(self.path.join(".lock")).map_err(filesystem)?;
        if held.dev() != current.dev() || held.ino() != current.ino() {
            return Err(filesystem("wrapper directory lock substituted"));
        }
        drop(self.guard);
        let probe = lock(&self.path.join(".lock"))?;
        match rustix::fs::flock(&probe, FlockOperation::NonBlockingLockExclusive) {
            Ok(()) => fs::remove_dir_all(&self.path).map_err(filesystem),
            Err(rustix::io::Errno::WOULDBLOCK) => Ok(()),
            Err(error) => Err(filesystem(error)),
        }
    }
}

fn cache_root() -> Result<PathBuf, super::Error> {
    let root =
        if let Some(cache) = std::env::var_os("XDG_CACHE_HOME").filter(|value| !value.is_empty()) {
            PathBuf::from(cache).join("gomad")
        } else {
            PathBuf::from(std::env::var_os("HOME").ok_or("HOME or XDG_CACHE_HOME is required")?)
                .join(".cache/gomad")
        };
    std::path::absolute(root).map_err(filesystem)
}

fn launch(
    arguments: &Arguments,
    original_path: &OsStr,
    compiler: &Path,
    installation: &Path,
    binary: &Path,
    directory: &Directory,
) -> Result<i32, super::Error> {
    let search = std::env::join_paths(
        std::iter::once(directory.path.clone()).chain(std::env::split_paths(original_path)),
    )
    .map_err(filesystem)?;
    let child = resolve(&arguments.child[0], &search)?;
    let artifacts = std::path::absolute(&arguments.artifacts).map_err(filesystem)?;
    // Descendants retain the lease if this wrapper is interrupted before they exit.
    rustix::io::fcntl_setfd(&directory.guard, FdFlags::empty()).map_err(filesystem)?;
    let status = Command::new(child)
        .arg0(&arguments.child[0])
        .args(&arguments.child[1..])
        .env("PATH", search)
        .env("GOMAD_SHIM_BINARY", binary)
        .env("GOMAD_SHIM_GO", compiler)
        .env("GOMAD_SHIM_INSTALLATION", installation)
        .env("GOMAD_SHIM_ARTIFACTS", artifacts)
        .env("GOMAD_SHIM_SEEDS", &arguments.seeds)
        .env("GOMAD_SHIM_RUN", &directory.identity)
        .env("GOMAD_SHIM_RUN_DIR", &directory.path)
        .env_remove("GOMAD_SHIM_INVOCATION")
        .env_remove("GOMAD_SHIM_INVOCATION_SHA256")
        .env_remove("GOMAD_SHIM_BUILD_CHILD")
        .env_remove("GOMAD_SHIM_BUILD_CHILD_SHA256")
        .stdin(Stdio::inherit())
        .stdout(Stdio::inherit())
        .stderr(Stdio::inherit())
        .status()
        .map_err(|error| super::Error::from(error.to_string()))?;
    Ok(status
        .code()
        .unwrap_or_else(|| 128 + status.signal().unwrap_or(1)))
}

/// # Errors
/// Refuses malformed wrapper flags, invalid installations and unavailable child commands.
pub fn run(
    arguments: &[OsString],
    explicit_installation: Option<&Path>,
) -> Result<i32, super::Error> {
    let mut arguments = parse(arguments, explicit_installation)?;
    let original_path = std::env::var_os("PATH").unwrap_or_default();
    let compiler = fs::canonicalize(resolve(OsStr::new("go"), &original_path)?)
        .map_err(|error| super::Error::from(error.to_string()))?;
    let selected = arguments.installation.take().or_else(|| {
        std::env::var_os("GOMAD_WASM_INSTALLATION")
            .filter(|value| !value.is_empty())
            .map(PathBuf::from)
    });
    let (installation, cache) = if let Some(selected) = selected {
        let installed = manifest::resolve(Some(&selected))?;
        if fs::canonicalize(&installed.manifest.compiler_path).map_err(filesystem)? != compiler {
            return Err("compiler on original PATH does not match installation".into());
        }
        manifest::executable(&compiler, &installed.manifest.compiler_sha256)?;
        (installed.path, installed.manifest.cache_root)
    } else {
        crate::config::repo::Repo::optional(&std::env::current_dir().map_err(filesystem)?)?;
        (PathBuf::new(), cache_root()?)
    };
    let binary =
        fs::canonicalize(std::env::current_exe().map_err(filesystem)?).map_err(filesystem)?;
    let directory = Directory::create(&cache, &binary)?;
    let result = launch(
        &arguments,
        &original_path,
        &compiler,
        &installation,
        &binary,
        &directory,
    );
    directory.clean()?;
    result
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::os::unix::ffi::OsStringExt as _;

    #[test]
    fn wrapper_preserves_child_flag_and_non_utf8_bytes() {
        let raw = OsString::from_vec(vec![0xff, b'a']);
        let args = [
            OsString::from("--seeds=7,8"),
            OsString::from("--artifacts"),
            OsString::from("out"),
            OsString::from("--"),
            OsString::from("replay"),
            OsString::from("--installation=child.json"),
            raw.clone(),
        ];
        let parsed = parse(&args, None).unwrap_or_else(|e| panic!("{}", e.message));
        assert_eq!(parsed.seeds, "7,8");
        assert_eq!(parsed.artifacts, PathBuf::from("out"));
        assert_eq!(parsed.installation, None);
        assert_eq!(parsed.child, args[4..]);
        assert_eq!(parsed.child[2], raw);
    }

    #[test]
    fn wrapper_requires_separator_and_child() {
        for args in [vec!["make"], vec!["--"], vec!["--json", "--", "true"]] {
            assert!(
                parse(
                    &args.into_iter().map(OsString::from).collect::<Vec<_>>(),
                    None
                )
                .is_err()
            );
        }
    }

    #[test]
    fn wrapper_installation_prefix_precedence_is_explicit() {
        let args = ["--installation=chosen.json", "--", "true"].map(OsString::from);
        let parsed = parse(&args, Some(Path::new("earlier.json")))
            .unwrap_or_else(|e| panic!("{}", e.message));
        assert_eq!(parsed.installation, Some(PathBuf::from("chosen.json")));
    }

    #[test]
    fn wrapper_path_search_keeps_explicit_relative_paths() {
        let root =
            crate::campaign::prepare::process::Temporary::new(&std::env::temp_dir()).unwrap();
        let path = root.0.join("child");
        fs::write(&path, b"#!/bin/sh\nexit 0\n").unwrap();
        fs::set_permissions(&path, fs::Permissions::from_mode(0o700)).unwrap();
        assert_eq!(
            resolve(OsStr::new("child"), root.0.as_os_str())
                .unwrap_or_else(|e| panic!("{}", e.message)),
            path
        );
        assert!(resolve(OsStr::new("./missing-child"), root.0.as_os_str()).is_err());
    }

    #[test]
    fn wrapper_directory_cleanup_preserves_live_descendant_lease() {
        let root =
            crate::campaign::prepare::process::Temporary::new(&std::env::temp_dir()).unwrap();
        let binary = std::env::current_exe().unwrap();
        let first = Directory::create(&root.0, &binary).unwrap_or_else(|e| panic!("{}", e.message));
        let stale = first.path.clone();
        assert_eq!(
            fs::metadata(&stale).unwrap().permissions().mode() & 0o777,
            0o700
        );
        assert_eq!(fs::read_link(stale.join("go")).unwrap(), binary);
        let descendant = first.guard.try_clone().unwrap();
        first.clean().unwrap_or_else(|e| panic!("{}", e.message));
        assert!(stale.is_dir());
        let second =
            Directory::create(&root.0, &binary).unwrap_or_else(|e| panic!("{}", e.message));
        assert!(stale.is_dir());
        drop(descendant);
        let third = Directory::create(&root.0, &binary).unwrap_or_else(|e| panic!("{}", e.message));
        assert!(!stale.exists());
        assert!(second.path.is_dir());
        second.clean().unwrap_or_else(|e| panic!("{}", e.message));
        let third_path = third.path.clone();
        third.clean().unwrap_or_else(|e| panic!("{}", e.message));
        assert!(!third_path.exists());
    }

    #[test]
    fn wrapper_child_status_and_modified_path_are_preserved() {
        let root =
            crate::campaign::prepare::process::Temporary::new(&std::env::temp_dir()).unwrap();
        let binary = std::env::current_exe().unwrap();
        let directory =
            Directory::create(&root.0, &binary).unwrap_or_else(|e| panic!("{}", e.message));
        for (command, expected) in [("/bin/true", 0), ("/bin/false", 1)] {
            let arguments = parse(&["--", command].map(OsString::from), None)
                .unwrap_or_else(|e| panic!("{}", e.message));
            assert_eq!(
                launch(
                    &arguments,
                    OsStr::new("/bin"),
                    Path::new("/original/go"),
                    Path::new(""),
                    &binary,
                    &directory
                )
                .unwrap_or_else(|e| panic!("{}", e.message)),
                expected
            );
        }
        let arguments = parse(&["--", "/bin/sh", "-c", "test \"$(command -v go)\" = \"$GOMAD_SHIM_RUN_DIR/go\" && test \"$GOMAD_SHIM_GO\" = /original/go && test -z \"$GOMAD_SHIM_INSTALLATION\"", "--installation=child"].map(OsString::from), None)
            .unwrap_or_else(|e| panic!("{}", e.message));
        assert_eq!(
            launch(
                &arguments,
                OsStr::new("/bin"),
                Path::new("/original/go"),
                Path::new(""),
                &binary,
                &directory
            )
            .unwrap_or_else(|e| panic!("{}", e.message)),
            0
        );
        directory
            .clean()
            .unwrap_or_else(|e| panic!("{}", e.message));
    }

    #[test]
    fn nested_wrapper_refuses_without_spawning_child() {
        use std::time::{Duration, Instant};
        let root =
            crate::campaign::prepare::process::Temporary::new(&std::env::temp_dir()).unwrap();
        let compiler_dir = root.0.join("compiler");
        fs::create_dir(&compiler_dir).unwrap();
        let compiler = compiler_dir.join("go");
        fs::write(
            &compiler,
            b"#!/bin/sh\nprintf invoked > \"$GOMAD_TEST_WRAPPER_ROOT/compiler-ran\"\nexit 98\n",
        )
        .unwrap();
        fs::set_permissions(&compiler, fs::Permissions::from_mode(0o700)).unwrap();
        let mut child = Command::new(std::env::current_exe().unwrap())
            .args([
                "--exact",
                "cli::wrapper::tests::wrapper_nested_process_probe",
                "--nocapture",
            ])
            .current_dir(&root.0)
            .env_clear()
            .env("PATH", &compiler_dir)
            .env("HOME", &root.0)
            .env("XDG_CACHE_HOME", root.0.join("cache"))
            .env("GOMAD_TEST_WRAPPER_ROOT", &root.0)
            .process_group(0)
            .stdout(Stdio::piped())
            .stderr(Stdio::piped())
            .spawn()
            .unwrap();
        let deadline = Instant::now() + Duration::from_secs(10);
        loop {
            if child.try_wait().unwrap().is_some() {
                break;
            }
            if Instant::now() >= deadline {
                let raw = i32::try_from(child.id()).unwrap();
                let pid = rustix::process::Pid::from_raw(raw).unwrap();
                let _ = rustix::process::kill_process_group(pid, rustix::process::Signal::KILL);
                let _ = child.kill();
                let _ = child.wait();
                panic!("nested wrapper regression exceeded its bound");
            }
            std::thread::sleep(Duration::from_millis(10));
        }
        let output = child.wait_with_output().unwrap();
        assert!(
            output.status.success(),
            "stdout={} stderr={}",
            String::from_utf8_lossy(&output.stdout),
            String::from_utf8_lossy(&output.stderr)
        );
        assert!(!root.0.join("child-ran").exists());
        assert!(!root.0.join("compiler-ran").exists());
    }

    #[test]
    fn wrapper_nested_process_probe() {
        let Some(root) = std::env::var_os("GOMAD_TEST_WRAPPER_ROOT") else {
            return;
        };
        if std::env::var_os("GOMAD_SHIM_RUN").is_some() {
            let arguments = [
                "--",
                "/bin/sh",
                "-c",
                "printf spawned > \"$GOMAD_TEST_WRAPPER_ROOT/child-ran\"",
            ]
            .map(OsString::from);
            match run(&arguments, None) {
                Err(error) => {
                    assert_eq!(error.status, 2);
                    assert!(
                        error.message.contains("nested wrapper"),
                        "{}",
                        error.message
                    );
                }
                Ok(status) => panic!("nested wrapper launched its child with status {status}"),
            }
            assert!(!Path::new(&root).join("child-ran").exists());
        } else {
            let arguments = [
                OsString::from("--"),
                std::env::current_exe().unwrap().into_os_string(),
                OsString::from("--exact"),
                OsString::from("cli::wrapper::tests::wrapper_nested_process_probe"),
                OsString::from("--nocapture"),
            ];
            let status = run(&arguments, None).unwrap_or_else(|error| panic!("{}", error.message));
            assert_eq!(status, 0);
        }
    }
}
