use super::*;

fn cases(scale: i128) -> Vec<(ExprKind, &'static [i128], bool)> {
    let child = |id| Expr::int(scale * id, Type::Int { min: -10, max: 10 });
    let boxed = |id| Box::new(child(id));
    vec![
        (ExprKind::Bool(true), &[], true),
        (ExprKind::Int(7), &[], true),
        (ExprKind::Text("text".into()), &[], true),
        (ExprKind::Unit, &[], true),
        (ExprKind::Var("state".into(), false), &[], true),
        (ExprKind::World("world".into()), &[], true),
        (ExprKind::Input("input".into()), &[], true),
        (ExprKind::Outcome, &[], true),
        (ExprKind::Local("local".into()), &[], false),
        (ExprKind::Enum("Variant".into()), &[], true),
        (ExprKind::Unary(Unary::Not, boxed(1)), &[1], true),
        (ExprKind::Unary(Unary::Neg, boxed(1)), &[1], false),
        (ExprKind::Binary(Op::Eq, boxed(1), boxed(2)), &[1, 2], true),
        (ExprKind::If(boxed(1), boxed(2), boxed(3)), &[1, 2, 3], true),
        (ExprKind::Tuple(vec![child(1), child(2), child(3), child(4)]), &[1, 2, 3, 4], true),
        (ExprKind::Sequence(vec![child(1), child(2), child(3), child(4)]), &[1, 2, 3, 4], true),
        (ExprKind::Bounded(boxed(1)), &[1], true),
        (ExprKind::Payload(boxed(1)), &[1], true),
        (ExprKind::Some(boxed(1)), &[1], false),
        (ExprKind::OptionalOr(boxed(1), boxed(2)), &[1, 2], true),
        (ExprKind::Length(boxed(1)), &[1], true),
        (ExprKind::Append(boxed(1), boxed(2)), &[1, 2], true),
        (ExprKind::GetOr(boxed(1), boxed(2), boxed(3)), &[1, 2, 3], false),
        (ExprKind::Checks(vec![child(1), child(2), child(3), child(4)]), &[1, 2, 3, 4], true),
        (ExprKind::Check(42, boxed(1)), &[1], true),
        (ExprKind::Tuple(vec![]), &[], true),
        (ExprKind::Sequence(vec![]), &[], true),
        (ExprKind::Checks(vec![]), &[], true),
    ]
}

#[test]
fn children_visit_every_variant_in_order_and_mutate_each_child_once() {
    for ((mut kind, expected, _), (mutated, _, _)) in cases(1).into_iter().zip(cases(-1)) {
        let values: Vec<_> = kind.children().map(|expr| expr.kind.clone()).collect();
        assert_eq!(values, expected.iter().copied().map(ExprKind::Int).collect::<Vec<_>>());
        let mut visited = Vec::new();
        for child in kind.children_mut() {
            let ExprKind::Int(value) = &mut child.kind else { panic!("unexpected child") };
            visited.push(*value);
            *value = -*value;
        }
        assert_eq!(visited, expected);
        assert_eq!(kind, mutated);
    }
    let nested = Expr { kind: ExprKind::Tuple(vec![Expr::bool(true)]), ty: Type::Unit };
    let kind = ExprKind::Check(0, Box::new(nested.clone()));
    assert_eq!(kind.children().collect::<Vec<_>>(), vec![&nested]);
}

#[test]
fn infallible_preserves_explicit_false_cases_and_propagates_child_failure() {
    for (kind, children, expected) in cases(1) {
        let expr = Expr { kind, ty: Type::Unit };
        assert_eq!(expr.infallible(), expected, "{:?}", expr.kind);
        for index in 0..children.len() {
            let mut failing = expr.clone();
            let Some(child) = failing.kind.children_mut().nth(index) else {
                panic!("missing child")
            };
            child.kind = ExprKind::Local("may_fail".into());
            assert!(!failing.infallible(), "child {index} of {:?}", failing.kind);
        }
    }
    for (op, expected) in [
        (Op::Add, false),
        (Op::Sub, false),
        (Op::Mul, false),
        (Op::Div, false),
        (Op::Rem, false),
        (Op::And, true),
        (Op::Or, true),
        (Op::Eq, true),
        (Op::Ne, true),
        (Op::Lt, true),
        (Op::Le, true),
        (Op::Gt, true),
        (Op::Ge, true),
    ] {
        let expr = Expr {
            kind: ExprKind::Binary(op, Box::new(Expr::unit()), Box::new(Expr::unit())),
            ty: Type::Unit,
        };
        assert_eq!(expr.infallible(), expected, "{op:?}");
    }
}
