use crate::{Type, expr::Expr};
use std::collections::BTreeMap;

/// An admitted Model. Its typed expression tree has no public stability promise.
#[derive(Clone, Debug)]
pub struct Lifted {
    pub(crate) file: String,
    pub(crate) name: String,
    pub(crate) state_name: String,
    pub(crate) outcome_type: Type,
    pub(crate) world_path: String,
    pub(crate) fields: Vec<FieldDef>,
    pub(crate) constants: BTreeMap<String, Expr>,
    pub(crate) actions: Vec<ActionDef>,
    pub(crate) invariants: Vec<ClaimDef>,
    pub(crate) postconditions: Vec<PostDef>,
    pub(crate) progress: Vec<ProgressDef>,
    pub(crate) assumptions: Vec<AssumptionDef>,
    pub(crate) negative: Vec<ClaimDef>,
}
#[derive(Clone, Debug)]
pub(crate) struct FieldDef {
    pub name: String,
    pub ty: Type,
    pub start: Start,
}
#[derive(Clone, Debug)]
pub(crate) enum Start {
    Value(Expr),
    Range(Expr, Expr),
    ExclusiveRange(Expr, Expr),
}
#[derive(Clone, Debug)]
pub(crate) struct ActionDef {
    pub name: String,
    pub operator: String,
    pub internal: bool,
    pub branches: Vec<Branch>,
}
#[derive(Clone, Debug)]
pub(crate) struct Branch {
    pub guard: Expr,
    pub updates: BTreeMap<String, Expr>,
    pub checks: Vec<Expr>,
    pub outcome: Expr,
    pub preserve_self_loop: bool,
}
#[derive(Clone, Debug)]
pub(crate) struct ClaimDef {
    pub name: String,
    pub expr: Expr,
}
#[derive(Clone, Debug)]
pub(crate) struct PostDef {
    pub name: String,
    pub expr: Expr,
    pub after: Option<String>,
}
#[derive(Clone, Debug)]
pub(crate) struct ProgressDef {
    pub name: String,
    pub trigger: Expr,
    pub target: Expr,
    pub assumption: AssumptionDef,
    pub bound: Option<u64>,
}
#[derive(Clone, Debug, Default)]
pub(crate) struct AssumptionDef {
    pub name: String,
    pub only: Option<Vec<String>>,
    pub without: Vec<String>,
    pub fair: Vec<String>,
}
impl AssumptionDef {
    pub(crate) fn same_scope(&self, other: &Self) -> bool {
        self.only == other.only && self.without == other.without && self.fair == other.fair
    }
}
