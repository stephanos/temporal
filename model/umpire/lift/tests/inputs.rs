//! Independent finite action-input admission, typed catalog binding and exact-payload emission.
use umpire_lift::{Binding, Concrete, Diagnostic, Domain, Lift, Literal, Tla};

const INPUTS: &str = r"
    use umpire_dsl::machine;
    #[machine] mod picks {
        use umpire_dsl::{Action, Finite, State, World, LeadsTo, check, path, stay};
        const limit: u8 = 3;
        #[derive(Finite)] enum Color { Red, Green }
        #[derive(World)] struct World { slots: u8 }
        fn slots(world: &World) -> impl Iterator<Item = u8> { 0..world.slots }
        #[derive(State)] struct Model { total: u8, color: Color, flag: bool }
        #[derive(Action)] enum User {
            Set(#[inputs([1, 1, 3])] u8, bool),
            Paint { color: Color },
            Take(#[domain(slots)] u8),
            Hold,
            Never(#[inputs(1..1)] u8),
        }
        impl Model {
            #[starts]
            fn starts(w: &World) -> impl IntoIterator<Item = Self> { [] }
            #[on(User)]
            fn user(&mut self, action: User, selected: &World) {
                match action {
                    User::Set(limit, _) => { self.flag = true; }
                    User::Set(amount, flag) if flag && amount < selected.slots => {
                        let amount = amount + 1;
                        self.total = amount;
                    }
                    User::Paint { color } if color != self.color => { self.color = color; }
                    User::Take(selected) if selected > self.total => { self.total = selected; }
                    User::Take(..) => stay!(),
                    _ => {}
                }
            }
            #[invariant]
            fn bounded(&self) { check!(self.total <= 9); }
            #[progress]
            fn paints(&self) -> LeadsTo {
                path![self.color == Color::Red, .., self.color == Color::Green]
                    .assuming(fair([User::Set(1, true), User::Paint { color: Color::Green }])
                        .without([User::Set(3, false)]))
            }
        }
    }
";

fn set(amount: i128, flag: bool) -> Concrete {
    Concrete {
        action: "User::Set".into(),
        fields: vec![("0".into(), Literal::Int(amount)), ("1".into(), Literal::Bool(flag))],
    }
}

fn catalog(slots: i128) -> Vec<Concrete> {
    let mut actions = Vec::new();
    for amount in [1, 1, 3] {
        for flag in [false, true] {
            actions.push(set(amount, flag));
        }
    }
    for color in ["Color::Red", "Color::Green"] {
        actions.push(Concrete {
            action: "User::Paint".into(),
            fields: vec![("color".into(), Literal::Enum(color.into()))],
        });
    }
    for slot in 0..slots {
        actions.push(Concrete {
            action: "User::Take".into(),
            fields: vec![("0".into(), Literal::Int(slot))],
        });
    }
    actions.push(Concrete { action: "User::Hold".into(), fields: Vec::new() });
    actions
}

fn binding(slots: i128) -> Binding {
    Binding {
        world: vec![("slots".into(), Literal::Int(slots))],
        starts: Vec::new(),
        actions: catalog(slots),
        domains: vec![Domain {
            action: "User::Take".into(),
            field: "0".into(),
            values: (0..slots).map(Literal::Int).collect(),
        }],
    }
}

fn located(error: &Diagnostic) {
    assert_eq!(error.file, "picks/model.rs");
    assert!(error.line > 1, "{error}");
    assert!(error.column > 0, "{error}");
}

fn line<'a>(tla: &'a str, operator: &str) -> &'a str {
    tla.lines()
        .find(|line| {
            line.starts_with(&format!("{operator} ==")) || line.starts_with(&format!("{operator}("))
        })
        .unwrap_or_default()
}

#[test]
fn the_source_limit_is_the_native_catalog_limit() {
    assert_eq!(umpire_lift::CATALOG_LIMIT, umpire_dsl::CATALOG_LIMIT);
}

