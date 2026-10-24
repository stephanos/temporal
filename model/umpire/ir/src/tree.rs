use crate::{Diagnostic, Type, expr::Expr};
use std::collections::BTreeMap;

/// An admitted Model. Its typed expression tree has no public stability promise.
#[derive(Clone, Debug, umpire_ir_derive::Print)]
#[print_path(::umpire_dsl::__lifted::Lifted)]
pub struct Lifted {
    /// Model name, from the machine module.
    pub name: String,
    /// Authored state struct name.
    pub state_name: String,
    /// Outcome type.
    pub outcome_type: Type,
    /// The authored fact carrier, or unit when no handler returns facts.
    pub fact_type: Type,
    /// Unit World constructor or declared World type path.
    pub world_path: String,
    /// Whether the Model declares a World, so emission needs a bound one.
    pub declared_world: bool,
    /// Declared World fields in declaration order.
    pub world_fields: Vec<WorldField>,
    /// The machine module, for refusals that name no narrower node.
    pub location: Diagnostic,
    /// State fields in declaration order.
    pub fields: Vec<FieldDef>,
    /// Module constants by authored name.
    pub constants: BTreeMap<String, Expr>,
    /// Actions in catalog order.
    pub actions: Vec<ActionDef>,
    /// Registered state claims.
    pub invariants: Vec<ClaimDef>,
    /// Registered action claims.
    pub postconditions: Vec<PostDef>,
    /// Progress claims.
    pub progress: Vec<ProgressDef>,
    /// Scheduling assumptions.
    pub assumptions: Vec<AssumptionDef>,
    /// Supplemental negative claims from golden hooks.
    pub negative: Vec<ClaimDef>,
}
#[derive(Clone, Debug, umpire_ir_derive::Print)]
#[print_path(::umpire_dsl::__lifted::tree::FieldDef)]
pub struct FieldDef {
    pub name: String,
    pub ty: Type,
    pub start: Start,
    pub location: Diagnostic,
}
#[derive(Clone, Debug, umpire_ir_derive::Print)]
#[print_path(::umpire_dsl::__lifted::tree::WorldField)]
pub struct WorldField {
    pub name: String,
    pub ty: Type,
    pub location: Diagnostic,
}
#[derive(Clone, Debug, umpire_ir_derive::Print)]
#[print_path(::umpire_dsl::__lifted::tree::Start)]
pub enum Start {
    Computed,
    Value(Expr),
    Range(Expr, Expr),
    ExclusiveRange(Expr, Expr),
}

impl Lifted {
    /// Whether this source requires an explicitly selected World and complete starts.
    #[must_use]
    pub const fn requires_binding(&self) -> bool {
        self.declared_world
    }

    /// Whether any declared action variant carries payload fields.
    pub fn has_inputs(&self) -> bool {
        self.actions.iter().any(|action| !action.fields.is_empty())
    }
}
#[derive(Clone, Debug, umpire_ir_derive::Print)]
#[print_path(::umpire_dsl::__lifted::tree::ActionDef)]
pub struct ActionDef {
    pub name: String,
    pub internal: bool,
    pub branches: Vec<Branch>,
    pub shape: Shape,
    pub fields: Vec<InputField>,
    pub location: Diagnostic,
}
#[derive(Clone, Copy, Debug, PartialEq, Eq, umpire_ir_derive::Print)]
#[print_path(::umpire_dsl::__lifted::tree::Shape)]
pub enum Shape {
    Unit,
    Tuple,
    Named,
}
/// One payload field of an action variant, with its independently admitted domain.
#[derive(Clone, Debug, umpire_ir_derive::Print)]
#[print_path(::umpire_dsl::__lifted::tree::InputField)]
pub struct InputField {
    /// Field name, or its position in a tuple variant.
    pub name: String,
    pub ty: Type,
    pub location: Diagnostic,
}
/// A concrete action named in an assumption: a variant and its exact payload values.
#[derive(Clone, Debug, PartialEq, Eq, umpire_ir_derive::Print)]
#[print_path(::umpire_dsl::__lifted::tree::ActionRef)]
pub struct ActionRef {
    pub name: String,
    pub values: Vec<Expr>,
}
#[derive(Clone, Debug, umpire_ir_derive::Print)]
#[print_path(::umpire_dsl::__lifted::tree::Branch)]
pub struct Branch {
    /// Single-assignment values, in dependency order, scoped to this branch.
    pub definitions: Vec<(String, Expr)>,
    pub guard: Expr,
    pub updates: BTreeMap<String, Expr>,
    pub checks: Vec<Expr>,
    pub outcome: Expr,
    pub facts: Expr,
    pub preserve_self_loop: bool,
}
#[derive(Clone, Debug, umpire_ir_derive::Print)]
#[print_path(::umpire_dsl::__lifted::tree::ClaimDef)]
pub struct ClaimDef {
    pub name: String,
    pub expr: Expr,
}
#[derive(Clone, Debug, umpire_ir_derive::Print)]
#[print_path(::umpire_dsl::__lifted::tree::PostDef)]
pub struct PostDef {
    pub name: String,
    pub expr: Expr,
    pub after: Option<String>,
}
#[derive(Clone, Debug, umpire_ir_derive::Print)]
#[print_path(::umpire_dsl::__lifted::tree::ProgressDef)]
pub struct ProgressDef {
    pub name: String,
    pub trigger: Expr,
    pub target: Expr,
    pub assumption: AssumptionDef,
    pub bound: Option<u64>,
}
#[derive(Clone, Debug, Default, umpire_ir_derive::Print)]
#[print_path(::umpire_dsl::__lifted::tree::AssumptionDef)]
pub struct AssumptionDef {
    pub name: String,
    pub only: Option<Vec<ActionRef>>,
    pub without: Vec<ActionRef>,
    pub fair: Vec<ActionRef>,
}
impl AssumptionDef {
    pub fn same_scope(&self, other: &Self) -> bool {
        self.only == other.only && self.without == other.without && self.fair == other.fair
    }
    /// Whether a concrete action is admitted, comparing exact values as native scopes do.
    pub fn allows(&self, action: &ActionRef) -> bool {
        self.only.as_ref().is_none_or(|only| only.contains(action))
            && !self.without.contains(action)
    }
    pub fn is_fair(&self, action: &ActionRef) -> bool {
        self.fair.contains(action)
    }
}
