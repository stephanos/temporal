#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn test_list_filters_summaries_without_losing_unicode_identifiers() {
        assert_eq!(
            parse_test_list(b"TestB\nTestA\nok pkg 1s\nwarning text\nTestA\nTest\xc3\x89\n"),
            ["TestA", "TestB", "TestÉ"]
        );
    }
    #[test]
    fn module_selection_preserves_nested_module_roots() {
        let root = Path::new("/fixture");
        let modules = vec![root.to_owned(), root.join("tests/nested")];
        assert_eq!(
            select_modules(root, &["./tests/...".into()], &modules).unwrap(),
            BTreeMap::from([
                (root.to_owned(), vec!["./tests/...".into()]),
                (root.join("tests/nested"), vec!["./...".into()])
            ])
        );
        assert_eq!(
            select_modules(root, &["./tests/nested/sub".into()], &modules).unwrap(),
            BTreeMap::from([(root.join("tests/nested"), vec!["./sub".into()])])
        );
    }
    #[test]
    fn compiler_environment_is_explicit_for_host_and_guest() {
        let original = BTreeMap::from([
            ("GOFLAGS".into(), "-mod=mod".into()),
            ("GOWORK".into(), "/bad".into()),
            ("HOME".into(), "/home".into()),
        ]);
        let host = go_environment(&original, false);
        assert_eq!(host["GOFLAGS"], "");
        assert_eq!(host["GOWORK"], "off");
        assert_eq!(host["HOME"], "/home");
        let guest = go_environment(&original, true);
        assert_eq!(guest["GOOS"], "wasip1");
        assert_eq!(guest["GOEXPERIMENT"], "nogreenteagc");
    }
}

