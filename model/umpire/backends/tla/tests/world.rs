//! Declared World source admission and complete-state binding controls.
use std::{collections::BTreeMap, path::Path};
use umpire_tla::{Binding, Concrete, Diagnostic, Lift, Literal, Tla, Type};
use umpire_tlc::{Tlc, Value, Verdict};

const WORLD: &str = r#"
    use umpire_dsl::machine;
    #[machine] mod counter {
        use umpire_dsl::{Action, State, World, Scope, LeadsTo, check, path};
        const value: u8 = 1;
        #[derive(World)] struct World { value: u8, limit: u8 }
        #[derive(State)] struct Model { value: u8, origin: u8 }
        #[derive(Action)] enum User { Advance }
        impl Model {
            #[starts]
            fn starts(w: &World) -> impl IntoIterator<Item = Self> {
                [Self { value: w.value, origin: w.value },
                 Self { value: w.value + 1, origin: w.value + 1 }]
            }
            fn below(&self, constants: &World) -> bool { self.value < constants.limit }
            #[on(User)]
            fn advance(&mut self, requested: User, selected: &World) {
                match requested {
                    User::Advance if self.below(selected) => { self.increment(selected); },
                    _ => {}
                }
            }
            #[effect]
            fn increment(&mut self, constants: &World) {
                if self.value < constants.limit { self.value += 1; }
            }
            #[invariant]
            fn bounded(&self, constants: &World) {
                check!(self.value <= constants.limit);
            }
            #[postcondition]
            fn receiver_only(&self) { check!(self.value >= value - 1); }
            #[postcondition]
            fn before_only(&self, selected: &Self) { check!(self.value > selected.value); }
            #[postcondition]
            fn before_outcome(&self, selected: &Self, result: &()) {
                check!(self.value > selected.value);
                check!(*result == ());
            }
            #[postcondition(after = User::Advance)]
            fn world_only(&self, before: &World) { check!(self.value <= before.limit); }
            #[postcondition]
            fn before_world(&self, world: &Self, before: &World) {
                check!(self.value > world.value);
                check!(self.value <= before.limit);
            }
            #[postcondition]
            fn before_outcome_world(&self, world: &Self, result: &(), before: &World) {
                check!(self.value > world.value);
                check!(*result == ());
                check!(self.value <= before.limit);
            }
            #[scope]
            fn fair_user(constants: &World) -> Scope<User> { Scope::fair([User::Advance]) }
            #[progress]
            fn reaches_limit(&self, selected: &World) -> LeadsTo {
                path![self.value == selected.value, .., self.value == selected.limit]
                    .assuming(Self::fair_user(selected))
            }
        }
    }
    umpire_testing::fixture!(counter::World::new(1, 3), 100,
        Claim::new("below_start_plus_two", |before: &counter::World, current: &counter::Model|
            (current.value < before.value + 2).into()));
"#;

#[test]
fn declared_world_computed_starts_admit_before_value_binding()
-> Result<(), Box<dyn std::error::Error>> {
    let lifted = Lift::parse("counter/model.rs", WORLD)?;
    assert!(lifted.requires_binding());
    let missing = Tla::emit(&lifted).err().ok_or("unbound World emitted")?;
    located(&missing);
    assert!(missing.message.contains("binding"));
    let module = Tla::emit(&lifted.bind(&binding(1, 3))?)?;
    assert_eq!(module.metadata.world_path, "counter::World");
    assert_eq!(module.metadata.parameters[0].field, "value");
    assert_eq!(module.metadata.parameters[0].symbol, "__umpire_world_value");
    assert_eq!(module.metadata.parameters[0].ty, Type::Int { min: 0, max: 255 });
    assert_eq!(
        module.metadata.binding.as_ref().ok_or("missing binding")?.world,
        binding(1, 3).world
    );
    assert_eq!(module.metadata.postconditions.len(), 6);
    assert!(!module.tla.contains("__umpire_world_limit'"));
    assert!(module.tla.contains("__umpire_outcome'"));
    // `world: &Self` is the before-state: its `value` is unprimed state, never the World constant.
    for (operator, outcome) in [
        ("Post_before_world", ""),
        ("Post_before_outcome_world", "((__umpire_outcome') = (<<>>)), "),
    ] {
        let line = module
            .tla
            .lines()
            .find(|line| line.starts_with(&format!("{operator} ==")))
            .ok_or(operator)?;
        assert!(
            line.contains(&format!(
                "<<((value') > (value)), {outcome}((value') <= (__umpire_world_limit))>>"
            )),
            "{line}"
        );
        assert!(!line.contains("__umpire_world_value"), "{line}");
    }
    Ok(())
}

