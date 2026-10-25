//! World selection precedes exact-action membership and emitted name reservation.
#![forbid(unsafe_code)]

use std::path::Path;
use umpire_dsl::{__lifted::BindingOf, ActionValues, Machine, machine};
use umpire_tla::{Binding, Embedded, Lift, Literal, Tla};

#[machine]
mod selected {
    use umpire_dsl::{Action, State, World, path, stay};

    #[derive(World)]
    pub(super) struct World {
        maximum: u8,
        enabled: bool,
    }
    fn attempts(world: &World) -> impl Iterator<Item = u8> {
        1..=world.maximum
    }
    #[derive(State)]
    pub(super) struct Model {
        pub(super) maximum: u8,
        pub(super) done: bool,
    }
    #[derive(Action, Copy)]
    pub(super) enum Worker {
        Poll(#[domain(attempts)] u8),
        Close,
        Pulse,
    }
    impl Model {
        #[starts]
        fn starts(w: &World) -> impl IntoIterator<Item = Self> {
            [Self { maximum: w.maximum, done: false }]
        }
        #[on(Worker)]
        fn worker(&mut self, action: Worker) {
            match action {
                Worker::Close if !self.done => self.done = true,
                Worker::Poll(_) | Worker::Pulse => stay!(),
                Worker::Close => {}
            }
        }
        #[assumption]
        fn current(selected: &World) -> Assumption {
            if selected.maximum == 1 {
                fair([Worker::Poll(1), Worker::Close])
            } else if selected.maximum == 2 {
                fair([Worker::Poll(1), Worker::Poll(2), Worker::Close])
            } else {
                fair([Worker::Poll(1), Worker::Poll(2), Worker::Poll(3), Worker::Close])
            }
        }
        #[assumption]
        fn spare() -> Assumption {
            fair([Worker::Close])
        }
        #[progress]
        fn settles(&self, selected: &World) -> LeadsTo {
            path![!self.done, .., self.done].assuming(Self::current(selected))
        }
    }
}

const SOURCE: &str = include_str!("world_fairness_binding.rs");

fn source() -> &'static str {
    SOURCE.split("const SOURCE:").next().unwrap_or_default()
}

fn binding(maximum: u8) -> Binding {
    let world = selected::World::new(maximum, true);
    Binding::of(&world, &world.starts(), &world.actions())
}
fn parsed(source: &str) -> Result<umpire_front::Lifted, Box<dyn std::error::Error>> {
    umpire_front::read(&syn::parse_file(source)?, &[])
        .map(|(model, _, _)| model)
        .map_err(|error| format!("{error:?}").into())
}
fn changed(from: &str, to: &str) -> String {
    assert!(source().contains(from), "missing source mutation anchor: {from}");
    source().replacen(from, to, 1)
}

#[test]
fn selected_leaves_keep_authored_world_identity_and_exact_fairness()
-> Result<(), Box<dyn std::error::Error>> {
    let lifted = Lift::parse("selected.rs", source())?;
    for (maximum, fair) in [(1, vec![0, 1]), (2, vec![0, 1, 2]), (3, vec![0, 1, 2, 3])] {
        let bound = lifted.bind(&binding(maximum))?;
        assert!(bound.assumptions.iter().all(umpire_front::tree::AssumptionDef::is_resolved));
        assert!(bound.progress.iter().all(|claim| claim.assumption.is_resolved()));
        assert_eq!(bound.assumptions[0].name, "current");
        let module = Tla::emit(&bound)?;
        let claim = &module.metadata.progress[0];
        assert_eq!(claim.fair_inputs, fair);
        assert_eq!(claim.allowed_inputs, (0..usize::from(maximum) + 2).collect::<Vec<_>>());
        assert_eq!(claim.fair, ["Worker::Poll", "Worker::Close"]);
        assert_eq!(module.tla.matches("WF_vars(").count(), usize::from(maximum) + 1);
        assert!(module.tla.contains("CONSTANT"));
        assert!(module.cfg.contains(&format!("CONSTANT __umpire_world_maximum = {maximum}")));
    }
    Ok(())
}

