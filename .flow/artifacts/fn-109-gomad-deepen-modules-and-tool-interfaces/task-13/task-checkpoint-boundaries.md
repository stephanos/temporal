# Task 13–18 source checkpoint boundaries

This inventory supports six separate progress checkpoints. Task 13 is the first boundary: 20 of its 22 retained source hashes match the working tree; the conductor recovered its host time source and descriptor in an isolated scratch tree, and this scout independently verified all 22 identities. Four shared identity mirrors match task 13, task 14, task 17, and the working tree exactly. They belong in task 13 and introduce no further delta in task 14 or task 17.

Observed HEAD: `4a74aaca83091d568ad08863f7ccdffd4dd7b185`. SHA comparisons were captured at `2026-10-04T05:59:04.452Z`; the repository environment date and actual command clock differ. The index was empty when inspected. Task 19 remains the production writer; source and index state must be checked again before conductor staging.

This scout read source, evidence, handovers, manifests, existing snapshots and Git diffs. It wrote only this report, with no staging, Git objects, worktrees, source changes, Go commands, generators, tests, or Flow lifecycle mutations. This is neither SHIP nor task completion nor supported-platform qualification. Codex thinking-scout requested pin: gpt-6.1-sol/high; Tier: session (jev-unavailable(no_key)); actual execution-model metadata is not established.

## Boundary summary

| Task | Retained entries | Current bytes matching | Historical bytes needed | Exact retained preimages found | Remaining complete-preimage gaps |
| --- | ---: | ---: | ---: | ---: | --- |
| 13 | 22 source | 20 | 2 | 0 old preimages; 2 conductor recoveries verified | None after verified recovery |
| 14 | 17 source | 11 | 6 | 6 | None |
| 15 | 2 tests + 1 design | 1 design | 2 tests | 1 | simulation_progress_fixture_test.go |
| 16 | 10 source | 10 | 0 | Not needed | None; simulation_model.go is unchanged |
| 17 | 17 source, using corrected manifest | 13 | 4 | 3 | simulation_root_integration_test.go |
| 18 | 14 source | 14 | 0 | Not needed | None |

“Matching” compares full SHA-256 of current bytes to the retained final identity, not only diff hunks. “Exclusive” means absent from the other task-13–18 final manifests; it is not a claim about future edits. The table below includes every retained final source entry, plus the task-15 design. Locations marked `WT` are the repository path shown in that row.

## Exact inventory

Path prefixes used below:

- `G/` = `tools/gomad3/`.
- `E/` = `tools/gomad3/runner/internal/execution/`.
- `O/` = `tools/gomad3/toolchain/runtime/overlay/`.
- `S/` = `tools/gomad3sim/`.
- `A/` = `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/`.
- `P17/` = `/tmp/gomad-task17.WFwzYI/`.
- `P18/` = `/tmp/gomad-task18.U1uS6Z/`.
- `C13/` = `/tmp/fn109-task13-checkpoint.FAl6gGl0/tools/gomad3/` (conductor recovery, independently verified).

### Task 13

Identity source: `A/task-13/evidence.json`.

