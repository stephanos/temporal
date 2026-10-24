use super::*;

fn optional(child: Expr) -> Expr {
    Expr { ty: child.ty.clone(), kind: ExprKind::Some(Box::new(child)) }
}

#[test]
fn priming_visits_each_nominal_child_and_keeps_world_identity() {
    let int = |kind| Expr { kind, ty: Type::Int { min: 0, max: 255 } };
    let state = int(ExprKind::Var("remaining".into(), false));
    let world = int(ExprKind::World("selected".into()));
    let expression = int(ExprKind::OptionalOr(
        Box::new(optional(int(ExprKind::Bounded(Box::new(state.clone()))))),
        Box::new(int(ExprKind::Payload(Box::new(world)))),
    ));
    let primed = prime(&expression);
    let text = rendered(&primed);
    assert!(text.contains("remaining'"), "{text}");
    assert!(text.contains("selected"), "{text}");
    assert!(!text.contains("selected'"), "{text}");
    let payload = int(ExprKind::Payload(Box::new(state)));
    assert_eq!(rendered(&prime(&payload)), "remaining'");
}

#[test]
fn optional_default_forces_eager_checked_fallback_in_main_emission() {
    let int = |kind| Expr { kind, ty: Type::Int { min: 0, max: 255 } };
    let fallback = int(ExprKind::Binary(
        Op::Div,
        Box::new(int(ExprKind::Int(1))),
        Box::new(int(ExprKind::Int(0))),
    ));
    let fallback_text = rendered(&fallback);
    let expression =
        int(ExprKind::OptionalOr(Box::new(optional(int(ExprKind::Int(1)))), Box::new(fallback)));
    let text = rendered(&expression);
    assert!(text.starts_with(&format!("(IF ({fallback_text}) = ({fallback_text}) THEN")), "{text}");
    assert!(text.contains("checked Rust integer overflow"), "{text}");
    let mut shared = Shared::new("optional");
    shared.count(&expression);
    assert!(shared.nodes.contains_key(&expression));
    assert_eq!(shared.nodes.len(), 5);
    assert!(text.contains("1..2147483647"), "{text}");
}
