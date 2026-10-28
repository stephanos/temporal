//! Native protobuf behavior controls with an in-process process-boundary fixture.
use super::*;
use std::os::unix::{fs::symlink, process::ExitStatusExt};

fn require(condition: bool, message: &str) -> Result<()> {
    if condition {
        Ok(())
    } else {
        Err(Error::Invalid(format!("protobuf control failed: {message}")))
    }
}
fn executable(path: &Path) -> Result<()> {
    fs::write(path, b"fixture executable; never executed")?;
    fs::set_permissions(path, fs::Permissions::from_mode(0o755))?;
    Ok(())
}
#[derive(Default)]
struct Runner {
    calls: Vec<Vec<OsString>>,
    failure: i32,
    wrong_version: bool,
    renamed_version: bool,
    version_status: i32,
    wrong_grpc: bool,
    last_output: Option<PathBuf>,
}
impl Runner {
    fn execute(&mut self, command: &mut Command) -> Result<Output> {
        let args: Vec<_> = command.get_args().map(ToOwned::to_owned).collect();
        self.calls.push(args.clone());
        for key in ["MISE_AUTO_INSTALL", "MISE_EXEC_AUTO_INSTALL"] {
            require(
                command
                    .get_envs()
                    .any(|(k, value)| k == key && value == Some(std::ffi::OsStr::new("false"))),
                "mise installation must stay disabled",
            )?;
        }
        for key in ["CARGO_HOME", "CARGO_TARGET_DIR"] {
            require(
                command.get_envs().any(|(k, value)| {
                    k == key && value.is_some_and(|p| Path::new(p).is_absolute())
                }),
                "caller paths must be absolute",
            )?;
        }
        let mut stdout = Vec::new();
        let mut status = 0;
        if command.get_program() == "mise" {
            require(args.starts_with(&["exec".into(), "--".into()]), "mise execution prefix")?;
            status = self.failure;
            if status == 0 && args.get(2).is_some_and(|arg| arg == "go") {
                require(args.get(3).is_some_and(|arg| arg == "install"), "Go plugin installation")?;
                let name = match args.get(4).and_then(|arg| arg.to_str()) {
                    Some("google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.6") => "protoc-gen-go",
                    Some("google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0") => {
                        "protoc-gen-go-grpc"
                    }
                    _ => return Err(Error::Invalid("unexpected plugin pin".into())),
                };
                let gobin = command
                    .get_envs()
                    .find(|(key, _)| key == "GOBIN")
                    .and_then(|(_, value)| value)
                    .ok_or_else(|| Error::Invalid("missing staging directory".into()))?;
                executable(&Path::new(gobin).join(name))?;
            } else if args.get(2).is_some_and(|arg| arg == "protoc") {
                let output = args
                    .iter()
                    .find_map(|arg| {
                        arg.to_str()
                            .and_then(|arg| arg.strip_prefix("--go_out="))
                            .map(PathBuf::from)
                    })
                    .ok_or_else(|| Error::Invalid("missing protobuf output".into()))?;
                self.last_output = Some(output.clone());
                // Materialize partial output even on failure to exercise cleanup.
                fs::create_dir_all(output.join("umpire"))?;
                fs::write(output.join("umpire/generated.go"), b"generated\n")?;
            }
        } else {
            require(args == [OsString::from("--version")], "generator receives version flag")?;
            let grpc = Path::new(command.get_program())
                .file_name()
                .is_some_and(|name| name.to_string_lossy().contains("grpc"));
            status = self.version_status;
            stdout = if self.wrong_version || (grpc && self.wrong_grpc) {
                b"wrong\n".to_vec()
            } else if grpc {
                b"protoc-gen-go-grpc 1.3.0\n".to_vec()
            } else if self.renamed_version {
                b"protoc-gen-go-v1.36.6 v1.36.6\n".to_vec()
            } else {
                b"protoc-gen-go v1.36.6\n".to_vec()
            };
        }
        Ok(Output {
            status: std::process::ExitStatus::from_raw(status << 8),
            stdout,
            stderr: Vec::new(),
        })
    }
}

