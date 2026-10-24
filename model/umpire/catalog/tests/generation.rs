//! Regression controls for generation over scalar presence and interacting constraints.

use umpire_catalog::*;

#[derive(Clone, Default)]
struct Message<T> {
    value: T,
}

#[derive(Clone, Default, PartialEq, Eq)]
struct Stuck(i32);

impl FieldValue for Stuck {
    fn supports(kind: &Kind) -> bool {
        matches!(kind, Kind::Counter(_))
    }
    fn present(&self) -> bool {
        self.0 != 0
    }
    fn violations(&self, kind: &Kind) -> Vec<&'static str> {
        self.0.violations(kind)
    }
    fn generate(_kind: &Kind, _rng: &mut Rng) -> Result<Self, GenerationError> {
        Ok(Self(1))
    }
}

fn catalog<T: FieldValue>(
    kind: Kind,
    repeated: Option<RepeatedRules>,
    presence: Option<Presence>,
) -> Result<Catalog<Message<T>>, AdmissionErrors> {
    let mut field = Field::new("value", kind, |m: &Message<T>| &m.value, |m| &mut m.value);
    if let Some(rules) = repeated {
        field = field.repeated(rules);
    }
    let mut builder = Catalog::response("example.Message").field(field);
    if let Some(presence) = presence {
        builder = builder.rule(Rule { presence, expected: None });
    }
    builder.build()
}

#[test]
fn presence_filling_uses_present_scalar_candidates() -> Result<(), Box<dyn std::error::Error>> {
    for presence in
        [Presence::Required(vec!["value".into()]), Presence::AnyOf(vec!["value".into()])]
    {
        let strings =
            catalog::<String>(Kind::Text(StringRules::default()), None, Some(presence.clone()))?;
        let counters = catalog::<i32>(
            Kind::Counter(NumberRules { gte: Some(0), lte: Some(1), ..Default::default() }),
            None,
            Some(presence.clone()),
        )?;
        let vectors = catalog::<Vec<i32>>(
            Kind::Counter(NumberRules::default()),
            Some(RepeatedRules { min_items: Some(0), max_items: Some(1), unique: false }),
            Some(presence),
        )?;
        for seed in 0..16 {
            let mut text = Message::default();
            fill(&mut text, &strings, &mut Rng::new(seed))?;
            assert_ne!(text.value, "");
            assert_eq!(validate(&text, &strings), []);
            let mut number = Message::default();
            fill(&mut number, &counters, &mut Rng::new(seed))?;
            assert_eq!(number.value, 1);
            assert_eq!(validate(&number, &counters), []);
            let mut vector = Message::default();
            fill(&mut vector, &vectors, &mut Rng::new(seed))?;
            assert_eq!(vector.value.len(), 1);
            assert_eq!(validate(&vector, &vectors), []);
        }
    }
    let impossible = catalog::<i32>(
        Kind::Counter(NumberRules { gte: Some(0), lte: Some(0), ..Default::default() }),
        None,
        Some(Presence::Required(vec!["value".into()])),
    )?;
    assert!(fill(&mut Message::default(), &impossible, &mut Rng::new(0)).is_err());
    Ok(())
}

#[test]
fn generated_text_satisfies_byte_and_scalar_bounds_together()
-> Result<(), Box<dyn std::error::Error>> {
    for (characters, bytes, prefix, possible) in [
        (1, 1, "", true),
        (1, 4, "", true),
        (2, 3, "é", true),
        (2, 5, "🙂", true),
        (2, 1, "", false),
        (1, 5, "", false),
        (1, 1, "é", false),
    ] {
        let kind = Kind::Text(StringRules {
            min_len: Some(characters),
            max_len: Some(characters),
            min_bytes: Some(bytes),
            max_bytes: Some(bytes),
            prefix: Some(prefix.into()),
            ..Default::default()
        });
        for seed in 0..8 {
            let result = String::generate(&kind, &mut Rng::new(seed));
            if possible {
                let value = result?;
                assert_eq!(value.chars().count(), characters);
                assert_eq!(value.len(), bytes);
                assert!(value.starts_with(prefix));
                assert_eq!(value.violations(&kind), Vec::<&str>::new());
                assert_eq!(value, String::generate(&kind, &mut Rng::new(seed))?);
            } else {
                assert_eq!(result, Err(GenerationError::ImpossibleConstraints));
            }
        }
    }
    let kind = Kind::Text(StringRules {
        min_bytes: Some(100),
        max_bytes: Some(100),
        ..Default::default()
    });
    assert_eq!(String::generate(&kind, &mut Rng::new(0))?.len(), 100);
    let kind =
        Kind::Text(StringRules { min_bytes: Some(4), max_bytes: Some(3), ..Default::default() });
    assert_eq!(
        String::generate(&kind, &mut Rng::new(0)),
        Err(GenerationError::ImpossibleConstraints)
    );
    Ok(())
}

