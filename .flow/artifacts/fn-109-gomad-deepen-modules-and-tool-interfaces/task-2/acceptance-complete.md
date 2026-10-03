- [x] Local and isolated execution consume the same normalized options value; no second hand-written field list exists for coordinator transport.
- [x] Callbacks, injected dependencies, resolved child commands and Runner identity are outside the serialized intent; campaign-plan and resume records keep their own schemas and bytes.
- [x] The characterization table yields identical selections, environments, error messages and precedence before and after; task 1's real isolated-execution tests still pass for all three strategies.
- [x] Unknown fields, trailing data and malformed coordinator requests remain rejected.
- [x] Exported CampaignSpec fields are unchanged.

Verified on darwin/arm64 with final source hashes, full host gate plus the scoped selector-fix rerun, and a fresh read-only SHIP review. Native linux/amd64 remains unverified and is required at fn-109.21 before overall spec completion.
