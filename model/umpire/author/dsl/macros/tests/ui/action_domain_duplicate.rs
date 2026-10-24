use umpire_dsl::Action;

#[derive(Action)]
enum User {
    Set(#[domain(sizes)] #[domain(other)] u8),
}

fn main() {}
