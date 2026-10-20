# fn-217.2 feedback-loop evidence

The worker measured the actual parser edit graph, not a facade-only edit:
`temporal/annotate/syntax/src/parse.rs`, followed by
`cargo test -p umpire-temporal-annotate --quiet` under the pinned mise toolchain.
The warm full gate was 15.494 s against the 30 s soft target. The parser-edit
loop was 17.825 s against the 10 s soft target. The timing command exited 0;
the inner-loop budget was not met.

The conductor requested investigation because this is a substantial overrun
under fn-210 R30. A repeat of the same parser-touch workload, adding only Cargo's
`--timings` reporting, took 17.556 s and exited 0. All 21 refusal fixtures and
the actual generated-API pass control remained selected.

| Measured component | Time |
| --- | ---: |
| Outer Cargo build, 114 fresh / 10 dirty units | 4.2 s |
| API build-script compilation | 0.64 s |
| API build-script execution, including Go linking/checking/prost | 0.54 s |
| Generated API Rust compilation | 2.08 s |
| UI test total | 13.08 s |
| Dependency build within UI total | 6.29 s |

The nested dependency build is included in UI time, not additive to it. The
dominant block is the UI runner and its second dependency build. The complete
outer build-script execution is only 0.54 s, so attributing the overrun to Go
linking would be unsupported. The generated API is compiled in both the outer
and trybuild graphs; reducing that coupling without removing real-API controls
is the identified optimization opportunity.

Source stayed frozen at `16f113317b10899fd47a54b55e746772c301e830` and the lane
remained clean. Evidence is retained in `.flow/tmp/fn2172-macro/model-time.log`,
`parser-profile.log` and `parser-profile-cargo-timing.html`. No benchmark subset,
assertion or budget was changed. R30 makes the targets soft and requires
investigation of substantial/repeated overruns; that investigation is recorded
here, not represented as a timing-budget pass.