| Repository path | Retained final SHA-256 | Current relation / exact byte location |
| --- | --- | --- |
| `G/choice/internal/wire/wire_generated.go` | `d4e6f2bd689bda6730106968635b40d26edb4e8d460429c8336614d933369fbc` | MATCH; shared same bytes with 14, 17; `WT` |
| `G/internal/gomadtool/conformance/runtime_campaign.go` | `ce8b0bb5e310abdb192aab317c5534a9966985a742822c32a2d63ce1f8ef78fa` | MATCH; exclusive; `WT` |
| `G/internal/gomadtool/conformance/runtime_test.go` | `fe2b99204a9439069360c41726f20f80ac6c1558e172d10b7995da5843998592` | MATCH; exclusive; `WT` |
| `G/internal/gomadtool/generation/protocol/protocol.go` | `3d179a9736d39b929bcee66321aacb3e5124c836a8f8612f356fa5a1aa7f021b` | MATCH; exclusive; `WT` |
| `G/internal/gomadtool/generation/protocol/protocol_test.go` | `526369d96209fa3c40a5808fa4ace08a3040ff312fcbfff00258fa84ebd21cf9` | MATCH; exclusive; `WT` |
| `E/simulation_time.go` | `496b81c6cc33e7d63ae19c3c812de9f05028cb89765b38a911bb0956eb0cf622` | LATER DELTA; current matches later task 16; verified conductor recovery `C13/runner/internal/execution/simulation_time.go` |
| `E/simulation_time_wire_generated.go` | `b38db5352526b162bef952dd8e31bbe6a60cfe2bed4f4dd67da0209d283cc519` | MATCH; exclusive; `WT` |
| `E/simulation_time_wire_generated_test.go` | `8eea52828f19cc7de96a4d51e57b794a45444e968f08f0ddaa88894360bb6bf9` | MATCH; exclusive; `WT` |
| `G/simulation/schema/timewire.json` | `b79745bde036ab57e908c42f8acb4548200a3b2d00a198a502dcba8c9d99fde8` | MATCH; exclusive; `WT` |
| `G/simulation/schema/timewire_host.go.tmpl` | `42c304bd271bf1bcf490e81f6e80848b12a39d4fc3dcbd31a6d4ac2070aee59f` | MATCH; exclusive; `WT` |
| `G/simulation/schema/timewire_host_test.go.tmpl` | `e0ae4c2008f6324eb8c70df0fd3a30d16933d6e56a4311c8d477aa4676b1e987` | MATCH; exclusive; `WT` |
| `G/simulation/schema/timewire_runtime.go.tmpl` | `d41cd9eb27af9f5b3a4ebef20aecdb120ae1c3fdd5ef03d39654274ea367227e` | MATCH; exclusive; `WT` |
| `G/simulation/schema/timewire_runtime_export_test.go.tmpl` | `49b5075dd73ea527175e99ffb5ecb3c3cd2ffa7a41b1627e2651b1f2fd7dbc93` | MATCH; exclusive; `WT` |
| `G/simulation/schema/timewire_runtime_test.go.tmpl` | `faaf0924b3684b05728fd2c31981e9ccc35fa457aee5c34fdc4dbaab8496ec97` | MATCH; exclusive; `WT` |
| `G/target/internal/livecap/protocol_generated.go` | `6dcb24fab85bbf42d195f6db023ac2cbd280ba85712b9f764903c96bcf2140af` | MATCH; shared same bytes with 14, 17; `WT` |
| `O/src/cmd/internal/gomadcap/protocol_generated.go` | `204e501a19e6229aa285cbf5b12fc478b270fc7a8d0f401a655e311622cdbddd` | MATCH; shared same bytes with 14, 17; `WT` |
| `O/src/internal/gomadchoicewire/wire_generated.go` | `a4ea7fd30686e2369f3adf48366a0c704a9920ad88e1ffa2b18247e8e2b4d56a` | MATCH; shared same bytes with 14, 17; `WT` |
| `O/src/runtime/gomad.go` | `9055744a08fa24cbc376edd6ca261b3a6fa401bdc407725809066d64962126e9` | MATCH; exclusive; `WT` |
| `O/src/runtime/gomad_timewire_export_generated_test.go` | `49b5075dd73ea527175e99ffb5ecb3c3cd2ffa7a41b1627e2651b1f2fd7dbc93` | MATCH; exclusive; `WT` |
| `O/src/runtime/gomad_timewire_generated.go` | `425d75ef372ba9eba431e0384b51bf21a0ebedf5d91a315d7894363f846efe87` | MATCH; exclusive; `WT` |
| `O/src/runtime/gomad_timewire_generated_test.go` | `d8dcca09321bd4ec618e8936d64f8109885bae01ccb6003678264bf828a2789b` | MATCH; exclusive; `WT` |
| `G/toolchain/version/version.json` | `8059b691e000d2d5c9c906b9b4c95323511e1586aa2e2d3466f65cdf915b5518` | LATER DELTA; current matches later task 18; verified conductor recovery `C13/toolchain/version/version.json` |

### Task 14

Identity source: `A/task-14/final-source.sha256`.