/// Exercise real native path, publication, comparison and cleanup logic without installing tools.
pub fn run() -> Result<i32> {
    let scratch = tempfile::tempdir()?;
    let root = scratch.path().canonicalize()?;
    let context =
        Context { root: root.clone(), home: root.join("cargo-home"), target: root.join("target") };
    let mut runner = Runner::default();
    require(
        tools_with(&context, &mut |command| runner.execute(command))? == 0,
        "initial plugin provision",
    )?;
    runner.renamed_version = true;
    runner.calls.clear();
    require(
        tools_with(&context, &mut |command| runner.execute(command))? == 0,
        "repeated plugin provision",
    )?;
    require(
        runner.calls.iter().all(|args| args == &[OsString::from("--version")]),
        "existing pins never reinstall",
    )?;
    let lock = root.join(".bin/proto-tools.lock");
    fs::create_dir(&lock)?;
    require(
        tools_with(&context, &mut |command| runner.execute(command)).is_err(),
        "concurrent lock refusal",
    )?;
    fs::remove_dir(&lock)?;
    let go = root.join(".bin").join(GO);
    fs::remove_file(&go)?;
    symlink("/dev/null", &go)?;
    require(
        tools_with(&context, &mut |command| runner.execute(command)).is_err(),
        "generator symlink refusal",
    )?;
    require(!lock.exists(), "refusal cleans owned lock")?;
    fs::remove_file(&go)?;
    runner.failure = 42;
    require(
        tools_with(&context, &mut |command| runner.execute(command))? == 42,
        "install failure status",
    )?;
    require(!lock.exists() && !go.exists(), "failed install cleanup")?;
    runner.failure = 0;
    require(tools_with(&context, &mut |command| runner.execute(command))? == 0, "retry provision")?;

    for kettle in [false, true] {
        let spec = framework_spec(&root, kettle);
        require(
            generate_with(&context, &spec, false, &mut |command| runner.execute(command))? == 0,
            "generation",
        )?;
        let args = runner
            .calls
            .last()
            .ok_or_else(|| Error::Invalid("missing protoc invocation".into()))?;
        require(
            args.contains(&OsString::from(if kettle {
                "reference/kettle/proto/kettle/v1/probe.proto"
            } else {
                "protocol/proto/umpire/run/v1/run.proto"
            })),
            "selector input closure",
        )?;
        require(
            args.iter().any(|arg| arg.to_string_lossy().starts_with("--go-grpc_out=")) == kettle,
            "gRPC only for selected module",
        )?;
        let destination = root.join(&spec.destination);
        require(
            generate_with(&context, &spec, true, &mut |command| runner.execute(command))? == 0,
            "fresh generated tree",
        )?;
        require(
            runner.last_output.as_ref().is_some_and(|path| !path.exists()),
            "private check cleanup",
        )?;
        fs::write(destination.join("stale.go"), "stale")?;
        require(
            generate_with(&context, &spec, true, &mut |command| runner.execute(command))? == 1,
            "stale extra file",
        )?;
        fs::remove_file(destination.join("stale.go"))?;
        fs::write(destination.join("umpire/generated.go"), "changed")?;
        require(
            generate_with(&context, &spec, true, &mut |command| runner.execute(command))? == 1,
            "changed generated bytes",
        )?;
        fs::write(destination.join("umpire/generated.go"), "generated\n")?;
        symlink("/dev/null", destination.join("link.go"))?;
        require(
            generate_with(&context, &spec, true, &mut |command| runner.execute(command)).is_err(),
            "generated symlink refusal",
        )?;
        fs::remove_file(destination.join("link.go"))?;
        runner.failure = 42;
        require(
            generate_with(&context, &spec, true, &mut |command| runner.execute(command))? == 42,
            "protoc failure status",
        )?;
        require(
            runner.last_output.as_ref().is_some_and(|path| !path.exists()),
            "failed protoc cleanup",
        )?;
        runner.failure = 0;
        runner.wrong_version = true;
        require(
            generate_with(&context, &spec, true, &mut |command| runner.execute(command)).is_err(),
            "wrong generator pin",
        )?;
        runner.wrong_version = false;
    }
    let mut spec = framework_spec(&root, false);
    spec.destination = "consumer".into();
    spec.check_subtree = Some("umpire".into());
    fs::create_dir_all(root.join("consumer"))?;
    fs::write(root.join("consumer/host.go"), "handwritten host")?;
    require(
        generate_with(&context, &spec, false, &mut |command| runner.execute(command))? == 0,
        "consumer generation",
    )?;
    require(
        generate_with(&context, &spec, true, &mut |command| runner.execute(command))? == 0,
        "consumer ignores handwritten sources outside subtree",
    )?;
    fs::write(root.join("consumer/umpire/stale.go"), "stale")?;
    require(
        generate_with(&context, &spec, true, &mut |command| runner.execute(command))? == 1,
        "consumer subtree extra output",
    )?;
    fs::remove_file(root.join("consumer/umpire/stale.go"))?;
    fs::rename(root.join("consumer/umpire"), root.join("consumer/saved"))?;
    symlink(root.join("consumer/saved"), root.join("consumer/umpire"))?;
    require(
        generate_with(&context, &spec, true, &mut |command| runner.execute(command)).is_err(),
        "consumer subtree symlink refusal",
    )?;
    require(
        runner.last_output.as_ref().is_some_and(|path| !path.exists()),
        "comparison refusal cleanup",
    )?;

    let core = framework_spec(&root, false);
    let kettle = framework_spec(&root, true);
    let grpc = root.join(".bin").join(GRPC);
    fs::rename(&grpc, root.join("saved-grpc"))?;
    require(
        generate_with(&context, &core, true, &mut |command| runner.execute(command))? == 0,
        "core requires no gRPC plugin",
    )?;
    require(
        generate_with(&context, &kettle, true, &mut |command| runner.execute(command)).is_err(),
        "missing gRPC plugin refusal",
    )?;
    fs::rename(root.join("saved-grpc"), &grpc)?;
    runner.wrong_grpc = true;
    require(
        generate_with(&context, &kettle, true, &mut |command| runner.execute(command)).is_err(),
        "wrong gRPC plugin pin",
    )?;
    runner.wrong_grpc = false;
    fs::rename(&go, root.join("saved-go"))?;
    require(
        generate_with(&context, &core, true, &mut |command| runner.execute(command)).is_err(),
        "missing Go plugin refusal",
    )?;
    fs::rename(root.join("saved-go"), &go)?;
    runner.version_status = 37;
    require(
        generate_with(&context, &core, true, &mut |command| runner.execute(command))? == 37,
        "generator failure status",
    )?;
    runner.version_status = 0;
    for args in
        [vec![], vec!["unknown"], vec!["core", "--bogus"], vec!["core", "--check", "surplus"]]
    {
        require(
            super::go(args.into_iter().map(OsString::from).collect()).is_err(),
            "selector argument refusal",
        )?;
    }
    fs::rename(root.join(".bin"), root.join("plugins"))?;
    symlink(root.join("plugins"), root.join(".bin"))?;
    require(
        tools_with(&context, &mut |command| runner.execute(command)).is_err(),
        "plugin directory symlink refusal",
    )?;
    writeln!(io::stdout(), "native protobuf controls passed")?;
    Ok(0)
}

#[cfg(test)]
mod tests {
    #[test]
    fn native_protobuf_behavior_controls() -> crate::Result<()> {
        assert_eq!(super::run()?, 0);
        Ok(())
    }
}
