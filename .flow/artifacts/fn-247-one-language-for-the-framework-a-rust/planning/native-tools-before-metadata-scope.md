## Root follow-up: actual before-metadata context

One offline, locked metadata command succeeded at `5aed4c237c` (137 packages, 34 workspace members, 38 Cargo-source hashes unchanged). Files are `.flow/tmp/fn2472-cargo-before-{metadata,inventory,source-hashes}.json`. Its all-platform normal-plus-build topology already reaches tempfile/rustix from every guest; this is existing host build-tool reachability, not proof of a guest runtime dependency or new regression. Example paths:

- umpire-temporal-guest --normal--> umpire-runtime --build--> prost-build --normal--> tempfile
- umpire-temporal-guest --normal--> umpire-runtime --build--> prost-build --normal--> tempfile --normal--> rustix

Target/build context must be distinguished in later compiled-guest accounting. Do not demand that host build-tool transitive packages disappear, or call the metadata union a compiled/runtime closure. This baseline may be reused only while its recorded manifest hashes still match. It creates no additional full gate or old-byte comparison.
