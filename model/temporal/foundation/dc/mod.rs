//! Server limit declarations and the explicit catalog-factory context.

use std::collections::BTreeMap;

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

/// One server limit with its documented default and setting kind.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct Limit {
    /// Dynamic configuration key.
    pub key: &'static str,
    /// Server default when no override is supplied.
    pub default: u64,
    /// Setting kind; precedence is resolved by the server for this kind.
    pub scope: Scope,
}

/// Maximum encoded identifier length.
pub const MAX_ID_LENGTH: Limit =
    Limit { key: "limit.maxIDLength", default: 1000, scope: Scope::Global };
/// Maximum payload size before rejection.
pub const BLOB_SIZE_ERROR: Limit =
    Limit { key: "limit.blobSize.error", default: 2 * 1024 * 1024, scope: Scope::Namespace };

/// Already-resolved limits for a catalog invocation.
#[derive(Clone, Debug, Default)]
pub struct Limits {
    values: BTreeMap<&'static str, u64>,
}

impl Limits {
    /// Returns an override or the declared server default.
    pub fn get(&self, limit: &Limit) -> u64 {
        self.values.get(limit.key).copied().unwrap_or(limit.default)
    }
    /// Supplies an explicitly resolved value for a declaration.
    pub fn set(&mut self, limit: &Limit, value: u64) {
        self.values.insert(limit.key, value);
    }
}
