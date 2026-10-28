//! Repository-owned historical name policy controls.
#![forbid(unsafe_code)]
#![allow(clippy::unwrap_used, clippy::expect_used)]
use std::{path::Path, process::Command};

#[test]
#[ignore = "isolated native policy child"]
fn scan_child() {
    let root = std::env::var_os("UMPIRE_RETIRED_POLICY_ROOT").expect("missing policy child root");
    std::process::exit(crate::retired_names::run(Path::new(&root)));
}

fn execute(command: &Command) -> std::process::Output {
    xtask::gate::scheduler::execute(command, false, false).unwrap()
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
    fn write(&self, path: impl AsRef<Path>, bytes: impl AsRef<[u8]>) {
        let path = self.0.join(path);
        std::fs::create_dir_all(path.parent().unwrap()).unwrap();
        std::fs::write(path, bytes).unwrap();
    }
    fn scan_command(&self) -> Command {
        let mut command = Command::new(std::env::current_exe().unwrap());
        command
            .args([
                "--exact",
                "retired_name_policy_controls::scan_child",
                "--nocapture",
                "--ignored",
            ])
            .env_remove("BASH_ENV")
            .env("UMPIRE_RETIRED_POLICY_ROOT", &self.0);
        command
    }
    fn scan(&self) -> std::process::Output {
        execute(&self.scan_command())
    }
}
impl Drop for Scratch {
    fn drop(&mut self) {
        if let Err(error) = std::fs::remove_dir_all(&self.0) {
            use std::io::Write;
            let _ = writeln!(std::io::stderr(), "retired policy scratch cleanup failed: {error}");
        }
    }
}

#[test]
fn active_spellings_and_historical_exclusions() {
    let scratch = Scratch::new();
    assert!(execute(Command::new("git").args(["init", "-q"]).arg(&scratch.0)).status.success());
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
        execute(Command::new("git").arg("-C").arg(&scratch.0).args(["add", "-A"])).status.success()
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
    for (body, expected) in [
        ("exit 7", 7),
        ("printf malformed", 2),
        ("printf '/escape\\000'", 2),
        ("printf '../escape\\000'", 2),
    ] {
        let bin = Scratch::new();
        bin.write("git", format!("#!/bin/sh\n{body}\n"));
        std::fs::set_permissions(bin.0.join("git"), std::fs::Permissions::from_mode(0o755))
            .unwrap();
        let path = std::env::join_paths(
            std::iter::once(bin.0.clone())
                .chain(std::env::split_paths(&std::env::var_os("PATH").unwrap())),
        )
        .unwrap();
        let output = execute(scratch.scan_command().env("PATH", &path));
        assert_eq!(output.status.code(), Some(expected), "{output:?}");
        assert!(!String::from_utf8_lossy(&output.stdout).contains("clean"));
    }
}

#[cfg(unix)]
#[test]
fn raw_path_bytes_and_io_failures_reach_native_policy() {
    use std::{
        ffi::OsString,
        os::unix::{
            ffi::OsStringExt,
            fs::{PermissionsExt, symlink},
        },
    };
    let scratch = Scratch::new();
    assert!(execute(Command::new("git").args(["init", "-q"]).arg(&scratch.0)).status.success());
    let raw = OsString::from_vec(b"raw-\xff.rs".to_vec());
    let contents = [b"umpire".as_slice(), b"-wasm \xfe\n"].concat();
    scratch.write(&raw, &contents);
    let output = scratch.scan();
    assert_eq!(output.status.code(), Some(1), "{output:?}");
    let record = [raw.into_vec(), b":1:".to_vec(), contents].concat();
    assert!(output.stdout.windows(record.len()).any(|bytes| bytes == record));
    assert!(!String::from_utf8_lossy(&output.stdout).contains("clean"));

    let bin = Scratch::new();
    bin.write("git", "#!/bin/sh\nprintf 'loop/active.rs\\000'\n");
    std::fs::set_permissions(bin.0.join("git"), std::fs::Permissions::from_mode(0o755)).unwrap();
    symlink("loop", scratch.0.join("loop")).unwrap();
    let path = std::env::join_paths(
        std::iter::once(bin.0.clone())
            .chain(std::env::split_paths(&std::env::var_os("PATH").unwrap())),
    )
    .unwrap();
    let output = execute(scratch.scan_command().env("PATH", path));
    assert_eq!(output.status.code(), Some(2), "{output:?}");
    assert!(!String::from_utf8_lossy(&output.stdout).contains("clean"));
    assert!(String::from_utf8_lossy(&output.stderr).contains("retired-name scan failed"));
}
