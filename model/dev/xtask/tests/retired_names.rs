//! Retired-name stream and shell policy controls.
#![allow(clippy::unwrap_used, clippy::expect_used)]
use std::{
    io::Write,
    process::{Command, Stdio},
};

fn filter(pattern: &str, input: &[u8]) -> std::process::Output {
    let mut child = Command::new(env!("CARGO_BIN_EXE_xtask"))
        .args(["retired-names", pattern])
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .unwrap();
    child.stdin.take().unwrap().write_all(input).unwrap();
    child.wait_with_output().unwrap()
}

#[test]
fn raw_records_exempt_only_author_facade_tokens() {
    let token = ["author/umpire/", "tests"].concat();
    let pattern = ["umpire/", "tests"].concat();
    let allowed = format!("file\0{}\0'{token}/consumer'\n", 17);
    assert_eq!(filter(&pattern, allowed.as_bytes()).status.code(), Some(0));
    for text in [
        format!("not_{token}/x"),
        format!("{token}/x /repo/{pattern}/x"),
        format!("{token}_suffix"),
    ] {
        let input = format!("path:with\nnewline\0{}\0{text}\n", 23);
        let result = filter(&pattern, input.as_bytes());
        assert_eq!(result.status.code(), Some(1));
        assert_eq!(result.stdout, format!("path:with\nnewline:23:{text}\n").as_bytes());
    }
    let input = b"nonutf8\xff\0\x30\x30\x37\0forbidden\xff";
    let result = filter("forbidden", input);
    assert_eq!(result.status.code(), Some(1));
    assert_eq!(result.stdout, b"nonutf8\xff:007:forbidden\xff\n");
    assert_eq!(filter("anything", b"").status.code(), Some(0));
}

#[test]
fn malformed_records_fail_closed() {
    for input in [b"path".as_slice(), b"path\0", b"path\0\0line", b"path\0nope\0line"] {
        let result = filter("anything", input);
        assert_eq!(result.status.code(), Some(2), "{result:?}");
        assert_ne!(result.stderr, b"");
    }
}

#[test]
fn posix_ere_recheck_and_grep_failure_propagate() {
    let token = ["author/umpire/", "tests/x"].concat();
    for (pattern, expected) in [("[[:digit:]]+", 1), ("[[:upper:]]+", 0), ("[", 2)] {
        let result = filter(pattern, format!("file\0{}\0{token} 123\n", 1).as_bytes());
        assert_eq!(result.status.code(), Some(expected), "{result:?}");
    }
}

struct Scratch(std::path::PathBuf);
impl Scratch {
    fn new() -> Self {
        static NEXT: std::sync::atomic::AtomicU64 = std::sync::atomic::AtomicU64::new(0);
        let path = std::env::temp_dir().join(format!(
            "retired-names-{}-{}",
            std::process::id(),
            NEXT.fetch_add(1, std::sync::atomic::Ordering::Relaxed)
        ));
        std::fs::create_dir(&path).unwrap();
        Self(path)
    }
    fn write(&self, path: &str, bytes: impl AsRef<[u8]>) {
        let path = self.0.join(path);
        std::fs::create_dir_all(path.parent().unwrap()).unwrap();
        std::fs::write(path, bytes).unwrap();
    }
    fn scan(&self) -> std::process::Output {
        Command::new("bash")
            .arg(self.0.join("model/scripts/retired-names.sh"))
            .env("UMPIRE_XTASK", env!("CARGO_BIN_EXE_xtask"))
            .output()
            .unwrap()
    }
}
impl Drop for Scratch {
    fn drop(&mut self) {
        std::fs::remove_dir_all(&self.0).unwrap();
    }
}