use super::{
    inventory,
    options::Options,
    process::{Command, Context, Executor},
};
use serde::{Deserialize, Serialize};
use std::{
    collections::{BTreeMap, BTreeSet},
    path::{Path, PathBuf},
    time::Duration,
};
#[derive(Clone, Debug, Serialize, Deserialize, PartialEq)]
#[serde(deny_unknown_fields)]
pub struct SuiteMethod {
    pub suite: String,
    pub name: String,
    pub file: String,
    pub units: Vec<String>,
}
#[derive(Clone, Debug, Serialize, Deserialize, PartialEq)]
#[serde(deny_unknown_fields)]
pub struct SuiteRegistration {
    pub wrapper: String,
    pub suite: String,
    pub path: String,
    pub file: String,
    pub methods: Vec<String>,
}
impl From<inventory::SuiteMethod> for SuiteMethod {
    fn from(method: inventory::SuiteMethod) -> Self {
        Self {
            suite: method.suite,
            name: method.name,
            file: method.file,
            units: method.units,
        }
    }
}
impl From<inventory::SuiteRegistration> for SuiteRegistration {
    fn from(registration: inventory::SuiteRegistration) -> Self {
        Self {
            wrapper: registration.wrapper,
            suite: registration.suite,
            path: registration.path,
            file: registration.file,
            methods: registration.methods,
        }
    }
}
#[derive(Clone, Debug, Serialize, Deserialize, PartialEq)]
#[serde(deny_unknown_fields)]
pub struct Package {
    pub import_path: String,
    pub pattern: String,
    pub module_directory: String,
    pub target: String,
    pub tests: Vec<String>,
    pub units: BTreeMap<String, Vec<String>>,
    pub methods: Vec<SuiteMethod>,
    pub registrations: Vec<SuiteRegistration>,
    pub top_level_tests: Vec<String>,
    pub guest_excluded: BTreeMap<String, String>,
    pub list_count: usize,
    pub parsed_count: usize,
}
pub fn guest_tags() -> Vec<String> {
    [
        "disable_grpc_modules",
        "gomad_wasm_slim",
        "sqlite3_dotlk",
        "test_dep",
    ]
    .map(str::to_owned)
    .to_vec()
}
#[derive(Deserialize)]
#[serde(rename_all = "PascalCase")]
struct Listed {
    import_path: String,
    dir: PathBuf,
    #[serde(default)]
    go_files: Vec<String>,
    #[serde(default)]
    test_go_files: Vec<String>,
    #[serde(default)]
    x_test_go_files: Vec<String>,
    error: Option<ListError>,
}
#[derive(Deserialize)]
struct ListError {
    #[serde(rename = "Err")]
    err: String,
}
pub fn go_environment(
    original: &BTreeMap<String, String>,
    guest: bool,
) -> BTreeMap<String, String> {
    let mut env = original.clone();
    for key in [
        "GOFLAGS",
        "GOWORK",
        "GOTOOLCHAIN",
        "GOOS",
        "GOARCH",
        "CGO_ENABLED",
        "GOENV",
        "GOEXPERIMENT",
    ] {
        env.remove(key);
    }
    for (key, value) in [
        ("GOFLAGS", ""),
        ("GOWORK", "off"),
        ("GOTOOLCHAIN", "local"),
        ("GOENV", "off"),
        ("CGO_ENABLED", "0"),
    ] {
        env.insert(key.into(), value.into());
    }
    if guest {
        for (key, value) in [
            ("GOOS", "wasip1"),
            ("GOARCH", "wasm"),
            ("GOEXPERIMENT", "nogreenteagc"),
        ] {
            env.insert(key.into(), value.into());
        }
    }
    env
}
fn relative(root: &Path, path: &Path) -> Result<String, String> {
    path.strip_prefix(root)
        .map(|p| {
            if p.as_os_str().is_empty() {
                "./.".into()
            } else {
                format!("./{}", p.display())
            }
        })
        .map_err(|_| format!("package directory {} outside checkout", path.display()))
}
fn module_roots(root: &Path) -> Result<Vec<PathBuf>, String> {
    let mut modules = vec![root.to_owned()];
    let mut pending = vec![root.join("tests")];
    while let Some(dir) = pending.pop() {
        for entry in std::fs::read_dir(dir).map_err(|e| e.to_string())? {
            let entry = entry.map_err(|e| e.to_string())?;
            let kind = entry.file_type().map_err(|e| e.to_string())?;
            let name = entry.file_name();
            let name = name.to_string_lossy();
            if kind.is_dir() && name != "vendor" && !name.starts_with('.') {
                pending.push(entry.path());
            }
            if kind.is_file() && name == "go.mod" {
                modules.push(
                    entry
                        .path()
                        .parent()
                        .ok_or("go.mod has no parent")?
                        .to_owned(),
                );
            }
        }
    }
    modules.sort();
    modules.dedup();
    Ok(modules)
}
fn select_modules(
    root: &Path,
    patterns: &[String],
    modules: &[PathBuf],
) -> Result<BTreeMap<PathBuf, Vec<String>>, String> {
    let mut selected: BTreeMap<PathBuf, Vec<String>> = BTreeMap::new();
    for pattern in patterns {
        let recursive = pattern.ends_with("/...");
        let base = root.join(
            pattern
                .strip_suffix("/...")
                .unwrap_or(pattern)
                .trim_start_matches("./"),
        );
        let owner = modules
            .iter()
            .filter(|m| base.starts_with(m))
            .max_by_key(|m| m.as_os_str().len())
            .ok_or("package has no module")?;
        let mut target = relative(owner, &base)?;
        if recursive {
            target = if target == "./." {
                "./...".into()
            } else {
                format!("{target}/...")
            };
        }
        selected.entry(owner.clone()).or_default().push(target);
        if recursive {
            for module in modules {
                if module != owner && module.starts_with(&base) {
                    selected
                        .entry(module.clone())
                        .or_default()
                        .push("./...".into());
                }
            }
        }
    }
    for patterns in selected.values_mut() {
        patterns.sort();
        patterns.dedup();
    }
    Ok(selected)
}
fn list(
    executor: &mut impl Executor,
    context: &Context,
    options: &Options,
    directory: &Path,
    patterns: &[String],
    guest: bool,
) -> Result<Vec<Listed>, String> {
    let mut args = vec![
        "list".into(),
        "-mod=readonly".into(),
        "-e".into(),
        "-json=ImportPath,Dir,GoFiles,TestGoFiles,XTestGoFiles,Error".into(),
        "-tags".into(),
        guest_tags().join(","),
    ];
    args.extend_from_slice(patterns);
    let completed = executor.exec(
        context,
        &Command {
            path: options.compiler.clone(),
            args,
            dir: directory.to_owned(),
            env: go_environment(&options.environ, guest),
            timeout: Duration::from_secs(600),
        },
    );
    if completed.exit != 0 || completed.error.is_some() {
        return Err(format!(
            "go list: exit {} {:?}: {}",
            completed.exit,
            completed.error,
            String::from_utf8_lossy(&completed.stderr)
        ));
    }
    let packages = serde_json::Deserializer::from_slice(&completed.stdout)
        .into_iter::<Listed>()
        .collect::<Result<Vec<_>, _>>()
        .map_err(|e| e.to_string())?;
    if packages.is_empty() {
        return Err("go list matched no packages".into());
    }
    for package in &packages {
        if let Some(error) = &package.error {
            return Err(format!("go list {}: {}", package.import_path, error.err));
        }
    }
    context.check()?;
    Ok(packages)
}
pub fn parse_test_list(output: &[u8]) -> Vec<String> {
    let mut names = String::from_utf8_lossy(output)
        .lines()
        .map(str::trim)
        .filter(|line| {
            !line.is_empty()
                && line
                    .chars()
                    .enumerate()
                    .all(|(i, c)| c == '_' || c.is_alphabetic() || i > 0 && c.is_numeric())
        })
        .map(str::to_owned)
        .collect::<Vec<_>>();
    names.sort();
    names.dedup();
    names
}
fn list_tests(
    executor: &mut impl Executor,
    context: &Context,
    options: &Options,
    directory: &Path,
    target: &str,
    pattern: &str,
) -> Result<Vec<String>, String> {
    let completed = executor.exec(
        context,
        &Command {
            path: options.compiler.clone(),
            args: vec![
                "test".into(),
                "-mod=readonly".into(),
                "-count=1".into(),
                "-tags".into(),
                guest_tags().join(","),
                "-list".into(),
                "^Test".into(),
                target.into(),
            ],
            dir: directory.to_owned(),
            env: go_environment(&options.environ, false),
            timeout: Duration::from_mins(30),
        },
    );
    context.check()?;
    if completed.exit != 0 || completed.error.is_some() {
        return Err(format!(
            "go test -list {pattern}: exit {} {:?}: {}",
            completed.exit,
            completed.error,
            String::from_utf8_lossy(&completed.stderr)
        ));
    }
    Ok(parse_test_list(&completed.stdout))
}

