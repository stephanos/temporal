use serde::Serialize;
use sha2::{Digest, Sha256};
use std::collections::BTreeMap;
use std::fs::OpenOptions;
use std::io::{self, BufRead, BufReader, Read, Write};
use std::path::{Path, PathBuf};
use std::process::{Command, Stdio};

type Result<T> = std::result::Result<T, Box<dyn std::error::Error>>;
const MAX_TREE_BYTES: u64 = 128 * 1024 * 1024;
const MAX_BLOB_BYTES: u64 = 64 * 1024 * 1024;

#[derive(Serialize)]
struct Report {
    schema: &'static str,
    definition: &'static str,
    revision: String,
    tree: String,
    repository_files: usize,
    repository_bytes: u64,
    paths_sha256: String,
    groups: BTreeMap<String, Metrics>,
    files: Vec<File>,
}

#[derive(Serialize)]
struct File {
    path: String,
    git_mode: String,
    git_blob: String,
    sha256: String,
    bytes: u64,
    #[serde(flatten, skip_serializing_if = "Option::is_none")]
    source: Option<Source>,
}

#[derive(Serialize)]
struct Source {
    language: &'static str,
    category: &'static str,
    subsystem: &'static str,
    total_lines: u64,
    nonblank_lines: u64,
    code_lines: u64,
}

#[derive(Default, Serialize)]
struct Metrics {
    files: u64,
    bytes: u64,
    total_lines: u64,
    nonblank_lines: u64,
    code_lines: u64,
}

fn git(repo: &Path) -> Command {
    let mut cmd = Command::new("git");
    cmd.arg("--no-replace-objects")
        .arg("-C")
        .arg(repo)
        .env_remove("GIT_DIR")
        .env_remove("GIT_WORK_TREE")
        .env_remove("GIT_INDEX_FILE")
        .env_remove("GIT_OBJECT_DIRECTORY")
        .env_remove("GIT_ALTERNATE_OBJECT_DIRECTORIES")
        .stdin(Stdio::null())
        .stdout(Stdio::piped())
        .stderr(Stdio::inherit());
    cmd
}

fn git_output(repo: &Path, args: &[&str], limit: u64) -> Result<Vec<u8>> {
    let mut child = git(repo).args(args).spawn()?;
    let mut bytes = Vec::new();
    let result = child
        .stdout
        .take()
        .ok_or("missing git stdout")?
        .take(limit + 1)
        .read_to_end(&mut bytes);
    if result.is_err() || bytes.len() as u64 > limit {
        child.kill().ok();
    }
    let status = child.wait()?;
    result?;
    if bytes.len() as u64 > limit {
        return Err("Git output exceeds byte limit".into());
    }
    if !status.success() {
        return Err(format!("git {} failed", args[0]).into());
    }
    Ok(bytes)
}

fn valid_oid(oid: &str) -> bool {
    matches!(oid.len(), 40 | 64)
        && oid
            .bytes()
            .all(|byte| byte.is_ascii_digit() || (b'a'..=b'f').contains(&byte))
}

fn git_oid(repo: &Path, args: &[&str]) -> Result<String> {
    let output = git_output(repo, args, 128)?;
    let oid = std::str::from_utf8(&output)?.trim();
    if !valid_oid(oid) {
        return Err("git returned an invalid object ID".into());
    }
    Ok(oid.to_owned())
}

fn validate_path(path: &str) -> Result<()> {
    if path.is_empty()
        || path.chars().any(char::is_control)
        || path.split('/').any(|part| matches!(part, "" | "." | ".."))
    {
        return Err("invalid Git tree path".into());
    }
    Ok(())
}

