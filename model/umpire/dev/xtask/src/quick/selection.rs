use super::{Options, Selection, WorkspaceSelection};
use crate::{Error, Result};
use serde::Deserialize;
use std::{
    collections::{BTreeMap, BTreeSet},
    path::{Path, PathBuf},
};

#[derive(Deserialize)]
struct Metadata {
    packages: Vec<Package>,
    workspace_members: Vec<String>,
    resolve: Option<Resolve>,
}
#[derive(Deserialize)]
struct Package {
    id: String,
    name: String,
    manifest_path: PathBuf,
    source: Option<String>,
    #[serde(default)]
    targets: Vec<Target>,
}
#[derive(Deserialize)]
struct Target {
    kind: Vec<String>,
    doctest: bool,
}
#[derive(Deserialize)]
struct Resolve {
    nodes: Vec<Node>,
}
#[derive(Deserialize)]
struct Node {
    id: String,
    dependencies: Vec<String>,
}

#[derive(Default)]
pub(super) struct Graph {
    packages: BTreeMap<PathBuf, String>,
    members: BTreeMap<PathBuf, BTreeSet<PathBuf>>,
    reverse: BTreeMap<PathBuf, BTreeSet<PathBuf>>,
    doctests: BTreeSet<PathBuf>,
}

impl Graph {
    pub(super) fn add(&mut self, root: &Path, bytes: &[u8]) -> Result<()> {
        let metadata: Metadata = serde_json::from_slice(bytes)
            .map_err(|error| Error::Invalid(format!("invalid Cargo metadata: {error}")))?;
        let packages: BTreeMap<_, _> = metadata
            .packages
            .into_iter()
            .filter(|package| package.source.is_none())
            .map(|package| (package.id.clone(), package))
            .collect();
        let members = metadata
            .workspace_members
            .iter()
            .map(|id| {
                packages
                    .get(id)
                    .map(|package| package.manifest_path.clone())
                    .ok_or_else(|| Error::Invalid(format!("missing workspace package {id}")))
            })
            .collect::<Result<_>>()?;
        self.members.insert(root.to_owned(), members);
        for package in packages.values() {
            self.packages.insert(package.manifest_path.clone(), package.name.clone());
            if package.targets.iter().any(|target| {
                target.doctest
                    && target.kind.iter().any(|kind| kind == "lib" || kind == "proc-macro")
            }) {
                self.doctests.insert(package.manifest_path.clone());
            }
        }
        let resolve = metadata
            .resolve
            .ok_or_else(|| Error::Invalid("Cargo metadata omitted dependency graph".into()))?;
        for node in resolve.nodes {
            let Some(package) = packages.get(&node.id) else { continue };
            for dependency in node.dependencies {
                if let Some(dependency) = packages.get(&dependency) {
                    self.reverse
                        .entry(dependency.manifest_path.clone())
                        .or_default()
                        .insert(package.manifest_path.clone());
                }
            }
        }
        Ok(())
    }

    pub(super) fn compiler_only(&self, root: &Path, package: &str) -> bool {
        [
            "author/umpire",
            "ir/derive",
            "reference/kettle/guest",
            "protocol/runtime/guests/stock",
            "protocol/runtime/guests/control",
            "protocol/runtime/guests/caught-panic",
            "protocol/runtime/guests/overflow",
        ]
        .iter()
        .any(|directory| {
            let manifest = root.join(directory).join("Cargo.toml");
            self.members.get(root).is_some_and(|members| members.contains(&manifest))
                && self.packages.get(&manifest).is_some_and(|name| name == package)
        })
    }

    fn members_named(&self, name: &str) -> BTreeSet<PathBuf> {
        self.members
            .values()
            .flatten()
            .filter(|manifest| self.packages.get(*manifest).is_some_and(|package| package == name))
            .cloned()
            .collect()
    }

    fn expand(&self, selected: &mut BTreeSet<PathBuf>) {
        let mut pending: Vec<_> = selected.iter().cloned().collect();
        while let Some(package) = pending.pop() {
            if let Some(dependents) = self.reverse.get(&package) {
                for dependent in dependents {
                    if selected.insert(dependent.clone()) {
                        pending.push(dependent.clone());
                    }
                }
            }
        }
    }

    fn known_non_model_input(&self, path: &Path) -> bool {
        let Some(root) = self
            .members
            .keys()
            .flat_map(|workspace| workspace.ancestors())
            .filter(|ancestor| path.starts_with(ancestor))
            .max_by_key(|ancestor| ancestor.components().count())
        else {
            return false;
        };
        let Ok(relative) = path.strip_prefix(root) else { return false };
        relative.components().next().is_some_and(|part| {
            [".flow", ".plans", ".turbo"].iter().any(|name| part.as_os_str() == *name)
                || (!self.members.keys().any(|workspace| path.starts_with(workspace))
                    && path.extension().is_some_and(|extension| extension == "md")
                    && (part.as_os_str() == "docs" || relative.components().count() == 1))
        })
    }