#[test]
fn inputs_bind_in_declaration_product_order_keeping_duplicates_and_empty_variants()
-> Result<(), Box<dyn std::error::Error>> {
    let lifted = Lift::parse("picks/model.rs", INPUTS)?;
    assert!(lifted.requires_binding());
    let module = Tla::emit(&lifted.bind(&binding(2))?)?;
    let tla = &module.tla;
    // Positional duplicates stay in the catalog and share the first equal ordinal.
    assert_eq!(
        line(tla, "User_Set_Inputs"),
        "User_Set_Inputs == <<<<0, 1, FALSE>>, <<1, 1, TRUE>>, <<0, 1, FALSE>>, <<1, 1, TRUE>>, <<4, 3, FALSE>>, <<5, 3, TRUE>>>>"
    );
    assert_eq!(
        line(tla, "User_Paint_Inputs"),
        "User_Paint_Inputs == <<<<6, \"Color::Red\">>, <<7, \"Color::Green\">>>>"
    );
    assert_eq!(line(tla, "User_Take_Inputs"), "User_Take_Inputs == <<<<8, 0>>, <<9, 1>>>>");
    assert_eq!(line(tla, "User_Never_Inputs"), "User_Never_Inputs == <<>>");
    assert!(line(tla, "User_Hold").contains("(__umpire_input' = 10)"));
    assert!(line(tla, "User_Set").starts_with(
        "User_Set == \\E __umpire_i \\in DOMAIN User_Set_Inputs : User_Set_At(__umpire_i)"
    ));
    assert_eq!(module.metadata.catalog, catalog(2));
    assert_eq!(
        module.metadata.observer_types["__umpire_input"],
        umpire_lift::Type::Int { min: 0, max: 10 }
    );
    let fields: Vec<_> = module.metadata.actions.iter().map(|action| action.fields.len()).collect();
    assert_eq!(fields, [2, 1, 1, 0, 1]);
    // The empty World domain keeps the variant declared, with zero catalog actions.
    let empty = Tla::emit(&lifted.bind(&binding(0))?)?;
    assert_eq!(line(&empty.tla, "User_Take_Inputs"), "User_Take_Inputs == <<>>");
    assert_eq!(empty.metadata.catalog.len(), 9);
    assert!(empty.tla.lines().any(|line| line.starts_with("Init == (FALSE) /\\")));
    Ok(())
}

#[test]
fn payload_patterns_resolve_lexically_through_their_typed_identities()
-> Result<(), Box<dyn std::error::Error>> {
    let module = Tla::emit(&Lift::parse("picks/model.rs", INPUTS)?.bind(&binding(2))?)?;
    let set = line(&module.tla, "User_Set_At");
    // `limit` names the source constant: a constant pattern, not a binding.
    assert!(set.contains("((__umpire_input_0) = (3))"), "{set}");
    // A later arm is guarded by the failure of the first, and its local shadows the payload.
    assert!(set.contains("~(((__umpire_input_0) = (3)))"), "{set}");
    assert!(set.contains("((__umpire_input_0) < (__umpire_world_slots))"), "{set}");
    assert!(set.contains("total' = (IF Assert(((__umpire_input_0) + (1))"), "{set}");
    let take = line(&module.tla, "User_Take_At");
    // `selected` binds the payload, shadowing the World parameter of the same name.
    assert!(take.contains("((__umpire_input_0) > (total))"), "{take}");
    assert!(!take.contains("__umpire_world_slots"), "{take}");
    let paint = line(&module.tla, "User_Paint_At");
    assert!(paint.contains("((__umpire_input_color) # (color))"), "{paint}");
    Ok(())
}

#[test]
fn exact_payload_scopes_and_fairness_specialize_the_variant_relation()
-> Result<(), Box<dyn std::error::Error>> {
    let module = Tla::emit(&Lift::parse("picks/model.rs", INPUTS)?.bind(&binding(2))?)?;
    assert_eq!(
        line(&module.tla, "Scoped_paints"),
        "Scoped_paints == (\\E __umpire_i \\in {1, 2, 3, 4, 6} : User_Set_At(__umpire_i)) \\/ (User_Paint) \\/ (User_Take) \\/ (User_Hold) \\/ (User_Never)"
    );
    // One obligation per exact fair payload; the duplicate position shares the first.
    assert_eq!(
        line(&module.tla, "ProgressSpec_paints"),
        "ProgressSpec_paints == Init /\\ [][ProgressNext_paints]_vars /\\ WF_vars(Fair_paints_User_Set_2) /\\ WF_vars(Fair_paints_User_Paint_2)"
    );
    assert!(line(&module.tla, "Fair_paints_User_Set_2").contains("User_Set_At(2)"));
    let progress = &module.metadata.progress[0];
    assert_eq!(progress.fair, ["User::Set", "User::Paint"]);
    assert_eq!(progress.fair_inputs, [1, 7]);
    assert_eq!(progress.allowed_inputs, [0, 1, 5, 6, 7, 8, 9, 10]);
    Ok(())
}

