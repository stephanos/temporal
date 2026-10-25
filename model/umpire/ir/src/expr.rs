use crate::{Type, value::Value};

#[derive(Clone, Debug, PartialEq, Eq, Hash, umpire_ir_derive::Print)]
#[must_use]
#[print_path(::umpire_dsl::__lifted::expr::Expr)]
pub struct Expr {
    pub kind: ExprKind,
    pub ty: Type,
}
#[derive(Clone, Debug, PartialEq, Eq, Hash, umpire_ir_derive::Print)]
#[print_path(::umpire_dsl::__lifted::expr::ExprKind)]
pub enum ExprKind {
    Bool(bool),
    Int(i128),
    Text(String),
    Unit,
    Var(String, bool),
    /// A World field, by name; a back end may rename it.
    World(String),
    /// A chosen action input value of the enclosing action, by field name; a back end may
    /// rename it.
    Input(String),
    /// The outcome of the edge a postcondition judges; a back end names it.
    Outcome,
    /// An ordered definition in the enclosing branch.
    Local(String),
    Enum(String),
    Unary(Unary, Box<Expr>),
    Binary(Op, Box<Expr>, Box<Expr>),
    If(Box<Expr>, Box<Expr>, Box<Expr>),
    Tuple(Vec<Expr>),
    Sequence(Vec<Expr>),
    /// Construct a nominal wrapper without introducing a finite-domain runtime guard.
    Bounded(Box<Expr>),
    /// Read a nominal wrapper's integer member.
    Payload(Box<Expr>),
    /// Encode a present nominal payload as its scalar ordinal.
    Some(Box<Expr>),
    /// Read a present payload or an eagerly evaluated nominal fallback.
    OptionalOr(Box<Expr>, Box<Expr>),
    Length(Box<Expr>),
    Append(Box<Expr>, Box<Expr>),
    GetOr(Box<Expr>, Box<Expr>, Box<Expr>),
    Checks(Vec<Expr>),
    /// One `check!` in a claim body, by its position in the Model's check table, which holds
    /// the text a failure reports; back ends that do not report it read through. The position
    /// is meaningful only in the tree `#[machine]` embeds, which carries that table.
    Check(usize, Box<Expr>),
}
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash, umpire_ir_derive::Print)]
#[print_path(::umpire_dsl::__lifted::expr::Unary)]
pub enum Unary {
    Not,
    Neg,
}
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash, umpire_ir_derive::Print)]
#[print_path(::umpire_dsl::__lifted::expr::Op)]
pub enum Op {
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
impl ExprKind {
    /// Immediate children in expression evaluation order.
    pub fn children(&self) -> impl Iterator<Item = &Expr> {
        let (fixed, many): ([Option<&Expr>; 3], &[Expr]) = match self {
            Self::Bool(_)
            | Self::Int(_)
            | Self::Text(_)
            | Self::Unit
            | Self::Var(..)
            | Self::World(_)
            | Self::Input(_)
            | Self::Outcome
            | Self::Local(_)
            | Self::Enum(_) => ([None, None, None], &[]),
            Self::Unary(_, value)
            | Self::Bounded(value)
            | Self::Payload(value)
            | Self::Some(value)
            | Self::Length(value)
            | Self::Check(_, value) => ([Some(value.as_ref()), None, None], &[]),
            Self::Binary(_, left, right)
            | Self::OptionalOr(left, right)
            | Self::Append(left, right) => ([Some(left.as_ref()), Some(right.as_ref()), None], &[]),
            Self::If(first, second, third) | Self::GetOr(first, second, third) => {
                ([Some(first.as_ref()), Some(second.as_ref()), Some(third.as_ref())], &[])
            }
            Self::Tuple(values) | Self::Sequence(values) | Self::Checks(values) => {
                ([None, None, None], values)
            }
        };
        fixed.into_iter().flatten().chain(many.iter())
    }
    /// Immediate children in expression evaluation order.
    pub fn children_mut(&mut self) -> impl Iterator<Item = &mut Expr> {
        let (fixed, many): ([Option<&mut Expr>; 3], &mut [Expr]) = match self {
            Self::Bool(_)
            | Self::Int(_)
            | Self::Text(_)
            | Self::Unit
            | Self::Var(..)
            | Self::World(_)
            | Self::Input(_)
            | Self::Outcome
            | Self::Local(_)
            | Self::Enum(_) => ([None, None, None], &mut []),
            Self::Unary(_, value)
            | Self::Bounded(value)
            | Self::Payload(value)
            | Self::Some(value)
            | Self::Length(value)
            | Self::Check(_, value) => ([Some(value.as_mut()), None, None], &mut []),
            Self::Binary(_, left, right)
            | Self::OptionalOr(left, right)
            | Self::Append(left, right) => {
                ([Some(left.as_mut()), Some(right.as_mut()), None], &mut [])
            }
            Self::If(first, second, third) | Self::GetOr(first, second, third) => {
                ([Some(first.as_mut()), Some(second.as_mut()), Some(third.as_mut())], &mut [])
            }
            Self::Tuple(values) | Self::Sequence(values) | Self::Checks(values) => {
                ([None, None, None], values)
            }
        };
        fixed.into_iter().flatten().chain(many.iter_mut())
    }
}

impl Expr {
    pub fn bool(value: bool) -> Self {
        Self { kind: ExprKind::Bool(value), ty: Type::Bool }
    }
    pub fn int(value: i128, ty: Type) -> Self {
        Self { kind: ExprKind::Int(value), ty }
    }
    pub fn unit() -> Self {
        Self { kind: ExprKind::Unit, ty: Type::Unit }
    }
    /// A value of `ty` as a constant; nominal and optional values keep their integer encoding.
    pub fn constant(ty: &Type, value: &Value) -> Self {
        let kind = match (value, ty) {
            (Value::Bool(value), _) => ExprKind::Bool(*value),
            (Value::Int(value), _) => ExprKind::Int(*value),
            (Value::Text(value), _) => ExprKind::Text(value.clone()),
            (Value::Unit, _) => ExprKind::Unit,
            (Value::Enum(index), Type::Enum { variants, .. }) if *index < variants.len() => {
                ExprKind::Enum(variants[*index].clone())
            }
            (Value::Tuple(values), Type::Tuple(types)) if values.len() == types.len() => {
                ExprKind::Tuple(
                    types.iter().zip(values).map(|(ty, value)| Self::constant(ty, value)).collect(),
                )
            }
            (Value::Seq(values), Type::Seq(inner)) => ExprKind::Sequence(
                values.iter().map(|value| Self::constant(inner, value)).collect(),
            ),
            _ => panic!("{value:?} is not a value of {ty:?}"),
        };
        Self { kind, ty: ty.clone() }
    }
    pub fn text(value: impl Into<String>) -> Self {
        Self { kind: ExprKind::Text(value.into()), ty: Type::Text }
    }
    pub fn var(name: impl Into<String>, ty: Type, primed: bool) -> Self {
        Self { kind: ExprKind::Var(name.into(), primed), ty }
    }
    pub fn binary(op: Op, left: Self, right: Self) -> Self {
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
    #[expect(clippy::should_implement_trait, reason = "folds a constant; not an operator")]
    pub fn not(self) -> Self {
        if let ExprKind::Bool(value) = self.kind {
            return Self::bool(!value);
        }
        Self { kind: ExprKind::Unary(Unary::Not, Box::new(self)), ty: Type::Bool }
    }
    pub fn and(self, other: Self) -> Self {
        Self::binary(Op::And, self, other)
    }
    pub fn or(self, other: Self) -> Self {
        Self::binary(Op::Or, self, other)
    }
    pub fn implies(self, other: Self) -> Self {
        self.not().or(other)
    }
    /// `IF condition THEN yes ELSE no`; equal arms fold when the condition cannot fail.
    pub fn select(condition: Self, yes: Self, no: Self) -> Self {
        if yes == no && condition.infallible() {
            return yes;
        }
        Self {
            ty: yes.ty.clone(),
            kind: ExprKind::If(Box::new(condition), Box::new(yes), Box::new(no)),
        }
    }
    /// Whether evaluating this cannot fail: no checked arithmetic and no sequence indexing.
    pub fn infallible(&self) -> bool {
        match &self.kind {
            ExprKind::Local(_)
            | ExprKind::Unary(Unary::Neg, _)
            | ExprKind::GetOr(..)
            | ExprKind::Some(_)
            | ExprKind::Binary(Op::Add | Op::Sub | Op::Mul | Op::Div | Op::Rem, ..) => false,
            _ => self.kind.children().all(Self::infallible),
        }
    }
}

#[cfg(test)]
#[path = "expr_tests.rs"]
mod tests;
