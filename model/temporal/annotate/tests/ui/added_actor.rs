enum Action {
    Client(Client),
    Added,
}
enum Client {
    Start,
}
umpire_temporal_annotate::annotate! {
    pub catalog controls;
    service "test.Service" { Start: Action("Client::Start"); }
    actions bind: Action {
        Action::Client => { Client::Start => Rpc("test.Service.Start"), },
    }
}
fn main() {}
