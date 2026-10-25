use umpire_dsl::Action;

#[derive(Action)]
#[metadata(example(enum_value))]
enum User {
    Toggle(#[metadata(example(field_value))] bool),
}

fn main() {}
