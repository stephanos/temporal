use crate::Type;
use std::collections::BTreeMap;

/// An admitted Model. Its typed expression tree has no public stability promise.
#[derive(Clone, Debug)]
pub struct Lifted {
    pub(crate) inputs: Vec<String>,
    pub(crate) starts: Option<Vec<BTreeMap<String, Expr>>>,
    pub(crate) file: String,
    pub(crate) name: String,
    pub(crate) state_name: String,
    pub(crate) outcome_type: Type,
    pub(crate) fact_type: Type,
    pub(crate) state_tuple: bool,
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
    pub choice: usize,
    pub outcome: Expr,
    pub facts: Vec<Expr>,
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
#[derive(Clone, Debug, PartialEq, Eq)]
pub(crate) struct Expr {
    pub kind: ExprKind,
    pub ty: Type,
}
#[derive(Clone, Debug, PartialEq, Eq)]
pub(crate) enum ExprKind {
    Bool(bool),
    Int(i128),
    Text(String),
    Unit,
    Var(String, bool),
    Enum(String),
    Unary(Unary, Box<Expr>),
    Binary(Op, Box<Expr>, Box<Expr>),
    If(Box<Expr>, Box<Expr>, Box<Expr>),
    Tuple(Vec<Expr>),
    Sequence(Vec<Expr>),
    Length(Box<Expr>),
    Append(Box<Expr>, Box<Expr>),
    GetOr(Box<Expr>, Box<Expr>, Box<Expr>),
    Checks(Vec<Expr>),
}
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) enum Unary {
    Not,
    Neg,
}
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) enum Op {
    Add,
    Sub,
    Mul,
    Div,
    Rem,
    And,
    Or,
    Eq,
    Ne,
    Lt,
    Le,
    Gt,
    Ge,
}
impl Expr {
    pub(crate) fn bool(value: bool) -> Self {
        Self { kind: ExprKind::Bool(value), ty: Type::Bool }
    }
    pub(crate) fn int(value: i128, ty: Type) -> Self {
        Self { kind: ExprKind::Int(value), ty }
    }
    pub(crate) fn unit() -> Self {
        Self { kind: ExprKind::Unit, ty: Type::Unit }
    }
    pub(crate) fn text(value: impl Into<String>) -> Self {
        Self { kind: ExprKind::Text(value.into()), ty: Type::Text }
    }
    pub(crate) fn var(name: impl Into<String>, ty: Type, primed: bool) -> Self {
        Self { kind: ExprKind::Var(name.into(), primed), ty }
    }
    pub(crate) fn binary(op: Op, left: Self, right: Self) -> Self {
        if let ExprKind::Bool(value) = left.kind {
            if op == Op::And {
                return if value { right } else { left };
            }
            if op == Op::Or {
                return if value { left } else { right };
            }
        }
        let ty = if matches!(op, Op::Add | Op::Sub | Op::Mul | Op::Div | Op::Rem) {
            left.ty.clone()
        } else {
            Type::Bool
        };
        Self { kind: ExprKind::Binary(op, Box::new(left), Box::new(right)), ty }
    }
    pub(crate) fn not(self) -> Self {
        if let ExprKind::Bool(value) = self.kind {
            return Self::bool(!value);
        }
        Self { kind: ExprKind::Unary(Unary::Not, Box::new(self)), ty: Type::Bool }
    }
    pub(crate) fn and(self, other: Self) -> Self {
        Self::binary(Op::And, self, other)
    }
    pub(crate) fn or(self, other: Self) -> Self {
        Self::binary(Op::Or, self, other)
    }
    pub(crate) fn implies(self, other: Self) -> Self {
        self.not().or(other)
    }
    pub(crate) fn select(condition: Self, yes: Self, no: Self) -> Self {
        Self {
            ty: yes.ty.clone(),
            kind: ExprKind::If(Box::new(condition), Box::new(yes), Box::new(no)),
        }
    }
}
