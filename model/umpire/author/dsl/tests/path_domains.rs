//! Wide finite input ranges retain their exact values without eager enumeration.

use umpire_dsl::{ActionDomain, Gap, InputDomain, Path, Pattern, PatternError, Waypoint};

#[test]
fn empty_alternatives_still_require_storage_admission() {
    struct Bomb;
    impl std::fmt::Display for Bomb {
        fn fmt(&self, _: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
            panic!("storage admission must happen before labeling alternatives")
        }
    }
    let pattern: Path<u8, Bomb> = Path::pattern(Pattern::Repeat(
        Box::new(Pattern::Repeat(
            Box::new(Pattern::action(ActionDomain::new(Ok(1), |_| Bomb), None, None)),
            0,
            0,
        )),
        0,
        1_000_001,
    ));
    assert_eq!(pattern.alternative_count(), Ok(1_000_002));
    assert!(matches!(pattern.alternatives(1_000_002), Err(PatternError::ExpansionTooLarge { .. })));
}

#[test]
fn repeated_domain_choices_do_not_describe_every_occurrence_as_its_first_value() {
    let domain = ActionDomain::new(Ok(2), |index| [0u8, 1][index]);
    let pattern: Path<u8, u8> =
        Path::pattern(Pattern::Repeat(Box::new(Pattern::action(domain, None, None)), 2, 2));
    let alternatives = pattern.alternatives(4).unwrap_or_else(|error| panic!("{error}"));
    assert_eq!(alternatives[1].choices, ["[one of 2 actions; 2]", "0", "1"]);
}

#[test]
fn state_path_case_budget_is_checked_before_cloning_waypoints() {
    struct CannotClone;
    impl Clone for CannotClone {
        fn clone(&self) -> Self {
            panic!("case admission must happen before cloning")
        }
    }
    let pattern: Path<CannotClone> =
        Waypoint::Is(CannotClone).then(Gap::Any, Waypoint::Holds(true));
    assert!(matches!(pattern.alternatives(0), Err(PatternError::Budget { count: 1, budget: 0 })));
}

#[test]
fn wide_ranges_keep_boundary_values_and_refuse_unrepresentable_counts() {
    assert_eq!(InputDomain::range(0u8..=u8::MAX).count(), Ok(256));
    let unsigned = InputDomain::range(u128::MAX - 1..=u128::MAX);
    assert_eq!(unsigned.count(), Ok(2));
    assert_eq!(unsigned.get(0), u128::MAX - 1);
    assert_eq!(unsigned.get(1), u128::MAX);
    let signed = InputDomain::range(i128::MIN..=i128::MIN + 1);
    assert_eq!(signed.count(), Ok(2));
    assert_eq!(signed.get(0), i128::MIN);
    assert_eq!(signed.get(1), i128::MIN + 1);
    assert_eq!(
        InputDomain::range(i128::MIN..=i128::MAX).count(),
        Err(PatternError::ExpansionOverflow)
    );
    assert_eq!(InputDomain::range(0u128..=u128::MAX).count(), Err(PatternError::ExpansionOverflow));
}