#[test]
fn native_interpretation_selects_before_converting_exact_actions()
-> Result<(), Box<dyn std::error::Error>> {
    let readmitted = parsed(source())?;
    for maximum in [1, 2, 3] {
        for world in [
            umpire_checker::Interpreted::new(selected::World::new(maximum, true))?,
            umpire_checker::Interpreted::from_tree(
                selected::World::new(maximum, true),
                &readmitted,
            )?,
        ] {
            let assumptions = world.assumptions();
            assert_eq!(assumptions[0].0, "current");
            assert_eq!(assumptions[0].1.fair.len(), usize::from(maximum) + 1);
            let last = assumptions[0].1.fair.last().ok_or("missing fair close")?;
            assert_eq!(ActionValues::<selected::World>::action_value(last).0, "Worker::Close");
        }
    }
    Ok(())
}

#[test]
fn every_selected_registration_and_progress_leaf_is_catalog_checked()
-> Result<(), Box<dyn std::error::Error>> {
    for (from, to) in [
        ("fair([Worker::Poll(1), Worker::Close])", "fair([Worker::Poll(2), Worker::Close])"),
        ("fair([Worker::Close])", "fair([Worker::Poll(2)])"),
        (".assuming(Self::current(selected))", ".assuming(fair([Worker::Poll(2)]))"),
    ] {
        let source = changed(from, to);
        let error = Lift::parse("selected.rs", &source)?
            .bind(&binding(1))
            .err()
            .ok_or("out-of-catalog selected scope admitted")?;
        assert!(error.message.contains("catalog lacks"), "{error}");
        let model = parsed(&source)?;
        let error = umpire_checker::Interpreted::from_tree(selected::World::new(1, true), &model)
            .err()
            .ok_or("native out-of-catalog selected scope admitted")?;
        assert!(error.contains("outside the catalog"), "{error}");
    }
    Ok(())
}

#[test]
fn world_shape_validation_precedes_selection() -> Result<(), Box<dyn std::error::Error>> {
    let lifted = Lift::parse("selected.rs", source())?;
    for mutation in ["missing", "duplicate", "undeclared", "mistyped", "range"] {
        let mut candidate = binding(1);
        match mutation {
            "missing" => {
                candidate.world.pop();
            }
            "duplicate" => candidate.world.push(candidate.world[0].clone()),
            "undeclared" => candidate.world.push(("extra".into(), Literal::Int(1))),
            "mistyped" => candidate.world[0].1 = Literal::Bool(true),
            _ => candidate.world[0].1 = Literal::Int(256),
        }
        let error = lifted.bind(&candidate).err().ok_or(mutation)?;
        assert!(error.line > 1 && error.column > 0, "{error}");
        assert!(error.message.contains("World"), "{error}");
    }
    Ok(())
}

#[test]
fn possible_leaf_names_and_post_selection_aliases_are_reserved()
-> Result<(), Box<dyn std::error::Error>> {
    for name in ["Fair_settles_Worker_Poll_3", "Fair_settles_Worker_Close", "Assumption_spare"] {
        let source = if name == "Assumption_spare" {
            changed("fair([Worker::Close])", "fair([Worker::Poll(1), Worker::Close])")
        } else {
            source().to_owned()
        };
        let source = source.replacen(
            "pub(super) done: bool",
            &format!("pub(super) done: bool, pub(super) {name}: bool"),
            1,
        );
        let error = Lift::parse("selected.rs", &source).err().ok_or(name)?;
        assert!(error.message.contains("collides"), "{error}");
        assert!(error.line > 1 && error.column > 0, "{error}");
    }
    Ok(())
}

#[test]
fn empty_selected_restrictions_keep_existing_refusal_and_empty_fair_lists()
-> Result<(), Box<dyn std::error::Error>> {
    let source = changed(
        ".assuming(Self::current(selected))",
        ".assuming(if selected.enabled { only([]) } else { fair([]) })",
    );
    let lifted = Lift::parse("selected.rs", &source)?;
    let error = lifted.bind(&binding(1)).err().ok_or("empty restriction admitted")?;
    assert!(error.message.contains("admits no catalog action"), "{error}");
    let mut disabled = binding(1);
    disabled.world[1].1 = Literal::Bool(false);
    let module = Tla::emit(&lifted.bind(&disabled)?)?;
    assert_eq!(module.metadata.progress[0].fair_inputs, [] as [usize; 0]);
    assert_eq!(module.metadata.progress[0].allowed_inputs, [0, 1, 2]);
    Ok(())
}