#[test]
fn required_substring_can_be_supplied_by_the_suffix() -> Result<(), Box<dyn std::error::Error>> {
    for prefix in [None, Some("a".to_owned())] {
        let catalog = catalog::<String>(
            Kind::Text(StringRules {
                prefix: prefix.clone(),
                contains: Some(if prefix.is_some() { "c" } else { "b" }.into()),
                suffix: Some("abc".into()),
                max_len: Some(3),
                ..Default::default()
            }),
            None,
            None,
        )?;
        assert_eq!(validate(&Message { value: "abc".into() }, &catalog), []);
        let mut message = Message::default();
        fill(&mut message, &catalog, &mut Rng::new(1))?;
        assert_eq!(message.value, "abc");
    }
    Ok(())
}

#[test]
fn exact_length_anchors_preserve_partial_overlap() -> Result<(), Box<dyn std::error::Error>> {
    for (prefix, suffix, contains, expected) in [
        ("ab", "bc", "", "abbc"),
        ("αβ", "βγ", "", "αββγ"),
        ("ab", "bc", "bb", "abbc"),
        ("ab", "bc", "ab", "abbc"),
    ] {
        let catalog = catalog::<String>(
            Kind::Text(StringRules {
                prefix: Some(prefix.into()),
                suffix: Some(suffix.into()),
                contains: Some(contains.into()),
                min_len: Some(4),
                max_len: Some(4),
                ..Default::default()
            }),
            None,
            None,
        )?;
        assert_eq!(validate(&Message { value: expected.into() }, &catalog), []);
        for seed in 0..8 {
            let mut a = Message::default();
            let mut b = Message::default();
            fill(&mut a, &catalog, &mut Rng::new(seed))?;
            fill(&mut b, &catalog, &mut Rng::new(seed))?;
            assert_eq!(a.value, expected);
            assert_eq!(a.value, b.value);
            assert_eq!(validate(&a, &catalog), []);
        }
    }
    let ranged = catalog::<String>(
        Kind::Text(StringRules {
            prefix: Some("a".into()),
            suffix: Some("a".into()),
            contains: Some("b".into()),
            min_len: Some(1),
            max_len: Some(3),
            ..Default::default()
        }),
        None,
        None,
    )?;
    for seed in 0..8 {
        let mut message = Message::default();
        fill(&mut message, &ranged, &mut Rng::new(seed))?;
        assert_eq!(message.value, "aba");
    }
    Ok(())
}

#[test]
fn anchored_generation_matches_small_exhaustive_string_domains()
-> Result<(), Box<dyn std::error::Error>> {
    let anchors = ["", "a", "b", "aa", "ab", "ba", "bb"];
    for len in 0..=5 {
        let words: Vec<String> = (0..(1_usize << len))
            .map(|bits| (0..len).map(|i| if bits & (1 << i) == 0 { 'a' } else { 'b' }).collect())
            .collect();
        for prefix in anchors {
            for suffix in anchors {
                for contains in anchors {
                    let possible = words.iter().any(|word| {
                        word.starts_with(prefix)
                            && word.ends_with(suffix)
                            && word.contains(contains)
                    });
                    let catalog = catalog::<String>(
                        Kind::Text(StringRules {
                            prefix: Some(prefix.into()),
                            suffix: Some(suffix.into()),
                            contains: Some(contains.into()),
                            min_len: Some(len),
                            max_len: Some(len),
                            ..Default::default()
                        }),
                        None,
                        None,
                    )?;
                    for seed in 0..3 {
                        let mut message = Message::default();
                        let result = fill(&mut message, &catalog, &mut Rng::new(seed));
                        assert_eq!(
                            result.is_ok(),
                            possible,
                            "len={len}, prefix={prefix:?}, suffix={suffix:?}, contains={contains:?}, seed={seed}: {result:?}"
                        );
                        if possible {
                            assert_eq!(validate(&message, &catalog), []);
                        }
                    }
                }
            }
        }
    }
    Ok(())
}

