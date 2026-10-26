use umpire_dsl::Action;

#[derive(Action)]
enum User {
    Set(#[inputs(0..=2)] #[domain(sizes)] u8),
}

fn main() {}
