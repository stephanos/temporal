import base64
import datetime
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys

ROOT = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
HERE = Path(__file__).resolve().parent
GO = '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go'
ENV = {**os.environ, 'GOTOOLCHAIN': 'local', 'GOWORK': 'off',
       'GOPROXY': 'off', 'GOSUMDB': 'off', 'GOENV': 'off', 'GOFLAGS': '',
       'PATH': str(Path(GO).parent) + ':' + os.environ['PATH']}
for key in ('GOMADSEED', 'GOMAD3_CHILD_SEED'):
    ENV.pop(key, None)


def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def observation(name, argv):
    started = datetime.datetime.now(datetime.timezone.utc).isoformat()
    result = subprocess.run(argv, cwd=ROOT, env=ENV, capture_output=True, timeout=600)
    item = {'name': name, 'argv': argv, 'cwd': str(ROOT), 'started': started,
            'ended': datetime.datetime.now(datetime.timezone.utc).isoformat(),
            'exit': result.returncode,
            'environment': {key: ENV[key] for key in ('GOTOOLCHAIN', 'GOWORK', 'GOPROXY',
                           'GOSUMDB', 'GOENV', 'GOFLAGS', 'PATH')},
            'unset_environment': ['GOMADSEED', 'GOMAD3_CHILD_SEED'],
            'stdout_base64': base64.b64encode(result.stdout).decode(),
            'stderr_base64': base64.b64encode(result.stderr).decode()}
    print(name, 'exit', result.returncode, flush=True)
    return item


mode = sys.argv[1]
out = HERE / (mode + '-checks.json')
if mode == 'baseline':
    commands = [
        ('generated-source-validation', ['make', '-C', 'tools/gomad3', 'validate']),
        ('option-contract-tests', [GO, '-C', 'tools/gomad3', 'test', '-tags', 'test_dep',
         '-count=1', '-json', '-run',
         '^(TestResolveExploreStrategy.*|TestResolveExploreGuidance.*|TestGuideRegressionRequiresGuidance|TestFullyAnsweredGuidance.*|TestGuidanceReports.*|TestResumeGuidanceMode.*|TestRunMinimizeForwardsFlags|TestRunResumeForwardsCampaignAndClassifiesResult)$',
         './cmd/gomad/internal/cli'])]
elif mode == 'final':
    commands = [('documentation-source-audit', [sys.executable, str(HERE / 'audit-docs.py')]),
                ('documented-error-and-identity-tests', [GO, '-C', 'tools/gomad3', 'test',
                 '-tags', 'test_dep', '-count=1', '-json', '-run',
                 '^(TestCharacterize(InputRejection|ByteSizeAndFlagValueRejection|ExploreRequestDefaultsAndWiring)|TestWorkspace(Resume.*|KeepsStatePerParentArtifact|RejectsSymbolicLinkStateRoot|StateGatesInitialRunsAndResumes)|TestIdentityForProjectsCanonicalEnvironmentAcrossSeedsAndProfiles|TestCorpusRejects(IdentityChangesAndNonMatchingReplay|PreviousAndFutureSchemaBeforeChangedIdentity|CaseWithChangedEnvironment)|TestPinnedCoverageBuildSettingsRejectInstrumentation|TestValidateProvenanceRejectsUnsupportedBuildModes)$',
                 './cmd/gomad/internal/cli', './runner/internal/minimizer',
                 './runner/internal/corpus', './target']),
                ('whitespace', ['git', 'diff', '--check'])]
elif mode == 'portable-completion':
    commands = [('portable-provenance-and-replay-refusal', [GO, '-C', 'tools/gomad3',
                 'test', '-tags', 'test_dep', '-count=1', '-json', '-run',
                 '^(TestValidateProvenanceRejectsUnsupportedBuildModes|TestReplayBuildInfoRejectsMatchingCoverageInstrumentation)$',
                 './target', './runner']),
                ('whitespace', ['git', 'diff', '--check'])]
elif mode == 'freeze':
    commands = [('documentation-source-audit', [sys.executable, str(HERE / 'audit-docs.py')]),
                ('whitespace', ['git', 'diff', '--check'])]
else:
    raise SystemExit('unknown mode')
items = []
for name, argv in commands:
    items.append(observation(name, argv))
    out.write_text(json.dumps({'mode': mode, 'go_sha256': digest(GO), 'observations': items}, indent=2) + '\n')
    if items[-1]['exit']:
        raise SystemExit(items[-1]['exit'])
