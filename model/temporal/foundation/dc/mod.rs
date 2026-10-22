//! Server limit declarations and the explicit catalog-factory context.

/// The setting kind controlling server-side precedence for one key.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Scope {
    /// Cluster-wide setting.
    Global,
    /// Namespace name setting.
    Namespace,
    /// Namespace identifier setting.
    NamespaceID,
    /// Task queue setting.
    TaskQueue,
    /// Shard identifier setting.
    ShardID,
    /// Task type setting.
    TaskType,
    /// Destination setting.
    Destination,
}

/// A server limit retaining the extension's setting scope.
pub type Limit = umpire_catalog::limits::Limit<Scope>;

/// Maximum encoded identifier length.
pub const MAX_ID_LENGTH: Limit =
    Limit { key: "limit.maxIDLength", default: 1000, scope: Scope::Global };
/// Maximum payload size before rejection.
pub const BLOB_SIZE_ERROR: Limit =
    Limit { key: "limit.blobSize.error", default: 2 * 1024 * 1024, scope: Scope::Namespace };

/// Resolved limits retain the extension's local inspection context identity.
#[derive(Clone, Debug, Default)]
pub struct Limits {
    values: umpire_catalog::limits::Limits,
}

impl Limits {
    /// Returns an override or the declared server default.
    pub fn get(&self, limit: &Limit) -> u64 {
        self.values.get(limit)
    }
    /// Supplies an explicitly resolved value for a declaration.
    pub fn set(&mut self, limit: &Limit, value: u64) {
        self.values.set(limit, value);
    }
}
