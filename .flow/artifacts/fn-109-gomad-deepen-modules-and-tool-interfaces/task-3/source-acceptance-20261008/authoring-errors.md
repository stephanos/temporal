The first helper capture used historical-reconstruction as both a command label and the generated proof basename. Reconstruction succeeded and wrote its proof; capture then refused to overwrite historical-reconstruction.json with its receipt (EEXIST). Its stdout and stderr remain retained. This was a helper naming error, not a Go test failure. Later capture labels use distinct basenames. No source or prior evidence was overwritten.

proof-authoring-first exited1 because the literal Failure completion mapping was authored as a single line, but the retained patch used a multiline literal. The exact retained multiline bytes corrected the helper; proof-authoring-second exited0. No production source or original evidence changed.

lint-attribution exited1 because function-name-only lookup confused public Replay with a receiver method. Exact signatures, then matching receiver presence, corrected this authoring ambiguity; lint-attribution-corrected exited0. No diagnostic was ignored.

coverage-migration-binding exited1 because the helper spread an array rather than its single selected function span. The corrected helper selects that exact element. These helper-authoring failures remain separate from actual test, lint and timeout outcomes.

coverage-migration-corrected exited1 because the preparation source path was mistakenly prefixed with runner/. The actual import is tools/gomad3/internal/preparation; the correction uses that real existing source. No host guard was changed.

diff-whitespace exited2 on root-owned task specification EOF blank lines. Root removed only its accidental extra EOF lines; final-diff-whitespace exited0. No worker edited Flow state.

completion-delta-binding exited1 on helper indentation mismatch: the authored matchers used one extra tab. The initial test patch also temporarily indented touched context lines one tab too deeply. The worker corrected only its own indentation; completion-delta-corrected proves exact additive transformation from the saved entire preimage, including byte-identical existing error assertions and all other functions. No assertion or trigger changed.

final-lint-preservation exited1 because completionCampaign has two identical switch lines in one function. The corrected additive-file proof compares the entire exact finding-owning function pre/post, then records the actual current line offset and context; it does not choose a match by a generic trimmed string.
