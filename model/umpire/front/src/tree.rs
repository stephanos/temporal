use crate::{Diagnostic, Type, expr::Expr};
use std::collections::BTreeMap;

/// An admitted Model. Its typed expression tree has no public stability promise.
#[derive(Clone, Debug)]
pub struct Lifted {
    /// TLA+ module name, from the machine module.
    pub name: String,
    /// Authored state struct name.
    pub state_name: String,
    /// Outcome type.
    pub outcome_type: Type,
    /// Unit World constructor or declared World type path.
    pub world_path: String,
    /// Whether the Model declares a World, so emission needs a bound one.
    pub declared_world: bool,
    /// Declared World fields in declaration order.
    pub world_fields: Vec<WorldField>,
    /// Selected World values, set by the consumer after binding.
    pub world_values: Option<Vec<Expr>>,
    /// Complete start states, set by the consumer after binding.
    pub complete_starts: Option<Vec<Vec<Expr>>>,
    /// The machine module, for refusals that name no narrower node.
    pub location: Diagnostic,
    /// State fields in declaration order.
    pub fields: Vec<FieldDef>,
    /// Module constants by authored name.
    pub constants: BTreeMap<String, Expr>,
    /// Generated TLA+ names of module constants, under a declared World.
    pub constant_symbols: BTreeMap<String, String>,
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
    /// The bound concrete catalog, in native catalog order.
    pub catalog: Option<Vec<Entry>>,
    /// The generated TLA+ name of a catalog position inside input relations.
    pub index_symbol: String,
}
/// One concrete catalog action: an index into `Lifted::actions` and its payload values.
#[derive(Clone, Debug)]
pub struct Entry {
    pub action: usize,
    pub values: Vec<Expr>,
    /// The first catalog ordinal with the same identity, so positional duplicates share one
    /// observable edge as they share one native catalog equality class.
    pub ordinal: usize,
}
#[derive(Clone, Debug)]
pub struct FieldDef {
    pub name: String,
    pub ty: Type,
    pub start: Start,
    pub location: Diagnostic,
}
#[derive(Clone, Debug)]
pub struct WorldField {
    pub name: String,
    pub ty: Type,
    pub symbol: String,
    pub location: Diagnostic,
}
#[derive(Clone, Debug)]
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
#[derive(Clone, Debug)]
pub struct ActionDef {
    pub name: String,
    pub operator: String,
    pub internal: bool,
    pub branches: Vec<Branch>,
    pub shape: Shape,
    pub fields: Vec<InputField>,
    pub location: Diagnostic,
}
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Shape {
    Unit,
    Tuple,
    Named,
}
/// One payload field of an action variant, with its independently admitted domain.
#[derive(Clone, Debug)]
pub struct InputField {
    /// Field name, or its position in a tuple variant.
    pub name: String,
    pub ty: Type,
    pub domain: InputDomain,
    /// The TLA+ name of the chosen value inside the variant's relation.
    pub symbol: String,
    pub location: Diagnostic,
}
#[derive(Clone, Debug)]
pub enum InputDomain {
    /// Values the source declares, in order, including positional duplicates.
    Values(Vec<Expr>),
    /// A World-dependent `#[domain]` function, whose values the selected binding supplies.
    World,
}
/// A concrete action named in an assumption: a variant and its exact payload values.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ActionRef {
    pub name: String,
    pub values: Vec<Expr>,
}
#[derive(Clone, Debug)]
pub struct Branch {
    pub guard: Expr,
    pub updates: BTreeMap<String, Expr>,
    pub checks: Vec<Expr>,
    pub outcome: Expr,
    pub preserve_self_loop: bool,
}
#[derive(Clone, Debug)]
pub struct ClaimDef {
    pub name: String,
    pub expr: Expr,
}
#[derive(Clone, Debug)]
pub struct PostDef {
    pub name: String,
    pub expr: Expr,
    pub after: Option<String>,
}
#[derive(Clone, Debug)]
pub struct ProgressDef {
    pub name: String,
    pub trigger: Expr,
    pub target: Expr,
    pub assumption: AssumptionDef,
    pub bound: Option<u64>,
}
#[derive(Clone, Debug, Default)]
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
