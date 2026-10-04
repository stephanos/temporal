#!/usr/bin/env python3
"""Retain exact workflow commands and immutable disposition source identities."""
import hashlib
import json
from pathlib import Path
import re
import subprocess

HERE = Path(__file__).resolve().parent
REPO = HERE.parents[3]

def digest(data):
    return hashlib.sha256(data).hexdigest()

sources = ['.github/workflows/gomad3.yml','.github/workflows/gomad3-smoke.yml','Makefile','tools/gomad3/Makefile','tools/gomad3/toolchain/version/version.json','tools/gomad3/deterministicio/boundary/manifest.json']
commands = []
for source in sources[:2]:
    lines = (REPO/source).read_text().splitlines()
    job = None
    name = None
    cwd = '.'
    index = 0
    while index < len(lines):
        line = lines[index]
        match = re.match(r'^  ([a-z][a-z0-9-]+):$',line)
        if match:
            job = match[1]
            cwd = '.'
        match = re.match(r'^      - name: (.+)$',line)
        if match:
            name = match[1]
            cwd = '.'
        match = re.match(r'^        working-directory: (.+)$',line)
        if match:
            cwd = match[1]
        match = re.match(r'^        run: (.*)$',line)
        if match:
            value = match[1]
            start = index+1
            if value in ['>-','|']:
                content = []
                index += 1
                while index < len(lines) and (lines[index].startswith('          ') or not lines[index]):
                    content.append(lines[index][10:])
                    index += 1
                value = ' '.join(content).strip() if value == '>-' else '\n'.join(content).rstrip()+'\n'
                index -= 1
            role = 'qualification gate'
            if name in ['Restore the soak ledger from the latest retained run','Publish the soak summary']:
                role = 'CI artifact administration; recorded for completeness, not a qualification gate'
            elif job and job.startswith('determinism-soak-'):
                role = 'scheduled/dispatched fn-112 soak obligation; Linux informational under unchanged D12'
            elif job == 'host-tools-linux':
                role = 'standalone platform-neutral CI tooling gate; developmental here'
            platforms = ['linux/amd64'] if job and ('linux' in job) else ['darwin/arm64']
            commands.append(dict(source=source,job=job,step=name,line=start,cwd=cwd,command=value,role=role,platforms={platform:'incomplete; no source-bound native result on this host' for platform in platforms},executed=False,exit_code=None))
        index += 1
required = [
    'make -C tools/gomad3 validate',
    "GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test",
    'go test -count=1 -tags test_dep ./tools/gomad3sim/...',
    'make gomad3-runner',
    'make gomad3-integration-test',
    'make gomad3-smoke-qualification',
    'make -C tools/gomad3 compatibility-pack-qualification core-qualification-set',
    'make gomad3-qualification',
    'tools/gomad3/.toolchain/bin/go test -count=1 -tags test_dep,gomad3_toolchain ./tools/gomad3sim',
    'make lint-code-fast',
]
targets = ['test-harness','test-toolchain','intercept-test','test-host','overlay-test','test-simulation','world-test','test-builder','test-live-capability','test-runtime','test-upstream']
for command in required + ['make -C tools/gomad3 '+target for target in targets]:
    commands.append(dict(source='task 21 Quick/parent R19/nested Makefile',command=command,cwd='.',role='required native gate; target rows expand full test coverage, not duplicate passes',platforms={platform:'incomplete; native host and patched toolchain absent' for platform in ['darwin/arm64','linux/amd64']},executed=False,exit_code=None))
make_lines = (REPO/'tools/gomad3/Makefile').read_text().splitlines()
active_target = None
recipe = []
recipe_start = None
def retain_recipe():
    if recipe and active_target in set(targets+['toolchain','runner','compatibility-pack-qualification','qualification-set','upgrade-dossier','clock-audit','validate-toolchain','validate-compatibility','validate-qualification']):
        platforms = ['darwin/arm64'] if active_target in ['clock-audit','upgrade-dossier'] else ['darwin/arm64','linux/amd64']
        commands.append(dict(source='tools/gomad3/Makefile',target=active_target,line=recipe_start,cwd='tools/gomad3',command='\n'.join(recipe),role='exact Make recipe; variables resolve through the named target, no invocation claimed',platforms={platform:'incomplete; required native target not executed' for platform in platforms},executed=False,exit_code=None))
for number,line in enumerate(make_lines,1):
    if line.startswith('\t'):
        if not recipe:
            recipe_start = number
        recipe.append(line[1:])
        if not line.endswith('\\'):
            retain_recipe()
            recipe = []
    else:
        retain_recipe()
        recipe = []
        match = re.match(r'^([a-z][a-z0-9-]*):(?:\s|$)',line)
        active_target = match[1] if match else None
current_dispositions = []
for relative in ['tools/gomad3integration/qualification','tools/gomad3/qualification']:
    for path in sorted((REPO/relative).glob('*.json')):
        name = str(path.relative_to(REPO))
        old = subprocess.run(['git','show','6782b55f49a0317b230e827ea2a63a37d116d502:'+name],cwd=REPO,stdout=subprocess.PIPE,stderr=subprocess.PIPE)
        current_dispositions.append(dict(path=name,current_sha256=digest(path.read_bytes()),historical_git_base_sha256=digest(old.stdout) if old.returncode==0 else None,historical_projection='Git base only. Dirty full-repository baseline was not reconstructed; do not infer exact whole-baseline disposition identity.',bytes_equal_to_git_base=old.returncode==0 and old.stdout==path.read_bytes()))
ledger = dict(candidate_revision=subprocess.run(['git','rev-parse','HEAD'],cwd=REPO,check=True,stdout=subprocess.PIPE,text=True).stdout.strip(),host='linux/arm64',patched_go_present=(REPO/'tools/gomad3/.toolchain/bin/go').is_file(),native_acceptance=False,source_sha256={name:digest((REPO/name).read_bytes()) for name in sources},commands=commands,disposition_sources=current_dispositions,core_report=dict(path='tools/gomad3/.toolchain/core-qualification-set.json',present=(REPO/'tools/gomad3/.toolchain/core-qualification-set.json').is_file(),qualification='no current integrated source-bound native report admitted'))
(HERE/'native-command-ledger.json').write_text(json.dumps(ledger,indent=2)+'\n')
assert len([row for row in commands if row['source'].endswith('.yml')]) > 20
assert all(not row['executed'] and row['exit_code'] is None for row in commands)
print(json.dumps(dict(workflow_and_required_command_rows=len(commands),disposition_files=len(current_dispositions),native_acceptance=False),indent=2))
