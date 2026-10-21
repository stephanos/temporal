#[macro_use]
#[path = "../../../feature/activity/api_requests.rs"]
mod requests;

fn main() {
    let _catalog = supplemental_request!(
        required Option<Vec<u8>>, "Failure", "Failure.info", "info",
        get = |message: &Option<Vec<u8>>| message.clone(),
        set = |message: &mut Option<Vec<u8>>, value| *message = value,
        covers = "server.RespondFailed",
    );
}