#[test]
fn malformed_bindings_refuse_at_the_source_declaration() -> Result<(), Box<dyn std::error::Error>> {
    let lifted = Lift::parse("picks/model.rs", INPUTS)?;
    let valid = binding(2);
    let mut cases: Vec<(Binding, &str)> = Vec::new();
    let mut changed = valid.clone();
    changed.domains.clear();
    cases.push((changed, "missing action domain"));
    let mut changed = valid.clone();
    changed.domains.push(valid.domains[0].clone());
    cases.push((changed, "duplicate action domain"));
    let mut changed = valid.clone();
    changed.domains.push(Domain { field: "1".into(), ..valid.domains[0].clone() });
    cases.push((changed, "extra action domain"));
    let mut changed = valid.clone();
    changed.domains[0].values[0] = Literal::Bool(true);
    cases.push((changed, "does not match admitted type"));
    let mut changed = valid.clone();
    changed.domains[0].values = vec![Literal::Int(0); umpire_lift::CATALOG_LIMIT + 1];
    cases.push((changed, "exceeds the catalog limit"));
    // A catalog from a crossed World disagrees with the bound domain product.
    let mut changed = valid.clone();
    changed.actions = catalog(3);
    cases.push((changed, "native catalog has 12 concrete actions; the source domains admit 11"));
    let mut changed = valid.clone();
    changed.actions.swap(4, 6);
    cases.push((changed, "native catalog ordinal 4 is `User::Paint`"));
    // A source-declared value the native catalog does not enumerate.
    let mut changed = valid.clone();
    changed.actions[4] = set(2, false);
    cases.push((changed, "native catalog ordinal 4 field `0` is Int(2)"));
    let mut changed = valid.clone();
    changed.actions[0].fields.reverse();
    cases.push((changed, "orders input field `1`"));
    let mut changed = valid.clone();
    changed.actions[0].fields.push(("0".into(), Literal::Int(1)));
    cases.push((changed, "duplicate input field `0`"));
    let mut changed = valid.clone();
    changed.actions[0].fields.pop();
    cases.push((changed, "missing input field `1`"));
    let mut changed = valid.clone();
    changed.actions[0].fields.push(("2".into(), Literal::Int(1)));
    cases.push((changed, "extra input field `2`"));
    let mut changed = valid.clone();
    changed.actions[6].fields[0].1 = Literal::Enum("Color::Blue".into());
    cases.push((changed, "native catalog ordinal 6 field `color`"));
    for (candidate, message) in cases {
        let error = lifted.bind(&candidate).err().ok_or(message)?;
        located(&error);
        assert!(error.message.contains(message), "{message}: {error}");
    }
    Ok(())
}

#[test]
fn assumptions_naming_values_outside_the_selected_catalog_refuse_as_natively()
-> Result<(), Box<dyn std::error::Error>> {
    for (from, to, message) in [
        ("User::Set(1, true)", "User::Set(2, true)", "names `User::Set(2, TRUE)`"),
        (
            ".without([User::Set(3, false)])",
            ".without([User::Hold]).only([User::Hold])",
            "admits no catalog action",
        ),
    ] {
        let lifted = Lift::parse("picks/model.rs", &INPUTS.replace(from, to))?;
        let error = lifted.bind(&binding(2)).err().ok_or(message)?;
        located(&error);
        assert!(error.message.contains(message), "{error}");
    }
    let lifted =
        Lift::parse("picks/model.rs", &INPUTS.replace("User::Set(1, true)", "User::Take(1)"))?;
    let error = lifted.bind(&binding(1)).err().ok_or("World-sized absent value admitted")?;
    assert!(error.message.contains("names `User::Take(1)`"), "{error}");
    lifted.bind(&binding(2))?;
    Ok(())
}

