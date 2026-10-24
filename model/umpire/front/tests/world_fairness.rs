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
        assert!(selected.is_resolved());
        assert!(selected.same_scope(
            &model.progress[0].assumption.select(&world).map_err(|error| format!("{error:?}"))?
        ));
        assert_eq!(selected.fair[0].values[0].kind, ExprKind::Int(1));
    }
    assert_eq!(model.assumptions[0].possible_scopes().len(), 3);
    Ok(())
}

const REFUSALS: &[(&str, &str, &str)] = &[
    ("} else if w.maximum <= TWO {", "} else if self.done {", "selected World fields"),
    ("w.enabled && w.maximum == 1", "false && self.done", "selected World fields"),
    ("w.enabled && w.maximum == 1", "true || Self::predicate(w)", "World-only"),
    ("w.enabled && w.maximum == 1", "false && w.maximum + 1 == 2", "arithmetic"),
    ("w.enabled && w.maximum == 1", "w.maximum < w.maximum", "primitive literal"),
    ("w.enabled && w.maximum == 1", "w.unknown", "selected World fields"),
    ("w.enabled && w.maximum == 1", "w.enabled == 1", "type mismatch"),
    ("w.enabled && w.maximum == 1", "w.maximum == true", "type mismatch"),
    ("w.enabled && w.maximum == 1", "w.maximum", "type mismatch"),
    ("w.enabled && w.maximum == 1", "unknown", "source constants"),
    ("w.enabled && w.maximum == 1", "before.done", "selected World fields"),
    ("w.enabled && w.maximum == 1", "after.done", "selected World fields"),
    ("w.enabled && w.maximum == 1", "outcome == ()", "source constants"),
    ("w.enabled && w.maximum == 1", "input == 1", "source constants"),
    ("w.enabled && w.maximum == 1", "loop { break true; }", "World-only"),
    (
        "w.enabled && w.maximum == 1",
        "match w.enabled { true => true, false => false }",
        "World-only",
    ),
    ("w.enabled && w.maximum == 1", "[true].len() == 1", "World-only"),
    ("Self::selected(chosen)", "Self::selected(self)", "selected World parameter"),
    ("Self::selected(chosen)", "Self::selected(&chosen)", "selected World parameter"),
    (
        "Self::selected(chosen)",
        "Self::selected(chosen).fair([Worker::Close])",
        "inside conditional leaves",
    ),
    ("if w.enabled &&", "let w = true; if w.enabled &&", "single expression"),
    ("fair([Worker::Poll(1), Worker::Close])", "Self::selected(w)", "helper-depth limit"),
    (
        "fair([Worker::Poll(1), Worker::Poll(2), Worker::Poll(3), Worker::Close])",
        "fair([Worker::Poll(w.maximum)])",
        "unresolved name",
    ),
    (
        "fair([Worker::Poll(1), Worker::Poll(2), Worker::Poll(3), Worker::Close])",
        "fair([Worker::Poll(1 + 1)])",
        "literal",
    ),
    (
        "fair([Worker::Poll(1), Worker::Poll(2), Worker::Poll(3), Worker::Close])",
        "fair([Worker::Missing])",
        "unknown action",
    ),
    (
        "fair([Worker::Poll(1), Worker::Poll(2), Worker::Poll(3), Worker::Close])",
        "fair(actions())",
        "explicit action array",
    ),
    (
        "fair([Worker::Poll(1), Worker::Poll(2), Worker::Poll(3), Worker::Close])",
        "match w.enabled { _ => fair([]) }",
        "named assumption expression",
    ),
];

#[test]
fn every_authored_branch_and_operand_refuses_before_folding() -> Result<(), String> {
    for &(from, to, message) in REFUSALS {
        assert_eq!(SOURCE.matches(from).count(), 1, "missing or ambiguous anchor {from}");
        let error =
            admit(&SOURCE.replace(from, to)).err().ok_or_else(|| format!("admitted {to}"))?;
        assert!(error.line > 1 && error.column > 0, "{error:?}");
        assert!(error.message.contains(message), "{to}: {error:?}");
    }
    Ok(())
}

