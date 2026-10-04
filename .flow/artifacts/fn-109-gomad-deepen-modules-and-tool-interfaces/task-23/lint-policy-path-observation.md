# Inherited lint path-policy observation

Read-only scout `/root/lint_policy_path_scout` examined the actual first-freeze
Gomad log, unchanged config and pinned v2.13.0 implementation. It ran no lint,
tests, builds or generators and changed no source or lifecycle state. Requested
thinking-scout routing was `gpt-6.1-sol/high`; actual model is not exposed.
The pre-dispatch bounded judge returned `jev-unavailable(no_key)`.

Two inherited configuration mechanisms require a separate owner; task 23
preserves the existing rules and excludes configuration edits from its Touches.

1. No `run.relative-path-mode` is configured. Pinned v2.13.0 defaults to `cfg`,
   making the config directory `.github` the path base. Thus live source paths
   become `../tools/gomad3/...`, independent of the module invocation cwd.
   Exclusion matching and displayed filenames both use this relative path;
   comparison filtering separately uses the working-directory-relative path.
   Reporting flags `--path-prefix` and `--path-mode` do not repair matching.
2. The existing plain YAML tools/revive path scalar is `^tools\\/.+\\.go`.
   Its two literal backslashes at each escape survive YAML loading and Unix
   normalization. Direct Go regexp compilation therefore expects literal
   backslashes; restoring the root path base alone cannot make it match Unix
   tools paths. Other plain path patterns need the same explicit inventory.

Primary implementation references under the local pinned module
`/home/agent/go/pkg/mod/github.com/golangci/golangci-lint/v2@v2.13.0`:
`pkg/fsutils/basepath.go:29`, `pkg/config/base_loader.go:189`,
`pkg/config/loader.go:112`, `.golangci.reference.yml:4859`,
`pkg/result/processors/path_relativity.go:44`, `base_rule.go:35,58`,
`exclusion_paths.go:97`, `path_prettifier.go:41`, `diff.go:88`,
`pkg/fsutils/path_unix.go:6` and `exclusion_rules.go:111`.

Task-5's retained `lint-final-unfiltered` evidence and fn-113 task-2
`nested-lint.log` already show `../tools/gomad3/...` revive findings. This
establishes inherited mechanisms and some earlier findings, not the provenance
or validity of every current reported issue. Config SHA-256 remains
`86d71dda338f89c748a7ecae99e989d03b71b8693280adddae3970eba04c930a`;
the actual golangci binary remains
`acacd04faf1d17489a890a4589ae36a62484c1076cf5fbcb7a33274cbc6a6bdc`.

A prospective bounded R19 policy-repair owner should verify the intended
root-relative exclusions, set the consistent Git-root base and repair literal
escaping with explicit matching/nonmatching regressions. This is a diagnosis,
not authority to blanket-suppress findings or a green qualification result.
Restoring the Git-root base also activates the existing `^.git` exclusion for
`.github`; retain that reporting limitation explicitly. Current failed logs,
remaining source findings and original acceptance requirements stay intact.
