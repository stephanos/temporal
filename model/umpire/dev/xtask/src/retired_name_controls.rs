use super::*;
use std::{
    io::Cursor,
    os::unix::{ffi::OsStrExt, fs::symlink},
};

#[test]
fn raw_records_keep_non_utf8_paths_lines_and_numeric_spelling() {
    let mut output = Vec::new();
    assert_eq!(
        run(
            OsStr::new("forbidden"),
            Cursor::new(b"path\xff\0\x30\x30\x37\0forbidden\xff"),
            &mut output
        ),
        1
    );
    assert_eq!(output, b"path\xff:007:forbidden\xff\n");
}

#[test]
fn rust_patterns_and_malformed_records_fail_closed() {
    for input in [b"path".as_slice(), b"path\0", b"path\0\0line", b"path\0nope\0line"] {
        assert_eq!(run(OsStr::new("line"), Cursor::new(input), Vec::new()), 2);
    }
    for pattern in [OsStr::new("["), OsStr::from_bytes(b"\xff")] {
        assert_eq!(run(pattern, Cursor::new(b""), Vec::new()), 2);
    }
    let mut output = Vec::new();
    assert_eq!(run(OsStr::new(r"(?-u:\xFF)"), Cursor::new(b"file\0\x31\0raw\xff"), &mut output), 1);
    assert_eq!(output, b"file:1:raw\xff\n");
}

#[test]
fn facade_token_exemption_checks_boundaries_and_remaining_matches() {
    let token = ["author/umpire/", "tests"].concat();
    let expression = ["umpire/", "tests"].concat();
    for line in [format!("'{token}/consumer'"), format!("{token}\t")] {
        assert_eq!(
            run(OsStr::new(&expression), Cursor::new(format!("file\0\x31\0{line}\n")), Vec::new()),
            0
        );
    }
    for line in [
        format!("not_{token}/x"),
        format!("{token}_suffix"),
        format!("{token}/x /repo/{expression}/x"),
    ] {
        assert_eq!(
            run(OsStr::new(&expression), Cursor::new(format!("file\0\x31\0{line}\n")), Vec::new()),
            1
        );
    }
}

#[test]
fn native_scan_handles_git_binary_missing_and_symlink_policy() -> io::Result<()> {
    let scratch = tempfile::tempdir()?;
    let text = scratch.path().join("text");
    let binary = scratch.path().join("binary");
    let link = scratch.path().join("link");
    let missing = scratch.path().join("missing");
    fs::write(&text, b"safe\nretired\xff\n")?;
    fs::write(&binary, b"retired\0bytes")?;
    symlink(&text, &link)?;
    let files = vec![
        (binary.clone(), b"binary".to_vec()),
        (missing, b"missing".to_vec()),
        (link.clone(), b"link".to_vec()),
        (text.clone(), b"path\xff".to_vec()),
    ];
    let mut output = Vec::new();
    assert_eq!(scan(OsStr::new("retired"), &files, true, &mut output)?, 1);
    assert_eq!(output, b"path\xff:2:retired\xff\n");
    assert!(scan(OsStr::new("retired"), &[(link, b"link".to_vec())], false, Vec::new()).is_err());
    assert_eq!(scan(OsStr::new("retired"), &[(binary, b"binary".to_vec())], false, Vec::new())?, 1);
    assert!(scan(OsStr::new("["), &[], true, Vec::new()).is_err());
    Ok(())
}

#[test]
fn writer_failure_returns_error_status() {
    struct Refuse;
    impl Write for Refuse {
        fn write(&mut self, _: &[u8]) -> io::Result<usize> {
            Err(io::Error::other("control"))
        }
        fn flush(&mut self) -> io::Result<()> {
            Ok(())
        }
    }
    assert_eq!(run(OsStr::new("x"), Cursor::new(b"file\0\x31\0x\n"), Refuse), 2);
}
