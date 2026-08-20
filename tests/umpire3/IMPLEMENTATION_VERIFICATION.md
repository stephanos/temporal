# Go implementation-verifier pilot

The M12 pilot evaluated the smallest candidate seam: deciding whether a returned task completion is
from the current ownership epoch. Its Lean contract is
`TaskDelivery.CurrentCompletion completionEpoch ownerEpoch`; Nexus and Update both consume the same
hashed guarantee, and exhaustive Go adapter tests exercise current, stale, and cancellation-won
cases.

Gobra was not adopted for 1.0. The candidate Go decision is embedded in stateful adapter code rather
than a stable production function, so a Gobra annotation would prove a test-harness transcription,
not a meaningful Temporal implementation-to-system seam. No implementation proof claim is made.
Reconsider after a production function owns this decision behind a stable interface; the pilot must
then identify exact code, concurrency assumptions, persistence assumptions, and the model theorem it
discharges. Grove/Goose remains out of scope because this seam did not expose a crash-logic gap that
justifies its higher proof cost.
