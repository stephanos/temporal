use umpire_dsl::Finite;

#[derive(Finite)]
enum Choice {
    Some(bool),
}

#[derive(Finite)]
struct Wide(#[inputs(0..=300)] u8);

#[derive(Finite)]
struct Text(#[inputs(0..=3)] String);

#[derive(Finite)]
struct Plain(u8);

fn main() {}
