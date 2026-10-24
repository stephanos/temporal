//! Bounded World-only assumption selection preserves the existing flat leaves.
use umpire_front::{Diagnostic, Lifted, Refusal, tree::AssumptionDef};

const SOURCE: &str = r"
#[machine] mod selected {
    use umpire_dsl::{Action, State, World, Scope, LeadsTo, path};
    const TWO: u8 = 2;
    #[derive(World)] struct World { maximum: u8, enabled: bool }
    #[derive(State)] struct Model { done: bool }
    #[derive(Action)] enum Worker { Poll(#[inputs(1..=3)] u8), Close }
    impl Model {
        #[on(Worker)] fn worker(&mut self, action: Worker) {
            match action { Worker::Close => { self.done = true; }, _ => {} }
        }
        #[assumption] fn selected(w: &World) -> Scope<Worker> {
            if w.enabled && w.maximum == 1 {
                fair([Worker::Poll(1), Worker::Close])
            } else if w.maximum <= TWO {
                fair([Worker::Poll(1), Worker::Poll(2), Worker::Close]).without([])
            } else {
                fair([Worker::Poll(1), Worker::Poll(2), Worker::Poll(3), Worker::Close])
            }
        }
        #[progress] fn closes(&self, chosen: &World) -> LeadsTo {
            path![!self.done, .., self.done].assuming(Self::selected(chosen))
        }
    }
}
";

fn admit(source: &str) -> Result<Lifted, Diagnostic> {
    let ast = syn::parse_file(source).map_err(|error| Diagnostic {
        line: error.span().start().line,
        column: error.span().start().column + 1,
        message: error.to_string(),
    })?;
    umpire_front::read(&ast, &[]).map(|(model, _, _)| model).map_err(|refusal| match refusal {
        Refusal::Model(diagnostic) | Refusal::Hook(_, diagnostic) => diagnostic,
    })
}

#[test]
fn exhaustive_helpers_keep_a_distinct_unresolved_scope() -> Result<(), String> {
    let model = admit(SOURCE).map_err(|error| format!("{error:?}"))?;
    assert!(!model.assumptions[0].same_scope(&AssumptionDef::default()));
    assert!(model.assumptions[0].same_scope(&model.progress[0].assumption));
    assert_eq!(model.assumptions[0].name, "selected");
    Ok(())
}

#[test]
fn primitive_worlds_select_one_flat_leaf_and_preserve_registration() -> Result<(), String> {
    use std::collections::BTreeMap;
    use umpire_front::{
        Type,
        expr::{Expr, ExprKind},
    };
    let model = admit(SOURCE).map_err(|error| format!("{error:?}"))?;
    let ty = Type::Int { min: 0, max: 255 };
    for (maximum, enabled, count) in [(1, true, 2), (1, false, 3), (2, true, 3), (3, false, 4)] {
        let world = BTreeMap::from([
            ("maximum".into(), Expr::int(maximum, ty.clone())),
            ("enabled".into(), Expr::bool(enabled)),
        ]);
        let selected = model.assumptions[0].select(&world).map_err(|error| format!("{error:?}"))?;
        assert_eq!(selected.name, "selected");
        assert_eq!(selected.fair.len(), count);
        assert!(selected.conditional.is_none());
        assert!(selected.same_scope(
            &model.progress[0].assumption.select(&world).map_err(|error| format!("{error:?}"))?
        ));
        assert_eq!(selected.fair[0].values[0].kind, ExprKind::Int(1));
    }
    assert_eq!(model.assumptions[0].possible_scopes().len(), 3);
    Ok(())
}
