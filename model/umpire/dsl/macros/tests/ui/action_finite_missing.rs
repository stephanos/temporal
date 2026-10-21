use umpire_dsl::Action;

#[derive(Clone, Debug, PartialEq, Eq, Hash)]
struct Opaque;

#[derive(Action)]
enum User {
    Set(Opaque),
}

fn main() {}
