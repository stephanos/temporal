use super::parameters::{Parameters, RawStart, validate_effective_world};
use crate::activity::activity::{Ticks, World};
use umpire_dsl::{__lifted::Lifts, Value, Values};

fn selected() -> Parameters {
    Parameters::new(3, [Some(1), Some(2), Some(3), None], true)
        .unwrap_or_else(|error| panic!("invalid selected parameters: {error}"))
}

fn feature() -> World {
    World::new(3, Some(Ticks(1)), Some(Ticks(2)), Some(Ticks(3)), None, true)
}

#[test]
fn effective_parameters_preserve_every_selected_field_and_provenance() -> umpire_testing::TestResult
{
    let expected = selected();
    assert_eq!(validate_effective_world(&feature())?, expected);
    let provenance = expected.validate_world(&feature())?;
    assert!(provenance.source.ends_with("activity/model.rs"));
    assert_eq!(provenance.fields.len(), 6);
    assert_eq!(provenance.fields[0].name, "maximum_attempts");
    assert_eq!(provenance.fields[4].value, Value::Optional(None));
    assert!(provenance.fields.iter().all(|field| field.location.line > 0));
    assert!(Parameters::new(1, [None; 4], false).is_ok());
    Ok(())
}

#[test]
fn effective_parameters_refuse_invalid_bounds_without_substitution() {
    for maximum in [0, 4, u8::MAX] {
        assert!(Parameters::new(maximum, [None; 4], false).is_err());
    }
    for field in 0..4 {
        for value in [0, 4, u8::MAX] {
            let mut timeouts = [None; 4];
            timeouts[field] = Some(value);
            assert!(Parameters::new(1, timeouts, false).is_err());
        }
    }
    let defaulted =
        World::new(3, Some(Ticks(1)), Some(Ticks(2)), Some(Ticks(3)), Some(Ticks(1)), true);
    assert!(selected().validate_world(&defaulted).is_err());
    let clamped = World::new(2, Some(Ticks(1)), Some(Ticks(2)), Some(Ticks(3)), None, true);
    assert!(selected().validate_world(&clamped).is_err());
    let zero = World::new(3, Some(Ticks(1)), Some(Ticks(2)), Some(Ticks(3)), Some(Ticks(0)), true);
    assert!(validate_effective_world(&zero).is_err());
}

struct Control<const MUTATION: u8>(Vec<(&'static str, Value)>);

impl<const MUTATION: u8> Values for Control<MUTATION> {
    fn values(&self) -> Vec<(&'static str, Value)> {
        self.0.clone()
    }
}

impl<const MUTATION: u8> Lifts for Control<MUTATION> {
    fn __umpire_lifted() -> umpire_dsl::__lifted::Embedded {
        use umpire_dsl::__lifted::Type;
        let mut embedded = World::__umpire_lifted();
        let model = embedded
            .model
            .as_mut()
            .unwrap_or_else(|error| panic!("feature source must be admitted: {error:?}"));
        match MUTATION {
            1 => {
                model.world_fields.pop();
            }
            2 => model.world_fields[0].name = "maximum".into(),
            3 => model.world_fields[0].ty = Type::Bool,
            4 => {
                let Type::Optional(bounded) = &mut model.world_fields[1].ty else { panic!() };
                bounded.high = 2;
            }
            5 => {
                let Type::Optional(bounded) = &mut model.world_fields[1].ty else { panic!() };
                bounded.name = "Remaining".into();
            }
            6 => model.world_fields.push(model.world_fields[0].clone()),
            7 => {
                let Type::Optional(bounded) = &mut model.world_fields[1].ty else { panic!() };
                bounded.field = Some("ticks".into());
            }
            8 => {
                let Type::Optional(bounded) = &mut model.world_fields[1].ty else { panic!() };
                bounded.backing = "u16".into();
            }
            9 => model.world_fields.swap(0, 5),
            _ => {}
        }
        embedded
    }
}

#[test]
fn exact_schema_refuses_missing_duplicate_renamed_retyped_and_changed_domain_fields() {
    let values = feature().values();
    assert!(validate_effective_world(&Control::<0>(values.clone())).is_ok());
    macro_rules! refuses {
        ($($mutation:literal),+ $(,)?) => { $(
            assert!(validate_effective_world(&Control::<$mutation>(values.clone())).is_err());
        )+ };
    }
    refuses!(1, 2, 3, 4, 5, 6, 7, 8, 9);
}

#[test]
fn exact_exports_refuse_missing_duplicate_renamed_retyped_and_nominally_wrong_values() {
    let values = feature().values();
    let mut missing = values.clone();
    missing.pop();
    let mut duplicate = values.clone();
    duplicate.push(values[0].clone());
    let mut renamed = values.clone();
    renamed[0].0 = "maximum";
    let mut kind = values.clone();
    kind[0].1 = Value::Bool(true);
    let mut nominal = values.clone();
    nominal[1].1 = Value::Optional(Some(Box::new(Value::Bounded { name: "Remaining", raw: 1 })));
    let mut reordered = values.clone();
    reordered.swap(0, 5);
    for mutant in [missing, duplicate, renamed, kind, nominal, reordered] {
        assert!(validate_effective_world(&Control::<0>(mutant)).is_err());
    }
}

#[test]
fn raw_start_applies_only_the_pinned_waiting_default_at_its_named_boundary()
-> umpire_testing::TestResult {
    let raw = RawStart {
        maximum_attempts: 3,
        schedule_to_start: None,
        start_to_close: None,
        schedule_to_close: Some(3),
        heartbeat_timeout: Some(2),
        asynchronous_cancel: true,
    };
    assert_eq!(raw.into_effective()?, Parameters::new(3, [Some(3), None, Some(3), Some(2)], true)?);
    assert_eq!(
        RawStart { schedule_to_start: Some(1), ..raw }.into_effective()?,
        Parameters::new(3, [Some(1), None, Some(3), Some(2)], true)?
    );
    assert!(RawStart { schedule_to_close: None, ..raw }.into_effective().is_err());
    let attempt_only = RawStart { start_to_close: Some(2), schedule_to_close: None, ..raw };
    assert_eq!(attempt_only.into_effective()?.schedule_to_start, None);
    for field in 0..4 {
        let mut invalid = raw;
        match field {
            0 => invalid.schedule_to_start = Some(0),
            1 => invalid.start_to_close = Some(0),
            2 => invalid.schedule_to_close = Some(0),
            _ => invalid.heartbeat_timeout = Some(0),
        }
        assert!(invalid.into_effective().is_err());
    }
    Ok(())
}
