use umpire_dsl::Action;

fn sizes(_: &()) -> impl Iterator<Item = u8> {
    0..2
}

#[derive(Action)]
enum User {
    Set(#[domain(sizes)] u8),
}

fn main() {}
