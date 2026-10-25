use super::*;
use umpire_front::expr::Op;

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

#[test]
fn priming_recurses_through_nested_vectors_and_three_children_without_changing_other_leaves() {
    let tree = |primed| Expr {
        kind: ExprKind::GetOr(
            Box::new(Expr {
                kind: ExprKind::Sequence(vec![
                    Expr::var("state", Type::Bool, primed),
                    Expr::var("already_primed", Type::Bool, true),
                ]),
                ty: Type::Seq(Box::new(Type::Bool)),
            }),
            Box::new(Expr::int(0, Type::Int { min: 0, max: 1 })),
            Box::new(Expr {
                kind: ExprKind::Tuple(vec![
                    Expr { kind: ExprKind::World("world".into()), ty: Type::Bool },
                    Expr { kind: ExprKind::Input("input".into()), ty: Type::Bool },
                    Expr { kind: ExprKind::Outcome, ty: Type::Bool },
                    Expr { kind: ExprKind::Local("local".into()), ty: Type::Bool },
                    Expr::var("fallback", Type::Bool, primed),
                ]),
                ty: Type::Unit,
            }),
        ),
        ty: Type::Bool,
    };
    let original = tree(false);
    assert_eq!(prime(&original), tree(true));
    assert_eq!(original, tree(false));
    assert_eq!(prime(&prime(&original)), tree(true));
}

#[test]
fn signed_remainder_asserts_overflow_and_divides_parenthesised_operands() {
    let int = |kind| Expr { kind, ty: Type::Int { min: -5, max: 5 } };
    let remainder = int(ExprKind::Binary(
        Op::Rem,
        Box::new(int(ExprKind::World("a".into()))),
        Box::new(int(ExprKind::Input("b".into()))),
    ));
    let raw = "(a) - ((IF Assert(~((a) = -5 /\\ (b) = -1), \"Rust signed division/remainder overflow\") THEN __umpire_Div((a), (b)) ELSE 0)) * (b)";
    assert_eq!(
        rendered(&remainder),
        format!(
            "(IF Assert(({raw}) \\in (-5..5), \"checked Rust integer overflow\") THEN ({raw}) ELSE ({raw}))"
        )
    );
}

#[test]
fn unsigned_division_has_no_overflow_guard() {
    let int = |kind| Expr { kind, ty: Type::Int { min: 0, max: 255 } };
    let quotient = int(ExprKind::Binary(
        Op::Div,
        Box::new(int(ExprKind::World("a".into()))),
        Box::new(int(ExprKind::Input("b".into()))),
    ));
    let raw = "__umpire_Div((a), (b))";
    assert_eq!(
        rendered(&quotient),
        format!(
            "(IF Assert(({raw}) \\in (0..255), \"checked Rust integer overflow\") THEN ({raw}) ELSE ({raw}))"
        )
    );
}
