use super::{path, require};
use crate::cycle::{
    custody,
    model::{Adapter, Mode, PlanConfig},
    plan::{
        self,
        leaf::{Binding, LeafBindings, LeafPlan},
    },
};
use proc_macro2::{Delimiter, TokenStream, TokenTree};
use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::{
    collections::{BTreeMap, BTreeSet},
    fs,
    path::{Path, PathBuf},
    str::FromStr,
};

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Tool {
    pub path: PathBuf,
    pub sha256: String,
}
#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct StaticInputs {
    pub repo: PathBuf,
    pub driver_config: PathBuf,
    pub tools: BTreeMap<String, Tool>,
    pub caches: BTreeMap<String, PathBuf>,
    pub lint_base: String,
}
#[derive(Clone, Debug, Deserialize, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct StaticPlan {
    pub inputs: StaticInputs,
    pub stages: BTreeMap<String, LeafPlan>,
    pub files: BTreeMap<String, String>,
    pub ignored: Value,
    pub workflows: Vec<PathBuf>,
}
pub(crate) fn regular(path: &Path) -> Result<Vec<u8>, String> {
    custody::read_file(path, 512 << 20)
}
fn canonical_dir(p: &Path) -> Result<(), String> {
    custody::clean_absolute(p)?;
    require(
        fs::canonicalize(p).map_err(|e| e.to_string())? == p && p.is_dir(),
        "cache must be an assigned canonical external directory",
    )
}
pub(crate) fn workflow_pair(repo: &Path) -> Result<Vec<PathBuf>, String> {
    let root = repo.join(".github/workflows");
    let expected = vec![
        root.join("gomad.yml"),
        root.join("gomad-behavioral-parity.yml"),
    ];
    let mut observed = BTreeSet::new();
    for item in fs::read_dir(&root).map_err(|e| e.to_string())? {
        let p = item.map_err(|e| e.to_string())?.path();
        if !matches!(p.extension().and_then(|v| v.to_str()), Some("yml" | "yaml")) {
            continue;
        }
        let bytes = regular(&p)?;
        let text = std::str::from_utf8(&bytes).map_err(|e| e.to_string())?;
        let routed = text.match_indices("tools/gomad").any(|(at, _)| {
            let boundary = |c: char| !c.is_alphanumeric() && !matches!(c, '_' | '.' | '-');
            text[..at].chars().next_back().is_none_or(boundary)
                && text[at + 11..].chars().next().is_none_or(boundary)
        });
        if p.file_name()
            .and_then(|v| v.to_str())
            .is_some_and(|v| v.starts_with("gomad"))
            || routed
        {
            observed.insert(p.clone());
        }
        if expected.contains(&p) {
            require(routed, "changed Gomad workflow module routing")?;
        }
    }
    require(
        observed == expected.iter().cloned().collect(),
        "missing/ambiguous current Gomad workflow inventory",
    )?;
    Ok(expected)
}
pub(crate) fn ignored_attributes(text: &str) -> Result<Vec<(String, String)>, String> {
    require(text.len() <= 16 << 20, "Rust source byte bound exceeded")?;
    let stream = TokenStream::from_str(text)
        .map_err(|e| format!("invalid Rust source token stream: {e}"))?;
    let mut rows = vec![];
    scan_tokens(stream, &mut rows)?;
    Ok(rows)
}
fn scan_tokens(stream: TokenStream, rows: &mut Vec<(String, String)>) -> Result<(), String> {
    let tokens: Vec<_> = stream.into_iter().collect();
    for (at, token) in tokens.iter().enumerate() {
        if let TokenTree::Group(group) = token {
            if group.delimiter() == Delimiter::Bracket
                && at > 0
                && matches!(&tokens[at-1],TokenTree::Punct(p) if p.as_char() == '#')
            {
                let attr: Vec<_> = group.stream().into_iter().collect();
                if matches!(attr.first(),Some(TokenTree::Ident(i)) if i == "cfg_attr") {
                    require(
                        !attr.iter().any(|t| {
                            if let TokenTree::Group(g) = t {
                                g.stream()
                                    .into_iter()
                                    .any(|v| matches!(v,TokenTree::Ident(i) if i=="ignore"))
                            } else {
                                false
                            }
                        }),
                        "conditional ignored disposition is unsupported",
                    )?;
                }
                if matches!(attr.first(),Some(TokenTree::Ident(i)) if i == "ignore") {
                    require(
                        attr.len() == 3
                            && matches!(&attr[1],TokenTree::Punct(p) if p.as_char() == '='),
                        "ignored test requires an explicit registered reason",
                    )?;
                    let TokenTree::Literal(reason) = &attr[2] else {
                        return Err("ignored test requires literal reason".into());
                    };
                    let reason: String = serde_json::from_str(&reason.to_string())
                        .map_err(|_| "ignored test requires string reason")?;
                    let mut tail = tokens[at + 1..].iter();
                    let mut name = None;
                    while let Some(next) = tail.next() {
                        match next {
                            TokenTree::Ident(i) if i == "fn" => {
                                if let Some(TokenTree::Ident(i)) = tail.next() {
                                    name = Some(i.to_string());
                                }
                                break;
                            }
                            TokenTree::Group(g) if g.delimiter() == Delimiter::Brace => break,
                            TokenTree::Punct(p) if p.as_char() == ';' => break,
                            _ => (),
                        }
                    }
                    rows.push((name.ok_or("ignored attribute has no function")?, reason));
                }
            }
            scan_tokens(group.stream(), rows)?;
        }
    }
    Ok(())
}
fn rust_files(directory: &Path, output: &mut Vec<PathBuf>) -> Result<(), String> {
    for item in fs::read_dir(directory).map_err(|e| e.to_string())? {
        let item = item.map_err(|e| e.to_string())?;
        let kind = item.file_type().map_err(|e| e.to_string())?;
        require(!kind.is_symlink(), "linked source inventory entry")?;
        if kind.is_dir() {
            rust_files(&item.path(), output)?;
        } else if item.path().extension().is_some_and(|ext| ext == "rs") {
            output.push(item.path());
        }
    }
    Ok(())
}
fn validate_broad_routes(row: &Value, source: &str, name: &str) -> Result<(), String> {
    if let Some(routes) = row.get("broad_routes") {
        let routes = routes
            .as_array()
            .filter(|rows| !rows.is_empty())
            .ok_or("ignored broad test lacks a closed actual route")?;
        let mut b = LeafBindings::default();
        b.paths.insert(Binding::Root, "/cycle".into());
        b.paths.insert(Binding::Repo, "/repo".into());
        let config = PlanConfig {
            binary: "/gomad".into(),
            config_path: "/config".into(),
            config_sha256: "0".repeat(64),
        };
        let mut actual = vec![];
        for stage in plan::derive(&config, Mode::Parallel)?.stages {
            if !matches!(
                stage.contract.label.as_str(),
                "cluster-provider" | "cluster-functional" | "retained-cluster"
            ) {
                continue;
            }
            for leaf in stage.leaf_commands(&b)?.commands {
                if !leaf.argv.iter().any(|arg| arg == "--ignored") {
                    continue;
                }
                let Some(selector) = leaf.argv.iter().skip(1).find(|arg| !arg.starts_with('-'))
                else {
                    continue;
                };
                let host = leaf.argv[0]
                    .rsplit('/')
                    .next()
                    .is_some_and(|role| role.starts_with("host-"));
                if host != (row["crate"] == "gomad") {
                    continue;
                }
                let selected = if leaf.argv.iter().any(|arg| arg == "--exact") {
                    selector == name || selector.ends_with(&format!("::{name}"))
                } else {
                    source.ends_with(&format!("/{}.rs", selector.replace("::", "/")))
                };
                if selected {
                    actual.push(serde_json::json!({"stage":stage.contract.label,"invocation":leaf.name,"selector":selector,"features":if host {vec!["determinism-fixture"]} else {vec![]}}));
                }
            }
        }
        require(
            routes == &actual && !actual.is_empty(),
            "ignored broad test changed its fixed executable, features or final selectors",
        )?;
    }
    Ok(())
}
pub(crate) fn ignored_inventory(repo: &Path) -> Result<Value, String> {
    let root = repo.join("tools/gomad/rust");
    let declared = custody::strict_json(&regular(&root.join("driver/static-ignored.json"))?)?;
    require(
        declared["schema"] == "gomad.driver-static-ignored/v1",
        "unknown ignored schema",
    )?;
    let declared_rows = declared["tests"]
        .as_array()
        .ok_or("missing ignored disposition inventory")?;
    let mut expected = BTreeSet::new();
    let final_ids = custody::strict_json(&regular(&root.join("driver/required-final.json"))?)?;
    let ids = final_ids["subgates"]
        .as_array()
        .ok_or("missing final obligations")?;
    for row in declared_rows {
        let source = row["source"].as_str().ok_or("missing ignored source")?;
        let name = row["name"].as_str().ok_or("missing ignored name")?;
        let reason = row["reason"]
            .as_str()
            .filter(|v| !v.is_empty())
            .ok_or("missing ignored reason")?;
        require(
            expected.insert((source.to_owned(), name.to_owned(), reason.to_owned())),
            "duplicate ignored test disposition",
        )?;
        require(
            row["final_id"].is_null() || ids.contains(&row["final_id"]),
            "unknown receiving obligation",
        )?;
        require(
            source.starts_with(&format!(
                "{}/",
                row["crate"].as_str().ok_or("missing ignored crate")?
            )),
            "ignored crate/source mismatch",
        )?;
        validate_broad_routes(row, source, name)?;
    }
    let mut observed = BTreeSet::new();
    let mut count = 0;
    for package in ["gomad", "wasmhost"] {
        for folder in ["src", "tests"] {
            let directory = root.join(package).join(folder);
            if !directory.exists() {
                continue;
            }
            let mut sources = vec![];
            rust_files(&directory, &mut sources)?;
            sources.sort();
            for source in sources {
                let bytes = regular(&source)?;
                let text = std::str::from_utf8(&bytes).map_err(|e| e.to_string())?;
                for (name, reason) in ignored_attributes(text)? {
                    count += 1;
                    observed.insert((
                        source
                            .strip_prefix(&root)
                            .map_err(|e| e.to_string())?
                            .to_str()
                            .ok_or("non UTF-8 source")?
                            .to_owned(),
                        name,
                        reason,
                    ));
                }
            }
        }
    }
    require(
        count == declared_rows.len() && observed == expected,
        "unknown/changed/missing ignored test disposition",
    )?;
    Ok(declared)
}
fn tool_files(inputs: &StaticInputs, b: &LeafBindings) -> Result<BTreeMap<String, String>, String> {
    let names = BTreeSet::from([
        "python",
        "go",
        "cargo",
        "rustc",
        "rustfmt",
        "make",
        "golangci",
        "errortype",
        "true",
        "rustdoc",
        "cargo_clippy",
        "clippy_driver",
        "cargo_fmt",
    ]);
    require(
        inputs
            .tools
            .keys()
            .map(String::as_str)
            .collect::<BTreeSet<_>>()
            == names,
        "missing/unexpected tools",
    )?;
    require(
        inputs
            .caches
            .keys()
            .map(String::as_str)
            .collect::<BTreeSet<_>>()
            == BTreeSet::from(["cargo_target", "cargo_home", "go_cache", "go_mod_cache"]),
        "missing/unexpected caches",
    )?;
    let mut files = BTreeMap::new();
    for (name, tool) in &inputs.tools {
        require(
            custody::valid_digest(&tool.sha256)
                && custody::digest(&regular(&tool.path)?) == tool.sha256,
            &format!("changed tool: {name}"),
        )?;
        files.insert(
            tool.path.to_str().ok_or("non UTF-8 tool")?.to_owned(),
            tool.sha256.clone(),
        );
    }
    let parent = inputs.tools["cargo"].path.parent();
    require(
        [
            "rustc",
            "rustdoc",
            "rustfmt",
            "cargo_clippy",
            "clippy_driver",
            "cargo_fmt",
        ]
        .iter()
        .all(|key| inputs.tools[*key].path.parent() == parent),
        "Rust tools must share pinned directory",
    )?;
    for (key, name) in [
        ("cargo_clippy", "cargo-clippy"),
        ("clippy_driver", "clippy-driver"),
        ("cargo_fmt", "cargo-fmt"),
    ] {
        require(
            inputs.tools[key]
                .path
                .file_name()
                .is_some_and(|v| v == name),
            "Rust sibling discovery requires original tool names",
        )?;
    }
    for (key, binding) in [
        ("python", Binding::Python),
        ("go", Binding::Go),
        ("cargo", Binding::Cargo),
        ("rustc", Binding::Rustc),
        ("cargo_clippy", Binding::Clippy),
        ("make", Binding::Make),
        ("golangci", Binding::Golangci),
        ("errortype", Binding::Errortype),
        ("true", Binding::True),
    ] {
        require(
            inputs.tools[key].path == Path::new(path(b, binding)?),
            "static/leaf tool binding differs",
        )?;
    }
    Ok(files)
}
pub(crate) fn static_plan(inputs: &StaticInputs, b: &LeafBindings) -> Result<StaticPlan, String> {
    canonical_dir(&inputs.repo)?;
    require(
        inputs.repo == Path::new(path(b, Binding::Repo)?)
            && inputs.driver_config == Path::new(path(b, Binding::Config)?),
        "static checkout/config binding differs",
    )?;
    custody::clean_absolute(&inputs.driver_config)?;
    require(
        inputs.lint_base.len() == 40
            && inputs
                .lint_base
                .bytes()
                .all(|c| c.is_ascii_digit() || (b'a'..=b'f').contains(&c))
            && inputs.lint_base == b.lint_base,
        "lint base must be an exact bound commit",
    )?;
    let mut files = tool_files(inputs, b)?;
    for cache in inputs.caches.values() {
        canonical_dir(cache)?;
        require(
            !cache.starts_with(&inputs.repo),
            "cache must stay outside frozen source",
        )?;
    }
    require(
        inputs.caches["cargo_home"] == Path::new(path(b, Binding::CargoHome)?),
        "static/leaf Cargo home differs",
    )?;
    let ignored = ignored_inventory(&inputs.repo)?;
    let workflows = workflow_pair(&inputs.repo)?;
    for p in workflows.iter().cloned().chain([
        inputs
            .repo
            .join("tools/gomad/rust/driver/static-ignored.json"),
        inputs
            .repo
            .join("tools/gomad/rust/driver/required-final.json"),
    ]) {
        files.insert(
            p.to_str().ok_or("non UTF-8 static input")?.to_owned(),
            custody::digest(&regular(&p)?),
        );
    }
    let config = PlanConfig {
        binary: path(b, Binding::Binary)?.into(),
        config_path: inputs.driver_config.clone(),
        config_sha256: "0".repeat(64),
    };
    let stages = plan::derive(&config, Mode::Parallel)?
        .stages
        .into_iter()
        .filter(|stage| stage.contract.adapter == Adapter::CheckedCommand)
        .map(|stage| Ok((stage.contract.label.clone(), stage.leaf_commands(b)?)))
        .collect::<Result<BTreeMap<_, _>, String>>()?;
    require(stages.len() == 24, "changed static stage ownership")?;
    Ok(StaticPlan {
        inputs: inputs.clone(),
        stages,
        files,
        ignored,
        workflows,
    })
}
pub(crate) fn recipe(plan: &StaticPlan, label: &str, b: &LeafBindings) -> Result<LeafPlan, String> {
    let config = PlanConfig {
        binary: path(b, Binding::Binary)?.into(),
        config_path: plan.inputs.driver_config.clone(),
        config_sha256: "0".repeat(64),
    };
    let stage = plan::derive(&config, Mode::Parallel)?
        .stages
        .into_iter()
        .find(|stage| {
            stage.contract.adapter == Adapter::CheckedCommand && stage.contract.label == label
        })
        .ok_or("only fixed static stages are owned")?;
    let expected = stage.leaf_commands(b)?;
    require(
        plan.stages.get(label) == Some(&expected),
        "changed immutable static recipe",
    )?;
    Ok(expected)
}
pub(crate) fn prepare_plan(
    inputs: &StaticInputs,
    b: &LeafBindings,
    output: &Path,
) -> Result<StaticPlan, String> {
    let plan = static_plan(inputs, b)?;
    let bytes = crate::cycle::preparation::inputs::canonical(
        &serde_json::to_value(&plan).map_err(|e| e.to_string())?,
    )?;
    if output.exists() {
        let old = custody::strict_json(&regular(output)?)?;
        require(
            old == serde_json::to_value(&plan).map_err(|e| e.to_string())?,
            "provisioned static plan differs from current fixed recipes or actual tool inputs",
        )?;
    } else {
        publish(output, &bytes)?;
    }
    Ok(plan)
}
pub(crate) fn tree_hashes(directory: &Path) -> Result<BTreeMap<String, String>, String> {
    fn walk(root: &Path, dir: &Path, result: &mut BTreeMap<String, String>) -> Result<(), String> {
        for item in fs::read_dir(dir).map_err(|e| e.to_string())? {
            let item = item.map_err(|e| e.to_string())?;
            let p = item.path();
            let kind = item.file_type().map_err(|e| e.to_string())?;
            require(!kind.is_symlink(), "linked corpus entry")?;
            if kind.is_dir() {
                walk(root, &p, result)?;
            } else {
                result.insert(
                    p.strip_prefix(root)
                        .map_err(|e| e.to_string())?
                        .to_str()
                        .ok_or("non UTF-8 corpus path")?
                        .into(),
                    custody::digest(&regular(&p)?),
                );
            }
        }
        Ok(())
    }
    canonical_dir(directory)?;
    let mut result = BTreeMap::new();
    walk(directory, directory, &mut result)?;
    Ok(result)
}

