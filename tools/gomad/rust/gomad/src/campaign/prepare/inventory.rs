use super::{MAXIMUM, canonical, failure, process, read};
use serde::Deserialize;
use std::{
    collections::{BTreeMap, BTreeSet},
    fs,
    path::{Path, PathBuf},
    time::Instant,
};

pub(crate) fn verify(inputs: &BTreeMap<PathBuf, String>) -> Result<(), String> {
    for (path, expected) in inputs {
        if canonical::digest(&read(path, MAXIMUM)?) != *expected {
            return Err(failure(format!("build input changed: {}", path.display())));
        }
    }
    Ok(())
}

pub(crate) struct CompilerInventory {
    pub inputs: BTreeMap<PathBuf, String>,
    pub root: PathBuf,
    pub command: process::ObservedCommand,
}

pub(crate) fn compiler(
    compiler: &Path,
    directory: &Path,
    environment: &BTreeMap<String, String>,
    scratch: &Path,
    deadline: Instant,
) -> Result<(BTreeMap<PathBuf, String>, PathBuf), String> {
    let observed = observed_compiler(compiler, directory, environment, scratch, deadline)?;
    Ok((observed.inputs, observed.root))
}

pub(crate) fn observed_compiler(
    compiler: &Path,
    directory: &Path,
    environment: &BTreeMap<String, String>,
    scratch: &Path,
    deadline: Instant,
) -> Result<CompilerInventory, String> {
    #[derive(Deserialize)]
    #[serde(rename_all = "UPPERCASE")]
    struct Identity {
        gotooldir: PathBuf,
        goroot: PathBuf,
    }
    let command = process::observed_command(
        compiler,
        &[
            "env".into(),
            "-json".into(),
            "GOTOOLDIR".into(),
            "GOROOT".into(),
        ],
        directory,
        environment,
        scratch,
        64 << 10,
        deadline,
    )?;
    let identity: Identity = serde_json::from_slice(&command.stdout).map_err(failure)?;
    if !identity.gotooldir.is_absolute() || !identity.goroot.is_absolute() {
        return Err(failure("absolute compiler distribution is required"));
    }
    let root = fs::canonicalize(identity.goroot).map_err(failure)?;
    let mut paths = BTreeSet::new();
    paths.insert(root.join("VERSION"));
    for name in ["asm", "compile", "link"] {
        paths.insert(identity.gotooldir.join(name));
    }
    let mut headers = 0;
    for entry in fs::read_dir(root.join("pkg/include")).map_err(failure)? {
        let path = entry.map_err(failure)?.path();
        if path.extension().is_some_and(|extension| extension == "h") {
            headers += 1;
            paths.insert(path);
        }
    }
    if headers == 0 {
        return Err(failure("compiler assembler headers are missing"));
    }
    hash(paths).map(|inputs| CompilerInventory {
        inputs,
        root,
        command,
    })
}

fn hash(paths: BTreeSet<PathBuf>) -> Result<BTreeMap<PathBuf, String>, String> {
    paths
        .into_iter()
        .map(|path| read(&path, MAXIMUM).map(|data| (path, canonical::digest(&data))))
        .collect()
}

