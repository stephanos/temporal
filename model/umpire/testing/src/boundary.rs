//! Lexical source-boundary checks with caller-owned vocabulary and exceptions.

use std::{
    fmt, fs, io,
    path::{Path, PathBuf},
};

/// One offending source line, relative to the scanned root.
#[derive(Debug, PartialEq, Eq)]
pub struct Violation {
    /// Source path relative to the root.
    pub path: PathBuf,
    /// One-based source line.
    pub line: usize,
}

impl fmt::Display for Violation {
    fn fmt(&self, out: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(out, "{}:{}: forbidden source dependency", self.path.display(), self.line)
    }
}

/// Scan every selected source file, reporting each offending line once in path order.
/// `allow` receives the whole line and the byte offset of each forbidden occurrence.
pub fn scan(
    root: &Path,
    extensions: &[&str],
    forbidden: &[&str],
    allow: impl Fn(&str, usize) -> bool,
) -> io::Result<Vec<Violation>> {
    let mut files = Vec::new();
    collect(root, extensions, &mut files)?;
    files.sort();
    let mut violations = Vec::new();
    for path in files {
        for (line, text) in fs::read_to_string(&path)?.lines().enumerate() {
            if forbidden
                .iter()
                .any(|word| text.match_indices(word).any(|(offset, _)| !allow(text, offset)))
            {
                violations.push(Violation {
                    path: path.strip_prefix(root).map_err(io::Error::other)?.into(),
                    line: line + 1,
                });
            }
        }
    }
    Ok(violations)
}

fn collect(root: &Path, extensions: &[&str], files: &mut Vec<PathBuf>) -> io::Result<()> {
    for entry in fs::read_dir(root)? {
        let entry = entry?;
        let kind = entry.file_type()?;
        let path = entry.path();
        if kind.is_symlink() {
            return Err(io::Error::other(format!(
                "source boundary contains symlink: {}",
                path.display()
            )));
        }
        if kind.is_dir() {
            collect(&path, extensions, files)?;
        } else if path
            .extension()
            .and_then(|value| value.to_str())
            .is_some_and(|ext| extensions.contains(&ext))
        {
            files.push(path);
        }
    }
    Ok(())
}