| Repository path | Retained final SHA-256 | Current relation / exact byte location |
| --- | --- | --- |
| `G/process_commands_ownership_test.go` | `0f42a38793408a38f765333a10dabb0fe7bdcc7b8374a9fc95c98afe15d2fffc` | MATCH; exclusive; `WT` |
| `O/src/internal/gomadio/process_commands.go` | `025373f3e03b240e032a2785658ce7a10dc9cf242aa722e4a681dd483979c92b` | MATCH; exclusive; `WT` |
| `O/src/internal/gomadio/process_commands_export_test.go` | `61240516651219c7e9ce812df63a538333ab7cc5a0007ea504875772a139a98d` | LATER DELTA; current matches later task 17; `P17/baseline-overlay/src/internal/gomadio/process_commands_export_test.go` |
| `O/src/internal/gomadio/process_commands_test.go` | `0752f0fa45c71bf4b1f127deb3a2330a4c86fa9d0dbddd95c63feb64f7057906` | MATCH; exclusive; `WT` |
| `O/src/internal/gomadio/process_network.go` | `a33d661688a96098702b7455a63d7776c59bd5cac168d661563bd0d10a778e55` | LATER DELTA; current matches later task 17; `P17/baseline-overlay/src/internal/gomadio/process_network.go` |
| `O/src/internal/gomadio/network.go` | `8123362e6cf1dc4f69c2cec12c9bbb35d7c28d182896a88043a3cca408a1abb5` | LATER DELTA; current matches later task 17; `P17/baseline-overlay/src/internal/gomadio/network.go` |
| `O/src/internal/gomadfs/process_commands.go` | `566d197b342e62f276d6a327bfc7cf482134ad8eb0f7274908454a261944b89b` | MATCH; exclusive; `WT` |
| `O/src/internal/gomadfs/process_commands_export_test.go` | `48d171cba430f8ff1c6c44acaf8ed37d9503d3844bfcfe5c49ccf59a78a1171c` | MATCH; exclusive; `WT` |
| `O/src/internal/gomadfs/process_commands_test.go` | `478a71de2035b5cd7a319c1a0f03f8cce899d845fad46a1c93956260d6901985` | MATCH; exclusive; `WT` |
| `O/src/internal/gomadfs/process_volume.go` | `e850022f49baf3e539efea0eda7680d9d3f8eb5aeab8b5d005b55ceaeec79449` | LATER DELTA; current matches later task 18; `P17/baseline-overlay/src/internal/gomadfs/process_volume.go` |
| `O/src/internal/gomadfs/process_volume_host.go` | `ab9a74b692fcdcdfb033fe94c0fd49f68ec885f9f0a39261467f34b93b3d2493` | MATCH; exclusive; `WT` |
| `O/src/internal/gomadfs/fs.go` | `87b4413f203c404935a789b16774408978c847a84d79358c4d79cc3306e583e1` | LATER DELTA; current matches later task 18; `P17/baseline-overlay/src/internal/gomadfs/fs.go` |
| `G/toolchain/version/version.json` | `b6190507eb2a4f23c810fd11b1ec58240d3e822ceee6c9c31a3fda6e85bad978` | LATER DELTA; current matches later task 18; `P17/baseline-version/version.json` |
| `G/choice/internal/wire/wire_generated.go` | `d4e6f2bd689bda6730106968635b40d26edb4e8d460429c8336614d933369fbc` | MATCH; shared same bytes with 13, 17; `WT` |
| `G/target/internal/livecap/protocol_generated.go` | `6dcb24fab85bbf42d195f6db023ac2cbd280ba85712b9f764903c96bcf2140af` | MATCH; shared same bytes with 13, 17; `WT` |
| `O/src/cmd/internal/gomadcap/protocol_generated.go` | `204e501a19e6229aa285cbf5b12fc478b270fc7a8d0f401a655e311622cdbddd` | MATCH; shared same bytes with 13, 17; `WT` |
| `O/src/internal/gomadchoicewire/wire_generated.go` | `a4ea7fd30686e2369f3adf48366a0c704a9920ad88e1ffa2b18247e8e2b4d56a` | MATCH; shared same bytes with 13, 17; `WT` |

### Task 15

Identity source: `A/task-15/evidence.json`.

| Repository path | Retained final SHA-256 | Current relation / exact byte location |
| --- | --- | --- |
| `E/simulation_progress_test.go` | `4d91f01eb5e378e5aa4824c2af655862d9fe0fe57772bd74f2dfa648418c0880` | LATER DELTA; current matches later task 16; `A/task-16/simulation_progress_test.before.txt` |
| `E/simulation_progress_fixture_test.go` | `35ed448549b3aa5d6ce959d86a631b37979056642144e31274e18bcbfb0e8e5c` | LATER DELTA; current matches later task 16; `GAP: no complete exact preimage located` |
| `A/simulation-progress-design.md` | `1f2fc94d417ad3ffe2d64a2b255787d3ad74e13701bc85a7294522a0629a60c9` | MATCH; exclusive; `WT` |

