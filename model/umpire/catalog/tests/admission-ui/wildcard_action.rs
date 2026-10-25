enum Action {
    Client(Client),
}
enum Client {
    Start,
    Added,
}
umpire_catalog::annotate! {
    pub catalog controls;
    actions bind: Action {
        Action::Client => { _ => Rpc("test.Service.Start"), },
    }
}
fn main() {}
