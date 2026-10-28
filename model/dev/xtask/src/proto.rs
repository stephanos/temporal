//! Temporal protobuf selector for the independent native generator.
use std::{ffi::OsString, io, os::unix::process::ExitStatusExt, path::Path, process::Command};

fn arguments(root: &Path, arguments: &[OsString]) -> io::Result<Vec<OsString>> {
    if arguments.is_empty()
        || arguments.len() > 2
        || arguments[0] != "temporal"
        || arguments.get(1).is_some_and(|argument| argument != "--check")
    {
        return Err(io::Error::other("usage: cargo xtask proto-go temporal [--check]"));
    }
    let repository =
        root.parent().ok_or_else(|| io::Error::other("Model root has no repository parent"))?;
    let mut result: Vec<OsString> = vec![
        "proto-go-spec".into(),
        "--root".into(),
        root.as_os_str().to_owned(),
        "--plugin-root".into(),
        repository.as_os_str().to_owned(),
        "--destination".into(),
        "temporal/hosts/go".into(),
        "--include".into(),
        "temporal/proto".into(),
        "--include".into(),
        "umpire/protocol/proto".into(),
    ];
    for name in ["temporal", "probe", "activity"] {
        result.extend([
            "--input".into(),
            format!("temporal/proto/umpire/temporal/v1/{name}.proto").into(),
        ]);
    }
    result.extend(["--check-subtree".into(), "umpire".into()]);
    if arguments.len() == 2 {
        result.push("--check".into());
    }
    Ok(result)
}

/// Execute the admitted framework generator without changing the caller's process context.
pub(crate) fn run(root: &Path, tool: &Path, input: &[OsString]) -> io::Result<i32> {
    run_with(root, tool, input, |command| {
        let status = xtask::gate::scheduler::execute(command, true, true)?.status;
        Ok(status.code().unwrap_or_else(|| 128 + status.signal().unwrap_or(0)))
    })
}

fn run_with(
    root: &Path,
    tool: &Path,
    input: &[OsString],
    mut execute: impl FnMut(&mut Command) -> io::Result<i32>,
) -> io::Result<i32> {
    let mut command = Command::new(tool);
    command.args(arguments(root, input)?);
    execute(&mut command)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn raw_repository_and_tool_paths_remain_lossless() -> io::Result<()> {
        use std::os::unix::ffi::OsStringExt;
        let root = std::path::PathBuf::from(OsString::from_vec(b"/repo-\xff/model".to_vec()));
        let tool = std::path::PathBuf::from(OsString::from_vec(b"/tool-\xfe".to_vec()));
        assert_eq!(
            run_with(&root, &tool, &["temporal".into()], |command| {
                assert_eq!(command.get_program(), tool.as_os_str());
                let arguments: Vec<_> = command.get_args().collect();
                assert_eq!(arguments[2], root.as_os_str());
                assert_eq!(
                    arguments[4],
                    root.parent().ok_or_else(|| io::Error::other("missing parent"))?.as_os_str()
                );
                Ok(37)
            })?,
            37
        );
        Ok(())
    }

    #[test]
    fn only_temporal_and_optional_check_are_admitted_before_execution() {
        for input in [
            vec![],
            vec!["core"],
            vec!["kettle"],
            vec!["temporal", "--bogus"],
            vec!["temporal", "--check", "surplus"],
        ] {
            let input: Vec<_> = input.into_iter().map(OsString::from).collect();
            assert!(
                run_with(Path::new("/repo/model"), Path::new("tool"), &input, |_| panic!(
                    "invalid selector executed"
                ))
                .is_err()
            );
        }
    }

    #[test]
    fn temporal_inputs_subtree_and_repository_plugin_preserve_raw_paths_and_status()
    -> io::Result<()> {
        let root = Path::new("/repo with spaces/model");
        for check in [false, true] {
            let mut input = vec!["temporal".into()];
            if check {
                input.push("--check".into());
            }
            assert_eq!(
                run_with(root, Path::new("/native tool"), &input, |command| {
                    assert_eq!(command.get_program(), "/native tool");
                    assert_eq!(command.get_current_dir(), None);
                    let args: Vec<_> =
                        command.get_args().map(std::ffi::OsStr::to_os_string).collect();
                    let expected: Vec<OsString> = [
                        "proto-go-spec",
                        "--root",
                        "/repo with spaces/model",
                        "--plugin-root",
                        "/repo with spaces",
                        "--destination",
                        "temporal/hosts/go",
                        "--include",
                        "temporal/proto",
                        "--include",
                        "umpire/protocol/proto",
                        "--input",
                        "temporal/proto/umpire/temporal/v1/temporal.proto",
                        "--input",
                        "temporal/proto/umpire/temporal/v1/probe.proto",
                        "--input",
                        "temporal/proto/umpire/temporal/v1/activity.proto",
                        "--check-subtree",
                        "umpire",
                    ]
                    .into_iter()
                    .map(OsString::from)
                    .chain(check.then(|| OsString::from("--check")))
                    .collect();
                    assert_eq!(args, expected);
                    assert!(!args.iter().any(|argument| argument == "--grpc"));
                    Ok(42)
                })?,
                42
            );
        }
        Ok(())
    }
}
