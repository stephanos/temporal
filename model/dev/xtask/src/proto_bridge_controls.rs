use super::*;
use crate::script_control_support::Scratch;
use std::{
    env, fs,
    io::Write,
    os::unix::fs::{PermissionsExt, symlink},
    path::PathBuf,
    process::Output,
};

#[test]
#[ignore = "isolated Temporal protobuf bridge child"]
fn bridge_child() -> io::Result<()> {
    let root = env::var_os("UMPIRE_TEMPORAL_PROTO_ROOT")
        .ok_or_else(|| io::Error::other("missing protobuf child root"))?;
    let input: Vec<String> =
        serde_json::from_str(&env::var("UMPIRE_TEMPORAL_PROTO_INPUT").map_err(io::Error::other)?)?;
    let input: Vec<_> = input.into_iter().map(OsString::from).collect();
    let result = run_with(Path::new(&root), Path::new("framework-proto"), &input, |command| {
        let arguments = command.get_args().map(ToOwned::to_owned).collect::<Vec<_>>();
        assert_eq!(arguments.first(), Some(&OsString::from("proto-go-spec")));
        xtask::proto::go_spec(arguments.into_iter().skip(1).collect()).map_err(io::Error::other)
    });
    let status = match result {
        Ok(status) => status,
        Err(error) => {
            writeln!(io::stderr(), "Temporal protobuf bridge: {error}")?;
            1
        }
    };
    std::process::exit(status)
}

struct Fixture {
    scratch: Scratch,
    model: PathBuf,
}
impl Fixture {
    fn new() -> io::Result<Self> {
        let scratch = Scratch::new()?;
        let repository = scratch.0.join("repo with spaces");
        let model = repository.join("model");
        for directory in
            [model.clone(), repository.join(".bin"), scratch.0.join("bin"), scratch.0.join("tmp")]
        {
            fs::create_dir_all(directory)?;
        }
        let plugin = repository.join(".bin/protoc-gen-go-v1.36.6");
        fs::write(&plugin, b"#!/bin/sh\nprintf 'protoc-gen-go v1.36.6\\n'\n")?;
        fs::set_permissions(plugin, fs::Permissions::from_mode(0o755))?;
        let mise = scratch.0.join("bin/mise");
        fs::write(
            &mise,
            br#"#!/bin/bash
set -euo pipefail
[[ "$MISE_AUTO_INSTALL:$MISE_EXEC_AUTO_INSTALL" = false:false ]]
[[ "$PWD" = "$EXPECTED_PROTO_ROOT" ]]
[[ "$1 $2 $3" = 'exec -- protoc' ]]
printf '%s\0' "$@" > "$PROTO_LOG"
for argument in "$@"; do
  case "$argument" in --go_out=*) output=${argument#*=} ;; esac
done
mkdir -p "$output/umpire"
printf 'generated\n' > "$output/umpire/generated.go"
[[ "${PROTO_FAIL:-0}" = 0 ]] || { printf 'protoc-failure diagnostic\n' >&2; exit 42; }
"#,
        )?;
        fs::set_permissions(mise, fs::Permissions::from_mode(0o755))?;
        Ok(Self { scratch, model })
    }

    fn destination(&self) -> PathBuf {
        self.model.join("temporal/hosts/go")
    }

    fn command(&self, input: &[&str]) -> io::Result<Command> {
        let mut command = Command::new(env::current_exe()?);
        command
            .args(["--exact", "proto::bridge_controls::bridge_child", "--nocapture", "--ignored"])
            .current_dir(&self.scratch.0)
            .env("UMPIRE_TEMPORAL_PROTO_ROOT", &self.model)
            .env("UMPIRE_TEMPORAL_PROTO_INPUT", serde_json::to_string(input)?)
            .env(
                "PATH",
                env::join_paths([self.scratch.0.join("bin"), "/usr/bin".into(), "/bin".into()])
                    .map_err(io::Error::other)?,
            )
            .env("EXPECTED_PROTO_ROOT", &self.model)
            .env("PROTO_LOG", self.scratch.0.join("protoc.log"))
            .env("TMPDIR", self.scratch.0.join("tmp"))
            .env("CARGO_HOME", self.scratch.0.join("cargo home"))
            .env("CARGO_TARGET_DIR", self.scratch.0.join("target"))
            .env("MISE_AUTO_INSTALL", "true")
            .env("MISE_EXEC_AUTO_INSTALL", "true")
            .env_remove("PROTO_FAIL")
            .env_remove("BASH_ENV");
        Ok(command)
    }