#[test]
fn unsupported_input_sources_refuse_by_location() -> Result<(), Box<dyn std::error::Error>> {
    for (from, to, message) in [
        ("Hold,", "Hold, Text(#[inputs([1])] String),", "unsupported type `String`"),
        ("Hold,", "Hold, Raw(u8),", "User::Raw field `0` requires #[inputs]"),
        ("#[derive(Finite)] enum Color", "enum Color", "its type has no admitted Finite domain"),
        (
            "Take(#[domain(slots)] u8)",
            "Take(#[domain(slots)] #[inputs(0..=1)] u8)",
            "declares both #[inputs] and #[domain]",
        ),
        // `slots` stays referenced, so only the unknown function refuses.
        (
            "Take(#[domain(slots)] u8)",
            "Take(#[domain(missing)] u8, #[domain(slots)] u8)",
            "#[domain(missing)] names no local function",
        ),
        ("Take(#[domain(slots)] u8)", "Take(#[domain(slots)] u16)", "item type differs"),
        ("fn slots(world: &World)", "fn slots(world: World)", "a #[domain] function is fn(&World)"),
        (
            "-> impl Iterator<Item = u8> { 0..",
            "-> Vec<u8> { 0..",
            "a #[domain] function is fn(&World)",
        ),
        // A free function no `#[domain]` names is no admitted declaration.
        (
            "fn slots(world: &World) -> impl Iterator<Item = u8> { 0..world.slots }",
            "fn helper() {}",
            "unsupported declaration in machine module",
        ),
        ("#[inputs([1, 1, 3])] u8", "#[inputs([1, limit + 1])] u8", "must be literals"),
        ("#[inputs([1, 1, 3])] u8", "#[inputs(0..)] u8", "requires a bounded range or finite list"),
        ("#[inputs([1, 1, 3])] u8", "#[inputs(0..=4096)] u16", "above the catalog limit 4096"),
        (
            "Never(#[inputs(1..1)] u8)",
            "Never(#[inputs(0..=100)] u8, #[inputs(0..=100)] u8)",
            "action catalog has 10210 concrete actions",
        ),
        (
            "User::Take(..) => stay!(),",
            "User::Take(ref slot) => stay!(),",
            "unsupported input pattern",
        ),
        (
            "User::Take(..) => stay!(),",
            "User::Take(0..=1) => stay!(),",
            "unsupported input pattern",
        ),
        ("User::Take(..) => stay!(),", "User::Take(a, b) => stay!(),", "pattern arity differs"),
        (
            "User::Take(..) => stay!(),",
            "User::Take => stay!(),",
            "carries inputs; match its fields",
        ),
        (
            "User::Take(..) => stay!(),",
            "User::Take(a) | User::Take(a) => stay!(),",
            "or-patterns binding action inputs",
        ),
        ("User::Paint { color }", "User::Paint { shade }", "has no field `shade`"),
        ("User::Paint { color }", "User::Paint {}", "pattern omits fields"),
        ("User::Set(1, true)", "User::Set", "names no concrete action"),
        ("User::Set(1, true)", "User::Set(1)", "requires exactly one value for `1`"),
        ("User::Set(1, true)", "User::Set(1, selected)", "unresolved name `selected`"),
        (
            "User::Paint { color: Color::Green }",
            "User::Paint { color: Color::Green, .. }",
            "unsupported action in an assumption list",
        ),
        (
            "#[progress]",
            "#[postcondition(after = User::Take)] fn after(&self) { check!(true); }\n#[progress]",
            "`User::Take` carries inputs; it is not one catalog action",
        ),
        // Generated relation names collide with an authored variant or obligation prefix.
        ("Hold,", "Hold, Set_At,", "generated TLA+ name `User_Set_At`"),
        (
            "const limit: u8 = 3;",
            "const limit: u8 = 3; const Fair_paints_User_Set_9: u8 = 0;",
            "generated TLA+ names `Fair_paints_User_Set_*`",
        ),
    ] {
        let source = INPUTS.replace(from, to);
        assert_ne!(source, INPUTS, "mutation target missing: {from}");
        let error = Lift::parse("picks/model.rs", &source)
            .err()
            .ok_or_else(|| format!("unsupported input source admitted: {to}"))?;
        located(&error);
        assert!(error.message.contains(message), "{to}: {error}");
    }
    Ok(())
}

