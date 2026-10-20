//! Regression controls for generation over scalar presence and interacting constraints.

use umpire_annotations::*;

#[derive(Clone, Default)]
struct Message<T> { value: T }

fn catalog<T: FieldValue>(kind: Kind, repeated: Option<RepeatedRules>, presence: Option<Presence>) -> Result<Catalog<Message<T>>, AdmissionErrors> {
    let mut field = Field::new("value", kind, |m: &Message<T>| &m.value, |m| &mut m.value);
    if let Some(rules) = repeated { field = field.repeated(rules); }
    let mut builder = Catalog::response("example.Message").field(field);
    if let Some(presence) = presence { builder = builder.rule(Rule { presence, expected: None }); }
    builder.build()
}

#[test]
fn presence_filling_uses_present_scalar_candidates() -> Result<(), Box<dyn std::error::Error>> {
    for presence in [Presence::Required(vec!["value".into()]), Presence::AnyOf(vec!["value".into()])] {
        let strings = catalog::<String>(Kind::Text(StringRules::default()), None, Some(presence.clone()))?;
        let counters = catalog::<i32>(Kind::Counter(NumberRules { gte: Some(0), lte: Some(1), ..Default::default() }), None, Some(presence.clone()))?;
        let vectors = catalog::<Vec<i32>>(Kind::Counter(NumberRules::default()), Some(RepeatedRules { min_items: Some(0), max_items: Some(1), unique: false }), Some(presence))?;
        for seed in 0..16 {
            let mut text = Message::default(); fill(&mut text, &strings, &mut Rng::new(seed))?;
            assert_ne!(text.value, ""); assert_eq!(validate(&text, &strings), []);
            let mut number = Message::default(); fill(&mut number, &counters, &mut Rng::new(seed))?;
            assert_eq!(number.value, 1); assert_eq!(validate(&number, &counters), []);
            let mut vector = Message::default(); fill(&mut vector, &vectors, &mut Rng::new(seed))?;
            assert_eq!(vector.value.len(), 1); assert_eq!(validate(&vector, &vectors), []);
        }
    }
    let impossible = catalog::<i32>(Kind::Counter(NumberRules { gte: Some(0), lte: Some(0), ..Default::default() }), None, Some(Presence::Required(vec!["value".into()])))?;
    assert!(fill(&mut Message::default(), &impossible, &mut Rng::new(0)).is_err());
    Ok(())
}

#[test]
fn required_substring_can_be_supplied_by_the_suffix() -> Result<(), Box<dyn std::error::Error>> {
    for prefix in [None, Some("a".to_owned())] {
        let catalog = catalog::<String>(Kind::Text(StringRules { prefix: prefix.clone(), contains: Some(if prefix.is_some() { "c" } else { "b" }.into()), suffix: Some("abc".into()), max_len: Some(3), ..Default::default() }), None, None)?;
        assert_eq!(validate(&Message { value: "abc".into() }, &catalog), []);
        let mut message = Message::default(); fill(&mut message, &catalog, &mut Rng::new(1))?;
        assert_eq!(message.value, "abc");
    }
    Ok(())
}