#[test]
fn unique_repeated_times_sample_the_allowed_domain() -> Result<(), Box<dyn std::error::Error>> {
    let rules = NumberRules { gte: Some(0), lte: Some(1), ..Default::default() };
    let repeated = RepeatedRules { min_items: Some(2), max_items: Some(2), unique: true };
    let durations =
        catalog::<Vec<Duration>>(Kind::Duration(rules.clone()), Some(repeated.clone()), None)?;
    let timestamps = catalog::<Vec<Timestamp>>(Kind::Timestamp(rules), Some(repeated), None)?;
    for seed in 0..16 {
        let mut a = Message::default();
        let mut b = Message::default();
        fill(&mut a, &durations, &mut Rng::new(seed))?;
        fill(&mut b, &durations, &mut Rng::new(seed))?;
        assert_eq!(a.value, b.value);
        assert!(a.value.contains(&Duration { seconds: 0, nanos: 0 }));
        assert!(a.value.contains(&Duration { seconds: 0, nanos: 1 }));
        assert_eq!(validate(&a, &durations), []);
        let mut a = Message::default();
        let mut b = Message::default();
        fill(&mut a, &timestamps, &mut Rng::new(seed))?;
        fill(&mut b, &timestamps, &mut Rng::new(seed))?;
        assert_eq!(a.value, b.value);
        assert!(a.value.contains(&Timestamp { seconds: 0, nanos: 0 }));
        assert!(a.value.contains(&Timestamp { seconds: 0, nanos: 1 }));
        assert_eq!(validate(&a, &timestamps), []);
    }
    let repeated = RepeatedRules { min_items: Some(2), max_items: Some(2), unique: true };
    let strings = catalog::<Vec<String>>(
        Kind::Text(StringRules {
            prefix: Some("x".into()),
            max_len: Some(2),
            ..Default::default()
        }),
        Some(repeated.clone()),
        None,
    )?;
    let bytes = catalog::<Vec<Vec<u8>>>(
        Kind::Bytes(BytesRules { min_len: Some(0), max_len: Some(1) }),
        Some(repeated.clone()),
        None,
    )?;
    let enums = catalog::<Vec<i32>>(
        Kind::Enum(EnumRules { defined_only: false, values: vec![0] }),
        Some(repeated),
        None,
    )?;
    let anchored = catalog::<Vec<String>>(
        Kind::Text(StringRules {
            prefix: Some("a".into()),
            suffix: Some("a".into()),
            contains: Some("aba".into()),
            min_len: Some(4),
            max_len: Some(4),
            ..Default::default()
        }),
        Some(RepeatedRules { min_items: Some(2), max_items: Some(2), unique: true }),
        None,
    )?;
    let mut variants = Message::default();
    fill(&mut variants, &anchored, &mut Rng::new(0))?;
    assert!(variants.value.iter().any(|value| value == "aaba"));
    assert!(variants.value.iter().any(|value| value == "abaa"));
    assert_eq!(validate(&variants, &anchored), []);
    let mut text = Message::default();
    let mut data = Message::default();
    let mut values = Message::default();
    let results = [
        fill(&mut text, &strings, &mut Rng::new(0)),
        fill(&mut data, &bytes, &mut Rng::new(0)),
        fill(&mut values, &enums, &mut Rng::new(0)),
    ];
    assert!(results.iter().all(Result::is_ok), "{results:?}");
    assert_eq!(validate(&text, &strings), []);
    assert_eq!(validate(&data, &bytes), []);
    assert_eq!(validate(&values, &enums), []);
    Ok(())
}