    fn execute(&self, command: &Command, expected: i32) -> io::Result<Output> {
        let output = xtask::gate::scheduler::execute(command, false, false)?;
        assert_eq!(output.status.code(), Some(expected), "{output:?}");
        assert!(fs::read_dir(self.scratch.0.join("tmp"))?.next().is_none());
        Ok(output)
    }
}

#[test]
fn temporal_spec_composes_with_native_generation_freshness_and_refusals() -> io::Result<()> {
    let fixture = Fixture::new()?;
    fixture.execute(&fixture.command(&["temporal"])?, 0)?;
    let arguments = fs::read(fixture.scratch.0.join("protoc.log"))?;
    let arguments: Vec<_> =
        arguments.split(|byte| *byte == 0).filter(|arg| !arg.is_empty()).collect();
    for expected in [
        b"--proto_path=temporal/proto".as_slice(),
        b"--proto_path=umpire/protocol/proto",
        b"temporal/proto/umpire/temporal/v1/temporal.proto",
        b"temporal/proto/umpire/temporal/v1/probe.proto",
        b"temporal/proto/umpire/temporal/v1/activity.proto",
    ] {
        assert!(arguments.contains(&expected), "missing {expected:?}");
    }
    assert!(!arguments.iter().any(|arg| arg.starts_with(b"--go-grpc")));
    let destination = fixture.destination();
    fs::write(destination.join("host.go"), b"handwritten\n")?;
    fixture.execute(&fixture.command(&["temporal", "--check"])?, 0)?;
    fs::write(destination.join("umpire/stale.go"), b"stale\n")?;
    let stale = fixture.execute(&fixture.command(&["temporal", "--check"])?, 1)?;
    assert!(String::from_utf8_lossy(&stale.stderr).contains("stale.go"));
    assert_eq!(fs::read(destination.join("umpire/stale.go"))?, b"stale\n");
    fs::remove_file(destination.join("umpire/stale.go"))?;
    symlink("/dev/null", destination.join("umpire/link.go"))?;
    let linked = fixture.execute(&fixture.command(&["temporal", "--check"])?, 1)?;
    assert!(String::from_utf8_lossy(&linked.stderr).contains("link.go"));
    assert!(fs::symlink_metadata(destination.join("umpire/link.go"))?.file_type().is_symlink());
    fs::remove_file(destination.join("umpire/link.go"))?;
    fixture.execute(&fixture.command(&["temporal", "--check"])?, 0)?;
    assert_eq!(fs::read(destination.join("host.go"))?, b"handwritten\n");

    fs::remove_file(fixture.scratch.0.join("protoc.log"))?;
    for input in [
        vec![],
        vec!["core"],
        vec!["kettle"],
        vec!["temporal", "--bogus"],
        vec!["temporal", "--check", "surplus"],
    ] {
        let output = fixture.execute(&fixture.command(&input)?, 1)?;
        assert!(String::from_utf8_lossy(&output.stderr).contains("usage:"));
        assert!(!fixture.scratch.0.join("protoc.log").exists());
    }
    let mut failed = fixture.command(&["temporal", "--check"])?;
    failed.env("PROTO_FAIL", "1");
    let failed = fixture.execute(&failed, 42)?;
    assert!(String::from_utf8_lossy(&failed.stderr).contains("protoc-failure diagnostic"));
    assert_eq!(fs::read(destination.join("host.go"))?, b"handwritten\n");
    assert_eq!(fs::read(destination.join("umpire/generated.go"))?, b"generated\n");
    Ok(())
}