#[test]
fn active_spellings_and_historical_exclusions() {
    let scratch = Scratch::new();
    scratch.write(
        "model/scripts/retired-names.sh",
        include_bytes!("../../../scripts/retired-names.sh"),
    );
    assert!(Command::new("git").args(["init", "-q"]).arg(&scratch.0).status().unwrap().success());
    let mut legacy = Vec::new();
    for separator in ["-", "_", "/"] {
        for name in ["wasm", "tests"] {
            legacy.push(format!("umpire{separator}{name}"));
        }
    }
    for prefix in ["/repo/model/umpire/", "/repo/umpire/"] {
        for name in ["wasm", "tests"] {
            legacy.push(format!("{prefix}{name}/x.rs"));
        }
    }
    for separator in ["-", "_"] {
        for name in ["wasm", "tests"] {
            legacy.push(format!("embedded_umpire{separator}{name}_suffix"));
        }
    }
    let mut layout = Vec::new();
    for separator in ["-", "_"] {
        for name in ["lift", "annotations"] {
            layout.push(format!("umpire{separator}{name}"));
        }
    }
    for prefix in ["", "model/", "../../", "../model/", "/repo/model/"] {
        for name in [
            "dsl",
            "suite",
            "checker",
            "lift",
            "runtime",
            "proto",
            "testing",
            "fixtures",
            "ir-derive",
            "catalog-build",
            "annotations",
        ] {
            layout.push(format!("{prefix}umpire/{name}"));
        }
    }
    for suffix in ["Cargo.toml", "README.md", "build.rs", "src/lib.rs", "tests/control.rs"] {
        layout.push(format!("model/umpire/{}/{suffix}", "annotations"));
    }
    for spelling in legacy.iter().chain(&layout) {
        scratch.write("model/source.rs", format!("{spelling}\n"));
        let result = scratch.scan();
        assert_eq!(result.status.code(), Some(1), "{spelling}: {result:?}");
        assert!(String::from_utf8_lossy(&result.stdout).contains("model/source.rs:1:"));
    }
    scratch.write("model/source.rs", ["umpire-runtime umpire_fixtures umpire/author/dsl umpire/protocol/proto umpire/annotations/v1 file_umpire_annotations_v1_annotations_proto __umpire_lifted ", "umpire/author/umpire/", "tests/consumer /repo/model/umpire/author/umpire/", "tests/consumer /repo/umpire/author/umpire/", "tests/consumer author/umpire/", "tests/consumer umpire/protocolv2 umpire/protobuf-lean/v1\n"].concat());
    for path in [
        "model/archived/source.rs",
        ".flow/tasks/history.md",
        ".plans/umpire-migration-audit.json",
        ".plans/umpire-migration-manifest.json",
        "model/design/rust-verdict-measurements.md",
    ] {
        scratch.write(path, legacy.iter().chain(&layout).cloned().collect::<Vec<_>>().join("\n"));
    }
    let history = [
        ".plans/archive/history.md",
        ".turbo/plans/history.md",
        "model/temporal/feature/activity/compatibility/original/rows.snap",
    ];
    for path in history {
        scratch.write(path, layout.join("\n"));
    }
    assert!(
        Command::new("git")
            .arg("-C")
            .arg(&scratch.0)
            .args(["add", "-A"])
            .status()
            .unwrap()
            .success()
    );
    assert_eq!(scratch.scan().status.code(), Some(0));
    for path in history {
        for spelling in &legacy {
            scratch.write(path, spelling);
            assert_eq!(scratch.scan().status.code(), Some(1), "{path}: {spelling}");
        }
        scratch.write(path, layout.join("\n"));
    }
    assert_eq!(scratch.scan().status.code(), Some(0));
}

