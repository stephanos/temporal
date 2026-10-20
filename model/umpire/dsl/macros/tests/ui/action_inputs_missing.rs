use umpire_dsl::Action;

#[derive(Action)]
enum User {
    Set { value: u8 },
}

fn main() {}