fn row(value: i128, origin: i128) -> Vec<(String, Literal)> {
    vec![("value".into(), Literal::Int(value)), ("origin".into(), Literal::Int(origin))]
}

fn binding(value: i128, limit: i128) -> Binding {
    Binding {
        world: vec![("value".into(), Literal::Int(value)), ("limit".into(), Literal::Int(limit))],
        starts: vec![row(value, value), row(value + 1, value + 1), row(value, value)],
        actions: unit("User::Advance"),
    }
}

fn unit(action: &str) -> Vec<Concrete> {
    vec![Concrete { action: action.into(), fields: Vec::new() }]
}

fn located(error: &Diagnostic) {
    assert_eq!(error.file, "counter/model.rs");
    assert!(error.line > 1, "{error}");
    assert!(error.column > 0, "{error}");
}

#[test]
fn binding_refuses_malformed_world_and_complete_state_values()
-> Result<(), Box<dyn std::error::Error>> {
    let lifted = Lift::parse("counter/model.rs", WORLD)?;
    let valid = binding(1, 3);
    let mut missing_world = valid.clone();
    missing_world.world.pop();
    let mut wrong_world = valid.clone();
    wrong_world.world[0].1 = Literal::Bool(true);
    let mut out_of_range = valid.clone();
    out_of_range.world[0].1 = Literal::Int(256);
    let mut missing_state = valid.clone();
    missing_state.starts[0].pop();
    let mut wrong_state = valid.clone();
    wrong_state.starts[0][0].1 = Literal::Bool(true);
    for (candidate, message) in [
        (missing_world, "missing World"),
        (wrong_world, "does not match"),
        (out_of_range, "does not match"),
        (missing_state, "missing complete start"),
        (wrong_state, "does not match"),
    ] {
        let error = lifted.bind(&candidate).err().ok_or("malformed binding admitted")?;
        located(&error);
        assert!(error.message.contains(message), "{error}");
    }
    // Admission already refuses integer types wider than TLC's i32, so only negatives reach here.
    let signed = Lift::parse(
        "counter/model.rs",
        r"
        use umpire_dsl::machine;
        #[machine] mod signed {
            use umpire_dsl::{Action, State, World};
            #[derive(World)] struct World { offset: i32, bias: i8 }
            #[derive(State)] struct Model { on: bool }
            #[derive(Action)] enum User { Hold }
            impl Model {
                #[starts]
                fn starts(w: &World) -> impl IntoIterator<Item = Self> { [] }
                #[on(User)] fn user(&mut self, action: User) { match action { _ => {} } }
            }
        }
    ",
    )?;
    let world = |offset: i128, bias: i128| Binding {
        world: vec![("offset".into(), Literal::Int(offset)), ("bias".into(), Literal::Int(bias))],
        starts: Vec::new(),
        actions: unit("User::Hold"),
    };
    for (offset, bias) in [(-1, 3), (i128::from(i32::MIN), 3), (1, -1)] {
        let error = signed.bind(&world(offset, bias)).err().ok_or("negative World admitted")?;
        located(&error);
        assert!(error.message.contains("not representable as a TLC .cfg constant"), "{error}");
    }
    let edge = Tla::emit(&signed.bind(&world(i128::from(i32::MAX), 0))?)?;
    assert!(edge.cfg.contains("= 2147483647"), "{}", edge.cfg);
    Ok(())
}

#[test]
fn complete_starts_preserve_correlation_deduplicate_and_keep_empty()
-> Result<(), Box<dyn std::error::Error>> {
    let lifted = Lift::parse("counter/model.rs", WORLD)?;
    let mut selected = binding(1, 3);
    let module = Tla::emit(&lifted.bind(&selected)?)?;
    let init = module.tla.lines().find(|line| line.starts_with("Init ==")).ok_or("missing Init")?;
    assert!(init.contains("(value = 1) /\\ (origin = 1)"), "{init}");
    assert!(init.contains("(value = 2) /\\ (origin = 2)"), "{init}");
    assert!(!init.contains("(value = 1) /\\ (origin = 2)"), "{init}");
    selected.starts.pop();
    selected.world.reverse();
    for state in &mut selected.starts {
        state.reverse();
    }
    assert_eq!(Tla::emit(&lifted.bind(&selected)?)?.tla, module.tla);
    selected.starts.clear();
    let empty = Tla::emit(&lifted.bind(&selected)?)?;
    assert!(empty.tla.lines().any(|line| line.starts_with("Init == (FALSE) /\\")));
    Ok(())
}

