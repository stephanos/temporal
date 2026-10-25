//! Retired-name stream and shell policy controls.
#![forbid(unsafe_code)]
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
    let empty = tempfile::tempdir().unwrap();
    let mut child = Command::new(env!("CARGO_BIN_EXE_xtask"))
        .args(["retired-names", "x"])
        .env("PATH", empty.path())
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