### Task 16

Identity source: `A/task-16/evidence.json`.

| Repository path | Retained final SHA-256 | Current relation / exact byte location |
| --- | --- | --- |
| `E/simulation_progress.go` | `a5446daf575f02fb909955eb6ae8c6d126e3a94048a53fb1337bc444e934dcd9` | MATCH; exclusive; `WT` |
| `E/simulation_time.go` | `5ba93587f60bad5cbb7ce8547257da8869ec822485f23856bf45867732313697` | MATCH; shared different versions in 13; `WT` |
| `E/simulation_unix.go` | `95b63f504cf575574129616bec3f008de6560c29be3019537f5e8a256b150881` | MATCH; exclusive; `WT` |
| `E/simulation_model.go` | `2a88cbdde3fc952428cb2890efd1194423c52c104fc940368e752f7ee388b668` | MATCH; unchanged predecessor and HEAD; omit from task-16 staging; `WT` |
| `E/process_unix.go` | `c9285406a0602cea4e301a80d88810a08c77f888659df158d4dd7e3d7a74f6a5` | MATCH; exclusive; `WT` |
| `E/simulation_progress_test.go` | `7b1051a4332f73ef1263b1054002ddff061330b1ff39d476e4d076f5b1febce3` | MATCH; shared different versions in 15; `WT` |
| `E/simulation_progress_fixture_test.go` | `12cbd8911c03b5ed4e0717397fec2742508daab3e28b8e91a6ca376671db8add` | MATCH; shared different versions in 15; `WT` |
| `E/simulation_progress_lifecycle_test.go` | `27430332b60b07ae2aebabd7f2cd8c251fae092f74bc397157699c661bcf8312` | MATCH; exclusive; `WT` |
| `E/simulation_time_test.go` | `3f83c32370640b54e0518f69739fd537369888f24837e1620cde23a65f8d6e90` | MATCH; exclusive; `WT` |
| `E/simulation_unix_test.go` | `0025f6826a715b3cb366fddc15752a91b102311903d6dbc817eecfb041e1a987` | MATCH; exclusive; `WT` |

### Task 17

Identity source: `A/task-17/final-source-post-correction.sha256` (17 entries; use this rather than the original 15-entry pre-correction manifest).