pub(crate) fn plan_config(
    plan_path: &Path,
    label: &str,
    b: &LeafBindings,
    config_path: &Path,
) -> Result<(StaticPlan, Value), String> {
    let raw = regular(plan_path)?;
    let plan: StaticPlan =
        serde_json::from_value(custody::strict_json(&raw)?).map_err(|e| e.to_string())?;
    require(
        plan == static_plan(&plan.inputs, b)?,
        "changed immutable static plan",
    )?;
    recipe(&plan, label, b)?;
    let config = custody::strict_json(&regular(config_path)?)?;
    require(
        config_path == plan.inputs.driver_config
            && config["repo"] == plan.inputs.repo.to_str().ok_or("non UTF-8 repository")?,
        "wrong frozen candidate/config",
    )?;
    let files: BTreeMap<String, String> =
        serde_json::from_value(config["files"].clone()).map_err(|e| e.to_string())?;
    require(
        files.get(plan_path.to_str().ok_or("non UTF-8 plan path")?) == Some(&custody::digest(&raw)),
        "unsealed static plan",
    )?;
    for (name, hash) in &plan.files {
        require(files.get(name) == Some(hash), "unsealed static input")?;
    }
    custody::check_files(&files)?;
    Ok((plan, config))
}

pub(super) fn publish(output: &Path, bytes: &[u8]) -> Result<(), String> {
    use rustix::fs::{Mode, OFlags, open, openat};
    use std::{fs::File, io::Write, path::Component};
    custody::clean_absolute(output)?;
    let flags = OFlags::RDONLY | OFlags::DIRECTORY | OFlags::NOFOLLOW | OFlags::CLOEXEC;
    let mut parent = File::from(open("/", flags, Mode::empty()).map_err(|e| e.to_string())?);
    let parts: Vec<_> = output
        .components()
        .filter_map(|part| {
            if let Component::Normal(name) = part {
                Some(name)
            } else {
                None
            }
        })
        .collect();
    let (name, parents) = parts
        .split_last()
        .ok_or("missing composition output name")?;
    for part in parents {
        parent =
            File::from(openat(&parent, *part, flags, Mode::empty()).map_err(|e| e.to_string())?);
    }
    let mut file = File::from(
        openat(
            &parent,
            *name,
            OFlags::WRONLY | OFlags::CREATE | OFlags::EXCL | OFlags::NOFOLLOW | OFlags::CLOEXEC,
            Mode::from_raw_mode(0o600),
        )
        .map_err(|e| format!("composition output must be fresh: {e}"))?,
    );
    file.write_all(bytes).map_err(|e| e.to_string())?;
    file.sync_all().map_err(|e| e.to_string())?;
    require(regular(output)? == bytes, "published composition changed")
}
