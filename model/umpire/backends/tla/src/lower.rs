//! What the two TLA+ lowerings share: the module's asserting `emit` and the safety module's
//! total `(value, valid)` form. They differ in how they guard partial operations and name
//! state, so only leaves, operator symbols, Rust's division shape and the term helpers both
//! build with live here.
use crate::Type;
use umpire_front::expr::{ExprKind, Op};

/// A context-free leaf: a literal, world constant or input. Variables and locals are not
/// leaves; each lowering names state and binds locals its own way. Callers list these same
/// kinds in their own match so it stays exhaustive.
pub(crate) fn leaf(kind: &ExprKind) -> String {
    match kind {
        ExprKind::Bool(v) => if *v { "TRUE" } else { "FALSE" }.into(),
        ExprKind::Int(v) => integer(*v),
        ExprKind::Text(v) | ExprKind::Enum(v) => text(v),
        ExprKind::Unit => "<<>>".into(),
        ExprKind::World(name) | ExprKind::Input(name) => name.clone(),
        _ => unreachable!("not a leaf"),
    }
}
/// The infix symbol of `op`; `None` for `Div` and `Rem`, which lower through [`division`].
pub(crate) fn symbol(op: Op) -> Option<&'static str> {
    Some(match op {
        Op::Add => "+",
        Op::Sub => "-",
        Op::Mul => "*",
        Op::And => "/\\",
        Op::Or => "\\/",
        Op::Eq => "=",
        Op::Ne => "#",
        Op::Lt => "<",
        Op::Le => "<=",
        Op::Gt => ">",
        Op::Ge => ">=",
        Op::Div | Op::Rem => return None,
    })
}
/// Whether `op`'s result must stay within its integer type.
pub(crate) fn arithmetic(op: Op) -> bool {
    matches!(op, Op::Add | Op::Sub | Op::Mul | Op::Div | Op::Rem)
}
/// Rust's truncating `left / right` or `left % right` given the guarded quotient term.
/// Operands must be parenthesised or names: a negative literal such as `-5` is not atomic,
/// and SANY rejects `* -5`.
pub(crate) fn division(op: Op, left: &str, right: &str, quotient: &str) -> String {
    match op {
        Op::Div => quotient.into(),
        Op::Rem => format!("{left} - ({quotient}) * {right}"),
        _ => unreachable!("{op:?} is not a division"),
    }
}
/// The one quotient Rust's signed division overflows on, when `ty` can hold it.
pub(crate) fn signed_overflow(left: &str, right: &str, ty: &Type) -> Option<String> {
    match ty {
        Type::Int { min, .. } if *min < 0 => {
            Some(format!("{left} = {} /\\ {right} = -1", integer(*min)))
        }
        _ => None,
    }
}
pub(crate) fn integer(value: i128) -> String {
    if value == i128::from(i32::MIN) { "(-2147483647 - 1)".into() } else { value.to_string() }
}
pub(crate) fn text(value: &str) -> String {
    format!(
        "\"{}\"",
        value
            .replace('\\', "\\\\")
            .replace('"', "\\\"")
            .replace('\n', "\\n")
            .replace('\r', "\\r")
            .replace('\t', "\\t")
    )
}
pub(crate) fn conjoin(terms: impl IntoIterator<Item = String>) -> String {
    let terms: Vec<_> = terms.into_iter().map(|s| format!("({s})")).collect();
    if terms.is_empty() { "TRUE".into() } else { terms.join(" /\\ ") }
}
pub(crate) fn disjoin(terms: impl IntoIterator<Item = String>) -> String {
    let terms: Vec<_> = terms.into_iter().map(|s| format!("({s})")).collect();
    if terms.is_empty() { "FALSE".into() } else { terms.join(" \\/ ") }
}
pub(crate) fn default(ty: &Type) -> String {
    match ty {
        Type::Bool => "FALSE".into(),
        Type::Int { .. } | Type::Optional(_) => "0".into(),
        Type::Bounded(bound) => integer(bound.low),
        Type::Enum { variants, .. } => text(&variants[0]),
        Type::Text => "\"\"".into(),
        Type::Unit | Type::Seq(_) => "<<>>".into(),
        Type::Tuple(types) => {
            format!("<<{}>>", types.iter().map(default).collect::<Vec<_>>().join(", "))
        }
    }
}
