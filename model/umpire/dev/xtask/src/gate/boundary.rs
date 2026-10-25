use crate::{Error, Result};
use serde::Deserialize;
use std::{
    ffi::OsStr,
    fs,
    io::{self, Write},
    os::unix::ffi::OsStrExt,
    path::{Path, PathBuf},
    process::Command,
};
const POLICY: &str = "dev/gate/policy/framework-boundary.txt";
#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Policy {
    version: u32,
    forbidden: Vec<String>,
    retired_pattern: String,
    organization_prefix: String,
    controls: Vec<String>,
}
fn policy(root: &Path) -> Result<Policy> {
    let path = root.join(POLICY);
    if !fs::symlink_metadata(&path)?.is_file() {
        return Err(Error::Invalid("framework boundary policy must be a regular file".into()));
    }
    let policy: Policy = toml::from_str(&fs::read_to_string(path)?)?;
    if policy.version != 1
        || policy.forbidden.is_empty()
        || policy.forbidden.iter().any(String::is_empty)
        || policy.retired_pattern.is_empty()
        || policy.organization_prefix.is_empty()
        || policy.controls.is_empty()
        || policy.controls.iter().any(String::is_empty)
    {
        return Err(Error::Invalid("invalid or empty framework boundary policy".into()));
    }
    Ok(policy)
}
fn files(root: &Path, target: &Path) -> Result<Vec<PathBuf>> {
    let target = normalize(target);
    let root = fs::canonicalize(root)?;
    if root.starts_with(&target) {
        return Err(Error::Invalid("output target must not hide framework sources".into()));
    }
    if target.starts_with(&root)
        && target != root.join("target")
        && !target.starts_with(root.join("target"))
        && target != root.join("build")
        && !target.starts_with(root.join("build"))
    {
        return Err(Error::Invalid("output target overlaps framework source inventory".into()));
    }
    let target = if target.exists() { fs::canonicalize(target)? } else { target };
    if root.starts_with(&target) {
        return Err(Error::Invalid("output target must not hide framework sources".into()));
    }
    let mut output = Vec::new();
    inventory(&root, &root, &target, &mut output)?;
    Ok(output)
}
fn normalize(path: &Path) -> PathBuf {
    let mut result = PathBuf::new();
    for part in path.components() {
        match part {
            std::path::Component::ParentDir => {
                result.pop();
            }
            std::path::Component::CurDir => (),
            other => result.push(other.as_os_str()),
        }
    }
    result
}
fn inventory(root: &Path, path: &Path, target: &Path, output: &mut Vec<PathBuf>) -> Result<()> {
    let mut children =
        fs::read_dir(path)?.map(|e| e.map(|e| e.path())).collect::<io::Result<Vec<_>>>()?;
    children.sort();
    for child in children {
        let relative = child.strip_prefix(root).map_err(|e| Error::Invalid(e.to_string()))?;
        if relative == Path::new(".git")
            || relative == Path::new("target")
            || relative == Path::new("build")
            || relative == Path::new(".bin")
            || child == target
        {
            continue;
        }
        let meta = fs::symlink_metadata(&child)?;
        if meta.file_type().is_symlink() {
            return Err(Error::Invalid(format!(
                "framework source symlink refused: {}",
                relative.display()
            )));
        }
        if meta.is_dir() {
            inventory(root, &child, target, output)?;
        } else if meta.is_file() {
            output.push(child);
        } else {
            return Err(Error::Invalid(format!(
                "framework input is not a regular file: {}",
                relative.display()
            )));
        }
    }
    Ok(())
}
fn contains(root: &Path, path: &Path, label: &str) -> Result<()> {
    let resolved = fs::canonicalize(path).map_err(|e| {
        Error::Invalid(format!("{label}: missing or inaccessible {}: {e}", path.display()))
    })?;
    if !resolved.starts_with(root) {
        return Err(Error::Invalid(format!(
            "{label}: path escapes framework root: {}",
            path.display()
        )));
    }
    Ok(())
}
fn forbidden(bytes: &[u8], p: &Policy, allow: bool) -> bool {
    p.forbidden.iter().any(|fragment| {
        bytes.windows(fragment.len()).enumerate().any(|(at, probe)| {
            if probe != fragment.as_bytes() {
                return false;
            }
            if allow {
                let prefix = p.organization_prefix.as_bytes();
                for start in 0..=at {
                    if bytes[start..].starts_with(prefix) && at < start + prefix.len() {
                        let end = start + prefix.len();
                        if bytes
                            .get(end)
                            .is_some_and(|b| b.is_ascii_alphanumeric() || *b == b'_' || *b == b'-')
                            && (start == 0 || !bytes[start - 1].is_ascii_alphanumeric())
                        {
                            return false;
                        }
                    }
                }
            }
            true
        })
    })
}
pub(super) fn check(root: &Path, target: &Path) -> Result<()> {
    let p = policy(root)?;
    for file in files(root, target)? {
        let relative = file.strip_prefix(root).map_err(|e| Error::Invalid(e.to_string()))?;
        if relative == Path::new(POLICY) {
            continue;
        }
        if forbidden(relative.as_os_str().as_bytes(), &p, false) {
            return Err(Error::Invalid(format!(
                "framework forbidden name in path: {}",
                relative.display()
            )));
        }
        let bytes = fs::read(&file)?;
        for (line, bytes) in bytes.split(|b| *b == b'\n').enumerate() {
            if forbidden(bytes, &p, true) {
                return Err(Error::Invalid(format!(
                    "framework forbidden name: {}:{}",
                    relative.display(),
                    line + 1
                )));
            }
        }
        if file.file_name() == Some(OsStr::new("Cargo.toml")) {
            manifest(root, &file)?;
        }
        if file.file_name() == Some(OsStr::new("go.mod")) {
            go_paths(root, &file, &bytes)?;
        }
        if file.extension() == Some(OsStr::new("rs")) {
            source_paths(root, &file, &bytes)?;
        }
    }
    Ok(())
}
fn manifest(root: &Path, file: &Path) -> Result<()> {
    let value: toml::Value = toml::from_str(&fs::read_to_string(file)?)?;
    declarations(root, file, &value, false)
}
fn declarations(root: &Path, file: &Path, value: &toml::Value, paths: bool) -> Result<()> {
    if let Some(table) = value.as_table() {
        for (key, value) in table {
            if paths && key == "path" {
                let path = value
                    .as_str()
                    .ok_or_else(|| Error::Invalid("dependency path must be a string".into()))?;
                contains(
                    root,
                    &file.parent().unwrap_or(root).join(path),
                    "framework dependency containment",
                )?;
            } else {
                declarations(
                    root,
                    file,
                    value,
                    paths
                        || matches!(
                            key.as_str(),
                            "dependencies"
                                | "dev-dependencies"
                                | "build-dependencies"
                                | "patch"
                                | "replace"
                        ),
                )?;
            }
        }
    }
    Ok(())
}
fn go_paths(root: &Path, file: &Path, bytes: &[u8]) -> Result<()> {
    let text = String::from_utf8_lossy(bytes);
    for line in text.lines() {
        if let Some((_, right)) = line.split_once("=>") {
            let target = right.split_whitespace().next().unwrap_or("").trim_matches('"');
            if target.starts_with('.') || Path::new(target).is_absolute() {
                contains(
                    root,
                    &file.parent().unwrap_or(root).join(target),
                    "framework Go dependency containment",
                )?;
            }
        }
    }
    Ok(())
}
#[derive(Debug)]
enum Token {
    Word(String),
    Literal(String),
    Punctuation(char),
}
fn tokens(text: &str) -> Vec<Token> {
    let bytes = text.as_bytes();
    let mut result = Vec::new();
    let mut at = 0;
    while at < bytes.len() {
        if bytes[at..].starts_with(b"//") {
            at += 2;
            while at < bytes.len() && bytes[at] != b'\n' {
                at += 1;
            }
            continue;
        }
        if bytes[at..].starts_with(b"/*") {
            at += 2;
            let mut depth = 1;
            while at < bytes.len() && depth > 0 {
                if bytes[at..].starts_with(b"/*") {
                    depth += 1;
                    at += 2;
                } else if bytes[at..].starts_with(b"*/") {
                    depth -= 1;
                    at += 2;
                } else {
                    at += 1;
                }
            }
            continue;
        }
        if bytes[at] == b'r' {
            let mut quote = at + 1;
            while bytes.get(quote) == Some(&b'#') {
                quote += 1;
            }
            if bytes.get(quote) == Some(&b'"') {
                let suffix = format!("\"{}", "#".repeat(quote - at - 1));
                let start = quote + 1;
                if let Some(end) = text[start..].find(&suffix) {
                    result.push(Token::Literal(text[start..start + end].into()));
                    at = start + end + suffix.len();
                    continue;
                }
            }
        }
        if bytes[at] == b'"' {
            at += 1;
            let mut value = String::new();
            while at < bytes.len() && bytes[at] != b'"' {
                if bytes[at] == b'\\' && at + 1 < bytes.len() {
                    at += 1;
                }
                value.push(char::from(bytes[at]));
                at += 1;
            }
            at += usize::from(at < bytes.len());
            result.push(Token::Literal(value));
            continue;
        }
        if bytes[at].is_ascii_alphanumeric() || bytes[at] == b'_' {
            let start = at;
            while at < bytes.len() && (bytes[at].is_ascii_alphanumeric() || bytes[at] == b'_') {
                at += 1;
            }
            result.push(Token::Word(text[start..at].into()));
            continue;
        }
        if !bytes[at].is_ascii_whitespace() {
            result.push(Token::Punctuation(char::from(bytes[at])));
        }
        at += 1;
    }
    result
}
fn punctuation(token: Option<&Token>, expected: char) -> bool {
    matches!(token,Some(Token::Punctuation(actual)) if *actual==expected)
}
fn source_contains(root: &Path, path: &Path) -> Result<()> {
    let path = normalize(path);
    let mut existing = path.as_path();
    let mut suffix = Vec::new();
    while !existing.exists() {
        suffix.push(
            existing
                .file_name()
                .ok_or_else(|| Error::Invalid("invalid framework source path".into()))?,
        );
        existing = existing
            .parent()
            .ok_or_else(|| Error::Invalid("invalid framework source parent".into()))?;
    }
    let mut resolved = fs::canonicalize(existing)?;
    for part in suffix.into_iter().rev() {
        resolved.push(part);
    }
    if !resolved.starts_with(root) {
        return Err(Error::Invalid(format!(
            "framework source-path containment: path escapes framework root: {}",
            path.display()
        )));
    }
    Ok(())
}
fn source_paths(root: &Path, file: &Path, bytes: &[u8]) -> Result<()> {
    let text = String::from_utf8_lossy(bytes);
    let tokens = tokens(&text);
    let mut manifest = file.parent().unwrap_or(root);
    while !manifest.join("Cargo.toml").is_file() && manifest != root {
        manifest = manifest.parent().unwrap_or(root);
    }
    for (index, token) in tokens.iter().enumerate() {
        if let Token::Word(name) = token {
            if ["include", "include_str", "include_bytes"].contains(&name.as_str())
                && punctuation(tokens.get(index + 1), '!')
                && punctuation(tokens.get(index + 2), '(')
                && let Some(Token::Literal(path)) = tokens.get(index + 3)
            {
                contains(
                    root,
                    &file.parent().unwrap_or(root).join(path),
                    "framework source-path containment",
                )?;
            }
            if name == "path"
                && punctuation(tokens.get(index + 1), '=')
                && index >= 2
                && punctuation(tokens.get(index - 1), '[')
                && punctuation(tokens.get(index - 2), '#')
                && let Some(Token::Literal(path)) = tokens.get(index + 2)
            {
                contains(
                    root,
                    &file.parent().unwrap_or(root).join(path),
                    "framework source-path containment",
                )?;
            }
            if name == "env"
                && punctuation(tokens.get(index + 1), '!')
                && punctuation(tokens.get(index + 2), '(')
                && matches!(tokens.get(index+3),Some(Token::Literal(v)) if v=="CARGO_MANIFEST_DIR")
            {
                if punctuation(tokens.get(index + 5), ',')
                    && let Some(Token::Literal(path)) = tokens.get(index + 6)
                {
                    source_contains(root, &manifest.join(path.trim_start_matches('/')))?;
                }
                let mut end = index + 5;
                while punctuation(tokens.get(end), ')') {
                    end += 1;
                }
                let mut joined = manifest.to_owned();
                while punctuation(tokens.get(end), '.')
                    && matches!(tokens.get(end+1),Some(Token::Word(name)) if name=="join")
                    && punctuation(tokens.get(end + 2), '(')
                {
                    if let Some(Token::Literal(path)) = tokens.get(end + 3) {
                        joined.push(path);
                        source_contains(root, &joined)?;
                    }
                    end += 5;
                }
            }
        }
    }
    Ok(())
}
pub(super) fn metadata(root: &Path, bytes: &[u8]) -> Result<()> {
    let value: serde_json::Value = serde_json::from_slice(bytes)?;
    let packages = value["packages"]
        .as_array()
        .ok_or_else(|| Error::Invalid("Cargo metadata has no package graph".into()))?;
    for package in packages {
        if package["source"].is_null() {
            let path = package["manifest_path"]
                .as_str()
                .ok_or_else(|| Error::Invalid("Cargo local package lacks manifest".into()))?;
            contains(root, Path::new(path), "framework resolved dependency containment")?;
        }
    }
    Ok(())
}
pub(super) fn retired(root: &Path, target: &Path) -> Result<i32> {
    let p = policy(root)?;
    let mut records = Vec::new();
    for file in files(root, target)? {
        let relative = file.strip_prefix(root).map_err(|e| Error::Invalid(e.to_string()))?;
        if relative == Path::new(POLICY) {
            continue;
        }
        let output = Command::new("grep")
            .args(["-a", "-n", "-E", "--", &p.retired_pattern])
            .arg(&file)
            .env("UMPIRE_QUICK_CACHE", "0")
            .env("UMPIRE_CACHE_DISABLE", "1")
            .output()?;
        match output.status.code() {
            Some(0) => (),
            Some(1) => continue,
            Some(code) => {
                io::stderr().write_all(&output.stderr)?;
                return Ok(code);
            }
            None => return Ok(2),
        }
        for record in output.stdout.split_inclusive(|b| *b == b'\n') {
            let colon = record
                .iter()
                .position(|b| *b == b':')
                .ok_or_else(|| Error::Invalid("malformed retired-name grep record".into()))?;
            records.extend_from_slice(relative.as_os_str().as_bytes());
            records.push(0);
            records.extend_from_slice(&record[..colon]);
            records.push(0);
            records.extend_from_slice(&record[colon + 1..]);
            if record.last() != Some(&b'\n') {
                records.push(b'\n');
            }
        }
    }
    Ok(crate::retired_names::run(
        OsStr::new(&p.retired_pattern),
        io::Cursor::new(records),
        io::stdout().lock(),
    ))
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn declared_unused_dependency_escape_is_refused_before_cargo() -> Result<()> {
        let directory = tempfile::tempdir()?;
        let root = directory.path().join("root");
        fs::create_dir(&root)?;
        let outside = directory.path().join("outside");
        fs::create_dir(&outside)?;
        let manifest = root.join("Cargo.toml");
        fs::write(
            &manifest,
            "[target.'cfg(any())'.dev-dependencies]\nrenamed = { path = '../outside' }\n",
        )?;
        assert!(super::manifest(&root, &manifest).is_err());
        Ok(())
    }
    #[test]
    fn source_escape_is_structural_and_contained_deep_paths_work() -> Result<()> {
        let directory = tempfile::tempdir()?;
        let root = directory.path();
        let child = root.join("a/b/c/d");
        fs::create_dir_all(&child)?;
        fs::write(root.join("Cargo.toml"), "")?;
        fs::write(root.join("fixture"), "")?;
        let file = child.join("test.rs");
        source_paths(root, &file, b"include_str!(\"../../../../fixture\")")?;
        assert!(source_paths(root, &file, b"include_str!(\"../../../../../outside\")").is_err());
        Ok(())
    }
    #[test]
    fn target_cannot_hide_sources() -> Result<()> {
        let directory = tempfile::tempdir()?;
        assert!(files(directory.path(), directory.path()).is_err());
        assert!(files(directory.path(), &directory.path().join("author")).is_err());
        Ok(())
    }
}