#[test]
fn world_free_starts_of_any_name_admit_and_bind() -> Result<(), Box<dyn std::error::Error>> {
    let source = WORLD.replace(
        "fn starts(w: &World) -> impl IntoIterator<Item = Self> {\n                [Self { value: w.value, origin: w.value },\n                 Self { value: w.value + 1, origin: w.value + 1 }]",
        "fn initial() -> impl IntoIterator<Item = Self> {\n                [Self { value: 1, origin: 1 }, Self { value: 2, origin: 2 }]",
    );
    assert_ne!(source, WORLD);
    let lifted = Lift::parse("counter/model.rs", &source)?;
    assert!(lifted.requires_binding());
    // The method body is native only: Init comes from the bound complete starts.
    let authored = Lift::parse("counter/model.rs", WORLD)?;
    assert_eq!(
        Tla::emit(&lifted.bind(&binding(1, 3))?)?.tla,
        Tla::emit(&authored.bind(&binding(1, 3))?)?.tla
    );
    Ok(())
}

#[test]
fn typed_world_signatures_and_outcome_fields_refuse_by_source_location()
-> Result<(), Box<dyn std::error::Error>> {
    for (from, to) in [
        ("selected: &World", "selected: &mut World"),
        ("selected: &World", "selected: World"),
        ("requested: User, selected: &World", "selected: &World, requested: User"),
        ("selected: &World", "selected: &other::World"),
        ("selected: &World", "selected: &World<u8>"),
        ("requested: User, selected: &World", "requested: User, first: &World, selected: &World"),
        ("struct World {", "struct World<T> {"),
        ("const value:", "#[starts] const value:"),
        ("#[derive(World)]", "#[starts] #[derive(World)]"),
        ("#[derive(Action)]", "#[starts] #[derive(Action)]"),
        ("impl Model {\n            #[starts]", "#[starts] impl Model {\n            #[starts]"),
        ("fn starts(w: &World)", "fn starts(&self, w: &World)"),
        ("fn starts(w: &World)", "fn starts(w: &World, extra: u8)"),
        ("fn below(", "#[starts] fn again() -> [Self; 0] { [] } fn below("),
        ("value: u8, limit: u8", "value: u8, limit: String"),
        ("struct Model { value: u8", "struct Model { #[starts(0..1)] value: u8"),
        ("check!(*result == ());", "check!(result.value == world.value);"),
        ("Self::fair_user(selected)", "Self::fair_user(missing)"),
        ("self.below(selected)", "self.below(missing)"),
    ] {
        let source = WORLD.replace(from, to);
        let error = Lift::parse("counter/model.rs", &source)
            .err()
            .ok_or_else(|| format!("unsupported source admitted: {to}"))?;
        located(&error);
    }
    Ok(())
}

#[test]
fn bare_world_and_before_parameters_do_not_resolve_as_source_constants()
-> Result<(), Box<dyn std::error::Error>> {
    for source in [
        WORLD
            .replace("const value: u8 = 1;", "const value: u8 = 1; const constants: u8 = 3;")
            .replace("check!(self.value <= constants.limit);", "check!(constants == 3);"),
        WORLD
            .replace("const value: u8 = 1;", "const value: u8 = 1; const selected: u8 = 1;")
            .replace("check!(self.value > selected.value);", "check!(selected == 1);"),
        WORLD.replace("check!(*result == ());", "check!(result == ());"),
    ] {
        let error = Lift::parse("counter/model.rs", &source)
            .err()
            .ok_or("a complete parameter resolved as a source constant")?;
        located(&error);
    }
    Ok(())
}

#[test]
fn scalar_enum_world_and_sequence_starts_use_the_admitted_types()
-> Result<(), Box<dyn std::error::Error>> {
    let source = r"
        use umpire_dsl::machine;
        #[machine] mod typed {
            use umpire_dsl::{Action, State, World};
            use imbl::Vector;
            enum Mode { Low, High }
            #[derive(World)] struct World { enabled: bool, mode: Mode }
            #[derive(State)] struct Model { mode: Mode, values: Vector<u8> }
            #[derive(Action)] enum User { Hold }
            impl Model {
                #[starts]
                fn starts(w: &World) -> impl IntoIterator<Item = Self> { [] }
                #[on(User)] fn user(&mut self, action: User) { match action { _ => {} } }
            }
        }
    ";
    let lifted = Lift::parse("counter/model.rs", source)?;
    let mut selected = Binding {
        world: vec![
            ("enabled".into(), Literal::Bool(true)),
            ("mode".into(), Literal::Enum("Mode::High".into())),
        ],
        starts: vec![vec![
            ("mode".into(), Literal::Enum("Mode::Low".into())),
            ("values".into(), Literal::Sequence(vec![Literal::Int(1)])),
        ]],
        actions: unit("User::Hold"),
    };
    let module = Tla::emit(&lifted.bind(&selected)?)?;
    assert!(module.tla.contains("mode = \"Mode::Low\""));
    assert!(module.tla.contains("values = <<1>>"));
    selected.world[1].1 = Literal::Enum("Other::High".into());
    located(&lifted.bind(&selected).err().ok_or("wrong enum type admitted")?);
    selected.world[1].1 = Literal::Enum("Mode::Missing".into());
    located(&lifted.bind(&selected).err().ok_or("unknown enum variant admitted")?);
    selected.world[1].1 = Literal::Enum("Mode::High".into());
    selected.starts[0][1].1 = Literal::Sequence(vec![Literal::Int(-1)]);
    located(&lifted.bind(&selected).err().ok_or("invalid sequence element admitted")?);
    Ok(())
}

