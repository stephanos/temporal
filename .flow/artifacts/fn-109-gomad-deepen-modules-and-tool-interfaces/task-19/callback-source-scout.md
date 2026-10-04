# Callback bindings for targeted effect analysis

Read-only follow-up to the architecture fitness scout, requested as Codex thinking scout gpt-6.1-sol/high. For this bounded assignment, judge first rejected an acceptance array; the conductor corrected its schema to a string and obtained `Tier: session (jev-unavailable(no_key))`. Explicit model routing stayed unchanged. The scout ran no tests/builds/generation and changed no files/Flow/history. Findings are source bindings and analysis requirements, not a completed transitive checker proof or native acceptance.

## Current production bindings

- SeedController: `runner/internal/campaign/controller.go:100` stores config.Next and :122 invokes it. The sole production constructor binding is the closure in `runner/campaign.go:57`, reached from `runner/runner.go:682`. It calls concrete pendingJobs.Next, concrete SeedIterator.Next (`runner/seeds.go:180`), normalizedCampaignShard (`campaign_shard.go:34`) and CampaignShard.Owns (:23). These mutate counters, inspect ranges/maps and calculate ordinal ownership. No further callback/interface dispatch or recursion appears in this chain. The effectful completion-channel loop in the same source file is an unrelated sibling, not a reachable function.
- NextRound: exactly two production instantiations, `runner/internal/exploration/choice/engine.go:186` and `simulation/frontier.go:216`, each passing named cloneCandidate. The choice clone at engine.go:647 copies PrefixBytes with append. The simulation clone at frontier.go:870 calls cloneForcedDecisions, allocates a slice and copies each Control byte slice. No host effects on these inspected bindings.
- SumBytes: exactly two production instantiations, choice/engine.go:608 and simulation/frontier.go:843, passing candidateBytes at :599 and :834. Both call canonicaljson.CanonicalJSON and return encoded length (choice additionally adds one). Generic SumBytes must carry the supplied size callback's effects, not receive an unconditional purity exemption.

Additional search matches are tests; the scout found no other production assignment or alias of the seed callback. Unknown-parameter callbacks remain a fixture case, not an unexplained current production binding.

## Concrete serialization paths

Choice Candidate contains SHA256 strings, integer depth and byte slices. Simulation Candidate contains SHA256 strings and ForcedDecision slices; each decision contains strings, integers and bytes. Candidate, ForcedDecision, Dimension and record.SHA256 have no JSON/text marshal methods. Choice Candidate's PrefixReplayPlan is unrelated. record.Uint64String.MarshalJSON exists but these candidate fields use ordinary uint64/uint32 and do not reach that method.

`internal/canonicaljson/canonical.go:16` validates strings, encodes to a concrete bytes.Buffer, decodes from a concrete bytes.Reader and canonicalizes to another buffer. validateStrings, consumeJSONValue and appendCanonical recurse over in-memory data; recursive analysis needs a fixed point, not an automatic purity/unknown decision.

Pinned Go 1.27.1 source evidence (the same GOROOT identified in source-scout.md):

- encoding/json/stream.go:206 stores the encoder writer; :245 invokes its Write. Here the receiver resolves to bytes.Buffer.Write (bytes/buffer.go:193), which grows/copies memory.
- json.Decoder refill at stream.go:179 invokes the actual reader, here bytes.Reader.
- json newTypeEncoder at encode.go:420 checks both value and pointer marshal method sets. A schema change can add an effectful method; the checker must inspect the actual value graph rather than bless CanonicalJSON globally.
- fmt.handleMethods at fmt/print.go:614 can invoke Error or Format even for %w. Preserve error provenance in exact formatting summaries.

## Minimal analysis obligations

NextRound[C] carries its clone argument's effect. SumBytes[C] carries its size argument's effect. SeedController.Next carries the closure stored through config.Next -> controller.next; track captures and the returned controller value. JSON encoding carries concrete writer and applicable marshal-method effects; decoding carries concrete reader and applicable unmarshal-method effects. Preserve concrete types across any conversions. Solve recursive call components to a fixed point. No generic helper, time/fmt/json package or unknown effect-bearing callback is unconditionally exempt.

Checker controls should accept an in-memory captured iterator and reject an effectful stored closure; accept a pure named/method-value clone through generic wrappers and reject an effectful helper; accept ordinary sizing errors and reject host-clock sizing; accept JSON into bytes.Buffer and reject os.File; inspect nested value/pointer MarshalJSON/MarshalText effects; reject or explicitly report unknown callback parameters; accept local-data recursion and reject one effectful branch in that component; keep unrelated effectful siblings outside a named root's reachable graph. Existing exploration/engine_test.go:10 has behavioral clone/size examples, including sizing error, but task 19 still needs actual checker assertions.

Writing-for-agents influenced placement: this evidence is disclosed beside the original checker design, so callback handling and its precise controls are available together without adding broad package exemptions or duplicating instructions in always-loaded AGENTS.md. Task 19 remains unstarted and follows tasks 17–18.
