//! Static and per-kettle validation limits resolved before catalog admission.

/// The server scope in which a limit is resolved.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Scope {
    /// Shared by every reference kettle.
    Global,
    /// Resolved for the current kettle identifier.
    Kettle,
}

/// A declared reference-server limit.
pub type Limit = umpire_catalog::limits::Limit<Scope>;

/// Maximum UTF-8 bytes in a kettle identifier.
pub const MAX_KETTLE_ID_BYTES: Limit =
    Limit { key: "kettle.maxIDBytes", default: 32, scope: Scope::Global };
/// Maximum number of tags, resolved in the current kettle's scope.
pub const MAX_TAGS: Limit = Limit { key: "kettle.maxTags", default: 4, scope: Scope::Kettle };

/// Resolved limits for one request's kettle identity.
#[derive(Clone, Debug, Default)]
pub struct Limits {
    values: umpire_catalog::limits::Limits,
}

impl Limits {
    /// Resolves the reference server's per-kettle override.
    pub fn for_kettle(kettle_id: &str) -> Self {
        let mut limits = Self::default();
        if kettle_id == "compact" {
            limits.values.set(&MAX_TAGS, 2);
        }
        limits
    }

    /// Returns the static default or the current kettle's scoped value.
    pub fn get(&self, limit: &Limit) -> u64 {
        self.values.get(limit)
    }
}
