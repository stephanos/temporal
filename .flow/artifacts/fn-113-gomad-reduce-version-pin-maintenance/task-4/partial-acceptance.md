- [x] README, SPEC, CLI.md, ARCHITECTURE, and the generated upgrade guide describe the bump procedure with the new commands
- [x] Manual steps per bump are reported before and after against the task 1 baseline
- [x] Roadmap COMPAT-8 and the milestones section reflect the delivered state
- [ ] `make -C tools/gomad3 validate` and `test`, compatibility-pack qualification, and the core set pass on darwin/arm64 and linux/amd64; a missing linux run leaves this task open
- [x] Links and command inventories checked

Local progress verified on 2026-10-03: R5 and the architecture correction received a SHIP verdict in `task-4/working-tree-review.json`. Darwin validation and the literal full test gate pass, all eight mapped pack requests qualify, and all seven core workloads qualify and replay exactly. Source and gate identities are retained in `task-4/source-binding.json`; actual and normalized step counts are distinguished in `task-4/measurement.json` and `walkthrough.json`. Native linux/amd64 execution evidence is missing, so R6 and this task remain incomplete. No changes were committed.
