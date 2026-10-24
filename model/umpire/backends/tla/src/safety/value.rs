use crate::{
    Type,
    emit::{conjoin, default, integer, text},
    snowcat::ty,
};
use std::{collections::HashMap, fmt::Write};
use umpire_front::expr::{Expr, ExprKind, Op, Unary};

#[derive(Clone)]
pub(super) struct Value {
    pub value: String,
    pub valid: String,
}
/// A dependency-ordered DAG of total values and their independent evaluation validity.
pub(super) struct Values<'a> {
    pub definitions: String,
    pub locals: HashMap<String, Value>,
    nodes: HashMap<&'a Expr, Value>,
    prefix: &'a str,
    pub post: bool,
    pub successor: bool,
    next: usize,
}
impl<'a> Values<'a> {
    pub(super) fn new(prefix: &'a str) -> Self {
        Self {
            definitions: String::new(),
            locals: HashMap::new(),
            nodes: HashMap::new(),
            prefix,
            post: false,
            successor: false,
            next: 0,
        }
    }
    pub(super) fn condition(&mut self, body: impl AsRef<str>) -> String {
        self.next += 1;
        let name = format!("{}{}", self.prefix, 2 * self.next);
        let _ = write!(self.definitions, "LET \\* @type: Bool;\n{name} == {} IN ", body.as_ref());
        name
    }
    pub(super) fn scope(&self, body: impl AsRef<str>) -> String {
        format!("{}{}", self.definitions, body.as_ref())
    }
    #[expect(clippy::too_many_lines, reason = "Exhaustive admitted expression translation")]
    pub(super) fn emit(&mut self, expression: &'a Expr) -> Value {
        if let Some(value) = self.nodes.get(expression) {
            return value.clone();
        }
        let leaf = |value: String| Value { value, valid: "TRUE".into() };
        let value = match &expression.kind {
            ExprKind::Bool(v) => leaf(if *v { "TRUE" } else { "FALSE" }.into()),
            ExprKind::Int(v) => leaf(integer(*v)),
            ExprKind::Text(v) | ExprKind::Enum(v) => leaf(text(v)),
            ExprKind::Unit => leaf("<<>>".into()),
            ExprKind::World(n) | ExprKind::Input(n) => leaf(n.clone()),
            ExprKind::Local(n) => self.locals[n].clone(),
            ExprKind::Check(_, inner) => return self.emit(inner),
            ExprKind::Var(n, primed) => leaf(if self.post && !primed {
                format!("__umpire_before_{n}{}", if self.successor { "'" } else { "" })
            } else {
                format!("{n}{}", if self.successor { "'" } else { "" })
            }),
            ExprKind::Outcome => {
                leaf(format!("__umpire_outcome{}", if self.successor { "'" } else { "" }))
            }
            ExprKind::Bounded(inner) | ExprKind::Payload(inner) => self.emit(inner),
            ExprKind::Some(inner) => {
                let v = self.emit(inner);
                let raw = format!("({}) + 1", v.value);
                let valid = conjoin([v.valid, format!("({raw}) >= 1 /\\ ({raw}) <= {}", i32::MAX)]);
                self.bind(expression, raw, valid, false)
            }
            ExprKind::OptionalOr(option, fallback) => {
                let o = self.emit(option);
                let f = self.emit(fallback);
                self.bind(
                    expression,
                    format!("IF {} # 0 THEN ({}) - 1 ELSE {}", o.value, o.value, f.value),
                    conjoin([o.valid, f.valid]),
                    false,
                )
            }
            ExprKind::Unary(op, inner) => {
                let v = self.emit(inner);
                let raw = match op {
                    Unary::Not => format!("~({})", v.value),
                    Unary::Neg => format!("-({})", v.value),
                };
                self.bind(expression, raw, v.valid, matches!(op, Unary::Neg))
            }
            ExprKind::Binary(op, left, right) => {
                let a = self.emit(left);
                let b = self.emit(right);
                let valid = match op {
                    Op::And => {
                        conjoin([a.valid, format!("IF {} THEN {} ELSE TRUE", a.value, b.valid)])
                    }
                    Op::Or => {
                        conjoin([a.valid, format!("IF {} THEN TRUE ELSE {}", a.value, b.valid)])
                    }
                    _ => conjoin([a.valid, b.valid]),
                };
                let symbol = match op {
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
                    Op::Div | Op::Rem => "",
                };
                let (raw, valid) = if matches!(op, Op::Div | Op::Rem) {
                    let a = a.value;
                    let b = b.value;
                    let mut checks = vec![valid, format!("{b} # 0")];
                    if let Type::Int { min, .. } = &expression.ty
                        && *min < 0
                    {
                        checks.push(format!("~({a} = {} /\\ {b} = -1)", integer(*min)));
                    }
                    let q = format!("__umpire_SafeDiv({a}, {b})");
                    (if *op == Op::Div { q } else { format!("{a} - ({q}) * {b}") }, conjoin(checks))
                } else {
                    (format!("({}) {symbol} ({})", a.value, b.value), valid)
                };
                self.bind(
                    expression,
                    raw,
                    valid,
                    matches!(op, Op::Add | Op::Sub | Op::Mul | Op::Div | Op::Rem),
                )
            }
            ExprKind::If(cond, yes, no) => {
                let c = self.emit(cond);
                let y = self.emit(yes);
                let n = self.emit(no);
                self.bind(
                    expression,
                    format!("IF {} THEN {} ELSE {}", c.value, y.value, n.value),
                    conjoin([c.valid, format!("IF {} THEN {} ELSE {}", c.value, y.valid, n.valid)]),
                    false,
                )
            }
            ExprKind::Tuple(elements)
            | ExprKind::Sequence(elements)
            | ExprKind::Checks(elements) => {
                let values: Vec<_> = elements.iter().map(|e| self.emit(e)).collect();
                let raw = format!(
                    "<<{}>>",
                    values.iter().map(|v| v.value.as_str()).collect::<Vec<_>>().join(", ")
                );
                let raw = if matches!(expression.kind, ExprKind::Checks(_)) {
                    conjoin(values.iter().map(|v| v.value.clone()))
                } else {
                    raw
                };
                self.bind(expression, raw, conjoin(values.into_iter().map(|v| v.valid)), false)
            }
            ExprKind::Length(seq) => {
                let s = self.emit(seq);
                self.bind(expression, format!("Len({})", s.value), s.valid, false)
            }
            ExprKind::Append(seq, element) => {
                let s = self.emit(seq);
                let e = self.emit(element);
                self.bind(
                    expression,
                    format!("Append({}, {})", s.value, e.value),
                    conjoin([s.valid, e.valid]),
                    false,
                )
            }
            ExprKind::GetOr(seq, index, fallback) => {
                let s = self.emit(seq);
                let i = self.emit(index);
                let f = self.emit(fallback);
                self.bind(
                    expression,
                    format!(
                        "IF {} >= 0 /\\ {} < Len({}) THEN {}[{} + 1] ELSE {}",
                        i.value, i.value, s.value, s.value, i.value, f.value
                    ),
                    conjoin([s.valid, i.valid, f.valid]),
                    false,
                )
            }
        };
        self.nodes.insert(expression, value.clone());
        value
    }
    fn bind(
        &mut self,
        expression: &Expr,
        raw: impl AsRef<str>,
        valid: String,
        range: bool,
    ) -> Value {
        let raw = raw.as_ref();
        self.next += 1;
        let v = format!("{}{}", self.prefix, 2 * self.next);
        let ok = format!("{}{}", self.prefix, 2 * self.next + 1);
        let valid = if range && let Type::Int { min, max } = &expression.ty {
            conjoin([
                valid,
                format!("({raw}) >= {} /\\ ({raw}) <= {}", integer(*min), integer(*max)),
            ])
        } else {
            valid
        };
        let _ = write!(self.definitions, "LET \\* @type: Bool;\n{ok} == {valid} IN ");
        let _ = write!(
            self.definitions,
            "LET \\* @type: {};\n{v} == IF {ok} THEN ({raw}) ELSE {} IN ",
            ty(&expression.ty),
            default(&expression.ty)
        );
        Value { value: v, valid: ok }
    }
}

#[cfg(test)]
mod optional_tests {
    use super::*;

    #[test]
    fn present_optional_requires_fallback_validity_and_checks_some_encoding() {
        let int = |kind| Expr { kind, ty: Type::Int { min: 0, max: 255 } };
        let option = int(ExprKind::Some(Box::new(int(ExprKind::Int(1)))));
        let fallback = int(ExprKind::Binary(
            Op::Div,
            Box::new(int(ExprKind::Int(1))),
            Box::new(int(ExprKind::Int(0))),
        ));
        let expression =
            int(ExprKind::OptionalOr(Box::new(option.clone()), Box::new(fallback.clone())));
        let mut values = Values::new("test");
        let o = values.emit(&option);
        let f = values.emit(&fallback);
        let result = values.emit(&expression);
        assert!(
            values
                .definitions
                .contains(&format!("{} == ({}) /\\ ({})", result.valid, o.valid, f.valid)),
            "{}",
            values.definitions
        );
        assert!(values.definitions.contains("0 # 0"), "{}", values.definitions);
        assert!(values.definitions.contains(&i32::MAX.to_string()));
        assert!(values.definitions.contains(">= 1"), "{}", values.definitions);
    }
}
