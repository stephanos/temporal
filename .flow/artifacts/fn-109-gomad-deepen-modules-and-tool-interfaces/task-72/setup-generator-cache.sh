#!/usr/bin/env bash
set -euo pipefail
cd /Users/stephan/Workspace/skunkworks/.gomad-scripted-and-spin-corrections.gFiXmTVr/seed-completion
export SANDBOX_START_DIR="$PWD"
unset BASH_ENV ENV MAKEFLAGS MFLAGS
packet=.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-72
target=/Users/stephan/Workspace/skunkworks/.gomad-scripted-and-spin-corrections.gFiXmTVr/retained-success/tools/gomad3/.toolchain/generator-cache
path=tools/gomad3/.toolchain/generator-cache
test ! -e "$packet/setup-generator-cache.log"
test ! -e "$packet/setup-generator-cache.json"
test ! -e "$path" && test ! -L "$path"
test -d "$target"
git check-ignore "$path" > "$packet/setup-generator-cache.log"
printf 'before=ABSENT path=%s target-existing=%s\n' "$path" "$target" >> "$packet/setup-generator-cache.log"
sha256sum "$packet/setup-generator-cache.sh" /usr/bin/bash /usr/bin/env /usr/bin/git /usr/bin/mkdir /usr/bin/ln /usr/bin/readlink /usr/bin/sha256sum /usr/bin/jq /usr/bin/date /usr/bin/cmp > "$packet/setup-tools-before.sha256"
started=$(date -u +%Y-%m-%dT%H:%M:%SZ)
start_seconds=$(date +%s)
mkdir -p tools/gomad3/.toolchain
ln -s "$target" "$path"
test "$(readlink "$path")" = "$target"
test "$(readlink -f "$path")" = "$target"
git check-ignore "$path" >> "$packet/setup-generator-cache.log"
printf 'after=symlink literal=%s resolved=%s\n' "$(readlink "$path")" "$(readlink -f "$path")" >> "$packet/setup-generator-cache.log"
ended=$(date -u +%Y-%m-%dT%H:%M:%SZ)
elapsed=$(($(date +%s)-start_seconds))
sha256sum "$packet/setup-generator-cache.sh" /usr/bin/bash /usr/bin/env /usr/bin/git /usr/bin/mkdir /usr/bin/ln /usr/bin/readlink /usr/bin/sha256sum /usr/bin/jq /usr/bin/date /usr/bin/cmp > "$packet/setup-tools-after.sha256"
cmp "$packet/setup-tools-before.sha256" "$packet/setup-tools-after.sha256"
jq -n --arg started "$started" --arg ended "$ended" --argjson elapsed "$elapsed" --arg target "$target" --arg path "$path" --arg raw "$(sha256sum "$packet/setup-generator-cache.log")" --arg before "$(sha256sum "$packet/setup-tools-before.sha256")" --arg after "$(sha256sum "$packet/setup-tools-after.sha256")" '{exit:0,started:$started,ended:$ended,elapsed_seconds:$elapsed,cwd:"/Users/stephan/Workspace/skunkworks/.gomad-scripted-and-spin-corrections.gFiXmTVr/seed-completion",before:"ABSENT",after:"ignored symlink",target:$target,path:$path,tools_before:$before,tools_after:$after,raw_log:$raw,operations:[["mkdir","-p","tools/gomad3/.toolchain"],["ln","-s",$target,$path]],cache_contents_qualified:false}' > "$packet/setup-generator-cache.json"