fn inventory(repo: &Path, revision: &str) -> Result<Report> {
    if revision.is_empty()
        || revision.starts_with('-')
        || revision.chars().any(char::is_control)
        || revision.len() > 4096
    {
        return Err("invalid revision".into());
    }
    let revision = git_oid(
        repo,
        &[
            "rev-parse",
            "--verify",
            "--end-of-options",
            &format!("{revision}^{{commit}}"),
        ],
    )?;
    let tree = git_oid(
        repo,
        &["rev-parse", "--verify", &format!("{revision}^{{tree}}")],
    )?;
    let listing = git_output(
        repo,
        &["ls-tree", "-rz", "--full-tree", &tree],
        MAX_TREE_BYTES,
    )?;
    let mut entries = Vec::new();
    for record in listing
        .split(|byte| *byte == 0)
        .filter(|record| !record.is_empty())
    {
        let (metadata, path) = record.split_at(
            record
                .iter()
                .position(|byte| *byte == b'\t')
                .ok_or("invalid Git tree entry")?,
        );
        let metadata: Vec<_> = std::str::from_utf8(metadata)?.split(' ').collect();
        if metadata.len() != 3 || !valid_oid(metadata[2]) {
            return Err("invalid Git tree metadata".into());
        }
        let path = std::str::from_utf8(&path[1..])?;
        validate_path(path)?;
        if metadata[1] == "commit" && metadata[0] == "160000" {
            continue;
        }
        if metadata[1] != "blob" || !matches!(metadata[0], "100644" | "100755" | "120000") {
            return Err("unsupported Git tree entry".into());
        }
        entries.push((
            path.to_owned(),
            metadata[0].to_owned(),
            metadata[2].to_owned(),
        ));
    }
    entries.sort();
    let mut process = git(repo)
        .args(["cat-file", "--batch"])
        .stdin(Stdio::piped())
        .spawn()?;
    let result = (|| -> Result<Report> {
        let mut input = process.stdin.take().ok_or("missing git stdin")?;
        let mut output = BufReader::new(process.stdout.take().ok_or("missing git stdout")?);
        let mut files = Vec::new();
        let mut groups: BTreeMap<String, Metrics> = BTreeMap::new();
        let mut paths = Sha256::new();
        let mut repository_bytes = 0;
        for (path, mode, oid) in entries {
            writeln!(input, "{oid}")?;
            input.flush()?;
            let raw = read_blob(&mut output, &oid)?;
            let size = raw.len() as u64;
            let source = if mode == "120000" {
                None
            } else {
                language(&path, &raw).map(|lang| {
                    let text = String::from_utf8_lossy(&raw);
                    Source {
                        language: lang,
                        category: category(&path, &raw),
                        subsystem: subsystem(&path),
                        total_lines: physical_lines(&text).count() as u64,
                        nonblank_lines: physical_lines(&text)
                            .filter(|line| !line.trim().is_empty())
                            .count() as u64,
                        code_lines: code_lines(&text, lang),
                    }
                })
            };
            if let Some(source) = &source {
                let group = groups
                    .entry(format!(
                        "{}/{}/{}",
                        source.subsystem, source.language, source.category
                    ))
                    .or_default();
                group.files += 1;
                group.bytes += size;
                group.total_lines += source.total_lines;
                group.nonblank_lines += source.nonblank_lines;
                group.code_lines += source.code_lines;
            }
            paths.update(path.as_bytes());
            paths.update(b"\n");
            repository_bytes += size;
            files.push(File {
                path,
                git_mode: mode,
                git_blob: oid,
                sha256: format!("{:x}", Sha256::digest(&raw)),
                bytes: size,
                source,
            });
        }
        drop(input);
        Ok(Report {
            schema: "gomad-source-size/v2",
            definition: "Physical nonblank lines with tokens outside comments; literal content counts as code. Not cloc. Categories classify whole files; inline tests remain in their production file. Generated markers take precedence over test paths. Full repository totals include all tracked blobs; per-file SHA256 and paths permit narrower comparisons.",
            revision,
            tree,
            repository_files: files.len(),
            repository_bytes,
            paths_sha256: format!("{:x}", paths.finalize()),
            groups,
            files,
        })
    })();
    if result.is_err() {
        process.kill().ok();
    }
    let status = process.wait()?;
    if !status.success() && result.is_ok() {
        return Err("Git batch reader failed".into());
    }
    result
}

fn read_blob(output: &mut impl BufRead, oid: &str) -> Result<Vec<u8>> {
    let mut header = Vec::new();
    Read::by_ref(output)
        .take(256)
        .read_until(b'\n', &mut header)?;
    let fields: Vec<_> = std::str::from_utf8(&header)?.split_whitespace().collect();
    if !header.ends_with(b"\n") || fields.len() != 3 || fields[0] != oid || fields[1] != "blob" {
        return Err("Git batch object header mismatch".into());
    }
    let size: u64 = fields[2].parse()?;
    if size > MAX_BLOB_BYTES {
        return Err(format!("blob exceeds {MAX_BLOB_BYTES} byte limit").into());
    }
    let mut raw = vec![0; usize::try_from(size)?];
    output.read_exact(&mut raw)?;
    let mut newline = [0];
    output.read_exact(&mut newline)?;
    if newline != [b'\n'] {
        return Err("invalid Git batch object terminator".into());
    }
    Ok(raw)
}