    fn seed(
        &self,
        options: &Options,
        selected: &mut BTreeSet<PathBuf>,
        reasons: &mut BTreeSet<String>,
    ) -> Result<()> {
        for name in options
            .always
            .iter()
            .chain(options.targets.keys())
            .chain(options.filters.keys())
            .chain(options.compiler_only.iter())
        {
            let packages = self.members_named(name);
            if packages.is_empty() {
                return Err(Error::Invalid(format!(
                    "quick package {name} is not a member of an explicit workspace"
                )));
            }
            if options.always.contains(name) {
                selected.extend(packages);
                reasons.insert(format!(
                    "always: {name} (native/interpreter/backend fixture execution)"
                ));
            }
        }
        for (_, name) in &options.mappings {
            if self.members_named(name).is_empty() {
                return Err(Error::Invalid(format!(
                    "quick mapping package {name} is not a workspace member"
                )));
            }
        }
        Ok(())
    }

    fn workspace_selection(&self, selected: &BTreeSet<PathBuf>) -> Vec<WorkspaceSelection> {
        self.members
            .iter()
            .map(|(workspace, members)| WorkspaceSelection {
                workspace: workspace.clone(),
                doctests: members
                    .intersection(selected)
                    .filter(|manifest| self.doctests.contains(*manifest))
                    .map(|manifest| self.packages[manifest].clone())
                    .collect(),
                packages: members
                    .intersection(selected)
                    .map(|manifest| self.packages[manifest].clone())
                    .collect::<BTreeSet<_>>()
                    .into_iter()
                    .collect(),
            })
            .collect()
    }

    pub(super) fn select(&self, options: &Options, changed: Vec<PathBuf>) -> Result<Selection> {
        let mut selected = BTreeSet::new();
        let mut test_consumers = BTreeSet::new();
        let mut reasons = BTreeSet::new();
        self.seed(options, &mut selected, &mut reasons)?;
        for path in &changed {
            for root in self.members.keys() {
                if path == &root.join("dev/fixtures/fixtures/sample_kettle/model.rs") {
                    for directory in
                        ["reference/kettle", "author/suite", "backends/tla", "backends/apalache"]
                    {
                        let consumer = root.join(directory).join("Cargo.toml");
                        if self.members[root].contains(&consumer) {
                            if directory == "reference/kettle" {
                                selected.insert(consumer);
                            } else {
                                test_consumers.insert(consumer);
                            }
                            reasons.insert(format!("shared Kettle source: {}", path.display()));
                        }
                    }
                }
            }
            let workspace = self
                .members
                .keys()
                .filter(|root| path.starts_with(root))
                .max_by_key(|root| root.components().count());
            let global = global_input(path, workspace);
            if !global {
                let mappings: Vec<_> = options
                    .mappings
                    .iter()
                    .filter(|(directory, _)| path.starts_with(directory))
                    .collect();
                if !mappings.is_empty() {
                    for (_, name) in mappings {
                        selected.extend(self.members_named(name));
                    }
                    reasons.insert(format!("mapped source: {}", path.display()));
                    continue;
                }
                let owner = self
                    .packages
                    .keys()
                    .filter(|manifest| {
                        manifest.parent().is_some_and(|directory| path.starts_with(directory))
                    })
                    .max_by_key(|manifest| manifest.components().count());
                if let Some(owner) = owner {
                    selected.insert(owner.clone());
                    reasons.insert(format!("package source: {}", path.display()));
                    continue;
                }
            }
            if !global && self.known_non_model_input(path) {
                reasons.insert(format!("ignored known non-model input: {}", path.display()));
            } else if let Some(root) = workspace {
                selected.extend(self.members[root].iter().cloned());
                reasons.insert(format!(
                    "whole workspace {}: {} input {}",
                    root.display(),
                    if global { "global" } else { "unmapped" },
                    path.display()
                ));
            } else {
                selected.extend(self.members.values().flatten().cloned());
                reasons.insert(format!(
                    "all workspaces: {} input {}",
                    if global { "global" } else { "unmapped" },
                    path.display()
                ));
            }
        }
        self.expand(&mut selected);
        selected.extend(test_consumers);
        if changed.is_empty() {
            reasons.insert("unchanged invocation: explicit always packages still execute".into());
        }
        let workspaces = self.workspace_selection(&selected);
        Ok(Selection {
            base: options
                .base
                .as_ref()
                .map_or_else(String::new, |base| base.to_string_lossy().into_owned()),
            changed,
            reasons,
            workspaces,
        })
    }
}

pub(super) fn global_input(path: &Path, workspace: Option<&PathBuf>) -> bool {
    path.file_name().is_some_and(|name| {
        [
            "Makefile",
            "Cargo.toml",
            "Cargo.lock",
            "build.rs",
            "rust-toolchain",
            "rust-toolchain.toml",
            "mise.toml",
            "mise.lock",
            ".mise.toml",
            "rustfmt.toml",
            ".rustfmt.toml",
            "clippy.toml",
            ".clippy.toml",
        ]
        .iter()
        .any(|candidate| name == *candidate)
    }) || path.components().any(|part| part.as_os_str() == ".cargo")
        || workspace.is_some_and(|root| {
            ["scripts", "dev/xtask", "dev/guest-driver"]
                .iter()
                .any(|directory| path.starts_with(root.join(directory)))
        })
}
