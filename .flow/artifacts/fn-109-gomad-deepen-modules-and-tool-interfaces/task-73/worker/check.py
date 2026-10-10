import collections
import hashlib
import json
from pathlib import Path
import re
import subprocess

ROOT = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal/.worktrees/fn-109-73-unix-mode-candidate')
OUTPUT = ROOT/'.flow/tmp/fn10973-evidence'
BASE = 'ca4d8b88cf95da0b0a911efdc3a18f89bcbb68ae'
SELECTED = 'tools/gomad3/runner/runner_mode_unix_test.go'
NAME = 'TestRunEnforcesBatchModesIndependentOfUmask'
original = subprocess.check_output(['/usr/bin/git','show',BASE+':'+SELECTED],cwd=ROOT)
current = (ROOT/SELECTED).read_bytes()
addition = b'\tconfigDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)\n'
assert current.count(addition) == 1
assert current.replace(addition,b'',1) == original
assert hashlib.sha256(original).hexdigest() == 'ebdc20609fd89c246bf345e0df40f3c126b09765d898770dabe4389518228168'
assert b'defer syscall.Umask(oldUmask)\n\n'+addition+b'\tsummary, err := exploreWith' in current
assert b't.Parallel' not in current
assert subprocess.check_output(['/usr/bin/git','diff','--name-only','--',':(exclude).flow'],cwd=ROOT).decode().splitlines() == [SELECTED]

def outcomes(name):
    found = {}
    for line in (OUTPUT/(name+'.log')).read_text().splitlines():
        event = json.loads(line)
        if event.get('Test') and event.get('Action') in ('pass','fail','skip'):
            assert event['Test'] not in found
            found[event['Test']] = event['Action']
    return found

old, new = outcomes('before-ordinary'), outcomes('after-ordinary')
assert old.keys() == new.keys()
changed = {key:[old[key],new[key]] for key in old if old[key]!=new[key]}
assert changed == {NAME:['fail','pass']}
assert outcomes('before-selected') == {NAME:'fail'}
assert outcomes('after-selected') == {NAME:'pass'}
assert outcomes('before-controls') == outcomes('after-controls')
assert len(outcomes('before-controls')) > 6
assert set(outcomes('after-controls').values()) == {'pass'}
assert outcomes('before-public-guard') == outcomes('after-public-guard')
assert set(outcomes('after-public-guard').values()) == {'pass'}

def blocks(name):
    lines = (OUTPUT/(name+'.log')).read_text().splitlines()
    found = []
    for i,line in enumerate(lines):
        if re.match(r'^tools/gomad3/[^:]+:\d+:\d+: .+$',line):
            assert i+2<len(lines) and '^' in lines[i+2]
            found.append('\n'.join(lines[i:i+3]))
    return found

before, after = blocks('before-aggregate-lint'), blocks('after-aggregate-lint')
assert before == after
assert len(before) == 50
assert collections.Counter(re.search(r'\(([^()]*)\)$',block.splitlines()[0])[1] for block in before) == {'forbidigo':8,'staticcheck':42}
joined_sha = hashlib.sha256('\n'.join(before).encode()).hexdigest()
assert joined_sha == '034b5959d8f6689fefbd9215234326d66b3809e204cf628c62b244e256398eea'
for phase in ('before','after'):
    receipt = json.loads((OUTPUT/(phase+'-aggregate-lint.json')).read_text())
    assert receipt['exit_code'] == 2
    assert '-vettool' not in (OUTPUT/(phase+'-aggregate-lint.log')).read_text()

settings = []
for phase in ('before','after'):
    for name in ('selected','controls','public-guard','ordinary','aggregate-lint'):
        binding = json.loads((OUTPUT/(phase+'-'+name+'-binding.json')).read_text())
        receipt = json.loads((OUTPUT/(phase+'-'+name+'.json')).read_text())
        assert receipt['source_before_after_equal'] and receipt['tools_before_after_equal'] and receipt['all_commands_terminal']
        before_settings = dict(binding['actual_go_settings'])
        after_settings = dict(receipt['actual_go_settings_after'])
        for value in (before_settings,after_settings):
            value['GOGCCFLAGS'] = re.sub(r'(?<=/tmp/)go-build[0-9]+(?==/tmp/go-build)', 'go-build<VOLATILE>',value['GOGCCFLAGS'])
        assert before_settings == after_settings
        settings.append(phase+'-'+name)
print(json.dumps({'whole_file_reconstruction':True,'base_sha256':hashlib.sha256(original).hexdigest(),'candidate_sha256':hashlib.sha256(current).hexdigest(),'ordinary_before_counts':dict(collections.Counter(old.values())),'ordinary_after_counts':dict(collections.Counter(new.values())),'emitted_named_set_count':len(old),'sole_outcome_change':changed,'unchanged_controls_count':len(outcomes('after-controls')),'lint_complete_blocks':len(before),'lint_complete_blocks_sha256':joined_sha,'lint_disposition':'RED50; 8forbidigo,42ST1005; aggregate errortype unreached','raw_settings_reconciled':settings,'GOGCCFLAGS_exception':'only numeric temporary go-build path; CGO_ENABLED=0; raw values retained'}))
