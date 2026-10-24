//! Bounded nominal optional values retain their source identity.
use umpire_tla::Lift;

const SOURCE: &str = r#"
use umpire_dsl::machine;
#[machine] mod countdown {
    use umpire_dsl::{Action, Finite, State, World};
    #[derive(Finite, Clone, Copy, Debug, PartialEq, Eq, Hash)]
    struct Ticks(#[inputs(1..=3)] u8);
    #[derive(World)] struct World { timeout: Option<Ticks> }
    #[derive(State)] struct Model { timeout: Option<Ticks> }
    #[derive(Action)] enum User { Clear }
    impl Model {
        #[starts] fn starts(w: &World) -> impl IntoIterator<Item = Self> {
            [Self { timeout: w.timeout }]
        }
        #[on(User)] fn user(&mut self, action: User) {
            match action { User::Clear => self.timeout = None }
        }
    }
}
"#;

#[test]
fn source_local_optional_ticks_are_admitted() -> Result<(), Box<dyn std::error::Error>> {
    Lift::parse("optional_ticks.rs", SOURCE)?;
    Ok(())
}

use std::collections::BTreeMap;
use umpire_tla::{Binding, Concrete, Literal, Tla, Type};

fn optional(name: &str, raw: Option<i128>) -> Literal {
    Literal::Optional(raw.map(|raw| Box::new(Literal::Bounded { name: name.into(), raw })))
}
fn binding(raw: Option<i128>) -> Binding {
    Binding {
        world: vec![("timeout".into(), optional("Ticks", raw))],
        starts: vec![vec![("timeout".into(), optional("Ticks", raw))]],
        actions: vec![Concrete { action: "User::Clear".into(), fields: Vec::new() }],
    }
}

#[test]
fn binding_preserves_nominal_identity_and_refuses_malformed_fields()
-> Result<(), Box<dyn std::error::Error>> {
    let lifted = Lift::parse("optional_ticks.rs", SOURCE)?;
    for raw in [None, Some(1), Some(3)] {
        let selected = binding(raw);
        let module = Tla::emit(&lifted.bind(&selected)?)?;
        assert_eq!(
            module.metadata.binding.as_ref().ok_or("binding missing")?.world,
            selected.world
        );
        assert!(matches!(module.metadata.parameters[0].ty, Type::Optional(_)));
        assert!(module.cfg.contains(&format!(
            "CONSTANT __umpire_world_timeout = {}",
            raw.map_or(0, |raw| raw + 1)
        )));
    }
    for value in [
        Literal::Int(2),
        Literal::Bool(true),
        optional("Remaining", Some(1)),
        optional("Ticks", Some(0)),
        optional("Ticks", Some(4)),
        optional("Ticks", Some(-1)),
        Literal::Optional(Some(Box::new(Literal::Int(1)))),
        Literal::Optional(Some(Box::new(optional("Ticks", Some(1))))),
    ] {
        for state in [false, true] {
            let mut selected = binding(Some(1));
            if state {
                selected.starts[0][0].1 = value.clone();
            } else {
                selected.world[0].1 = value.clone();
            }
            let error = lifted.bind(&selected).err().ok_or("malformed optional admitted")?;
            assert!(error.message.contains("timeout"), "{error}");
            assert_eq!(error.file, "optional_ticks.rs");
            assert!(error.line > 1 && error.column > 0, "{error}");
        }
    }
    for state in [false, true] {
        for mutation in ["missing", "duplicate", "undeclared"] {
            let mut selected = binding(Some(1));
            let fields = if state { &mut selected.starts[0] } else { &mut selected.world };
            match mutation {
                "missing" => {
                    fields.clear();
                }
                "duplicate" => fields.push(fields[0].clone()),
                _ => fields.push(("extra".into(), Literal::Bool(false))),
            }
            let error = lifted.bind(&selected).err().ok_or("malformed fields admitted")?;
            assert!(error.message.contains(mutation), "{error}");
        }
    }
    Ok(())
}

#[test]
fn optional_complete_starts_keep_whole_row_correlation_and_empty_selection()
-> Result<(), Box<dyn std::error::Error>> {
    let source = SOURCE
        .replace(
            "struct Model { timeout: Option<Ticks> }",
            "struct Model { timeout: Option<Ticks>, origin: Option<Ticks> }",
        )
        .replace("Self { timeout: w.timeout }", "Self { timeout: w.timeout, origin: w.timeout }");
    let lifted = Lift::parse("optional_ticks.rs", &source)?;
    let row = |raw| {
        vec![("timeout".into(), optional("Ticks", raw)), ("origin".into(), optional("Ticks", raw))]
    };
    let mut selected = binding(Some(1));
    selected.starts = vec![row(None), row(Some(3)), row(None)];
    let full = Tla::emit(&lifted.bind(&selected)?)?;
    let init = full.tla.lines().find(|line| line.starts_with("Init ==")).ok_or("Init missing")?;
    assert!(init.contains("(timeout = 0) /\\ (origin = 0)"), "{init}");
    assert!(init.contains("(timeout = 4) /\\ (origin = 4)"), "{init}");
    assert!(!init.contains("(timeout = 0) /\\ (origin = 4)"), "{init}");
    selected.starts.pop();
    assert_eq!(full.tla, Tla::emit(&lifted.bind(&selected)?)?.tla);
    selected.starts.clear();
    let empty = Tla::emit(&lifted.bind(&selected)?)?;
    assert!(empty.tla.lines().any(|line| line == "Init == FALSE"));
    Ok(())
}