#[test]
fn unique_strings_use_the_permitted_scalar_and_byte_domains()
-> Result<(), Box<dyn std::error::Error>> {
    let repeated = RepeatedRules { min_items: Some(27), max_items: Some(27), unique: true };
    let texts = catalog::<Vec<String>>(
        Kind::Text(StringRules { min_len: Some(1), max_len: Some(1), ..Default::default() }),
        Some(repeated.clone()),
        None,
    )?;
    let random = catalog::<Vec<String>>(
        Kind::Random(BytesRules { min_len: Some(1), max_len: Some(1) }),
        Some(repeated),
        None,
    )?;
    let witnesses = Message {
        value: (0..27).map(|i| char::from_u32(0x400 + i).unwrap_or('x').to_string()).collect(),
    };
    assert_eq!(validate(&witnesses, &texts), []);
    let witnesses = Message { value: (0..27).map(|i| char::from(i).to_string()).collect() };
    assert_eq!(validate(&witnesses, &random), []);
    let mut text = Message::default();
    let mut bytes = Message::default();
    let results =
        [fill(&mut text, &texts, &mut Rng::new(0)), fill(&mut bytes, &random, &mut Rng::new(0))];
    assert!(results.iter().all(Result::is_ok), "{results:?}");
    assert_eq!(text.value.len(), 27);
    assert_eq!(bytes.value.len(), 27);
    assert!(text.value.iter().any(|value| !value.is_ascii()));
    assert!(bytes.value.iter().all(|value| value.len() == 1));
    assert_eq!(validate(&text, &texts), []);
    assert_eq!(validate(&bytes, &random), []);
    let mut multibyte = false;
    for len in 0..=8 {
        let catalog = catalog::<String>(
            Kind::Random(BytesRules { min_len: Some(len), max_len: Some(len) }),
            None,
            None,
        )?;
        for seed in 0..8 {
            let mut a = Message::default();
            let mut b = Message::default();
            fill(&mut a, &catalog, &mut Rng::new(seed))?;
            fill(&mut b, &catalog, &mut Rng::new(seed))?;
            assert_eq!(a.value.len(), len);
            assert_eq!(a.value, b.value);
            assert_eq!(validate(&a, &catalog), []);
            multibyte |= !a.value.is_ascii();
        }
    }
    assert!(multibyte);
    Ok(())
}

#[test]
fn bounded_unique_sampling_refuses_without_claiming_unsatisfiability()
-> Result<(), Box<dyn std::error::Error>> {
    fn limit<T: FieldValue>(kind: Kind, count: usize) -> Result<(), Box<dyn std::error::Error>> {
        let catalog = catalog::<Vec<T>>(
            kind,
            Some(RepeatedRules { min_items: Some(count), max_items: Some(count), unique: true }),
            None,
        )?;
        assert!(matches!(
            fill(&mut Message::default(), &catalog, &mut Rng::new(0)),
            Err(FillError::Field { reason: GenerationError::SizeLimit, .. })
        ));
        Ok(())
    }
    limit::<String>(
        Kind::Text(StringRules {
            min_len: Some(1),
            max_len: Some(1),
            prefix: Some("x".into()),
            ..Default::default()
        }),
        2,
    )?;
    limit::<Vec<u8>>(Kind::Bytes(BytesRules { min_len: Some(1), max_len: Some(1) }), 257)?;
    limit::<i32>(Kind::Enum(EnumRules { defined_only: true, values: vec![1, 1] }), 2)?;
    let zero = NumberRules { gte: Some(0), lte: Some(0), ..Default::default() };
    limit::<i32>(Kind::Counter(zero.clone()), 2)?;
    limit::<Timestamp>(Kind::Timestamp(zero.clone()), 2)?;
    limit::<Duration>(Kind::Duration(zero), 2)?;
    let catalog = catalog::<Vec<Stuck>>(
        Kind::Counter(NumberRules::default()),
        Some(RepeatedRules { min_items: Some(2), max_items: Some(2), unique: true }),
        None,
    )?;
    assert_eq!(validate(&Message { value: vec![Stuck(1), Stuck(2)] }, &catalog), []);
    assert!(matches!(
        fill(&mut Message::default(), &catalog, &mut Rng::new(0)),
        Err(FillError::Field { reason: GenerationError::SizeLimit, .. })
    ));
    Ok(())
}

