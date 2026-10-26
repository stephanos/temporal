use umpire_dsl::Action;

#[derive(Action)]
enum User {
    #[fault]
    Fail,
}

fn main() {}