#[test]
fn exact_optional_decode_includes_before_observers_and_rejects_wrong_kinds()
-> Result<(), Box<dyn std::error::Error>> {
    use umpire_tlc::{Schema, Type as BackendType, Value, decode_state};
    let schema = Schema {
        fields: BTreeMap::from([
            ("configured".into(), BackendType::OptionalInt { low: 1, high: 3 }),
            ("remaining".into(), BackendType::OptionalInt { low: 0, high: 3 }),
            ("raw".into(), BackendType::Int { min: 1, max: 3 }),
        ]),
        observers: BTreeMap::from([(
            "__umpire_before_configured".into(),
            BackendType::OptionalInt { low: 1, high: 3 },
        )]),
    };
    let dump =
        "/\\ configured = 2\n/\\ remaining = 1\n/\\ raw = 1\n/\\ __umpire_before_configured = 0";
    assert_eq!(decode_state(dump, &schema)?.state["remaining"], Value::Int(1));
    for field in ["configured", "__umpire_before_configured"] {
        let initial = if field == "configured" { "2" } else { "0" };
        for invalid in ["1", "5", "-1", "TRUE", "\"2\"", "<<2>>"] {
            let forged =
                dump.replace(&format!("{field} = {initial}"), &format!("{field} = {invalid}"));
            assert!(decode_state(&forged, &schema).is_err(), "{forged}");
        }
    }
    for forged in [
        dump.replace("raw = 1", "raw = 0"),
        dump.replace("raw = 1", "raw = 4"),
        dump.replace("/\\ raw = 1\n", ""),
        format!("{dump}\n/\\ raw = 1"),
        format!("{dump}\n/\\ extra = 0"),
    ] {
        assert!(decode_state(&forged, &schema).is_err(), "{forged}");
    }
    for (low, high) in [(-1, 3), (3, 1), (0, i128::from(i32::MAX))] {
        let malformed = Schema {
            fields: BTreeMap::from([(
                "x".into(),
                BackendType::Seq(Box::new(BackendType::OptionalInt { low, high })),
            )]),
            observers: BTreeMap::new(),
        };
        assert!(decode_state("x = <<>>", &malformed).is_err());
    }
    Ok(())
}

#[test]
fn shared_countdown_keeps_selected_codes_in_all_cfgs_and_before_metadata()
-> Result<(), Box<dyn std::error::Error>> {
    let source = include_str!("../../../dev/fixtures/fixtures/optional_ticks/model.rs");
    let source = format!(
        "{source}\numpire_testing::fixture!(countdown::World::new(None, None, false, false, false), 100, Claim::new(\"still_unstarted\", |selected: &countdown::World, current: &countdown::Model| (!current.started).into()));"
    );
    let lifted = Lift::parse("optional_ticks/model.rs", &source)?;
    for raw in [None, Some(1), Some(3)] {
        let selected = Binding {
            world: vec![
                ("timeout".into(), optional("Ticks", raw)),
                ("remaining".into(), optional("Remaining", None)),
                ("empty".into(), Literal::Bool(false)),
                ("duplicates".into(), Literal::Bool(false)),
                ("invalid_update".into(), Literal::Bool(false)),
            ],
            starts: vec![vec![
                ("remaining".into(), optional("Remaining", None)),
                ("started".into(), Literal::Bool(false)),
            ]],
            actions: ["Clock::Start", "Clock::Tick"]
                .into_iter()
                .map(|name| Concrete { action: name.into(), fields: Vec::new() })
                .collect(),
        };
        let bound = lifted.bind(&selected)?;
        let module = Tla::emit(&bound)?;
        let assignment =
            format!("CONSTANT __umpire_world_timeout = {}", raw.map_or(0, |raw| raw + 1));
        let cfgs = [
            &module.cfg,
            &module.metadata.invariants[0].cfg,
            &module.metadata.postconditions[0].cfg,
            &module.metadata.negative[0].cfg,
            &module.metadata.progress[0].cfg,
        ];
        for cfg in cfgs {
            assert!(cfg.contains(&assignment), "{cfg}");
        }
        assert!(!module.tla.contains("__umpire_world_timeout'"));
        let post = module
            .tla
            .lines()
            .find(|line| line.starts_with("Post_decrements =="))
            .ok_or("decrement post missing")?;
        assert!(post.contains("remaining'"), "{post}");
        assert!(post.contains("remaining)"), "{post}");
        let safety = Tla::safety(&bound)?;
        assert!(safety.cfg.contains(&assignment));
        assert_eq!(
            safety.metadata.observer_types["__umpire_before_remaining"],
            module.metadata.fields[0].ty
        );
        assert!(matches!(
            safety.metadata.observer_types["__umpire_before_remaining"],
            Type::Optional(_)
        ));
    }
    Ok(())
}