fn physical_lines(text: &str) -> impl Iterator<Item = &str> {
    text.split_inclusive([
        '\n', '\r', '\u{b}', '\u{c}', '\u{1c}', '\u{1d}', '\u{1e}', '\u{85}', '\u{2028}',
        '\u{2029}',
    ])
    .scan(false, |after_cr, part| {
        let skip = *after_cr && part == "\n";
        *after_cr = part.ends_with('\r');
        Some((!skip).then(|| {
            part.trim_end_matches([
                '\n', '\r', '\u{b}', '\u{c}', '\u{1c}', '\u{1d}', '\u{1e}', '\u{85}', '\u{2028}',
                '\u{2029}',
            ])
        }))
    })
    .flatten()
}

fn language(path: &str, raw: &[u8]) -> Option<&'static str> {
    let name = path.strip_suffix(".txt").unwrap_or(path);
    let name = name.strip_suffix(".tmpl").unwrap_or(name);
    if name.rsplit('/').next() == Some("Makefile") {
        return Some("Make");
    }
    let lang = match Path::new(name).extension().and_then(|ext| ext.to_str()) {
        Some("go") => Some("Go"),
        Some("rs") => Some("Rust"),
        Some("py") => Some("Python"),
        Some("sh" | "bash") => Some("Shell"),
        Some("c" | "h") => Some("C"),
        Some("cc" | "cpp") => Some("C++"),
        Some("js") => Some("JavaScript"),
        Some("ts" | "tsx") => Some("TypeScript"),
        Some("sql") => Some("SQL"),
        Some("yaml" | "yml") => Some("YAML"),
        Some("toml") => Some("TOML"),
        Some("proto") => Some("Protobuf"),
        Some("css") => Some("CSS"),
        _ => None,
    };
    lang.or_else(|| {
        if !raw.starts_with(b"#!") {
            return None;
        }
        let first = raw.split(|byte| *byte == b'\n').next().unwrap_or_default();
        if first.windows(6).any(|part| part == b"python") {
            Some("Python")
        } else if first.windows(2).any(|part| part == b"sh") {
            Some("Shell")
        } else {
            None
        }
    })
}

fn category(path: &str, raw: &[u8]) -> &'static str {
    let prefix = &raw[..raw.len().min(2048)];
    let generated = prefix.windows(10).any(|part| part == b"@generated")
        || prefix.split(|byte| *byte == b'\n').any(|line| {
            line.windows(14)
                .position(|part| part == b"Code generated")
                .is_some_and(|start| {
                    line[start + 14..]
                        .windows(11)
                        .any(|part| part == b"DO NOT EDIT")
                })
        });
    if generated {
        return "generated";
    }
    let stem = Path::new(path)
        .file_stem()
        .and_then(|stem| stem.to_str())
        .unwrap_or("");
    if path
        .split('/')
        .any(|part| matches!(part, "testdata" | "tests"))
        || path.contains("/testfixture/")
        || stem.starts_with("test_")
        || stem.ends_with("_test")
        || stem.ends_with("_tests")
    {
        "tests"
    } else {
        "production"
    }
}

fn subsystem(path: &str) -> &'static str {
    for (prefix, name) in [
        (
            "tools/gomad/internal/backends/wasm/wasmhost/",
            "preexisting-rust-helper",
        ),
        ("tools/gomad/rust/wasmhost/", "linked-rust-engine"),
        ("tools/gomad/rust/gomad/", "rust-host"),
        ("tools/gomad/cmd/", "go-host-or-reference-fixtures"),
        (
            "tools/gomad/internal/runner/",
            "go-host-or-reference-fixtures",
        ),
        (
            "tools/gomad/internal/host/",
            "go-host-or-reference-fixtures",
        ),
        (
            "tools/gomad/internal/evidence/",
            "go-host-or-reference-fixtures",
        ),
        (
            "tools/gomad/internal/model/wasi/",
            "retained-wasi-reference",
        ),
        (
            "tools/gomad/internal/",
            "go-model-oracle-toolchain-and-support",
        ),
        ("tools/gomad/", "gomad-support"),
    ] {
        if path.starts_with(prefix) {
            return name;
        }
    }
    "repository-other"
}

