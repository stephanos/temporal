use umpire_dsl::Action;

#[derive(Action)]
enum User {
    #[metadata(example = "value")]
    Toggle,
}

fn main() {}
