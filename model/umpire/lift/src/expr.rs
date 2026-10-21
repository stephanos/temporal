use crate::Type;

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
    World(String),
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