#[test]
fn generated_world_constants_avoid_synthesized_action_names()
-> Result<(), Box<dyn std::error::Error>> {
    let source = WORLD
        .replace("enum User { Advance }", "enum __umpire { world_value }")
        .replace("User::Advance", "__umpire::world_value")
        .replace("#[on(User)]", "#[on(__umpire)]")
        .replace("requested: User", "requested: __umpire");
    let selected = Binding { actions: unit("__umpire::world_value"), ..binding(1, 3) };
    let module = Tla::emit(&Lift::parse("counter/model.rs", &source)?.bind(&selected)?)?;
    assert!(module.tla.contains("CONSTANT \\* @type: Int;\n__umpire_world_value_1"));
    assert!(module.tla.contains("CONSTANT \\* @type: Int;\n__umpire_world_limit"));
    assert!(module.tla.contains("__umpire_world_value =="));
    assert_eq!(module.metadata.parameters[0].symbol, "__umpire_world_value_1");
    Ok(())
}

#[test]
#[allow(clippy::print_stderr)]
fn real_tlc_executes_each_cfg_family_with_both_world_bindings()
-> Result<(), Box<dyn std::error::Error>> {
    if std::env::var("UMPIRE_BACKENDS").as_deref() != Ok("require") {
        return Ok(());
    }
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..");
    let tlc = Tlc::discover(&root)?;
    let lifted = Lift::parse("counter/model.rs", WORLD)?;
    for (start, limit, count, negative) in
        [(0, 1, 3, Verdict::Verified), (1, 3, 5, Verdict::Refuted)]
    {
        let module = Tla::emit(&lifted.bind(&binding(start, limit))?)?;
        for (family, cfg, property, verdict) in [
            ("main", &module.cfg, None, Verdict::Verified),
            (
                "invariant",
                &module.metadata.invariants[0].cfg,
                Some(module.metadata.invariants[0].operator.as_str()),
                Verdict::Verified,
            ),
            (
                "postcondition",
                &module.metadata.postconditions[5].cfg,
                Some(module.metadata.postconditions[5].operator.as_str()),
                Verdict::Verified,
            ),
            (
                "negative",
                &module.metadata.negative[0].cfg,
                Some(module.metadata.negative[0].operator.as_str()),
                negative,
            ),
            (
                "progress",
                &module.metadata.progress[0].cfg,
                Some(module.metadata.progress[0].operator.as_str()),
                Verdict::Verified,
            ),
        ] {
            let result = tlc.check(&module.name, &module.tla, cfg, &module.schema()?)?;
            eprintln!("World({start},{limit})/{family} TLC raw output:\n{}", result.output);
            assert_eq!(result.states.len(), count, "World({start},{limit})/{family}");
            assert!(result.states.contains(&BTreeMap::from([
                ("value".into(), Value::Int(limit)),
                ("origin".into(), Value::Int(start))
            ])));
            if let Some(property) = property {
                assert_eq!(
                    result.properties[property].verdict, verdict,
                    "World({start},{limit})/{family}"
                );
                if verdict == Verdict::Refuted {
                    let trace = result.properties[property]
                        .trace
                        .as_ref()
                        .ok_or("missing negative trace")?;
                    assert_eq!(trace.states.len(), 2);
                    assert_eq!(trace.states[0].state["value"], Value::Int(2));
                    assert_eq!(trace.states[1].state["value"], Value::Int(3));
                }
            } else {
                assert!(
                    result
                        .properties
                        .values()
                        .all(|property| property.verdict == Verdict::Verified)
                );
            }
        }
    }
    Ok(())
}