pub(crate) fn sources(
    data: &[u8],
    include: &Path,
    replacements: &BTreeMap<String, String>,
) -> Result<BTreeMap<PathBuf, String>, String> {
    #[derive(Deserialize)]
    #[serde(rename_all = "PascalCase")]
    struct Module {
        go_mod: Option<String>,
        replace: Option<serde_json::Value>,
    }
    #[derive(Deserialize)]
    #[serde(rename_all = "PascalCase")]
    struct Package {
        dir: PathBuf,
        #[serde(default)]
        go_files: Vec<String>,
        #[serde(default)]
        embed_files: Vec<String>,
        #[serde(default)]
        test_go_files: Vec<String>,
        #[serde(default, rename = "XTestGoFiles")]
        x_test_go_files: Vec<String>,
        #[serde(default)]
        test_embed_files: Vec<String>,
        #[serde(default, rename = "XTestEmbedFiles")]
        x_test_embed_files: Vec<String>,
        #[serde(default, rename = "SFiles")]
        s_files: Vec<String>,
        #[serde(default)]
        syso_files: Vec<String>,
        #[serde(default, rename = "HFiles")]
        h_files: Vec<String>,
        module: Option<Module>,
    }
    let mut paths = BTreeSet::new();
    let mut packages = 0;
    for value in serde_json::Deserializer::from_slice(data).into_iter::<Package>() {
        let package = value.map_err(failure)?;
        packages += 1;
        if let Some(module) = package.module {
            if module.replace.is_some() {
                return Err("unsupported_target: ambient module replacement is unsupported".into());
            }
            if let Some(path) = module.go_mod.filter(|value| !value.is_empty()) {
                let path = PathBuf::from(path);
                paths.insert(path.clone());
                let sum = path.with_file_name("go.sum");
                match fs::metadata(&sum) {
                    Ok(_) => {
                        paths.insert(sum);
                    }
                    Err(error) if error.kind() == std::io::ErrorKind::NotFound => (),
                    Err(error) => return Err(failure(error)),
                }
            }
        }
        for file in [
            &package.go_files,
            &package.embed_files,
            &package.test_go_files,
            &package.x_test_go_files,
            &package.test_embed_files,
            &package.x_test_embed_files,
            &package.s_files,
            &package.syso_files,
            &package.h_files,
        ]
        .into_iter()
        .flatten()
        {
            let path = package.dir.join(file);
            if let Some(replacement) = replacements.get(path.to_string_lossy().as_ref()) {
                if replacement.is_empty() {
                    return Err(failure("runtime overlay removed selected build input"));
                }
                paths.insert(PathBuf::from(replacement));
                match fs::metadata(&path) {
                    Err(error) if error.kind() == std::io::ErrorKind::NotFound => continue,
                    Err(error) => return Err(failure(error)),
                    Ok(_) => (),
                }
            }
            paths.insert(path);
        }
        for file in &package.s_files {
            let path = package.dir.join(file);
            assembly(
                &path,
                &package.dir,
                path.parent()
                    .ok_or_else(|| failure("assembly directory missing"))?,
                include,
                &mut paths,
                &mut BTreeSet::new(),
            )?;
        }
    }
    if packages == 0 || paths.is_empty() {
        return Err(failure("empty selected source inventory"));
    }
    hash(paths)
}

fn assembly(
    path: &Path,
    package: &Path,
    source: &Path,
    include: &Path,
    paths: &mut BTreeSet<PathBuf>,
    visited: &mut BTreeSet<PathBuf>,
) -> Result<(), String> {
    if !visited.insert(path.into()) {
        return Ok(());
    }
    let bytes = read(path, MAXIMUM)?;
    let text = std::str::from_utf8(&bytes).map_err(failure)?;
    let mut in_comment = false;
    for line in text.lines() {
        let mut code = String::new();
        let mut chars = line.chars().peekable();
        let mut quoted = None;
        while let Some(character) = chars.next() {
            if in_comment {
                if character == '*' && chars.peek() == Some(&'/') {
                    chars.next();
                    in_comment = false;
                }
                continue;
            }
            if let Some(quote) = quoted {
                code.push(character);
                if character == '\\' && quote != '`' {
                    if let Some(escaped) = chars.next() {
                        code.push(escaped);
                    }
                } else if character == quote {
                    quoted = None;
                }
                continue;
            }
            if matches!(character, '"' | '\'' | '`') {
                quoted = Some(character);
                code.push(character);
                continue;
            }
            if character == '/' && chars.peek() == Some(&'*') {
                chars.next();
                in_comment = true;
                code.push(' ');
                continue;
            }
            if character == '/' && chars.peek() == Some(&'/') {
                break;
            }
            code.push(character);
        }
        let Some(directive) = code.trim_start().strip_prefix('#') else {
            continue;
        };
        let directive = directive.trim_start();
        let Some(rest) = directive
            .strip_prefix("include")
            .filter(|rest| rest.starts_with(char::is_whitespace) || rest.starts_with('"'))
        else {
            continue;
        };
        let literal = rest.trim();
        let name: String = serde_json::from_str(literal).map_err(|_| {
            failure(format!(
                "unsupported assembler include in {}",
                path.display()
            ))
        })?;
        let mut candidates = vec![package.join(&name), source.join(&name)];
        if name != "go_asm.h" {
            candidates.push(include.join(&name));
        }
        let mut selected = None;
        for candidate in candidates {
            match fs::metadata(&candidate) {
                Ok(info) if info.is_file() => {
                    selected = Some(candidate);
                    break;
                }
                Ok(_) => return Err(failure("assembler include is not a regular file")),
                Err(error) if error.kind() == std::io::ErrorKind::NotFound => (),
                Err(error) => return Err(failure(error)),
            }
        }
        let Some(selected) = selected else {
            if name == "go_asm.h" {
                continue;
            }
            return Err(failure(format!(
                "assembler include cannot be captured: {name}"
            )));
        };
        paths.insert(selected.clone());
        assembly(&selected, package, source, include, paths, visited)?;
    }
    if in_comment {
        return Err(failure("unterminated assembler comment"));
    }
    Ok(())
}
