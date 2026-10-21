use umpire_temporal::realization;

struct Binding;

#[realization(activity)]
impl Binding {
    #[answer(PollActivityTaskQueueResponse, reads(input))]
    fn first(task: PollActivityTaskQueueResponse) -> Result<Reply<Action>, Refusal> { todo!() }
}

fn main() {}
