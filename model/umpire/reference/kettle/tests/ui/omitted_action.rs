use umpire_reference_kettle::kettle::{Action, User};
fn bind(action: Action) {
    match action {
        Action::User(User::PressButton) | Action::Heater(_) => (),
    }
}
fn main() {}
