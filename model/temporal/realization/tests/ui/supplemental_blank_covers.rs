#[macro_use]
#[path = "../../../features/activity/api_requests.rs"]
mod requests;

fn main() {
    let _catalog = supplemental_request!(
        uuid String, "Request", "Request.request_id",
        get = |message| message, set = |message| message,
        expected = umpire_annotations::Expectation::Accepts,
        covers = " \t\n",
    );
    let _catalog = supplemental_request!(
        uuid String, "Request", "Request.request_id",
        get = |message| message, set = |message| message,
        expected = umpire_annotations::Expectation::Accepts,
        covers = "\u{2003}",
    );
}
