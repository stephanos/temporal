use super::*;
use umpire_dsl::__lifted::Bounded;

fn bounded(low: i128) -> Bounded {
    Bounded {
        name: "Ticks".into(),
        field: None,
        backing: "u8".into(),
        min: 0,
        max: 255,
        low,
        high: 3,
    }
}

fn evaluate(node: &Node) -> Value {
    Frame::new(|(): &(), _| unreachable!(), None, None, &[], None, &[]).eval(node)
}

fn int(value: i128) -> Node {
    Node::Const(Value::Int(value))
}

#[test]
fn optional_default_is_eager_and_decodes_zero_payload() {
    for (code, expected) in [(0, 3), (1, 0), (2, 1), (4, 3)] {
        assert_eq!(
            evaluate(&Node::OptionalOr(Box::new(int(code)), Box::new(int(3)))),
            Value::Int(expected)
        );
    }
    for op in [Op::Div, Op::Add] {
        let (left, right) = if op == Op::Div { (1, 0) } else { (255, 1) };
        let fallback = Node::Binary(op, Box::new(int(left)), Box::new(int(right)), (0, 255));
        assert!(
            std::panic::catch_unwind(|| evaluate(&Node::OptionalOr(
                Box::new(int(2)),
                Box::new(fallback)
            )))
            .is_err()
        );
    }
}

#[test]
fn some_encoding_checks_ordinal_bounds_without_finite_validation() {
    assert_eq!(evaluate(&Node::Some(Box::new(int(255)))), Value::Int(256));
    assert_eq!(evaluate(&Node::Some(Box::new(int(0)))), Value::Int(1));
    assert_eq!(
        evaluate(&Node::Some(Box::new(int(i128::from(i32::MAX) - 1)))),
        Value::Int(i128::from(i32::MAX))
    );
    for raw in [-1, i128::from(i32::MAX), i128::MAX] {
        assert!(std::panic::catch_unwind(|| evaluate(&Node::Some(Box::new(int(raw))))).is_err());
    }
}

#[test]
fn resolver_preserves_scalar_conversions_and_backing_arithmetic() {
    let model = Lifted {
        name: "scalar".into(),
        state_name: "Model".into(),
        outcome_type: Type::Unit,
        fact_type: Type::Unit,
        world_path: "world".into(),
        declared_world: false,
        world_fields: vec![],
        location: umpire_dsl::__lifted::Diagnostic { line: 1, column: 1, message: String::new() },
        fields: vec![],
        constants: std::collections::BTreeMap::new(),
        actions: vec![],
        invariants: vec![],
        postconditions: vec![],
        progress: vec![],
        assumptions: vec![],
        negative: vec![],
    };
    let world = HashMap::new();
    let resolver = Resolver { model: &model, world: &world, checks: &[], inputs: &[], locals: &[] };
    let nominal = |raw| Expr {
        kind: ExprKind::Bounded(Box::new(Expr {
            kind: ExprKind::Int(raw),
            ty: bounded(1).backing_type(),
        })),
        ty: Type::Bounded(bounded(1)),
    };
    let payload =
        Expr { kind: ExprKind::Payload(Box::new(nominal(9))), ty: bounded(1).backing_type() };
    assert_eq!(resolver.constant(&payload), Ok(Value::Int(9)));
    let present =
        Expr { kind: ExprKind::Some(Box::new(nominal(3))), ty: Type::Optional(bounded(1)) };
    let defaulted = Expr {
        kind: ExprKind::OptionalOr(Box::new(present), Box::new(nominal(1))),
        ty: Type::Bounded(bounded(1)),
    };
    assert_eq!(resolver.constant(&defaulted), Ok(Value::Int(3)));
    let overflow = Expr {
        kind: ExprKind::Binary(
            Op::Add,
            Box::new(Expr::int(255, bounded(1).backing_type())),
            Box::new(Expr::int(1, bounded(1).backing_type())),
        ),
        ty: bounded(1).backing_type(),
    };
    assert!(std::panic::catch_unwind(|| resolver.constant(&overflow)).is_err());
}
