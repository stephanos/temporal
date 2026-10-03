- [x] Explore and portable planning obtain a validated prepared target with adapter identities attached from one preparation owner; neither attaches adapters nor orders validation itself.
- [x] Implementation-only workspace cleanup is owned by the module, and cleanup failure is reported; campaign and bundle owners keep their durable destinations and journal transitions.
- [x] Tests through the new interface cover fresh and cache builds, an external module with a local replacement, a custom preparer and two independent preparations of the same target.
- [x] Invalid sums, replacement conflicts, changed binaries and cleanup failures keep their existing classifications and `HostError` reasons, including cancellation and overall-timeout reasons.
- [x] The new package has a registered architectural owner and introduces no import cycle; fixed-input `target.Prepared` values match the pre-change values.