#[test]
fn primitive_guards_and_existing_flat_builders_keep_their_types() -> Result<(), String> {
    use std::collections::BTreeMap;
    use umpire_front::{Type, expr::Expr};
    let world = BTreeMap::from([
        ("maximum".into(), Expr::int(1, Type::Int { min: 0, max: 255 })),
        ("enabled".into(), Expr::bool(true)),
    ]);
    for (condition, count) in [
        ("w.enabled", 2),
        ("!w.enabled", 3),
        ("w.enabled == true", 2),
        ("w.enabled == false", 3),
        ("w.enabled != false", 2),
        ("(w.maximum == 1)", 2),
        ("1 == w.maximum", 2),
        ("w.maximum != 1", 3),
        ("w.maximum < TWO", 2),
        ("w.maximum > 1", 3),
        ("w.maximum >= 1", 2),
        ("w.maximum <= 1", 2),
        ("w.enabled || false", 2),
        ("(w.enabled || false) && w.maximum <= TWO", 2),
    ] {
        let model = admit(&SOURCE.replace("w.enabled && w.maximum == 1", condition))
            .map_err(|error| format!("{condition}: {error:?}"))?;
        let selected = model.assumptions[0].select(&world).map_err(|error| format!("{error:?}"))?;
        assert_eq!(selected.fair.len(), count, "{condition}");
    }
    let source = SOURCE
        .replace("maximum: u8", "maximum: i8")
        .replace("TWO: u8", "TWO: i8")
        .replace("w.enabled && w.maximum == 1", "w.maximum > -1");
    admit(&source).map_err(|error| format!("{error:?}"))?;
    let source = SOURCE
        .replace("const TWO: u8 = 2;", "const TWO: u8 = 2; const FLAG: bool = true;")
        .replace("w.enabled && w.maximum == 1", "w.enabled == FLAG");
    admit(&source).map_err(|error| format!("{error:?}"))?;
    let flat = SOURCE.replace(
        "Self::selected(chosen)",
        "fair([Worker::Close]).only([Worker::Close]).without([]).fair([Worker::Poll(1)])",
    );
    let model = admit(&flat).map_err(|error| format!("{error:?}"))?;
    let flat = &model.progress[0].assumption;
    assert!(flat.conditional.is_none());
    assert_eq!(flat.fair.len(), 2);
    assert_eq!(flat.only.as_ref().map(Vec::len), Some(1));
    Ok(())
}

#[test]
fn nonprimitive_worlds_missing_else_and_deep_helpers_refuse() -> Result<(), String> {
    let nominal = SOURCE
        .replace(
            "const TWO: u8 = 2;",
            "const TWO: u8 = 2; #[derive(Finite)] struct Count(#[inputs(1..=3)] u8);",
        )
        .replace("maximum: u8", "maximum: Count");
    for source in [
        nominal.clone(),
        nominal.replace("w.maximum == 1", "w.maximum.0 == 1"),
        nominal
            .replace("maximum: Count", "maximum: Option<Count>")
            .replace("w.maximum == 1", "w.maximum.unwrap_or(Count(1)).0 == 1"),
    ] {
        assert!(admit(&source).is_err(), "nonprimitive guard admitted");
    }
    let missing = SOURCE.replace(" else {\n                fair([Worker::Poll(1), Worker::Poll(2), Worker::Poll(3), Worker::Close])\n            }", "");
    assert_ne!(missing, SOURCE);
    let error = admit(&missing).err().ok_or("nonexhaustive assumption admitted")?;
    assert!(error.message.contains("exhaustive else"), "{error:?}");
    let deep = SOURCE.replace(".without([])", &".fair([])".repeat(40));
    let error = admit(&deep).err().ok_or("deep helper admitted")?;
    assert!(error.message.contains("recursive assumption helper"), "{error:?}");
    Ok(())
}

#[test]
fn folded_arithmetic_is_still_an_authored_forbidden_operand() -> Result<(), String> {
    let equal_leaves = SOURCE
        .replace(
            "fair([Worker::Poll(1), Worker::Poll(2), Worker::Close]).without([])",
            "fair([Worker::Poll(1), Worker::Close])",
        )
        .replace(
            "fair([Worker::Poll(1), Worker::Poll(2), Worker::Poll(3), Worker::Close])",
            "fair([Worker::Poll(1), Worker::Close])",
        );
    for source in [SOURCE, &equal_leaves] {
        let source = source.replace("w.enabled && w.maximum == 1", "false && w.maximum + 1 == 2");
        let error = admit(&source).err().ok_or("folded arithmetic guard admitted")?;
        assert!(error.message.contains("arithmetic"), "{error:?}");
    }
    Ok(())
}
