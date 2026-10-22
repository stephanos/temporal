//! Caller-owned scope policies over explicitly resolved catalog limits.
use std::collections::BTreeMap;

/// A declared limit with its caller-owned scope identity.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct Limit<S> {
    /// Stable configuration key.
    pub key: &'static str,
    /// Value when no resolved override is supplied.
    pub default: u64,
    /// Scope policy owned by the consuming system.
    pub scope: S,
}

/// Resolved values for one invocation; callers resolve their own scope precedence.
#[derive(Clone, Debug, Default)]
pub struct Limits {
    values: BTreeMap<&'static str, u64>,
}
impl Limits {
    /// Returns a resolved override or its declared default.
    pub fn get<S>(&self, limit: &Limit<S>) -> u64 {
        self.values.get(limit.key).copied().unwrap_or(limit.default)
    }
    /// Records a value already resolved for the caller's invocation scope.
    pub fn set<S>(&mut self, limit: &Limit<S>, value: u64) {
        self.values.insert(limit.key, value);
    }
}