#[cfg(unix)]
#[test]
fn tool_failures_never_report_clean() {
    use std::os::unix::fs::PermissionsExt;
    let scratch = Scratch::new();
    scratch.write(
        "model/scripts/retired-names.sh",
        include_bytes!("../../../scripts/retired-names.sh"),
    );
    for (tool, body, expected) in
        [("git", "exit 7", 7), ("git", "printf malformed", 2), ("grep", "exit 9", 9)]
    {
        let bin = Scratch::new();
        bin.write(tool, format!("#!/bin/sh\n{body}\n"));
        std::fs::set_permissions(bin.0.join(tool), std::fs::Permissions::from_mode(0o755)).unwrap();
        let path = std::env::join_paths(
            std::iter::once(bin.0.clone())
                .chain(std::env::split_paths(&std::env::var_os("PATH").unwrap())),
        )
        .unwrap();
        let output = if tool == "git" {
            Command::new("bash")
                .arg(scratch.0.join("model/scripts/retired-names.sh"))
                .env("UMPIRE_XTASK", env!("CARGO_BIN_EXE_xtask"))
                .env("PATH", &path)
                .output()
                .unwrap()
        } else {
            let mut child = Command::new(env!("CARGO_BIN_EXE_xtask"))
                .args(["retired-names", "anything"])
                .env("PATH", &path)
                .stdin(Stdio::piped())
                .stdout(Stdio::piped())
                .stderr(Stdio::piped())
                .spawn()
                .unwrap();
            child
                .stdin
                .take()
                .unwrap()
                .write_all(["file\0", "1\0author/umpire/", "tests/x\n"].concat().as_bytes())
                .unwrap();
            child.wait_with_output().unwrap()
        };
        assert_eq!(output.status.code(), Some(expected), "{output:?}");
        assert!(!String::from_utf8_lossy(&output.stdout).contains("clean"));
    }
}

#[test]
fn io_and_missing_grep_errors_fail_closed() {
    struct Refuse;
    impl Write for Refuse {
        fn write(&mut self, _: &[u8]) -> std::io::Result<usize> {
            Err(std::io::Error::other("control"))
        }
        fn flush(&mut self) -> std::io::Result<()> {
            Ok(())
        }
    }
    assert_eq!(
        xtask::retired_names::run(std::ffi::OsStr::new("x"), b"file\0\x31\0x\n".as_slice(), Refuse),
        2
    );
    let empty = Scratch::new();
    let mut child = Command::new(env!("CARGO_BIN_EXE_xtask"))
        .args(["retired-names", "x"])
        .env("PATH", &empty.0)
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .unwrap();
    child
        .stdin
        .take()
        .unwrap()
        .write_all(["file\0", "1\0author/umpire/", "tests/x\n"].concat().as_bytes())
        .unwrap();
    assert_eq!(child.wait_with_output().unwrap().status.code(), Some(2));
}

#[cfg(unix)]
#[test]
fn shell_preserves_filter_failure_when_git_grep_gets_sigpipe() {
    use std::os::unix::fs::PermissionsExt;
    let scratch = Scratch::new();
    scratch.write(
        "model/scripts/retired-names.sh",
        include_bytes!("../../../scripts/retired-names.sh"),
    );
    assert!(Command::new("git").args(["init", "-q"]).arg(&scratch.0).status().unwrap().success());
    scratch.write("model/source.rs", ["author/umpire/", "tests/x\n"].concat().repeat(50_000));
    for (git_failure, code, expected) in [(None, 2, 2), (None, 9, 9), (Some(7), 9, 7)] {
        let bin = Scratch::new();
        bin.write("grep", format!("#!/bin/sh\nexit {code}\n"));
        std::fs::set_permissions(bin.0.join("grep"), std::fs::Permissions::from_mode(0o755))
            .unwrap();
        if let Some(git_failure) = git_failure {
            bin.write(
                "git",
                format!(
                    "#!/bin/sh\nprintf 'file\\0001\\000{}\\n'\nexit {git_failure}\n",
                    ["author/umpire/", "tests/x"].concat()
                ),
            );
            std::fs::set_permissions(bin.0.join("git"), std::fs::Permissions::from_mode(0o755))
                .unwrap();
        }
        let path = std::env::join_paths(
            std::iter::once(bin.0.clone())
                .chain(std::env::split_paths(&std::env::var_os("PATH").unwrap())),
        )
        .unwrap();
        let output = Command::new("bash")
            .arg(scratch.0.join("model/scripts/retired-names.sh"))
            .env("BASH_ENV", "/dev/null")
            .env("UMPIRE_XTASK", env!("CARGO_BIN_EXE_xtask"))
            .env("PATH", path)
            .output()
            .unwrap();
        assert_eq!(output.status.code(), Some(expected), "{output:?}");
        assert!(!String::from_utf8_lossy(&output.stdout).contains("clean"));
    }
}
