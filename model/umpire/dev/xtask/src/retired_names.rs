//! Native retired-name matching with raw records and author-facade token exemptions.
use std::{
    ffi::OsStr,
    fs,
    io::{self, BufRead, Write},
    path::PathBuf,
};

fn exempt(line: &[u8]) -> Vec<u8> {
    let token = b"author/umpire/tests";
    let mut probe = Vec::with_capacity(line.len());
    let mut offset = 0;
    while offset < line.len() {
        let before = offset == 0
            || line[offset - 1].is_ascii_whitespace()
            || b"/\"'`=([{ ".contains(&line[offset - 1]);
        let end = offset + token.len();
        if before
            && line[offset..].starts_with(token)
            && (end == line.len()
                || line[end].is_ascii_whitespace()
                || b"/\"'`)]}".contains(&line[end]))
        {
            probe.extend_from_slice(b"author/umpire/facade_checks");
            offset = end;
        } else {
            probe.push(line[offset]);
            offset += 1;
        }
    }
    probe
}

fn field(input: &mut impl BufRead, label: &str) -> io::Result<Vec<u8>> {
    let mut bytes = Vec::new();
    input.read_until(0, &mut bytes)?;
    if bytes.pop() != Some(0) {
        return Err(io::Error::new(
            io::ErrorKind::InvalidData,
            format!("missing {label} separator"),
        ));
    }
    Ok(bytes)
}

fn pattern(pattern: &OsStr) -> io::Result<regex::bytes::Regex> {
    let pattern = pattern.to_str().ok_or_else(|| {
        io::Error::new(io::ErrorKind::InvalidInput, "retired-name pattern must be UTF-8")
    })?;
    regex::bytes::RegexBuilder::new(pattern)
        .unicode(false)
        .build()
        .map_err(|error| io::Error::new(io::ErrorKind::InvalidInput, error))
}

fn record(output: &mut impl Write, path: &[u8], number: &[u8], line: &[u8]) -> io::Result<()> {
    for bytes in [path, b":", number, b":", line, b"\n"] {
        output.write_all(bytes)?;
    }
    Ok(())
}

/// Return 0 for exempt/empty input, 1 for forbidden matches, and 2 for invalid
/// patterns, malformed records or I/O failure. Paths and line bytes remain raw.
pub fn run(pattern: &OsStr, mut input: impl BufRead, mut output: impl Write) -> i32 {
    let result = pattern_matches(pattern, &mut input, &mut output);
    match result {
        Ok(status) => status,
        Err(error) => {
            let _ = writeln!(io::stderr(), "retired-name filtering failed: {error}");
            2
        }
    }
}

fn pattern_matches(
    expression: &OsStr,
    input: &mut impl BufRead,
    output: &mut impl Write,
) -> io::Result<i32> {
    let pattern = pattern(expression)?;
    let mut found = false;
    while !input.fill_buf()?.is_empty() {
        let path = field(input, "path")?;
        let number = field(input, "line number")?;
        if number.is_empty() || !number.iter().all(u8::is_ascii_digit) {
            return Err(io::Error::new(io::ErrorKind::InvalidData, "invalid grep line number"));
        }
        let mut line = Vec::new();
        input.read_until(b'\n', &mut line)?;
        if line.last() == Some(&b'\n') {
            line.pop();
        }
        if pattern.is_match(&exempt(&line)) {
            record(output, &path, &number, &line)?;
            found = true;
        }
    }
    Ok(i32::from(found))
}

/// Scan regular files without a grep child. Each entry carries its physical path
/// and raw display path; `skip_binary` preserves the repository policy's `-I` mode.
/// In binary-skipping Git mode, missing tracked files and symlinks are skipped.
pub fn scan(
    expression: &OsStr,
    files: &[(PathBuf, Vec<u8>)],
    skip_binary: bool,
    mut output: impl Write,
) -> io::Result<i32> {
    let pattern = pattern(expression)?;
    let mut found = false;
    for (file, display) in files {
        let metadata = match fs::symlink_metadata(file) {
            Ok(metadata) => metadata,
            Err(error) if skip_binary && error.kind() == io::ErrorKind::NotFound => continue,
            Err(error) => return Err(error),
        };
        if skip_binary && !metadata.is_file() {
            continue;
        }
        if !metadata.is_file() {
            return Err(io::Error::new(
                io::ErrorKind::InvalidData,
                "retired-name input is not a file",
            ));
        }
        let bytes = fs::read(file)?;
        if skip_binary && bytes.contains(&0) {
            continue;
        }
        let mut input = bytes.as_slice();
        let mut number = 0_usize;
        while !input.is_empty() {
            let mut line = Vec::new();
            input.read_until(b'\n', &mut line)?;
            number += 1;
            if line.last() == Some(&b'\n') {
                line.pop();
            }
            if pattern.is_match(&exempt(&line)) {
                record(&mut output, display, number.to_string().as_bytes(), &line)?;
                found = true;
            }
        }
    }
    Ok(i32::from(found))
}

#[cfg(test)]
#[path = "retired_name_controls.rs"]
mod controls;
