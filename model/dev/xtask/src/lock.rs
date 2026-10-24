//! Portable non-development lock closures.
use std::{
    collections::{BTreeMap, BTreeSet},
    fs,
    path::{Path, PathBuf},
};

use serde::{Deserialize, Serialize};

use crate::{Error, Result, support};

#[derive(Deserialize)]
struct Metadata {
    packages: Vec<Package>,
    resolve: Resolve,
}

#[derive(Deserialize)]
struct Package {
    id: String,
    name: String,
    version: String,
    source: Option<String>,
}

#[derive(Deserialize)]
struct Resolve {
    nodes: Vec<Node>,
}

#[derive(Deserialize)]
struct Node {
    id: String,
    deps: Vec<Dependency>,
}

#[derive(Deserialize)]
struct Dependency {
    pkg: String,
    dep_kinds: Vec<DependencyKind>,
}

#[derive(Deserialize)]
struct DependencyKind {
    kind: Option<Kind>,
}

#[derive(Deserialize, PartialEq)]
#[serde(rename_all = "lowercase")]
enum Kind {
    Normal,
    Build,
    Dev,
}

#[derive(Deserialize)]
struct Lockfile {
    version: u32,
    #[serde(default)]
    package: Vec<LockPackage>,
}

#[derive(Deserialize)]
struct LockPackage {
    name: String,
    version: String,
    source: Option<String>,
    checksum: Option<String>,
}

#[derive(Clone, PartialEq, Eq, PartialOrd, Ord, Serialize)]
struct Coordinate<'a> {
    name: &'a str,
    version: &'a str,
    source: Option<&'a str>,
}

impl Package {
    fn coordinate(&self) -> Coordinate<'_> {
        Coordinate { name: &self.name, version: &self.version, source: self.source.as_deref() }
    }
}

#[derive(Serialize)]
struct Member<'a> {
    package: Coordinate<'a>,
    checksum: Option<&'a str>,
    dependencies: BTreeSet<Coordinate<'a>>,
}

#[derive(Serialize)]
struct Closure<'a> {
    schema: u32,
    lock: u32,
    packages: Vec<Member<'a>>,
}

#[derive(Serialize)]
struct LockMap {
    schema: u32,
    closures: BTreeMap<String, Option<String>>,
}

fn resolve_edges<'a>(
    metadata: &'a Metadata,
    packages: &BTreeMap<&str, &Package>,
) -> Result<BTreeMap<&'a str, BTreeSet<&'a str>>> {
    let mut edges = BTreeMap::new();
    for node in &metadata.resolve.nodes {
        if !packages.contains_key(node.id.as_str()) {
            return Err(Error::Invalid(format!("unknown resolve package: {}", node.id)));
        }
        let mut dependencies = BTreeSet::new();
        for dependency in &node.deps {
            if !packages.contains_key(dependency.pkg.as_str()) {
                return Err(Error::Invalid(format!("unknown dependency: {}", dependency.pkg)));
            }
            if dependency.dep_kinds.is_empty() {
                return Err(Error::Invalid(format!("dependency has no kinds: {}", dependency.pkg)));
            }
            if dependency.dep_kinds.iter().any(|kind| kind.kind != Some(Kind::Dev)) {
                dependencies.insert(dependency.pkg.as_str());
            }
        }
        if edges.insert(node.id.as_str(), dependencies).is_some() {
            return Err(Error::Invalid(format!("duplicate resolve node: {}", node.id)));
        }
    }
    if packages.keys().ne(edges.keys()) {
        return Err(Error::Invalid("package is missing a resolve node".into()));
    }
    Ok(edges)
}

fn closures(metadata: &Metadata, lock: &Lockfile) -> Result<BTreeMap<String, Option<String>>> {
    let mut packages = BTreeMap::new();
    let mut coordinates = BTreeSet::new();
    for package in &metadata.packages {
        if package.id.is_empty() || package.name.is_empty() || package.version.is_empty() {
            return Err(Error::Invalid("empty package identity".into()));
        }
        if !coordinates.insert(package.coordinate()) {
            return Err(Error::Invalid(format!(
                "duplicate package coordinate: {} {}",
                package.name, package.version
            )));
        }
        if packages.insert(package.id.as_str(), package).is_some() {
            return Err(Error::Invalid(format!("duplicate package ID: {}", package.id)));
        }
    }
    let mut entries = BTreeMap::new();
    for entry in &lock.package {
        let coordinate = Coordinate {
            name: &entry.name,
            version: &entry.version,
            source: entry.source.as_deref(),
        };
        if entries.insert(coordinate, entry.checksum.as_deref()).is_some() {
            return Err(Error::Invalid(format!(
                "duplicate lockfile entry: {} {}",
                entry.name, entry.version
            )));
        }
        if entry.checksum.as_deref().is_some_and(|checksum| !support::valid_hash(checksum)) {
            return Err(Error::Invalid(format!(
                "invalid lockfile checksum: {} {}",
                entry.name, entry.version
            )));
        }
    }
    let edges = resolve_edges(metadata, &packages)?;
    let mut result = BTreeMap::new();
    for root in edges.keys() {
        let mut seen = BTreeSet::new();
        let mut pending = vec![*root];
        while let Some(current) = pending.pop() {
            if seen.insert(current) {
                if edges[current].contains(root) {
                    return Err(Error::Invalid("cyclic non-development dependency graph".into()));
                }
                pending.extend(&edges[current]);
            }
        }
        let mut members = Vec::with_capacity(seen.len());
        for id in seen {
            let coordinate = packages[id].coordinate();
            let checksum = match entries.get(&coordinate) {
                Some(checksum) => *checksum,
                None if coordinate.source.is_none() => None,
                None => {
                    return Err(Error::Invalid(format!(
                        "no lockfile entry for {} {} from {}",
                        coordinate.name,
                        coordinate.version,
                        coordinate.source.unwrap_or_default()
                    )));
                }
            };
            members.push(Member {
                package: coordinate,
                checksum,
                dependencies: edges[id]
                    .iter()
                    .map(|dependency| packages[dependency].coordinate())
                    .collect(),
            });
        }
        members.sort_by(|a, b| a.package.cmp(&b.package));
        let digest = support::hash(&support::canonical(&Closure {
            schema: support::SCHEMA,
            lock: lock.version,
            packages: members,
        })?);
        let package = packages[root];
        let key = format!("{} {}", package.name, package.version);
        result.entry(key).and_modify(|identity| *identity = None).or_insert(Some(digest));
    }
    Ok(result)
}

/// Publish a content-addressed native schema-2 lock map from metadata JSON and TOML.
/// Normal and build edges include every platform; development-only edges are excluded.
/// Package IDs identify graph nodes only, so checkout paths never enter the content.
pub fn publish(metadata: &[u8], lockfile: &Path, directory: &Path) -> Result<PathBuf> {
    let metadata: Metadata = serde_json::from_slice(metadata)?;
    let lock: Lockfile = toml::from_str(&fs::read_to_string(lockfile)?)?;
    let bytes = support::canonical(&LockMap {
        schema: support::SCHEMA,
        closures: closures(&metadata, &lock)?,
    })?;
    let output = directory.join(format!("umpire-lock-{}.json", support::hash(&bytes)));
    support::publish(&output, &bytes)?;
    Ok(output)
}
