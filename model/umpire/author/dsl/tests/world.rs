//! Declared constants preserve field order, authored visibility and conditional value bounds.

#![forbid(unsafe_code)]

#[derive(Clone, Debug, PartialEq, Eq, Hash)]
struct Owned(String);

#[derive(umpire_dsl::World)]
struct Constants<T> {
    initial: T,
    limit: u8,
}

#[test]
fn constructor_and_value_traits_do_not_require_copy_or_default() {
    let world = Constants::new(Owned("initial".into()), 3);
    let copy = world.clone();
    assert_eq!(world, copy);
    assert_eq!(world.initial.0, "initial");
    assert_eq!(world.limit, 3);
    assert_eq!(world.to_string(), "Constants { initial: Owned(\"initial\"), limit: 3 }");
    let mut values = std::collections::HashSet::new();
    assert!(values.insert(world));
    assert!(!values.insert(copy));
}