#[test]
fn zero_maximum_repeated_length_is_fillable() -> Result<(), Box<dyn std::error::Error>> {
    for min_items in [None, Some(0), Some(1)] {
        let catalog = catalog::<Vec<i32>>(
            Kind::Counter(NumberRules::default()),
            Some(RepeatedRules { min_items, max_items: Some(0), unique: false }),
            None,
        )?;
        let mut message = Message::default();
        let result = fill(&mut message, &catalog, &mut Rng::new(1));
        if min_items == Some(1) {
            assert!(matches!(
                result,
                Err(FillError::Field { reason: GenerationError::ImpossibleConstraints, .. })
            ));
        } else {
            result?;
            assert_eq!(message.value, []);
            assert_eq!(validate(&message, &catalog), []);
        }
    }
    Ok(())
}

#[test]
fn numeric_range_semantics_match_protovalidate_endpoint_combinations()
-> Result<(), Box<dyn std::error::Error>> {
    for inclusive_lower in [false, true] {
        for inclusive_upper in [false, true] {
            for (lower, upper) in [(5, 10), (10, 5), (5, 5)] {
                let rules = NumberRules {
                    gt: (!inclusive_lower).then_some(lower),
                    gte: inclusive_lower.then_some(lower),
                    lt: (!inclusive_upper).then_some(upper),
                    lte: inclusive_upper.then_some(upper),
                };
                let integers = catalog::<i32>(Kind::Counter(rules.clone()), None, None)?;
                let timestamps = catalog::<Timestamp>(Kind::Timestamp(rules.clone()), None, None)?;
                let durations = catalog::<Duration>(Kind::Duration(rules), None, None)?;
                for n in 0..=15 {
                    let low = if inclusive_lower { n >= lower } else { n > lower };
                    let high = if inclusive_upper { n <= upper } else { n < upper };
                    let valid = if lower > upper { low || high } else { low && high };
                    assert_eq!(
                        validate(&Message { value: i32::try_from(n)? }, &integers).is_empty(),
                        valid,
                        "{lower}..{upper}, {inclusive_lower}, {inclusive_upper}: {n}"
                    );
                    assert_eq!(
                        validate(
                            &Message { value: Timestamp { seconds: 0, nanos: i32::try_from(n)? } },
                            &timestamps
                        )
                        .is_empty(),
                        valid
                    );
                    assert_eq!(
                        validate(
                            &Message { value: Duration { seconds: 0, nanos: i32::try_from(n)? } },
                            &durations
                        )
                        .is_empty(),
                        valid
                    );
                }
                for seed in 0..16 {
                    let mut a = Message::default();
                    let mut b = Message::default();
                    let mut c = Message::default();
                    let ra = fill(&mut a, &integers, &mut Rng::new(seed));
                    let rb = fill(&mut b, &timestamps, &mut Rng::new(seed));
                    let rc = fill(&mut c, &durations, &mut Rng::new(seed));
                    if lower == upper && !(inclusive_lower && inclusive_upper) {
                        assert!(ra.is_err());
                        assert!(rb.is_err());
                        assert!(rc.is_err());
                    } else {
                        ra?;
                        rb?;
                        rc?;
                        assert_eq!(validate(&a, &integers), []);
                        assert_eq!(validate(&b, &timestamps), []);
                        assert_eq!(validate(&c, &durations), []);
                    }
                }
            }
        }
    }
    let edges = catalog::<Vec<u64>>(
        Kind::Counter(NumberRules {
            gte: Some(i128::from(u64::MAX)),
            lte: Some(0),
            ..Default::default()
        }),
        Some(RepeatedRules { min_items: Some(2), max_items: Some(2), unique: true }),
        None,
    )?;
    for seed in 0..16 {
        let mut message = Message::default();
        fill(&mut message, &edges, &mut Rng::new(seed))?;
        assert!(message.value.contains(&0));
        assert!(message.value.contains(&u64::MAX));
        assert_eq!(validate(&message, &edges), []);
    }
    let clipped = catalog::<i64>(
        Kind::Counter(NumberRules { gt: Some(i128::MAX), lt: Some(0), ..Default::default() }),
        None,
        None,
    )?;
    assert_eq!(validate(&Message { value: -1 }, &clipped), []);
    assert_ne!(validate(&Message { value: 0 }, &clipped), []);
    let mut message = Message::default();
    fill(&mut message, &clipped, &mut Rng::new(0))?;
    assert!(message.value < 0);
    assert_eq!(validate(&message, &clipped), []);
    Ok(())
}
