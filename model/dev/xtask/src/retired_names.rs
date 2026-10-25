//! Filter `git grep -n -z` records, exempting author-facade test paths.
use std::{
    ffi::OsStr,
    io::{self, BufRead, Write},
    process::{Command, Stdio},
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

/// Return 0 for exempt/empty input, 1 for forbidden matches, 2 for malformed
/// records or I/O failure, and the external grep failure status otherwise.
/// Records retain raw path/line bytes; facade-token boundaries use ASCII whitespace.
pub fn run(pattern: &OsStr, mut input: impl BufRead, mut output: impl Write) -> i32 {
    match filter(pattern, &mut input, &mut output) {
        Ok(status) => status,
        Err(error) => {
            let _ = writeln!(io::stderr(), "retired-name filtering failed: {error}");
            2
        }
    }
}

fn filter(pattern: &OsStr, input: &mut impl BufRead, output: &mut impl Write) -> io::Result<i32> {
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
        let mut probe = exempt(&line);
        if probe != line {
            probe.push(b'\n');
            let mut child = Command::new("grep")
                .args([OsStr::new("-E"), OsStr::new("-q"), OsStr::new("--"), pattern])
                .stdin(Stdio::piped())
                .spawn()?;
            let write = child
                .stdin
                .take()
                .ok_or_else(|| io::Error::other("missing grep stdin"))
                .and_then(|mut stdin| stdin.write_all(&probe));
            let status = child.wait()?;
            let code = status.code().unwrap_or(2);
            if code != 0 && code != 1 {
                return Ok(code);
            }
            if let Err(error) = write
                && error.kind() != io::ErrorKind::BrokenPipe
            {
                return Err(error);
            }
            if code == 1 {
                continue;
            }
        }
        for bytes in [path.as_slice(), b":", number.as_slice(), b":", line.as_slice(), b"\n"] {
            output.write_all(bytes)?;
        }
        found = true;
    }
    Ok(i32::from(found))
}