fn embedded_suite_files(
    executor: &mut impl Executor,
    context: &Context,
    options: &Options,
    directory: &Path,
    import: &str,
) -> Result<Vec<PathBuf>, String> {
    let external = list(
        executor,
        context,
        options,
        directory,
        &[import.into()],
        false,
    )?;
    if external.len() != 1 || external[0].import_path != import {
        return Err(format!("unresolved embedded suite package {import}"));
    }
    Ok(external[0]
        .go_files
        .iter()
        .map(|file| external[0].dir.join(file))
        .collect())
}

pub fn enumerate(
    executor: &mut impl Executor,
    context: &Context,
    options: &Options,
) -> Result<Vec<Package>, String> {
    let mut packages = Vec::new();
    for (directory, patterns) in select_modules(
        &options.checkout,
        &options.packages,
        &module_roots(&options.checkout)?,
    )? {
        let host = list(executor, context, options, &directory, &patterns, false)?;
        let guest = list(executor, context, options, &directory, &patterns, true)?;
        let guest_files: BTreeMap<_, BTreeSet<_>> = guest
            .into_iter()
            .map(|pkg| {
                (
                    pkg.import_path,
                    pkg.go_files
                        .into_iter()
                        .chain(pkg.test_go_files)
                        .chain(pkg.x_test_go_files)
                        .collect(),
                )
            })
            .collect();
        for pkg in host {
            if pkg.test_go_files.is_empty() && pkg.x_test_go_files.is_empty() {
                continue;
            }
            let files = pkg
                .go_files
                .iter()
                .chain(&pkg.test_go_files)
                .chain(&pkg.x_test_go_files)
                .map(|name| pkg.dir.join(name))
                .collect::<Vec<_>>();
            let source = inventory::source_inventory_with_loader(&files, |import| {
                embedded_suite_files(executor, context, options, &directory, import)
            })?;
            let tests = &source.tests;
            let top_level_tests = tests.keys().cloned().collect::<Vec<_>>();
            let target = relative(&directory, &pkg.dir)?;
            let pattern = relative(&options.checkout, &pkg.dir)?;
            let listed = list_tests(executor, context, options, &directory, &target, &pattern)?;
            if top_level_tests != listed {
                return Err(format!(
                    "enumeration mismatch: {pattern}: parsed {top_level_tests:?}, go test -list {listed:?}"
                ));
            }
            let units = tests
                .values()
                .flatten()
                .cloned()
                .collect::<BTreeSet<_>>()
                .into_iter()
                .collect::<Vec<_>>();
            let mut excluded = BTreeMap::new();
            // Reconcile each selected unit against its source files, never against a textual function search.
            for unit in &units {
                for file in source
                    .unit_files
                    .get(unit)
                    .ok_or_else(|| format!("source files unresolved for {unit}"))?
                {
                    let filename = file
                        .file_name()
                        .and_then(|n| n.to_str())
                        .ok_or("non-UTF8 Go file")?;
                    if !guest_files
                        .get(&pkg.import_path)
                        .is_some_and(|set| set.contains(filename))
                    {
                        excluded.insert(unit.clone(), filename.into());
                        break;
                    }
                }
            }
            packages.push(Package {
                import_path: pkg.import_path,
                pattern,
                module_directory: directory.display().to_string(),
                target,
                tests: units,
                units: source.tests.clone(),
                methods: source.methods.into_iter().map(SuiteMethod::from).collect(),
                registrations: source
                    .registrations
                    .into_iter()
                    .map(SuiteRegistration::from)
                    .collect(),
                list_count: listed.len(),
                parsed_count: top_level_tests.len(),
                top_level_tests,
                guest_excluded: excluded,
            });
        }
    }
    packages.sort_by(|a, b| a.pattern.cmp(&b.pattern));
    if packages.is_empty() {
        return Err("enumeration matched no test packages".into());
    }
    Ok(packages)
}

#[cfg(test)]
#[path = "enumerate_tests.rs"]
mod receiver_tests;