| Repository path | Retained final SHA-256 | Current relation / exact byte location |
| --- | --- | --- |
| `G/network_handles_ownership_test.go` | `819c8ee11a5425f5cf483f3a3ca0b3230ff60ba52aae9db929edc3ec75309b35` | MATCH; exclusive; `WT` |
| `E/simulation_root_integration_test.go` | `bc4c86b6210232026f0eb94edbee209fa9817a62b66d58a2da7b627e9bbbf833` | LATER DELTA; current matches later task 18; `GAP: no complete exact preimage located` |
| `O/src/internal/gomadio/network.go` | `57b98bb88190fac515b758d07be31e8c9d428a16159a07d4f754ef2dd39b9356` | MATCH; shared different versions in 14; `WT` |
| `O/src/internal/gomadio/network_handles_test.go` | `cfead548fb2ecf987d4314d34fa2be1161bb1941786e2ba08dc7bfcdb0b59fba` | MATCH; exclusive; `WT` |
| `O/src/internal/gomadio/process_network.go` | `584ac64bf45d35e0f89453f732acc22448d244be2984e284743b23c6c142acf1` | MATCH; shared different versions in 14; `WT` |
| `O/src/internal/gomadio/process_commands_export_test.go` | `37a1aa079922906b97f94e178fe65b82da6225eb7cbd4f172bf89ebe03eeec97` | MATCH; shared different versions in 14; `WT` |
| `O/src/internal/gomadio/simulation_network.go` | `a8dd8cf68d3d50679e2a0ca839e602bb1bf7f20a9e8eabc8e9427bd35c92624d` | MATCH; exclusive; `WT` |
| `O/src/internal/gomadio/simulation_handles.go` | `b38594b3713a73f7c0bc1e2b06890ab4c27f9ee60efd6bf7ae23504e88c3a1e7` | MATCH; exclusive; `WT` |
| `G/toolchain/version/version.json` | `75d2429a49e42df374e2fc2b77710d1694322c5198a4b54f6c9bfb95ab88c8de` | LATER DELTA; current matches later task 18; `P18/baseline-version/version.json` |
| `G/choice/internal/wire/wire_generated.go` | `d4e6f2bd689bda6730106968635b40d26edb4e8d460429c8336614d933369fbc` | MATCH; shared same bytes with 13, 14; `WT` |
| `G/target/internal/livecap/protocol_generated.go` | `6dcb24fab85bbf42d195f6db023ac2cbd280ba85712b9f764903c96bcf2140af` | MATCH; shared same bytes with 13, 14; `WT` |
| `O/src/cmd/internal/gomadcap/protocol_generated.go` | `204e501a19e6229aa285cbf5b12fc478b270fc7a8d0f401a655e311622cdbddd` | MATCH; shared same bytes with 13, 14; `WT` |
| `O/src/internal/gomadchoicewire/wire_generated.go` | `a4ea7fd30686e2369f3adf48366a0c704a9920ad88e1ffa2b18247e8e2b4d56a` | MATCH; shared same bytes with 13, 14; `WT` |
| `S/network_handles_toolchain_test.go` | `dce395ca9c04360dbea1d0963697b9a2be5b69edeefe2d2d90d67b57106f096d` | MATCH; exclusive; `WT` |
| `S/network_process_handles_toolchain_test.go` | `61908294fa667549a3c3fd310f5358ce4d50c3ed552ce7860bb117cd014200de` | MATCH; exclusive; `WT` |
| `G/Makefile` | `28318e32d3360cfcd644298944e864e704670ac80f55c550d32461a55afc8497` | LATER DELTA; current matches later task 18; `P18/Makefile` |
| `G/simulation_gate_selection_test.go` | `57febc14101facc1b4db77ba3f8dba55669e51036e23790eb1372c27dec38bbb` | LATER DELTA; current matches later task 18; `P18/simulation_gate_selection_test.go` |

### Task 18

Identity source: `A/task-18/final-source.sha256`.

| Repository path | Retained final SHA-256 | Current relation / exact byte location |
| --- | --- | --- |
| `G/filesystem_handles_ownership_test.go` | `67e23f90d84ee19c00884d4277b80641079d96580c80e97950de572ac6f87257` | MATCH; exclusive; `WT` |
| `G/Makefile` | `a25443d9c1adaefe838a39809d8deeffd6323a0916f4ac3a52a1083da598a200` | MATCH; shared different versions in 17; `WT` |
| `G/simulation_gate_selection_test.go` | `2b830d04d7ea3c3ad70cc33b6f9a11e96cdde4dd0ad5995f644e04429208f979` | MATCH; shared different versions in 17; `WT` |
| `E/simulation_root_integration_test.go` | `02544185057eff83b65b288b6062c374376b3a10137dc526405c222e052ec682` | MATCH; shared different versions in 17; `WT` |
| `G/toolchain/version/version.json` | `94358dc0d221c0c0ba0e4f487303b0e9acd46afa212d24091ab44b124f276779` | MATCH; shared different versions in 13, 14, 17; `WT` |
| `O/src/internal/gomadfs/fs.go` | `17a6163daca49c776abee8eac53aca8f68e47f2ef09d51a2364f83bf52d13453` | MATCH; shared different versions in 14; `WT` |
| `O/src/internal/gomadfs/handles.go` | `5b95fb83701ca10dbb8af5c53af8e9c7997cdddf8b78384a222bb585bf9de9d0` | MATCH; exclusive; `WT` |
| `O/src/internal/gomadfs/local_handles.go` | `9e74dcd7201572f4618e2933b1f272439c24c478ce5e92483ca0de00e742eb98` | MATCH; exclusive; `WT` |
| `O/src/internal/gomadfs/process_volume.go` | `983194fedffe08a092ee14e113149472a91e65c2576b1806934e22b02d32e4c2` | MATCH; shared different versions in 14; `WT` |
| `O/src/internal/gomadfs/volume.go` | `efd241e55910a4006df26db847a03d90f7b44b35e5d4df357367b4b60902afa1` | MATCH; exclusive; `WT` |
| `O/src/internal/gomadfs/export_test.go` | `8f04c57217c92ed2e09e3fff4268a2c369267a55ffc9d31c48f18fca77ce3220` | MATCH; exclusive; `WT` |
| `O/src/internal/gomadfs/handles_test.go` | `7b99ae8033ee7a94a63d7227e20e612f6f94b175b87978a1b1e87ae3716a268b` | MATCH; exclusive; `WT` |
| `O/src/internal/gomadfs/process_handles_test.go` | `801685588aa34bc16ff361936149726492470a167a029cb2dc88dd5c22db1ff5` | MATCH; exclusive; `WT` |
| `S/filesystem_handles_toolchain_test.go` | `408bab9e04f15ccb6ded28380e5bad236f901b13a4071dd0e9d4d2e237d5e70b` | MATCH; exclusive; `WT` |

