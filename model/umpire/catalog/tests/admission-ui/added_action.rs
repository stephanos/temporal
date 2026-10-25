enum Action {
    Client(Client),
}
enum Client {
    Start,
    Added,
}
umpire_catalog::annotate! {
    pub catalog controls;
    service "test.Service" { Start: Action("Client::Start"); }
    actions bind: Action {
        Action::Client => { Client::Start => Rpc("test.Service.Start"), },
    }
}
fn main() {}
