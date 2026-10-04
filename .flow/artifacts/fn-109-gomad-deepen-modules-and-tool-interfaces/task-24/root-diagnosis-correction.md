# Corrected lint-policy diagnosis, 2026-10-04

The config-relative path defect is confirmed; the earlier doubled-literal
escaping claim is false for the unchanged config. Preserve working regexes.

Task 24's pre-edit [behavioral RED](policy-red.log) passes the independent
parsed-YAML/Go-regexp path-expression cases but fails actual pinned golangci
fixtures with `../` issue paths. Its first fixture also contains unrelated
revive/gci setup findings, so the worker must retain a clean meaningful RED
after correcting only fixture setup.

Root independently inspected the config's bytes with
`sed -n '232,243p' .github/.golangci.yml | od -An -t x1c` at 12:13:59 UTC.
The tools expression contains `5c 2f` and `5c 2e`: one backslash before slash
and dot, not two. Config SHA-256 remains
`86d71dda338f89c748a7ecae99e989d03b71b8693280adddae3970eba04c930a`.
Serialized tool/log display escaping is not raw YAML content.

The task-23 observation and review remain immutable historical evidence. This
new correction supersedes their regex-defect claim, not their passing source
checks or red qualification. Root updated only task 24's new, derived cause and
first acceptance clause through Flow API: keep all working path expressions
unchanged, require the same independent match/nonmatch inventory and real-tool
positive/negative controls, and repair the actual path base. No original fn-109
criterion, baseline, rule, pin or native gate changed. This correction narrows
an unsupported fix hypothesis; it does not turn failing product lint green.
