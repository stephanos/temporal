use umpire_reference_kettle::KettleServiceMethod;
fn classify(method: KettleServiceMethod) {
    match method {
        KettleServiceMethod::Heat | KettleServiceMethod::Pour | KettleServiceMethod::Describe
        | KettleServiceMethod::PollHeating | KettleServiceMethod::RespondHeating
        | KettleServiceMethod::ListKettles | KettleServiceMethod::ResetKettle => (),
    }
}
fn main() {}