fn code_lines(text: &str, lang: &str) -> u64 {
    let hash_comments = matches!(lang, "Python" | "Shell" | "Make" | "YAML" | "TOML");
    let slash_comments = matches!(
        lang,
        "Go" | "Rust" | "C" | "C++" | "JavaScript" | "TypeScript" | "Protobuf" | "CSS"
    );
    let mut block = 0;
    let mut literal: Option<String> = None;
    let mut raw_literal = false;
    let mut count = 0;
    for line in physical_lines(text) {
        let mut i = 0;
        let mut has_code = false;
        while i < line.len() {
            let tail = &line[i..];
            let ch = tail.chars().next().unwrap();
            if block > 0 {
                if lang == "Rust" && tail.starts_with("/*") {
                    block += 1;
                    i += 2;
                } else if tail.starts_with("*/") {
                    block -= 1;
                    i += 2;
                } else {
                    i += ch.len_utf8();
                }
            } else if let Some(end) = &literal {
                has_code |= !line.trim().is_empty();
                if tail.starts_with(end.as_str()) {
                    i += end.len();
                    literal = None;
                    raw_literal = false;
                } else if ch == '\\' && end != "`" && !raw_literal {
                    i += 1;
                    if let Some(escaped) = line[i..].chars().next() {
                        i += escaped.len_utf8();
                    }
                } else {
                    i += ch.len_utf8();
                }
            } else if ch.is_whitespace() {
                i += ch.len_utf8();
            } else if (hash_comments && ch == '#')
                || (slash_comments && tail.starts_with("//"))
                || (lang == "SQL" && tail.starts_with("--"))
            {
                break;
            } else if (slash_comments || lang == "SQL") && tail.starts_with("/*") {
                block = 1;
                i += 2;
            } else {
                has_code = true;
                let rust_raw = if lang == "Rust" {
                    tail.strip_prefix("br")
                        .or_else(|| tail.strip_prefix('r'))
                        .and_then(|rest| {
                            let hashes = rest.bytes().take_while(|byte| *byte == b'#').count();
                            rest[hashes..].starts_with('"').then_some(hashes)
                        })
                } else {
                    None
                };
                if let Some(hashes) = rust_raw {
                    literal = Some(format!("\"{}", "#".repeat(hashes)));
                    raw_literal = true;
                    i += if tail.starts_with("br") {
                        3 + hashes
                    } else {
                        2 + hashes
                    };
                } else if lang == "Python"
                    && (tail.starts_with("\"\"\"") || tail.starts_with("'''"))
                {
                    literal = Some(tail[..3].to_owned());
                    i += 3;
                } else if matches!(ch, '"' | '\'' | '`')
                    && !(lang == "Rust" && ch == '\'' && !rust_character(tail))
                {
                    literal = Some(ch.to_string());
                    i += 1;
                } else {
                    i += ch.len_utf8();
                }
            }
        }
        count += u64::from(has_code);
        if matches!(lang, "Go" | "Rust" | "C" | "C++")
            && matches!(literal.as_deref(), Some("\"" | "'"))
        {
            literal = None;
        }
    }
    count
}

fn rust_character(text: &str) -> bool {
    let mut chars = text.chars().skip(1);
    match chars.next() {
        Some('\\') => chars.next().is_some() && chars.next() == Some('\''),
        Some('\'') | None => false,
        Some(_) => chars.next() == Some('\''),
    }
}

fn run() -> Result<()> {
    let mut args = std::env::args_os().skip(1);
    let (mut repo, mut revision, mut output) = (None, None, None);
    while let Some(arg) = args.next() {
        let slot =
            match arg.to_str() {
                Some("--repo") => &mut repo,
                Some("--rev") => &mut revision,
                Some("--output") => &mut output,
                _ => return Err(
                    "usage: gomad-source-size --repo REPOSITORY --rev REVISION [--output NEW_FILE]"
                        .into(),
                ),
            };
        if slot.is_some() {
            return Err("duplicate argument".into());
        }
        *slot = Some(args.next().ok_or("missing argument value")?);
    }
    let repo = PathBuf::from(repo.ok_or("--repo is required")?).canonicalize()?;
    let revision = revision
        .ok_or("--rev is required")?
        .into_string()
        .map_err(|_| "revision must be UTF-8")?;
    let report = inventory(&repo, &revision)?;
    let mut writer: Box<dyn Write> = match output {
        Some(path) => Box::new(OpenOptions::new().write(true).create_new(true).open(path)?),
        None => Box::new(io::stdout().lock()),
    };
    serde_json::to_writer_pretty(&mut writer, &report)?;
    writeln!(writer)?;
    writer.flush()?;
    Ok(())
}

fn main() {
    if let Err(error) = run() {
        eprintln!("gomad-source-size: {error}");
        std::process::exit(1);
    }
}

#[cfg(test)]
mod tests;
