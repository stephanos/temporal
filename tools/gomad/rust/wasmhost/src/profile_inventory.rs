//! Closed attribution names shared by the collector and independent readers.
pub(crate) const OWNER_NAMES: [&str; 18] = [
    "clock",
    "scheduler",
    "scheduler_handoff",
    "namespace",
    "volume",
    "network",
    "world",
    "storage_transport",
    "cache",
    "transcript",
    "evidence",
    "encoding",
    "remote",
    "inputs",
    "inspection",
    "lifecycle",
    "instrumentation",
    "handoff_wait",
];
#[must_use]
pub fn operation_names() -> Vec<String> {
    crate::abi::Operation::ALL
        .iter()
        .map(|op| format!("{}.{}", op.module(), op.name()))
        .collect()
}
#[must_use]
pub fn owner_names() -> &'static [&'static str] {
    &OWNER_NAMES
}
