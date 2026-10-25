//! One catalog system's build configuration and its generated files.

use std::{
    fs, io,
    path::{Path, PathBuf},
};

use heck::ToShoutySnakeCase;
use umpire_catalog_syntax::proto::Graph;

use crate::{coverage, services, wire_schema};

/// A protobuf service, its generated Rust root and its catalog declarations.
///
/// Generated names derive from `service` and `prefix`: for `pkg.KettleService` and `kettle`,
/// `KettleServiceMethod`, `KettleServiceMethodInfo`, `KETTLE_SERVICE_METHODS`,
/// `dispatch_kettle_annotations!`, `bind_kettle_action!`, `kettle_request_fields` and
/// `mutate_kettle_request`, `kettle_request_findings!` and `dispatch_kettle_request!`.
/// Request inspection expands in the catalog owner so its `Inspect` implementations and
/// supplemental hooks stay downstream of the generated API.
#[derive(Clone, Debug)]
pub struct System<'a> {
    /// Fully qualified protobuf service name.
    pub service: &'a str,
    /// Snake-case stem of the generated functions and macros.
    pub prefix: &'a str,
    /// Module of the generated protobuf packages, as `crate::…` or `::crate_name::…`; the
    /// absolute form also names them for other crates, such as a realization macro.
    pub rust_root: &'a str,
    /// Methods whose request closures are admitted for wire mutation.
    pub request_roots: &'a [&'a str],
    /// Directory whose catalog declarations cover the descriptor.
    pub catalog_root: PathBuf,
}

impl System<'_> {
    /// Check the catalogs against `descriptor` and write `services.rs` and `wire_schema.rs`
    /// into `out_dir`, emitting every input as a Cargo build input.
    #[allow(clippy::print_stdout, reason = "Cargo consumes build input directives on stdout")]
    pub fn generate(&self, descriptor: &Path, out_dir: &Path) -> io::Result<()> {
        println!("cargo:rerun-if-changed={}", descriptor.display());
        let bytes = fs::read(descriptor)?;
        let graph = Graph::new(&bytes, self.rust_root).map_err(io::Error::other)?;
        let (coverage, classifications) =
            coverage::check(&self.catalog_root, &graph, self.service, &self.method_enum())?;
        fs::write(
            out_dir.join("services.rs"),
            services::generate(&graph, &coverage, &classifications, self)
                .map_err(io::Error::other)?,
        )?;
        let local = Graph::new(&bytes, &self.local_root()).map_err(io::Error::other)?;
        fs::write(
            out_dir.join("wire_schema.rs"),
            wire_schema::generate(&local, self).map_err(io::Error::other)?,
        )
    }

    /// Generated method enum, `<Service>Method`.
    pub fn method_enum(&self) -> String {
        format!("{}Method", self.short_name())
    }

    /// Generated ordered method inventory constant, `<SERVICE>_METHODS`.
    pub fn methods_const(&self) -> String {
        format!("{}_METHODS", self.short_name().to_shouty_snake_case())
    }

    /// `rust_root` as written inside the generating crate; empty when the packages sit at the
    /// crate root, beside the included generated files.
    pub fn local_root(&self) -> String {
        let root = self.rebased("crate");
        if root == "crate" { String::new() } else { root }
    }

    /// `rust_root` as written inside the generating crate's exported macros.
    pub fn macro_root(&self) -> String {
        self.rebased("$crate")
    }

    /// `path` with its `rust_root` prefix replaced by `root`, when `path` lies under
    /// `rust_root` on a path-segment boundary.
    pub fn rebase(&self, path: &str, root: &str) -> Option<String> {
        let rest = path.strip_prefix(self.rust_root)?;
        (rest.is_empty() || rest.starts_with("::")).then(|| format!("{root}{rest}"))
    }

    fn short_name(&self) -> &str {
        self.service.rsplit('.').next().unwrap_or(self.service)
    }

    fn rebased(&self, root: &str) -> String {
        let path = self.rust_root.strip_prefix("::").unwrap_or(self.rust_root);
        path.split_once("::").map_or_else(|| root.to_owned(), |(_, rest)| format!("{root}::{rest}"))
    }
}
