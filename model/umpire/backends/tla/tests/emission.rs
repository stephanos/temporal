//! Execute emitted arithmetic and sequential mutations with the pinned TLC engine.
use std::{collections::BTreeMap, path::Path};
use umpire_tla::{Lift, Module, Tla, Type};
use umpire_tlc::{Schema, Tlc, Value};

fn schema(module: &Module) -> Schema {
    Schema {
        fields: module
            .metadata
            .fields
            .iter()
            .map(|field| (field.name.clone(), kind(&field.ty)))
            .collect(),
        observers: module
            .metadata
            .observer_types
            .iter()
            .map(|(name, ty)| (name.clone(), kind(ty)))
            .collect(),
    }
}
fn kind(ty: &Type) -> umpire_tlc::Type {
    match ty {
        Type::Bool => umpire_tlc::Type::Bool,
        Type::Int { min, max } => umpire_tlc::Type::Int { min: *min, max: *max },
        Type::Bounded(bound) => umpire_tlc::Type::Int { min: bound.low, max: bound.high },
        Type::Optional(bound) => umpire_tlc::Type::OptionalInt { low: bound.low, high: bound.high },
        Type::Text => umpire_tlc::Type::Text,
        Type::Enum { variants, .. } => umpire_tlc::Type::Enum(variants.clone()),
        Type::Unit => umpire_tlc::Type::Tuple(Vec::new()),
        Type::Tuple(values) => umpire_tlc::Type::Tuple(values.iter().map(kind).collect()),
        Type::Seq(value) => umpire_tlc::Type::Seq(Box::new(kind(value))),
    }
}

#[test]
#[allow(clippy::print_stderr)]
fn real_tlc_preserves_ordered_mutations_and_reports_checked_overflow()
-> Result<(), Box<dyn std::error::Error>> {
    if std::env::var("UMPIRE_BACKENDS").as_deref() != Ok("require") {
        eprintln!(
            "Emitted arithmetic controls skipped: no TLC evidence (set UMPIRE_BACKENDS=require)"
        );
        return Ok(());
    }
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..");
    let tlc = Tlc::discover(&root)?;
    let source = r"
        use umpire_dsl::machine;
        #[machine] mod sequential {
            use umpire_dsl::{State,Action};
            #[derive(State)] struct Model { value:u8, copy:u8 }
            #[derive(Action)] enum User { Set }
            impl Model {
                #[on(User)] fn user(&mut self,action:User) {
                    match action { User::Set if self.value==0 => { self.value+=1; self.copy=self.value+1; }, _=>{} }
                }
            }
        }
    ";
    let module = Tla::emit(&Lift::parse("ordered.rs", source)?)?;
    let result = tlc.check(&module.name, &module.tla, &module.cfg, &schema(&module))?;
    assert_eq!(result.states.len(), 2);
    assert!(result.states.contains(&BTreeMap::from([
        ("value".into(), Value::Int(1)),
        ("copy".into(), Value::Int(2))
    ])));

    let overflow = source
        .replace("value:u8", "#[starts(255..=255)] value:u8")
        .replace("self.value==0", "self.value==255");
    let module = Tla::emit(&Lift::parse("overflow.rs", &overflow)?)?;
    let error = tlc
        .check(&module.name, &module.tla, &module.cfg, &schema(&module))
        .err()
        .ok_or("overflow was pruned into a passing graph")?;
    assert!(error.to_string().contains("overflow"), "{error}");

    let minimum = source
        .replace("value:u8", "#[starts(-2147483648..=-2147483648)] value:i32")
        .replace("copy:u8", "copy:i32")
        .replace("self.value==0", "self.value==1");
    let module = Tla::emit(&Lift::parse("minimum.rs", &minimum)?)?;
    let result = tlc.check(&module.name, &module.tla, &module.cfg, &schema(&module))?;
    assert_eq!(result.states.len(), 1);
    Ok(())
}

#[test]
#[allow(clippy::print_stderr)]
fn real_tlc_signed_division_and_remainder_use_rust_rounding()
-> Result<(), Box<dyn std::error::Error>> {
    if std::env::var("UMPIRE_BACKENDS").as_deref() != Ok("require") {
        eprintln!(
            "Signed division controls skipped: no TLC evidence (set UMPIRE_BACKENDS=require)"
        );
        return Ok(());
    }
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..");
    let tlc = Tlc::discover(&root)?;
    let source = r"
        use umpire_dsl::machine;
        #[machine] mod division {
            use umpire_dsl::{State,Action};
            #[derive(State)] struct Model { #[starts(START..=START)] value:i32, copy:i32, done:bool }
            #[derive(Action)] enum User { Set }
            impl Model {
                #[on(User)] fn user(&mut self,action:User) {
                    match action { User::Set if !self.done => { self.copy=EXPRESSION; self.done=true; }, _=>{} }
                }
            }
        }
    ";
    for (start, expression, expected) in [
        ("-2147483648", "self.value / 1", Some(-2_147_483_648)),
        ("-2147483648", "self.value % 3", Some(-2)),
        ("-7", "self.value / 3", Some(-2)),
        ("-7", "self.value / -3", Some(2)),
        ("7", "self.value / -3", Some(-2)),
        ("-2147483648", "self.value / -2147483648", Some(1)),
        ("-2147483648", "self.value / -1", None),
    ] {
        let source = source.replace("START", start).replace("EXPRESSION", expression);
        let module = Tla::emit(&Lift::parse("division.rs", &source)?)?;
        let result = tlc.check(&module.name, &module.tla, &module.cfg, &schema(&module));
        if let Some(expected) = expected {
            let result = result?;
            assert!(
                result.states.iter().any(|state| state["done"] == Value::Bool(true)
                    && state["copy"] == Value::Int(expected)),
                "{expression}: {:?}",
                result.states
            );
        } else {
            assert!(result.is_err(), "Rust division overflow became a passing graph");
        }
    }
    Ok(())
}

#[test]
#[allow(clippy::print_stderr)]
fn real_tlc_empty_half_open_starts_have_no_checked_underflow()
-> Result<(), Box<dyn std::error::Error>> {
    if std::env::var("UMPIRE_BACKENDS").as_deref() != Ok("require") {
        eprintln!("Empty-start control skipped: no TLC evidence (set UMPIRE_BACKENDS=require)");
        return Ok(());
    }
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..");
    let tlc = Tlc::discover(&root)?;
    let source = r"
        use umpire_dsl::machine;
        #[machine] mod counter {
            use umpire_dsl::{State,Action};
            #[derive(State)] struct Model { #[starts(0..0)] value:u8 }
            #[derive(Action)] enum User { Set }
            impl Model {
                #[on(User)] fn user(&mut self,action:User) { match action { User::Set=>{self.value=1;} } }
            }
        }
    ";
    let module = Tla::emit(&Lift::parse("empty.rs", source)?)?;
    let result = tlc.check(&module.name, &module.tla, &module.cfg, &schema(&module))?;
    assert_eq!(result.states.len(), 0);
    Ok(())
}