## Current identities for changed earlier entries

These are the full current SHA-256 values behind every “LATER DELTA” row. They already appear as the retained later-task identities above; they must not be staged wholesale at the earlier boundary.

| Earlier task | Path | Current SHA-256 | Later owner |
| --- | --- | --- | --- |
| 13 | `E/simulation_time.go` | `5ba93587f60bad5cbb7ce8547257da8869ec822485f23856bf45867732313697` | 16 |
| 13 | `G/toolchain/version/version.json` | `94358dc0d221c0c0ba0e4f487303b0e9acd46afa212d24091ab44b124f276779` | 18 |
| 14 | `O/src/internal/gomadio/process_commands_export_test.go` | `37a1aa079922906b97f94e178fe65b82da6225eb7cbd4f172bf89ebe03eeec97` | 17 |
| 14 | `O/src/internal/gomadio/process_network.go` | `584ac64bf45d35e0f89453f732acc22448d244be2984e284743b23c6c142acf1` | 17 |
| 14 | `O/src/internal/gomadio/network.go` | `57b98bb88190fac515b758d07be31e8c9d428a16159a07d4f754ef2dd39b9356` | 17 |
| 14 | `O/src/internal/gomadfs/process_volume.go` | `983194fedffe08a092ee14e113149472a91e65c2576b1806934e22b02d32e4c2` | 18 |
| 14 | `O/src/internal/gomadfs/fs.go` | `17a6163daca49c776abee8eac53aca8f68e47f2ef09d51a2364f83bf52d13453` | 18 |
| 14 | `G/toolchain/version/version.json` | `94358dc0d221c0c0ba0e4f487303b0e9acd46afa212d24091ab44b124f276779` | 18 |
| 15 | `E/simulation_progress_test.go` | `7b1051a4332f73ef1263b1054002ddff061330b1ff39d476e4d076f5b1febce3` | 16 |
| 15 | `E/simulation_progress_fixture_test.go` | `12cbd8911c03b5ed4e0717397fec2742508daab3e28b8e91a6ca376671db8add` | 16 |
| 17 | `E/simulation_root_integration_test.go` | `02544185057eff83b65b288b6062c374376b3a10137dc526405c222e052ec682` | 18 |
| 17 | `G/toolchain/version/version.json` | `94358dc0d221c0c0ba0e4f487303b0e9acd46afa212d24091ab44b124f276779` | 18 |
| 17 | `G/Makefile` | `a25443d9c1adaefe838a39809d8deeffd6323a0916f4ac3a52a1083da598a200` | 18 |
| 17 | `G/simulation_gate_selection_test.go` | `2b830d04d7ea3c3ad70cc33b6f9a11e96cdde4dd0ad5995f644e04429208f979` | 18 |

## Unchanged production recorded by task 15

Task 15 changes only characterization tests and the design. Its four production “final” hashes equal its baseline hashes; they are preservation evidence, not task-15 implementation files.

| Path | Task-15 baseline/final SHA-256 | Relation to current HEAD and working tree |
| --- | --- | --- |
| `E/simulation_time.go` | `496b81c6cc33e7d63ae19c3c812de9f05028cb89765b38a911bb0956eb0cf622` | task-13 version; WT contains task-16 delta; HEAD is pre-task-13 |
| `E/simulation_unix.go` | `64cced8a2d448316965f38cfdfe44a700dc005bdc71e0d50743e68483b8a7e3d` | matches HEAD; WT contains task-16 delta |
| `E/simulation_model.go` | `2a88cbdde3fc952428cb2890efd1194423c52c104fc940368e752f7ee388b668` | matches HEAD and WT; no task-15 or task-16 delta |
| `E/process_unix.go` | `edd93e2d813c79831c93a0e3de23a73b9e7a5644afd0474b78041ca56e0d546a` | matches HEAD; WT contains task-16 delta |

