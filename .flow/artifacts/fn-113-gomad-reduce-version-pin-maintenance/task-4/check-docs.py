#!/usr/bin/env python3
"""fn-113 task 4 documentation checks (run from the repository root).

1. Flags: every `--flag` after `gomadtool <command> [<subcommand>]` in a shell
   fence or inline code span of the checked documents must be a flag the
   built gomadtool accepts for that command (from `-h`).
2. Inventory: every gomadtool command in the usage line has a CLI.md command
   index row, and every row names a real command.
3. Links: every relative Markdown link and #fragment in the checked documents
   resolves (GitHub-style heading slugs).
"""
import os, re, subprocess, sys
GT = sys.argv[1]
DOCS = ["tools/gomad3/README.md", "tools/gomad3/CLI.md", "tools/gomad3/SPEC.md",
        "tools/gomad3/ARCHITECTURE.md", "tools/gomad3/deterministicio/boundary/upgrade-go1.27.1.md",
        ".plans/GOMAD_NEXT.md", "MILESTONES.md"]
SUBS = {"compatibility-pack": ["discover", "review", "generate", "qualify", "check", "refresh"]}
errors = []
def helpflags(words):
    out = subprocess.run([GT, *words, "-h"], capture_output=True, text=True)
    return set(re.findall(r"^  -([a-z][a-z0-9-]*)", out.stdout + out.stderr, re.M))
usage = subprocess.run([GT], capture_output=True, text=True)
commands = re.search(r"usage: gomadtool (\S+)", usage.stdout + usage.stderr).group(1).split("|")
cache = {}
checked = 0
for doc in DOCS:
    text = open(doc).read()
    # join backslash continuations so a multi-line command is one line
    joined = re.sub(r"\\\n\s*", " ", text)
    for line in joined.splitlines():
        for m in re.finditer(r"gomadtool (" + "|".join(map(re.escape, commands)) + r")\b([^`\n]*)", line):
            command, rest = m.group(1), m.group(2)
            words = [command]
            sub = re.match(r"\s+([a-z-]+)", rest)
            if command in SUBS and sub and sub.group(1) in SUBS[command]:
                words.append(sub.group(1))
            elif command in SUBS:
                continue
            key = tuple(words)
            if key not in cache:
                cache[key] = helpflags(words)
            for flag in re.findall(r"(?<![\w-])--([a-z][a-z0-9-]*)", rest):
                checked += 1
                if flag not in cache[key]:
                    errors.append(f"{doc}: gomadtool {' '.join(words)} --{flag} is not accepted")
index = open("tools/gomad3/CLI.md").read().split("### `gomadtool`", 1)[1]
rows = re.findall(r"^\| `([a-z-]+)` \|", index, re.M)
for c in commands:
    if c not in rows: errors.append(f"CLI.md command index lacks gomadtool {c}")
for r in rows:
    if r not in commands: errors.append(f"CLI.md command index names unknown gomadtool {r}")
def slugs(path):
    out = set()
    for h in re.findall(r"^#+ (.+)$", open(path).read(), re.M):
        s = re.sub(r"[^\w\- ]", "", h.strip().lower()).replace(" ", "-")
        out.add(s)
    return out
links = 0
for doc in DOCS:
    text = re.sub(r"```.*?```", "", open(doc).read(), flags=re.S)
    for target in re.findall(r"\]\(([^)\s]+)\)", text):
        if re.match(r"[a-z]+:", target): continue
        links += 1
        path, _, frag = target.partition("#")
        full = os.path.normpath(os.path.join(os.path.dirname(doc), path)) if path else doc
        if not os.path.exists(full):
            errors.append(f"{doc}: link {target} does not resolve"); continue
        if frag and full.endswith(".md") and frag not in slugs(full):
            errors.append(f"{doc}: fragment {target} does not resolve")
print(f"flags checked: {checked}; commands: {len(commands)}; index rows: {len(rows)}; links checked: {links}")
for e in errors: print("ERROR", e)
sys.exit(1 if errors else 0)