#[test]
fn unresolved_embedded_plan_cannot_emit_as_an_empty_scope() -> Result<(), Box<dyn std::error::Error>>
{
    let (mut model, source, _) = umpire_front::read(&syn::parse_file(source())?, &[])
        .map_err(|error| format!("{error:?}"))?;
    model.declared_world = false;
    model.world_fields.clear();
    for field in &mut model.fields {
        field.start = umpire_front::tree::Start::Value(umpire_front::expr::Expr {
            kind: match field.ty {
                umpire_front::Type::Bool => umpire_front::expr::ExprKind::Bool(false),
                _ => umpire_front::expr::ExprKind::Int(1),
            },
            ty: field.ty.clone(),
        });
    }
    let path = Path::new(env!("CARGO_MANIFEST_DIR")).join("tests/main/world_fairness_binding.rs");
    let embedded = Embedded { file: path.to_string_lossy().into_owned(), model: Ok(model), source };
    let lifted = Lift::embedded(Some(&embedded), &path)?;
    let error = Tla::emit(&lifted).err().ok_or("unresolved conditional scope emitted")?;
    assert!(error.message.contains("unresolved"), "{error}");
    Ok(())
}

#[test]
fn source_plans_resolve_before_any_action_conversion() -> Result<(), Box<dyn std::error::Error>> {
    let mut model = parsed(source())?;
    model.assumptions[0].conditional.as_mut().ok_or("missing conditional plan")?.condition =
        umpire_front::expr::Expr {
            kind: umpire_front::expr::ExprKind::World("missing".into()),
            ty: umpire_front::Type::Bool,
        };
    model.progress[0].assumption.fair.clear();
    let error = umpire_checker::Interpreted::from_tree(selected::World::new(1, true), &model)
        .err()
        .ok_or("unresolved registered plan admitted")?;
    assert!(error.contains("lacks World field"), "{error}");
    Ok(())
}

#[test]
fn rebinding_reselects_original_conditional_branches_and_preserves_flat_scopes()
-> Result<(), Box<dyn std::error::Error>> {
    use umpire_dsl::__lifted::Lifts;
    for embedded in [false, true] {
        let lifted = if embedded {
            let path =
                Path::new(env!("CARGO_MANIFEST_DIR")).join("tests/main/world_fairness_binding.rs");
            let embedded = Embedded {
                file: path.to_string_lossy().into_owned(),
                ..selected::World::__umpire_lifted()
            };
            Lift::embedded(Some(&embedded), &path)?
        } else {
            Lift::parse("selected.rs", source())?
        };
        for (first, second) in [(1, 3), (3, 1), (2, 2)] {
            let original = lifted.bind(&binding(first))?;
            let rebound = original.clone().bind(&binding(second))?;
            let fresh = Tla::emit(&lifted.bind(&binding(second))?)?;
            let rebound = Tla::emit(&rebound)?;
            assert_eq!(rebound.tla, fresh.tla);
            assert_eq!(rebound.cfg, fresh.cfg);
            assert_eq!(
                rebound.metadata.progress[0].fair_inputs,
                fresh.metadata.progress[0].fair_inputs
            );
        }
    }
    let flat_source = changed(".assuming(Self::current(selected))", ".assuming(Self::spare())");
    let flat_source = flat_source.replace("#[assumption]\n        fn current", "fn current");
    let flat = Lift::parse("selected.rs", &flat_source)?;
    assert!(flat.assumptions.iter().all(|scope| scope.conditional.is_none()));
    let rebound = flat.bind(&binding(1))?.bind(&binding(3))?;
    assert_eq!(Tla::emit(&rebound)?.metadata.progress[0].fair_inputs, [3]);
    Ok(())
}