## Existing historical bytes and unresolved gaps

All preimage locations in the main inventory were independently SHA-256 checked against the retained final identities. Task 14's six shared-file preimages are in the task-17 baseline overlay/version directories. Its unchanged domain codec files also exist there with exact task-14 hashes. Task 18's baseline overlay provides a second exact copy of task 14's `gomadfs/fs.go` and `gomadfs/process_volume.go`. These copies preserve task 14 independently of current task-18 source.

Task 13's runtime source and three generated runtime outputs are additionally retained at `/tmp/gomad-task14.UFfeZq/baseline-overlay/src/runtime/`, with exact task-13 hashes. That overlay does not retain the descriptor or host time file. The conductor recovery below fills those two historical-byte gaps without treating this overlay as a missing preimage.

Task 15's complete `simulation_progress_test.go` preimage at `A/task-16/simulation_progress_test.before.txt` exactly matches task-15 final SHA `4d91f01eb5e378e5aa4824c2af655862d9fe0fe57772bd74f2dfa648418c0880`. The task-16 evidence records the old fixture hash, but its retained `preimage` field points only to the characterization test, not to a complete fixture copy. The valid-test-body comparison is narrower evidence than an entire fixture preimage.

Task 17's descriptor, corrected Makefile, and gate-selection test are retained as complete exact files in `P18/`. The corrected Makefile hash is `28318e32d3360cfcd644298944e864e704670ac80f55c550d32461a55afc8497`; `A/task-17/pre-correction-Makefile` has `e5802cb89258e7b28abf4aa368af5bb3e8eb2c1508741bc4e78a752ff32b9fff` and is not an acceptable substitute. Task 18's baseline manifest records task 17's Runner selector hash but no complete retained Runner selector copy was located.

Read-only searches covered task-13–18 artifacts, fn-109 preimage/snapshot paths, known task-14/17/18 scratch roots, task-13 runtime scratch, fn-109 baseline-measurement scratch trees, and narrowed matching filenames throughout `/tmp` and the workspace. Existing integration/final-merge checkouts and fn-109 baseline reconstruction trees contain older, nonmatching host time/descriptor/Runner-selector bytes. “Not located” is bounded to these searches; it does not assert that no copy exists elsewhere.

An existing tracked-source patch, `P17/baseline.diff`, has SHA-256 `cff064cdefea694335315a0065811b3928cdbfc4de590395b2d0f18423886f68`. Its `simulation_time.go` section starts at line 713; `version.json` starts at line 3012. It includes task-13/14/16 tracked changes, not a task-13-only full-file snapshot, and omits untracked test files. It is a useful provenance pointer for conductor reconstruction, but its presence alone does not establish any missing full-file SHA. No missing bytes were synthesized by this scout.

Two complete-preimage gaps remain after the independently verified conductor task-13 recovery:

| Boundary | Missing exact file | Required SHA-256 |
| --- | --- | --- |
| Task 15 | `E/simulation_progress_fixture_test.go` | `35ed448549b3aa5d6ce959d86a631b37979056642144e31274e18bcbfb0e8e5c` |
| Task 17 | `E/simulation_root_integration_test.go` | `bc4c86b6210232026f0eb94edbee209fa9817a62b66d58a2da7b627e9bbbf833` |

## Conductor task-13 recovery audit

The conductor supplied `A/task-13/reconstruct-checkpoint.py` and `A/task-13/checkpoint-reconstruction.json` after the preimage search. This scout inspected both and the pinned `A/task-21/baseline-reconstruction/reconstruct.py` helper read-only, without running a reconstruction. The script SHA-256 was `3cb643563b8af980aea1490a65d7b5f383ef2bf7ecbc62014c20ca6753d19bc3`; its reconstruction report SHA-256 was `7a936b5d6806af1a1f54a783a4fc9fce133bcf787a21f1465430aedc4d9e7115`. The helper's actual SHA-256 matched its pinned `f19edebc9708e3d6716682e58d91b09aa9cc30dcd9d7d6d666f7cfbba6e2796c`, and the historical patch matched the SHA above.