#[test]
fn a_fair_unit_variant_named_like_a_fair_position_refuses() -> Result<(), Box<dyn std::error::Error>>
{
    // `Fair_paints_User_Set_2` would be both the unit variant's obligation and the second
    // position of the fair payload variant `User::Set`.
    let source = INPUTS
        .replace("Hold,", "Hold, Set_2,")
        .replace("fair([User::Set(1, true),", "fair([User::Set(1, true), User::Set_2,");
    let error =
        Lift::parse("picks/model.rs", &source).err().ok_or("colliding fairness admitted")?;
    located(&error);
    assert!(error.message.contains("generated TLA+ name `Fair_paints_User_Set_2`"), "{error}");
    Ok(())
}

#[test]
fn inputs_without_a_declared_world_refuse_at_the_variant() -> Result<(), Box<dyn std::error::Error>>
{
    // Native generation admits this Model under the unit World; its catalog identity needs a
    // declared World's binding, so the lifted reading refuses it.
    let source = r"
        use umpire_dsl::machine;
        #[machine] mod picks {
            use umpire_dsl::{Action, State};
            #[derive(State)] struct Model { total: u8 }
            #[derive(Action)] enum User { Set(#[inputs(0..=1)] u8) }
            impl Model {
                #[on(User)]
                fn user(&mut self, action: User) { match action { User::Set(amount) => { self.total = amount; } } }
            }
        }
    ";
    let error = Lift::parse("picks/model.rs", source).err().ok_or("World-less inputs admitted")?;
    assert_eq!((error.line, error.column), (6, 43), "{error}");
    assert!(error.message.contains("action inputs require a declared World"), "{error}");
    Ok(())
}

#[test]
fn the_catalog_limit_admits_its_boundary_and_refuses_one_more()
-> Result<(), Box<dyn std::error::Error>> {
    // Nine static actions plus the World-sized Take variant.
    let boundary = INPUTS.replace("Hold,", "Hold, Big(#[inputs(0..=4086)] u16),");
    let lifted = Lift::parse("picks/model.rs", &boundary)?;
    let big = |actions: &mut Vec<Concrete>| {
        actions.extend((0..=4086).map(|value| Concrete {
            action: "User::Big".into(),
            fields: vec![("0".into(), Literal::Int(value))],
        }));
    };
    let mut selected = binding(0);
    big(&mut selected.actions);
    assert_eq!(Tla::emit(&lifted.bind(&selected)?)?.metadata.catalog.len(), 4096);
    let mut over = binding(1);
    big(&mut over.actions);
    let error = lifted.bind(&over).err().ok_or("4097 actions admitted")?;
    located(&error);
    assert!(error.message.contains("has 4097 concrete actions, above the limit 4096"), "{error}");
    let error = Lift::parse("picks/model.rs", &boundary.replace("0..=4086", "0..=4087"))
        .err()
        .ok_or("4097 static actions admitted")?;
    located(&error);
    assert!(error.message.contains("has 4097 concrete actions"), "{error}");
    Ok(())
}

#[test]
fn internal_payload_actors_keep_their_classification() -> Result<(), Box<dyn std::error::Error>> {
    let source =
        INPUTS.replace("#[derive(Action)] enum User", "#[derive(Action)] #[internal] enum User");
    let module = Tla::emit(&Lift::parse("picks/model.rs", &source)?.bind(&binding(2))?)?;
    assert!(module.metadata.actions.iter().all(|action| action.internal));
    assert_eq!(module.metadata.actions[0].fields.len(), 2);
    assert!(!module.tla.contains("WF_vars(User_Set)"));
    Ok(())
}
