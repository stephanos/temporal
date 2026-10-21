use super::{Site, Symbol, Symbols};
use crate::evidence::inspection::Choices;
use gomad_wasmhost::linked;
use serde::Serialize;

#[derive(Serialize)]
struct Symbolized<'a, T> {
    #[serde(flatten)]
    value: T,
    symbol: &'a Symbol,
}

#[derive(Serialize)]
pub(crate) struct Trace<'a> {
    records: u64,
    decisions: u64,
    branching_records: u64,
    runnable: u64,
    select_poll: u64,
    select_result: u64,
    sites: Vec<Symbolized<'a, linked::TraceSite>>,
    replay_decisions: Vec<Symbolized<'a, linked::TraceDecision>>,
    tape_sha256: String,
}

pub(crate) fn choice_sites(choices: &Choices) -> impl Iterator<Item = Site> + '_ {
    choices
        .trace
        .sites
        .iter()
        .map(|site| Site {
            value: site.site_offset,
            missing: site.site_missing,
        })
        .chain(choices.trace.replay_decisions.iter().map(|decision| Site {
            value: decision.site_offset,
            missing: decision.site_missing,
        }))
}

pub(crate) fn symbolize_choices(choices: Choices, symbols: &Symbols) -> Choices<Trace<'_>> {
    let trace = choices.trace;
    Choices {
        schema: choices.schema,
        profile: choices.profile,
        implementation_sha256: choices.implementation_sha256,
        limit: choices.limit,
        payload_bytes: choices.payload_bytes,
        sha256: choices.sha256,
        terminal_state: choices.terminal_state,
        exact_replay_available: choices.exact_replay_available,
        trace: Trace {
            records: trace.records,
            decisions: trace.decisions,
            branching_records: trace.branching_records,
            runnable: trace.runnable,
            select_poll: trace.select_poll,
            select_result: trace.select_result,
            sites: trace
                .sites
                .into_iter()
                .map(|value| Symbolized {
                    symbol: symbols.get(Site {
                        value: value.site_offset,
                        missing: value.site_missing,
                    }),
                    value,
                })
                .collect(),
            replay_decisions: trace
                .replay_decisions
                .into_iter()
                .map(|value| Symbolized {
                    symbol: symbols.get(Site {
                        value: value.site_offset,
                        missing: value.site_missing,
                    }),
                    value,
                })
                .collect(),
            tape_sha256: trace.tape_sha256,
        },
    }
}