The script archives only `tools/gomad3` from base `0dd05b313acd0986312da7fd3159520e6a21f1bf`, restricts historical patch parsing to exactly the host time source and descriptor, and replaces only the 22 task-13 source-map entries. The exact-hunk helper checks source/context bytes, hunk counts and output positions. Time recovery selects only the three codec-removal hunks at old lines 1, 12 and 40, stopping before old line 196; lifecycle hunks are excluded. Descriptor recovery selects only the time-wire allowlist hunk at old line 174, changing its new-position header from 180 to 174 to exclude the earlier task-14 domain insertions. Each recovered whole file must match its full task-13 hash.

Independent byte checks support this separation: all 22 retained final hashes match the actual scratch files, all 867 rows in the reconstruction manifest match actual scratch bytes, and comparing every manifest-listed source against base found no changed or new path outside the task-13 map. The scratch architecture file remains base bytes; active task-19 architecture source was not imported. Four generated identity mirrors are the exact task-13 identities even though task-14/17 manifests repeat them. Current task-16 lifecycle source, task-14/17/18 domain allowlist additions, task-17/18 gate changes, and other task-19 files are excluded.

The conductor's recovered copies at `C13/runner/internal/execution/simulation_time.go` and `C13/toolchain/version/version.json` are now eligible byte sources for a task-13 checkpoint. They are newly reconstructed, full-hash-verified bytes, not previously retained complete preimages. This audit supports source ownership/provenance only; the conductor's scratch gates, review/evidence checks and any index operations remain its responsibility.

## Safe task-order staging recommendations

These are recommendations to the conductor, the sole committer. Preserve current source and the task-19 writer. Historical bytes can be staged through the index without copying them over working-tree files. Before each checkpoint, verify staged path lists and staged file content hashes against its exact boundary; a working-tree test over later source does not by itself verify an earlier staged tree.

1. **Task 13:** stage its 16 exclusive matching files plus the four matching shared mirrors from WT. Supply the two independently verified `C13/` recovered files at their index paths. The two current files carry later deltas and must not be included wholesale. All 22 files can alternatively come from the verified scratch tree, after conductor gates and final identity rechecks. The source ownership separation is supported by the recovery audit above.
2. **Task 14:** after task 13 is committed, stage the seven exclusive matching files from WT and the six exact historical shared-file preimages identified above. The four mirrors already belong to task 13 and provide no new task-14 delta. Keep task-17 network changes and task-18 filesystem changes out of this boundary.
3. **Task 15:** stage the exact retained characterization preimage and matching design, plus a complete fixture matching its retained task-15 hash. Current fixture/test bytes are task-16 versions. The missing fixture blocks a faithful complete task-15 source checkpoint until located or independently reconstructed and full-hash verified. Stage no production file for task 15.
4. **Task 16:** after task 15, stage all nine changed source/test entries directly from WT. Omit unchanged `simulation_model.go`. Keep the task-15 characterization preimage as evidence and record that only the two historical negative tests change their bodies.
5. **Task 17:** stage its nine changed files that already match WT, plus the historical descriptor, corrected Makefile, gate-selection test and exact task-17 Runner selector. The last remains a gap. The four mirrors already committed with task 13 provide no further delta. Do not stage task-18 Runner/gate/filesystem selection changes at this boundary.
6. **Task 18:** after task 17, all 14 retained source files can be staged directly from WT, provided their full hashes still match. This advances the shared descriptor, filesystem files and gate/Runner selectors to their current final versions.

For each task, include only its reviewed task-owned tests, documentation and Flow records whose identities and ownership the conductor checks. The current task records can contain successor admission or reconciliation updates, so directory membership alone is not sufficient. Do not bulk-stage `.flow`, `tools/gomad3`, or the repository.

`architecture_test.go` is absent from the task-13 final source hash map and is currently owned by active task 19; this report does not establish a task-13 architecture delta. Current `ARCHITECTURE.md`, `README.md`, target/record/World/compatibility changes, architecture fixtures and other task-19 work are outside these source manifests. Preserve those changes and unrelated `.turbo` state.

MILESTONES requires separate task commits after source checks and review, permits verified progress commits with unavailable native gates recorded, and keeps acceptance open until all required native gates pass. The retained handovers' older “user owns commits” wording is superseded by that instruction and the user's explicit checkpoint authorization. Source hash identity alone does not upgrade developmental tests to darwin/arm64 or linux/amd64 qualification. The conductor must check each required gate and review/evidence scope before committing; this scout supplies byte boundaries only.
