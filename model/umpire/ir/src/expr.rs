use crate::Type;

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
            ExprKind::Bool(_)
            | ExprKind::Int(_)
            | ExprKind::Text(_)
            | ExprKind::Unit
            | ExprKind::Var(..)
            | ExprKind::World(_)
            | ExprKind::Input(_)
            | ExprKind::Outcome
            | ExprKind::Enum(_) => true,
            ExprKind::Local(_)
            | ExprKind::Unary(Unary::Neg, _)
            | ExprKind::GetOr(..)
            | ExprKind::Some(_) => false,
            ExprKind::Unary(Unary::Not, value)
            | ExprKind::Length(value)
            | ExprKind::Check(_, value)
            | ExprKind::Bounded(value)
            | ExprKind::Payload(value) => value.infallible(),
            ExprKind::Binary(op, left, right) => {
                !matches!(op, Op::Add | Op::Sub | Op::Mul | Op::Div | Op::Rem)
                    && left.infallible()
                    && right.infallible()
            }
            ExprKind::If(a, b, c) => a.infallible() && b.infallible() && c.infallible(),
            ExprKind::Append(a, b) | ExprKind::OptionalOr(a, b) => a.infallible() && b.infallible(),
            ExprKind::Tuple(values) | ExprKind::Sequence(values) | ExprKind::Checks(values) => {
                values.iter().all(Self::infallible)
            }
        }
    }
}
