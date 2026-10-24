#![cfg_attr(
    not(test),
    allow(
        dead_code,
        reason = "Cycle entrypoint integration belongs to fn-242 successor tasks."
    )
)]

mod admission;
mod execution;
mod graph;

pub(crate) use admission::Policy;
use admission::can_start;
#[allow(
    unused_imports,
    reason = "Shared Linux sampler is consumed by cycle admission integration."
)]
pub(crate) use admission::{mem_available_gib, parse_mem_available};
#[allow(
    unused_imports,
    reason = "Public scheduler executor seam is consumed by cycle process integration."
)]
pub(crate) use execution::{Completion, Executor, RunResult, run};
#[allow(
    unused_imports,
    reason = "Public graph and original DAG decoder are consumed by cycle configuration integration."
)]
pub(crate) use graph::{Graph, parse_dag};

use super::model::{Flag, Isolation, Plan, Stage};
use std::collections::{BTreeMap, BTreeSet};

type LiveMemory<'a> = Option<&'a dyn Fn() -> Result<f64, String>>;

#[cfg(test)]
mod tests;
