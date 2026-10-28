//! Repository-owned historical exclusions over the shared native matcher.
use std::{
    ffi::{OsStr, OsString},
    io::{self, Write},
    os::unix::{ffi::OsStringExt, process::ExitStatusExt},
    path::{Path, PathBuf},
    process::Command,
};

const HISTORICAL: &[&str] = &[
    ":!model/archived",
    ":!.flow",
    ":!model/umpire",
    ":!.plans/umpire-migration-audit.json",
    ":!.plans/umpire-migration-manifest.json",
    ":!model/design/rust-verdict-measurements.md",
];
const LAYOUT_HISTORY: &[&str] = &[
    ":!.plans/archive",
    ":!.turbo/plans",
    ":!model/temporal/feature/activity/compatibility/original",
];
const LEGACY: &str = r#"umpire[-_](wasm|tests)|umpire/(wasm|tests)(/|["`[:space:]]|$)"#;
const LAYOUT: &str = r#"(^|[^[:alnum:]_])umpire[-_](lift|annotations)([^[:alnum:]_]|$)|umpire/(dsl|suite|checker|lift|runtime|proto|testing|fixtures|ir-derive|catalog-build)(/|["`[:space:]]|$)|umpire/annotations(/(src|tests|Cargo[.]toml|README[.]md|build[.]rs)(/|["`[:space:]]|$)|["`[:space:]]|$)"#;

fn inventory(root: &Path, extra: &[&str]) -> io::Result<Result<Vec<(PathBuf, Vec<u8>)>, i32>> {
    inventory_with(root, extra, OsStr::new("git"))
}

fn inventory_with(
    root: &Path,
    extra: &[&str],
    git: &OsStr,
) -> io::Result<Result<Vec<(PathBuf, Vec<u8>)>, i32>> {
    let mut command = Command::new(git);
    command
        .current_dir(root)
        .args(["ls-files", "--cached", "--others", "--exclude-standard", "-z", "--", "."])
        .args(HISTORICAL)
        .args(extra);
    let output = xtask::gate::scheduler::execute(&command, false, false)?;
    if !output.status.success() {
        io::stderr().write_all(&output.stderr)?;
        return Ok(Err(output
            .status
            .code()
            .unwrap_or_else(|| 128 + output.status.signal().unwrap_or(0))));
    }
    if !output.stdout.is_empty() && output.stdout.last() != Some(&0) {
        return Err(io::Error::new(io::ErrorKind::InvalidData, "unterminated Git path inventory"));
    }
    let mut paths = Vec::new();
    for bytes in output.stdout.split(|byte| *byte == 0).filter(|path| !path.is_empty()) {
        let relative = PathBuf::from(OsString::from_vec(bytes.to_vec()));
        if relative.is_absolute()
            || relative.components().any(|part| matches!(part, std::path::Component::ParentDir))
        {
            return Err(io::Error::new(io::ErrorKind::InvalidData, "Git path escapes repository"));
        }
        paths.push((root.join(relative), bytes.to_vec()));
    }
    paths.sort_by(|left, right| left.1.cmp(&right.1));
    paths.dedup_by(|left, right| left.1 == right.1);
    Ok(Ok(paths))
}

fn check(root: &Path) -> io::Result<i32> {
    for (pattern, history) in [(LEGACY, &[][..]), (LAYOUT, LAYOUT_HISTORY)] {
        let paths = match inventory(root, history)? {
            Ok(paths) => paths,
            Err(status) => return Ok(status),
        };
        let status =
            xtask::retired_names::scan(OsStr::new(pattern), &paths, true, io::stdout().lock())?;
        if status != 0 {
            writeln!(io::stderr(), "retired Umpire crate name or path in active files")?;
            return Ok(status);
        }
    }
    writeln!(io::stdout(), "retired Umpire crate names: clean")?;
    Ok(0)
}

/// Scan a repository root, preserving Git failures and refusing all local I/O failures.
pub(crate) fn run(root: &Path) -> i32 {
    match check(root) {
        Ok(status) => status,
        Err(error) => {
            let _ = writeln!(io::stderr(), "retired-name scan failed: {error}");
            2
        }
    }
}

#[cfg(test)]
#[path = "retired_name_controls.rs"]
mod tests;