#[test]
fn unique_repeated_times_sample_the_allowed_domain() -> Result<(), Box<dyn std::error::Error>> {
    let rules = NumberRules { gte: Some(0), lte: Some(1), ..Default::default() };
    let repeated = RepeatedRules { min_items: Some(2), max_items: Some(2), unique: true };
    let durations = catalog::<Vec<Duration>>(Kind::Duration(rules.clone()), Some(repeated.clone()), None)?;
    let timestamps = catalog::<Vec<Timestamp>>(Kind::Timestamp(rules), Some(repeated), None)?;
    for seed in 0..16 {
        let mut a = Message::default(); let mut b = Message::default();
        fill(&mut a, &durations, &mut Rng::new(seed))?; fill(&mut b, &durations, &mut Rng::new(seed))?;
        assert_eq!(a.value, b.value);
        assert!(a.value.contains(&Duration { seconds: 0, nanos: 0 }));
        assert!(a.value.contains(&Duration { seconds: 0, nanos: 1 }));
        assert_eq!(validate(&a, &durations), []);
        let mut a = Message::default(); let mut b = Message::default();
        fill(&mut a, &timestamps, &mut Rng::new(seed))?; fill(&mut b, &timestamps, &mut Rng::new(seed))?;
        assert_eq!(a.value, b.value);
        assert!(a.value.contains(&Timestamp { seconds: 0, nanos: 0 }));
        assert!(a.value.contains(&Timestamp { seconds: 0, nanos: 1 }));
        assert_eq!(validate(&a, &timestamps), []);
    }
    Ok(())
}

#[test]
fn zero_maximum_repeated_length_is_fillable() -> Result<(), Box<dyn std::error::Error>> {
    for min_items in [None, Some(0), Some(1)] {
        let catalog = catalog::<Vec<i32>>(Kind::Counter(NumberRules::default()), Some(RepeatedRules { min_items, max_items: Some(0), unique: false }), None)?;
        let mut message = Message::default();
        let result = fill(&mut message, &catalog, &mut Rng::new(1));
        if min_items == Some(1) { assert!(matches!(result, Err(FillError::Field { reason: GenerationError::ImpossibleConstraints, .. }))); }
        else { result?; assert_eq!(message.value, []); assert_eq!(validate(&message, &catalog), []); }
    }
    Ok(())
}

#[test]
fn numeric_range_semantics_match_protovalidate_endpoint_combinations() -> Result<(), Box<dyn std::error::Error>> {
    for inclusive_lower in [false, true] {
        for inclusive_upper in [false, true] {
            for (lower, upper) in [(5, 10), (10, 5), (5, 5)] {
                let rules = NumberRules {
                    gt: (!inclusive_lower).then_some(lower), gte: inclusive_lower.then_some(lower),
                    lt: (!inclusive_upper).then_some(upper), lte: inclusive_upper.then_some(upper),
                };
                let integers = catalog::<i32>(Kind::Counter(rules.clone()), None, None)?;
                let timestamps = catalog::<Timestamp>(Kind::Timestamp(rules.clone()), None, None)?;
                let durations = catalog::<Duration>(Kind::Duration(rules), None, None)?;
                for n in 0..=15 {
                    let low = if inclusive_lower { n >= lower } else { n > lower };
                    let high = if inclusive_upper { n <= upper } else { n < upper };
                    let valid = if lower > upper { low || high } else { low && high };
                    assert_eq!(validate(&Message { value: i32::try_from(n)? }, &integers).is_empty(), valid, "{lower}..{upper}, {inclusive_lower}, {inclusive_upper}: {n}");
                    assert_eq!(validate(&Message { value: Timestamp { seconds: 0, nanos: i32::try_from(n)? } }, &timestamps).is_empty(), valid);
                    assert_eq!(validate(&Message { value: Duration { seconds: 0, nanos: i32::try_from(n)? } }, &durations).is_empty(), valid);
                }
                for seed in 0..16 {
                    let mut a = Message::default(); let mut b = Message::default(); let mut c = Message::default();
                    let ra = fill(&mut a, &integers, &mut Rng::new(seed));
                    let rb = fill(&mut b, &timestamps, &mut Rng::new(seed));
                    let rc = fill(&mut c, &durations, &mut Rng::new(seed));
                    if lower == upper && !(inclusive_lower && inclusive_upper) { assert!(ra.is_err()); assert!(rb.is_err()); assert!(rc.is_err()); }
                    else { ra?; rb?; rc?; assert_eq!(validate(&a, &integers), []); assert_eq!(validate(&b, &timestamps), []); assert_eq!(validate(&c, &durations), []); }
                }
            }
        }
    }
    Ok(())
}
